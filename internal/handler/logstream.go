package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/exp/slog"
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
	userID := c.Query("user")
	var logStream entity.LogStream
	var err error

	if userID != "" {
		// Get user-specific logs
		logStream, err = l.logStreamService.GetLogsForUser(userID)
	} else {
		// Get global logs (backward compatibility)
		logStream, err = l.logStreamService.GetLogs()
	}

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

	userID := c.Query("user")
	var err error

	if userID != "" {
		// Append to user-specific logs
		err = l.logStreamService.AppendLogsForUser(userID, logs)
	} else {
		// Append to global logs (backward compatibility)
		err = l.logStreamService.AppendLogs(logs)
	}

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
		err = l.logStreamService.ClearLogsForUser(userID)
	} else {
		// Clear global logs (backward compatibility)
		err = l.logStreamService.ClearLogs()
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

	userID := c.Query("user")
	var err error

	if userID != "" {
		// Set user-specific logs
		err = l.logStreamService.SetLogsForUser(userID, logStream.Logs)
	} else {
		// Set global logs (backward compatibility)
		err = l.logStreamService.SetLogs(logStream.Logs)
	}

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

// authenticateWebSocketConnection handles WebSocket authentication via messages
func (l *logStreamHandler) authenticateWebSocketConnection(ws *websocket.Conn) (string, error) {
	// Set read timeout for authentication
	ws.SetReadDeadline(time.Now().Add(30 * time.Second))

	var authMsg entity.WSMessage
	if err := ws.ReadJSON(&authMsg); err != nil {
		return "", fmt.Errorf("failed to read auth message: %w", err)
	}

	if authMsg.Type != entity.WSMsgTypeAuth {
		return "", fmt.Errorf("expected auth message, got: %s", authMsg.Type)
	}

	// Extract auth data
	authDataBytes, err := json.Marshal(authMsg.Data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth data: %w", err)
	}

	var authData entity.WSAuthMessage
	if err := json.Unmarshal(authDataBytes, &authData); err != nil {
		return "", fmt.Errorf("failed to unmarshal auth data: %w", err)
	}

	if authData.Token == "" {
		return "", fmt.Errorf("no token provided")
	}

	// Remove Bearer prefix if present
	token := strings.TrimPrefix(authData.Token, "Bearer ")

	// Extract user principal from token
	userPrincipal, err := helper.GetUserPrincipalFromMSALAuthToken(token)
	if err != nil {
		return "", fmt.Errorf("failed to extract user principal: %w", err)
	}

	// Clear read timeout after successful authentication
	ws.SetReadDeadline(time.Time{})

	return userPrincipal, nil
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
		slog.Error("failed to upgrade log stream web socket connection", err)
		return
	}
	defer ws.Close()

	slog.Info("WebSocket connection established, waiting for authentication")

	// Wait for authentication message
	userID, err := l.authenticateWebSocketConnection(ws)
	if err != nil {
		slog.Error("WebSocket authentication failed", "error", err)
		l.sendErrorMessage(ws, "Authentication failed: "+err.Error())
		return
	}

	slog.Info("WebSocket authenticated successfully", "userID", userID)

	// Send authentication success response
	authResp := entity.WSAuthResponse{
		Success: true,
		UserID:  userID,
	}
	if err := l.sendMessage(ws, entity.WSMsgTypeAuthResp, authResp); err != nil {
		slog.Error("failed to send auth response", "userID", userID, "error", err)
		return
	}

	// Get initial logs for this specific user
	initialLogs, err := l.logStreamService.GetLogsForUser(userID)
	if err != nil {
		slog.Error("failed to retrieve initial logs for user", "userID", userID, "error", err)
		l.sendErrorMessage(ws, "Failed to retrieve initial logs")
		return
	}

	// Send initial logs
	if err := l.sendMessage(ws, entity.WSMsgTypeLogs, initialLogs); err != nil {
		slog.Error("failed to write initial logs to websocket", "userID", userID, "error", err)
		return
	}

	slog.Info("Initial Logs Sent",
		slog.String("logs", initialLogs.Logs),
	)

	// Start listening for log changes
	for {
		logStream, err := l.logStreamService.WaitForLogsChangeForUser(userID)
		if err != nil {
			slog.Error("failed to wait for logs change for user", "userID", userID, "error", err)
			l.sendErrorMessage(ws, "Failed to get log updates")
			return
		}

		if err := l.sendMessage(ws, entity.WSMsgTypeLogs, logStream); err != nil {
			slog.Error("failed to write logs to websocket for user", "userID", userID, "error", err)
			return
		}
	}
}
