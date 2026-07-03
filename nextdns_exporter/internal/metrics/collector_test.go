package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNextDNSMetricsCreation 测试NextDNSMetrics的创建
func TestNextDNSMetricsCreation(t *testing.T) {
	metrics := NewNextDNSMetrics("test_profile", "test_api_key")
	
	assert.NotNil(t, metrics)
	assert.Equal(t, "test_profile", metrics.profile)
	assert.Equal(t, "test_api_key", metrics.apiKey)
}
// Part 2 commit for nextdns_exporter/internal/metrics/collector_test.go
