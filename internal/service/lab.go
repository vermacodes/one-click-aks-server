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
	lab := entity.LabType{}
	out, err := l.labRepository.GetLabFromRedis(ctx)
	if err != nil {

		// If the lab was not found in redis then we will set to default.

		logging.LogDebug(ctx, "lab not found in redis, setting default.")

		defaultLab, err := l.HelperDefaultLab(ctx)
		if err != nil {
			logging.LogError(ctx, "not able to generate default lab", "error", err)
			return lab, err
		}

		if err := l.SetLabInRedis(ctx, defaultLab); err != nil {
			logging.LogError(ctx, "not able to set default lab in redis.", "error", err)
		}

		return defaultLab, nil
	}
	logging.LogDebug(ctx, "lab found in redis")

	if err := json.Unmarshal([]byte(out), &lab); err != nil {
		logging.LogError(ctx, "not able to unmarshal lab in redis to object", "error", err)
	}

	return lab, nil
}

func (l *labService) SetLabInRedis(ctx context.Context, lab entity.LabType) error {

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
		logging.LogError(ctx, "not able to marshal object", "error", err)
		return err
	}

	if err := l.labRepository.SetLabInRedis(ctx, string(val)); err != nil {
		logging.LogError(ctx, "not able set lab in redis", "error", "error", err)
		return err
	}

	return nil
}

func (l *labService) DeleteLabFromRedis(ctx context.Context) error {
	return l.labRepository.DeleteLabFromRedis(ctx)
}

func (l *labService) GetProtectedLab(ctx context.Context, typeOfLab string, labId string) (entity.LabType, error) {
	logging.LogInfo(ctx, "getting protected lab",
		"typeOfLab", typeOfLab,
		"labId", labId,
	)

	lab := entity.LabType{}

	if labId == "" || typeOfLab == "" {
		logging.LogError(ctx, "required typeOfLab or labId is empty",
			"typeOfLab", typeOfLab,
			"labId", labId,
		)
		return lab, fmt.Errorf("required typeOfLab or labId is empty")
	}

	typeOfLab = l.OriginalTypeOfLab(ctx, typeOfLab)

	logging.LogInfo(ctx, "getting protected lab (original typeOfLab)",
		"typeOfLab", typeOfLab,
		"labId", labId,
	)

	// http call to actlabs-auth
	labString, err := l.labRepository.GetProtectedLab(ctx, typeOfLab, labId)
	if err != nil {
		logging.LogError(ctx, "not able to get protected lab request",
			"typeOfLab", typeOfLab,
			"labId", labId,
			"error", err.Error(),
		)
		return lab, fmt.Errorf("not able to get protected %s", err.Error())
	}

	if err := json.Unmarshal([]byte(labString), &lab); err != nil {
		logging.LogError(ctx, "not able to unmarshal lab object",
			"typeOfLab", typeOfLab,
			"labId", labId,
			"error", err.Error(),
		)
		return lab, fmt.Errorf("not able to unmarshal lab object %s", err.Error())
	}

	if lab.ExtendScript == "redacted" || lab.ExtendScript == "" {
		logging.LogError(ctx, "got the lab, but the extend script is redacted or empty",
			"labId", labId,
			"labName", lab.Name,
			"labType", lab.Type,
			"extendScript", lab.ExtendScript,
		)

		return lab, fmt.Errorf("got the lab, but the extend script is redacted or empty")
	}

	lab.Type = l.RedactedTypeOfLab(ctx, lab.Type)

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

	var defaultResourceGroup = entity.TfvarResourceGroupType{
		Location: "East US",
	}

	var defaultNodePool = entity.TfvarDefaultNodePoolType{
		EnableAutoScaling:         false,
		MinCount:                  1,
		MaxCount:                  1,
		VmSize:                    "Standard_D2_v5",
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
		logging.LogError(ctx, "Not able to get extend script template. Defaulting to empty string.", err)
		extendScript = ""
	}

	var defaultLab = entity.LabType{
		Tags:         []string{},
		Template:     defaultTfvar,
		Type:         "privatelab",
		ExtendScript: extendScript,
	}

	return defaultLab, nil
}
