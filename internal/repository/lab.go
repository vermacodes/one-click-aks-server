package repository

import (
	"context"
	"io"
	"net/http"
	"os"
	"os/exec"

	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"

	"github.com/redis/go-redis/v9"
)

type labRepository struct {
	appConfig *config.Config
}

func NewLabRepository(appConfig *config.Config) entity.LabRepository {
	return &labRepository{
		appConfig: appConfig,
	}
}

var labCtx = context.Background()

func newLabRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func (l *labRepository) GetLabFromRedis() (string, error) {
	rdb := newLabRedisClient()
	return rdb.Get(labCtx, "lab").Result()
}

func (l *labRepository) SetLabInRedis(val string) error {
	rdb := newLabRedisClient()
	return rdb.Set(labCtx, "lab", val, 0).Err()
}

func (l *labRepository) DeleteLabFromRedis() error {
	rdb := newLabRedisClient()
	return rdb.Del(labCtx, "lab").Err()
}

func (l *labRepository) GetProtectedLab(typeOfLab string, labId string) (string, error) {
	actlabsAuthEndpoint := os.Getenv("ACTLABS_AUTH_URL")
	if actlabsAuthEndpoint == "" {
		actlabsAuthEndpoint = "https://actlabs-auth.azurewebsites.net/"
	}
	// http call to actlabs-auth
	req, err := http.NewRequest("GET", actlabsAuthEndpoint+"lab/protected/"+typeOfLab+"/"+labId, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("ACTLABS_AUTH_TOKEN"))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("ProtectedLabSecret", l.appConfig.ProtectedLabSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func (l *labRepository) GetExtendScriptTemplate() (string, error) {
	out, err := exec.Command("bash", "-c", "cat ${ROOT_DIR}/scripts/template.sh | base64").Output()
	return string(out), err
}
