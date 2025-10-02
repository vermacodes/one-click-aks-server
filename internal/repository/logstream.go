package repository

import (
	"context"

	"one-click-aks-server/internal/entity"

	"github.com/redis/go-redis/v9"
)

type logStreamRepository struct{}

func NewLogStreamRepository() entity.LogStreamRepository {
	return &logStreamRepository{}
}

var logStreamCtx = context.Background()

func newLogStreamRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func (l *logStreamRepository) SetLogsInRedis(logStream string) error {
	rdb := newLogStreamRedisClient()
	if err := rdb.Set(logStreamCtx, "logs", logStream, 0).Err(); err != nil {
		return err
	}

	if err := rdb.Publish(logStreamCtx, "redis-log-stream-pubsub-channel", logStream).Err(); err != nil {
		return err
	}

	return nil
}

func (l *logStreamRepository) GetLogsFromRedis() (string, error) {
	rdb := newLogStreamRedisClient()
	return rdb.Get(logStreamCtx, "logs").Result()
}

func (l *logStreamRepository) WaitForLogsChange() (string, error) {
	rdb := newLogStreamRedisClient().Subscribe(logStreamCtx, "redis-log-stream-pubsub-channel")
	defer rdb.Close()

	for {
		msg, err := rdb.ReceiveMessage(logStreamCtx)
		if err != nil {
			return "", err
		}

		return msg.Payload, nil
	}
}

// User-specific methods

func (l *logStreamRepository) SetLogsInRedisForUser(userID, logStream string) error {
	rdb := newLogStreamRedisClient()
	userLogKey := "logs:" + userID
	userChannelKey := "redis-log-stream-pubsub-channel:" + userID

	if err := rdb.Set(logStreamCtx, userLogKey, logStream, 0).Err(); err != nil {
		return err
	}

	if err := rdb.Publish(logStreamCtx, userChannelKey, logStream).Err(); err != nil {
		return err
	}

	return nil
}

func (l *logStreamRepository) GetLogsFromRedisForUser(userID string) (string, error) {
	rdb := newLogStreamRedisClient()
	userLogKey := "logs:" + userID
	return rdb.Get(logStreamCtx, userLogKey).Result()
}

func (l *logStreamRepository) WaitForLogsChangeForUser(userID string) (string, error) {
	rdb := newLogStreamRedisClient()
	userChannelKey := "redis-log-stream-pubsub-channel:" + userID
	pubsub := rdb.Subscribe(logStreamCtx, userChannelKey)
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(logStreamCtx)
		if err != nil {
			return "", err
		}

		return msg.Payload, nil
	}
}
