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

// CorrespondentInfo 封装联系人信息
type CorrespondentInfo struct {
	ID                  int64
	Name                string
	Slug                string
	DocumentCount       int64
	LastCorrespondence  *time.Time
	LastUpdated         time.Time
}

// CorrespondentCache 联系人信息缓存
type CorrespondentCache struct {
	mu    sync.RWMutex
	items map[int64]CorrespondentInfo
}

type CorrespondentLister interface {
    ListAllCorrespondents(context.Context, 
		client.ListCorrespondentsOptions, 
		func(context.Context, client.Correspondent) error) error
}


// TODO: implement functions
