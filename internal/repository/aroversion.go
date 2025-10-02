package repository

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type aroVersionRepository struct {
	auth      *auth.Auth
	rdb       *redis.Client
	appConfig *config.Config
}

func NewAROVersionRepository(appConfig *config.Config, auth *auth.Auth, rdb *redis.Client) entity.AROVersionRepository {
	return &aroVersionRepository{
		auth:      auth,
		rdb:       rdb,
		appConfig: appConfig,
	}
}

func (a *aroVersionRepository) GetAROVersions(ctx context.Context, location string) (string, error) {

	// Check if the orchestrator versions are already cached in Redis
	aroVersions, err := a.rdb.Get(ctx, "aroVersions-"+location).Result()
	if err == nil {
		return aroVersions, nil
	}

	accessToken, err := a.auth.GetARMAccessToken()
	if err != nil {
		logging.LogError(ctx, "not able to get arm access token",
			slog.Any("error", err),
		)
		return "", err
	}

	// Make HTTP request to retrieve ARO versions
	url := fmt.Sprintf(a.appConfig.AroVersionApiUrlTemplate, a.appConfig.SubscriptionID, location)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logging.LogError(ctx, "not able to make http get request to get ARO versions",
			slog.Any("error", err),
		)
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: time.Second * time.Duration(a.appConfig.HttpRequestTimeoutSeconds),
	}

	resp, err := client.Do(req)
	if err != nil {
		logging.LogError(ctx, "not able to read http response",
			slog.Any("error", err),
		)
		return "", err
	}
	defer resp.Body.Close()

	// Check if the response status code indicates success
	if resp.StatusCode != http.StatusOK {
		logging.LogError(ctx, "http request was not successful",
			slog.Any("error", err),
		)
		return "", fmt.Errorf("failed to get ARO versions: status code %d", resp.StatusCode)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.LogError(ctx, "not able to read response body",
			slog.Any("error", err),
		)
		return "", err
	}

	// Set the response body in Redis
	err = a.rdb.Set(ctx, "aroVersions-"+location, string(body), 0).Err()
	if err != nil {
		logging.LogError(ctx, "failed to set aro versions in redis",
			slog.Any("error", err),
		)
	}

	return string(body), nil
}
