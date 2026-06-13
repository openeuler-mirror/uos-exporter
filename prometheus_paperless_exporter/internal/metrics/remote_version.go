package metrics

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// VersionInfo 封装版本信息
type VersionInfo struct {
	CurrentVersion    string
	LatestVersion     string
	UpdateAvailable   bool
	LastChecked       time.Time
	CheckError        error
	VersionComponents map[string]int // 分解版本号为组件
}

// VersionCache 版本信息缓存
type VersionCache struct {
	mu    sync.RWMutex
	info  VersionInfo
}

type RemoteVersionClient interface {
	GetRemoteVersion(ctx context.Context) (*client.RemoteVersion, *client.Response, error)
}


// TODO: implement functions
