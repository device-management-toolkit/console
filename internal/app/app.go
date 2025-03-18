// Package app configures and runs application.
package app

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/open-amt-cloud-toolkit/console/config"
	consolehttp "github.com/open-amt-cloud-toolkit/console/internal/controller/http"
	wsv1 "github.com/open-amt-cloud-toolkit/console/internal/controller/ws/v1"
	"github.com/open-amt-cloud-toolkit/console/internal/usecase"
	"github.com/open-amt-cloud-toolkit/console/pkg/db"
	"github.com/open-amt-cloud-toolkit/console/pkg/httpserver"
	"github.com/open-amt-cloud-toolkit/console/pkg/logger"
)

var Version = "DEVELOPMENT"

// hideTerminal hides the terminal window based on the platform.
func hideTerminal() {
	switch runtime.GOOS {
	case "windows":
		hideTerminalWindows()
	case "darwin", "linux":
		hideTerminalUnix()
	}
}

// Default (no-op) function for non-Windows platforms
func hideTerminalWindows() {
	// Do nothing on non-Windows platforms
}
func hideTerminalUnix() {
	if os.Getppid() != 1 { // Not already detached
		syscall.Setsid()
	}
}

// Run creates objects via constructors.
func Run(cfg *config.Config) {
	hideTerminal()
	log := logger.New(cfg.Log.Level)
	cfg.App.Version = Version
	log.Info("app - Run - version: " + cfg.App.Version)
	// Repository
	database, err := db.New(cfg.DB.URL, sql.Open, db.MaxPoolSize(cfg.DB.PoolMax), db.EnableForeignKeys(true))
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
	defaultConfig.AllowOrigins = cfg.HTTP.AllowedOrigins
	defaultConfig.AllowHeaders = cfg.HTTP.AllowedHeaders

	handler.Use(cors.New(defaultConfig))
	consolehttp.NewRouter(handler, log, *usecases, cfg)

	upgrader := &websocket.Upgrader{
		ReadBufferSize:  4096,
		WriteBufferSize: 4096,
		Subprotocols:    []string{"direct"},
		CheckOrigin: func(_ *http.Request) bool {
			return true
		},
		EnableCompression: false,
	}

	wsv1.RegisterRoutes(handler, log, usecases.Devices, upgrader)
	httpServer := httpserver.New(handler, httpserver.Port(cfg.HTTP.Host, cfg.HTTP.Port))

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
