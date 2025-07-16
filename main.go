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

	"github.com/mehmetcc/definitive-authentication-service/internal/authentication"
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
	if err := db.AutoMigrate(&person.Person{}, &authentication.RefreshTokenRecord{}); err != nil {
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

	//
	// SWAGGER (protected by Basic Auth, not JWT)
	//
	swaggerGroup := router.Group("/swagger", gin.BasicAuth(gin.Accounts{
		cfg.Admin.Username: cfg.Admin.Password,
	}))
	swaggerGroup.GET("", ginSwagger.WrapHandler(swaggerFiles.Handler))
	swaggerGroup.GET("/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	//
	// WIRE UP SERVICES
	//
	personRepo := person.NewPersonRepository(db)
	personService := person.NewPersonService(personRepo, logger)

	recordRepo := authentication.NewRecordRepository(db)
	authService := authentication.NewAuthenticationService(
		personService,
		recordRepo,
		logger,
		// access token settings
		cfg.Token.AccessTokenSecret,
		15*time.Minute,
		// refresh token settings
		cfg.Token.RefreshTokenSecret,
		time.Duration(cfg.Token.RefreshTokenExpiry)*time.Hour,
	)

	api := router.Group("/api/v1")
	authentication.NewAuthHandler(api, authService, logger)

	api.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	authGroup := api.Group("/")
	authGroup.Use(
		authentication.AuthMiddleware(personService, cfg.Token.AccessTokenSecret, logger),
	)
	authGroup.GET("/persons/me", func(c *gin.Context) {
		raw, _ := c.Get(person.ContextUserKey)
		user := raw.(*person.Person)
		c.JSON(http.StatusOK, user)
	})

	adminGroup := api.Group("/")
	adminGroup.Use(
		authentication.AuthMiddleware(personService, cfg.Token.AccessTokenSecret, logger),
		func(c *gin.Context) {
			raw, exists := c.Get(person.ContextUserKey)
			if !exists {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
				return
			}
			user := raw.(*person.Person)
			if user.Role != person.Admin {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
				return
			}
			c.Next()
		},
	)
	person.NewPersonHandler(adminGroup, personService, logger)

	//
	// START SERVER
	//
	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}
	go func() {
		logger.Info("starting HTTP server", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("listen failed", zap.Error(err))
		}
	}()

	// graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", zap.Error(err))
	} else {
		logger.Info("server stopped gracefully")
	}
}
