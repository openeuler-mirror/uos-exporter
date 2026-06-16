package server

import (
	"newrelic_exporter/pkg/ratelimit"
	"newrelic_exporter/pkg/cmdline"
	"time"
)

var (
	rateLimitInterval *time.Duration
	rateLimitSize     *int
	UseRatelimit      *bool
)


// TODO: implement functions
