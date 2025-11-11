package handler

import (
	"net/http"

	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
)

type labHandler struct {
	labService entity.LabService
}

func NewLabHandler(r *gin.RouterGroup, labService entity.LabService) {
	handler := &labHandler{
		labService: labService,
	}
	r.GET("/lab", handler.GetLabFromRedis)
	r.PUT("/lab", handler.SetLabInRedis)
	r.DELETE("/lab/redis", handler.DeleteLabFromRedis)
	// r.POST("/lab", handler.AddMyLab)
	// r.DELETE("/lab", handler.DeleteMyLab)
	// r.GET("/lab/my", handler.GetMyLabs)
}

func (l *labHandler) GetLabFromRedis(c *gin.Context) {
	lab, err := l.labService.GetLabFromRedis(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, lab)
}

func (l *labHandler) SetLabInRedis(c *gin.Context) {
	lab := entity.LabType{}

	if err := c.Bind(&lab); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := l.labService.SetLabInRedis(c.Request.Context(), lab); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}

func (l *labHandler) DeleteLabFromRedis(c *gin.Context) {
	if err := l.labService.DeleteLabFromRedis(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}
