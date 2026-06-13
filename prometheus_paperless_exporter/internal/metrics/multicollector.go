package metrics

import (
	"context"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/sync/errgroup"
)

type multiCollectorMember interface {
	Describe(chan<- *prometheus.Desc)
	Collect(context.Context, chan<- prometheus.Metric) error
}

type multiCollector struct {
	// Impose a timeout on collection if non-zero.
	timeout time.Duration

	logger *log.Logger

	warningsDesc *prometheus.Desc

	members []multiCollectorMember
}

var _ prometheus.Collector = (*multiCollector)(nil)


// TODO: implement functions
