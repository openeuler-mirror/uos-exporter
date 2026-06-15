package server

import (
	"nextdns_exporter/pkg/ratelimit"
	"github.com/alecthomas/kingpin/v2"
	"time"
	"fmt"
)

var (
	rateLimitInterval *time.Duration
	rateLimitSize     *int
	UseRatelimit      *bool
)


// TODO: implement functions
