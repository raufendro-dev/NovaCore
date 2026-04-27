package middleware

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	api "github.com/raufendro/novacore/internal/http"
	"github.com/raufendro/novacore/pkg/security"
	"go.uber.org/zap"
)

func Logger(log *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("request_id", c.GetString("request_id")),
		)
	}
}

func Recovery(log *zap.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		log.Error("panic recovered", zap.Any("error", recovered))
		api.Error(c, http.StatusInternalServerError, "Internal server error", nil)
	})
}

func CORS(origins []string) gin.HandlerFunc {
	cfg := cors.DefaultConfig()
	cfg.AllowOrigins = origins
	cfg.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}
	cfg.AllowHeaders = []string{"Authorization", "Content-Type", "X-Request-ID"}
	cfg.ExposeHeaders = []string{"X-Request-ID"}
	if len(origins) == 1 && origins[0] == "*" {
		cfg.AllowAllOrigins = true
		cfg.AllowOrigins = nil
	}
	return cors.New(cfg)
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = time.Now().UTC().Format("20060102150405.000000000")
		}
		c.Set("request_id", id)
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("Referrer-Policy", "no-referrer")
		c.Next()
	}
}

func Timeout(duration time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		if duration <= 0 {
			c.Next()
			return
		}
		timeout := time.AfterFunc(duration, func() {
			c.AbortWithStatusJSON(http.StatusGatewayTimeout, api.Envelope{Success: false, Message: "Request timeout"})
		})
		defer timeout.Stop()
		c.Next()
	}
}

func Auth(jwt *security.JWTManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if !strings.HasPrefix(header, "Bearer ") {
			api.Error(c, http.StatusUnauthorized, "Missing bearer token", nil)
			c.Abort()
			return
		}
		claims, err := jwt.Validate(strings.TrimPrefix(header, "Bearer "))
		if err != nil || claims.Type != "access" {
			api.Error(c, http.StatusUnauthorized, "Invalid token", nil)
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}

func Role(required ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{}
	for _, role := range required {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		rolesAny, _ := c.Get("roles")
		roles, _ := rolesAny.([]string)
		for _, role := range roles {
			if _, ok := allowed[role]; ok {
				c.Next()
				return
			}
		}
		api.Error(c, http.StatusForbidden, "Forbidden", nil)
		c.Abort()
	}
}

func RateLimiter(max int, window time.Duration) gin.HandlerFunc {
	type bucket struct {
		count int
		reset time.Time
	}
	var mu sync.Mutex
	buckets := map[string]*bucket{}
	if max <= 0 {
		max = 100
	}
	if window <= 0 {
		window = time.Minute
	}
	return func(c *gin.Context) {
		key := c.ClientIP()
		now := time.Now()
		mu.Lock()
		b, ok := buckets[key]
		if !ok || now.After(b.reset) {
			b = &bucket{reset: now.Add(window)}
			buckets[key] = b
		}
		b.count++
		blocked := b.count > max
		mu.Unlock()
		if blocked {
			api.Error(c, http.StatusTooManyRequests, "Too many requests", nil)
			c.Abort()
			return
		}
		c.Next()
	}
}
