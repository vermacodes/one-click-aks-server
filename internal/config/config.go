package config

import (
	"os"

	"golang.org/x/exp/slog"
)

type Config struct {
	SubscriptionID                  string
	KubernetesVersionApiUrlTemplate string
	// Add other configuration fields as needed
}

func NewConfig() *Config {
	subscriptionID := os.Getenv("AZURE_SUBSCRIPTION_ID")
	if subscriptionID == "" {
		slog.Error("AZURE_SUBSCRIPTION_ID not set")
		os.Exit(1)
	}
	slog.Info("AZURE_SUBSCRIPTION_ID: " + subscriptionID)

	rootDir := os.Getenv("ROOT_DIR")
	if rootDir == "" {
		slog.Error("ROOT_DIR not set")
		os.Exit(1)
	}
	slog.Info("ROOT_DIR: " + rootDir)

	kubernetesVersionApiUrlTemplate := os.Getenv("KUBERNETES_VERSION_API_URL_TEMPLATE")
	if kubernetesVersionApiUrlTemplate == "" {
		kubernetesVersionApiUrlTemplate = "https://management.azure.com/subscriptions/%s/providers/Microsoft.ContainerService/locations/%s/kubernetesVersions?api-version=2023-09-01"
	}
	// Retrieve other environment variables and check them as needed

	return &Config{
		SubscriptionID:                  subscriptionID,
		KubernetesVersionApiUrlTemplate: kubernetesVersionApiUrlTemplate,
		// Set other fields
	}
}
