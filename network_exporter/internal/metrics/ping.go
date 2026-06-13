package metrics

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"time"

	"network_exporter/config"
	"network_exporter/pkg/common"
	"network_exporter/pkg/ping"
	
	"github.com/prometheus/client_golang/prometheus"
)

// PingCacheEntry PING缓存条目
type PingCacheEntry struct {
	result    *ping.PingResult
	timestamp time.Time
}

// PingMetrics ping相关的metrics
type PingMetrics struct {
	*baseMetrics
	logger   *slog.Logger
	resolver *net.Resolver
	icmpID   *common.IcmpID
	cache    map[string]*PingCacheEntry
	cacheMux sync.RWMutex
	cacheTTL time.Duration
}

// NewPingMetrics 创建新的ping metrics实例

// TODO: implement functions
