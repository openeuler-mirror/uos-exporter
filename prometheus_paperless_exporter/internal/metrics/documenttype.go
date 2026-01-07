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

// DocumentTypeInfo 封装文档类型信息
type DocumentTypeInfo struct {
	ID            int64
	Name          string
	Slug          string
	DocumentCount int64
	LastUpdated   time.Time
}

// DocumentTypeCache 文档类型信息缓存
type DocumentTypeCache struct {
	mu    sync.RWMutex
	items map[int64]DocumentTypeInfo
}

type documentTypeClient interface {
	ListAllDocumentTypes(context.Context, 
		client.ListDocumentTypesOptions, 
		func(context.Context, client.DocumentType) error) error
}


// TODO: implement functions
