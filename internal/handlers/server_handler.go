package handlers

import (
	"net/http"

	"wg-panel/internal/logging"
	"wg-panel/internal/services"

	"github.com/gin-gonic/gin"
)

type ServerHandler struct {
	service *services.ServerService
}

func NewServerHandler(service *services.ServerService) *ServerHandler {
	return &ServerHandler{
		service: service,
	}
}

func (h *ServerHandler) CreateServer(c *gin.Context) {
	ifId := c.Param("ifId")
	logging.LogVerbose("Creating server for interface: %s", ifId)

	var req services.ServerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logging.LogError("Failed to bind JSON for server creation: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.service.CreateServer(ifId, req)
	if err != nil {
		logging.LogError("Failed to create server %s for interface %s: %v", req.Name, ifId, err)
		if err.Error() == "interface not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "at least one of IPv4 or IPv6 must be enabled" {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "network overlaps with existing server network in VRF" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logging.LogInfo("Successfully created server %s (ID: %s) for interface %s", server.Name, server.ID, ifId)
	c.JSON(http.StatusCreated, server)
}

func (h *ServerHandler) GetServer(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	logging.LogVerbose("Getting server %s for interface %s", serverId, ifId)

	server, err := h.service.GetServer(ifId, serverId)
	if err != nil {
		logging.LogError("Server %s not found for interface %s: %v", serverId, ifId, err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
		return
	}

	logging.LogVerbose("Retrieved server %s (%s) for interface %s", server.Name, serverId, ifId)
	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) ListServers(c *gin.Context) {
	ifId := c.Param("ifId")

	servers, err := h.service.GetServers(ifId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Interface not found"})
		return
	}

	c.JSON(http.StatusOK, servers)
}

func (h *ServerHandler) UpdateServer(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	logging.LogInfo("Updating server %s for interface %s", serverId, ifId)

	var req services.ServerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logging.LogError("Failed to bind JSON for server update: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.service.UpdateServer(ifId, serverId, req)
	if err != nil {
		logging.LogError("Failed to update server %s for interface %s: %v", serverId, ifId, err)
		if err.Error() == "interface not found" || err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
			return
		}
		if err.Error() == "network overlaps with existing server network in VRF" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logging.LogInfo("Successfully updated server %s (%s) for interface %s", server.Name, serverId, ifId)
	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) DeleteServer(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	logging.LogInfo("Deleting server %s for interface %s", serverId, ifId)

	err := h.service.DeleteServer(ifId, serverId)
	if err != nil {
		logging.LogError("Failed to delete server %s for interface %s: %v", serverId, ifId, err)
		if err.Error() == "interface not found" || err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logging.LogInfo("Successfully deleted server %s for interface %s", serverId, ifId)
	c.Status(http.StatusNoContent)
}

func (h *ServerHandler) SetServerEnabled(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	logging.LogVerbose("Setting server %s enabled state for interface %s", serverId, ifId)

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logging.LogError("Failed to bind JSON for server enable/disable: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SetServerEnabled(ifId, serverId, req.Enabled, true)
	if err != nil {
		logging.LogError("Failed to set server %s enabled=%t for interface %s: %v", serverId, req.Enabled, ifId, err)
		if err.Error() == "interface not found" || err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	logging.LogInfo("Successfully set server %s enabled=%t for interface %s", serverId, req.Enabled, ifId)
	c.Status(http.StatusNoContent)
}

func (h *ServerHandler) MoveServer(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")

	var req struct {
		NewInterfaceId string `json:"newInterfaceId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.MoveServer(ifId, serverId, req.NewInterfaceId)
	if err != nil {
		if err.Error() == "source interface not found" ||
			err.Error() == "destination interface not found" ||
			err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Source or destination Interface, or Server not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ServerHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("", h.ListServers)
	router.POST("", h.CreateServer)
	router.GET("/:serverId", h.GetServer)
	router.PUT("/:serverId", h.UpdateServer)
	router.DELETE("/:serverId", h.DeleteServer)
	router.POST("/:serverId/set-enable", h.SetServerEnabled)
	router.POST("/:serverId/move", h.MoveServer)
}
