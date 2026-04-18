package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web/context"
	"golang.org/x/time/rate"
)

type RateLimiter struct {
	visitors map[string]*rate.Limiter
	mu       sync.RWMutex
	rate     rate.Limit
	burst    int
	cleanup  time.Duration
}

func NewRateLimiter(r rate.Limit, b int, cleanup time.Duration) *RateLimiter {
	rl := &RateLimiter{
		visitors: make(map[string]*rate.Limiter),
		rate:     r,
		burst:    b,
		cleanup:  cleanup,
	}

	go rl.cleanupVisitors()
	return rl
}

func (rl *RateLimiter) GetVisitor(ip string) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.visitors[ip]
	if !exists {
		limiter = rate.NewLimiter(rl.rate, rl.burst)
		rl.visitors[ip] = limiter
	}

	return limiter
}

func (rl *RateLimiter) cleanupVisitors() {
	for {
		time.Sleep(rl.cleanup)
		rl.mu.Lock()
		for ip, limiter := range rl.visitors {
			if limiter.Allow() {
				delete(rl.visitors, ip)
			}
		}
		rl.mu.Unlock()
	}
}

var rateLimiter = NewRateLimiter(rate.Limit(10), 20, 10*time.Minute)

func RateLimitMiddleware(ctx *context.Context) {
	ip := getClientIP(ctx)
	limiter := rateLimiter.GetVisitor(ip)

	if !limiter.Allow() {
		ctx.Output.SetStatus(http.StatusTooManyRequests)
		ctx.Output.JSON(map[string]interface{}{
			"error":   "Rate limit exceeded",
			"code":    http.StatusTooManyRequests,
			"message": "Too many requests. Please try again later.",
		}, false, false)
		return
	}
}

func getClientIP(ctx *context.Context) string {
	forwarded := ctx.Input.Header("X-Forwarded-For")
	if forwarded != "" {
		return forwarded
	}

	realIP := ctx.Input.Header("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	return ctx.Input.IP()
}
