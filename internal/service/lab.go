package service

import (
	"encoding/json"
	"fmt"

	"one-click-aks-server/internal/entity"

	"golang.org/x/exp/slog"
)

type labService struct {
	labRepository         entity.LabRepository
	kVersionService       entity.KVersionService
	storageAccountService entity.StorageAccountService // Some information is needed from storage account service.
	authService           entity.AuthService
}

func NewLabService(repo entity.LabRepository, kVersionService entity.KVersionService, storageAccountService entity.StorageAccountService, authService entity.AuthService) entity.LabService {
	return &labService{
		labRepository:         repo,
		kVersionService:       kVersionService,
		storageAccountService: storageAccountService,
		authService:           authService,
	}
}

func (l *labService) GetLabFromRedis() (entity.LabType, error) {
	lab := entity.LabType{}
	out, err := l.labRepository.GetLabFromRedis()
	if err != nil {

		// If the lab was not found in redis then we will set to default.

		slog.Info("lab not found in redis. Setting default.")

		defaultLab, err := l.HelperDefaultLab()
		if err != nil {
			slog.Error("not able to generate default lab", err)
			return lab, err
		}

		if err := l.SetLabInRedis(defaultLab); err != nil {
			slog.Error("not able to set default lab in redis.", err)
		}

		return defaultLab, nil
	}
	slog.Debug("lab found in redis")

	if err := json.Unmarshal([]byte(out), &lab); err != nil {
		slog.Error("not able to unmarshal lab in redis to object", err)
	}

	return lab, nil
}

func (l *labService) SetLabInRedis(lab entity.LabType) error {

	for i := range lab.Template.KubernetesClusters {
		if lab.Template.KubernetesClusters[i].KubernetesVersion == "" {
			lab.Template.KubernetesClusters[i].KubernetesVersion = l.kVersionService.GetDefaultVersion()
		}
	}

	val, err := json.Marshal(lab)
	if err != nil || string(val) == "" {
		slog.Error("not able to marshal object", err)
		return err
	}

	if err := l.labRepository.SetLabInRedis(string(val)); err != nil {
		slog.Error("not able set lab in redis", err)
		return err
	}

	return nil
}

func (l *labService) DeleteLabFromRedis() error {
	return l.labRepository.DeleteLabFromRedis()
}

func (l *labService) GetProtectedLab(typeOfLab string, labId string) (entity.LabType, error) {
	slog.Info("getting protected lab",
		slog.String("typeOfLab", typeOfLab),
		slog.String("labId", labId),
	)

	lab := entity.LabType{}

	if labId == "" || typeOfLab == "" {
		slog.Error("required typeOfLab or labId is empty",
			slog.String("typeOfLab", typeOfLab),
			slog.String("labId", labId),
		)
		return lab, fmt.Errorf("required typeOfLab or labId is empty")
	}

	typeOfLab = l.OriginalTypeOfLab(typeOfLab)

	slog.Info("getting protected lab (original typeOfLab)",
		slog.String("typeOfLab", typeOfLab),
		slog.String("labId", labId),
	)

	// http call to actlabs-auth
	labString, err := l.labRepository.GetProtectedLab(typeOfLab, labId)
	if err != nil {
		slog.Error("not able to get protected lab request",
			slog.String("typeOfLab", typeOfLab),
			slog.String("labId", labId),
			slog.String("error", err.Error()),
		)
		return lab, fmt.Errorf("not able to get protected %s", err.Error())
	}

	if err := json.Unmarshal([]byte(labString), &lab); err != nil {
		slog.Error("not able to unmarshal lab object",
			slog.String("typeOfLab", typeOfLab),
			slog.String("labId", labId),
			slog.String("error", err.Error()),
		)
		return lab, fmt.Errorf("not able to unmarshal lab object %s", err.Error())
	}

	if lab.ExtendScript == "redacted" || lab.ExtendScript == "" {
		slog.Error("got the lab, but the extend script is redacted or empty",
			slog.String("labId", labId),
			slog.String("labName", lab.Name),
			slog.String("labType", lab.Type),
			slog.String("extendScript", lab.ExtendScript),
		)

		return lab, fmt.Errorf("got the lab, but the extend script is redacted or empty")
	}

	lab.Type = l.RedactedTypeOfLab(lab.Type)

	return lab, nil
}

func (l *labService) OriginalTypeOfLab(typeOfLab string) string {
	// change typeOfLab to match the real type of lab
	if typeOfLab == "assignment" {
		return "readinesslab"
	}
	if typeOfLab == "challenge" {
		return "challengelab"
	}

	return typeOfLab
}

func (l *labService) RedactedTypeOfLab(typeOfLab string) string {
	// change typeOfLab to match the real type of lab
	if typeOfLab == "readinesslab" {
		return "assignment"
	}
	if typeOfLab == "challengelab" {
		return "challenge"
	}

	return typeOfLab
}

func (l *labService) HelperDefaultLab() (entity.LabType, error) {

	var defaultResourceGroup = entity.TfvarResourceGroupType{
		Location: "East US",
	}

	var defaultNodePool = entity.TfvarDefaultNodePoolType{
		EnableAutoScaling: false,
		MinCount:          1,
		MaxCount:          1,
    VmSize:           "Standard_D2_v5",
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
			KubernetesVersion:     l.kVersionService.GetDefaultVersion(),
			NetworkPlugin:         "kubenet",
			NetworkPolicy:         "null",
			NetworkPluginMode:     "null",
			OutboundType:          "loadBalancer",
			PrivateClusterEnabled: "false",
			Addons:                defaultAddons,
			DefaultNodePool:       defaultNodePool,
		},
	}

	var defaultTfvar = entity.TfvarConfigType{
		ResourceGroup:         defaultResourceGroup,
		KubernetesClusters:    defaultKubernetesClusters,
		VirtualNetworks:       []entity.TfvarVirtualNetworkType{},
		NetworkSecurityGroups: []entity.TfvarNetworkSecurityGroupType{},
		Subnets:               []entity.TfvarSubnetType{},
		Jumpservers:           []entity.TfvarJumpserverType{},
		Firewalls:             []entity.TfvarFirewallType{},
		ContainerRegistries:   []entity.ContainerRegistryType{},
		AppGateways:           []entity.AppGatewayType{},
	}

	extendScript, err := l.labRepository.GetExtendScriptTemplate()
	if err != nil {
		slog.Error("Not able to get extend script template. Defaulting to empty string.", err)
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
