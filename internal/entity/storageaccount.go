package entity

import "context"

type StorageAccountService interface {
	GetStorageAccountName(ctx context.Context) (string, error)
	BreakBlobLease(ctx context.Context, storageAccountName string, containerName string, workspaceName string) error
}

type StorageAccountRepository interface {
	GetStorageAccountName(ctx context.Context) (string, error)
	BreakBlobLease(ctx context.Context, storageAccountName string, containerName string, blobName string) error
}
