package repository

import (
	"context"

	"one-click-aks-server/internal/cache"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"

	"github.com/redis/go-redis/v9"
)

type logStreamRepository struct {
	rdb *redis.Client
}

func NewLogStreamRepository() entity.LogStreamRepository {
	return &logStreamRepository{
		rdb: cache.NewRedisClient(),
	}
}

func (l *logStreamRepository) SetLogsInRedis(ctx context.Context, logStream string) error {
	if err := l.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-logs", logStream, 0).Err(); err != nil {
		return err
	}

	if err := l.rdb.Publish(ctx, helper.GetUserIDFromContext(ctx)+"-redis-log-stream-pubsub-channel", logStream).Err(); err != nil {
		return err
	}

	return nil
}

func (l *logStreamRepository) GetLogsFromRedis(ctx context.Context) (string, error) {
	return l.rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-logs").Result()
}

func (l *logStreamRepository) WaitForLogsChange(ctx context.Context) (string, error) {
	rdb := l.rdb.Subscribe(ctx, helper.GetUserIDFromContext(ctx)+"-redis-log-stream-pubsub-channel")
	defer rdb.Close()

	for {
		msg, err := rdb.ReceiveMessage(ctx)
		if err != nil {
			return "", err
		}

		return msg.Payload, nil
	}
}

// User-specific methods

// func (l *logStreamRepository) SetLogsInRedisForUser(ctx context.Context, userID, logStream string) error {
// 	rdb := newLogStreamRedisClient()
// 	userLogKey := helper.GetUserIDFromContext(ctx) + "-logs"
// 	userChannelKey := helper.GetUserIDFromContext(ctx) + "-redis-log-stream-pubsub-channel"

// 	if err := rdb.Set(ctx, userLogKey, logStream, 0).Err(); err != nil {
// 		return err
// 	}

// 	if err := rdb.Publish(ctx, userChannelKey, logStream).Err(); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (l *logStreamRepository) GetLogsFromRedisForUser(ctx context.Context, userID string) (string, error) {
// 	rdb := newLogStreamRedisClient()
// 	userLogKey := helper.GetUserIDFromContext(ctx) + "-logs"
// 	return rdb.Get(ctx, userLogKey).Result()
// }

// func (l *logStreamRepository) WaitForLogsChangeForUser(ctx context.Context, userID string) (string, error) {
// 	rdb := newLogStreamRedisClient()
// 	userChannelKey := helper.GetUserIDFromContext(ctx) + "-redis-log-stream-pubsub-channel"
// 	pubsub := rdb.Subscribe(ctx, userChannelKey)
// 	defer pubsub.Close()

// 	for {
// 		msg, err := pubsub.ReceiveMessage(ctx)
// 		if err != nil {
// 			return "", err
// 		}

// 		return msg.Payload, nil
// 	}
// }
