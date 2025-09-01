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

	"wg-panel/internal/config"
	"wg-panel/internal/internalservice"
	"wg-panel/internal/models"
	"wg-panel/internal/server"
	"wg-panel/internal/utils"

	"golang.org/x/crypto/bcrypt"
)

//go:embed all:frontend/build
var frontendFS embed.FS

func main() {
	var configPath = flag.String("c", "./config.json", "Path to configuration file")
	var newPassword = flag.String("p", "", "Set new password in configuration file")
	flag.Parse()

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

	// Initialize services
	firewallService := internalservice.NewFirewallService()
	pseudoBridgeService := internalservice.NewPseudoBridgeService()
	snatRoamingService := internalservice.NewSNATRoamingService(pseudoBridgeService, firewallService)
	cfg.LoadInternalServices(pseudoBridgeService, snatRoamingService)
	// Start HTTP server
	srv := server.NewServer(cfg, frontendFS)
	log.Printf("Starting WireGuard Panel on %s:%d", cfg.ListenIP, cfg.ListenPort)
	log.Fatal(srv.Start(firewallService))
}

func loadOrCreateConfig(configPath, newPassword string) (*config.Config, bool, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create new config with random password
		randomPassword, err := generateRandomPassword()
		if err != nil {
			return nil, false, fmt.Errorf("failed to generate random password: %v", err)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(randomPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, false, fmt.Errorf("failed to hash password: %v", err)
		}

		// Generate server ID
		serverId, err := utils.GenerateRandomString("", 6)
		if err != nil {
			return nil, false, fmt.Errorf("failed to generate server ID: %v", err)
		}

		cfg := &config.Config{
			WireGuardConfigPath: "/etc/wireguard",
			User:                "admin",
			Password:            string(hashedPassword),
			ListenIP:            "0.0.0.0",
			ListenPort:          5000,
			BasePath:            "/",
			SiteFrontendPath:    "./frontend/build",
			APIPrefix:           "/api",
			ServerId:            serverId,
			Interfaces:          make(map[string]*models.Interface),
			Sessions:            make(map[string]*config.Session),
		}

		if err := saveConfig(configPath, cfg); err != nil {
			return nil, false, fmt.Errorf("failed to save new config: %v", err)
		}

		fmt.Printf("Generated random password: %s\n", randomPassword)
		return cfg, true, nil
	}

	// Load existing config
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return nil, false, err
	}

	// Update password if requested
	if newPassword != "" {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
		if err != nil {
			return nil, false, fmt.Errorf("failed to hash new password: %v", err)
		}
		cfg.Password = string(hashedPassword)
		if err := saveConfig(configPath, cfg); err != nil {
			return nil, false, fmt.Errorf("failed to save updated config: %v", err)
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
