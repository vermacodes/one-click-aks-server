package repository

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"

	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/redis/go-redis/v9"
)

type preferenceRepository struct {
	auth      *auth.Auth
	appConfig *config.Config
}

func NewPreferenceRepository(auth *auth.Auth, appConfig *config.Config) entity.PreferenceRepository {
	return &preferenceRepository{
		auth:      auth,
		appConfig: appConfig,
	}
}

func newPreferenceRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func (p *preferenceRepository) GetPreferenceFromBlob(ctx context.Context, storageAccountName string) (string, error) {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName)

	// Use this for local emulator
	if storageAccountName == "devstoreaccount1" {
		serviceURL = "https://127.0.0.1:10000/devstoreaccount1/"
	}

	// Create a new Blob Service Client with the AAD credential
	client, err := azblob.NewClient(serviceURL, p.auth.Cred, nil)
	if err != nil {
		logging.LogError(ctx, "not able to create blob client",
			slog.String("serviceURL", serviceURL),
			slog.String("error", err.Error()),
		)
		return "", err
	}

	// Download the blob
	downloadResponse, err := client.DownloadStream(ctx, "repro-project-preferences", p.appConfig.UserAlias+"-preference.json", nil)
	if err != nil {
		logging.LogError(ctx, "not able to download stream",
			slog.String("containerName", "repro-project-preferences"),
			slog.String("blobName", p.appConfig.UserAlias+"-preference.json"),
			slog.String("error", err.Error()),
		)
		return "", err
	}
	defer downloadResponse.Body.Close()

	// Read the blob content
	actualBlobData, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		logging.LogError(ctx, "not able to read all from download response",
			slog.String("error", err.Error()),
		)
		return "", err
	}

	return string(actualBlobData), nil
}

func (p *preferenceRepository) PutPreferenceInBlob(ctx context.Context, val string, storageAccountName string) error {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName)

	// Use this for local emulator
	if storageAccountName == "devstoreaccount1" {
		serviceURL = "https://127.0.0.1:10000/devstoreaccount1/"
	}

	// Create a new Blob Service Client with the AAD credential
	client, err := azblob.NewClient(serviceURL, p.auth.Cred, nil)
	if err != nil {
		logging.LogError(ctx, "not able to create blob client",
			slog.String("serviceURL", serviceURL),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Upload the blob
	_, err = client.UploadBuffer(ctx, "repro-project-preferences", p.appConfig.UserAlias+"-preference.json", []byte(val), nil)
	if err != nil {
		logging.LogError(ctx, "not able to upload buffer",
			slog.String("containerName", "repro-project-preferences"),
			slog.String("blobName", p.appConfig.UserAlias+"-preference.json"),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (p *preferenceRepository) GetPreferenceFromRedis(ctx context.Context) (string, error) {
	rdb := newPreferenceRedisClient()
	return rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-preference").Result()
}

func (p *preferenceRepository) PutPreferenceInRedis(ctx context.Context, val string) error {
	rdb := newPreferenceRedisClient()
	return rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-preference", val, 0).Err()
}

func (p *preferenceRepository) DeletePreferenceFromRedis(ctx context.Context) error {
	rdb := newPreferenceRedisClient()
	return rdb.Del(ctx, helper.GetUserIDFromContext(ctx)+"-preference").Err()
}

func (p *preferenceRepository) DeleteKubernetesVersionsFromRedis(ctx context.Context) error {
	rdb := newPreferenceRedisClient()
	return rdb.Del(ctx, helper.GetUserIDFromContext(ctx)+"-kubernetesVersions").Err()
}
