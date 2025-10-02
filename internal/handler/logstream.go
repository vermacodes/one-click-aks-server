package handler

import (
	"net/http"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

type logStreamHandler struct {
	logStreamService entity.LogStreamService
}

func NewLogStreamHandler(r *gin.Engine, service entity.LogStreamService) {
	handler := &logStreamHandler{
		logStreamService: service,
	}

	r.GET("/logs", handler.GetLogs)
	r.PUT("/logs", handler.SetLogs)
	r.PUT("/logs/append", handler.AppendLogs)
	r.DELETE("/logs", handler.DeleteLogs)
	r.GET("/logsws", func(c *gin.Context) {
		handler.GetLogsWs(c.Writer, c.Request)
	})
}

func (l *logStreamHandler) GetLogs(c *gin.Context) {
	logStream, err := l.logStreamService.GetLogs(c.Request.Context())

	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.IndentedJSON(http.StatusOK, logStream)
}

func (l *logStreamHandler) AppendLogs(c *gin.Context) {
	var logs string
	if err := c.Bind(&logs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := l.logStreamService.AppendLogs(c.Request.Context(), logs)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Status(http.StatusOK)
}

func (l *logStreamHandler) DeleteLogs(c *gin.Context) {
	userID := c.Query("user")
	var err error

	if userID != "" {
		// Clear user-specific logs
		err = l.logStreamService.ClearLogs(c.Request.Context())
	} else {
		// Clear global logs (backward compatibility)
		err = l.logStreamService.ClearLogs(c.Request.Context())
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (l *logStreamHandler) SetLogs(c *gin.Context) {
	logStream := entity.LogStream{}
	if err := c.Bind(&logStream); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := l.logStreamService.SetLogs(c.Request.Context(), logStream.Logs)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

var logStreamUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// sendMessage sends a structured message via WebSocket
func (l *logStreamHandler) sendMessage(ws *websocket.Conn, msgType entity.WSMessageType, data interface{}) error {
	msg := entity.WSMessage{
		Type: msgType,
		Data: data,
	}
	return ws.WriteJSON(msg)
}

// sendErrorMessage sends an error message via WebSocket
func (l *logStreamHandler) sendErrorMessage(ws *websocket.Conn, errorMsg string) {
	errorData := entity.WSErrorMessage{
		Message: errorMsg,
	}
	l.sendMessage(ws, entity.WSMsgTypeError, errorData)
}

func (l *logStreamHandler) GetLogsWs(w http.ResponseWriter, r *http.Request) {
	ws, err := logStreamUpgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.LogError(r.Context(), "failed to upgrade log stream web socket connection", "error", err)
		return
	}
	defer ws.Close()

	logging.LogInfo(r.Context(), "webSocket connection established, waiting for authentication")

	// Wait for authentication message
	userID, err := auth.AuthenticateWebSocketConnection(ws)
	if err != nil {
		logging.LogError(r.Context(), "webSocket authentication failed", "error", err)
		l.sendErrorMessage(ws, "Authentication failed: "+err.Error())
		return
	}

	// now that we have user id, add it to context
	ctx := logging.WithUserID(r.Context(), userID)

	logging.LogInfo(ctx, "webSocket authenticated successfully", "userID", userID)

	// Send authentication success response
	authResp := entity.WSAuthResponse{
		Success: true,
		UserID:  userID,
	}
	if err := l.sendMessage(ws, entity.WSMsgTypeAuthResp, authResp); err != nil {
		logging.LogError(ctx, "failed to send auth response", "error", err)
		return
	}

	// Get initial logs for this specific user
	initialLogs, err := l.logStreamService.GetLogs(ctx)
	if err != nil {
		logging.LogError(ctx, "failed to retrieve initial logs for user", "error", err)
		l.sendErrorMessage(ws, "Failed to retrieve initial logs")
		return
	}

	// Send initial logs
	if err := l.sendMessage(ws, entity.WSMsgTypeLogs, initialLogs); err != nil {
		logging.LogError(ctx, "failed to write initial logs to websocket", "error", err)
		return
	}

	logging.LogInfo(ctx, "initial logs sent")

	// background context for long running operation
	bgCtx := logging.CreateBackgroundContextWithValues(ctx)

	// Start listening for log changes
	for {
		logStream, err := l.logStreamService.WaitForLogsChange(bgCtx)
		if err != nil {
			logging.LogError(bgCtx, "failed to wait for logs change for user", "error", err)
			l.sendErrorMessage(ws, "Failed to get log updates")
			return
		}

		if err := l.sendMessage(ws, entity.WSMsgTypeLogs, logStream); err != nil {
			logging.LogError(bgCtx, "failed to write logs to websocket for user", "error", err)
			return
		}
	}
}
