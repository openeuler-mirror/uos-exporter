package ratelimit

import (
	"fmt"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter implements rate limiting functionality
type RateLimiter struct {
	limiter *rate.Limiter
}

// NewRateLimiter creates a new rate limiter with the given interval and size
func NewRateLimiter(interval string, size int) (*RateLimiter, error) {
	d, err := time.ParseDuration(interval)
	if err != nil {
		return nil, fmt.Errorf("invalid interval: %v", err)
	}

	limit := rate.Every(d)
	l := &RateLimiter{
		limiter: rate.NewLimiter(limit, size),
	}
	return l, nil
}

// Allow checks if a request is allowed
func (l *RateLimiter) Allow() bool {
	return l.limiter.Allow()
}
