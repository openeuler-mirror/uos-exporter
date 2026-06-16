package server

import (
	"fmt"
	"time"

	"pacemaker_exporter/pkg/ratelimit"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
)

var (
	rateLimitInterval *time.Duration
	rateLimitSize     *int
	UseRatelimit      *bool
)

func init() {
	rateLimitInterval = kingpin.Flag("rate_limit_interval",
		"Rate limit interval (e.g., 1s, 1m)").Default("1s").Duration()
	rateLimitSize = kingpin.Flag("rate_limit_size",
		"Maximum number of requests per interval").Default("100").Int()
	UseRatelimit = kingpin.Flag("use_ratelimit",
		"Enable rate limiting middleware").Bool()
}

// Ratelimit returns a middleware function that applies rate limiting
func Ratelimit(rateLimiter *ratelimit.RateLimiter) HandlerFunc {
	if rateLimiter == nil {
		logrus.Error("Rate limiter cannot be nil")
		return func(req *Request) {
			req.Error = fmt.Errorf("rate limiter not properly configured")
			req.Fail(500)
		}
	}

	logrus.WithFields(logrus.Fields{
		"interval": *rateLimitInterval,
		"size":     *rateLimitSize,
	}).Info("Rate limiting middleware initialized")

	return func(req *Request) {
		if err := rateLimiter.Get(); err != nil {
			logrus.WithFields(logrus.Fields{
				"client_ip":    req.Request.RemoteAddr,
				"user_agent":   req.Request.Header.Get("User-Agent"),
				"request_path": req.Request.URL.Path,
				"method":       req.Request.Method,
				"error":        err,
			}).Warn("Rate limit exceeded")

			req.Error = fmt.Errorf("rate limit exceeded: %w", err)
			req.Fail(429)
		}
	}
}
