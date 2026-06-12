package metrics

import (
	"testing"
	"github.com/prometheus/client_golang/prometheus"
)

func TestIPVSCollectors(t *testing.T) {
	collectors := []interface{}{
		NewIPVSConnections(),
		NewIPVSIncomingPackets(),
		NewIPVSOutgoingPackets(),
		NewIPVSIncomingBytes(),
		NewIPVSOutgoingBytes(),
		NewIPVSBackendConnectionsActive(),
		NewIPVSBackendConnectionsInactive(),
		NewIPVSBackendWeight(),
	}
	for i, c := range collectors {
		if c == nil {
			t.Errorf("collector %d is nil", i)
		}
	}
}

func TestIPVSConnections(t *testing.T) {
	collector := NewIPVSConnections()
	if collector == nil {
		t.Fatal("NewIPVSConnections returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestIPVSIncomingPackets(t *testing.T) {
	collector := NewIPVSIncomingPackets()
	if collector == nil {
		t.Fatal("NewIPVSIncomingPackets returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestIPVSOutgoingPackets(t *testing.T) {
	collector := NewIPVSOutgoingPackets()
	if collector == nil {
		t.Fatal("NewIPVSOutgoingPackets returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestIPVSIncomingBytes(t *testing.T) {
	collector := NewIPVSIncomingBytes()
	if collector == nil {
		t.Fatal("NewIPVSIncomingBytes returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestIPVSOutgoingBytes(t *testing.T) {
	collector := NewIPVSOutgoingBytes()
	if collector == nil {
		t.Fatal("NewIPVSOutgoingBytes returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestIPVSBackendConnectionsActive(t *testing.T) {
	collector := NewIPVSBackendConnectionsActive()
	if collector == nil {
		t.Fatal("NewIPVSBackendConnectionsActive returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestIPVSBackendConnectionsInactive(t *testing.T) {
	collector := NewIPVSBackendConnectionsInactive()
	if collector == nil {
		t.Fatal("NewIPVSBackendConnectionsInactive returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}

func TestIPVSBackendWeight(t *testing.T) {
	collector := NewIPVSBackendWeight()
	if collector == nil {
		t.Fatal("NewIPVSBackendWeight returned nil")
	}
	
	// Test that Collect doesn't panic
	ch := make(chan prometheus.Metric, 10)
	defer close(ch)
	
	collector.Collect(ch)
}
