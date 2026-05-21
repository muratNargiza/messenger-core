package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/gliedabrennung/messenger-core/internal/common/api"
)

type visitor struct {
	tokens   float64
	lastSeen time.Time
}

type RateLimiter struct {
	mu        sync.Mutex
	visitors  map[string]*visitor
	rate      float64
	burst     int
	lastClean time.Time
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{
		visitors:  make(map[string]*visitor),
		rate:      rps,
		burst:     burst,
		lastClean: time.Now(),
	}
}

func (rl *RateLimiter) Handler() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		ip := c.ClientIP()
		now := time.Now()

		rl.mu.Lock()

		if now.Sub(rl.lastClean) > time.Minute {
			for k, v := range rl.visitors {
				if now.Sub(v.lastSeen) > 3*time.Minute {
					delete(rl.visitors, k)
				}
			}
			rl.lastClean = now
		}

		v, exists := rl.visitors[ip]
		if !exists {
			v = &visitor{tokens: float64(rl.burst), lastSeen: now}
			rl.visitors[ip] = v
		}

		elapsed := now.Sub(v.lastSeen).Seconds()
		v.lastSeen = now
		v.tokens += elapsed * rl.rate
		if v.tokens > float64(rl.burst) {
			v.tokens = float64(rl.burst)
		}

		if v.tokens < 1 {
			rl.mu.Unlock()
			api.ErrorResponse(c, http.StatusTooManyRequests,
				"RATE_LIMITED", "too many requests", nil)
			c.Abort()
			return
		}

		v.tokens--
		rl.mu.Unlock()
		c.Next(ctx)
	}
}
