package repository

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"

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

var preferenceCtx = context.Background()

func newPreferenceRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func (p *preferenceRepository) GetPreferenceFromBlob(storageAccountName string) (string, error) {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName)

	// Create a new Blob Service Client with the AAD credential
	client, err := azblob.NewClient(serviceURL, p.auth.Cred, nil)
	if err != nil {
		slog.Debug("not able to create blob client",
			slog.String("serviceURL", serviceURL),
			slog.String("error", err.Error()),
		)
		return "", err
	}

	// Download the blob
	ctx := context.Background()
	downloadResponse, err := client.DownloadStream(ctx, "tfstate", "preference.json", nil)
	if err != nil {
		slog.Debug("not able to download stream",
			slog.String("containerName", "tfstate"),
			slog.String("blobName", "preference.json"),
			slog.String("error", err.Error()),
		)
		return "", err
	}
	defer downloadResponse.Body.Close()

	// Read the blob content
	actualBlobData, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		slog.Debug("not able to read all from download response",
			slog.String("error", err.Error()),
		)
		return "", err
	}

	return string(actualBlobData), nil
}

func (p *preferenceRepository) PutPreferenceInBlob(val string, storageAccountName string) error {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName)

	// Create a new Blob Service Client with the AAD credential
	client, err := azblob.NewClient(serviceURL, p.auth.Cred, nil)
	if err != nil {
		slog.Debug("not able to create blob client",
			slog.String("serviceURL", serviceURL),
			slog.String("error", err.Error()),
		)
		return err
	}

	// Upload the blob
	ctx := context.Background()
	_, err = client.UploadBuffer(ctx, "tfstate", "preference.json", []byte(val), nil)
	if err != nil {
		slog.Debug("not able to upload buffer",
			slog.String("containerName", "tfstate"),
			slog.String("blobName", "preference.json"),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func (p *preferenceRepository) GetPreferenceFromRedis() (string, error) {
	rdb := newPreferenceRedisClient()
	return rdb.Get(preferenceCtx, "preference").Result()
}

func (p *preferenceRepository) PutPreferenceInRedis(val string) error {
	rdb := newPreferenceRedisClient()
	return rdb.Set(preferenceCtx, "preference", val, 0).Err()
}
