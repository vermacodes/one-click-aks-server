package service

import (
	"encoding/base64"

	"one-click-aks-server/internal/entity"

	"golang.org/x/exp/slog"
)

type logStreamService struct {
	logStreamRepository entity.LogStreamRepository
}

func NewLogStreamService(logStreamRepository entity.LogStreamRepository) entity.LogStreamService {
	return &logStreamService{
		logStreamRepository: logStreamRepository,
	}
}

// Appends to already set logs in redis.
func (l *logStreamService) AppendLogs(logs string) error {
	logStream, err := l.GetLogs()
	if err != nil {
		slog.Debug("not able to get logs from redis. setting new")
		logStream.Logs = ""
	}

	logStream.Logs += logs

	return l.SetLogs(logStream.Logs)
}

func (l *logStreamService) ClearLogs() error {
	return l.SetLogs("")
}

// sets the logs in redis.
func (l *logStreamService) SetLogs(logs string) error {
	// this is a hack to continue the logs from where they are right now.
	if logs == "continue" {
		prevLogStream, err := l.GetLogs()
		if err != nil {
			logs = ""
		} else {
			logs = prevLogStream.Logs
		}
	}

	// encode the logs string and store it in redis.
	encodedLogs := base64.StdEncoding.EncodeToString([]byte(logs))
	return l.logStreamRepository.SetLogsInRedis(encodedLogs)
}

// gets the logs from redis and returns the object.
func (l *logStreamService) GetLogs() (entity.LogStream, error) {
	encodedLogs, err := l.logStreamRepository.GetLogsFromRedis()
	if err != nil {
		slog.Info("not able to get logs from redis")

		// Default to empty.
		defaultLogStream := entity.LogStream{
			Logs: "",
		}

		if err := l.SetLogs(""); err != nil {
			slog.Error("not able to set default log stream", err)
			return defaultLogStream, err
		}

		return defaultLogStream, nil
	}

	// decode the logs string and return the object.
	return helperEncodedStringToLogStreamObject(encodedLogs)
}

// waits for the logs to change and returns the new logs.
func (l *logStreamService) WaitForLogsChange() (entity.LogStream, error) {
	logsString, err := l.logStreamRepository.WaitForLogsChange()
	if err != nil {
		return entity.LogStream{}, err
	}

	return helperEncodedStringToLogStreamObject(logsString)
}

// User-specific methods

// AppendLogsForUser appends logs for a specific user
func (l *logStreamService) AppendLogsForUser(userID, logs string) error {
	logStream, err := l.GetLogsForUser(userID)
	if err != nil {
		slog.Debug("not able to get logs from redis for user. setting new", "userID", userID)
		logStream.Logs = ""
	}

	logStream.Logs += logs

	return l.SetLogsForUser(userID, logStream.Logs)
}

// SetLogsForUser sets the logs for a specific user in redis
func (l *logStreamService) SetLogsForUser(userID, logs string) error {
	// this is a hack to continue the logs from where they are right now.
	if logs == "continue" {
		prevLogStream, err := l.GetLogsForUser(userID)
		if err != nil {
			logs = ""
		} else {
			logs = prevLogStream.Logs
		}
	}

	// encode the logs string and store it in redis.
	encodedLogs := base64.StdEncoding.EncodeToString([]byte(logs))
	return l.logStreamRepository.SetLogsInRedisForUser(userID, encodedLogs)
}

// GetLogsForUser gets the logs for a specific user from redis and returns the object
func (l *logStreamService) GetLogsForUser(userID string) (entity.LogStream, error) {
	encodedLogs, err := l.logStreamRepository.GetLogsFromRedisForUser(userID)
	if err != nil {
		slog.Info("not able to get logs from redis for user", "userID", userID)

		// Default to empty.
		defaultLogStream := entity.LogStream{
			Logs: "",
		}

		if err := l.SetLogsForUser(userID, ""); err != nil {
			slog.Error("not able to set default log stream for user", "userID", userID, "error", err)
			return defaultLogStream, err
		}

		return defaultLogStream, nil
	}

	// decode the logs string and return the object.
	return helperEncodedStringToLogStreamObject(encodedLogs)
}

// ClearLogsForUser clears the logs for a specific user
func (l *logStreamService) ClearLogsForUser(userID string) error {
	return l.SetLogsForUser(userID, "")
}

// WaitForLogsChangeForUser waits for the logs to change for a specific user and returns the new logs
func (l *logStreamService) WaitForLogsChangeForUser(userID string) (entity.LogStream, error) {
	logsString, err := l.logStreamRepository.WaitForLogsChangeForUser(userID)
	if err != nil {
		return entity.LogStream{}, err
	}

	return helperEncodedStringToLogStreamObject(logsString)
}

// decodes the encoded string and returns the object.
func helperEncodedStringToLogStreamObject(encodedLogs string) (entity.LogStream, error) {
	logBytes, err := base64.StdEncoding.DecodeString(encodedLogs)
	if err != nil {
		slog.Error("not able to decode logs", err)
		return entity.LogStream{}, err
	}

	logs := string(logBytes)
	logStream := entity.LogStream{
		Logs: logs,
	}

	return logStream, nil
}
