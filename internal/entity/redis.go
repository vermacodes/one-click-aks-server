package entity

import "context"

type RedisService interface {
	ResetServerCache(ctx context.Context) error
}

type RedisRepository interface {
	ResetServerCache(ctx context.Context) error
}
