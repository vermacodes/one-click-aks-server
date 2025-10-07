package repository

import (
	"context"
	"fmt"
	"io"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/cache"
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
	rdb       *redis.Client
}

func NewPreferenceRepository(auth *auth.Auth, appConfig *config.Config) entity.PreferenceRepository {
	return &preferenceRepository{
		auth:      auth,
		appConfig: appConfig,
		rdb:       cache.NewRedisClient(),
	}
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
			"serviceURL", serviceURL,
			"error", err.Error(),
		)
		return "", err
	}

	userAlias := helper.GetUserAliasFromContext(ctx)

	// Download the blob
	downloadResponse, err := client.DownloadStream(ctx, "repro-project-preferences", userAlias+"-preference.json", nil)
	if err != nil {
		logging.LogError(ctx, "not able to download stream",
			"containerName", "repro-project-preferences",
			"blobName", userAlias+"-preference.json",
			"error", err.Error(),
		)
		return "", err
	}
	defer downloadResponse.Body.Close()

	// Read the blob content
	actualBlobData, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		logging.LogError(ctx, "not able to read all from download response",
			"error", err.Error(),
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
			"serviceURL", serviceURL,
			"error", err.Error(),
		)
		return err
	}

	userAlias := helper.GetUserAliasFromContext(ctx)

	// Upload the blob
	_, err = client.UploadBuffer(ctx, "repro-project-preferences", userAlias+"-preference.json", []byte(val), nil)
	if err != nil {
		logging.LogError(ctx, "not able to upload buffer",
			"containerName", "repro-project-preferences",
			"blobName", userAlias+"-preference.json",
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (p *preferenceRepository) GetPreferenceFromRedis(ctx context.Context) (string, error) {
	return p.rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-preference").Result()
}

func (p *preferenceRepository) PutPreferenceInRedis(ctx context.Context, val string) error {
	return p.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-preference", val, 0).Err()
}

func (p *preferenceRepository) DeletePreferenceFromRedis(ctx context.Context) error {
	return p.rdb.Del(ctx, helper.GetUserIDFromContext(ctx)+"-preference").Err()
}

func (p *preferenceRepository) DeleteKubernetesVersionsFromRedis(ctx context.Context) error {
	return p.rdb.Del(ctx, helper.GetUserIDFromContext(ctx)+"-kubernetesVersions").Err()
}
