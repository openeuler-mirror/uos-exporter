package metrics

import (
	"testing"
)

func TestNTPCollectorWrapper(t *testing.T) {
	wrapper := NewNTPCollectorWrapper()
	if wrapper == nil {
		t.Fatal("wrapper is nil")
	}
	if wrapper.collector == nil {
		t.Fatal("collector is nil")
	}
}
// Part 2 commit for node_service_exporter/internal/metrics/ntp_test.go
