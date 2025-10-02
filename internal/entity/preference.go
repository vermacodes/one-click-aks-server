package entity

import "context"

type Preference struct {
	AzureRegion        string `json:"azureRegion"`
	TerminalAutoScroll bool   `json:"terminalAutoScroll"`
}

type PreferenceService interface {
	GetPreference(ctx context.Context) (Preference, error)
	SetPreference(ctx context.Context, preference Preference) error
}

type PreferenceRepository interface {
	GetPreferenceFromBlob(ctx context.Context, storageAccountName string) (string, error)
	PutPreferenceInBlob(ctx context.Context, val string, storageAccountName string) error
	GetPreferenceFromRedis(ctx context.Context) (string, error)
	PutPreferenceInRedis(ctx context.Context, val string) error
	DeletePreferenceFromRedis(ctx context.Context) error
	DeleteKubernetesVersionsFromRedis(ctx context.Context) error
}
