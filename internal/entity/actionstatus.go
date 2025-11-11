package entity

import "context"

type ActionStatus struct {
	InProgress bool `json:"inProgress"`
}

type TerraformOperation struct {
	OperationId string           `json:"operationId"`
	InProgress  bool             `json:"inProgress"`
	Status      DeploymentStatus `json:"status"`
}

type ServerNotificationType string

const (
	Info    ServerNotificationType = "info"
	Error   ServerNotificationType = "error"
	Success ServerNotificationType = "success"
	Default ServerNotificationType = "default"
	Warning ServerNotificationType = "warning"
)

type ServerNotification struct {
	Id               string                 `json:"id"`
	NotificationType ServerNotificationType `json:"type"`
	Message          string                 `json:"message"`
	AutoClose        int                    `json:"autoClose"` // 0 to never close
}

type ActionStatusService interface {
	GetActionStatus(ctx context.Context) (ActionStatus, error)
	SetActionStatus(ctx context.Context, status ActionStatus) error
	SetActionStart(ctx context.Context) error
	SetActionEnd(ctx context.Context) error
	WaitForActionStatusChange(ctx context.Context) (ActionStatus, error)

	SetTerraformOperation(ctx context.Context, op TerraformOperation) error
	GetTerraformOperation(ctx context.Context) (TerraformOperation, error)
	WaitForTerraformOperationChange(ctx context.Context) (TerraformOperation, error)

	SetServerNotification(ctx context.Context, notification ServerNotification) error
	GetServerNotification(ctx context.Context) (ServerNotification, error)
	WaitForServerNotificationChange(ctx context.Context) (ServerNotification, error)
}

type ActionStatusRepository interface {
	GetActionStatus(ctx context.Context) (string, error)
	SetActionStatus(ctx context.Context, val string) error
	WaitForActionStatusChange(ctx context.Context) (string, error)

	SetTerraformOperation(ctx context.Context, val string) error
	GetTerraformOperation(ctx context.Context) (string, error)
	WaitForTerraformOperationChange(ctx context.Context) (string, error)

	SetServerNotification(ctx context.Context, val string) error
	GetServerNotification(ctx context.Context) (string, error)
	WaitForServerNotificationChange(ctx context.Context) (string, error)
}
