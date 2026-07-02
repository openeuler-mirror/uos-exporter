package server

import (
	"node_system_exporter/pkg/ratelimit"
	"github.com/alecthomas/kingpin"
	"time"
)

var (
	rateLimitInterval *time.Duration
	rateLimitSize     *int
	UseRatelimit      *bool
)

func init() {
	rateLimitInterval = kingpin.Flag("rate_limit_interval",
		"rate limit interval").Default("1s").Duration()
	rateLimitSize = kingpin.Flag("rate_limit_size",
		"rate limit size").Default("100").Int()
	UseRatelimit = kingpin.Flag("use_ratelimit",
		"use rate limit").Bool()
}

func Ratelimit(ratelimiter *ratelimit.RateLimiter) HandlerFunc {
	return func(req *Request) {
		if err := ratelimiter.Get(); err != nil {
			req.Error = err
			req.Fail(429)
		}
	}
}
// Part 2 commit for node_system_exporter/internal/server/ratelimit_middlware.go
