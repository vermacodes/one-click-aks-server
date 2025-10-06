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

func (a *authService) GetSubscriptionDetails(ctx context.Context) (entity.Account, error) {
	subscription, err := a.authRepository.GetSubscriptionDetails(ctx)
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

func (a *authService) GetSubscriptionId(ctx context.Context) string {
	subscriptionId, err := a.authRepository.GetSubscriptionId(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get subscription id", "error", err)
		return ""
	}

	return subscriptionId
}
