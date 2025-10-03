package service

import (
	"context"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
)

type authService struct {
	authRepository entity.AuthRepository
}

func NewAuthService(authRepository entity.AuthRepository) entity.AuthService {
	return &authService{
		authRepository: authRepository,
	}
}

func (a *authService) GetSubscriptionDetails() (entity.Account, error) {
	ctx := context.Background()
	subscription, err := a.authRepository.GetSubscriptionDetails()
	if err != nil {
		logging.LogError(ctx, "not able to get subscription details", "error", err)
		return entity.Account{}, err
	}

	return entity.Account{
		Id:        *subscription.SubscriptionID,
		IsDefault: true,
		Name:      *subscription.DisplayName,
	}, nil

}
