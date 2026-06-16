package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCollector 是一个模拟的 crmMonCollector
type MockCollector struct {
	mock.Mock
	crmMonCollector
}

// TestExposeResourcesGroup 测试 exposeResourcesGroup 函数

// TODO: implement functions
