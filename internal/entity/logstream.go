package entity

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
	AppendLogs(logs string) error
	SetLogs(logs string) error
	GetLogs() (LogStream, error)
	ClearLogs() error
	WaitForLogsChange() (LogStream, error)

	// User-specific methods
	AppendLogsForUser(userID, logs string) error
	SetLogsForUser(userID, logs string) error
	GetLogsForUser(userID string) (LogStream, error)
	ClearLogsForUser(userID string) error
	WaitForLogsChangeForUser(userID string) (LogStream, error)
}

type LogStreamRepository interface {
	SetLogsInRedis(logStream string) error
	GetLogsFromRedis() (string, error)
	WaitForLogsChange() (string, error)

	// User-specific methods
	SetLogsInRedisForUser(userID, logStream string) error
	GetLogsFromRedisForUser(userID string) (string, error)
	WaitForLogsChangeForUser(userID string) (string, error)
}
