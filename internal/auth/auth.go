package auth

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"golang.org/x/exp/slog"
)

type Auth struct {
	Cred *azidentity.DefaultAzureCredential
}

func NewAuth() *Auth {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		log.Fatalf("Failed to initialize auth: %v", err)
	}
	return &Auth{Cred: cred}
}

func (a *Auth) GetARMAccessToken() (string, error) {
	accessToken, err := a.Cred.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return "", err
	}
	return accessToken.Token, nil
}

func (a *Auth) GetStorageAccessToken() (string, error) {
	accessToken, err := a.Cred.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://storage.azure.com/.default"},
	})
	if err != nil {
		return "", err
	}
	return accessToken.Token, nil
}

func (a *Auth) GetStorageAccountKey(subscriptionId string, resourceGroup string, storageAccountName string) (string, error) {
	client, err := armstorage.NewAccountsClient(subscriptionId, a.Cred, nil)
	if err != nil {
		slog.Error("not able to create client factory to get storage account key", err)
		return "", err
	}

	resp, err := client.ListKeys(context.Background(), resourceGroup, storageAccountName, nil)
	if err != nil {
		slog.Error("not able to get storage account key", err)
		return "", err
	}

	if len(resp.Keys) == 0 {
		slog.Error("no storage account key found")
		return "", nil
	}

	return *resp.Keys[0].Value, nil
}
