package service

import (
	"errors"
	"one-click-aks-server/internal/entity"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/storage/armstorage"
	"golang.org/x/exp/slog"
)

type storageAccountService struct {
	storageAccountRepository entity.StorageAccountRepository
}

func NewStorageAccountService(storageAccountRepo entity.StorageAccountRepository) entity.StorageAccountService {
	return &storageAccountService{
		storageAccountRepository: storageAccountRepo,
	}
}

func (s *storageAccountService) GetStorageAccount() (armstorage.Account, error) {
	storageAccount, err := s.storageAccountRepository.GetStorageAccount()
	if err != nil {
		slog.Error("not able to get storage account", err)
		return armstorage.Account{}, err
	}
	return storageAccount, nil
}

func (s *storageAccountService) GetStorageAccountName() (string, error) {
	storageAccountName, err := s.GetStorageAccount()
	if err != nil {
		slog.Error("not able to get storage account name", err)
		return "", err
	}

	return *storageAccountName.Name, nil
}

func (s *storageAccountService) BreakBlobLease(storageAccountName string, containerName string, workspaceName string) error {

	// If workspace name is default, then blob name is terraform.tfstate
	// else it is terraform.tfstateenv:<workspaceName>
	blobName := "terraform.tfstate"
	if workspaceName != "default" {
		blobName = "terraform.tfstateenv:" + workspaceName
	}

	err := s.storageAccountRepository.BreakBlobLease(storageAccountName, containerName, blobName)
	if err != nil {
		slog.Error("not able to break blob lease", err)

		if strings.Contains(err.Error(), "RESPONSE 409: 409 There is currently no lease on the blob") {
			return errors.New("there is currently no lease on the blob")
		}
		if strings.Contains(err.Error(), "RESPONSE 404: 404 The specified blob does not exist.") {
			return errors.New("the specified blob does not exist")
		}

		return err
	}

	slog.Debug("state lease broken for workspace " + workspaceName + " in storage account " + storageAccountName + " in container " + containerName)
	return nil
}
