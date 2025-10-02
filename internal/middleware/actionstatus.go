package middleware

import (
	"net/http"

	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

// ActionStatusMiddleware checks for already running operation and rejects new requests.
func ActionStatusMiddleware(actionStatusService entity.ActionStatusService) gin.HandlerFunc {
	return func(c *gin.Context) {

		actionStatus, err := actionStatusService.GetActionStatus(c.Request.Context())
		if err != nil {
			slog.Error("not able to get current action status", err)

			// Defaulting to no action
			actionStatus := entity.ActionStatus{
				InProgress: false,
			}
			if err := actionStatusService.SetActionStatus(c.Request.Context(), actionStatus); err != nil {
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

		// set action status
		actionStatusService.SetActionStart(c.Request.Context())

		defer func() {
			// reset action status
			actionStatusService.SetActionEnd(c.Request.Context())
		}()

		c.Next()
	}
}
