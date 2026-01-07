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

// TagInfo 封装标签信息
type TagInfo struct {
	ID            int64
	Name          string
	Slug          string
	DocumentCount int64
	IsInboxTag    bool
	LastUpdated   time.Time
}

// TagCache 标签信息缓存
type TagCache struct {
	mu    sync.RWMutex
	items map[int64]TagInfo
}

type TagClient interface {
	ListAllTags(context.Context, 
		client.ListTagsOptions, 
		func(context.Context, client.Tag) error) error
}


// TODO: implement functions
