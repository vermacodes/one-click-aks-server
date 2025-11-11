package config

import (
	"context"
	"log"
	"one-click-aks-server/internal/logging"
	"os"
	"strconv"
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
	ActlabsHubURLInternal           string
	HttpRequestTimeoutSeconds       int
	MiseEndpoint                    string
	MiseVerboseLogging              bool
	AuthVerifyMode                  string
	APIKey                          string
	CorsAllowOrigins                string
	CorsAllowMethods                string
	CorsAllowHeaders                string
	// Add other configuration fields as needed
}

func NewConfig() *Config {

	ctx := context.Background()

	// Environment variables are now loaded in main.go before any config initialization
	// This ensures they're available when this function runs

	actLabsHubSubscriptionID := os.Getenv("ACTLABS_HUB_SUBSCRIPTION_ID")
	if actLabsHubSubscriptionID == "" {
		logging.LogError(ctx, "ACTLABS_HUB_SUBSCRIPTION_ID not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "ACTLABS_HUB_SUBSCRIPTION_ID: "+actLabsHubSubscriptionID)

	actLabsHubResourceGroupName := os.Getenv("ACTLABS_HUB_RESOURCE_GROUP_NAME")
	if actLabsHubResourceGroupName == "" {
		logging.LogError(ctx, "ACTLABS_HUB_RESOURCE_GROUP_NAME not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "ACTLABS_HUB_RESOURCE_GROUP_NAME: "+actLabsHubResourceGroupName)

	actLabsHubStorageAccountName := os.Getenv("ACTLABS_HUB_STORAGE_ACCOUNT_NAME")
	if actLabsHubStorageAccountName == "" {
		logging.LogError(ctx, "ACTLABS_HUB_STORAGE_ACCOUNT_NAME not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "ACTLABS_HUB_STORAGE_ACCOUNT_NAME: "+actLabsHubStorageAccountName)

	authTokenAud := os.Getenv("AUTH_TOKEN_AUD")
	if authTokenAud == "" {
		logging.LogError(ctx, "AUTH_TOKEN_AUD not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "AUTH_TOKEN_AUD: "+authTokenAud)

	authTokenIss := os.Getenv("AUTH_TOKEN_ISS")
	if authTokenIss == "" {
		logging.LogError(ctx, "AUTH_TOKEN_ISS not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "AUTH_TOKEN_ISS: "+authTokenIss)

	rootDir := os.Getenv("ROOT_DIR")
	if rootDir == "" {
		logging.LogError(ctx, "ROOT_DIR not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "ROOT_DIR: "+rootDir)

	useMsiString := os.Getenv("USE_MSI")
	if useMsiString == "" {
		logging.LogError(ctx, "USE_MSI not set")
		os.Exit(1)
	}
	useMsi := false
	if useMsiString == "true" {
		logging.LogDebug(ctx, "USE_MSI: true")
		useMsi = true
	} else {
		logging.LogDebug(ctx, "USE_MSI: false")
	}

	useServicePrincipalString := os.Getenv("USE_SERVICE_PRINCIPAL")
	if useServicePrincipalString == "" {
		logging.LogError(ctx, "USE_SERVICE_PRINCIPAL not set")
		os.Exit(1)
	}

	useServicePrincipal := false
	if useServicePrincipalString == "true" {
		logging.LogDebug(ctx, "USE_SERVICE_PRINCIPAL: true")
		useServicePrincipal = true
	} else {
		logging.LogDebug(ctx, "USE_SERVICE_PRINCIPAL: false")
	}

	azureClientId := os.Getenv("AZURE_CLIENT_ID")
	if azureClientId == "" && (useServicePrincipal || useMsi) {
		logging.LogError(ctx, "AZURE_CLIENT_ID not set")
		os.Exit(1)
	}

	azureClientSecret := os.Getenv("AZURE_CLIENT_SECRET")
	if azureClientSecret == "" && useServicePrincipal {
		logging.LogError(ctx, "AZURE_CLIENT_SECRET not set")
		os.Exit(1)
	}

	azureTenantID := os.Getenv("AZURE_TENANT_ID")
	if azureTenantID == "" && useServicePrincipal {
		logging.LogError(ctx, "AZURE_TENANT_ID not set")
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
		logging.LogError(ctx, "AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "AZURE_RED_HAT_OPENSHIFT_RP_FIRST_PARTY_SP_ID: "+aroRpFirstPartySpID)

	ActlabsHubURLInternal := os.Getenv("ACTLABS_HUB_URL_INTERNAL")
	if ActlabsHubURLInternal == "" {
		logging.LogError(ctx, "ACTLABS_HUB_URL_INTERNAL not set")
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
		logging.LogError(ctx, "MISE_ENDPOINT not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "MISE_ENDPOINT: "+miseEndpoint)

	miseVerboseLoggingString := os.Getenv("MISE_VERBOSE_LOGGING")
	if miseVerboseLoggingString == "" {
		miseVerboseLoggingString = "false" // default value
	}
	miseVerboseLogging := false
	if miseVerboseLoggingString == "true" {
		logging.LogDebug(ctx, "MISE_VERBOSE_LOGGING: true")
		miseVerboseLogging = true
	} else {
		logging.LogDebug(ctx, "MISE_VERBOSE_LOGGING: false")
	}

	authVerifyMode := os.Getenv("AUTH_VERIFY_MODE")
	if authVerifyMode == "" {
		authVerifyMode = "Custom" // default value
	}
	logging.LogDebug(ctx, "AUTH_VERIFY_MODE: "+authVerifyMode)

	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
		logging.LogError(ctx, "API_KEY not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "API_KEY is set")

	corsAllowOrigins := os.Getenv("CORS_ALLOW_ORIGINS")
	if corsAllowOrigins == "" {
		logging.LogError(ctx, "CORS_ALLOW_ORIGINS not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "CORS_ALLOW_ORIGINS: "+corsAllowOrigins)

	corsAllowMethods := os.Getenv("CORS_ALLOW_METHODS")
	if corsAllowMethods == "" {
		logging.LogError(ctx, "CORS_ALLOW_METHODS not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "CORS_ALLOW_METHODS: "+corsAllowMethods)

	corsAllowHeaders := os.Getenv("CORS_ALLOW_HEADERS")
	if corsAllowHeaders == "" {
		logging.LogError(ctx, "CORS_ALLOW_HEADERS not set")
		os.Exit(1)
	}
	logging.LogDebug(ctx, "CORS_ALLOW_HEADERS: "+corsAllowHeaders)

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
		ActlabsHubURLInternal:           ActlabsHubURLInternal,
		HttpRequestTimeoutSeconds:       httpRequestTimeoutSeconds,
		MiseEndpoint:                    miseEndpoint,
		MiseVerboseLogging:              miseVerboseLogging,
		AuthVerifyMode:                  authVerifyMode,
		APIKey:                          apiKey,
		CorsAllowOrigins:                corsAllowOrigins,
		CorsAllowMethods:                corsAllowMethods,
		CorsAllowHeaders:                corsAllowHeaders,
		// Add other configuration fields as needed
	}
}
