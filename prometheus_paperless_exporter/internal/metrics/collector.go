package metrics

import (
    "fmt"
    "time"
    "errors"
    "github.com/hansmi/paperhooks/pkg/client"
    "github.com/prometheus/client_golang/prometheus"
)

// CollectorFactory is responsible for creating prometheus.Collector instances
// with configurable options.
type CollectorFactory struct {
    client               *client.Client
    timeout              time.Duration
    enableRemoteNetwork  bool
}

// NewCollectorFactory creates a new CollectorFactory instance with the given parameters.

// TODO: implement functions
