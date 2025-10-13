package handler

import (
	"net/http"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"one-click-aks-server/internal/helper"
)

type terraformHandler struct {
	terraformService    entity.TerraformService
	actionStatusService entity.ActionStatusService
	deploymentService   entity.DeploymentService
	workspaceService    entity.WorkspaceService
}

func NewTerraformWithActionStatusHandler(r *gin.RouterGroup,
	service entity.TerraformService,
	actionStatusService entity.ActionStatusService,
	deploymentService entity.DeploymentService,
	workspaceService entity.WorkspaceService) {
	handler := &terraformHandler{
		terraformService:    service,
		actionStatusService: actionStatusService,
		deploymentService:   deploymentService,
		workspaceService:    workspaceService,
	}

	r.POST("/terraform/init/:operationId", handler.Init)
	r.POST("/terraform/plan/:operationId", handler.Plan)
	r.POST("/terraform/apply/:operationId", handler.Apply)
	r.POST("/terraform/destroy/:operationId", handler.Destroy)
	r.POST("/terraform/extend/:mode/:operationId", handler.Extend)
}

func NewTerraformWithAPIKeyAndActionStatusHandler(r *gin.RouterGroup,
	service entity.TerraformService,
	actionStatusService entity.ActionStatusService,
	deploymentService entity.DeploymentService,
	workspaceService entity.WorkspaceService) {
	handler := &terraformHandler{
		terraformService:    service,
		actionStatusService: actionStatusService,
		deploymentService:   deploymentService,
		workspaceService:    workspaceService,
	}

	r.POST("/api/terraform/destroy/:operationId", handler.Destroy)
}

func (t *terraformHandler) Init(c *gin.Context) {
	notification := entity.ServerNotification{
		Id:               uuid.New().String(),
		NotificationType: entity.Info,
		Message:          string(entity.InitInProgress),
		AutoClose:        2000,
	}

	if err := t.actionStatusService.SetServerNotification(c.Request.Context(), notification); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}

	// Create a background context for long-running operation
	bgCtx := logging.CreateBackgroundContextWithValues(c.Request.Context())

	// Start the long-running operation in a goroutine
	go func() {
		// Ensure action always ends when goroutine exits
		defer func() {
			if err := t.actionStatusService.SetActionEnd(bgCtx); err != nil {
				logging.LogError(bgCtx, "error setting action end", "error", err)
			}
		}()

		t.actionStatusService.SetActionStart(bgCtx)
		if err := t.terraformService.Init(bgCtx); err != nil {
			notification.NotificationType = entity.Error
			notification.Message = string(entity.InitFailed)
		} else {
			notification.NotificationType = entity.Success
			notification.Message = string(entity.InitCompleted)
		}
		if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
			logging.LogError(bgCtx, "error setting server notification", "error", err)
		}
	}()

	// Respond back to the request with the operation ID
	c.IndentedJSON(http.StatusOK, notification)
}

func (t *terraformHandler) Plan(c *gin.Context) {
	deployment := entity.Deployment{}
	if err := c.Bind(&deployment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lab := deployment.DeploymentLab

	notification := entity.ServerNotification{
		Id:               uuid.New().String(),
		NotificationType: entity.Info,
		Message:          string(entity.PlanInProgress),
		AutoClose:        2000,
	}

	if err := t.actionStatusService.SetServerNotification(c.Request.Context(), notification); err != nil {
		logging.LogError(c.Request.Context(), "error setting server notification", "error", err)
	}

	// Start the long-running operation in a goroutine
	bgCtx := logging.CreateBackgroundContextWithValues(c.Request.Context())

	go func() {
		// Ensure action always ends when goroutine exits
		defer func() {
			if err := t.actionStatusService.SetActionEnd(bgCtx); err != nil {
				logging.LogError(bgCtx, "error setting action end", "error", err)
			}
		}()

		// background context for long running operation
		t.actionStatusService.SetActionStart(bgCtx)

		// Ensure terraform is initialized
		if err := t.terraformService.EnsureInit(bgCtx); err != nil {
			logging.LogError(bgCtx, "error initializing terraform", "error", err)
		}

		// This doesn't change deployment status, just resets the workspaces.
		if err := t.deploymentService.UpsertDeployment(bgCtx, deployment); err != nil {
			logging.LogError(bgCtx, "error updating deployment", "error", err)
		}

		if err := t.terraformService.Plan(bgCtx, lab); err != nil {
			notification.NotificationType = entity.Error
			notification.Message = string(entity.PlanFailed)
		} else {
			notification.NotificationType = entity.Success
			notification.Message = string(entity.PlanCompleted)
		}
		if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
			logging.LogError(bgCtx, "error setting server notification", "error", err)
		}
	}()

	// Respond back to the request with the operation ID
	c.Status(http.StatusAccepted)
}

func (t *terraformHandler) Apply(c *gin.Context) {

	deployment := entity.Deployment{}
	if err := c.Bind(&deployment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lab := deployment.DeploymentLab

	notification := entity.ServerNotification{
		Id:               uuid.New().String(),
		NotificationType: entity.Info,
		Message:          string(entity.DeploymentInProgress),
		AutoClose:        2000,
	}

	// background context for long running operation
	bgCtx := logging.CreateBackgroundContextWithValues(c.Request.Context())

	// Start the long-running operation in a goroutine
	go func() {
		// Ensure action always ends when goroutine exits
		defer func() {
			if err := t.actionStatusService.SetActionEnd(bgCtx); err != nil {
				logging.LogError(bgCtx, "error setting action end", "error", err)
			}
		}()

		// Set Action start.
		t.actionStatusService.SetActionStart(bgCtx)
		if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
			logging.LogError(bgCtx, "error setting server notification", "error", err)
		}

		// Ensure terraform is initialized
		if err := t.terraformService.EnsureInit(bgCtx); err != nil {
			logging.LogError(bgCtx, "error initializing terraform", "error", err)
		}

		// Update deployment status
		deployment.DeploymentStatus = entity.DeploymentInProgress
		helper.CalculateNewEpochTimeForDeployment(&deployment)
		if err := t.deploymentService.UpsertDeployment(bgCtx, deployment); err != nil {
			logging.LogError(bgCtx, "error updating deployment", "error", err)
		}

		// Apply
		if err := t.terraformService.Apply(bgCtx, lab); err != nil {
			notification.NotificationType = entity.Error
			notification.Message = string(entity.DeploymentFailed) + ". " + err.Error()
			notification.AutoClose = 5000
			deployment.DeploymentStatus = entity.DeploymentFailed
		} else {
			notification.NotificationType = entity.Success
			notification.Message = string(entity.DeploymentCompleted)
			deployment.DeploymentStatus = entity.DeploymentCompleted
		}

		// Send notification
		if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
			logging.LogError(bgCtx, "error setting server notification", "error", err)
		}

		// Update Deployment
		helper.CalculateNewEpochTimeForDeployment(&deployment)
		if err := t.deploymentService.UpsertDeployment(bgCtx, deployment); err != nil {
			logging.LogError(bgCtx, "error updating deployment", "error", err)
		}
	}()

	// Respond back to the request with the operation ID
	c.Status(http.StatusAccepted)
}

func (t *terraformHandler) Extend(c *gin.Context) {
	mode := c.Param("mode")

	deployment := entity.Deployment{}
	if err := c.Bind(&deployment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lab := deployment.DeploymentLab

	notification := entity.ServerNotification{
		Id:               uuid.New().String(),
		NotificationType: entity.Info,
		Message:          mode + " in progress",
		AutoClose:        2000,
	}

	if err := t.actionStatusService.SetServerNotification(c.Request.Context(), notification); err != nil {
		logging.LogError(c.Request.Context(), "error setting server notification", "error", err)
	}

	// background context for long running operation
	bgCtx := logging.CreateBackgroundContextWithValues(c.Request.Context())

	// Start the long-running operation in a goroutine
	go func() {
		// Ensure action always ends when goroutine exits
		defer func() {
			if err := t.actionStatusService.SetActionEnd(bgCtx); err != nil {
				logging.LogError(bgCtx, "error setting action end", "error", err)
			}
		}()

		if err := t.actionStatusService.SetActionStart(bgCtx); err != nil {
			logging.LogError(bgCtx, "error setting action start", "error", err)
			notification.NotificationType = entity.Error
			notification.Message = mode + " failed : Not able to update action status."
			return
		}
		if err := t.terraformService.Extend(bgCtx, lab, mode); err != nil {
			notification.NotificationType = entity.Error
			notification.AutoClose = 5000
			notification.Message = mode + " failed. " + err.Error()
		} else {
			notification.NotificationType = entity.Success
			notification.Message = mode + " completed."
		}
		if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
			logging.LogError(bgCtx, "error setting server notification", "error", err)
		}
	}()

	// Respond back to the request with the operation ID
	c.Status(http.StatusAccepted)
}

func (t *terraformHandler) Destroy(c *gin.Context) {
	deployment := entity.Deployment{}
	if err := c.Bind(&deployment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lab := deployment.DeploymentLab

	notification := entity.ServerNotification{
		Id:               uuid.New().String(),
		NotificationType: entity.Info,
		Message:          string(entity.DestroyInProgress),
		AutoClose:        2000,
	}

	// background context for long running operation
	bgCtx := logging.CreateBackgroundContextWithValues(c.Request.Context())

	// Start the long-running operation in a goroutine
	go func() {
		// Ensure action always ends when goroutine exits
		defer func() {
			if err := t.actionStatusService.SetActionEnd(bgCtx); err != nil {
				logging.LogError(bgCtx, "error setting action end", "error", err)
			}
		}()

		if err := t.actionStatusService.SetActionStart(bgCtx); err != nil {
			logging.LogError(bgCtx, "error setting action start", "error", err)
			notification.NotificationType = entity.Error
			notification.Message = "Failed to start destroy operation"
			if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
				logging.LogError(bgCtx, "error setting server notification", "error", err)
			}
			return
		}
		if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
			logging.LogError(bgCtx, "error setting server notification", "error", err)
		}

		// Ensure terraform is initialized
		if err := t.terraformService.EnsureInit(bgCtx); err != nil {
			logging.LogError(bgCtx, "error initializing terraform", "error", err)
			notification.NotificationType = entity.Error
			notification.Message = "error initializing terraform"
			if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
				logging.LogError(bgCtx, "error setting server notification", "error", err)
			}
			return
		}

		// Ensure workspace is as in deployment
		workspaceAtStart, err := t.workspaceService.GetSelectedWorkspace(bgCtx)
		reSelectWorkspace := false
		if err != nil {
			logging.LogError(bgCtx, "not able to get selected workspace", "error", err)
			notification.NotificationType = entity.Error
			notification.Message = "unable to get selected workspace"
			if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
				logging.LogError(bgCtx, "error setting server notification", "error", err)
			}

			return
		}

		if workspaceAtStart.Name != deployment.DeploymentWorkspace {
			logging.LogDebug(bgCtx, "selected workspace is not same as workspace in deployment to be destroyed selecting desired workspace", "selected_workspace", workspaceAtStart.Name, "desired_workspace", deployment.DeploymentWorkspace)

			if err := t.workspaceService.Select(bgCtx, entity.Workspace{Name: deployment.DeploymentWorkspace}); err != nil {
				logging.LogError(bgCtx, "not able to select workspace", "error", err)
				notification.NotificationType = entity.Error
				notification.Message = "error selecting workspace"
				if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
					logging.LogError(bgCtx, "error setting server notification", "error", err)
				}
				return
			}
			reSelectWorkspace = true
		}

		deployment.DeploymentStatus = entity.DestroyInProgress
		if err := t.deploymentService.UpsertDeployment(bgCtx, deployment); err != nil {
			logging.LogError(bgCtx, "error updating deployment", "error", err)
		}

		if err := t.terraformService.Destroy(bgCtx, lab); err != nil {
			notification.NotificationType = entity.Error
			notification.Message = string(entity.DestroyFailed)
			deployment.DeploymentStatus = entity.DestroyFailed
		} else {
			notification.NotificationType = entity.Success
			notification.Message = string(entity.DestroyCompleted)
			deployment.DeploymentStatus = entity.DestroyCompleted
		}

		if reSelectWorkspace {
			logging.LogDebug(bgCtx, "selecting previous workspace as the deployment is now complete", "previous workspace", workspaceAtStart.Name)
			if err := t.workspaceService.Select(bgCtx, workspaceAtStart); err != nil {
				logging.LogError(bgCtx, "not able to select workspace as it was at the start of destroy operation", "error", err)
				notification.NotificationType = entity.Error
				notification.Message = "error selecting workspace"
				if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
					logging.LogError(bgCtx, "error setting server notification", "error", err)
				}
			}
		}

		if err := t.actionStatusService.SetServerNotification(bgCtx, notification); err != nil {
			logging.LogError(bgCtx, "error setting server notification", "error", err)
		}
		if err := t.deploymentService.UpsertDeployment(bgCtx, deployment); err != nil {
			logging.LogError(bgCtx, "error updating deployment", "error", err)
		}
	}()

	// Respond back to the request with the operation ID
	c.Status(http.StatusAccepted)
}
