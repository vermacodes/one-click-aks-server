package handler

import (
	"net/http"
	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
)

type aroVersionHandler struct {
	aroVersionService entity.AROVersionService
}

func NewAROVersionHandler(router *gin.RouterGroup, aroVersionService entity.AROVersionService) {
	handler := &aroVersionHandler{
		aroVersionService: aroVersionService,
	}

	router.GET("/aroversion", handler.GetAROVersions)
	router.GET("/aroversion/default", handler.GetDefaultAROVersion)
}

func (a *aroVersionHandler) GetAROVersions(c *gin.Context) {
	aroVersions, err := a.aroVersionService.GetAROVersions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, aroVersions)
}

func (a *aroVersionHandler) GetDefaultAROVersion(c *gin.Context) {
	defaultVersion := a.aroVersionService.GetDefaultAROVersion()
	if defaultVersion == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "default version not found"})
		return
	}
	c.JSON(http.StatusOK, defaultVersion)
}
