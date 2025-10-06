package entity

import "context"

// Terraform workspace.
// Terraform workspace has a name and its selected or not.
// There is always a 'default' workspace and is selected by default.
type Workspace struct {
	Name     string `json:"name"`
	Selected bool   `json:"selected"`
}

type WorkspaceService interface {
	List(ctx context.Context) ([]Workspace, error)
	GetSelectedWorkspace(ctx context.Context) (Workspace, error)

	//Following mutating operations delete resources from redis.
	Add(ctx context.Context, workspace Workspace) error
	Select(ctx context.Context, workspace Workspace) error
	Delete(ctx context.Context, workspace Workspace) error

	// Resources of selected workspace
	Resources(ctx context.Context) (string, error)

	// Invalidate Cache
	DeleteAllWorkspaceFromRedis(ctx context.Context) error
}

// Implements Terraform Repository
// All the operations are done using a script or bash commands.
// Persistence of this is take care of by terraform itself.
type WorkspaceRepository interface {
	// List all the workspaces. Workspaces is the output sent in string.
	List(ctx context.Context, storageAccountName string, subscriptionId string) (string, error)

	GetListFromRedis(ctx context.Context) (string, error)
	AddListToRedis(ctx context.Context, val string)

	// All mutating operations on workspaces must delete list from redis.
	DeleteListFromRedis(ctx context.Context)

	// As a new workspace is added, it becomes the selected workspace.
	// Thats a terraform feature.
	// Client is expected to run List query after update is made.
	Add(ctx context.Context, workspace Workspace) error

	// Selects the workspace.
	Select(ctx context.Context, workspace Workspace) error

	// Deletes the workspace.
	Delete(ctx context.Context, workspace Workspace) error

	// Gets the resources in current selected workspace.
	// The Resources are just a string and thus returned as is.
	Resources(ctx context.Context, storageAccount string, subscriptionId string) (string, error)

	GetResourcesFromRedis(ctx context.Context) (string, error)
	AddResourcesToRedis(ctx context.Context, val string)

	// This is used to delete resources from redis.
	// All mutating terraform operations must delete resources
	// from redis to ensure fresh data.
	DeleteResourcesFromRedis(ctx context.Context)
}
