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

	"github.com/device-management-toolkit/go-wsman-messages/v2/pkg/security"

	"github.com/device-management-toolkit/console/config"
	"github.com/device-management-toolkit/console/internal/controller/httpapi"
	"github.com/device-management-toolkit/console/internal/controller/tcp/cira"
	wsv1 "github.com/device-management-toolkit/console/internal/controller/ws/v1"
	"github.com/device-management-toolkit/console/internal/usecase"
	"github.com/device-management-toolkit/console/pkg/db"
	"github.com/device-management-toolkit/console/pkg/httpserver"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// CertStore holds the certificate store for domain certificates (set during Init).
var CertStore security.Storager

var Version = "DEVELOPMENT"

// Run creates objects via constructors.
func Run(cfg *config.Config) {
	log := logger.New(cfg.Level)
	cfg.Version = Version
	log.Info("app - Run - version: " + cfg.Version)
	// route standard and Gin logs through our JSON logger
	logger.SetupStdLog(log)
	logger.SetupGin(log)
	// Repository
	database, err := db.New(cfg.DB.URL, sql.Open, db.MaxPoolSize(cfg.PoolMax), db.EnableForeignKeys(true))
	if err != nil {
		log.Fatal(fmt.Errorf("app - Run - db.New: %w", err))
	}

	defer database.Close()

	// Use case
	usecases := usecase.NewUseCases(database, log, CertStore)

	handler := setupHTTPHandler(cfg, log, usecases)

	ciraServer := setupCIRAServer(cfg, log, database, usecases)

	httpServer := httpserver.New(
		handler,
		httpserver.Port(cfg.Host, cfg.Port),
		httpserver.TLS(cfg.TLS.Enabled, cfg.TLS.CertFile, cfg.TLS.KeyFile),
		httpserver.Logger(log),
	)

	waitForShutdown(log, httpServer, ciraServer)
	shutdownServers(log, httpServer, ciraServer)
}

func setupHTTPHandler(cfg *config.Config, log logger.Interface, usecases *usecase.Usecases) *gin.Engine {
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	handler := gin.New()

	defaultConfig := cors.DefaultConfig()
	defaultConfig.AllowOrigins = cfg.AllowedOrigins
	defaultConfig.AllowHeaders = cfg.AllowedHeaders

	handler.Use(cors.New(defaultConfig))
	httpapi.NewRouter(handler, log, *usecases, cfg)

	// Optionally enable pprof endpoints (e.g., for staging) via env ENABLE_PPROF=true
	if os.Getenv("ENABLE_PPROF") == "true" {
		ginpprof.Register(handler, "debug/pprof")
		log.Info("pprof enabled at /debug/pprof/")
	}

	upgrader := &websocket.Upgrader{
		ReadBufferSize:    64 * 1024,
		WriteBufferSize:   64 * 1024,
		Subprotocols:      []string{"direct"},
		CheckOrigin:       func(_ *http.Request) bool { return true },
		EnableCompression: cfg.WSCompression,
	}

	wsv1.RegisterRoutes(handler, log, usecases.Devices, upgrader)

	return handler
}

func setupCIRAServer(cfg *config.Config, log logger.Interface, database *db.SQL, usecases *usecase.Usecases) *cira.Server {
	if cfg.DisableCIRA {
		return nil
	}

	ciraCertFile := fmt.Sprintf("config/%s_cert.pem", cfg.CommonName)
	ciraKeyFile := fmt.Sprintf("config/%s_key.pem", cfg.CommonName)

	ciraServer, err := cira.NewServer(ciraCertFile, ciraKeyFile, usecases.Devices, log)
	if err != nil {
		database.Close()
		log.Fatal("CIRA Server failed: %v", err)
	}

	return ciraServer
}

func waitForShutdown(log logger.Interface, httpServer *httpserver.Server, ciraServer *cira.Server) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	if ciraServer != nil {
		select {
		case s := <-interrupt:
			log.Info("app - Run - signal: " + s.String())
		case err := <-httpServer.Notify():
			log.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
		case ciraErr := <-ciraServer.Notify():
			log.Error(fmt.Errorf("app - Run - ciraServer.Notify: %w", ciraErr))
		}
	} else {
		select {
		case s := <-interrupt:
			log.Info("app - Run - signal: " + s.String())
		case err := <-httpServer.Notify():
			log.Error(fmt.Errorf("app - Run - httpServer.Notify: %w", err))
		}
	}
}

func shutdownServers(log logger.Interface, httpServer *httpserver.Server, ciraServer *cira.Server) {
	if err := httpServer.Shutdown(); err != nil {
		log.Error(fmt.Errorf("app - Run - httpServer.Shutdown: %w", err))
	}

	if ciraServer != nil {
		if err := ciraServer.Shutdown(); err != nil {
			log.Error(fmt.Errorf("app - Run - ciraServer.Shutdown: %w", err))
		}
	}
}
