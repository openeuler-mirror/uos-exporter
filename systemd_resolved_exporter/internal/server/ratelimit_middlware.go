package server

import (
	"systemd_resolved_exporter/pkg/ratelimit"
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
