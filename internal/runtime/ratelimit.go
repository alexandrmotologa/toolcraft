package runtime

import (
	"sync"
	"time"
)

// RateLimiter enforces a token-bucket rate limit.
type RateLimiter struct {
	mu         sync.Mutex
	tokens     float64
	maxTokens  float64
	fillRate   float64 // tokens per second
	lastRefill time.Time
}

// NewRateLimiter creates a rate limiter with the specified requests per minute.
func NewRateLimiter(rpm int) *RateLimiter {
	if rpm <= 0 {
		rpm = 120
	}
	rate := float64(rpm) / 60.0
	return &RateLimiter{
		tokens:     float64(rpm),
		maxTokens:  float64(rpm),
		fillRate:   rate,
		lastRefill: time.Now(),
	}
}

// Allow checks if a request is permitted under the rate limit.
func (r *RateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill).Seconds()
	r.tokens += elapsed * r.fillRate
	if r.tokens > r.maxTokens {
		r.tokens = r.maxTokens
	}
	r.lastRefill = now

	if r.tokens >= 1.0 {
		r.tokens -= 1.0
		return true
	}
	return false
}
