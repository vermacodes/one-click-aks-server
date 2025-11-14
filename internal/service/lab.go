package service

import (
	"context"
	"encoding/json"
	"fmt"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
)

type labService struct {
	labRepository         entity.LabRepository
	kVersionService       entity.KVersionService
	aroVersionService     entity.AROVersionService
	storageAccountService entity.StorageAccountService // Some information is needed from storage account service.
	authService           entity.AuthService
}

func NewLabService(repo entity.LabRepository, kVersionService entity.KVersionService, aroVersionService entity.AROVersionService, storageAccountService entity.StorageAccountService, authService entity.AuthService) entity.LabService {
	return &labService{
		labRepository:         repo,
		kVersionService:       kVersionService,
		aroVersionService:     aroVersionService,
		storageAccountService: storageAccountService,
		authService:           authService,
	}
}

func (l *labService) GetLabFromRedis(ctx context.Context) (entity.LabType, error) {
	logging.LogInfo(ctx, "starting lab retrieval from Redis")

	lab := entity.LabType{}
	out, err := l.labRepository.GetLabFromRedis(ctx)
	if err != nil {
		// If the lab was not found in redis then we will set to default.
		logging.LogInfo(ctx, "lab not found in Redis, creating default lab")

		defaultLab, err := l.HelperDefaultLab(ctx)
		if err != nil {
			logging.LogError(ctx, "failed to generate default lab", "error", err.Error())
			return lab, err
		}

		if err := l.SetLabInRedis(ctx, defaultLab); err != nil {
			logging.LogError(ctx, "failed to set default lab in Redis", "error", err.Error())
		}

		logging.LogInfo(ctx, "default lab created and cached successfully")
		return defaultLab, nil
	}

	if err := json.Unmarshal([]byte(out), &lab); err != nil {
		logging.LogError(ctx, "failed to unmarshal lab from Redis", "error", err.Error())
		return lab, err
	}

	logging.LogInfo(ctx, "lab retrieved from Redis successfully")
	return lab, nil
}

func (l *labService) SetLabInRedis(ctx context.Context, lab entity.LabType) error {
	logging.LogInfo(ctx, "starting lab configuration in Redis")

	for i := range lab.Template.KubernetesClusters {
		if lab.Template.KubernetesClusters[i].KubernetesVersion == "" {
			lab.Template.KubernetesClusters[i].KubernetesVersion = l.kVersionService.GetDefaultVersion(ctx)
		}
	}

	for i := range lab.Template.AroClusters {
		if lab.Template.AroClusters[i].Version == "" {
			lab.Template.AroClusters[i].Version = l.aroVersionService.GetDefaultAROVersion(ctx)
		}
	}

	val, err := json.Marshal(lab)
	if err != nil || string(val) == "" {
		logging.LogError(ctx, "failed to marshal lab object", "error", err.Error())
		return err
	}

	if err := l.labRepository.SetLabInRedis(ctx, string(val)); err != nil {
		logging.LogError(ctx, "failed to store lab in Redis", "error", err.Error())
		return err
	}

	logging.LogInfo(ctx, "lab configured in Redis successfully")
	return nil
}

func (l *labService) DeleteLabFromRedis(ctx context.Context) error {
	logging.LogInfo(ctx, "starting lab deletion from Redis")

	err := l.labRepository.DeleteLabFromRedis(ctx)
	if err != nil {
		logging.LogError(ctx, "failed to delete lab from Redis", "error", err.Error())
		return err
	}

	logging.LogInfo(ctx, "lab deleted from Redis successfully")
	return nil
}

func (l *labService) GetProtectedLab(ctx context.Context, typeOfLab string, labId string) (entity.LabType, error) {
	logging.LogInfo(ctx, "starting protected lab retrieval",
		"typeOfLab", typeOfLab,
		"labId", labId,
	)

	lab := entity.LabType{}

	if labId == "" || typeOfLab == "" {
		logging.LogError(ctx, "validation failed: required parameters are empty",
			"typeOfLab", typeOfLab,
			"labId", labId,
		)
		return lab, fmt.Errorf("required typeOfLab or labId is empty")
	}

	originalTypeOfLab := l.OriginalTypeOfLab(ctx, typeOfLab)

	// http call to actlabs-auth
	labString, err := l.labRepository.GetProtectedLab(ctx, originalTypeOfLab, labId)
	if err != nil {
		logging.LogError(ctx, "failed to retrieve protected lab from repository",
			"typeOfLab", originalTypeOfLab,
			"labId", labId,
			"error", err.Error(),
		)
		return lab, fmt.Errorf("failed to get protected lab: %s", err.Error())
	}

	if err := json.Unmarshal([]byte(labString), &lab); err != nil {
		logging.LogError(ctx, "failed to unmarshal protected lab",
			"typeOfLab", originalTypeOfLab,
			"labId", labId,
			"error", err.Error(),
		)
		return lab, fmt.Errorf("failed to unmarshal lab object: %s", err.Error())
	}

	if lab.ExtendScript == "redacted" || lab.ExtendScript == "" {
		logging.LogError(ctx, "business rule violation: extend script is redacted or empty",
			"labId", labId,
			"labName", lab.Name,
			"labType", lab.Type,
		)
		return lab, fmt.Errorf("extend script is not available for this lab")
	}

	lab.Type = l.RedactedTypeOfLab(ctx, lab.Type)

	logging.LogInfo(ctx, "protected lab retrieved successfully",
		"labId", labId,
		"labName", lab.Name,
	)
	return lab, nil
}

func (l *labService) OriginalTypeOfLab(ctx context.Context, typeOfLab string) string {
	// change typeOfLab to match the real type of lab
	if typeOfLab == "assignment" {
		return "readinesslab"
	}
	if typeOfLab == "challenge" {
		return "challengelab"
	}

	return typeOfLab
}

func (l *labService) RedactedTypeOfLab(ctx context.Context, typeOfLab string) string {
	// change typeOfLab to match the real type of lab
	if typeOfLab == "readinesslab" {
		return "assignment"
	}
	if typeOfLab == "challengelab" {
		return "challenge"
	}

	return typeOfLab
}

func (l *labService) HelperDefaultLab(ctx context.Context) (entity.LabType, error) {
	logging.LogInfo(ctx, "creating default lab configuration")

	var defaultResourceGroup = entity.TfvarResourceGroupType{
		Location: "East US",
	}

	var defaultNodePool = entity.TfvarDefaultNodePoolType{
		EnableAutoScaling:         false,
		MinCount:                  1,
		MaxCount:                  1,
		VmSize:                    "UserDefaultVMSize",
		OnlyCriticalAddonsEnabled: false,
		OsSku:                     "Ubuntu",
	}

	var defaultServiceMesh = entity.TfvarServiceMeshType{
		Enabled:                       false,
		Mode:                          "Istio",
		InternalIngressGatewayEnabled: false,
		ExternalIngressGatewayEnabled: false,
	}

	var defaultAddons = entity.TfvarAddonsType{
		AppGateway:             false,
		MicrosoftDefender:      false,
		VirtualNode:            false,
		HttpApplicationRouting: false,
		ServiceMesh:            defaultServiceMesh,
	}

	var defaultKubernetesClusters = []entity.TfvarKubernetesClusterType{
		{
			KubernetesVersion:       l.kVersionService.GetDefaultVersion(ctx),
			NetworkPlugin:           "kubenet",
			NetworkPolicy:           "null",
			NetworkPluginMode:       "null",
			OutboundType:            "loadBalancer",
			PrivateClusterEnabled:   "false",
			OidcIssuerEnabled:       false,
			WorkloadIdentityEnabled: false,
			Addons:                  defaultAddons,
			DefaultNodePool:         defaultNodePool,
		},
	}

	var defaultTfvar = entity.TfvarConfigType{
		ResourceGroup:         defaultResourceGroup,
		KubernetesClusters:    defaultKubernetesClusters,
		AroClusters:           []entity.TfvarAroClusterType{},
		VirtualNetworks:       []entity.TfvarVirtualNetworkType{},
		NetworkSecurityGroups: []entity.TfvarNetworkSecurityGroupType{},
		Subnets:               []entity.TfvarSubnetType{},
		Jumpservers:           []entity.TfvarJumpserverType{},
		Firewalls:             []entity.TfvarFirewallType{},
		ContainerRegistries:   []entity.ContainerRegistryType{},
		AppGateways:           []entity.AppGatewayType{},
	}

	extendScript, err := l.labRepository.GetExtendScriptTemplate(ctx)
	if err != nil {
		logging.LogError(ctx, "failed to get extend script template, using empty string", "error", err.Error())
		extendScript = ""
	}

	var defaultLab = entity.LabType{
		Tags:         []string{},
		Template:     defaultTfvar,
		Type:         "privatelab",
		ExtendScript: extendScript,
	}

	logging.LogInfo(ctx, "default lab configuration created successfully")
	return defaultLab, nil
}
