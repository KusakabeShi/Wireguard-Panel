package handlers

import (
	"net/http"

	"wg-panel/internal/services"

	"github.com/gin-gonic/gin"
)

type ClientHandler struct {
	service *services.ClientService
}

func NewClientHandler(service *services.ClientService) *ClientHandler {
	return &ClientHandler{
		service: service,
	}
}

func (h *ClientHandler) CreateClient(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")

	var req services.ClientCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.service.CreateClient(ifId, serverId, req)
	if err != nil {
		if err.Error() == "interface not found" || err.Error() == "server not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
			return
		}
		if err.Error() == "at least one of IPv4 or IPv6 must be specified" ||
			err.Error() == "no available IPv4 addresses in network" ||
			err.Error() == "no available IPv6 addresses in network" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client_frontend, _ := h.service.ToClientFrontend(ifId, serverId, client)
	c.JSON(http.StatusCreated, client_frontend)
}

func (h *ClientHandler) GetClient(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	clientId := c.Param("clientId")

	client, err := h.service.GetClient(ifId, serverId, clientId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client, Server, or Interface not found"})
		return
	}
	client_frontend, _ := h.service.ToClientFrontend(ifId, serverId, client)
	c.JSON(http.StatusOK, client_frontend)
}

func (h *ClientHandler) GetServerClients(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")

	clients, err := h.service.GetClientsFrontendWithState(ifId, serverId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
		return
	}

	c.JSON(http.StatusOK, clients)
}

func (h *ClientHandler) UpdateClient(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	clientId := c.Param("clientId")

	var req services.ClientUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	client, err := h.service.UpdateClient(ifId, serverId, clientId, req)
	if err != nil {
		if err.Error() == "interface not found" ||
			err.Error() == "server not found" ||
			err.Error() == "client not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client, Server, or Interface not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	client_frontend, _ := h.service.ToClientFrontend(ifId, serverId, client)
	c.JSON(http.StatusOK, client_frontend)
}

func (h *ClientHandler) DeleteClient(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	clientId := c.Param("clientId")

	err := h.service.DeleteClient(ifId, serverId, clientId)
	if err != nil {
		if err.Error() == "interface not found" ||
			err.Error() == "server not found" ||
			err.Error() == "client not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client, Server, or Interface not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ClientHandler) SetClientEnabled(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	clientId := c.Param("clientId")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SetClientEnabled(ifId, serverId, clientId, req.Enabled)
	if err != nil {
		if err.Error() == "interface not found" ||
			err.Error() == "server not found" ||
			err.Error() == "client not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Client, Server, or Interface not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *ClientHandler) GetClientWGState(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	clientId := c.Param("clientId")

	state, err := h.service.GetClientWGState(ifId, serverId, clientId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client, Server, or Interface not found"})
		return
	}

	c.JSON(http.StatusOK, state)
}

func (h *ClientHandler) GetClientConfig(c *gin.Context) {
	ifId := c.Param("ifId")
	serverId := c.Param("serverId")
	clientId := c.Param("clientId")

	config, err := h.service.GetClientConfig(ifId, serverId, clientId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Client, Server, or Interface not found"})
		return
	}

	c.Data(http.StatusOK, "text/plain", []byte(config))
}

func (h *ClientHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/clients", h.GetServerClients)
	router.POST("/clients", h.CreateClient)
	router.GET("/clients/:clientId", h.GetClient)
	router.PUT("/clients/:clientId", h.UpdateClient)
	router.DELETE("/clients/:clientId", h.DeleteClient)
	router.POST("/clients/:clientId/set-enable", h.SetClientEnabled)
	router.GET("/clients/:clientId/state", h.GetClientWGState)
	router.GET("/clients/:clientId/config", h.GetClientConfig)
}
