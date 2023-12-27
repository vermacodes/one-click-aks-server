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

type kVersionRepository struct {
	auth      *auth.Auth
	rdb       *redis.Client
	appConfig *config.Config
}

func NewKVersionRepository(appConfig *config.Config, auth *auth.Auth, rdb *redis.Client) entity.KVersionRepository {
	return &kVersionRepository{
		auth:      auth,
		rdb:       rdb,
		appConfig: appConfig,
	}
}

func (k *kVersionRepository) GetOrchestrator(location string) (string, error) {
	slog.Info("Getting Kubernetes versions for location " + location)

	// Check if the orchestrator versions are already cached in Redis
	kubernetesVersions, err := k.rdb.Get(context.Background(), "kubernetesVersions").Result()
	if err == nil {
		return kubernetesVersions, nil
	}

	accessToken, err := k.auth.GetARMAccessToken()
	if err != nil {
		return "", err
	}

	// Make HTTP request to retrieve Kubernetes versions
	url := fmt.Sprintf(k.appConfig.KubernetesVersionApiUrlTemplate, k.appConfig.SubscriptionID, location)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{
		Timeout: time.Second * time.Duration(k.appConfig.HttpRequestTimeoutSeconds),
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
	err = k.rdb.Set(context.Background(), "kubernetesVersions", string(body), 0).Err()
	if err != nil {
		slog.Error("failed to set kubernetes versions in redis", err)
	}

	return string(body), nil
}
