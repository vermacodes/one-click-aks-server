package middleware

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"
	"one-click-aks-server/internal/mise"

	"github.com/gin-gonic/gin"
)

func AuthRequired(miseServer mise.Server, config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the auth token from the request header
		authToken := c.GetHeader("Authorization")

		if authToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no auth token provided"})
			return
		}

		var userPrincipal string

		if config.AuthVerifyMode == "MISE" {
			// MISE Implementation
			result, err := miseServer.DelegateAuthToContainer(authToken, c.Request.URL.String(), c.Request.Method, c.ClientIP())
			if err != nil {
				var validationErr *mise.ErrTokenValidation
				if errors.As(err, &validationErr) {
					// can access validationErr.ErrorDescription, validationErr.WWWAuthenticate, validationErr.StatusCode
					logging.LogError(c.Request.Context(), "token validation error", "error", validationErr)
				} else {
					logging.LogError(c.Request.Context(), "error while delegating auth to container", "error", err)
				}

				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication failed"})
				return
			}

			userPrincipals, ok := result.SubjectClaims["preferred_username"]
			if !ok || len(userPrincipals) == 0 {
				logging.LogError(c.Request.Context(), "preferred_username claim not found in subject claims")
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "preferred_username claim not found in token"})
				return
			}

			userPrincipal = userPrincipals[0] // take the first one
		}
		if config.AuthVerifyMode == "Custom" {

			isAADToken, err := helper.VerifyToken(authToken)
			if err != nil || !isAADToken {
				logging.LogError(c.Request.Context(), "invalid auth token", "error", err)
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth token" + err.Error()})
				return
			}

			// Remove Bearer from the authToken
			authToken = strings.Split(authToken, "Bearer ")[1]

			if authToken == "" {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no auth token provided"})
				return
			}

			userPrincipal, err = helper.GetUserPrincipalFromMSALAuthToken(authToken)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid auth token"})
				return
			}
		}
		// Set user ID in context for tracing and user-specific operations
		SetUserIDInGin(c, userPrincipal)
		ctx := GetContextFromGin(c)

		// Log authentication success with context (includes trace_id and user_id)
		logging.LogDebug(ctx, "user authenticated successfully",
			"user", userPrincipal,
			"user_principal", userPrincipal)

		os.Setenv("ACTLABS_AUTH_TOKEN", authToken) // used by repositories to authenticate with other services

		c.Next()
	}
}

func APIKeyAuthRequired(config config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the api key from the request header
		reqApiKey := c.GetHeader("x-api-key")

		if reqApiKey == "" || reqApiKey != config.APIKey {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid api key"})
			return
		}

		reqUserPrincipal := c.GetHeader("x-user-id")
		if reqUserPrincipal == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "no user principal provided"})
			return
		}

		SetUserIDInGin(c, reqUserPrincipal)
		ctx := GetContextFromGin(c)

		logging.LogDebug(ctx, "api call authenticated successfully",
			"user", reqUserPrincipal)

		c.Next()
	}
}
