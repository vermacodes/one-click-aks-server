package auth

import (
	"context"
	"log"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
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
