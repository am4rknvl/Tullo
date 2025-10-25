package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	limiters map[uuid.UUID]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
}

func NewRateLimiter(rps int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[uuid.UUID]*rate.Limiter),
		rate:     rate.Limit(rps),
		burst:    rps * 2,
	}
}

func (rl *RateLimiter) getLimiter(userID uuid.UUID) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[userID]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.limiters[userID] = limiter
	}

	return limiter
}

// Cleanup removes old limiters
func (rl *RateLimiter) Cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	go func() {
		for range ticker.C {
			rl.mu.Lock()
			// Simple cleanup - in production, track last access time
			if len(rl.limiters) > 10000 {
				rl.limiters = make(map[uuid.UUID]*rate.Limiter)
			}
			rl.mu.Unlock()
		}
	}()
}

// RateLimitMiddleware limits requests per user
func RateLimitMiddleware(rl *RateLimiter) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		uid, ok := userID.(uuid.UUID)
		if !ok {
			c.Next()
			return
		}

		limiter := rl.getLimiter(uid)
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Rate limit exceeded"})
			c.Abort()
			return
		}

		c.Next()
	}
}
