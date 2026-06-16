package server

import (
	"bind_exporter/pkg/ratelimit"
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
)

var (
	rateLimitInterval *time.Duration
	rateLimitSize     *int
	UseRatelimit      *bool
)


// TODO: implement functions
