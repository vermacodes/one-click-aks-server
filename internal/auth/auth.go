package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"

	"github.com/gorilla/websocket"
)

type Auth struct {
	Cred azcore.TokenCredential
}

func NewAuth(appConfig *config.Config) *Auth {
	var cred azcore.TokenCredential
	var err error

	if appConfig.UseServicePrincipal {

		ctx := context.Background()
		logging.LogDebug(ctx, "using service principal for auth")

		cred, err = azidentity.NewClientSecretCredential(appConfig.AzureTenantID, appConfig.AzureClientID, appConfig.AzureClientSecret, nil)
		if err != nil {
			log.Fatalf("Failed to initialize service principal auth: %v", err)
		}

		// AzureCLILoginByServicePrincipal(appConfig.AzureClientID, appConfig.AzureClientSecret, appConfig.SubscriptionID, appConfig.AzureTenantID)

	} else if appConfig.UseMsi {

		ctx := context.Background()
		logging.LogDebug(ctx, "using managed identity for auth")

		cred, err = azidentity.NewManagedIdentityCredential(&azidentity.ManagedIdentityCredentialOptions{
			ID: azidentity.ClientID(appConfig.AzureClientID),
		})

		if err != nil {
			log.Fatalf("Failed to initialize managed identity auth: %v", err)
		}

		// AzureCLILoginByMSI(appConfig.AzureClientID, appConfig.SubscriptionID)

	} else {

		ctx := context.Background()
		logging.LogDebug(ctx, "using default auth")

		cred, err = azidentity.NewDefaultAzureCredential(nil)
		if err != nil {
			log.Fatalf("Failed to initialize default auth: %v", err)
		}
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
	ctx := context.Background()
	client, err := armstorage.NewAccountsClient(subscriptionId, a.Cred, nil)
	if err != nil {
		logging.LogError(ctx, "not able to create client factory to get storage account key", "error", err)
		return "", err
	}

	resp, err := client.ListKeys(context.Background(), resourceGroup, storageAccountName, nil)
	if err != nil {
		logging.LogError(ctx, "not able to get storage account key", "error", err)
		return "", err
	}

	if len(resp.Keys) == 0 {
		logging.LogError(ctx, "no storage account key found")
		return "", nil
	}

	return *resp.Keys[0].Value, nil
}

// AuthenticateWebSocketConnection handles WebSocket authentication via messages
func AuthenticateWebSocketConnection(ws *websocket.Conn) (string, error) {
	// Set read timeout for authentication
	ws.SetReadDeadline(time.Now().Add(30 * time.Second))

	var authMsg entity.WSMessage
	if err := ws.ReadJSON(&authMsg); err != nil {
		return "", fmt.Errorf("failed to read auth message: %w", err)
	}

	if authMsg.Type != entity.WSMsgTypeAuth {
		return "", fmt.Errorf("expected auth message, got: %s", authMsg.Type)
	}

	// Extract auth data
	authDataBytes, err := json.Marshal(authMsg.Data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth data: %w", err)
	}

	var authData entity.WSAuthMessage
	if err := json.Unmarshal(authDataBytes, &authData); err != nil {
		return "", fmt.Errorf("failed to unmarshal auth data: %w", err)
	}

	if authData.Token == "" {
		return "", fmt.Errorf("no token provided")
	}

	// Remove Bearer prefix if present
	token := strings.TrimPrefix(authData.Token, "Bearer ")

	// Extract user principal from token
	userPrincipal, err := helper.GetUserPrincipalFromMSALAuthToken(token)
	if err != nil {
		return "", fmt.Errorf("failed to extract user principal: %w", err)
	}

	// Clear read timeout after successful authentication
	ws.SetReadDeadline(time.Time{})

	return userPrincipal, nil
}
