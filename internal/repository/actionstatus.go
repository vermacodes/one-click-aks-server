package repository

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"one-click-aks-server/internal/cache"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"

	"github.com/redis/go-redis/v9"
)

type actionStatusRepository struct {
	rdb *redis.Client
}

func NewActionStatusRepository() entity.ActionStatusRepository {
	repo := &actionStatusRepository{
		rdb: cache.NewRedisClient(),
	}

	// Enable key expiration notifications and start listener
	go repo.listenForKeyExpirations()

	return repo
}

func (a *actionStatusRepository) listenForKeyExpirations() {
	ctx := context.Background()

	// Enable key expiration notifications in Redis
	a.rdb.ConfigSet(ctx, "notify-keyspace-events", "Ex")

	// Subscribe to key expiration events
	pubsub := a.rdb.PSubscribe(ctx, "__keyevent@*__:expired")
	defer pubsub.Close()

	for msg := range pubsub.Channel() {
		expiredKey := msg.Payload

		// Check if this is an action status key
		if strings.HasSuffix(expiredKey, "-actionstatus") {
			userID := strings.TrimSuffix(expiredKey, "-actionstatus")

			// Create expired action status
			expiredStatus := entity.ActionStatus{InProgress: false}
			expiredVal, _ := json.Marshal(expiredStatus)

			// Publish to the action status channel
			channelName := userID + "-redis-action-status-pubsub-channel"
			a.rdb.Publish(ctx, channelName, string(expiredVal))
		}
	}
}

func (a *actionStatusRepository) GetActionStatus(ctx context.Context) (string, error) {
	return a.rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-actionstatus").Result()
}

func (a *actionStatusRepository) SetActionStatus(ctx context.Context, val string) error {
	logging.LogDebug(ctx, "called set action status repository", "action_status_value", val)
	// Parse the action status to check if it's in progress
	var actionStatus entity.ActionStatus
	var ttl time.Duration = 0 // Default: no expiration

	if err := json.Unmarshal([]byte(val), &actionStatus); err == nil {
		if actionStatus.InProgress {
			ttl = 30 * time.Second // Set 30 second TTL only if in progress
		}
	}

	// Set the value in redis with conditional TTL
	if err := a.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-actionstatus", val, ttl).Err(); err != nil {
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
