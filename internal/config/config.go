package config

import (
	"context"
	"log"
	"one-click-aks-server/internal/logging"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	ActLabsHubSubscriptionID        string
	ActLabsHubResourceGroupName     string
	ActLabsHubStorageAccountName    string
	KubernetesVersionApiUrlTemplate string
	AroVersionApiUrlTemplate        string
	AroRpFirstPartySpID             string
	AuthTokenAud                    string
	AuthTokenIss                    string
	RootDir                         string
	UseMsi                          bool
	UseServicePrincipal             bool
	AzureClientID                   string
	AzureClientSecret               string
	AzureTenantID                   string
	ActlabsHubURL                   string
	HttpRequestTimeoutSeconds       int
	MiseEndpoint                    string
	MiseVerboseLogging              bool
	AuthVerifyMode                  string
	CorsAllowOrigins                string
	CorsAllowMethods                string
	CorsAllowHeaders                string
	// Add other configuration fields as needed
}

func NewConfig() *Config {

	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		logging.LogError(context.Background(), "Error loading .env file")
	}

	actLabsHubSubscriptionID := os.Getenv("ACTLABS_HUB_SUBSCRIPTION_ID")
	if actLabsHubSubscriptionID == "" {
		logging.LogError(context.Background(), "ACTLABS_HUB_SUBSCRIPTION_ID not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "ACTLABS_HUB_SUBSCRIPTION_ID: "+actLabsHubSubscriptionID)

	actLabsHubResourceGroupName := os.Getenv("ACTLABS_HUB_RESOURCE_GROUP_NAME")
	if actLabsHubResourceGroupName == "" {
		logging.LogError(context.Background(), "ACTLABS_HUB_RESOURCE_GROUP_NAME not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "ACTLABS_HUB_RESOURCE_GROUP_NAME: "+actLabsHubResourceGroupName)

	actLabsHubStorageAccountName := os.Getenv("ACTLABS_HUB_STORAGE_ACCOUNT_NAME")
	if actLabsHubStorageAccountName == "" {
		logging.LogError(context.Background(), "ACTLABS_HUB_STORAGE_ACCOUNT_NAME not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "ACTLABS_HUB_STORAGE_ACCOUNT_NAME: "+actLabsHubStorageAccountName)

	authTokenAud := os.Getenv("AUTH_TOKEN_AUD")
	if authTokenAud == "" {
		logging.LogError(context.Background(), "AUTH_TOKEN_AUD not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "AUTH_TOKEN_AUD: "+authTokenAud)

	authTokenIss := os.Getenv("AUTH_TOKEN_ISS")
	if authTokenIss == "" {
		logging.LogError(context.Background(), "AUTH_TOKEN_ISS not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "AUTH_TOKEN_ISS: "+authTokenIss)

	rootDir := os.Getenv("ROOT_DIR")
	if rootDir == "" {
		logging.LogError(context.Background(), "ROOT_DIR not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "ROOT_DIR: "+rootDir)

	useMsiString := os.Getenv("USE_MSI")
	if useMsiString == "" {
		logging.LogError(context.Background(), "USE_MSI not set")
		os.Exit(1)
	}
	useMsi := false
	if useMsiString == "true" {
		logging.LogDebug(context.Background(), "USE_MSI: true")
		useMsi = true
	} else {
		logging.LogDebug(context.Background(), "USE_MSI: false")
	}

	useServicePrincipalString := os.Getenv("USE_SERVICE_PRINCIPAL")
	if useServicePrincipalString == "" {
		logging.LogError(context.Background(), "USE_SERVICE_PRINCIPAL not set")
		os.Exit(1)
	}

	useServicePrincipal := false
	if useServicePrincipalString == "true" {
		logging.LogDebug(context.Background(), "USE_SERVICE_PRINCIPAL: true")
		useServicePrincipal = true
	} else {
		logging.LogDebug(context.Background(), "USE_SERVICE_PRINCIPAL: false")
	}

	azureClientId := os.Getenv("AZURE_CLIENT_ID")
	if azureClientId == "" && useServicePrincipal {
		logging.LogError(context.Background(), "AZURE_CLIENT_ID not set")
		os.Exit(1)
	}

	azureClientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	if azureClientSecret == "" && useServicePrincipal {
		logging.LogError(context.Background(), "AZURE_CLIENT_SECRET not set")
		os.Exit(1)
	}

	azureTenantID := os.Getenv("AZURE_TENANT_ID")
	if azureTenantID == "" && useServicePrincipal {
		logging.LogError(context.Background(), "AZURE_TENANT_ID not set")
		os.Exit(1)
	}

	kubernetesVersionApiUrlTemplate := os.Getenv("KUBERNETES_VERSION_API_URL_TEMPLATE")
	if kubernetesVersionApiUrlTemplate == "" {
		kubernetesVersionApiUrlTemplate = "https://management.azure.com/subscriptions/%s/providers/Microsoft.ContainerService/locations/%s/kubernetesVersions?api-version=2023-09-01"
	}

	aroVersionApiUrlTemplate := os.Getenv("ARO_VERSION_API_URL_TEMPLATE")
	if aroVersionApiUrlTemplate == "" {
		aroVersionApiUrlTemplate = "https://management.azure.com/subscriptions/%s/providers/Microsoft.RedHatOpenShift/locations/%s/openshiftversions?api-version=2024-08-12-preview"
	}

	aroRpFirstPartySpID := os.Getenv("AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID")
	if aroRpFirstPartySpID == "" {
		logging.LogError(context.Background(), "AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID: "+aroRpFirstPartySpID)

	actlabsHubURL := os.Getenv("ACTLABS_HUB_URL")
	if actlabsHubURL == "" {
		logging.LogError(context.Background(), "ACTLABS_HUB_URL not set")
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

	miseEndpoint := os.Getenv("MISE_ENDPOINT")
	if miseEndpoint == "" {
		logging.LogError(context.Background(), "MISE_ENDPOINT not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "MISE_ENDPOINT: "+miseEndpoint)

	miseVerboseLoggingString := os.Getenv("MISE_VERBOSE_LOGGING")
	if miseVerboseLoggingString == "" {
		miseVerboseLoggingString = "false" // default value
	}
	miseVerboseLogging := false
	if miseVerboseLoggingString == "true" {
		logging.LogDebug(context.Background(), "MISE_VERBOSE_LOGGING: true")
		miseVerboseLogging = true
	} else {
		logging.LogDebug(context.Background(), "MISE_VERBOSE_LOGGING: false")
	}

	authVerifyMode := os.Getenv("AUTH_VERIFY_MODE")
	if authVerifyMode == "" {
		authVerifyMode = "Custom" // default value
	}
	logging.LogDebug(context.Background(), "AUTH_VERIFY_MODE: "+authVerifyMode)

	corsAllowOrigins := os.Getenv("CORS_ALLOW_ORIGINS")
	if corsAllowOrigins == "" {
		logging.LogError(context.Background(), "CORS_ALLOW_ORIGINS not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "CORS_ALLOW_ORIGINS: "+corsAllowOrigins)

	corsAllowMethods := os.Getenv("CORS_ALLOW_METHODS")
	if corsAllowMethods == "" {
		logging.LogError(context.Background(), "CORS_ALLOW_METHODS not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "CORS_ALLOW_METHODS: "+corsAllowMethods)

	corsAllowHeaders := os.Getenv("CORS_ALLOW_HEADERS")
	if corsAllowHeaders == "" {
		logging.LogError(context.Background(), "CORS_ALLOW_HEADERS not set")
		os.Exit(1)
	}
	logging.LogDebug(context.Background(), "CORS_ALLOW_HEADERS: "+corsAllowHeaders)

	// Retrieve other environment variables and check them as needed

	return &Config{
		ActLabsHubSubscriptionID:        actLabsHubSubscriptionID,
		ActLabsHubResourceGroupName:     actLabsHubResourceGroupName,
		ActLabsHubStorageAccountName:    actLabsHubStorageAccountName,
		KubernetesVersionApiUrlTemplate: kubernetesVersionApiUrlTemplate,
		AroVersionApiUrlTemplate:        aroVersionApiUrlTemplate,
		AroRpFirstPartySpID:             aroRpFirstPartySpID,
		AuthTokenAud:                    authTokenAud,
		AuthTokenIss:                    authTokenIss,
		RootDir:                         rootDir,
		UseMsi:                          useMsi,
		UseServicePrincipal:             useServicePrincipal,
		AzureClientID:                   azureClientId,
		AzureClientSecret:               azureClientSecret,
		AzureTenantID:                   azureTenantID,
		ActlabsHubURL:                   actlabsHubURL,
		HttpRequestTimeoutSeconds:       httpRequestTimeoutSeconds,
		MiseEndpoint:                    miseEndpoint,
		MiseVerboseLogging:              miseVerboseLogging,
		AuthVerifyMode:                  authVerifyMode,
		CorsAllowOrigins:                corsAllowOrigins,
		CorsAllowMethods:                corsAllowMethods,
		CorsAllowHeaders:                corsAllowHeaders,
		// Add other configuration fields as needed
	}
}
