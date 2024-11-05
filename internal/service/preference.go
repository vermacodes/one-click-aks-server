package service

import (
	"encoding/json"

	"one-click-aks-server/internal/entity"

	"golang.org/x/exp/slog"
)

type preferenceService struct {
	preferenceRepository  entity.PreferenceRepository
	storageAccountService entity.StorageAccountService
}

func NewPreferenceService(preferenceRepo entity.PreferenceRepository, storageAccountService entity.StorageAccountService) entity.PreferenceService {
	return &preferenceService{
		preferenceRepository:  preferenceRepo,
		storageAccountService: storageAccountService,
	}
}

func (p *preferenceService) GetPreference() (entity.Preference, error) {
	preference := entity.Preference{}

	preferenceString, err := p.preferenceRepository.GetPreferenceFromRedis()
	if err == nil {
		slog.Debug("preference found in redis.")
		errJson := json.Unmarshal([]byte(preferenceString), &preference)
		if errJson == nil {
			return preference, errJson
		}
		slog.Error("not able to marshal the preference in redis", errJson)
	}

	// Rest of function will execute if issue in getting preference from redis.

	storageAccountName, err := p.storageAccountService.GetStorageAccountName()
	if err != nil {
		slog.Error("not able to get storage account name", err)
		return preference, err
	}

	preferenceString, err = p.preferenceRepository.GetPreferenceFromBlob(storageAccountName)
	if err != nil || preferenceString == "" {
		slog.Error("not able to get preference from storage account, fall back to default", err)

		// Setting and returning default preference
		if err := p.SetPreference(defaultPreference()); err != nil {
			slog.Error("not able to set default preference in storage", err)
		}
		return defaultPreference(), nil
	}

	// Add preference to redis.
	if err := p.preferenceRepository.PutPreferenceInRedis(preferenceString); err != nil {
		slog.Error("not able to put preference in redis.", err)
	}

	if err := json.Unmarshal([]byte(preferenceString), &preference); err != nil {
		slog.Error("not able to unmarshal preference from blob to object", err)
		return preference, err
	}

	return preference, nil
}

func (p *preferenceService) SetPreference(preference entity.Preference) error {
	storageAccountName, err := p.storageAccountService.GetStorageAccountName()
	if err != nil {
		slog.Error("not able to get storage account name", err)
		return err
	}
	slog.Debug("storage account name -> " + storageAccountName)

	out, err := json.Marshal(preference)
	if err != nil || string(out) == "" {
		slog.Error("Error marshaling json", err)
		return err
	}

	slog.Debug("preference -> " + string(out))

	if err := p.preferenceRepository.PutPreferenceInBlob(string(out), storageAccountName); err != nil {
		slog.Error("not able to put preference in blob", err)
		return err
	}

	// Cleanup Cache
	if err := p.preferenceRepository.DeletePreferenceFromRedis(); err != nil {
		slog.Error("not able to delete preference from redis", err)
		return err
	}

	if err := p.preferenceRepository.DeleteKubernetesVersionsFromRedis(); err != nil {
		slog.Error("not able to delete kubernetes versions from redis", err)
		return err
	}

	if err := p.preferenceRepository.PutPreferenceInRedis(string(out)); err != nil {
		slog.Error("not able to put preference in redis", err)
		return err
	}

	return nil
}

func defaultPreference() entity.Preference {
	return entity.Preference{
		AzureRegion:        "East US",
		TerminalAutoScroll: false,
	}
}
