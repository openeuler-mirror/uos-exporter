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

// TODO: implement functions
