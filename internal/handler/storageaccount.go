package handler

import (
	"net/http"

	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
)

type StorageAccountHandler struct {
	storageAccountService entity.StorageAccountService
}

func NewStorageAccountWithActionStatusHandler(r *gin.RouterGroup, service entity.StorageAccountService) {
	handler := &StorageAccountHandler{
		storageAccountService: service,
	}
	r.PUT("/storageaccount/breakbloblease/:workspaceName", handler.BreakBlobLease)
}

func (s *StorageAccountHandler) BreakBlobLease(c *gin.Context) {

	workspaceName := c.Param("workspaceName")
	if workspaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspaceName is required"})
		return
	}

	storageAccountName, err := s.storageAccountService.GetStorageAccountName(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = s.storageAccountService.BreakBlobLease(c.Request.Context(), storageAccountName, "repro-project-tf-state-files", workspaceName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"status": "success"})
}
