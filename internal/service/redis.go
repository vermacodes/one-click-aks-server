package service

import (
	"context"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"
)

type redisService struct {
	redisRepository entity.RedisRepository
}

func NewRedisService(redisRepository entity.RedisRepository) entity.RedisService {
	return &redisService{
		redisRepository: redisRepository,
	}
}

func (r *redisService) ResetServerCache(ctx context.Context) error {
	logging.LogInfo(ctx, "resetting server cache")
	if err := r.redisRepository.ResetServerCache(ctx); err != nil {
		logging.LogError(ctx, "not able to reset server cache", "error", err)
		return err
	}

	logging.LogDebug(ctx, "server cache reset complete")
	return nil
}
