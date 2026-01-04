//go:build !test
// +build !test

package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"sync"
)

var (
	metricsMutex = sync.RWMutex{}
	// metrics is a map of unique metrics.
	metrics = map[string]prometheus.Collector{}
)

// Register a collector to the metrics.

// TODO: implement functions
