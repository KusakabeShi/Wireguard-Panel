package handlers

import (
	"net/http"

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

	var req services.ServerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.service.CreateServer(ifId, req)
	if err != nil {
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

	c.JSON(http.StatusCreated, server)
}

func (h *ServerHandler) GetServer(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")

	server, err := h.service.GetServer(ifId, serverId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
		return
	}

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

	var req services.ServerCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	server, err := h.service.UpdateServer(ifId, serverId, req)
	if err != nil {
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

	c.JSON(http.StatusOK, server)
}

func (h *ServerHandler) DeleteServer(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")

	err := h.service.DeleteServer(ifId, serverId)
	if err != nil {
		if err.Error() == "interface not found" || err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ServerHandler) SetServerEnabled(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SetServerEnabled(ifId, serverId, req.Enabled)
	if err != nil {
		if err.Error() == "interface not found" || err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

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
