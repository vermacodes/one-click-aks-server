package entity

import "github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/subscription/armsubscription"

type Account struct {
	Id        string `json:"id"`
	IsDefault bool   `json:"isDefault"`
	Name      string `json:"name"`
}

type AuthService interface {
	GetSubscriptionDetails() (Account, error)
}

type AuthRepository interface {
	GetSubscriptionDetails() (*armsubscription.Subscription, error)
}
