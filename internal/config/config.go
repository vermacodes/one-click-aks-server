package config

import (
	"log"
	"os"
	"strconv"

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
	UseMsi                          bool
	AzureClientID                   string
	ActlabsHubURL                   string
	HttpRequestTimeoutSeconds       int
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

	actlabsHubURL := os.Getenv("ACTLABS_HUB_URL")
	if actlabsHubURL == "" {
		slog.Error("ACTLABS_HUB_URL not set")
		os.Exit(1)
	}

	httpRequestTimeoutSecondsStr := os.Getenv("HTTP_REQUEST_TIMEOUT_SECONDS")
	httpRequestTimeoutSeconds := 30 // default value
	if httpRequestTimeoutSecondsStr != "" {
		var err error
		httpRequestTimeoutSeconds, err = strconv.Atoi(httpRequestTimeoutSecondsStr)
		if err != nil {
			log.Fatalf("Invalid value for HTTP_REQUEST_TIMEOUT_SECONDS: %v", err)
		}
	}
	// Retrieve other environment variables and check them as needed

	return &Config{
		SubscriptionID:                  subscriptionID,
		KubernetesVersionApiUrlTemplate: kubernetesVersionApiUrlTemplate,
		ArmUserPrincipalName:            armUserPrincipalName,
		AuthTokenAud:                    authTokenAud,
		AuthTokenIss:                    authTokenIss,
		RootDir:                         rootDir,
		UseMsi:                          useMsi,
		AzureClientID:                   azureClientId,
		ActlabsHubURL:                   actlabsHubURL,
		HttpRequestTimeoutSeconds:       httpRequestTimeoutSeconds,
		// Set other fields
	}
}
