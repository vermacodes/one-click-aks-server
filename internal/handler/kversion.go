package handler

import (
	"net/http"
	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
)

type kVersionHandler struct {
	kVersionService entity.KVersionService
}

func NewKVersionHandler(r *gin.RouterGroup, service entity.KVersionService) {
	handler := &kVersionHandler{
		kVersionService: service,
	}

	r.GET("/kubernetesorchestrators", handler.GetOrchestrator)
	r.GET("/kubernetesdefaultversion", handler.GetDefaultVersion)
}

func (k *kVersionHandler) GetOrchestrator(c *gin.Context) {
	kubernetesOrchestrator, err := k.kVersionService.GetOrchestrator(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, kubernetesOrchestrator)
}

// Default Kubernetes Version
func (k *kVersionHandler) GetDefaultVersion(c *gin.Context) {
	defaultVersion := k.kVersionService.GetDefaultVersion(c.Request.Context())
	if defaultVersion == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "not able to get default version"})
		return
	}

	c.IndentedJSON(http.StatusOK, defaultVersion)
}
