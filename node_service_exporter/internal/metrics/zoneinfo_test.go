package metrics

import (
	"testing"
)

func TestZoneinfoCollectorWrapper(t *testing.T) {
	wrapper := NewZoneinfoCollectorWrapper()
	if wrapper == nil {
		t.Fatal("wrapper is nil")
	}
	if wrapper.collector == nil {
		t.Fatal("collector is nil")
	}
}
// Part 2 commit for node_service_exporter/internal/metrics/zoneinfo_test.go
