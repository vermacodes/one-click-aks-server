package auth

import (
	"context"
	"log"
	"one-click-aks-server/internal/config"
	"os"
	"os/exec"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"golang.org/x/exp/slog"
)

type Auth struct {
	Cred azcore.TokenCredential
}

func NewAuth(appConfig *config.Config) *Auth {
	var cred azcore.TokenCredential
	var err error

	if appConfig.UseMsi {
		cred, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ClientID(appConfig.AzureClientID),
		})
		if err != nil {
			log.Fatalf("Failed to initialize managed identity auth: %v", err)
		}
	} else {
		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			log.Fatalf("Failed to initialize default auth: %v", err)
		}
	}

	return &Auth{Cred: cred}
}

// login using msi
func AzureCLILoginByMSI(username string) {
	out, err := exec.Command("bash", "-c", "az login --identity --username "+username).Output()
	if err != nil {
		slog.Error("not able to login using msi", err)
		os.Exit(1)
	}

	slog.Info("az login --identity output: " + string(out))
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
