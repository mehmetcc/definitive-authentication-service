package authentication

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/mehmetcc/definitive-authentication-service/internal/person"
	"github.com/mehmetcc/definitive-authentication-service/internal/utils"
)

func AuthMiddleware(personService person.PersonService, accessSecret string, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		parts := strings.Fields(authHeader)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header format must be Bearer <token>"})
			return
		}
		rawToken := parts[1]

		// Parse and validate access JWT
		claims, err := utils.ParseAccessToken(rawToken, accessSecret)
		if err != nil {
			logger.Warn("access token parse failed", zap.Error(err))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired access token"})
			return
		}

		// Extract subject as user ID
		userID, err := strconv.ParseUint(claims.Subject, 10, 64)
		if err != nil {
			logger.Error("invalid subject claim", zap.Error(err), zap.String("subject", claims.Subject))
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token subject"})
			return
		}

		// Load Person
		user, err := personService.ReadPersonByID(c.Request.Context(), uint(userID))
		if err != nil {
			if errors.Is(err, person.ErrPersonNotFound) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "user not found"})
				return
			}
			logger.Error("failed to load person by ID", zap.Error(err), zap.Uint("userID", uint(userID)))
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not validate user"})
			return
		}

		// Set person into context and proceed
		c.Set(person.ContextUserKey, user)
		c.Next()
	}
}

func RoleMiddleware(requiredRole person.Role, logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw, exists := c.Get(person.ContextUserKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		user := raw.(*person.Person)
		if user.Role != requiredRole {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		c.Next()
	}
}
