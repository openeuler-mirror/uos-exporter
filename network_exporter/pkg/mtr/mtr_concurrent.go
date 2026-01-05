package mtr

import (
	"context"
	"fmt"
	"math"
	"network_exporter/pkg/common"
	"network_exporter/pkg/icmp"
	"sync"
	"sync/atomic"
	"time"
)

// ConcurrentMTROptions 并发MTR的配置选项
type ConcurrentMTROptions struct {
	MaxWorkers     int           // 最大工作器数量
	BatchSize      int           // 批处理大小
	EarlyStop      bool          // 是否启用提前停止
	ProgressReport bool          // 是否启用进度报告
	Timeout        time.Duration // 总体超时时间
}

// DefaultConcurrentMTROptions 返回默认的并发MTR配置

// TODO: implement functions
