package repository

import (
	"context"
	"os/exec"

	"one-click-aks-server/internal/entity"

	"github.com/redis/go-redis/v9"
)

type preferenceRepository struct{}

func NewPreferenceRepository() entity.PreferenceRepository {
	return &preferenceRepository{}
}

var preferenceCtx = context.Background()

func newPreferenceRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
}

func (p *preferenceRepository) GetPreferenceFromBlob(storageAccountName string) (string, error) {
	out, err := exec.Command("bash", "-c", "az storage blob download -c tfstate -n preference.json --account-name "+storageAccountName+" --file /tmp/preference > /dev/null 2>&1 && cat /tmp/preference && rm /tmp/preference").Output()
	return string(out), err
}

func (p *preferenceRepository) PutPreferenceInBlob(val string, storageAccountName string) error {
	_, err := exec.Command("bash", "-c", "echo '"+val+"' | az storage blob upload --data @- -c tfstate -n preference.json --account-name "+storageAccountName+" --overwrite").Output()
	return err
}

func (p *preferenceRepository) GetPreferenceFromRedis() (string, error) {
	rdb := newPreferenceRedisClient()
	return rdb.Get(preferenceCtx, "preference").Result()
}

func (p *preferenceRepository) PutPreferenceInRedis(val string) error {
	rdb := newPreferenceRedisClient()
	return rdb.Set(preferenceCtx, "preference", val, 0).Err()
}
