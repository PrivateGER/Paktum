package main

import (
	"Paktum/Database"
	"bytes"
	"flag"
	"fmt"
	"github.com/go-redis/redis/v8"
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

var meiliClient *meilisearch.Client
var redisClient *redis.Client

func main() {
	var mode string
	flag.StringVar(&mode, "mode", "", "The mode to run in. Either 'scrape', 'process', 'cleanup' or 'server'")

	var serverBaseURL string
	flag.StringVar(&serverBaseURL, "base-url", "http://localhost:8080", "The base URL of the server. No trailing slash.")

	// redis is shared by server and scrape mode and used as a transfer layer
	var redisHostname string
	flag.StringVar(&redisHostname, "redis", "localhost:6379", "The redis server to connect to")
	var redisPass string
	flag.StringVar(&redisPass, "redisPass", "", "The password for the redis server")

	// meili is shared by server and process mode and used as search index
	var meiliHostname string
	flag.StringVar(&meiliHostname, "meilihost", "http://localhost:7700", "The meilisearch server to connect to")
	var meiliKey string
	flag.StringVar(&meiliKey, "meilikey", "", "The meilisearch master-key to use")

	// process mode is used to process the images
	var imageDir string
	flag.StringVar(&imageDir, "imageDir", "./images/", "The directory to store images in")

	// server mode
	var port int
	flag.IntVar(&port, "port", 9000, "The port to run the server on")

	var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	flag.Parse()

	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
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

	if mode == "scrape" {
		ScrapeMode()
	} else if mode == "process" {
		ProcessMode(imageDir)
	} else if mode == "cleanup" {
		CleanupMode()
	} else if mode == "server" {
		ServerMode(imageDir)
	}
}

func imageExists(meiliIndex *meilisearch.Index, md5 string) bool {
	filter := []string{fmt.Sprintf(`ID = %s`, md5)}

	log.Trace("Constructed meilisearch filter:", filter)
	search, err := meiliIndex.Search("", &meilisearch.SearchRequest{
		Filter: filter,
	})
	if err != nil {
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
	defer resp.Body.Close()
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
		cmd := exec.Command("/usr/bin/ffmpeg", "-i", imageDir+filename, "-vframes", "1", "-s", fmt.Sprintf("%dx%d", 1920, 1080), "-f", "singlejpeg", "-")

		var videoFrame bytes.Buffer
		cmd.Stdout = &videoFrame // overwrite the main buffer with video frame
		var stderrBuffer bytes.Buffer
		cmd.Stderr = &stderrBuffer
		err = cmd.Run()

		if err != nil {
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

func waitForMeilisearchTask(info *meilisearch.TaskInfo, client *meilisearch.Client) bool {
	for {
		task, err := client.GetTask(info.TaskUID)
		if err != nil {
			log.Fatal("Failed to get task:", err)
			return false
		}
		if task.Status == "failed" {
			if task.Error.Code == "index_already_exists" {
				return true
			}
			log.Error("MeiliSearch task failed:", task.Error.Message, "-", task.Error.Code)
			return false
		}
		if task.Status == "succeeded" {
			return true
		}
		time.Sleep(time.Millisecond * 500)
	}
}

func GetMeilisearch() *meilisearch.Client {
	return meiliClient
}

func GetRedis() *redis.Client {
	return redisClient
}
