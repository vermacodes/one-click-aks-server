package service

import (
	"context"
	"encoding/json"
	"time"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
)

type actionStatusService struct {
	actionStatusRepository entity.ActionStatusRepository
}

func NewActionStatusService(actionStatusRepository entity.ActionStatusRepository) entity.ActionStatusService {
	return &actionStatusService{
		actionStatusRepository: actionStatusRepository,
	}
}

func (a *actionStatusService) GetActionStatus(ctx context.Context) (entity.ActionStatus, error) {
	logging.LogDebug(ctx, "getting action status")
	actionStatus := entity.ActionStatus{}
	val, err := a.actionStatusRepository.GetActionStatus(ctx)
	if err != nil {
		logging.LogDebug(ctx, "action status not found in redis")

		// Reset to default.
		defaultActionStatus := entity.ActionStatus{
			InProgress: false,
		}

		if err := a.SetActionStatus(ctx, defaultActionStatus); err != nil {
			logging.LogError(ctx, "not able to set default action status in redis.", "error", err)
		}

		return defaultActionStatus, nil
	}

	if err = json.Unmarshal([]byte(val), &actionStatus); err != nil {
		logging.LogError(ctx, "not able to translate action status string to object", "error", err)
		return actionStatus, err
	}

	return actionStatus, nil
}

func (a *actionStatusService) SetActionStatus(ctx context.Context, actionStatus entity.ActionStatus) error {
	logging.LogDebug(ctx, "setting action status")
	val, err := json.Marshal(actionStatus)
	if err != nil {
		logging.LogError(ctx, "not able to marshal object to string", "error", err)
		return err
	}

	// Set action status with conditional TTL
	if err = a.actionStatusRepository.SetActionStatus(ctx, string(val)); err != nil {
		logging.LogError(ctx, "not able to set actions status in redis", "error", err)
		return err
	}

	// Only start auto-renewal if action is in progress
	if actionStatus.InProgress {
		logging.LogDebug(ctx, "action in progress, starting auto-renewal with 30s TTL")

		// Start auto-renewal goroutine that renews every 20 seconds
		go func() {
			ticker := time.NewTicker(20 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					// Context cancelled, stop renewal
					logging.LogInfo(ctx, "context cancelled, stopping action status auto-renewal")
					return
				case <-ticker.C:
					// Check current status before renewing
					currentStatus, err := a.GetActionStatus(ctx)
					if err != nil {
						logging.LogError(ctx, "failed to get current action status for renewal", "error", err)
						continue // Continue trying on next tick
					}

					if !currentStatus.InProgress {
						// Action is no longer in progress, stop renewal
						logging.LogInfo(ctx, "action no longer in progress, stopping auto-renewal")
						return
					}

					// Only renew if still in progress - use current status
					currentVal, err := json.Marshal(currentStatus)
					if err != nil {
						logging.LogError(ctx, "failed to marshal current status for renewal", "error", err)
						continue
					}

					if err := a.actionStatusRepository.SetActionStatus(ctx, string(currentVal)); err != nil {
						logging.LogError(ctx, "failed to renew action status TTL", "error", err)
						// Continue trying to renew on next tick
					} else {
						logging.LogDebug(ctx, "action status TTL renewed successfully")
					}
				}
			}
		}()
	} else {
		logging.LogDebug(ctx, "action not in progress, set with no expiration")
	}

	return nil
}

func (a *actionStatusService) SetActionStart(ctx context.Context) error {
	logging.LogInfo(ctx, "setting action status start")
	actionStatus, err := a.GetActionStatus(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get current action status", "error", err)
		return err
	}

	if !actionStatus.InProgress {
		actionStatus.InProgress = true
	}

	if err := a.SetActionStatus(ctx, actionStatus); err != nil {
		logging.LogError(ctx, "not able to set action status to start", "error", err)
	}

	return nil
}

func (a *actionStatusService) SetActionEnd(ctx context.Context) error {
	logging.LogInfo(ctx, "setting action status end")
	actionStatus, err := a.GetActionStatus(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get current action status", "error", err)
		return err
	}

	if actionStatus.InProgress {
		actionStatus.InProgress = false
	}

	if err := a.SetActionStatus(ctx, actionStatus); err != nil {
		logging.LogError(ctx, "not able to set action status to end", "error", err)
	}

	return nil
}

func (a *actionStatusService) WaitForActionStatusChange(ctx context.Context) (entity.ActionStatus, error) {
	logging.LogInfo(ctx, "waiting for action status change")
	actionStatus := entity.ActionStatus{}
	val, err := a.actionStatusRepository.WaitForActionStatusChange(ctx)
	if err != nil {
		logging.LogError(ctx, "action status not found in redis", "error", err)
		return actionStatus, err
	}

	if err = json.Unmarshal([]byte(val), &actionStatus); err != nil {
		logging.LogError(ctx, "not able to translate action status string to object", "error", err)
		return actionStatus, err
	}

	return actionStatus, nil
}

func (a *actionStatusService) SetTerraformOperation(ctx context.Context, terraformOperation entity.TerraformOperation) error {
	logging.LogInfo(ctx, "setting terraform operation")
	val, err := json.Marshal(terraformOperation)
	if err != nil {
		logging.LogError(ctx, "not able to marshal object to string", "error", err)
		return err
	}

	if err = a.actionStatusRepository.SetTerraformOperation(ctx, string(val)); err != nil {
		logging.LogError(ctx, "not able to set terraform operation in redis", "error", err)
	}

	return nil
}

func (a *actionStatusService) GetTerraformOperation(ctx context.Context) (entity.TerraformOperation, error) {
	logging.LogInfo(ctx, "getting terraform operation")
	terraformOperation := entity.TerraformOperation{}
	val, err := a.actionStatusRepository.GetTerraformOperation(ctx)
	if err != nil {
		logging.LogError(ctx, "terraform operation not found in redis", "error", err)
		return terraformOperation, err
	}

	if err = json.Unmarshal([]byte(val), &terraformOperation); err != nil {
		logging.LogError(ctx, "not able to translate terraform operation string to object", "error", err)
		return terraformOperation, err
	}

	return terraformOperation, nil
}

func (a *actionStatusService) WaitForTerraformOperationChange(ctx context.Context) (entity.TerraformOperation, error) {
	logging.LogInfo(ctx, "waiting for terraform operation change")
	terraformOperation := entity.TerraformOperation{}
	val, err := a.actionStatusRepository.WaitForTerraformOperationChange(ctx)
	if err != nil {
		logging.LogError(ctx, "action status not found in redis", "error", err)
		return terraformOperation, err
	}

	if err = json.Unmarshal([]byte(val), &terraformOperation); err != nil {
		logging.LogError(ctx, "not able to translate action status string to object", "error", err)
		return terraformOperation, err
	}

	return terraformOperation, nil
}

func (a *actionStatusService) SetServerNotification(ctx context.Context, serverNotification entity.ServerNotification) error {
	logging.LogInfo(ctx, "setting server notification")
	val, err := json.Marshal(serverNotification)
	if err != nil {
		logging.LogError(ctx, "not able to marshal object to string", "error", err)
		return err
	}

	if err = a.actionStatusRepository.SetServerNotification(ctx, string(val)); err != nil {
		logging.LogError(ctx, "not able to set server notification in redis", "error", err)
	}

	return nil
}

func (a *actionStatusService) GetServerNotification(ctx context.Context) (entity.ServerNotification, error) {
	logging.LogInfo(ctx, "getting server notification")
	serverNotification := entity.ServerNotification{}
	val, err := a.actionStatusRepository.GetServerNotification(ctx)
	if err != nil {
		logging.LogError(ctx, "server notification not found in redis", "error", err)
		return serverNotification, err
	}

	if err = json.Unmarshal([]byte(val), &serverNotification); err != nil {
		logging.LogError(ctx, "not able to translate server notification string to object", "error", err)
		return serverNotification, err
	}

	return serverNotification, nil
}

func (a *actionStatusService) WaitForServerNotificationChange(ctx context.Context) (entity.ServerNotification, error) {
	logging.LogInfo(ctx, "waiting for server notification change")
	serverNotification := entity.ServerNotification{}
	val, err := a.actionStatusRepository.WaitForServerNotificationChange(ctx)
	if err != nil {
		logging.LogError(ctx, "server notification not found in redis", "error", err)
		return serverNotification, err
	}

	if err = json.Unmarshal([]byte(val), &serverNotification); err != nil {
		logging.LogError(ctx, "not able to translate server notification string to object", "error", err)
		return serverNotification, err
	}

	return serverNotification, nil
}
