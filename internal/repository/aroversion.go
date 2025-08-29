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

func (a *aroVersionRepository) GetAROVersions(location string) (string, error) {
	slog.Info("Getting ARO versions for location " + location)

	// Check if the orchestrator versions are already cached in Redis
	aroVersions, err := a.rdb.Get(context.Background(), "aroVersions").Result()
	if err == nil {
		return aroVersions, nil
	}

	accessToken, err := a.auth.GetARMAccessToken()
	if err != nil {
		return "", err
	}

	// Make HTTP request to retrieve ARO versions
	url := fmt.Sprintf(a.appConfig.AroVersionApiUrlTemplate, a.appConfig.SubscriptionID, location)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: time.Second * time.Duration(a.appConfig.HttpRequestTimeoutSeconds),
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// Set the response body in Redis
	err = a.rdb.Set(context.Background(), "aroVersions", string(body), 0).Err()
	if err != nil {
		slog.Error("failed to set aro versions in redis", err)
	}

	return string(body), nil
}
