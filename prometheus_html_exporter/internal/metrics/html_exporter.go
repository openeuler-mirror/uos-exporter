package metrics

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/antchfx/htmlquery"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// MetricConfig 指标配置
type MetricConfig struct {
	Name   string
	Help   string
	Type   string
	Labels map[string]string
}

// 初始化时注册HTML指标收集器

// TODO: implement functions
