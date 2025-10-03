package cache

import (
	"context"
	"os"

	"one-click-aks-server/internal/logging"

	"github.com/redis/go-redis/v9"
)

func NewRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		logging.LogError(context.Background(), "failed to connect to redis", "error", err)
		os.Exit(1)
	}

	return client
}
