package main

import (
	"Paktum/Database"
	"bytes"
	"flag"
	"fmt"
	"github.com/getsentry/sentry-go"
	env_flag "github.com/jnovack/flag"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"io"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"
)

func init() {
	lvl, ok := os.LookupEnv("LOG_LEVEL")
	// LOG_LEVEL not set, let's default to debug
	if !ok {
		lvl = "debug"
	}
	// parse string, this is built-in feature of logrus
	ll, err := log.ParseLevel(lvl)
	if err != nil {
		ll = log.DebugLevel
	}
	// set global log level
	log.SetLevel(ll)
}

func main() {
	err := sentry.Init(sentry.ClientOptions{
		Dsn: "https://c00568e1589646fe828fd7cd2196f734@glitchtip.pxroute.net/1",
	})
	if err != nil {
		log.Error("Failed to initialize sentry error logging:", err.Error())
	}

	var mode string
	env_flag.StringVar(&mode, "mode", "", "The mode to run in. Either 'scrape', 'process', 'cleanup', 'inference' or 'server'")

	var enableCors bool
	env_flag.BoolVar(&enableCors, "enable-cors", false, "Enable CORS headers, restricting API access to your set base URL")

	var serverBaseURL string
	env_flag.StringVar(&serverBaseURL, "base-url", "http://paktum.localtest.me", "The base URL of the Paktum server. No trailing slash.")

	var imgproxyBaseURL string
	env_flag.StringVar(&imgproxyBaseURL, "imgproxy-url", "http://imgproxy.localtest.me", "The base URL of the imgproxy server. No trailing slash.")

	var imgproxyKey string
	env_flag.StringVar(&imgproxyKey, "imgproxy-key", "943b421c9eb07c830af81030552c86009268de4e532ba2ee2eab8247c6da0881", "The key to use for imgproxy.")

	var imgproxySalt string
	env_flag.StringVar(&imgproxySalt, "imgproxy-salt", "520f986b998545b4785e0defbc4f3c1203f22de2374a3d53cb7a7fe9fea309c5", "The salt to use for imgproxy.")

	// redis is shared by server and scrape mode and used as a transfer layer
	var redisHostname string
	env_flag.StringVar(&redisHostname, "redis", "localhost:6379", "The redis server to connect to")
	var redisPass string
	env_flag.StringVar(&redisPass, "redisPass", "", "The password for the redis server")

	// meili is shared by server and process mode and used as search index
	var meiliHostname string
	env_flag.StringVar(&meiliHostname, "meilihost", "http://localhost:7700", "The meilisearch server to connect to")
	var meiliKey string
	env_flag.StringVar(&meiliKey, "meilikey", "", "The meilisearch master-key to use")

	// process mode is used to process the images
	var imageDir string
	env_flag.StringVar(&imageDir, "imageDir", "./images/", "The directory to store images in")

	// server mode
	var port int
	env_flag.IntVar(&port, "port", 9000, "The port to run the server on")

	var adminToken string
	env_flag.StringVar(&adminToken, "admin-token", "", "The admin token to use for the GraphQL API")
	if adminToken == "" {
		log.Warning("No admin token set, access to administrative features will be disabled")
	}

	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	env_flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			sentry.CaptureException(err)
			log.Fatal(err)
		}
		err = pprof.StartCPUProfile(f)
		if err != nil {
			log.Error("Failed to start CPU profile:", err)
		}

		// Hook the SIGINT (CTRL+C) event to write profile on exit
		c := make(chan os.Signal, 2)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM) // subscribe to system signals
		onKill := func(c chan os.Signal) {
			select {
			case <-c:
				defer os.Exit(0)
				defer f.Close()
				defer pprof.StopCPUProfile()
			}
		}

		go onKill(c)
	}

	if mode != "scrape" && mode != "server" && mode != "process" && mode != "cleanup" {
		log.Error("Please choose either scraping or server mode")
		flag.Usage()
		os.Exit(1)
		return
	}

	Database.ConnectRedis(redisHostname, redisPass, 0)
	Database.ConnectMeilisearch(meiliHostname, meiliKey)
	Database.SetBaseURL(serverBaseURL)
	Database.SetImgproxyBaseUrl(imgproxyBaseURL)
	Database.SetImgproxySecrets(imgproxyKey, imgproxySalt)
	Database.SetCorsEnabled(enableCors)
	Database.SetAdminToken(adminToken)

	func() { // Sentry harness to catch any panic that propagates to the top level
		defer func() {
			err := sentry.Recover()
			if err != nil {
				log.Error("Sentry recovered panic:", err)
				sentry.CurrentHub().Recover(err)
				sentry.Flush(5 * time.Second)
			}
		}()

		if mode == "scrape" {
			ScrapeMode()
		} else if mode == "process" {
			ProcessMode(imageDir)
		} else if mode == "cleanup" {
			CleanupMode(imageDir)
		} else if mode == "server" {
			ServerMode(imageDir)
		} else {
			log.Error("Please choose a valid server operation mode")
			flag.Usage()
			os.Exit(1)
		}
	}()
}

func imageExists(meiliIndex *meilisearch.Index, md5 string) bool {
	filter := []string{fmt.Sprintf(`ID = %s`, md5)}

	log.Trace("Constructed meilisearch filter:", filter)
	search, err := meiliIndex.Search("", &meilisearch.SearchRequest{
		Filter: filter,
	})
	if err != nil {
		sentry.CaptureException(err)
		log.Error("Failed to search meili:", err.Error())
		return false
	}
	if len(search.Hits) != 0 {
		log.Info("Image already exists, skipping")
		return true
	}
	return false
}

// download image
// returns the pHash as uint64
// and the size in bytes as int
// and the image dimensions, width and height as int
func downloadImage(url string, imageDir string, filename string) (error, uint64, int, int, int) {
	temporaryImageFile, err := os.CreateTemp(imageDir, "temp-paktum-")
	if err != nil {
		log.Error("Failed to create file:", err.Error())
		return err, 0, 0, 0, 0
	}

	resp, err := http.Get(url)
	defer func(Body io.ReadCloser) {
		err := Body.Close()

		if err != nil {
			sentry.CaptureException(err)
			log.Error("Failed to close body:", err.Error())
		}
	}(resp.Body)
	if err != nil {
		log.Error("Failed to download image:", err.Error())
		return err, 0, 0, 0, 0
	}
	log.Trace("CONTENT-LENGTH:", resp.ContentLength)
	if resp.ContentLength < 1 {
		log.Error("EMPTY RESPONSE, url: ", url)
		return err, 0, 0, 0, 0
	}

	if resp.StatusCode != http.StatusOK {
		log.Error("Failed to download image, response code: ", resp.Status, " on url: ", url)
		return err, 0, 0, 0, 0
	}

	var buffer bytes.Buffer
	responseBytes, _ := io.ReadAll(resp.Body)
	written, err := buffer.Write(responseBytes)
	if err != nil {
		log.Error("Failed to copy image to buffer:", err.Error())
		return err, 0, 0, 0, 0
	}
	reader := bytes.NewReader(buffer.Bytes())
	log.Trace("Written bytes to buffer:", written)

	size := int(resp.ContentLength)

	imgByteCount, err := io.Copy(temporaryImageFile, reader)
	reader.Seek(0, 0)
	if err != nil {
		log.Error("Failed to write data into image:", err.Error())
		return err, 0, 0, 0, 0
	}
	log.Trace("Downloaded image, size: ", imgByteCount, " bytes")

	err = temporaryImageFile.Close()
	if err != nil {
		return err, 0, 0, 0, 0
	}

	// rename temp image file to proper name
	err = os.Rename(temporaryImageFile.Name(), imageDir+filename)
	if err != nil {
		log.Error("Failed to move image:", err.Error())
		return err, 0, 0, 0, 0
	}

	if strings.HasSuffix(filename, ".webm") || strings.HasSuffix(filename, ".mp4") {
		// video files have their frame extracted by ffmpeg
		// and the frame is used as the image
		log.Trace("Launching ffmpeg subprocess to extract frame from video")
		log.Info("Extracting first frame from video file ", filename)
		cmd := exec.Command("/usr/bin/ffmpeg", "-i", imageDir+filename, "-frames:v", "1", "-s", fmt.Sprintf("%dx%d", 640, 480), "-c:v", "mjpeg", "-f", "mjpeg", "-")
		log.Info(cmd.Path, cmd.Args)

		var videoFrame bytes.Buffer
		cmd.Stdout = &videoFrame // overwrite the main buffer with video frame
		var stderrBuffer bytes.Buffer
		cmd.Stderr = &stderrBuffer
		err = cmd.Run()

		if err != nil {
			sentry.CaptureException(err)
			log.Error("ffmpeg error: ", err, stderrBuffer.String())
		}

		reader = bytes.NewReader(videoFrame.Bytes())
		reader.Seek(0, 0)
	}

	// calculate pHash
	decodedImage := DecodeImage(io.NopCloser(reader))
	if decodedImage == nil {
		return nil, 0, size, 0, 0
	}

	return nil, GeneratePHash(decodedImage), size, decodedImage.Bounds().Dx(), decodedImage.Bounds().Dy()
}
