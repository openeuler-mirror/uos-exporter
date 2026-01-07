package metrics

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/hansmi/paperhooks/pkg/client"
	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// StoragePathInfo 封装存储路径信息
type StoragePathInfo struct {
	ID            int64
	Name          string
	Slug          string
	DocumentCount int64
	Path          string
	LastUpdated   time.Time
}

// StoragePathCache 存储路径信息缓存
type StoragePathCache struct {
	mu    sync.RWMutex
	items map[int64]StoragePathInfo
}

type storagePathClient interface {
	ListAllStoragePaths(context.Context, 
		client.ListStoragePathsOptions, 
		func(context.Context, client.StoragePath) error) error
}


// TODO: implement functions
