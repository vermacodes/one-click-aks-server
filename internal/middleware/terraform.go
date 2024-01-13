package middleware

import (
	"net/http"

	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// TerraformMiddleware checks for already running operation and rejects new requests.
func TerraformActionMiddleware(actionStatusService entity.ActionStatusService) gin.HandlerFunc {
	return func(c *gin.Context) {
		actionStatus, err := actionStatusService.GetActionStatus()
		if err != nil {
			slog.Error("not able to get current action status", err)

			// Defaulting to no action
			actionStatus := entity.ActionStatus{
				InProgress: false,
			}
			if err := actionStatusService.SetActionStatus(actionStatus); err != nil {
				slog.Error("not able to set default action status.", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}

		if actionStatus.InProgress {
			slog.Info("action in progress")
			c.AbortWithStatus(http.StatusConflict)
			return
		}

		if err := actionStatusService.SetActionStart(); err != nil {
			slog.Error("not able to set action start", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Next()
	}
}
