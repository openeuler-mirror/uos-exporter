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


// TODO: implement functions
