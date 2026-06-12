package middleware

import (
	"net/http"
	"strings"
	"time"

	"begmt2/config"
	"begmt2/models"
	"begmt2/utils"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func AuthMiddleware(cfg config.Config, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "authorization header is required"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "authorization header must use Bearer token"})
			return
		}

		claims, err := utils.ParseJWT(parts[1], cfg.JWTSecret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid or expired token"})
			return
		}
		if claims.SessionID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "invalid session"})
			return
		}

		var session models.AuthSession
		err = db.Where(
			"session_id = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?",
			claims.SessionID,
			claims.UserID,
			time.Now(),
		).First(&session).Error
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "session expired or revoked"})
			return
		}

		var user models.User
		if err := db.First(&user, claims.UserID).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "user not found"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", user.Email)
		c.Set("role", string(user.Role))
		c.Set("session_id", claims.SessionID)
		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "role is missing"})
			return
		}

		currentRole, ok := role.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "invalid role"})
			return
		}

		for _, allowedRole := range allowedRoles {
			if currentRole == allowedRole {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "you do not have access to this resource"})
	}
}

func AgentStatusMiddleware(db *gorm.DB, allowedStatuses ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetUint("user_id")
		if userID == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "user is missing"})
			return
		}

		var detail models.DetailUser
		if err := db.Where("user_id = ?", userID).First(&detail).Error; err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "agent application status is missing"})
			return
		}
		if detail.Status == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "agent application status is missing"})
			return
		}

		for _, allowedStatus := range allowedStatuses {
			if *detail.Status == allowedStatus {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": "agent status is not allowed for this resource"})
	}
}
