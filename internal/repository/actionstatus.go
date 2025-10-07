package repository

import (
	"context"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"

	"github.com/redis/go-redis/v9"
)

type actionStatusRepository struct{}

func NewActionStatusRepository() entity.ActionStatusRepository {
	return &actionStatusRepository{}
}

func newActionStatusRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func (a *actionStatusRepository) GetActionStatus(ctx context.Context) (string, error) {
	rdb := newActionStatusRedisClient()
	return rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-actionstatus").Result()
}

func (a *actionStatusRepository) SetActionStatus(ctx context.Context, val string) error {
	rdb := newActionStatusRedisClient()

	// Set the value in redis.
	if err := rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-actionstatus", val, 0).Err(); err != nil {
		return err
	}

	// Publish the value to the pubsub channel.
	if err := rdb.Publish(ctx, helper.GetUserIDFromContext(ctx)+"-redis-action-status-pubsub-channel", val).Err(); err != nil {
		return err
	}

	return nil
}

func (a *actionStatusRepository) WaitForActionStatusChange(ctx context.Context) (string, error) {
	rdb := newActionStatusRedisClient().Subscribe(ctx, helper.GetUserIDFromContext(ctx)+"-redis-action-status-pubsub-channel")
	defer rdb.Close()

	for {
		msg, err := rdb.ReceiveMessage(ctx)
		if err != nil {
			return "", err
		}

		return msg.Payload, nil
	}
}

func (a *actionStatusRepository) SetTerraformOperation(ctx context.Context, val string) error {
	rdb := newActionStatusRedisClient()
	if err := rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-terraform-operation", val, 0).Err(); err != nil {
		return err
	}

	return rdb.Publish(ctx, helper.GetUserIDFromContext(ctx)+"-redis-terraform-operation-pubsub-channel", val).Err()
}

func (a *actionStatusRepository) GetTerraformOperation(ctx context.Context) (string, error) {
	rdb := newActionStatusRedisClient()
	return rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-terraform-operation").Result()
}

func (a *actionStatusRepository) WaitForTerraformOperationChange(ctx context.Context) (string, error) {
	rdb := newActionStatusRedisClient().Subscribe(ctx, helper.GetUserIDFromContext(ctx)+"-redis-terraform-operation-pubsub-channel")
	defer rdb.Close()

	for {
		msg, err := rdb.ReceiveMessage(ctx)
		if err != nil {
			return "", err
		}

		return msg.Payload, nil
	}
}

func (a *actionStatusRepository) SetServerNotification(ctx context.Context, val string) error {
	rdb := newActionStatusRedisClient()
	if err := rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-server-notification", val, 0).Err(); err != nil {
		return err
	}

	return rdb.Publish(ctx, helper.GetUserIDFromContext(ctx)+"-redis-server-notification-pubsub-channel", val).Err()
}

func (a *actionStatusRepository) GetServerNotification(ctx context.Context) (string, error) {
	rdb := newActionStatusRedisClient()
	return rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-server-notification").Result()
}

func (a *actionStatusRepository) WaitForServerNotificationChange(ctx context.Context) (string, error) {
	rdb := newActionStatusRedisClient().Subscribe(ctx, helper.GetUserIDFromContext(ctx)+"-redis-server-notification-pubsub-channel")
	defer rdb.Close()

	for {
		msg, err := rdb.ReceiveMessage(ctx)
		if err != nil {
			return "", err
		}

		return msg.Payload, nil
	}
}
