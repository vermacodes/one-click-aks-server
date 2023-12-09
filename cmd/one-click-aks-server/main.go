package main

import (
	"net/http"
	"one-click-aks-server/internal/handler"
	"os"
	"strconv"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/vermacodes/one-click-aks/app/server/middleware"
	"github.com/vermacodes/one-click-aks/app/server/repository"
	"github.com/vermacodes/one-click-aks/app/server/service"
	"golang.org/x/exp/slog"
)

type Status struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

var version string

func status(c *gin.Context) {

	status := Status{}
	status.Status = "OK"
	status.Version = version

	c.IndentedJSON(http.StatusOK, status)
}

func main() {

	// Logging config.
	logLevel := os.Getenv("LOG_LEVEL")
	logLevelInt, err := strconv.Atoi(logLevel)
	if err != nil {
		logLevelInt = 0
	}

	opts := &slog.HandlerOptions{
		AddSource: true,
		Level:     slog.Level(logLevelInt),
	}

	slogHandler := slog.NewTextHandler(os.Stdout, opts)
	slog.SetDefault(slog.New(slogHandler))

	// repositories
	logStreamRepository := repository.NewLogStreamRepository()
	loggingRepository := repository.NewLoggingRepository()
	actionStatusRepository := repository.NewActionStatusRepository()
	redisRepository := repository.NewRedisRepository()
	authRepository := repository.NewAuthRepository()
	storageAccountRepository := repository.NewStorageAccountRepository()
	workspaceRepository := repository.NewTfWorkspaceRepository()
	prefRepository := repository.NewPreferenceRepository()
	kVersionRepository := repository.NewKVersionRepository()
	labRepository := repository.NewLabRepository()
	terraformRepository := repository.NewTerraformRepository()
	deploymentRepository := repository.NewDeploymentRepository()

	// services
	logStreamService := service.NewLogStreamService(logStreamRepository)
	loggingService := service.NewLoggingService(loggingRepository)
	actionStatusService := service.NewActionStatusService(actionStatusRepository)
	redisService := service.NewRedisService(redisRepository)
	authService := service.NewAuthService(authRepository, actionStatusService, loggingService, redisRepository)
	storageAccountService := service.NewStorageAccountService(storageAccountRepository)
	workspaceService := service.NewWorkspaceService(workspaceRepository, storageAccountService, actionStatusService)
	prefService := service.NewPreferenceService(prefRepository, storageAccountService)
	kVersionService := service.NewKVersionService(kVersionRepository, prefService)
	labService := service.NewLabService(labRepository, kVersionService, storageAccountService, authService)
	terraformService := service.NewTerraformService(terraformRepository, labService, workspaceService, logStreamService, actionStatusService, kVersionService, storageAccountService, loggingService, authService)
	deploymentService := service.NewDeploymentService(deploymentRepository, labService, terraformService, actionStatusService, logStreamService, authService, workspaceService)

	// gin routers
	router := gin.Default()
	router.SetTrustedProxies(nil)

	config := cors.DefaultConfig()
	config.AllowOrigins = []string{"http://localhost:3000", "http://localhost:5173", "https://ashisverma.z13.web.core.windows.net", "https://actlabs.z13.web.core.windows.net", "https://actlabsbeta.z13.web.core.windows.net", "https://actlabs.azureedge.net", "https://*.azurewebsites.net"}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Authorization", "Content-Type"}

	router.Use(cors.New(config))

	authRouter := router.Group("/")
	authRouter.Use(middleware.AuthRequired(authService, logStreamService))

	actionStatusRouter := router.Group("/")
	actionStatusRouter.Use(middleware.ActionStatusMiddleware(actionStatusService))

	authWithActionRouter := authRouter.Group("/")
	authWithActionRouter.Use(middleware.ActionStatusMiddleware(actionStatusService))

	authWithTerraformActionRouter := authRouter.Group("/")
	authWithTerraformActionRouter.Use(middleware.TerraformActionMiddleware(actionStatusService))

	// server status
	router.GET("/status", status)

	// handlers
	handler.NewLogStreamHandler(router, logStreamService)
	handler.NewActionStatusHandler(router, actionStatusService)
	handler.NewRedisHandler(actionStatusRouter, redisService)
	handler.NewLoginHandler(router, authService)
	handler.NewAuthActionStatusHandler(authRouter, actionStatusService)
	handler.NewAuthHandler(authRouter, authService)
	handler.NewAuthWithActionStatusHandler(authWithActionRouter, authService)
	handler.NewStorageAccountHandler(authRouter, storageAccountService)
	handler.NewStorageAccountWithActionStatusHandler(authWithActionRouter, storageAccountService)
	handler.NewWorkspaceHandler(authRouter, workspaceService)
	handler.NewPreferenceHandler(authRouter, prefService)
	handler.NewKVersionHandler(authRouter, kVersionService)
	handler.NewLabHandler(authRouter, labService)
	handler.NewDeploymentHandler(authRouter, deploymentService, terraformService, actionStatusService)
	handler.NewDeploymentWithActionStatusHandler(authWithActionRouter, deploymentService, terraformService, actionStatusService)
	handler.NewTerraformWithActionStatusHandler(authWithTerraformActionRouter, terraformService, actionStatusService, deploymentService)

	// go routine to poll and delete deployments.
	// take seconds and multiply with 1000000000 and pass it to the function.
	go deploymentService.PollAndDeleteDeployments(60 * 1000000000)

	// run server
	router.Run()
}
