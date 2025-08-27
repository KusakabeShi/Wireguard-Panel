package server

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"wg-panel/internal/config"
	"wg-panel/internal/handlers"
	"wg-panel/internal/internalservice"
	"wg-panel/internal/middleware"
	"wg-panel/internal/services"
	"wg-panel/internal/utils"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg                 *config.Config
	engine              *gin.Engine
	pseudoBridgeService *internalservice.PseudoBridgeService
	snatRoamingService  *internalservice.SNATRoamingService
}

func NewServer(cfg *config.Config, pseudoBridge *internalservice.PseudoBridgeService, snatRoaming *internalservice.SNATRoamingService) *Server {
	return &Server{
		cfg:                 cfg,
		pseudoBridgeService: pseudoBridge,
		snatRoamingService:  snatRoaming,
	}
}

func (s *Server) Start(fw *internalservice.FirewallService) error {
	// Set Gin mode
	gin.SetMode(gin.ReleaseMode)
	s.engine = gin.New()
	s.engine.Use(gin.Logger(), gin.Recovery())

	// Setup services
	wgService := services.NewWireGuardService(s.cfg.WireGuardConfigPath)
	firewallService := fw
	startupService := services.NewStartupService(s.cfg, wgService, firewallService)

	interfaceService := services.NewInterfaceService(s.cfg, wgService)
	serverService := services.NewServerService(s.cfg, wgService, firewallService)
	clientService := services.NewClientService(s.cfg, wgService)

	// Initialize interfaces and firewall rules during startup
	if err := utils.CleanupRules(s.cfg.ServerId, 46, true); err != nil {
		log.Printf("Warning: failed to cleanup orphaned rules: %v", err)
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

	// Start background services configuration update
	go s.updateServicesConfiguration()

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
	apiPath := s.cfg.SiteURLPrefix + s.cfg.APIPrefix
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

	// Static file serving (after API routes)
	frontendPath := s.cfg.SiteFrontendPath
	if !filepath.IsAbs(frontendPath) {
		frontendPath = filepath.Join(".", frontendPath)
	}

	// Serve index.html at SiteURLPrefix
	sitePrefix := s.cfg.SiteURLPrefix
	if sitePrefix == "/" {
		s.engine.GET("/", func(c *gin.Context) {
			c.File(filepath.Join(frontendPath, "index.html"))
		})
	} else {
		s.engine.GET(sitePrefix, func(c *gin.Context) {
			c.File(filepath.Join(frontendPath, "index.html"))
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

		// Check if request is for frontend (starts with SiteURLPrefix)
		if len(requestPath) >= len(sitePrefix) && requestPath[:len(sitePrefix)] == sitePrefix {
			// Remove site prefix to get relative path
			relativePath := requestPath[len(sitePrefix):]
			if relativePath == "" {
				relativePath = "/"
			}

			// Try to serve static file from frontend directory
			filePath := filepath.Join(frontendPath, relativePath)

			// Check if file exists and is not a directory
			if info, err := http.Dir(frontendPath).Open(relativePath); err == nil {
				defer info.Close()
				if stat, err := info.Stat(); err == nil && !stat.IsDir() {
					c.File(filePath)
					return
				}
			}

			// Fallback to index.html for SPA routing
			c.File(filepath.Join(frontendPath, "index.html"))
			return
		}

		// Not a frontend or API request
		c.JSON(http.StatusNotFound, gin.H{"error": "Not found"})
	})
}

func (s *Server) updateServicesConfiguration() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Clean up expired sessions
			s.cfg.CleanExpiredSessions()
		}
	}
}
