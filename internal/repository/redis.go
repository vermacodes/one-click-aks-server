package repository

import (
	"context"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"

	"github.com/redis/go-redis/v9"
)

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

func (r *RedisRepository) ResetServerCache(ctx context.Context) error {
	rdb := newRedisClient()
	userId := helper.GetUserIDFromContext(ctx)

	// Find all keys that start with the userId
	pattern := userId + "*"
	keys, err := rdb.Keys(ctx, pattern).Result()
	if err != nil {
		return err
	}

	// Delete the keys if any found
	if len(keys) > 0 {
		return rdb.Del(ctx, keys...).Err()
	}

	return nil
}
