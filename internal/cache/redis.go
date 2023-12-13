package cache

import (
	"context"
	"os"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

func NewRedisClient() *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	_, err := client.Ping(context.Background()).Result()
	if err != nil {
		slog.Error("failed to connect to redis", err)
		os.Exit(1)
	}

	return client
}
