package httpx

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/wilddog64/shopping-cart-order/internal/auth"
	"go.uber.org/zap"
)

const customerIDKey contextKey = "customerID"

func SetCustomerID(c *gin.Context, id string) { c.Set(string(customerIDKey), id) }

func GetCustomerID(c *gin.Context) string { return c.GetString(string(customerIDKey)) }

func AuthMiddleware(validator *auth.JWTValidator, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Authorization header required"})
			return
		}
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Invalid authorization header format"})
			return
		}
		claims, err := validator.ValidateToken(c.Request.Context(), parts[1])
		if err != nil {
			logger.Warn("token validation failed", zap.Error(err), zap.String("ip", clientIP(c.Request)))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"})
			return
		}
		SetCustomerID(c, claims.Subject)
		c.Set("claims", claims)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}

func MockAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID := c.GetHeader("X-User-ID")
		if customerID == "" {
			customerID = "dev-user"
		}
		SetCustomerID(c, customerID)
		c.Set("roles", []string{"order-user"})
		c.Next()
	}
}
