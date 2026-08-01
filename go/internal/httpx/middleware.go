package httpx

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

type contextKey string

const correlationIDKey contextKey = "correlationID"

func CorrelationID() gin.HandlerFunc {
	return func(c *gin.Context) {
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.NewString()
		}
		c.Set(string(correlationIDKey), correlationID)
		c.Header("X-Correlation-ID", correlationID)
		c.Next()
	}
}

func GetCorrelationID(c *gin.Context) string { return c.GetString(string(correlationIDKey)) }

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none'; form-action 'self'")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		if c.Request.TLS != nil {
			c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		c.Next()
	}
}

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", time.Since(start)),
			zap.String("ip", clientIP(c.Request)),
		}
		if query := c.Request.URL.RawQuery; query != "" {
			fields = append(fields, zap.String("query", query))
		}
		if correlationID := c.GetString(string(correlationIDKey)); correlationID != "" {
			fields = append(fields, zap.String("correlationId", correlationID))
		}

		switch {
		case c.Writer.Status() >= http.StatusInternalServerError:
			logger.Error("request completed", fields...)
		case c.Writer.Status() >= http.StatusBadRequest:
			logger.Warn("request completed", fields...)
		default:
			logger.Info("request completed", fields...)
		}
	}
}

type RateLimiter struct {
	mu       sync.Mutex
	visitors map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
	stopCh   chan struct{}
	ttl      time.Duration
}

func NewRateLimiter(requestsPerSecond, burst int) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
		stopCh:   make(chan struct{}),
		ttl:      time.Hour,
	}

	go rl.cleanupLoop()
	return rl
}

func (r *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/actuator/") {
			c.Next()
			return
		}

		limiter := r.limiterFor(clientIP(c.Request))
		reservation := limiter.Reserve()
		if !reservation.OK() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Too many requests. Please retry later.",
			})
			return
		}
		if delay := reservation.Delay(); delay > 0 {
			reservation.Cancel()
			seconds := int(delay.Seconds())
			if seconds < 1 {
				seconds = 1
			}
			c.Header("Retry-After", strconv.Itoa(seconds))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"code":    "RATE_LIMIT_EXCEEDED",
				"message": "Too many requests. Please retry later.",
			})
			return
		}

		c.Next()
	}
}

func (r *RateLimiter) limiterFor(ip string) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limiter, ok := r.visitors[ip]; ok {
		return limiter
	}

	limiter := rate.NewLimiter(r.rate, r.burst)
	r.visitors[ip] = limiter
	return limiter
}

func (r *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			r.mu.Lock()
			if len(r.visitors) > 0 {
				r.visitors = make(map[string]*rate.Limiter)
			}
			r.mu.Unlock()
		case <-r.stopCh:
			return
		}
	}
}

func clientIP(request *http.Request) string {
	if xff := request.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if ip := strings.TrimSpace(parts[0]); ip != "" {
				return ip
			}
		}
	}
	if xri := request.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(request.RemoteAddr)
	if err == nil && host != "" {
		return host
	}
	return request.RemoteAddr
}
