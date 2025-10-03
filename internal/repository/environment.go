package repository

import (
	"context"
	"os"

	"one-click-aks-server/internal/logging"
)

func setEnvironmentVariable(key string, value string) {
	err := os.Setenv(key, value)
	if err != nil {
		logging.LogError(context.Background(), "not able to set environment variable", "error", err)
	}
}
