package repository

import (
	"context"
	"fmt"
	"strings"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type storageAccountRepository struct {
	auth   *auth.Auth
	rdb    *redis.Client
	config *config.Config
}

func NewStorageAccountRepository(auth *auth.Auth, rdb *redis.Client, config *config.Config) entity.StorageAccountRepository {
	return &storageAccountRepository{
		auth:   auth,
		rdb:    rdb,
		config: config,
	}
}

// https://learn.microsoft.com/en-us/rest/api/storagerp/storage-accounts/list-by-resource-group?view=rest-storagerp-2023-01-01&tabs=Go
func (s *storageAccountRepository) GetStorageAccountName() (string, error) {

	return s.config.ActLabsHubStorageAccountName, nil
	// slog.Debug("getting storage account name from resource group")

	// // Check if the storage account is already cached in Redis
	// storageAccountName, err := s.rdb.Get(context.Background(), "storageAccount").Result()
	// if err == nil {
	// 	slog.Debug("storage account found in redis")
	// 	return storageAccountName, nil
	// }

	// slog.Debug("storage account not found in redis")

	// clientFactory, err := armstorage.NewClientFactory(s.config.SubscriptionID, s.auth.Cred, nil)
	// if err != nil {
	// 	slog.Error("not able to create client factory to get storage account", err)
	// 	return "", err
	// }

	// pager := clientFactory.NewAccountsClient().NewListByResourceGroupPager("repro-project", nil)
	// for pager.More() {
	// 	page, err := pager.NextPage(context.Background())
	// 	if err != nil {
	// 		slog.Error("not able to get next page for storage account", err)
	// 		return "", err
	// 	}
	// 	for _, account := range page.Value {

	// 		// Check if the storage account name is the user's alias
	// 		if !strings.HasPrefix(*account.Name, UserAliasForStorageAccount(s.config.ArmUserPrincipalName)) {
	// 			continue
	// 		}

	// 		err = s.rdb.Set(context.Background(), "storageAccount", account.Name, 0).Err()
	// 		if err != nil {
	// 			slog.Error("not able to set storage account in redis", err)
	// 		}

	// 		return *account.Name, nil // return the storage account found.
	// 	}
	// }

	// return "", errors.New("storage account not found in resource group repro-project")
}

// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob@v1.2.0/lease#BlobClient.BreakLease
func (s *storageAccountRepository) BreakBlobLease(storageAccountName string, containerName string, blobName string) error {
	slog.Debug("breaking blob lease for blob", blobName+" in container "+containerName+" in storage account "+storageAccountName)

	// Append user alias to blob name
	blobName = s.config.UserAlias + "-" + blobName

	accountKey, err := s.auth.GetStorageAccountKey(s.config.ActLabsHubSubscriptionID, s.config.ActLabsHubResourceGroupName, storageAccountName)
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

func UserAliasForStorageAccount(userPrincipalName string) string {
	// change to lowercase
	userPrincipalName = strings.ToLower(userPrincipalName)

	// drop the domain
	userPrincipalName = strings.Split(userPrincipalName, "@")[0]

	// drop the suffix `v-`
	userPrincipalName = strings.TrimPrefix(userPrincipalName, "v-")

	return userPrincipalName
}
