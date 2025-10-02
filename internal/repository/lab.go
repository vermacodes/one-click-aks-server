package repository

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os/exec"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"

	"github.com/redis/go-redis/v9"
)

type labRepository struct {
	appConfig *config.Config
	auth      *auth.Auth
}

func NewLabRepository(appConfig *config.Config, auth *auth.Auth) entity.LabRepository {
	return &labRepository{
		appConfig: appConfig,
		auth:      auth,
	}
}

func newLabRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func (l *labRepository) GetLabFromRedis(ctx context.Context) (string, error) {
	rdb := newLabRedisClient()
	return rdb.Get(ctx, "lab").Result()
}

func (l *labRepository) SetLabInRedis(ctx context.Context, lab string) error {
	rdb := newLabRedisClient()
	return rdb.Set(ctx, "lab", lab, 0).Err()
}

func (l *labRepository) DeleteLabFromRedis(ctx context.Context) error {
	rdb := newLabRedisClient()
	return rdb.Del(ctx, "lab").Err()
}

func (l *labRepository) GetProtectedLab(ctx context.Context, typeOfLab string, labId string) (string, error) {
	actlabsAuthEndpoint := l.appConfig.ActlabsHubURL
	// http call to actlabs-auth
	req, err := http.NewRequest("GET", actlabsAuthEndpoint+"lab/protected/"+typeOfLab+"/"+labId, nil)
	if err != nil {
		return "", err
	}

	armAccessToken, err := l.auth.GetARMAccessToken()
	if err != nil {
		logging.LogError(ctx, "error getting arm access token",
			slog.Any("error", err),
		)
		return "", err
	}

	req.Header.Set("Authorization", "Bearer "+armAccessToken)
	req.Header.Set("x-ms-client-principal-name", l.appConfig.ArmUserPrincipalName)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("ProtectedLabSecret", entity.ProtectedLabSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logging.LogError(ctx, "not able to make http call to get protected lab",
			slog.Any("error", err),
		)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.LogError(ctx, "not able to make http call successfully",
			slog.Any("http_code", resp.StatusCode),
			slog.Any("error", err),
		)
		return "", fmt.Errorf("not able to get protected lab, received error code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logging.LogError(ctx, "not able to read response body",
			slog.Any("error", err),
		)
		return "", err
	}

	return string(body), nil
}

func (l *labRepository) GetExtendScriptTemplate(ctx context.Context) (string, error) {
	out, err := exec.Command("bash", "-c", "cat ${ROOT_DIR}/scripts/template.sh | base64").Output()
	return string(out), err
}
