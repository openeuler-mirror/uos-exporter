package server

import (
	"elasticsearch_exporter/pkg/ratelimit"
	"github.com/alecthomas/kingpin/v2"
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
