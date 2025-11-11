package service

import (
	"context"

	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"
)

type DeploymentService struct {
	deploymentRepository entity.DeploymentRepository
	labService           entity.LabService
	terraformService     entity.TerraformService
	workspaceService     entity.WorkspaceService
	actionStatusService  entity.ActionStatusService
	logstreamService     entity.LogStreamService
	authService          entity.AuthService
	config               config.Config
}

func NewDeploymentService(deploymentRepo entity.DeploymentRepository,
	labService entity.LabService,
	terraformService entity.TerraformService,
	actionStatusService entity.ActionStatusService,
	logstreamService entity.LogStreamService,
	authService entity.AuthService,
	workspaceService entity.WorkspaceService,
	config config.Config) entity.DeploymentService {
	return &DeploymentService{
		deploymentRepository: deploymentRepo,
		labService:           labService,
		terraformService:     terraformService,
		actionStatusService:  actionStatusService,
		logstreamService:     logstreamService,
		authService:          authService,
		workspaceService:     workspaceService,
		config:               config,
	}
}

func (d *DeploymentService) GetDeployments(ctx context.Context) ([]entity.Deployment, error) {
	return d.deploymentRepository.GetDeployments(ctx)
}

func (d *DeploymentService) GetMyDeployments(ctx context.Context, userId string) ([]entity.Deployment, error) {

	// get all deployments
	deployments, err := d.deploymentRepository.GetMyDeployments(ctx, userId, d.authService.GetSubscriptionId(ctx))
	if err != nil {
		return nil, err
	}

	// filter deployments for active account
	var filteredDeployments []entity.Deployment
	for _, deployment := range deployments {
		logging.LogDebug(ctx, "Deployment Filter", "workspace", deployment.DeploymentWorkspace, "subscription", deployment.DeploymentSubscriptionId, "active", d.authService.GetSubscriptionId(ctx))
		if deployment.DeploymentSubscriptionId == d.authService.GetSubscriptionId(ctx) {
			filteredDeployments = append(filteredDeployments, deployment)
		}
	}

	// if no deployments found for active account, create default deployment.
	if len(filteredDeployments) == 0 {

		defaultLab, err := d.labService.HelperDefaultLab(ctx)
		if err != nil {
			return nil, err
		}

		deployment := entity.Deployment{
			DeploymentUserId:             userId,
			DeploymentWorkspace:          "default",
			DeploymentSubscriptionId:     d.authService.GetSubscriptionId(ctx),
			DeploymentId:                 userId + "-default-" + d.authService.GetSubscriptionId(ctx),
			DeploymentLab:                defaultLab,
			DeploymentAutoDelete:         false,
			DeploymentLifespan:           28800,
			DeploymentAutoDeleteUnixTime: 0,
		}

		if err := d.deploymentRepository.UpsertDeployment(ctx, deployment); err != nil {
			return nil, err
		}

		filteredDeployments = append(filteredDeployments, deployment)
	}

	return filteredDeployments, err
}

func (d *DeploymentService) GetDeployment(ctx context.Context, userId string, workspace string, subscriptionId string) (entity.Deployment, error) {
	return d.deploymentRepository.GetDeployment(ctx, userId, workspace, subscriptionId)
}

func (d *DeploymentService) GetSelectedDeployment(ctx context.Context) (entity.Deployment, error) {
	// get selected workspace
	selectedWorkspace, err := d.workspaceService.GetSelectedWorkspace(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get selected workspace", err)
		return entity.Deployment{}, err
	}

	//Get user principal from env variable.
	userPrincipal := helper.GetUserIDFromContext(ctx)

	//Get all deployments.
	deployments, err := d.GetMyDeployments(ctx, userPrincipal)
	if err != nil {
		logging.LogError(ctx, "not able to get deployments", err)
		return entity.Deployment{}, nil
	}

	// Find the deployment for the selected workspace.
	for _, deployment := range deployments {
		if deployment.DeploymentWorkspace == selectedWorkspace.Name {
			return deployment, nil
		}
	}

	return entity.Deployment{}, nil
}

func (d *DeploymentService) SelectDeployment(ctx context.Context, deployment entity.Deployment) error {

	// check if workspace exists, if not add it.
	if err := checkAndAddWorkspace(ctx, d, &entity.Deployment{DeploymentWorkspace: deployment.DeploymentWorkspace}); err != nil {
		return err
	}

	// set workspace as selected.
	if err := d.workspaceService.Select(ctx, entity.Workspace{Name: deployment.DeploymentWorkspace, Selected: true}); err != nil {
		logging.LogError(ctx, "not able to select workspace", err)
		return err
	}

	return nil
}

func (d *DeploymentService) UpsertDeployment(ctx context.Context, deployment entity.Deployment) error {
	deployment.DeploymentSubscriptionId = d.authService.GetSubscriptionId(ctx)

	// check if workspace exists, if not add it.
	if err := checkAndAddWorkspace(ctx, d, &deployment); err != nil {
		return err
	}

	return d.deploymentRepository.UpsertDeployment(ctx, deployment)

}

func (d *DeploymentService) DeleteDeployment(ctx context.Context, userId string, workspace string, subscriptionId string) error {

	// default deployment cant be deleted.
	if workspace == "default" {
		return nil
	}

	// select default workspace
	if err := d.workspaceService.Select(ctx, entity.Workspace{Name: "default", Selected: true}); err != nil {
		logging.LogError(ctx, "not able to select workspace", err)
		return err
	}

	// delete workspace
	if err := d.workspaceService.Delete(ctx, entity.Workspace{Name: workspace}); err != nil {
		logging.LogError(ctx, "not able to delete workspace", err)
		return err
	}

	return d.deploymentRepository.DeleteDeployment(ctx, userId, workspace, subscriptionId)
}

func (d *DeploymentService) ChangeTerraformWorkspace(ctx context.Context, deployment entity.Deployment) error {
	// change terraform workspace if not same as deployments
	workspaces, err := d.workspaceService.List(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get workspaces", err)
		return err
	}
	selectedWorkspace := entity.Workspace{}
	for _, workspace := range workspaces {
		if workspace.Selected {
			selectedWorkspace = workspace
		}
	}
	if selectedWorkspace.Name != deployment.DeploymentWorkspace {
		logging.LogInfo(ctx, "changing workspace to "+deployment.DeploymentWorkspace)
		if err := d.workspaceService.Select(ctx, entity.Workspace{Name: deployment.DeploymentWorkspace}); err != nil {
			logging.LogError(ctx, "not able to select workspace", err)
			return err
		}
	}
	return nil
}

func checkAndAddWorkspace(ctx context.Context, d *DeploymentService, deployment *entity.Deployment) error {
	// check if workspace exists, if not add it.
	workspaces, err := d.workspaceService.List(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get workspaces", err)
		return err
	}

	workspaceExists := false
	for _, workspace := range workspaces {
		if workspace.Name == deployment.DeploymentWorkspace {
			workspaceExists = true
			break
		}
	}

	if !workspaceExists {
		logging.LogInfo(ctx, "adding workspace "+deployment.DeploymentWorkspace)
		if err := d.workspaceService.Add(ctx, entity.Workspace{Name: deployment.DeploymentWorkspace}); err != nil {
			logging.LogError(ctx, "not able to add workspace", err)
			return err
		}
	}

	return nil
}
