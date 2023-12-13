package logger

import (
	"os"
	"strconv"

	"golang.org/x/exp/slog"
)

func SetupLogger() {
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		slog.Error("LOG_LEVEL not set")
	}

	logLevelInt, err := strconv.Atoi(logLevel)
	if err != nil {
		slog.Error("Error converting LOG_LEVEL to int", err)
		logLevelInt = 0
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.Level(logLevelInt),
	}

	slogHandler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(slogHandler))
}
