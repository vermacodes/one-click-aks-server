package handler

import (
	"net/http"
	"one-click-aks-server/internal/entity"

	"github.com/gin-gonic/gin"
)

type authHandler struct {
	authService entity.AuthService
}

func NewAuthHandler(r *gin.RouterGroup, service entity.AuthService) {
	handler := &authHandler{
		authService: service,
	}

	// r.GET("/service-principal-login", handler.ServicePrincipalLogin)
	r.GET("/accounts", handler.GetAccounts)
}

func (a *authHandler) GetAccounts(c *gin.Context) {
	account, err := a.authService.GetSubscriptionDetails(c.Request.Context())
	accounts := []entity.Account{account}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.IndentedJSON(http.StatusOK, accounts)
}
