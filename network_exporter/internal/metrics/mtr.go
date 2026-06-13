package metrics

import (
	"context"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"network_exporter/config"
	"network_exporter/pkg/common"
	"network_exporter/pkg/mtr"
	"github.com/prometheus/client_golang/prometheus"
)

// MTRCacheEntry 缓存条目
type MTRCacheEntry struct {
	result    *mtr.MtrResult
	timestamp time.Time
}

// MTRMetrics MTR相关的metrics
type MTRMetrics struct {
	*baseMetrics
	logger   *slog.Logger
	resolver *net.Resolver
	cache    map[string]*MTRCacheEntry
	cacheMux sync.RWMutex
	cacheTTL time.Duration
}

// NewMTRMetrics 创建新的MTR metrics实例

// TODO: implement functions
