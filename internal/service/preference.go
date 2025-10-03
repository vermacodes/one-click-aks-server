package service

import (
	"context"
	"encoding/json"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
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

func (p *preferenceService) GetPreference(ctx context.Context) (entity.Preference, error) {
	logging.LogInfo(ctx, "getting preference")
	preference := entity.Preference{}

	preferenceString, err := p.preferenceRepository.GetPreferenceFromRedis(ctx)
	if err == nil {
		logging.LogDebug(ctx, "preference found in redis.")
		errJson := json.Unmarshal([]byte(preferenceString), &preference)
		if errJson == nil {
			return preference, errJson
		}
		logging.LogError(ctx, "not able to marshal the preference in redis", "error", errJson)
	}

	// Rest of function will execute if issue in getting preference from redis.

	storageAccountName, err := p.storageAccountService.GetStorageAccountName(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get storage account name", "error", err)
		return preference, err
	}

	preferenceString, err = p.preferenceRepository.GetPreferenceFromBlob(ctx, storageAccountName)
	if err != nil || preferenceString == "" {
		logging.LogError(ctx, "not able to get preference from storage account, fall back to default", "error", err)

		// Setting and returning default preference
		if err := p.SetPreference(ctx, defaultPreference()); err != nil {
			logging.LogError(ctx, "not able to set default preference in storage", "error", err)
		}
		return defaultPreference(), nil
	}

	// Add preference to redis.
	if err := p.preferenceRepository.PutPreferenceInRedis(ctx, preferenceString); err != nil {
		logging.LogError(ctx, "not able to put preference in redis.", err)
	}

	if err := json.Unmarshal([]byte(preferenceString), &preference); err != nil {
		logging.LogError(ctx, "not able to unmarshal preference from blob to object", "error", err)
		return preference, err
	}

	return preference, nil
}

func (p *preferenceService) SetPreference(ctx context.Context, preference entity.Preference) error {
	storageAccountName, err := p.storageAccountService.GetStorageAccountName(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get storage account name", "error", err)
		return err
	}
	logging.LogDebug(ctx, "storage account name -> "+storageAccountName)

	out, err := json.Marshal(preference)
	if err != nil || string(out) == "" {
		logging.LogError(ctx, "Error marshaling json", err)
		return err
	}

	logging.LogDebug(ctx, "preference -> "+string(out))

	if err := p.preferenceRepository.PutPreferenceInBlob(ctx, string(out), storageAccountName); err != nil {
		logging.LogError(ctx, "not able to put preference in blob", "error", err)
		return err
	}

	// Cleanup Cache
	if err := p.preferenceRepository.DeletePreferenceFromRedis(ctx); err != nil {
		logging.LogError(ctx, "not able to delete preference from redis", "error", err)
		return err
	}

	if err := p.preferenceRepository.DeleteKubernetesVersionsFromRedis(ctx); err != nil {
		logging.LogError(ctx, "not able to delete kubernetes versions from redis", "error", err)
		return err
	}

	if err := p.preferenceRepository.PutPreferenceInRedis(ctx, string(out)); err != nil {
		logging.LogError(ctx, "not able to put preference in redis", "error", err)
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
