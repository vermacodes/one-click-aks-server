package service

import (
	"encoding/json"
	"one-click-aks-server/internal/entity"

	"golang.org/x/exp/slog"
)

type aroVersionService struct {
	aroVersionRepository entity.AROVersionRepository
	preferenceService    entity.PreferenceService
}

func NewAROVersionService(aroVersionRepo entity.AROVersionRepository, preferenceService entity.PreferenceService) entity.AROVersionService {
	return &aroVersionService{
		aroVersionRepository: aroVersionRepo,
		preferenceService:    preferenceService,
	}
}

func (a *aroVersionService) GetAROVersions() (entity.AROVersions, error) {
	slog.Info("Fetching ARO versions")
	aroVersions := entity.AROVersions{}

	preference, err := a.preferenceService.GetPreference()
	if err != nil {
		slog.Error("not able to get user's preference", err)
		return aroVersions, err
	}

	slog.Info("Getting ARO versions for location " + preference.AzureRegion)
	out, err := a.aroVersionRepository.GetAROVersions(preference.AzureRegion)
	if err != nil {
		slog.Error("not able to get ARO versions for location "+preference.AzureRegion, err)
		return aroVersions, err
	}

	if err := json.Unmarshal([]byte(out), &aroVersions); err != nil {
		slog.Error("not able to unmarshal ARO versions", err)
		return aroVersions, err
	}

	return aroVersions, nil
}

func (a *aroVersionService) GetDefaultAROVersion() string {
	slog.Info("Fetching default ARO version")
	aroVersions, err := a.GetAROVersions()
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

func (a *aroVersionService) DoesVersionExist(version string) bool {
	slog.Info("Checking if ARO version exists: " + version)
	aroVersions, err := a.GetAROVersions()
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
