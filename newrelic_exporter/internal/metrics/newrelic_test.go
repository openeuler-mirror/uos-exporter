package metrics

import (
	"testing"
	"time"
	
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/http/httptest"
	"net/url"
	
	"newrelic_exporter/internal/exporter"
	"newrelic_exporter/pkg/newrelic"
)

// 测试初始化函数是否被正确调用

// TODO: implement functions
