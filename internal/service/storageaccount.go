package service

import (
	"context"
	"errors"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
	"strings"
)

type storageAccountService struct {
	storageAccountRepository entity.StorageAccountRepository
}

func NewStorageAccountService(storageAccountRepo entity.StorageAccountRepository) entity.StorageAccountService {
	return &storageAccountService{
		storageAccountRepository: storageAccountRepo,
	}
}

func (s *storageAccountService) GetStorageAccountName(ctx context.Context) (string, error) {
	logging.LogInfo(ctx, "getting storage account name")
	storageAccountName, err := s.storageAccountRepository.GetStorageAccountName(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get storage account name", "error", err)
		return "", err
	}

	return storageAccountName, nil
}

func (s *storageAccountService) BreakBlobLease(ctx context.Context, storageAccountName string, containerName string, workspaceName string) error {
	logging.LogInfo(ctx, "breaking blob lease", "storage_account_name", storageAccountName, "container_name", containerName, "workspace_name", workspaceName)
	// If workspace name is default, then blob name is terraform.tfstate
	// else it is terraform.tfstateenv:<workspaceName>
	blobName := "terraform.tfstate"
	if workspaceName != "default" {
		blobName = "terraform.tfstateenv:" + workspaceName
	}

	err := s.storageAccountRepository.BreakBlobLease(ctx, storageAccountName, containerName, blobName)
	if err != nil {
		logging.LogError(ctx, "not able to break blob lease", "error", err)

		if strings.Contains(err.Error(), "RESPONSE 409: 409 There is currently no lease on the blob") {
			return errors.New("there is currently no lease on the blob")
		}
		if strings.Contains(err.Error(), "RESPONSE 404: 404 The specified blob does not exist.") {
			return errors.New("the specified blob does not exist")
		}

		return err
	}

	logging.LogDebug(ctx, "state lease broken for workspace", "workspace_name", workspaceName, "storage_account_name", storageAccountName, "container_name", containerName)
	return nil
}
