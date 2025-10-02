package middleware

import (
	"context"
	"crypto/rand"
	"fmt"

	"one-click-aks-server/internal/logging"

	"github.com/gin-gonic/gin"
)

// ContextMiddleware adds trace ID and other context data to requests
func ContextMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for existing X-Trace-ID header, otherwise generate new one
		traceID := c.GetHeader("X-Trace-ID")
		if traceID == "" {
			traceID = generateTraceID()
		}

		// Get or create context
		ctx := c.Request.Context()

		// Add trace ID to context
		ctx = context.WithValue(ctx, logging.TraceIDKey, traceID)

		// Update request with new context
		c.Request = c.Request.WithContext(ctx)

		// Set trace ID header for downstream services
		c.Header("X-Trace-ID", traceID)

		c.Next()
	}
}

// generateTraceID creates a random GUID-formatted trace ID
func generateTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple GUID if crypto/rand fails
		return "00000000-0000-0000-0000-000000000000"
	}

	// Set version (4) and variant bits according to RFC 4122
	bytes[6] = (bytes[6] & 0x0f) | 0x40 // Version 4
	bytes[8] = (bytes[8] & 0x3f) | 0x80 // Variant 10

	// Format as GUID: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		bytes[0:4],
		bytes[4:6],
		bytes[6:8],
		bytes[8:10],
		bytes[10:16])
}

// GetContextFromGin extracts context from Gin context
func GetContextFromGin(c *gin.Context) context.Context {
	return c.Request.Context()
}

// SetUserIDInGin sets user ID in Gin context
func SetUserIDInGin(c *gin.Context, userID string) {
	ctx := logging.WithUserID(c.Request.Context(), userID)
	c.Request = c.Request.WithContext(ctx)
}
