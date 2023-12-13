package handler

import (
	"net/http"

	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
)

type RedisHandler struct {
	RedisService entity.RedisService
}

func NewRedisHandler(r *gin.RouterGroup, redisService entity.RedisService) {
	handler := &RedisHandler{
		RedisService: redisService,
	}

	r.DELETE("/cache", handler.DeleteServerCache)
}

func (r *RedisHandler) DeleteServerCache(c *gin.Context) {
	if err := r.RedisService.ResetServerCache(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
