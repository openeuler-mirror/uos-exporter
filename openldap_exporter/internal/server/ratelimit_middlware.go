package server

import (
	"time"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"openldap_exporter/pkg/ratelimit"
)

var (
	rateLimitInterval *time.Duration
	rateLimitSize     *int
	UseRatelimit      *bool
)


// TODO: implement functions
