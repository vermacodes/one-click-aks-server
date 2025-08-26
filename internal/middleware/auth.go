package middleware

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/mise"

	"github.com/gin-gonic/gin"
	"golang.org/x/exp/slog"
)

func AuthRequired(miseServer mise.Server, authService entity.AuthService, logStream entity.LogStreamService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")

		if authToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no auth token provided"})
			return
		}

		// MISE Implementation
		result, err := miseServer.DelegateAuthToContainer(authToken, c.Request.URL.String(), c.Request.Method, c.ClientIP())
		if err != nil {
			var validationErr *mise.ErrTokenValidation
			if errors.As(err, &validationErr) {
				// can access validationErr.ErrorDescription, validationErr.WWWAuthenticate, validationErr.StatusCode
				slog.Error("token validation error", validationErr)
			} else {
				slog.Error("error while delegating auth to container", err)
			}

			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
			return
		}

		userName, ok := result.SubjectClaims["Preferred_username"]
		if !ok || userName == "" {
			slog.Error("Preferred_username claim not found in subject claims", nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Preferred_username claim not found in token"})
			return
		}
		slog.Info("authenticated user", "user", userName)

		// Keeping the custom auth validation in place, just in case MISE isn't working as expected.
		isAADToken, err := helper.VerifyToken(authToken)
		if err != nil || !isAADToken {
			slog.Error("invalid auth token", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth token" + err.Error()})
			return
		}

		// Remove Bearer from the authToken
		authToken = strings.Split(authToken, "Bearer ")[1]

		if authToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no auth token provided"})
			return
		}

		userPrincipal, err := helper.GetUserPrincipalFromMSALAuthToken(authToken)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth token"})
			return
		}

		// ensure user principal matches with the one in env
		if userPrincipal != os.Getenv("ARM_USER_PRINCIPAL_NAME") {
			slog.Error("principal mismatch : token issued to "+userPrincipal+" but found user "+os.Getenv("ARM_USER_PRINCIPAL_NAME"), nil)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "principal mismatch : token issued to " + userPrincipal + " but found user " + os.Getenv("ARM_USER_PRINCIPAL_NAME")})
			return
		}

		// loginStatus, err := authService.ServicePrincipalLoginStatus()
		// if err != nil {
		// 	slog.Error("not able to get auth status", err)
		// 	c.AbortWithStatus(http.StatusUnauthorized)
		// 	return
		// }

		// if !loginStatus.IsLoggedIn {
		// 	slog.Info("authentication required")
		// 	c.AbortWithStatus(http.StatusUnauthorized)
		// 	return
		// }

		os.Setenv("ACTLABS_AUTH_TOKEN", authToken) // used by repositories to authenticate with other services

		c.Next()
	}
}
