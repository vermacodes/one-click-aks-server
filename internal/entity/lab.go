package entity

import (
	"actlabs/labentity"
	"context"
)

// Re-export shared types for backward compatibility within server
type TfvarResourceGroupType = labentity.TfvarResourceGroupType
type TfvarDefaultNodePoolType = labentity.TfvarDefaultNodePoolType
type TfvarServiceMeshType = labentity.TfvarServiceMeshType
type TfvarAddonsType = labentity.TfvarAddonsType
type TfvarKubernetesClusterType = labentity.TfvarKubernetesClusterType
type TfvarAroClusterType = labentity.TfvarAroClusterType
type TfvarVirtualNetworkType = labentity.TfvarVirtualNetworkType
type TfvarSubnetType = labentity.TfvarSubnetType
type TfvarNetworkSecurityGroupType = labentity.TfvarNetworkSecurityGroupType
type TfvarJumpserverType = labentity.TfvarJumpserverType
type TfvarFirewallType = labentity.TfvarFirewallType
type ContainerRegistryType = labentity.ContainerRegistryType
type AppGatewayType = labentity.AppGatewayType
type TfvarConfigType = labentity.TfvarConfigType
type Blob = labentity.Blob
type Blobs = labentity.Blobs
type EnumerationResults = labentity.EnumerationResults
type BlobType = labentity.BlobType
type LabType = labentity.LabType

type LabService interface {
	GetLabFromRedis(ctx context.Context) (LabType, error)
	SetLabInRedis(ctx context.Context, lab LabType) error
	DeleteLabFromRedis(ctx context.Context) error

	GetProtectedLab(ctx context.Context, typeOfLab string, labId string) (LabType, error)
	HelperDefaultLab(ctx context.Context) (LabType, error)
}

type LabRepository interface {
	GetLabFromRedis(ctx context.Context) (string, error)
	SetLabInRedis(ctx context.Context, lab string) error
	DeleteLabFromRedis(ctx context.Context) error

	GetProtectedLab(ctx context.Context, typeOfLab string, labId string) (string, error)

	GetExtendScriptTemplate(ctx context.Context) (string, error)
}
