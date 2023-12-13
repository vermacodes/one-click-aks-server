package repository

import (
	"context"
	"encoding/json"
	"errors"
	"os/exec"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"

	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
)

type storageAccountRepository struct {
	cred           *azidentity.DefaultAzureCredential
	rdb            *redis.Client
	subscriptionId string
}

func NewStorageAccountRepository(auth *auth.Auth, rdb *redis.Client, config *config.Config) entity.StorageAccountRepository {
	return &storageAccountRepository{
		cred:           auth.Cred,
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

	clientFactory, err := armstorage.NewClientFactory(s.subscriptionId, s.cred, nil)
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

// var storageAccountCtx = context.Background()

// func newStorageAccountRedisClient() *redis.Client {
// 	return redis.NewClient(&redis.Options{
// 		Addr:     "localhost:6379",
// 		Password: "", // no password set
// 		DB:       0,  // use default DB
// 	})
// }

// // This returns the name of the storage account after running azure cli command.
// func (s *storageAccountRepository) GetStorageAccountName() (string, error) {

// 	out, err := exec.Command("bash", "-c", "az storage account list -g repro-project --output tsv --query [].name").Output()
// 	if err != nil {
// 		return "", err
// 	}

// 	return strings.TrimSuffix(string(out), "\n"), nil
// }

// // This returns storage account name from Redis.
// func (s *storageAccountRepository) GetStorageAccountNameFromRedis() (string, error) {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Get(storageAccountCtx, "storageAccountName").Result()
// }

// // This sets storage account name in redis.
// func (s *storageAccountRepository) SetStorageAccountNameInRedis(val string) error {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Set(storageAccountCtx, "storageAccountName", val, 0).Err()
// }

// func (s *storageAccountRepository) DelStorageAccountNameFromRedis() error {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Del(storageAccountCtx, "storageAccountName").Err()
// }

// // Blob Container
// func (s *storageAccountRepository) GetBlobContainer(storageAccountName string, containerName string) (string, error) {
// 	out, err := exec.Command("bash", "-c", "az storage container show -n "+containerName+" --account-name "+storageAccountName+" --output json").Output()
// 	return string(out), err
// }

// func (s *storageAccountRepository) GetBlobContainerFromRedis() (string, error) {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Get(storageAccountCtx, "blobcontainer").Result()
// }

// func (s *storageAccountRepository) SetBlobContainerInRedis(val string) error {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Set(storageAccountCtx, "blobcontainer", val, 0).Err()
// }

// func (s *storageAccountRepository) DelBlobContainerFromRedis() error {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Del(storageAccountCtx, "blobcontainer").Err()
// }

// // Resource Group
// func (s *storageAccountRepository) GetResourceGroup() (string, error) {
// 	out, err := exec.Command("bash", "-c", "az group show --name repro-project --output json").Output()
// 	return string(out), err
// }

// func (s *storageAccountRepository) GetResourceGroupFromRedis() (string, error) {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Get(storageAccountCtx, "resourcegroup").Result()
// }

// func (s *storageAccountRepository) SetResourceGroupInRedis(val string) error {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Set(storageAccountCtx, "resourcegroup", val, 0).Err()
// }

// func (s *storageAccountRepository) DelResourceGroupFromRedis() error {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Del(storageAccountCtx, "resourcegroup").Err()
// }

// // Storage Account
// func (s *storageAccountRepository) GetStorageAccount(storageAccountName string) (string, error) {
// 	out, err := exec.Command("bash", "-c", "az storage account show -g repro-project --name "+storageAccountName+" --output json").Output()
// 	return string(out), err
// }

// func (s *storageAccountRepository) GetStorageAccountFromRedis() (string, error) {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Get(storageAccountCtx, "storageaccount").Result()
// }

// func (s *storageAccountRepository) SetStorageAccountInRedis(val string) error {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Set(storageAccountCtx, "storageaccount", val, 0).Err()
// }

// func (s *storageAccountRepository) DelStorageAccountFromRedis() error {
// 	rdb := newStorageAccountRedisClient()
// 	return rdb.Del(storageAccountCtx, "storageaccount").Err()
// }

// func (s *storageAccountRepository) CreateResourceGroup() (string, error) {
// 	out, err := exec.Command("bash", "-c", "az group create -l eastus -n repro-project -o json").Output()
// 	return string(out), err
// }

// func (s *storageAccountRepository) CreateStorageAccount(storageAccountName string) (string, error) {
// 	out, err := exec.Command("bash", "-c", "az storage account create -g repro-project --name "+storageAccountName+" --kind StorageV2 --sku Standard_LRS --output json").Output()
// 	return string(out), err
// }

// func (s *storageAccountRepository) CreateBlobContainer(storageAccountName string, containerName string) (string, error) {
// 	out, err := exec.Command("bash", "-c", "az storage container create -n "+containerName+" -g repro-project --account-name "+storageAccountName+" --output tsv --query created").Output()
// 	return string(out), err
// }

func (s *storageAccountRepository) BreakBlobLease(storageAccountName string, containerName string, blobName string) error {
	_, err := exec.Command("bash", "-c", "az storage blob lease break -c "+containerName+" -b "+blobName+" --account-name "+storageAccountName+" --output tsv").Output()
	return err
}
