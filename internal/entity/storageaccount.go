package entity

type StorageAccountService interface {
	GetStorageAccountName() (string, error)
	BreakBlobLease(storageAccountName string, containerName string, workspaceName string) error
}

type StorageAccountRepository interface {
	GetStorageAccountName() (string, error)
	BreakBlobLease(storageAccountName string, containerName string, blobName string) error
}
