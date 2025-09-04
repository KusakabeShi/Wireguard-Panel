package handlers

import (
	"net/http"

	"wg-panel/internal/config"
	"wg-panel/internal/middleware"

	"github.com/gin-gonic/gin"
)

type ServiceHandler struct {
	cfg  *config.Config
	auth *middleware.AuthMiddleware
}

func NewServiceHandler(cfg *config.Config, auth *middleware.AuthMiddleware) *ServiceHandler {
	return &ServiceHandler{
		cfg:  cfg,
		auth: auth,
	}
}

func (h *ServiceHandler) GetServiceConfig(c *gin.Context) {
	response := map[string]interface{}{
		"wireguardConfigPath": h.cfg.WireGuardConfigPath,
		"user":                h.cfg.User,
		"listenIP":            h.cfg.ListenIP,
		"listenPort":          h.cfg.ListenPort,
		"siteUrlPrefix":       h.cfg.BasePath,
		"apiPrefix":           h.cfg.APIPrefix,
		"wgIfPrefix":          h.cfg.WgIfPrefix,
	}

	c.JSON(http.StatusOK, response)
}

func (h *ServiceHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/login", h.auth.Login)
	router.POST("/logout", h.auth.RequireAuth(), h.auth.Logout)
	router.GET("/config", h.auth.RequireAuth(), h.GetServiceConfig)
	router.PUT("/password", h.auth.RequireAuth(), h.auth.ChangePassword)
}
