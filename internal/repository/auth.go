package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/redis/go-redis/v9"
)

type authRepository struct {
	config *config.Config
	auth   *auth.Auth
	rdb    *redis.Client
}

func NewAuthRepository(config *config.Config, auth *auth.Auth, rdb *redis.Client) entity.AuthRepository {
	return &authRepository{
		config: config,
		auth:   auth,
		rdb:    rdb,
	}
}

func (a *authRepository) GetSubscriptionDetails() (*armsubscription.Subscription, error) {

	// check if subscription id is already set in redis
	subscription, ok := a.getSubscriptionFromRedis()
	if ok {
		return subscription, nil
	}

	ctx := context.Background()
	clientFactory, err := armsubscription.NewClientFactory(a.auth.Cred, nil)
	if err != nil {
		return nil, fmt.Errorf("not able to create subscription client factory: %v", err)
	}

	pager := clientFactory.NewSubscriptionsClient().NewListPager(nil)
	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to advance page: %v", err)
		}
		for _, sub := range page.Value {
			// slog.Debug("Subscription ID:" + *sub.SubscriptionID)
			// slog.Debug("Looking for subscription ID:" + a.config.SubscriptionID)
			if *sub.SubscriptionID == a.config.SubscriptionID {
				a.addSubscriptionToRedis(sub) // add subscription to redis
				return sub, nil
			}
		}
	}

	return nil, fmt.Errorf("subscription not found")
}

// Get subscription from redis, return ok if found
func (a *authRepository) getSubscriptionFromRedis() (*armsubscription.Subscription, bool) {
	subscription, err := a.rdb.Get(context.Background(), "subscription").Result()
	if err == nil {
		logging.LogDebug(context.Background(), "subscription found in redis")
		var sub armsubscription.Subscription
		err = json.Unmarshal([]byte(subscription), &sub)
		if err != nil {
			logging.LogError(context.Background(), "failed to unmarshal subscription", "error", err)
			return nil, false
		}
		return &sub, true
	}

	return nil, false
}

func (a *authRepository) addSubscriptionToRedis(subscription *armsubscription.Subscription) error {
	subscriptionJson, err := json.Marshal(subscription)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}
	err = a.rdb.Set(context.Background(), "subscription", subscriptionJson, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set subscription in redis: %w", err)
	}
	return nil
}
