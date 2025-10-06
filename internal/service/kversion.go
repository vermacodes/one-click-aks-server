package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
)

type kVersionService struct {
	kVersionRepository entity.KVersionRepository
	preferenceService  entity.PreferenceService
	authService        entity.AuthService
}

func NewKVersionService(kVersionRepo entity.KVersionRepository, preferenceService entity.PreferenceService) entity.KVersionService {
	return &kVersionService{
		kVersionRepository: kVersionRepo,
		preferenceService:  preferenceService,
	}
}

func (k *kVersionService) GetOrchestrator(ctx context.Context) (entity.KubernetesVersions, error) {
	logging.LogInfo(ctx, "getting kubernetes versions")
	kubernetesVersions := entity.KubernetesVersions{}

	preference, err := k.preferenceService.GetPreference(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get user's preference", err)
		return kubernetesVersions, err
	}

	logging.LogDebug(ctx, "getting kubernetes versions for location "+preference.AzureRegion)
	out, err := k.kVersionRepository.GetOrchestrator(ctx, preference.AzureRegion, k.authService.GetSubscriptionId(ctx))
	if err != nil {
		logging.LogError(ctx, "not able to get orchestrator", err)
		return kubernetesVersions, err
	}

	if err := json.Unmarshal([]byte(out), &kubernetesVersions); err != nil {
		logging.LogError(ctx, "not able to unmarshal output from cli to object", err)
		return kubernetesVersions, err
	}

	// Filter Kubernetes versions
	filteredVersions := entity.KubernetesVersions{}
	for _, version := range kubernetesVersions.Values {
		for _, capability := range version.Capabilities.SupportPlan {
			if capability == "KubernetesOfficial" {
				logging.LogDebug(ctx, "adding version "+version.Version)
				filteredVersions.Values = append(filteredVersions.Values, version)
				break
			}
		}
	}

	return filteredVersions, nil
}

func (k *kVersionService) GetMostRecentVersion(ctx context.Context) string {
	o, err := k.GetOrchestrator(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get orchestrator", err)
		return ""
	}

	mostRecentVersionInt := 0
	mostRecentVersionString := ""

	// return the most recent version.
	for _, v := range o.Values {
		// Iterate over PatchVersions
		for patchVersion := range v.PatchVersions {
			versionParts := strings.Split(patchVersion, ".")

			if len(versionParts) < 3 {
				logging.LogError(ctx, "invalid version string", err)
				return ""
			}

			major, err := strconv.Atoi(versionParts[0])
			if err != nil {
				fmt.Println("Error:", err)
				return ""
			}

			minor, err := strconv.Atoi(versionParts[1])
			if err != nil {
				fmt.Println("Error:", err)
				return ""
			}

			patch, err := strconv.Atoi(versionParts[2])
			if err != nil {
				fmt.Println("Error:", err)
				return ""
			}

			thisVersionInt := major*10000 + minor*100 + patch

			if thisVersionInt > mostRecentVersionInt {
				mostRecentVersionString = patchVersion
				mostRecentVersionInt = thisVersionInt
			}

		}
	}
	return mostRecentVersionString
}

func (k *kVersionService) GetOldestVersion(ctx context.Context) string {
	o, err := k.GetOrchestrator(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get orchestrator", err)
		return ""
	}

	oldestVersionInt := 0
	oldestVersionString := ""

	// return the most recent version.
	for _, v := range o.Values {
		// Iterate over PatchVersions
		for patchVersion := range v.PatchVersions {
			versionParts := strings.Split(patchVersion, ".")

			if len(versionParts) < 3 {
				logging.LogError(ctx, "invalid version string", err)
				return ""
			}

			major, err := strconv.Atoi(versionParts[0])
			if err != nil {
				fmt.Println("Error:", err)
				return ""
			}

			minor, err := strconv.Atoi(versionParts[1])
			if err != nil {
				fmt.Println("Error:", err)
				return ""
			}

			patch, err := strconv.Atoi(versionParts[2])
			if err != nil {
				fmt.Println("Error:", err)
				return ""
			}

			thisVersionInt := major*10000 + minor*100 + patch

			if thisVersionInt > oldestVersionInt {
				oldestVersionString = patchVersion
				oldestVersionInt = thisVersionInt
			}

		}
	}
	return oldestVersionString
}

func (k *kVersionService) DoesVersionExist(ctx context.Context, version string) bool {
	o, err := k.GetOrchestrator(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get orchestrator", err)
		return false
	}

	// return the most recent version.
	for _, v := range o.Values {
		// Iterate over PatchVersions
		for patchVersion := range v.PatchVersions {
			logging.LogDebug(ctx, "patch version "+patchVersion)
			if patchVersion == version {
				return true
			}
		}
	}
	return false
}

func (k *kVersionService) GetDefaultVersion(ctx context.Context) string {
	o, err := k.GetOrchestrator(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get orchestrator", err)
		return ""
	}

	// Filter out preview versions and sort the remaining versions in descending order
	var sortedValues []entity.Value
	for _, v := range o.Values {
		if v.IsPreview != nil && *v.IsPreview {
			continue
		}
		sortedValues = append(sortedValues, v)
	}
	sort.Slice(sortedValues, func(i, j int) bool {
		return versionGreater(sortedValues[i].Version, sortedValues[j].Version)
	})

	// If there are at least two versions, select the second from top minor version
	if len(sortedValues) >= 2 {
		secondTopMinorVersion := sortedValues[1]

		// Get the patch versions of the second from top minor version
		var patchVersions []string
		for patchVersion := range secondTopMinorVersion.PatchVersions {
			patchVersions = append(patchVersions, patchVersion)
		}

		// Sort the patch versions in descending order
		sort.Slice(patchVersions, func(i, j int) bool {
			return versionGreater(patchVersions[i], patchVersions[j])
		})

		// If there are any patch versions, return the highest one
		if len(patchVersions) > 0 {
			return patchVersions[0]
		}
	}

	logging.LogError(ctx, "not able to get default version", nil)
	return ""
}

// Returns true if version a is greater than version b.
func versionGreater(a, b string) bool {
	versionPartsA := strings.Split(a, ".")
	versionPartsB := strings.Split(b, ".")

	for i := 0; i < 3; i++ {
		partA, _ := strconv.Atoi(versionPartsA[i])
		partB, _ := strconv.Atoi(versionPartsB[i])

		if partA != partB {
			return partA > partB
		}
	}

	return false
}
