package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type storageAccountRepository struct {
	auth           *auth.Auth
	rdb            *redis.Client
	subscriptionId string
}

func NewStorageAccountRepository(auth *auth.Auth, rdb *redis.Client, config *config.Config) entity.StorageAccountRepository {
	return &storageAccountRepository{
		auth:           auth,
		rdb:            rdb,
		subscriptionId: config.SubscriptionID,
	}
}

// https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/list-by-resource-group?view=rest-storagerp-2023-01-01&tabs=Go
func (s *storageAccountRepository) GetStorageAccount() (armstorage.Account, error) {

	// Check if the storage account is already cached in Redis
	storageAccountStr, err := s.rdb.Get(context.Background(), "storageAccount").Result()
	if err == nil {
		var storageAccount armstorage.Account
		err = json.Unmarshal([]byte(storageAccountStr), &storageAccount)
		if err != nil {
			return armstorage.Account{}, err
		}
		return storageAccount, nil
	}

	clientFactory, err := armstorage.NewClientFactory(s.subscriptionId, s.auth.Cred, nil)
	if err != nil {
		slog.Error("not able to create client factory to get storage account", err)
		return armstorage.Account{}, err
	}

	pager := clientFactory.NewAccountsClient().NewListByResourceGroupPager("repro-project", nil)
	for pager.More() {
		page, err := pager.NextPage(context.Background())
		if err != nil {
			slog.Error("not able to get next page for storage account", err)
			return armstorage.Account{}, err
		}
		for _, account := range page.Value {
			// Cache storage account in Redis
			storageAccountStr, err := json.Marshal(account)
			if err != nil {
				slog.Error("not able to marshal storage account", err)
			}
			err = s.rdb.Set(context.Background(), "storageAccount", storageAccountStr, 0).Err()
			if err != nil {
				slog.Error("not able to set storage account in redis", err)
			}

			return *account, nil // return the first storage account found.
		}
	}

	return armstorage.Account{}, errors.New("storage account not found in resource group repro-project")
}

// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob@v1.2.0/lease#BlobClient.BreakLease
func (s *storageAccountRepository) BreakBlobLease(storageAccountName string, containerName string, blobName string) error {
	slog.Debug("breaking blob lease for blob", blobName+" in container "+containerName+" in storage account "+storageAccountName)

	accountKey, err := s.auth.GetStorageAccountKey(s.subscriptionId, "repro-project", storageAccountName)
	if err != nil {
		return fmt.Errorf("failed to get storage account key: %w", err)
	}

	leaseBlobClient, err := s.createLeaseBlobClient(storageAccountName, accountKey, containerName, blobName)
	if err != nil {
		return fmt.Errorf("not able to create lease blob client: %w", err)
	}

	_, err = leaseBlobClient.BreakLease(context.Background(), &lease.BlobBreakOptions{
		BreakPeriod: to.Ptr(int32(0)),
	})
	if err != nil {
		return fmt.Errorf("failed to break blob lease: %w", err)
	}

	return nil
}

func (s *storageAccountRepository) createLeaseBlobClient(storageAccountName string, accountKey string, containerName string, blobName string) (*lease.BlobClient, error) {
	cred, err := azblob.NewSharedKeyCredential(storageAccountName, accountKey)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", storageAccountName, containerName, blobName)

	blobClient, err := blob.NewClientWithSharedKeyCredential(url, cred, nil)
	if err != nil {
		return nil, err
	}

	leaseBlobClient, err := lease.NewBlobClient(blobClient, nil)
	if err != nil {
		return nil, err
	}

	return leaseBlobClient, nil
}
