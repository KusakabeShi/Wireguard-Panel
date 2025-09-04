package handlers

import (
	"net/http"

	"wg-panel/internal/services"

	"github.com/gin-gonic/gin"
)

type InterfaceHandler struct {
	service *services.InterfaceService
}

func NewInterfaceHandler(service *services.InterfaceService) *InterfaceHandler {
	return &InterfaceHandler{
		service: service,
	}
}

func (h *InterfaceHandler) CreateInterface(c *gin.Context) {
	var req services.InterfaceCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	iface, err := h.service.CreateInterface(req)
	if err != nil {
		if err.Error() == "interface with ifname '"+req.Ifname+"' already exists" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, iface)
}

func (h *InterfaceHandler) GetInterface(c *gin.Context) {
	ifId := c.Param("ifId")

	iface, err := h.service.GetInterface(ifId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Interface not found"})
		return
	}

	c.JSON(http.StatusOK, iface)
}

func (h *InterfaceHandler) ListInterfaces(c *gin.Context) {
	interfaces := h.service.GetAllInterfaces()
	c.JSON(http.StatusOK, interfaces)
}

func (h *InterfaceHandler) UpdateInterface(c *gin.Context) {
	ifId := c.Param("ifId")

	var req services.InterfaceUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	iface, err := h.service.UpdateInterface(ifId, req)
	if err != nil {
		if err.Error() == "interface not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		if err.Error() == "network overlap detected in VRF" {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, iface)
}

func (h *InterfaceHandler) SetInterfaceEnabled(c *gin.Context) {
	ifId := c.Param("ifId")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.service.SetInterfaceEnabled(ifId, req.Enabled)
	if err != nil {
		if err.Error() == "interface not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Server or Interface not found"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *InterfaceHandler) DeleteInterface(c *gin.Context) {
	ifId := c.Param("ifId")

	err := h.service.DeleteInterface(ifId)
	if err != nil {
		if err.Error() == "interface not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *InterfaceHandler) GetInterfaceClientsState(c *gin.Context) {
	ifId := c.Param("ifId")

	clientsState, milliseconds, err := h.service.GetInterfaceClientsState(ifId)
	if err != nil {
		if err.Error() == "interface not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"state": clientsState, "timestamp": milliseconds})
}

func (h *InterfaceHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("", h.ListInterfaces)
	router.POST("", h.CreateInterface)
	router.GET("/:ifId", h.GetInterface)
	router.PUT("/:ifId", h.UpdateInterface)
	router.POST("/:ifId/set-enable", h.SetInterfaceEnabled)
	router.GET("/:ifId/clients-state", h.GetInterfaceClientsState)
	router.DELETE("/:ifId", h.DeleteInterface)
}
