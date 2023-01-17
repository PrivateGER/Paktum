package DBMigrations

import (
	"Paktum/Database"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
	"os"
)

func init() {
	Database.RegisterMigration(Database.Migration{
		Version: 1,
		Handler: func() {
			var meiliClient = Database.GetMeiliClient()
			taskid, err := meiliClient.CreateIndex(&meilisearch.IndexConfig{
				Uid:        "images",
				PrimaryKey: "id",
			})
			if err != nil {
				return
			}
			if !Database.WaitForMeilisearchTask(taskid) {
				log.Error("Migration failed: Failed to create MeiliSearch index")
				os.Exit(1)
			}

			// Update filterable attributes
			imageCollection := Database.GetMeiliClient().Index("images")
			taskid, err = imageCollection.UpdateFilterableAttributes(&[]string{"ID", "Tagstring", "Rating", "Tags", "Filename"})
			if err != nil {
				log.Error("Migration failed: Failed to update filterable attributes:", err)
			}
			if !Database.WaitForMeilisearchTask(taskid) {
				log.Error("Migration failed: Failed to update filterable attributes")
				os.Exit(1)
				return
			}

			// Update sortable attributes
			taskid, err = imageCollection.UpdateSortableAttributes(&[]string{"Added"})
			if err != nil {
				log.Error("Migration failed: Failed to update sortable attributes:", err)
			}
			if !Database.WaitForMeilisearchTask(taskid) {
				log.Error("Migration failed: Failed to update sortable attributes")
				os.Exit(1)
				return
			}

			return
		},
	})
}
