package metrics

import (
	"context"
	"log/slog"
	"net"
	"time"

	"network_exporter/config"
	"network_exporter/pkg/common"
	"network_exporter/pkg/tcp"
	"github.com/prometheus/client_golang/prometheus"
)

// TCPMetrics TCP相关的metrics
type TCPMetrics struct {
	*baseMetrics
	logger   *slog.Logger
	resolver *net.Resolver
}

// NewTCPMetrics 创建新的TCP metrics实例

// TODO: implement functions
