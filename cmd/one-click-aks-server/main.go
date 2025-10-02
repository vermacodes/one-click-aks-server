package main

import (
	"net/http"
	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/cache"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/handler"
	"one-click-aks-server/internal/logging"
	"one-click-aks-server/internal/mise"
	"one-click-aks-server/internal/miseadapter"
	"strings"

	"one-click-aks-server/internal/middleware"
	"one-click-aks-server/internal/repository"
	"one-click-aks-server/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
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
	logging.SetupLogger()

	// Disable Gin's default logging and use our custom logger
	middleware.DisableGinDefaultLogging()

	appConfig := config.NewConfig()
	auth := auth.NewAuth(appConfig)
	rdb := cache.NewRedisClient()

	// mise
	miseServer := mise.Server{
		ContainerClient: miseadapter.NewMISEAdapter(http.DefaultClient, appConfig.MiseEndpoint),
		VerboseLogging:  appConfig.MiseVerboseLogging,
	}

	// repositories
	logStreamRepository := repository.NewLogStreamRepository()
	actionStatusRepository := repository.NewActionStatusRepository()
	redisRepository := repository.NewRedisRepository()
	authRepository := repository.NewAuthRepository(appConfig, auth, rdb)
	storageAccountRepository := repository.NewStorageAccountRepository(auth, rdb, appConfig)
	workspaceRepository := repository.NewTfWorkspaceRepository(appConfig)
	prefRepository := repository.NewPreferenceRepository(auth, appConfig)
	kVersionRepository := repository.NewKVersionRepository(appConfig, auth, rdb)
	aroVersionRepository := repository.NewAROVersionRepository(appConfig, auth, rdb)
	labRepository := repository.NewLabRepository(appConfig, auth)
	terraformRepository := repository.NewTerraformRepository(appConfig)
	deploymentRepository := repository.NewDeploymentRepository(appConfig, auth, rdb)

	// services
	logStreamService := service.NewLogStreamService(logStreamRepository)
	actionStatusService := service.NewActionStatusService(actionStatusRepository)
	redisService := service.NewRedisService(redisRepository)
	authService := service.NewAuthService(authRepository)
	storageAccountService := service.NewStorageAccountService(storageAccountRepository)
	workspaceService := service.NewWorkspaceService(workspaceRepository, storageAccountService, actionStatusService)
	prefService := service.NewPreferenceService(prefRepository, storageAccountService)
	kVersionService := service.NewKVersionService(kVersionRepository, prefService)
	aroVersionService := service.NewAROVersionService(aroVersionRepository, prefService)
	labService := service.NewLabService(labRepository, kVersionService, aroVersionService, storageAccountService, authService)
	terraformService := service.NewTerraformService(terraformRepository, labService, workspaceService, logStreamService, actionStatusService, kVersionService, aroVersionService, storageAccountService, authService)
	deploymentService := service.NewDeploymentService(deploymentRepository, labService, terraformService, actionStatusService, logStreamService, authService, workspaceService, *appConfig)

	// gin routers
	router := gin.New() // Use gin.New() instead of gin.Default() to avoid default middleware
	router.SetTrustedProxies(nil)

	// Add our custom middlewares in order
	router.Use(middleware.ContextMiddleware())    // First: Generate trace ID
	router.Use(middleware.GinLoggerWithTraceID()) // Second: Log with trace ID
	router.Use(gin.Recovery())                    // Third: Recovery middleware

	config := cors.DefaultConfig()
	config.AllowOrigins = strings.Split(appConfig.CorsAllowOrigins, ",")
	config.AllowMethods = strings.Split(appConfig.CorsAllowMethods, ",")
	config.AllowHeaders = strings.Split(appConfig.CorsAllowHeaders, ",")

	router.Use(cors.New(config))

	authRouter := router.Group("/")
	authRouter.Use(middleware.AuthRequired(miseServer, authService, logStreamService))

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
	// handler.NewLoginHandler(router, authService)
	handler.NewAuthActionStatusHandler(authRouter, actionStatusService)
	handler.NewAuthHandler(authRouter, authService)
	// handler.NewAuthWithActionStatusHandler(authWithActionRouter, authService)
	// handler.NewStorageAccountHandler(authRouter, storageAccountService)
	handler.NewStorageAccountWithActionStatusHandler(authWithActionRouter, storageAccountService)
	handler.NewWorkspaceHandler(authRouter, workspaceService)
	handler.NewPreferenceHandler(authRouter, prefService)
	handler.NewKVersionHandler(authRouter, kVersionService)
	handler.NewAROVersionHandler(authRouter, aroVersionService)
	handler.NewLabHandler(authRouter, labService)
	handler.NewDeploymentHandler(authRouter, deploymentService, terraformService, actionStatusService)
	handler.NewDeploymentWithActionStatusHandler(authWithActionRouter, deploymentService, terraformService, actionStatusService)
	handler.NewDeploymentWithTerraformActionStatusHandler(authWithTerraformActionRouter, deploymentService, terraformService, actionStatusService)
	handler.NewTerraformWithActionStatusHandler(authWithTerraformActionRouter, terraformService, actionStatusService, deploymentService)

	// go routine to poll and delete deployments.
	// take seconds and multiply with 1000000000 and pass it to the function.
	// go deploymentService.PollAndDeleteDeployments(60 * 1000000000)

	// run server
	router.Run()
}
