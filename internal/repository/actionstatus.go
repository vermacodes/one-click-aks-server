package repository

import (
	"context"

	"one-click-aks-server/internal/cache"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"

	"github.com/redis/go-redis/v9"
)

type actionStatusRepository struct {
	rdb *redis.Client
}

func NewActionStatusRepository() entity.ActionStatusRepository {
	return &actionStatusRepository{
		rdb: cache.NewRedisClient(),
	}
}

func (a *actionStatusRepository) GetActionStatus(ctx context.Context) (string, error) {
	return a.rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-actionstatus").Result()
}

func (a *actionStatusRepository) SetActionStatus(ctx context.Context, val string) error {
	// Set the value in redis.
	if err := a.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-actionstatus", val, 0).Err(); err != nil {
		return err
	}

	// Publish the value to the pubsub channel.
	if err := a.rdb.Publish(ctx, helper.GetUserIDFromContext(ctx)+"-redis-action-status-pubsub-channel", val).Err(); err != nil {
		return err
	}

	return nil
}

func (a *actionStatusRepository) WaitForActionStatusChange(ctx context.Context) (string, error) {
	rdb := a.rdb.Subscribe(ctx, helper.GetUserIDFromContext(ctx)+"-redis-action-status-pubsub-channel")
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
	if err := a.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-terraform-operation", val, 0).Err(); err != nil {
		return err
	}

	return a.rdb.Publish(ctx, helper.GetUserIDFromContext(ctx)+"-redis-terraform-operation-pubsub-channel", val).Err()
}

func (a *actionStatusRepository) GetTerraformOperation(ctx context.Context) (string, error) {
	return a.rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-terraform-operation").Result()
}

func (a *actionStatusRepository) WaitForTerraformOperationChange(ctx context.Context) (string, error) {
	rdb := a.rdb.Subscribe(ctx, helper.GetUserIDFromContext(ctx)+"-redis-terraform-operation-pubsub-channel")
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
	if err := a.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-server-notification", val, 0).Err(); err != nil {
		return err
	}

	return a.rdb.Publish(ctx, helper.GetUserIDFromContext(ctx)+"-redis-server-notification-pubsub-channel", val).Err()
}

func (a *actionStatusRepository) GetServerNotification(ctx context.Context) (string, error) {
	return a.rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-server-notification").Result()
}

func (a *actionStatusRepository) WaitForServerNotificationChange(ctx context.Context) (string, error) {
	rdb := a.rdb.Subscribe(ctx, helper.GetUserIDFromContext(ctx)+"-redis-server-notification-pubsub-channel")
	defer rdb.Close()

	for {
		msg, err := rdb.ReceiveMessage(ctx)
		if err != nil {
			return "", err
		}

		return msg.Payload, nil
	}
}
