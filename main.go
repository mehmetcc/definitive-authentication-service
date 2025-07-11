// @title           Definitive Authentication Service API
// @version         1.0
// @description     This is the definitive authentication service for user management.
// @host            localhost:666
// @BasePath        /api/v1

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/mehmetcc/definitive-authentication-service/config"
	_ "github.com/mehmetcc/definitive-authentication-service/docs"
	"github.com/mehmetcc/definitive-authentication-service/internal/db"
	"github.com/mehmetcc/definitive-authentication-service/internal/person"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.uber.org/zap"
)

// @contact.name   Not me
// @contact.email  i@dont.care
// @license.name   Do What The F*ck You Want To Public License
// @license.version 1.0
// @license.url    https://www.wtfpl.net/txt/copying/

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("failed to load config, reverting to default settings", zap.Error(err))
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Name,
		cfg.Database.SSLMode,
	)

	if err := db.Init(dsn, *logger); err != nil {
		logger.Fatal("failed to connect to database", zap.Error(err))
	}

	personRepo := person.NewPersonRepository(db.DB, logger)
	personHandler := person.NewPersonHandler(personRepo, logger, cfg)

	rootRouter := chi.NewRouter()
	rootRouter.Use(middleware.RequestID)
	rootRouter.Use(middleware.RealIP)
	rootRouter.Use(middleware.Logger)
	rootRouter.Use(middleware.Recoverer)
	rootRouter.Use(middleware.Timeout(15 * time.Second))

	personRouter := chi.NewRouter()
	personHandler.RegisterRoutes(personRouter)
	rootRouter.Mount(cfg.Server.BasePath, personRouter)

	rootRouter.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://localhost:%d/swagger/doc.json", cfg.Server.Port)),
	))

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Server.Port),
		Handler: rootRouter,

		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	logger.Info("starting server", zap.String("addr", srv.Addr))
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Fatal("server error", zap.Error(err))
	}
}
