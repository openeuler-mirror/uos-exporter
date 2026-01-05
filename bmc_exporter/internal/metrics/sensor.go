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

type SensorCollector struct {
	mu       sync.Mutex
	client   *ipmi.Client
	sensor   SensorMetrics
	cacheTTL time.Duration

	lastUpdate time.Time
	cachedData struct {
		fanMetrics  map[string]float64
		tempMetrics map[string]float64
		psuMetrics  map[string]float64
	}
}

type SensorMetrics struct {

	// 风扇指标
	fanSpeed      *prometheus.GaugeVec
	fanSpeedRatio *prometheus.GaugeVec
	fanSpeedState *prometheus.GaugeVec

	cpuTemp       *prometheus.GaugeVec
	psuVoltage    *prometheus.GaugeVec
	psuCurrent    *prometheus.GaugeVec
	psuPower      *prometheus.GaugeVec
	componentTemp *prometheus.GaugeVec
}


// TODO: implement functions
