package main

import (
	"crypto/rand"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"wg-panel/internal/config"
	"wg-panel/internal/internalservice"
	"wg-panel/internal/logging"
	"wg-panel/internal/models"
	"wg-panel/internal/server"
	"wg-panel/internal/utils"
	"wg-panel/internal/version"

	"golang.org/x/crypto/bcrypt"
)

//go:embed all:frontend/build
var frontendFS embed.FS

func main() {
	var configPath = flag.String("c", "./config.json", "Path to configuration file")
	var newPassword = flag.String("p", "", "Set new password in configuration file")
	var showVersion = flag.Bool("v", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Println(version.GetVersionInfo())
		return
	}

	// Ensure absolute path
	if !filepath.IsAbs(*configPath) {
		absPath, err := filepath.Abs(*configPath)
		if err != nil {
			log.Fatalf("Failed to get absolute path for config: %v", err)
		}
		*configPath = absPath
	}

	cfg, isNewConfig, err := loadOrCreateConfig(*configPath, *newPassword)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	if isNewConfig {
		fmt.Printf("Created new configuration file with random password printed above\n")
	}

	if *newPassword != "" {
		fmt.Printf("Password updated successfully\n")
		return
	}

	// Initialize logger with configured log level
	logging.InitLogger(cfg.LogLevel)
	logging.LogInfo("Starting %s with log level: %s", version.GetVersionInfo(), cfg.LogLevel.String())

	// Perform system checks before starting services
	fnedmsg := config.ToFrontendMessage{}
	forward_accept := false
	if forward_accept, err = performSystemChecks(); err != nil {
		logging.LogError("System check failed: %v", err)
		fmt.Printf("Warning: %v\n", err)
		fnedmsg.InitWarningMsg = err.Error()
	}
	fnedmsg.Firewalldefault = !forward_accept

	// Initialize services
	firewallService := internalservice.NewFirewallService()
	pseudoBridgeService := internalservice.NewPseudoBridgeService()
	snatRoamingService := internalservice.NewSNATRoamingService(pseudoBridgeService, firewallService)
	cfg.LoadInternalServices(pseudoBridgeService, snatRoamingService, fnedmsg)
	// Start HTTP server
	srv := server.NewServer(cfg, frontendFS)
	logging.LogInfo("Starting WireGuard Panel on %s:%d", cfg.ListenIP, cfg.ListenPort)
	log.Fatal(srv.Start(firewallService, cfg.LogLevel))
}

func loadOrCreateConfig(configPath, newPassword string) (*config.Config, bool, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create new config with random password
		randomPassword, err := generateRandomPassword()
		if err != nil {
			return nil, false, fmt.Errorf("failed to generate random password:-> %v", err)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, false, fmt.Errorf("failed to hash password:-> %v", err)
		}

		// Generate server ID
		serverId, err := utils.GenerateRandomString("", 6)
		if err != nil {
			return nil, false, fmt.Errorf("failed to generate server ID:-> %v", err)
		}

		cfg := &config.Config{
			WireGuardConfigPath: "/etc/wireguard",
			WgIfPrefix:          "wg-",
			LogLevel:            logging.LogLevelInfo,
			User:                "admin",
			Password:            string(hashedPassword),
			ListenIP:            "0.0.0.0",
			ListenPort:          5000,
			BasePath:            "/",
			APIPrefix:           "/api",
			WGPanelId:           serverId,
			Interfaces:          make(map[string]*models.Interface),
			Sessions:            make(map[string]*config.Session),
		}

		if err := saveConfig(configPath, cfg); err != nil {
			return nil, false, fmt.Errorf("failed to save new config:-> %v", err)
		}

		fmt.Printf("Generated random password: %s\n", randomPassword)
		return cfg, true, nil
	}

	// Load existing config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, false, err
	}

	if cfg.BasePath == "" {
		cfg.BasePath = "/"
	}
	if cfg.APIPrefix == "" {
		cfg.APIPrefix = "/"
	}

	// Update password if requested
	if newPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, false, fmt.Errorf("failed to hash new password:-> %v", err)
		}
		cfg.Password = string(hashedPassword)
		if err := saveConfig(configPath, cfg); err != nil {
			return nil, false, fmt.Errorf("failed to save updated config:-> %v", err)
		}
	}

	cfg.ConfigPath = configPath
	return cfg, false, nil
}

func generateRandomPassword() (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}
	return string(b), nil
}

func saveConfig(configPath string, cfg *config.Config) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return utils.WriteFileAtomic(configPath, data, 0600)
}

// performSystemChecks validates system configuration and required tools
func performSystemChecks() (forward_accept bool, err error) {
	var warnings []string

	// Check IP forwarding settings
	if err := checkIPForwarding(&warnings); err != nil {
		warnings = append(warnings, err.Error())
	}

	// Check required tools installation
	if err := checkRequiredTools(&warnings); err != nil {
		warnings = append(warnings, err.Error())
	}

	// Check iptables FORWARD policies
	if forward_accept, err = checkFirewallPolicies(&warnings); err != nil {
		warnings = append(warnings, err.Error())
	}

	if len(warnings) > 0 {
		return forward_accept, fmt.Errorf("system configuration issues found:\n  - %s", strings.Join(warnings, "\n  - "))
	}

	return forward_accept, nil
}

// checkIPForwarding verifies IPv4 and IPv6 forwarding is enabled
func checkIPForwarding(warnings *[]string) error {
	// Check IPv4 forwarding
	output, err := utils.RunCommandWithOutput("cat", "/proc/sys/net/ipv4/ip_forward")
	if err != nil {
		*warnings = append(*warnings, "unable to check IPv4 forwarding status")
	} else if strings.TrimSpace(output) != "1" {
		*warnings = append(*warnings, "IPv4 forwarding is disabled. Enable with: sysctl -w net.ipv4.ip_forward=1")
	}

	// Check IPv6 forwarding
	output, err = utils.RunCommandWithOutput("cat", "/proc/sys/net/ipv6/conf/all/forwarding")
	if err != nil {
		*warnings = append(*warnings, "unable to check IPv6 forwarding status")
	} else if strings.TrimSpace(output) != "1" {
		*warnings = append(*warnings, "IPv6 forwarding is disabled. Enable with: sysctl -w net.ipv6.conf.all.forwarding=1")
	}

	return nil
}

// checkRequiredTools verifies all required system tools are installed
func checkRequiredTools(warnings *[]string) error {
	requiredTools := []string{"ip", "wg", "wg-quick", "iptables", "ip6tables", "iptables-save", "ip6tables-save"}

	for _, tool := range requiredTools {
		if err := utils.RunCommand("which", tool); err != nil {
			switch tool {
			case "ip":
				*warnings = append(*warnings, fmt.Sprintf("%s not found. Install with: apt-get install iproute2 (Ubuntu/Debian) or yum install iproute (RHEL/CentOS)", tool))
			case "wg", "wg-quick":
				*warnings = append(*warnings, fmt.Sprintf("%s not found. Install WireGuard tools with: apt-get install wireguard-tools (Ubuntu/Debian) or yum install wireguard-tools (RHEL/CentOS)", tool))
			case "iptables", "ip6tables", "iptables-save", "ip6tables-save":
				*warnings = append(*warnings, fmt.Sprintf("%s not found. Install with: apt-get install iptables (Ubuntu/Debian) or yum install iptables (RHEL/CentOS)", tool))
			default:
				*warnings = append(*warnings, fmt.Sprintf("%s not found in PATH", tool))
			}
		}
	}

	return nil
}

// checkFirewallPolicies verifies iptables FORWARD chain policies
func checkFirewallPolicies(warnings *[]string) (forward_accept bool, err error) {
	forward_accept = true
	// Check IPv4 iptables FORWARD policy
	output, err := utils.RunCommandWithOutput("iptables", "-L", "FORWARD", "-n")
	if err != nil {
		forward_accept = false
	} else if len(output) > 0 && !strings.Contains(strings.Split(output, "\n")[0], "policy ACCEPT") {
		forward_accept = false
	}

	// Check IPv6 ip6tables FORWARD policy
	output, err = utils.RunCommandWithOutput("ip6tables", "-L", "FORWARD", "-n")
	if err != nil {
		forward_accept = false
	} else if len(output) > 0 && !strings.Contains(strings.Split(output, "\n")[0], "policy ACCEPT") {
		forward_accept = false
	}

	return forward_accept, nil
}
