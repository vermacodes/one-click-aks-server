package service

import (
	"context"
	"os"
	"strconv"
	"time"

	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
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
	userPrincipal := os.Getenv("ARM_USER_PRINCIPAL_NAME")

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

func (d *DeploymentService) PollAndDeleteDeployments(interval time.Duration) error {

	ctx := context.WithValue(context.Background(), logging.UserIDKey, "pool_and_delete_deployment_bg_service")

	dataChannel := make(chan []entity.Deployment)
	go func() {
		for {
			deployments := d.FetchDeploymentsToBeDeleted(ctx)
			dataChannel <- deployments
			logging.LogDebug(ctx, "polling for deployments to be deleted found "+strconv.Itoa(len(deployments))+" deployments")
			time.Sleep(interval)
		}
	}()

	for {
		deployments := <-dataChannel
		for _, deployment := range deployments {
			logging.LogInfo(ctx, "deleting deployment "+deployment.DeploymentWorkspace)

			actionStatus, err := d.actionStatusService.GetActionStatus(ctx)
			if err != nil {
				logging.LogError(ctx, "not able to get action status", err)
				continue
			}

			// Wait for any action in progress to complete.

			for {
				if actionStatus.InProgress {
					logging.LogInfo(ctx, "action in progress. waiting for 60 seconds")
					time.Sleep(60 * time.Second)
					actionStatus, err = d.actionStatusService.GetActionStatus(ctx)
					if err != nil {
						logging.LogError(ctx, "not able to get action status", err)
						continue
					}
					continue
				}
				break
			}

			// Get the current workspace.
			prevSelectedDeployment, err := d.GetSelectedDeployment(ctx)
			if err != nil {
				logging.LogError(ctx, "not able to get current workspace", err)
				continue
			}

			// Change terraform workspace.
			if err := d.ChangeTerraformWorkspace(ctx, deployment); err != nil {
				logging.LogError(ctx, "not able to change terraform workspace", err)
				continue
			}

			// Update deployment status to deleting.
			deployment.DeploymentStatus = entity.DestroyInProgress
			if err := d.UpsertDeployment(ctx, deployment); err != nil {
				logging.LogError(ctx, "not able to update deployment", err)
				continue
			}

			// Update action status to in progress.
			d.actionStatusService.SetActionStart(ctx)

			//Run extend script in 'destroy' mode.
			if err := d.terraformService.Extend(ctx, deployment.DeploymentLab, "destroy"); err != nil {
				logging.LogError(ctx, "not able to run extend script", err)

				// Update deployment status to failed.
				deployment.DeploymentStatus = entity.DestroyFailed
				if err := d.UpsertDeployment(ctx, deployment); err != nil {
					logging.LogError(ctx, "not able to update deployment", "error", err)
				}

				d.actionStatusService.SetActionEnd(ctx)
				continue
			}

			// Run terraform destroy.
			if err := d.terraformService.Destroy(ctx, deployment.DeploymentLab); err != nil {
				logging.LogError(ctx, "not able to run terraform destroy", err)

				// Update deployment status to failed.
				deployment.DeploymentStatus = entity.DestroyFailed
				if err := d.UpsertDeployment(ctx, deployment); err != nil {
					logging.LogError(ctx, "not able to update deployment", err)
				}

				d.actionStatusService.SetActionEnd(ctx)
				continue
			}

			// Update deployment status to destroyed.
			deployment.DeploymentStatus = entity.DestroyCompleted
			if err := d.UpsertDeployment(ctx, deployment); err != nil {
				logging.LogError(ctx, "not able to update deployment", err)
				d.actionStatusService.SetActionEnd(ctx)
				continue
			}

			// Change back to the original workspace.
			if err := d.ChangeTerraformWorkspace(ctx, prevSelectedDeployment); err != nil {
				logging.LogError(ctx, "not able to change back to original workspace", err)
				continue
			}

			d.actionStatusService.SetActionEnd(ctx)
		}
	}
}

func (d *DeploymentService) FetchDeploymentsToBeDeleted(ctx context.Context) []entity.Deployment {
	//Get user principal from env variable.
	userPrincipal := os.Getenv("ARM_USER_PRINCIPAL_NAME")

	//Get all deployments.
	deployments, err := d.GetMyDeployments(ctx, userPrincipal)
	if err != nil {
		logging.LogError(ctx, "not able to get deployments", err)
		return nil
	}

	// Filter deployments where auto delete is true and auto delete unix time is less than current unix time.
	var deploymentsToBeDeleted []entity.Deployment

	for _, deployment := range deployments {
		currentEpochTime := time.Now().Unix()
		logging.LogDebug(ctx, "currentEpochTime: "+strconv.FormatInt(currentEpochTime, 10))
		if deployment.DeploymentAutoDelete &&
			deployment.DeploymentAutoDeleteUnixTime < currentEpochTime &&
			deployment.DeploymentAutoDeleteUnixTime != 0 &&
			(deployment.DeploymentStatus == entity.DeploymentCompleted ||
				deployment.DeploymentStatus == entity.DeploymentFailed) {
			deploymentsToBeDeleted = append(deploymentsToBeDeleted, deployment)
		}
	}

	return deploymentsToBeDeleted
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
