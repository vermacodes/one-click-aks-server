package entity

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
	GetAROVersions() (AROVersions, error)
	GetDefaultAROVersion() string
	DoesVersionExist(version string) bool
}

// AROVersionRepository defines the interface for interacting with ARO version data.
type AROVersionRepository interface {
	GetAROVersions(location string) (string, error)
}
