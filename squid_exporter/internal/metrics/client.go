package metrics

import (
	"bufio"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Counter 表示从Squid获取的计数器指标
type Counter struct {
	Key       string
	Value     float64
	VarLabels []VarLabel
}

// VarLabel 表示变量标签
type VarLabel struct {
	Key   string
	Value string
}

// SquidClient 提供连接到Squid服务器的功能
type SquidClient interface {
	GetCounters() ([]Counter, error)
	GetServiceTimes() ([]Counter, error)
	GetInfos() ([]Counter, error)
}

// CacheObjectClient 保存Squid缓存对象管理器的信息
type CacheObjectClient struct {
	ch              connectionHandler
	basicAuthString string
	headers         []string
}

type connectionHandler interface {
	connect() (net.Conn, error)
}

type connectionHandlerImpl struct {
	hostname string
	port     int
}

type CacheObjectRequest struct {
	Hostname string
	Port     int
	Login    string
	Password string
	Headers  []string
}

const (
	requestProtocol = "GET cache_object://localhost/%s HTTP/1.0"
	timeout         = 10 * time.Second
)

// 连接到指定的主机和端口
func (c *connectionHandlerImpl) connect() (net.Conn, error) {
	return net.DialTimeout("tcp", fmt.Sprintf("%s:%d", c.hostname, c.port), timeout)
}

// 创建基本认证字符串

// TODO: implement functions
