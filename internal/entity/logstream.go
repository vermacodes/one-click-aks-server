package entity

import "context"

type LogStream struct {
	Logs string `json:"logs"`
}

// WebSocket message types
type WSMessageType string

const (
	WSMsgTypeAuth     WSMessageType = "auth"
	WSMsgTypeAuthResp WSMessageType = "auth_response"
	WSMsgTypeLogs     WSMessageType = "logs"
	WSMsgTypeError    WSMessageType = "error"
)

// WebSocket message structure
type WSMessage struct {
	Type WSMessageType `json:"type"`
	Data interface{}   `json:"data"`
}

// Authentication message payload
type WSAuthMessage struct {
	Token string `json:"token"`
}

// Authentication response message
type WSAuthResponse struct {
	Success bool   `json:"success"`
	UserID  string `json:"user_id,omitempty"`
	Error   string `json:"error,omitempty"`
}

// Error message payload
type WSErrorMessage struct {
	Message string `json:"message"`
}

type LogStreamService interface {
	AppendLogs(ctx context.Context, logs string) error
	SetLogs(ctx context.Context, logs string) error
	GetLogs(ctx context.Context) (LogStream, error)
	ClearLogs(ctx context.Context) error
	WaitForLogsChange(ctx context.Context) (LogStream, error)

	// User-specific methods
	// AppendLogsForUser(ctx context.Context, userID, logs string) error
	// SetLogsForUser(ctx context.Context, userID, logs string) error
	// GetLogsForUser(ctx context.Context, userID string) (LogStream, error)
	// ClearLogsForUser(ctx context.Context, userID string) error
	// WaitForLogsChangeForUser(ctx context.Context, userID string) (LogStream, error)
}

type LogStreamRepository interface {
	SetLogsInRedis(ctx context.Context, logStream string) error
	GetLogsFromRedis(ctx context.Context) (string, error)
	WaitForLogsChange(ctx context.Context) (string, error)

	// User-specific methods
	// SetLogsInRedisForUser(ctx context.Context, userID, logStream string) error
	// GetLogsFromRedisForUser(ctx context.Context, userID string) (string, error)
	// WaitForLogsChangeForUser(ctx context.Context, userID string) (string, error)
}
