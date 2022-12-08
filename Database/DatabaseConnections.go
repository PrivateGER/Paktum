package Database

import (
	"context"
	"github.com/go-redis/redis/v8"
	"github.com/meilisearch/meilisearch-go"
	log "github.com/sirupsen/logrus"
)

var meiliClient *meilisearch.Client
var redisClient *redis.Client

func ConnectMeilisearch(host string, apiKey string) *meilisearch.Client {
	meiliClient = meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   host,
		APIKey: apiKey,
	})

	return meiliClient
}

func ConnectRedis(host string, password string, db int) *redis.Client {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     host,
		Password: password,
		DB:       db,
	})

	return redisClient
}

func GetMeiliClient() *meilisearch.Client {
	if meiliClient == nil {
		log.Fatal("Meili client not initialized")
	}

	if !meiliClient.IsHealthy() {
		log.Fatal("Meili client is not healthy")
	}

	log.Debug("Meili client is healthy and initialized, returning instance")

	return meiliClient
}

func GetRedis() *redis.Client {
	if redisClient == nil {
		log.Fatal("Redis client not initialized")
	}

	// check if redis is healthy
	_, err := redisClient.Ping(context.Background()).Result()
	if err != nil {
		log.Fatal("Redis client is not healthy")
	}

	log.Debug("Redis client is healthy and initialized, returning instance")

	return redisClient
}
