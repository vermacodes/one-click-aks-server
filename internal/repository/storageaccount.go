package repository

import (
	"context"
	"fmt"
	"strings"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/blob"
	"github.com/Azure/azure-sdk-for-go/sdk/storage/azblob/lease"
	"github.com/redis/go-redis/v9"
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
func (s *storageAccountRepository) GetStorageAccountName(ctx context.Context) (string, error) {

	return s.config.ActLabsHubStorageAccountName, nil
}

// https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/storage/azblob@v1.2.0/lease#BlobClient.BreakLease
func (s *storageAccountRepository) BreakBlobLease(ctx context.Context, storageAccountName string, containerName string, blobName string) error {
	// Append user alias to blob name
	blobName = helper.GetUserAliasFromContext(ctx) + "-" + blobName

	if s.config.UseMsi {
		return s.BreakBlobLeaseUsingMsi(ctx, storageAccountName, containerName, blobName)
	}

	accountKey, err := s.auth.GetStorageAccountKey(s.config.ActLabsHubSubscriptionID, s.config.ActLabsHubResourceGroupName, storageAccountName)
	if err != nil {
		return fmt.Errorf("failed to get storage account key: %w", err)
	}

	leaseBlobClient, err := s.createLeaseBlobClient(storageAccountName, accountKey, containerName, blobName)
	if err != nil {
		return fmt.Errorf("not able to create lease blob client: %w", err)
	}

	_, err = leaseBlobClient.BreakLease(ctx, &lease.BlobBreakOptions{
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

// Break blob lease for the given blob name in the given container in the given storage account
// using MSI authentication
func (s *storageAccountRepository) BreakBlobLeaseUsingMsi(ctx context.Context, storageAccountName string, containerName string, blobName string) error {
	url := fmt.Sprintf("https://%s.blob.core.windows.net/%s/%s", storageAccountName, containerName, blobName)

	blobClient, err := blob.NewClient(url, s.auth.Cred, nil)
	if err != nil {
		return fmt.Errorf("not able to create blob client: %w", err)
	}

	leaseBlobClient, err := lease.NewBlobClient(blobClient, nil)
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

func UserAliasForStorageAccount(ctx context.Context, userPrincipalName string) string {
	// change to lowercase
	userPrincipalName = strings.ToLower(userPrincipalName)

	// drop the domain
	userPrincipalName = strings.Split(userPrincipalName, "@")[0]

	// drop the suffix `v-`
	userPrincipalName = strings.TrimPrefix(userPrincipalName, "v-")

	return userPrincipalName
}
