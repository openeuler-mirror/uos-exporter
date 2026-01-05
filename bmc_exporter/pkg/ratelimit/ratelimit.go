package ratelimit

import (
	"errors"
	"time"
)

var (
	ErrRateLimited   = errors.New("rate limited")
	ErrRateLimitSize = errors.New("limit must be greater than zero")
	ErrRateLimitTime = errors.New("invalid limit")
)

type RateLimiter struct {
	tokens chan struct{}
	limit  time.Duration
	ticker *time.Ticker
}


// TODO: implement functions
