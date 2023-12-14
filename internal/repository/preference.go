package repository

import (
	"context"
	"fmt"
	"io"

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

	client, err := azblob.NewClient(serviceURL, p.auth.Cred, nil)
	if err != nil {
		return "", err
	}

	// Download the blob
	downloadResponse, err := client.DownloadStream(ctx, "tfstate", "preference.json", nil)
	if err != nil {
		return "", err
	}

	// Assert that the content is correct
	actualBlobData, err := io.ReadAll(downloadResponse.Body)
	if err != nil {
		return "", err
	}

	return string(actualBlobData), nil
}

func (p *preferenceRepository) PutPreferenceInBlob(val string, storageAccountName string) error {
	serviceURL := fmt.Sprintf("https://%s.blob.core.windows.net/", storageAccountName)

	client, err := azblob.NewClient(serviceURL, p.auth.Cred, nil)
	if err != nil {
		return err
	}

	_, err = client.UploadBuffer(ctx, "tfstate", "preference.json", []byte(val), nil)
	if err != nil {
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
