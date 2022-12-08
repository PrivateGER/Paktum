package Database

import (
	"encoding/hex"
	log "github.com/sirupsen/logrus"
)

var corsEnabled bool

func SetCorsEnabled(enabled bool) {
	corsEnabled = enabled
}

func GetCorsEnabled() bool {
	return corsEnabled
}

var imgproxyBaseUrl string
var imgproxyKey []byte
var imgproxySalt []byte

func SetImgproxyBaseUrl(url string) {
	imgproxyBaseUrl = url
}

func GetImgproxyBaseUrl() string {
	if imgproxyBaseUrl == "" {
		log.Fatal("Imgproxy base URL not set")
	}

	return imgproxyBaseUrl
}

func GetImgproxyKey() []byte {
	if len(imgproxyKey) == 0 {
		log.Fatal("Imgproxy key not set")
	}

	return imgproxyKey
}

func GetImgproxySalt() []byte {
	if len(imgproxySalt) == 0 {
		log.Fatal("Imgproxy salt not set")
	}

	return imgproxySalt
}

func SetImgproxySecrets(key string, salt string) {
	if key == "943b421c9eb07c830af81030552c86009268de4e532ba2ee2eab8247c6da0881" || salt == "520f986b998545b4785e0defbc4f3c1203f22de2374a3d53cb7a7fe9fea309c5" {
		log.Warning("Using default imgproxy secrets, this is not recommended in production and is a DoS risk")
	}

	imgproxyKey, _ = hex.DecodeString(key)
	imgproxySalt, _ = hex.DecodeString(salt)
}

var baseURL string

func SetBaseURL(url string) {
	baseURL = url
}

func GetBaseURL() string {
	if baseURL == "" {
		log.Fatal("Base URL not set")
	}

	return baseURL
}
