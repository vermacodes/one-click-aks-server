package handler

import (
	"net/http"

	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
)

type StorageAccountHandler struct {
	storageAccountService entity.StorageAccountService
}

func NewStorageAccountHandler(r *gin.RouterGroup, service entity.StorageAccountService) {
	handler := &StorageAccountHandler{
		storageAccountService: service,
	}

	//r.GET("/storageaccount", handler.GetStorageAccountConfiguration)
	r.GET("/storageaccount", handler.GetStorageAccount)
}

func NewStorageAccountWithActionStatusHandler(r *gin.RouterGroup, service entity.StorageAccountService) {
	handler := &StorageAccountHandler{
		storageAccountService: service,
	}

	//r.POST("/storageaccount", handler.ConfigureStorageAccount)
	r.PUT("/storageaccount/breakbloblease/:workspaceName", handler.BreakBlobLease)
}

func (s *StorageAccountHandler) GetStorageAccount(c *gin.Context) {
	storageAccount, err := s.storageAccountService.GetStorageAccount()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.IndentedJSON(http.StatusOK, storageAccount)
}

// func (s *StorageAccountHandler) GetStorageAccountConfiguration(c *gin.Context) {
// 	configuration := entity.StateConfiguration{}
// 	storageAccountName, err := s.storageAccountService.GetStorageAccountName()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	resourceGroup, err := s.storageAccountService.GetResourceGroup()
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	storageAccount, err := s.storageAccountService.GetStorageAccount(storageAccountName)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}
// 	blobContainer, err := s.storageAccountService.GetBlobContainer(storageAccountName, "tfstate")
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
// 		return
// 	}

// 	configuration.ResourceGroup = resourceGroup
// 	configuration.StorageAccount = storageAccount
// 	configuration.BlobContainer = blobContainer

// 	c.IndentedJSON(http.StatusOK, configuration)
// }

func (s *StorageAccountHandler) BreakBlobLease(c *gin.Context) {

	workspaceName := c.Param("workspaceName")
	if workspaceName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "workspaceName is required"})
		return
	}

	storageAccountName, err := s.storageAccountService.GetStorageAccountName()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = s.storageAccountService.BreakBlobLease(storageAccountName, "tfstate", workspaceName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, gin.H{"status": "success"})
}
