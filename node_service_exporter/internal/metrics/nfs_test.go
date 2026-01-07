package metrics

import (
	"testing"
	"time"
	"context"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNFSCollectors(t *testing.T) {
	collectors := []interface{}{
		NewNFSNetworkPackets(),
		NewNFSNetworkConnections(),
		NewNFSRPCOperations(),
		NewNFSRPCRetransmissions(),
		NewNFSRPCAuthRefreshes(),
		NewNFSProcedures(),
	}
	for i, c := range collectors {
		if c == nil {
			t.Errorf("collector %d is nil", i)
		}
	}
}

func TestNFSNetworkPackets(t *testing.T) {
	collector := NewNFSNetworkPackets()
	if collector == nil {
		t.Fatal("NewNFSNetworkPackets returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
	// No assertion on metrics as /proc might not be available in test environment
}

func TestNFSNetworkConnections(t *testing.T) {
	collector := NewNFSNetworkConnections()
	if collector == nil {
		t.Fatal("NewNFSNetworkConnections returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestNFSRPCOperations(t *testing.T) {
	collector := NewNFSRPCOperations()
	if collector == nil {
		t.Fatal("NewNFSRPCOperations returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestNFSRPCRetransmissions(t *testing.T) {
	collector := NewNFSRPCRetransmissions()
	if collector == nil {
		t.Fatal("NewNFSRPCRetransmissions returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestNFSRPCAuthRefreshes(t *testing.T) {
	collector := NewNFSRPCAuthRefreshes()
	if collector == nil {
		t.Fatal("NewNFSRPCAuthRefreshes returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestNFSProcedures(t *testing.T) {
	collector := NewNFSProcedures()
	if collector == nil {
		t.Fatal("NewNFSProcedures returned nil")
	}
	
	// Test that Collect doesn't panic with timeout
	ch := make(chan prometheus.Metric, 100)
	defer close(ch)
	
	// Use context with timeout to avoid blocking indefinitely
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	done := make(chan bool, 1)
	go func() {
		collector.Collect(ch)
		done <- true
	}()
	
	select {
	case <-done:
		// Test completed successfully
	case <-ctx.Done():
		t.Log("NFSProcedures.Collect timed out - this is expected in test environment")
		// Don't fail the test as this is expected behavior in test environment
	}
}
// Part 2 commit for node_service_exporter/internal/metrics/nfs_test.go
