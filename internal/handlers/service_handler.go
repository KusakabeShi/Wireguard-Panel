package handlers

import (
	"fmt"
	"net/http"

	"wg-panel/internal/config"
	"wg-panel/internal/middleware"
	"wg-panel/internal/models"
	"wg-panel/internal/utils"

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
		"panelID":             h.cfg.WGPanelId,
		"wgIfPrefix":          h.cfg.WgIfPrefix,
		"WGPanelTitle":        h.cfg.WGPanelTitle,
	}

	c.JSON(http.StatusOK, response)
}

func (h *ServiceHandler) CheckSNATRoamingOffsetValid(c *gin.Context) {
	masterInterface := c.Query("ifname")
	netmapsrc_str := c.Query("netmapsrc")
	offsetstr := c.Query("offset")
	vrfstr := c.Query("vrf")
	afstr := c.Query("af")
	var baseipnet *models.IPNetWrapper
	var af int
	if masterInterface == "" || offsetstr == "" || afstr == "" {
		err_param := ""
		if masterInterface == "" {
			err_param = "ifname"
		} else if offsetstr == "" {
			err_param = "offset"
		} else if afstr == "" {
			err_param = "af"
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing required parameters", "error_params": err_param})
		return
	}

	ipv4, ipv6, err := utils.GetInterfaceIP(masterInterface)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get interface IP: " + err.Error(), "error_params": "ifname"})
		return
	}
	switch afstr {
	case "4":
		baseipnet = ipv4
		af = 4
	case "6":
		baseipnet = ipv6
		af = 6
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address family, must be 4 or 6", "error_params": "af"})
		return
	}
	if baseipnet == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("There is no IPv%v address found on interface %v", af, masterInterface), "error_params": "ifname"})
		return
	}
	ifVrf, err := utils.GetInterfaceVRF(&masterInterface)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to get interface VRF: " + err.Error(), "error_params": "ifname"})
		return
	}
	if vrfstr != ifVrf {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Interface VRF does not match, wireguard interface on \"%v\", %v on \"%v\"", vrfstr, masterInterface, ifVrf), "error_params": "ifname"})
		return
	}

	netmapsrc, err := models.ParseCIDRFromIPAf(af, netmapsrc_str)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse netmapsrc: " + err.Error(), "error_params": "ifname"})
		return
	}
	offset, err := models.ParseCIDRFromIPAf(af, offsetstr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset: " + err.Error(), "error_params": "offset"})
		return
	}
	if offset.IsSingleIP() {
		if offset.EqualZero(af) {
			c.JSON(http.StatusOK, gin.H{"type": "SNAT", "src network": netmapsrc.NetworkStr(), "mapped network": baseipnet.IP.String()})
		} else {
			zerostr := ""
			if af == 4 {
				zerostr = "0.0.0.0"
			} else if af == 6 {
				zerostr = "::"
			}
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid offset: must be " + zerostr + " for SNAT Mode", "error_params": "offset"})
		}
		return
	}
	mappedIPNet, err := baseipnet.GetSubnetByOffset(offset)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to calculate mapped IP: " + err.Error(), "error_params": "offset"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"type": "NETMAP", "src network": netmapsrc.NetworkStr(), "mapped network": mappedIPNet.String()})
}

func (h *ServiceHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/login", h.auth.Login)
	router.POST("/logout", h.auth.RequireAuth(), h.auth.Logout)
	router.GET("/config", h.auth.RequireAuth(), h.GetServiceConfig)
	router.GET("/snatroamingoffsetvalid", h.auth.RequireAuth(), h.CheckSNATRoamingOffsetValid)
	router.PUT("/password", h.auth.RequireAuth(), h.auth.ChangePassword)
}
