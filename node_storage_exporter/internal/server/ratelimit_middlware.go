package server

import (
	"node_storage_exporter/pkg/ratelimit"
	"github.com/alecthomas/kingpin"
	"time"
)

var (
	rateLimitInterval *time.Duration
	rateLimitSize     *int
	UseRatelimit      *bool
)


// TODO: implement functions
