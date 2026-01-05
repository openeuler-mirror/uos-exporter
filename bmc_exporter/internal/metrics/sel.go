package metrics

import (
	"bmc_exporter/internal/ipmi"
	"context"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type SELCollector struct {
	mu     sync.Mutex
	client *ipmi.Client

	metrics struct {
		entriesTotal    prometheus.Gauge
		freeSpace       prometheus.Gauge
		countByState    *prometheus.GaugeVec
		countByName     *prometheus.GaugeVec
		latestTimestamp *prometheus.GaugeVec
	}

	lastUpdate time.Time
	cacheTTL   time.Duration
	cachedData struct {
		spaceInfo map[string]float64
		entries   []map[string]string
	}
}


// TODO: implement functions
