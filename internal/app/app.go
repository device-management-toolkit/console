// Package app configures and runs application.
package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-contrib/cors"
	ginpprof "github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/device-management-toolkit/console/config"
	consolehttp "github.com/device-management-toolkit/console/internal/controller/http"
	wsv1 "github.com/device-management-toolkit/console/internal/controller/ws/v1"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/httpserver"
	"github.com/device-management-toolkit/console/pkg/logger"
)

var Version = "DEVELOPMENT"

// Run creates objects via constructors.
func Run(cfg *config.Config) {
	log := logger.New(cfg.Level)
	cfg.Version = Version
	log.Info("app - Run - version: " + cfg.Version)
	// Repository
	database, err := db.New(cfg.DB.URL, sql.Open, db.MaxPoolSize(cfg.PoolMax), db.EnableForeignKeys(true))
	if err != nil {
		log.Fatal(fmt.Errorf("app - Run - db.New: %w", err))
	}
	defer database.Close()

	// Use case
	usecases := usecase.NewUseCases(database, log)

	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	// HTTP Server
	handler := gin.New()

	defaultConfig := cors.DefaultConfig()
	defaultConfig.AllowOrigins = cfg.AllowedOrigins
	defaultConfig.AllowHeaders = cfg.AllowedHeaders

	handler.Use(cors.New(defaultConfig))
	consolehttp.NewRouter(handler, log, *usecases, cfg)

	// Optionally enable pprof endpoints (e.g., for staging) via env ENABLE_PPROF=true
	if os.Getenv("ENABLE_PPROF") == "true" {
		// Register pprof handlers under /debug/pprof without exposing DefaultServeMux
		ginpprof.Register(handler, "debug/pprof")
		log.Info("pprof enabled at /debug/pprof/")
	}

	upgrader := &websocket.Upgrader{
		// Larger buffers reduce per-frame overhead and syscalls for KVM streaming
		ReadBufferSize:  64 * 1024,
		WriteBufferSize: 64 * 1024,
		Subprotocols:    []string{"direct"},
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
		EnableCompression: cfg.WSCompression,
	}

	wsv1.RegisterRoutes(handler, log, usecases.Devices, upgrader)
	httpServer := httpserver.New(handler, httpserver.Port(cfg.Host, cfg.Port))

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		log.Info("app - Run - signal: " + s.String())
	case err = <-httpServer.Notify():
		log.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		log.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}
}
