package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
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

func (a *authRepository) GetSubscriptionDetails(ctx context.Context) (*armsubscription.Subscription, error) {

	// check if subscription id is already set in redis
	subscription, ok := a.getSubscriptionFromRedis(ctx)
	if ok {
		return subscription, nil
	}

	subscriptionId, err := a.GetSubscriptionId(ctx)
	if err != nil {
		return nil, fmt.Errorf("not able to get subscription id: %v", err)
	}

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
			if *sub.SubscriptionID == subscriptionId {
				a.addSubscriptionToRedis(ctx, sub) // add subscription to redis
				return sub, nil
			}
		}
	}

	return nil, fmt.Errorf("subscription not found")
}

// Get subscription ID from server registration
func (a *authRepository) GetSubscriptionId(ctx context.Context) (string, error) {
	// Get subscription id from redis
	userId := helper.GetUserIDFromContext(ctx)

	// Check if user ID is valid
	if userId == "" || userId == "unknown-user" {
		logging.LogError(ctx, "GetSubscriptionId called without valid user ID in context",
			"userID", userId)
		return "", fmt.Errorf("authentication required: user ID not found in context")
	}

	subscriptionId, err := a.rdb.Get(ctx, userId+"-subscription-id").Result()
	if err == nil {
		logging.LogDebug(context.Background(), "subscription id found in redis")
		return subscriptionId, nil
	}

	actlabsAuthEndpoint := a.config.ActlabsHubURLInternal
	// http call to actlabs-auth
	req, err := http.NewRequest("GET", actlabsAuthEndpoint+"arm/server/"+userId, nil)
	if err != nil {
		logging.LogError(ctx, "error creating new http request")
		return "", err
	}

	armAccessToken, err := a.auth.GetARMAccessToken()
	if err != nil {
		logging.LogError(ctx, "error getting arm access token", "error", err)
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+armAccessToken)
	req.Header.Set("x-ms-client-principal-name", helper.GetUserIDFromContext(ctx))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("ProtectedLabSecret", entity.ProtectedLabSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logging.LogError(ctx, "not able to make http call to get protected lab",
			slog.Any("error", err),
		)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.LogError(ctx, "not able to make http call successfully",
			slog.Any("http_code", resp.StatusCode),
			slog.Any("error", err),
		)
		return "", fmt.Errorf("not able to get protected lab, received error code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.LogError(ctx, "not able to read response body",
			slog.Any("error", err),
		)
		return "", err
	}

	// we are expecting the body to be like
	// {
	//   "id": "subscription-id-value"
	// }
	//
	// get id from body and return it

	var response struct {
		ID string `json:"id"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		logging.LogError(ctx, "failed to unmarshal subscription response", "error", err)
		return "", err
	}

	if response.ID == "" {
		logging.LogError(ctx, "subscription id not found in response")
		return "", fmt.Errorf("subscription id not found in response")
	}

	// Add subscription id to redis.
	err = a.rdb.Set(ctx, userId+"-subscription-id", response.ID, 0).Err()
	if err != nil {
		logging.LogError(ctx, "not able to set subscription id in redis", "error", err)
	}

	return response.ID, nil
}

// Get subscription from redis, return ok if found
func (a *authRepository) getSubscriptionFromRedis(ctx context.Context) (*armsubscription.Subscription, bool) {
	userId := helper.GetUserIDFromContext(ctx)

	// Check if user ID is valid
	if userId == "" || userId == "unknown-user" {
		logging.LogError(ctx, "getSubscriptionFromRedis called without valid user ID in context",
			"userID", userId)
		return nil, false
	}

	subscription, err := a.rdb.Get(ctx, userId+"-subscription").Result()
	if err == nil {
		logging.LogDebug(ctx, "subscription found in redis")
		var sub armsubscription.Subscription
		err = json.Unmarshal([]byte(subscription), &sub)
		if err != nil {
			logging.LogError(ctx, "failed to unmarshal subscription", "error", err)
			return nil, false
		}
		return &sub, true
	}

	return nil, false
}

func (a *authRepository) addSubscriptionToRedis(ctx context.Context, subscription *armsubscription.Subscription) error {
	subscriptionJson, err := json.Marshal(subscription)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription: %w", err)
	}
	err = a.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-subscription", subscriptionJson, 0).Err()
	if err != nil {
		return fmt.Errorf("failed to set subscription in redis: %w", err)
	}
	return nil
}
