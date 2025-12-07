package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type RateLimiter struct {
	clients map[string]*ClientLimiter
	mu      sync.RWMutex
	logger  *logrus.Logger
	rate    int           // requests per minute
	burst   int           // burst size
	window  time.Duration // time window
}

type ClientLimiter struct {
	tokens    int
	lastRefill time.Time
	mu        sync.Mutex
}

func NewRateLimiter(rate int, burst int, logger *logrus.Logger) *RateLimiter {
	return &RateLimiter{
		clients: make(map[string]*ClientLimiter),
		logger:  logger,
		rate:    rate,
		burst:   burst,
		window:  time.Minute,
	}
}

func (rl *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := getClientIP(r)
		
		if !rl.allow(clientIP) {
			rl.logger.WithField("ip", clientIP).Warn("Rate limit exceeded")
			w.Header().Set("Retry-After", "60")
			http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) allow(clientIP string) bool {
	rl.mu.RLock()
	limiter, exists := rl.clients[clientIP]
	rl.mu.RUnlock()
	
	if !exists {
		rl.mu.Lock()
		// Double-check after acquiring write lock
		limiter, exists = rl.clients[clientIP]
		if !exists {
			limiter = &ClientLimiter{
				tokens:     rl.burst,
				lastRefill: time.Now(),
			}
			rl.clients[clientIP] = limiter
		}
		rl.mu.Unlock()
	}
	
	limiter.mu.Lock()
	defer limiter.mu.Unlock()
	
	// Refill tokens based on time elapsed
	now := time.Now()
	elapsed := now.Sub(limiter.lastRefill)
	tokensToAdd := int(elapsed.Seconds() * float64(rl.rate) / rl.window.Seconds())
	
	if tokensToAdd > 0 {
		limiter.tokens += tokensToAdd
		if limiter.tokens > rl.burst {
			limiter.tokens = rl.burst
		}
		limiter.lastRefill = now
	}
	
	// Check if we have tokens available
	if limiter.tokens > 0 {
		limiter.tokens--
		return true
	}
	
	return false
}

func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP in the list
		if idx := len(xff); idx > 0 {
			if commaIdx := 0; commaIdx < idx {
				for i, c := range xff {
					if c == ',' {
						commaIdx = i
						break
					}
				}
				if commaIdx > 0 {
					return xff[:commaIdx]
				}
			}
			return xff
		}
	}
	
	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// Fall back to RemoteAddr
	return r.RemoteAddr
}
