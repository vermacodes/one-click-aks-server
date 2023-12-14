package config

import (
	"os"

	"github.com/joho/godotenv"
	"golang.org/x/exp/slog"
)

type Config struct {
	SubscriptionID                  string
	KubernetesVersionApiUrlTemplate string
	ArmUserPrincipalName            string
	AuthTokenAud                    string
	AuthTokenIss                    string
	RootDir                         string
	ProtectedLabSecret              string
	UseMsi                          bool
	AzureClientID                   string
	// Add other configuration fields as needed
}

func NewConfig() *Config {

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		slog.Error("Error loading .env file")
	}

	armUserPrincipalName := os.Getenv("ARM_USER_PRINCIPAL_NAME")
	slog.Info("ARM_USER_PRINCIPAL_NAME: " + armUserPrincipalName)

	if armUserPrincipalName == "" {
		slog.Error("ARM_USER_PRINCIPAL_NAME not set")
		os.Exit(1)
	}
	slog.Info("ARM_USER_PRINCIPAL_NAME: " + armUserPrincipalName)

	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		slog.Error("AZURE_SUBSCRIPTION_ID not set")
		os.Exit(1)
	}
	slog.Info("AZURE_SUBSCRIPTION_ID: " + subscriptionID)

	authTokenAud := os.Getenv("AUTH_TOKEN_AUD")
	if authTokenAud == "" {
		slog.Error("AUTH_TOKEN_AUD not set")
		os.Exit(1)
	}
	slog.Info("AUTH_TOKEN_AUD: " + authTokenAud)

	authTokenIss := os.Getenv("AUTH_TOKEN_ISS")
	if authTokenIss == "" {
		slog.Error("AUTH_TOKEN_ISS not set")
		os.Exit(1)
	}
	slog.Info("AUTH_TOKEN_ISS: " + authTokenIss)

	rootDir := os.Getenv("ROOT_DIR")
	if rootDir == "" {
		slog.Error("ROOT_DIR not set")
		os.Exit(1)
	}
	slog.Info("ROOT_DIR: " + rootDir)

	protectedLabSecret := os.Getenv("PROTECTED_LAB_SECRET")
	if protectedLabSecret == "" {
		slog.Error("PROTECTED_LAB_SECRET not set")
		os.Exit(1)
	}

	useMsiString := os.Getenv("USE_MSI")
	if useMsiString == "" {
		slog.Error("USE_MSI not set")
		os.Exit(1)
	}
	useMsi := false
	if useMsiString == "true" {
		slog.Info("USE_MSI: true")
		useMsi = true
	} else {
		slog.Info("USE_MSI: false")
	}

	azureClientId := os.Getenv("AZURE_CLIENT_ID")
	if azureClientId == "" {
		slog.Error("AZURE_CLIENT_ID not set")
		os.Exit(1)
	}

	kubernetesVersionApiUrlTemplate := os.Getenv("KUBERNETES_VERSION_API_URL_TEMPLATE")
	if kubernetesVersionApiUrlTemplate == "" {
		kubernetesVersionApiUrlTemplate = "https://management.azure.com/subscriptions/%s/providers/Microsoft.ContainerService/locations/%s/kubernetesVersions?api-version=2023-09-01"
	}
	// Retrieve other environment variables and check them as needed

	return &Config{
		SubscriptionID:                  subscriptionID,
		KubernetesVersionApiUrlTemplate: kubernetesVersionApiUrlTemplate,
		ArmUserPrincipalName:            armUserPrincipalName,
		AuthTokenAud:                    authTokenAud,
		AuthTokenIss:                    authTokenIss,
		RootDir:                         rootDir,
		ProtectedLabSecret:              protectedLabSecret,
		UseMsi:                          useMsi,
		AzureClientID:                   azureClientId,
		// Set other fields
	}
}
