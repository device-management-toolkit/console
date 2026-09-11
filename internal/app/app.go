// Package app configures and runs application.
package app

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"slices"
	"strings"
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
	"github.com/device-management-toolkit/console/pkg/httpserver"
	"github.com/device-management-toolkit/console/pkg/logger"
)

// CertStore holds the certificate store for domain certificates (set during Init).
var CertStore security.Storager

var Version = "DEVELOPMENT"

// Run creates objects via constructors.
func Run(cfg *config.Config, log logger.Interface) {
	cfg.Version = Version
	log.Info("app - Run - version: " + cfg.Version)
	// route standard and Gin logs through our JSON logger
	logger.SetupStdLog(log)
	logger.SetupGin(log)
	// Repositories — provider (postgres/sqlite/mongo) chosen by config.
	repos, err := buildRepos(cfg, log)
	if err != nil {
		log.Fatal(fmt.Errorf("app - Run - buildRepos: %w", err))
	}

	defer func() {
		if cerr := repos.Closer.Close(); cerr != nil {
			log.Error(fmt.Errorf("app - Run - repos.Closer.Close: %w", cerr))
		}
	}()

	// Use case
	usecases := usecase.NewUseCases(repos, log, CertStore)

	handler := setupHTTPHandler(cfg, log, usecases)

	ciraServer := setupCIRAServer(cfg, log, repos.Closer, usecases)

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
	// Ahead of CORS on purpose: the CORS middleware answers preflights and
	// rejects disallowed origins itself, so anything after it never runs.
	handler.Use(securityHeaders())

	wildcardOrigin := slices.Contains(cfg.AllowedOrigins, "*")
	if wildcardOrigin {
		log.Warn(`http.allowed_origins contains "*": every site may read API responses ` +
			`(Access-Control-Allow-Origin: *), cookie auth is disabled, and cross-origin ` +
			`redirection (KVM/SOL/IDER) is refused. List the Console UI origins explicitly.`)
	}

	defaultConfig := cors.DefaultConfig()
	defaultConfig.AllowOrigins = cfg.AllowedOrigins
	defaultConfig.AllowHeaders = cfg.AllowedHeaders
	defaultConfig.AllowCredentials = cfg.AllowCredentials && !wildcardOrigin

	handler.Use(cors.New(defaultConfig))
	httpapi.NewRouter(handler, log, *usecases, cfg)

	// Optionally enable pprof endpoints (e.g., for staging) via env ENABLE_PPROF=true
	if os.Getenv("ENABLE_PPROF") == "true" {
		ginpprof.Register(handler, "debug/pprof")
		log.Info("pprof enabled at /debug/pprof/")
	}

	// Subprotocols is deliberately unset: the relay negotiates the caller's
	// redirection token as the subprotocol, so wsv1 sets it on a per-request
	// copy of this upgrader. Anything configured here would be overwritten.
	upgrader := &websocket.Upgrader{
		ReadBufferSize:    64 * 1024,
		WriteBufferSize:   64 * 1024,
		CheckOrigin:       newOriginChecker(cfg.AllowedOrigins),
		EnableCompression: cfg.WSCompression,
	}

	wsv1.RegisterRoutes(handler, log, usecases.Devices, upgrader)

	return handler
}

// newOriginChecker builds the websocket.Upgrader CheckOrigin callback guarding
// the redirection relay (/relay/webrelay.ashx) against Cross-Site WebSocket
// Hijacking: without it, any page could open a KVM/SOL/IDER relay using a
// session the browser attaches automatically.
//
// Unlike the CORS middleware it deliberately does not honor "*". No site has a
// legitimate reason to relay someone else's KVM session, so a wildcard
// allowlist degrades to same-origin only rather than allowing everything —
// which keeps pre-existing `allowed_origins: ["*"]` installs (the shipped
// default until now) safe without an admin editing config.yml.
func newOriginChecker(allowedOrigins []string) func(*http.Request) bool {
	// If "*" is present, degrade to same-origin only for the relay.
	if slices.Contains(allowedOrigins, "*") {
		allowedOrigins = nil
	}

	normalizedAllowed := normalizeAllowedOrigins(allowedOrigins)

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		// Non-browser clients (rpc-go, CLI tooling, tests) send no Origin at
		// all. Browsers always send one on a websocket handshake, so allowing
		// the empty case cannot be used by the hijacking this guards against —
		// and the JWT in Sec-Websocket-Protocol is still required either way.
		if origin == "" {
			return true
		}

		originURL := parseOrigin(origin)
		// Opaque origin: "null" from a sandboxed iframe, or a data:/file: page.
		if originURL == nil {
			return false
		}

		// The embedded UI is served by this very server, so same-origin always
		// passes regardless of what the allowlist says.
		if strings.EqualFold(originURL.Host, r.Host) {
			return true
		}

		return slices.Contains(normalizedAllowed, normalizedFrom(originURL))
	}
}

func normalizeAllowedOrigins(allowedOrigins []string) []string {
	normalizedAllowed := make([]string, 0, len(allowedOrigins))
	for _, allowedOrigin := range allowedOrigins {
		if normalizedOrigin := normalizeOrigin(allowedOrigin); normalizedOrigin != "" {
			normalizedAllowed = append(normalizedAllowed, normalizedOrigin)
		}
	}

	return normalizedAllowed
}

// parseOrigin returns the parsed origin, or nil if it is not a usable
// scheme://host pair ("", "*", "null", "/some/path").
func parseOrigin(origin string) *url.URL {
	if origin == "" {
		return nil
	}

	originURL, err := url.Parse(origin)
	if err != nil || originURL.Scheme == "" || originURL.Host == "" {
		return nil
	}

	return originURL
}

// normalizedFrom reduces a parsed origin to scheme://host, dropping any path
// and lowercasing the host. url.Parse lowercases the scheme but not the host,
// and gin-contrib/cors lowercases both — matching it here keeps a mixed-case
// entry in allowed_origins from passing CORS but failing the relay.
func normalizedFrom(originURL *url.URL) string {
	return originURL.Scheme + "://" + strings.ToLower(originURL.Host)
}

func normalizeOrigin(origin string) string {
	originURL := parseOrigin(origin)
	if originURL == nil {
		return ""
	}

	return normalizedFrom(originURL)
}

// securityHeaders sets X-Content-Type-Options: nosniff to stop MIME sniffing.
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")

		c.Next()
	}
}

func setupCIRAServer(cfg *config.Config, log logger.Interface, closer io.Closer, usecases *usecase.Usecases) *cira.Server {
	if cfg.DisableCIRA {
		return nil
	}

	ciraCertFile := fmt.Sprintf("config/%s_cert.pem", cfg.CommonName)
	ciraKeyFile := fmt.Sprintf("config/%s_key.pem", cfg.CommonName)

	ciraServer, err := cira.NewServer(ciraCertFile, ciraKeyFile, usecases.Devices, log)
	if err != nil {
		_ = closer.Close()

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
