package Database

import (
	_ "embed"
	log "github.com/sirupsen/logrus"
	"time"
)

//go:generate bash get_git_hash.sh
//go:embed git_hash.txt
var version string

func GetVersion() string {
	return version
}

var startTime time.Time

func init() {
	startTime = time.Now()

	// strip newline from version
	version = version[:len(version)-1]
}

func GetUptime() time.Duration {
	return time.Since(startTime)
}

var adminToken string

func SetAdminToken(token string) {
	adminToken = token
}

func GetAdminToken() string {
	if adminToken == "" {
		log.Fatal("Admin token not set")
	}

	return adminToken
}
