package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/mehmetcc/definitive-authentication-service/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/mehmetcc/definitive-authentication-service/internal/person"
	"github.com/mehmetcc/definitive-authentication-service/internal/utils"
	"go.uber.org/zap"
)

// @title           Authentication Service API
// @version         1.0
// @description     This service provides user management endpoints.
// @termsOfService  http://example.com/terms/
//
// @host      localhost:666
// @BasePath  /api/v1
func main() {
	// load config
	cfg, err := utils.LoadConfig(".env")
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// init database
	db, err := utils.InitDatabase(cfg.Database.DSN())
	if err != nil {
		panic("Failed to connect to the database: " + err.Error())
	}
	if err := db.AutoMigrate(&person.Person{}); err != nil {
		panic("Failed to migrate database: " + err.Error())
	}

	// init logger
	logger, err := zap.NewProduction()
	if err != nil {
		panic("Failed to initialize logger: " + err.Error())
	}

	// init Gin router
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	// swagger endpoint
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// health check endpoint
	router.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// wire up person endpoints
	personRepo := person.NewPersonRepository(db)
	personService := person.NewPersonService(personRepo, logger)
	person.NewPersonHandler(router.Group("/api/v1"), personService, logger)

	// configure http server with graceful shutdown
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// start server in background
	go func() {
		logger.Info("starting HTTP server", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen failed", zap.Error(err))
		}
	}()

	// wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	// allow up to 10s for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	} else {
		logger.Info("server stopped gracefully")
	}
}
