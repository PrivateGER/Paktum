package Database

import (
	_ "Paktum/Database/DBMigrations"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"sort"
	"time"
)

var currentDBVersion = 999999
var migrations []Migration

// fetch the database version from the database
func setVersion() {
	meiliClient := GetMeiliClient()
	versionIndex := meiliClient.Index("version")

	type Version struct {
		Id      int `json:"id"`
		Version int `json:"version"`
	}
	var version Version
	err := versionIndex.GetDocument("1", &meilisearch.DocumentQuery{
		Fields: []string{"version"},
	}, &version)
	if err != nil {
		log.Error("Failed to get database version:", err)
		version.Version = 0
	}

	if version.Version == 0 {
		log.Info("Failed to get database version, assuming version 0")
		currentDBVersion = 0
	} else {
		currentDBVersion = version.Version
	}

	log.Info("Current database version: ", currentDBVersion)
}

type Migration struct {
	Version int
	Handler func()
}

func RegisterMigration(migration Migration) {
	migrations = append(migrations, migration)
	log.Info("Registered migration for version ", migration.Version)
}

func ExecuteMigrations() {
	setVersion()
	startTime = time.Now()

	// Check which migrations need to be executed,
	// sort by version and execute them
	var migrationsToExecute []Migration
	for _, migration := range migrations {
		if migration.Version > currentDBVersion {
			migrationsToExecute = append(migrationsToExecute, migration)
		}
	}

	// Sort migrations by version
	sort.Slice(migrationsToExecute, func(i, j int) bool {
		return migrationsToExecute[i].Version < migrationsToExecute[j].Version
	})

	// Execute migrations
	for _, migration := range migrationsToExecute {
		log.Println("Executing migration for version ", migration.Version)
		migration.Handler()
		log.Println("Migration for version ", migration.Version, " executed")
	}

	// Update database version
	meiliClient := GetMeiliClient()
	versionIndex := meiliClient.Index("version")
	_, err := versionIndex.UpdateDocuments(&map[string]int{
		"id":      1,
		"version": currentDBVersion,
	})
	if err != nil {
		log.Error("Failed to update database version:", err)
		return
	}

	log.Println("Migrations finished in", time.Since(startTime))
}

// Wait for a meilisearch task to finish, return true if successful
func WaitForMeilisearchTask(info *meilisearch.TaskInfo) bool {
	client := GetMeiliClient()

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
