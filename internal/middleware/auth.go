package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"wg-panel/internal/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type AuthMiddleware struct {
	cfg *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{cfg: cfg}
}

func (a *AuthMiddleware) RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		cookie, err := c.Cookie("session_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authentication required"})
			c.Abort()
			return
		}

		session := a.cfg.GetSession(cookie)
		if session == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid session"})
			c.Abort()
			return
		}

		// Check if session is expired (24 hours)
		if time.Since(session.LastSeen) > 24*time.Hour {
			a.cfg.DeleteSession(cookie)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired"})
			c.Abort()
			return
		}

		// Update last seen
		session.LastSeen = time.Now()

		c.Set("username", session.Username)
		c.Next()
	}
}

func (a *AuthMiddleware) Login(c *gin.Context) {
	var loginReq struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check credentials
	if loginReq.Username != a.cfg.User {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	err := bcrypt.CompareHashAndPassword([]byte(a.cfg.Password), []byte(loginReq.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Generate session token
	token, err := generateSessionToken()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate session"})
		return
	}

	// Create session
	session := &config.Session{
		Username:  loginReq.Username,
		CreatedAt: time.Now(),
		LastSeen:  time.Now(),
	}

	a.cfg.AddSession(token, session)
	a.cfg.CleanExpiredSessions()

	// Set cookie
	c.SetCookie("session_token", token, 24*3600, "/", "", false, true)
	c.Status(http.StatusOK)
}

func (a *AuthMiddleware) Logout(c *gin.Context) {
	cookie, err := c.Cookie("session_token")
	if err == nil {
		a.cfg.DeleteSession(cookie)
	}

	c.SetCookie("session_token", "", -1, "/", "", false, true)
	c.Status(http.StatusNoContent)
}

func (a *AuthMiddleware) ChangePassword(c *gin.Context) {
	var passwordReq struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		Password        string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&passwordReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify current password
	err := bcrypt.CompareHashAndPassword([]byte(a.cfg.Password), []byte(passwordReq.CurrentPassword))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Current password is incorrect"})
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordReq.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to hash password"})
		return
	}

	a.cfg.Password = string(hashedPassword)
	if err := a.cfg.Save(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save configuration"})
		return
	}

	c.Status(http.StatusNoContent)
}

func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
