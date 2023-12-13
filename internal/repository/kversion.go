package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"one-click-aks-server/internal/entity"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"golang.org/x/exp/slog"
)

type kVersionRepository struct {
	cred *azidentity.DefaultAzureCredential
}

func NewKVersionRepository() entity.KVersionRepository {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		panic(err)
	}
	return &kVersionRepository{
		cred: cred,
	}
}

func (k *kVersionRepository) GetOrchestrator(location string) (string, error) {
	slog.Info("Getting Kubernetes versions for location " + location)
	// Get access token from cred
	accessToken, err := k.cred.GetToken(context.Background(), policy.TokenRequestOptions{
		Scopes: []string{"https://management.azure.com/.default"},
	})
	if err != nil {
		return "", err
	}

	slog.Debug("Access Token" + accessToken.Token)

	// Make HTTP request to retrieve Kubernetes versions
	url := "https://management.azure.com/subscriptions/da846304-0089-48e0-bfa7-65f68a3eb74f/providers/Microsoft.ContainerService/locations/" + location + "/kubernetesVersions?api-version=2023-09-01"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return "", err
	}

	// Extract the Kubernetes version from the response
	versions := result["value"].([]interface{})
	if len(versions) > 0 {
		version := versions[0].(map[string]interface{})
		return version["name"].(string), nil
	}

	return "", fmt.Errorf("No Kubernetes versions found")
}
