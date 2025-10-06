package entity

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"
)

type Account struct {
	Id        string `json:"id"`
	IsDefault bool   `json:"isDefault"`
	Name      string `json:"name"`
}

type AuthService interface {
	GetSubscriptionId(ctx context.Context) string
	GetSubscriptionDetails(ctx context.Context) (Account, error)
}

type AuthRepository interface {
	GetSubscriptionId(ctx context.Context) (string, error)
	GetSubscriptionDetails(ctx context.Context) (*armsubscription.Subscription, error)
}
