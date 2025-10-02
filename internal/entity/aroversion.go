package entity

import "context"

// AROVersions represents the structure of the ARO versions API response.
type AROVersions struct {
	Value []struct {
		Name       string `json:"name"`
		Type       string `json:"type"`
		Properties struct {
			Version string `json:"version"`
		} `json:"properties"`
	} `json:"value"`
}

// AROVersionService defines the interface for fetching ARO versions.
type AROVersionService interface {
	GetAROVersions(ctx context.Context) (AROVersions, error)
	GetDefaultAROVersion(ctx context.Context) string
	DoesVersionExist(ctx context.Context, version string) bool
}

// AROVersionRepository defines the interface for interacting with ARO version data.
type AROVersionRepository interface {
	GetAROVersions(ctx context.Context, location string) (string, error)
}
