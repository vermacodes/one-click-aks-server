package service

import (
	"one-click-aks-server/internal/entity"

	"golang.org/x/exp/slog"
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
	subscription, err := a.authRepository.GetSubscriptionDetails()
	if err != nil {
		slog.Error("not able to get subscription details", err)
		return entity.Account{}, err
	}

	return entity.Account{
		Id:        *subscription.SubscriptionID,
		IsDefault: true,
		Name:      *subscription.DisplayName,
	}, nil

}
