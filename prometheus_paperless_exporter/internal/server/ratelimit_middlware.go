package server

import (
	"prometheus_paperless_exporter/pkg/ratelimit"
	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"time"
)

var (
	rateLimitInterval *time.Duration
	rateLimitSize     *int
	UseRatelimit      *bool
)


// TODO: implement functions
