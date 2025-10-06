package service

import (
	"context"
	"strings"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
)

type workspaceService struct {
	workspaceRepository   entity.WorkspaceRepository
	storageAccountService entity.StorageAccountService // Some information is needed from storage account service.
	actionStatusService   entity.ActionStatusService
	authService           entity.AuthService
}

func NewWorkspaceService(workspaceRepo entity.WorkspaceRepository, storageAccountService entity.StorageAccountService, actionStatusService entity.ActionStatusService) entity.WorkspaceService {
	return &workspaceService{
		workspaceRepository:   workspaceRepo,
		storageAccountService: storageAccountService,
		actionStatusService:   actionStatusService,
	}
}

func (w *workspaceService) List(ctx context.Context) ([]entity.Workspace, error) {
	logging.LogInfo(ctx, "listing workspaces")
	workspaces := []entity.Workspace{}

	// Send workspaces from redis.
	val, err := w.workspaceRepository.GetListFromRedis(ctx)
	if err == nil {
		return helperStringToWorkspaces(ctx, val), nil
	}

	// rest of the function will be executed only if the workspace was not found in redis.

	storageAccountName, err := w.storageAccountService.GetStorageAccountName(ctx)

	if err != nil {
		logging.LogError(ctx, "Not able to get storage account name", "error", err)
		return workspaces, err
	}

	val, err = w.workspaceRepository.List(ctx, storageAccountName, w.authService.GetSubscriptionId(ctx))
	if err != nil {
		logging.LogError(ctx, "Not able to list workspaces", "error", err)
		return workspaces, err
	}

	if val == "" {
		logging.LogError(ctx, "No workspaces found", "error", err)
		return workspaces, err
	}

	// Adding workspaces in redis.
	w.workspaceRepository.AddListToRedis(ctx, val)

	return helperStringToWorkspaces(ctx, val), nil
}

func (w *workspaceService) GetSelectedWorkspace(ctx context.Context) (entity.Workspace, error) {
	logging.LogInfo(ctx, "getting selected workspace")
	workspaces, err := w.List(ctx)
	if err != nil {
		return entity.Workspace{}, err
	}

	for _, workspace := range workspaces {
		if workspace.Selected {
			return workspace, nil
		}
	}

	return entity.Workspace{}, nil

}

func (w *workspaceService) Add(ctx context.Context, workspace entity.Workspace) error {

	if err := w.workspaceRepository.Add(ctx, workspace); err != nil {
		logging.LogError(ctx, "not able to add workspace", "error", err)
		return err
	}

	// Since we just updated the workspaces, Redis info is now stale.
	// Remove it so that it gets updated correctly.
	w.workspaceRepository.DeleteListFromRedis(ctx)
	w.workspaceRepository.DeleteResourcesFromRedis(ctx)

	return nil
}

func (w *workspaceService) Select(ctx context.Context, workspace entity.Workspace) error {

	// add workspace if not exists

	if err := w.workspaceRepository.Select(ctx, workspace); err != nil {
		logging.LogError(ctx, "not able to select the workspace", "error", err)
		return err
	}

	w.workspaceRepository.DeleteListFromRedis(ctx)
	w.workspaceRepository.DeleteResourcesFromRedis(ctx)
	return nil
}

func (w *workspaceService) Delete(ctx context.Context, workspace entity.Workspace) error {
	if err := w.workspaceRepository.Delete(ctx, workspace); err != nil {
		logging.LogError(ctx, "not able to delete workspace", "error", err)
		return err
	}

	w.workspaceRepository.DeleteListFromRedis(ctx)
	w.workspaceRepository.DeleteResourcesFromRedis(ctx)
	return nil
}

func (w *workspaceService) Resources(ctx context.Context) (string, error) {
	logging.LogInfo(ctx, "getting resources")
	// Get resources from redis.
	resources, err := w.workspaceRepository.GetResourcesFromRedis(ctx)
	if err == nil {
		return resources, err
	}

	// rest of the function executes only if resources not found in redis.

	storageAccountName, err := w.storageAccountService.GetStorageAccountName(ctx)

	if err != nil {
		logging.LogError(ctx, "Not able to get storage account name", "error", err)
		return "", err
	}
	resources, err = w.workspaceRepository.Resources(ctx, storageAccountName, w.authService.GetSubscriptionId(ctx))
	if err != nil {
		logging.LogError(ctx, "not able to get resources", "error", err)
	}

	w.workspaceRepository.AddResourcesToRedis(ctx, resources)
	return resources, err
}

func (w *workspaceService) DeleteAllWorkspaceFromRedis(ctx context.Context) error {
	w.workspaceRepository.DeleteListFromRedis(ctx)
	w.workspaceRepository.DeleteResourcesFromRedis(ctx)
	return nil
}

// this is a helper function which takes a string (output from the command)
// and converts to a list of workspaces.
func helperStringToWorkspaces(ctx context.Context, val string) []entity.Workspace {
	logging.LogDebug(ctx, "workspaces : "+val)
	workspaces := []entity.Workspace{}
	sliceOut := strings.Split(string(val), ",")
	for _, v := range sliceOut {

		var workspace = new(entity.Workspace)
		workspace.Selected = false

		// Check for selected workspace and remove leading [* ] or remove the leading space
		if strings.HasPrefix(v, "*") {
			workspace.Selected = true
			v = strings.Split(v, "* ")[1]
		} else {
			// Removes the leading whitespace.
			v = strings.Split(v, " ")[1]
		}

		// Add workspace name.
		workspace.Name = v

		// Append workspace.
		workspaces = append(workspaces, *workspace)
	}

	return workspaces
}
