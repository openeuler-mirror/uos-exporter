package metrics

import (
	"log/slog"
	"net"
	"net/http"
	"time"

	"network_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

// HTTPMetrics HTTP相关的metrics
type HTTPMetrics struct {
	*baseMetrics
	logger   *slog.Logger
	resolver *net.Resolver
}

// NewHTTPMetrics 创建新的HTTP metrics实例

// TODO: implement functions
