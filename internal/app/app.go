package app

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raufendro/novacore/internal/cache"
	"github.com/raufendro/novacore/internal/config"
	"github.com/raufendro/novacore/internal/database"
	"github.com/raufendro/novacore/internal/middleware"
	"github.com/raufendro/novacore/internal/modules/auth"
	"github.com/raufendro/novacore/internal/modules/health"
	"github.com/raufendro/novacore/internal/routes"
	"github.com/raufendro/novacore/pkg/logger"
	"github.com/raufendro/novacore/pkg/security"
)

func Run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log, err := logger.New(cfg.App)
	if err != nil {
		return err
	}
	defer log.Sync()

	conn, err := database.Connect(cfg)
	if err != nil {
		return err
	}
	if conn.SQL != nil {
		if err := conn.SQL.AutoMigrate(&auth.User{}); err != nil {
			return err
		}
	}
	ctx := context.Background()
	redisClient, err := cache.Connect(ctx, cfg.Redis)
	if err != nil {
		return err
	}
	if redisClient != nil {
		defer redisClient.Close()
	}

	if cfg.App.Env == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := gin.New()
	engine.Use(
		middleware.RequestID(),
		middleware.Logger(log),
		middleware.Recovery(log),
		middleware.CORS(cfg.CORS.AllowedOrigins),
		middleware.SecureHeaders(),
		middleware.RateLimiter(cfg.Rate.Requests, cfg.Rate.Window),
		middleware.Timeout(cfg.Timeout),
	)

	jwt := security.NewJWTManager(cfg.JWT)
	api := engine.Group("/api/v1")
	health.RegisterRoutes(api)
	if conn.SQL != nil {
		authRepo := auth.NewRepository(conn.SQL)
		authService := auth.NewService(authRepo, jwt)
		auth.RegisterRoutes(api, auth.NewHandler(authService), jwt)
		routes.RegisterGenerated(api, conn.SQL, jwt)
	}

	server := &http.Server{Addr: ":" + cfg.App.Port, Handler: engine}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()
	fmt.Printf("%s running on http://localhost:%s\n", cfg.App.Name, cfg.App.Port)
	err = server.ListenAndServe()
	if err == http.ErrServerClosed {
		return nil
	}
	return err
}
