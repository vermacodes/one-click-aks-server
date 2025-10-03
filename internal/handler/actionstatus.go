package handler

import (
	"net/http"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type actionStatusHandler struct {
	actionStatusService entity.ActionStatusService
}

func NewActionStatusHandler(r *gin.Engine, service entity.ActionStatusService) {
	handler := &actionStatusHandler{
		actionStatusService: service,
	}

	r.GET("/terraform/statusws", func(c *gin.Context) {
		handler.GetTerraformOperationWs(c.Writer, c.Request)
	})
	r.GET("/actionstatusws", func(c *gin.Context) {
		handler.GetActionStatusWs(c.Writer, c.Request)
	})

	r.GET("/serverNotificationWs", func(c *gin.Context) {
		handler.GetServerNotificationWs(c.Writer, c.Request)
	})

	r.GET("/secureServerNotificationWs", func(c *gin.Context) {
		handler.GetServerNotificationWs(c.Writer, c.Request)
	})
}

func NewAuthActionStatusHandler(r *gin.RouterGroup, service entity.ActionStatusService) {
	handler := &actionStatusHandler{
		actionStatusService: service,
	}

	r.GET("/actionstatus", handler.GetActionStatus)
	r.PUT("/actionstatus", handler.SetActionStatus)
	r.GET("/terraform/status", handler.GetTerraformOperationStatus)
}

func (a *actionStatusHandler) GetActionStatus(c *gin.Context) {
	logging.LogInfo(c.Request.Context(), "getting action status")

	actionStatus, err := a.actionStatusService.GetActionStatus(c.Request.Context())
	if err != nil {
		logging.LogError(c.Request.Context(), "failed to get action status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, actionStatus)
}

func (a *actionStatusHandler) SetActionStatus(c *gin.Context) {
	logging.LogInfo(c.Request.Context(), "setting action status")

	actionStatus := entity.ActionStatus{}
	if err := c.Bind(&actionStatus); err != nil {
		logging.LogError(c.Request.Context(), "invalid request payload for action status", "error", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	a.actionStatusService.SetActionStatus(c.Request.Context(), actionStatus)
	c.Status(http.StatusOK)
}

func (a *actionStatusHandler) GetTerraformOperationStatus(c *gin.Context) {
	logging.LogInfo(c.Request.Context(), "getting terraform operation status")

	terraformOperation, err := a.actionStatusService.GetTerraformOperation(c.Request.Context())
	if err != nil {
		logging.LogError(c.Request.Context(), "failed to get terraform operation status", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, terraformOperation)
}

var actionStatusUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (a *actionStatusHandler) GetActionStatusWs(w http.ResponseWriter, r *http.Request) {
	conn, err := actionStatusUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.LogError(r.Context(), "failed to upgrade action status websocket connection", "error", err)
		return
	}

	defer conn.Close()

	logging.LogInfo(r.Context(), "websocket connection for action status established, waiting for authentication")

	// Wait for authentication message
	userID, err := auth.AuthenticateWebSocketConnection(conn)
	if err != nil {
		logging.LogError(r.Context(), "failed to authenticate action status websocket connection", "error", err)
		return
	}

	// now that we have user id, add it to context
	ctx := logging.WithUserID(r.Context(), userID)

	logging.LogInfo(ctx, "action status websocket authenticated successfully", "user_id", userID)

	// Get initial action status
	initialActionStatus, err := a.actionStatusService.GetActionStatus(ctx)
	if err != nil {
		logging.LogError(ctx, "failed to retrieve initial action status", "error", err)
		return
	}

	// Send the initial action status to the client
	if err := conn.WriteJSON(initialActionStatus); err != nil {
		logging.LogError(ctx, "failed to send initial action status to client", "error", err)
		return
	}

	for {
		// Get the current action status
		actionStatus, err := a.actionStatusService.WaitForActionStatusChange(ctx)
		if err != nil {
			logging.LogError(ctx, "failed to retrieve action status", "error", err)
			return
		}

		// Check for changes in action status
		if err := conn.WriteJSON(actionStatus); err != nil {
			logging.LogError(ctx, "failed to send action status to client", "error", err)
			return
		}
	}
}

func (a *actionStatusHandler) GetTerraformOperationWs(w http.ResponseWriter, r *http.Request) {
	conn, err := actionStatusUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.LogError(r.Context(), "failed to upgrade terraform operation websocket connection", "error", err)
		return
	}

	defer conn.Close()

	logging.LogInfo(r.Context(), "websocket connection for terraform operation established, waiting for authentication")

	// Wait for authentication message
	userID, err := auth.AuthenticateWebSocketConnection(conn)
	if err != nil {
		logging.LogError(r.Context(), "failed to authenticate terraform operation websocket connection", "error", err)
		return
	}

	// now that we have user id, add it to context
	ctx := logging.WithUserID(r.Context(), userID)

	logging.LogInfo(ctx, "terraform operation websocket authenticated successfully", "user_id", userID)

	// Get initial terraform operation status
	initialTerraformOperation, err := a.actionStatusService.GetTerraformOperation(ctx)
	if err != nil {
		logging.LogError(ctx, "failed to retrieve initial terraform operation status", "error", err)
		return
	}

	// Send the initial terraform operation status to the client
	if err := conn.WriteJSON(initialTerraformOperation); err != nil {
		logging.LogError(ctx, "failed to send initial terraform operation status to client", "error", err)
		return
	}

	for {
		// Get the current terraform operation status
		terraformOperation, err := a.actionStatusService.WaitForTerraformOperationChange(ctx)
		if err != nil {
			logging.LogError(ctx, "failed to retrieve terraform operation status", "error", err)
			return
		}

		// Check for changes in terraform operation status
		if err := conn.WriteJSON(terraformOperation); err != nil {
			logging.LogError(ctx, "failed to send terraform operation status to client", "error", err)
			return
		}
	}
}

func (a *actionStatusHandler) GetServerNotificationWs(w http.ResponseWriter, r *http.Request) {
	conn, err := actionStatusUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.LogError(r.Context(), "failed to upgrade server notification websocket connection", "error", err)
		return
	}

	defer conn.Close()

	logging.LogInfo(r.Context(), "websocket connection for server notification established, waiting for authentication")

	// Wait for authentication message
	userID, err := auth.AuthenticateWebSocketConnection(conn)
	if err != nil {
		logging.LogError(r.Context(), "failed to authenticate server notification websocket connection", "error", err)
		return
	}

	// now that we have user id, add it to context
	ctx := logging.WithUserID(r.Context(), userID)

	logging.LogInfo(ctx, "server notification websocket authenticated successfully", "user_id", userID)

	// Get initial server notification
	initialNotification, err := a.actionStatusService.GetServerNotification(ctx)
	if err != nil {
		logging.LogError(ctx, "failed to retrieve initial server notification", "error", err)
		return
	}

	// Send the initial server notification to the client
	if err := conn.WriteJSON(initialNotification); err != nil {
		logging.LogError(ctx, "failed to send initial server notification to client", "error", err)
		return
	}

	for {
		// Get the current server notification
		notification, err := a.actionStatusService.WaitForServerNotificationChange(ctx)
		if err != nil {
			logging.LogError(ctx, "failed to retrieve server notification", "error", err)
			return
		}

		// Check for changes in server notification
		if err := conn.WriteJSON(notification); err != nil {
			logging.LogError(ctx, "failed to send server notification to client", "error", err)
			return
		}
	}
}
