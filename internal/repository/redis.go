package repository

import (
	"context"

	"one-click-aks-server/internal/entity"

	"github.com/redis/go-redis/v9"
)

var ctx = context.Background()

func newRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

type RedisRepository struct{}

func NewRedisRepository() entity.RedisRepository {
	return &RedisRepository{}
}

func (r *RedisRepository) ResetServerCache() error {
	rdb := newRedisClient()
	return rdb.FlushAll(ctx).Err()
}
