package middleware

import (
	"net/http"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"

	"github.com/gin-gonic/gin"
)

// TerraformMiddleware checks for already running operation and rejects new requests.
func TerraformActionMiddleware(actionStatusService entity.ActionStatusService) gin.HandlerFunc {
	return func(c *gin.Context) {
		actionStatus, err := actionStatusService.GetActionStatus(c.Request.Context())
		if err != nil {
			logging.LogError(c.Request.Context(), "not able to get current action status", "error", err)

			// Defaulting to no action
			actionStatus := entity.ActionStatus{
				InProgress: false,
			}
			if err := actionStatusService.SetActionStatus(c.Request.Context(), actionStatus); err != nil {
				logging.LogError(c.Request.Context(), "not able to set default action status", "error", err)
				c.AbortWithStatus(http.StatusInternalServerError)
				return
			}
		}

		if actionStatus.InProgress {
			logging.LogInfo(c.Request.Context(), "action in progress")
			c.AbortWithStatus(http.StatusConflict)
			return
		}

		if err := actionStatusService.SetActionStart(c.Request.Context()); err != nil {
			logging.LogError(c.Request.Context(), "not able to set action start", "error", err)
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		c.Next()
	}
}
