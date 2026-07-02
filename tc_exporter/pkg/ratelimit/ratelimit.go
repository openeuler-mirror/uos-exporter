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

func NewRateLimiter(limit time.Duration, chanSize int) (*RateLimiter, error) {
	if chanSize <= 0 {
		return nil, ErrRateLimitSize
	}
	if limit <= 0 {
		return nil, ErrRateLimitTime
	}
	rl := &RateLimiter{
		tokens: make(chan struct{}, chanSize),
		limit:  limit,
		ticker: time.NewTicker(limit),
	}

	for i := 0; i < chanSize; i++ {
		rl.tokens <- struct{}{}
	}

	go rl.startRefreshTokens()
	return rl, nil
}

func (rl *RateLimiter) startRefreshTokens() {
	for range rl.ticker.C {
		select {
		case rl.tokens <- struct{}{}:
		default:
		}
	}
}

func (rl *RateLimiter) Get() error {
	select {
	case _, ok := <-rl.tokens:
		if ok {
			return nil
		} else {
			return ErrRateLimited
		}
	default:
		return ErrRateLimited
	}
}

func (rl *RateLimiter) Stop() {
	rl.ticker.Stop()
	close(rl.tokens)
	clearChannel(rl.tokens)

}

func clearChannel(ch chan struct{}) {
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return
			}
		default:
			return
		}
	}
}
// Part 2 commit for tc_exporter/pkg/ratelimit/ratelimit.go
