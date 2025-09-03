package server

import (
	"embed"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"wg-panel/internal/config"
	"wg-panel/internal/handlers"
	"wg-panel/internal/internalservice"
	"wg-panel/internal/logging"
	"wg-panel/internal/middleware"
	"wg-panel/internal/services"
	"wg-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg        *config.Config
	engine     *gin.Engine
	frontendFS embed.FS
}

func NewServer(cfg *config.Config, frontendFS embed.FS) *Server {
	return &Server{
		cfg:        cfg,
		frontendFS: frontendFS,
	}
}

func (s *Server) Start(fw *internalservice.FirewallService, logLevel logging.LogLevel) error {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()
	s.engine.Use(CustomLogger(logLevel), gin.Recovery())

	// Setup services
	wgService := services.NewWireGuardService(s.cfg.WireGuardConfigPath)
	firewallService := fw
	startupService := services.NewStartupService(s.cfg, wgService, firewallService)

	interfaceService := services.NewInterfaceService(s.cfg, wgService)
	serverService := services.NewServerService(s.cfg, wgService, firewallService)
	clientService := services.NewClientService(s.cfg, wgService)

	// Initialize interfaces and firewall rules during startup
	if err := utils.CleanupRules(s.cfg.ServerId, 46, nil, true); err != nil {
		logging.LogError("Warning: failed to cleanup orphaned rules: %v", err)
	}
	if err := startupService.InitializeInterfaces(); err != nil {
		return fmt.Errorf("failed to initialize interfaces: %v", err)
	}

	// Setup middleware
	authMiddleware := middleware.NewAuthMiddleware(s.cfg)

	// Setup handlers
	serviceHandler := handlers.NewServiceHandler(s.cfg, authMiddleware)
	interfaceHandler := handlers.NewInterfaceHandler(interfaceService)
	serverHandler := handlers.NewServerHandler(serverService)
	clientHandler := handlers.NewClientHandler(clientService)

	// Setup routes
	s.setupRoutes(serviceHandler, interfaceHandler, serverHandler, clientHandler, authMiddleware)

	// Start server
	var listenAddr string
	if strings.Contains(s.cfg.ListenIP, ":") {
		listenAddr = fmt.Sprintf("[%s]:%d", s.cfg.ListenIP, s.cfg.ListenPort)
	} else {
		listenAddr = fmt.Sprintf("%s:%d", s.cfg.ListenIP, s.cfg.ListenPort)
	}
	return http.ListenAndServe(listenAddr, s.engine)
}

func (s *Server) setupRoutes(
	serviceHandler *handlers.ServiceHandler,
	interfaceHandler *handlers.InterfaceHandler,
	serverHandler *handlers.ServerHandler,
	clientHandler *handlers.ClientHandler,
	authMiddleware *middleware.AuthMiddleware,
) {
	// API routes first to avoid conflicts
	apiPath := s.cfg.BasePath + s.cfg.APIPrefix
	if apiPath[len(apiPath)-1] != '/' {
		apiPath += "/"
	}
	api := s.engine.Group(apiPath[:len(apiPath)-1]) // Remove trailing slash

	// Service routes
	serviceGroup := api.Group("/service")
	serviceHandler.RegisterRoutes(serviceGroup)

	// Protect all other routes with authentication
	protected := api.Group("")
	protected.Use(authMiddleware.RequireAuth())

	// Interface routes
	interfacesGroup := protected.Group("/interfaces")
	interfaceHandler.RegisterRoutes(interfacesGroup)

	// Server routes (nested under interfaces)
	interfacesGroup.Group("/:ifId/servers").Use(func(c *gin.Context) {
		// Pass ifId to nested handlers
		c.Next()
	})
	serversGroup := interfacesGroup.Group("/:ifId/servers")
	serverHandler.RegisterRoutes(serversGroup)

	// Client routes (nested under servers)
	serversWithClientGroup := interfacesGroup.Group("/:ifId/servers/:serverId")
	clientHandler.RegisterRoutes(serversWithClientGroup)

	// Static file serving (after API routes) using embedded filesystem
	sitePrefix := s.cfg.BasePath
	if sitePrefix == "/" {
		s.engine.GET("/", func(c *gin.Context) {
			data, err := s.frontendFS.ReadFile("frontend/build/index.html")
			if err != nil {
				c.String(http.StatusNotFound, "File not found")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		})
	} else {
		s.engine.GET(sitePrefix, func(c *gin.Context) {
			data, err := s.frontendFS.ReadFile("frontend/build/index.html")
			if err != nil {
				c.String(http.StatusNotFound, "File not found")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", data)
		})
	}

	// Handle all other routes - serve static files or SPA fallback
	s.engine.NoRoute(func(c *gin.Context) {
		requestPath := c.Request.URL.Path

		// If request starts with API prefix, return 404
		apiPrefix := apiPath[:len(apiPath)-1] // Remove trailing slash for comparison
		if len(requestPath) >= len(apiPrefix) && requestPath[:len(apiPrefix)] == apiPrefix {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
			return
		}

		// Check if request is for frontend (starts with BasePath)
		if len(requestPath) >= len(sitePrefix) && requestPath[:len(sitePrefix)] == sitePrefix {
			// Remove site prefix to get relative path
			relativePath := requestPath[len(sitePrefix):]
			if relativePath == "" {
				relativePath = "/"
			}

			// Try to serve static file from embedded filesystem
			embedPath := "frontend/build/" + relativePath
			if data, err := s.frontendFS.ReadFile(embedPath); err == nil {
				// Determine content type based on file extension
				contentType := "text/plain"
				if ext := filepath.Ext(relativePath); ext != "" {
					switch ext {
					case ".html":
						contentType = "text/html; charset=utf-8"
					case ".css":
						contentType = "text/css; charset=utf-8"
					case ".js":
						contentType = "application/javascript; charset=utf-8"
					case ".json":
						contentType = "application/json"
					case ".png":
						contentType = "image/png"
					case ".jpg", ".jpeg":
						contentType = "image/jpeg"
					case ".gif":
						contentType = "image/gif"
					case ".svg":
						contentType = "image/svg+xml"
					case ".ico":
						contentType = "image/x-icon"
					}
				}
				c.Data(http.StatusOK, contentType, data)
				return
			}

			// Fallback to index.html for SPA routing
			// if data, err := s.frontendFS.ReadFile("frontend/build/index.html"); err == nil {
			// 	c.Data(http.StatusOK, "text/html; charset=utf-8", data)
			// } else {
			// 	c.String(http.StatusNotFound, "File not found")
			// }
			// return
		}
		// Not a frontend or API request
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	})
}

func CustomLogger(level logging.LogLevel) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()
		method := c.Request.Method
		path := c.Request.URL.Path
		clientIP := c.ClientIP()

		// Determine prefix from status/method
		var reqlevel logging.LogLevel
		switch {
		case status >= 400:
			reqlevel = logging.LogLevelError
		case method != "GET":
			reqlevel = logging.LogLevelInfo
		default:
			reqlevel = logging.LogLevelVerbose
		}

		var prefix string
		switch reqlevel {
		case logging.LogLevelError:
			prefix = "[ERROR]"
		case logging.LogLevelInfo:
			prefix = "[INFO]"
		case logging.LogLevelVerbose:
			prefix = "[VERBOSE]"
		}

		if reqlevel > level {
			return
		}

		log.Printf("%s %d | %13v | %15s | %-7s %s",
			prefix, status, latency, clientIP, method, path)
	}
}
