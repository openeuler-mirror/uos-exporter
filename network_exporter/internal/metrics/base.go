package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

// metricInfo 存储单个metric的信息
type metricInfo struct {
	desc       *prometheus.Desc
	labelNames []string
	values     map[string]float64 // key是标签值的组合
}

// baseMetrics 基础metrics结构
type baseMetrics struct {
	prefix  string
	metrics map[string]*metricInfo
	mutex   sync.RWMutex
}

// newBaseMetrics 创建新的baseMetrics实例

// TODO: implement functions
