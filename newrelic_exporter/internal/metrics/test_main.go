package metrics

import (
	"os"
	"testing"

	"github.com/prometheus/client_golang/prometheus"

	"newrelic_exporter/internal/exporter"
	"newrelic_exporter/pkg/newrelic"
)

// 全局测试状态
var (
	originalCollector *NewRelicMetricsCollector
)

// TestMain 控制测试环境

// TODO: implement functions
