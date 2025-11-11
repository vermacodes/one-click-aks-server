package repository

import (
	"context"

	"one-click-aks-server/internal/cache"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"

	"github.com/redis/go-redis/v9"
)

type RedisRepository struct {
	rdb *redis.Client
}

func NewRedisRepository() entity.RedisRepository {
	return &RedisRepository{
		rdb: cache.NewRedisClient(),
	}
}

func (r *RedisRepository) ResetServerCache(ctx context.Context) error {
	userId := helper.GetUserIDFromContext(ctx)

	logging.LogInfo(ctx, "attempting to reset cache for user", "user_id", userId)

	// Based on the actual key structure we see: ashisverma@microsoft.com-logs, ashisverma@microsoft.com-actionstatus
	// Try multiple patterns that might match user keys
	patterns := []string{
		userId + "*",         // Direct user prefix: user@example.com*
		userId + "-*",        // User with dash separator: user@example.com-*
		"*-" + userId + "-*", // User in middle: *-user@example.com-*
		"*" + userId + "*",   // User anywhere in key: *user@example.com*
	}

	var allKeys []string
	for _, pattern := range patterns {
		keys, err := r.rdb.Keys(ctx, pattern).Result()
		if err != nil {
			logging.LogError(ctx, "failed to get keys for pattern", "pattern", pattern, "error", err)
			continue
		}
		allKeys = append(allKeys, keys...)
		logging.LogDebug(ctx, "found keys for pattern", "pattern", pattern, "count", len(keys), "keys", keys)
	}

	// Remove duplicates
	uniqueKeys := make(map[string]bool)
	var keysToDelete []string
	for _, key := range allKeys {
		if !uniqueKeys[key] {
			uniqueKeys[key] = true
			keysToDelete = append(keysToDelete, key)
		}
	}

	logging.LogInfo(ctx, "found user keys to delete", "count", len(keysToDelete), "keys", keysToDelete)

	// Delete the keys if any found
	if len(keysToDelete) > 0 {
		err := r.rdb.Del(ctx, keysToDelete...).Err()
		if err != nil {
			logging.LogError(ctx, "failed to delete user keys", "error", err)
			return err
		}
		logging.LogInfo(ctx, "successfully deleted user keys", "count", len(keysToDelete))
	} else {
		logging.LogInfo(ctx, "no user keys found to delete")
	}

	return nil
}
