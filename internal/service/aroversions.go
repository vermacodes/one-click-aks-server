package service

import (
	"context"
	"encoding/json"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
)

type aroVersionService struct {
	aroVersionRepository entity.AROVersionRepository
	preferenceService    entity.PreferenceService
	authService          entity.AuthService
}

func NewAROVersionService(aroVersionRepo entity.AROVersionRepository, preferenceService entity.PreferenceService, authService entity.AuthService) entity.AROVersionService {
	return &aroVersionService{
		aroVersionRepository: aroVersionRepo,
		preferenceService:    preferenceService,
		authService:          authService,
	}
}

func (a *aroVersionService) GetAROVersions(ctx context.Context) (entity.AROVersions, error) {
	logging.LogInfo(ctx, "getting aro versions")
	aroVersions := entity.AROVersions{}

	preference, err := a.preferenceService.GetPreference(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get user's preference", err)
		return aroVersions, err
	}

	logging.LogInfo(ctx, "getting aro versions for location "+preference.AzureRegion)
	out, err := a.aroVersionRepository.GetAROVersions(ctx, preference.AzureRegion, a.authService.GetSubscriptionId(ctx))
	if err != nil {
		logging.LogError(ctx, "not able to get ARO versions for location "+preference.AzureRegion, err)
		return aroVersions, err
	}

	if err := json.Unmarshal([]byte(out), &aroVersions); err != nil {
		logging.LogError(ctx, "not able to unmarshal ARO versions", err)
		return aroVersions, err
	}

	return aroVersions, nil
}

func (a *aroVersionService) GetDefaultAROVersion(ctx context.Context) string {
	logging.LogInfo(ctx, "getting default aro version")
	aroVersions, err := a.GetAROVersions(ctx)
	if err != nil {
		return ""
	}

	// Return the middle of array
	if len(aroVersions.Value) == 0 {
		return ""
	}
	middleIndex := len(aroVersions.Value) / 2
	return aroVersions.Value[middleIndex].Properties.Version
}

func (a *aroVersionService) DoesVersionExist(ctx context.Context, version string) bool {
	logging.LogInfo(ctx, "checking if aro version exists: "+version)
	aroVersions, err := a.GetAROVersions(ctx)
	if err != nil {
		return false
	}

	for _, v := range aroVersions.Value {
		if v.Properties.Version == version {
			return true
		}
	}
	return false
}
