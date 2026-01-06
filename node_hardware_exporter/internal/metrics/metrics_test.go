package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewMetrics(t *testing.T) {
	tests := []struct {
		name   string
		fqname string
		help   string
		labels []string
	}{
		{
			name:   "basic metrics",
			fqname: "test_metric",
			help:   "Test metric help",
			labels: []string{"label1", "label2"},
		},
		{
			name:   "no labels",
			fqname: "test_metric_no_labels",
			help:   "Test metric with no labels",
			labels: []string{},
		},
		{
			name:   "single label",
			fqname: "test_metric_single_label",
			help:   "Test metric with single label",
			labels: []string{"cpu"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMetrics(tt.fqname, tt.help, tt.labels)
			if m == nil {
				t.Fatal("NewMetrics returned nil")
			}
			if len(m.labels) != len(tt.labels) {
				t.Errorf("expected %d labels, got %d", len(tt.labels), len(m.labels))
			}
			if m.desc == nil {
				t.Fatal("desc should not be nil")
			}
		})
	}
}

func TestBaseMetricsCollect(t *testing.T) {
	m := NewMetrics("test_metric", "Test metric", []string{"cpu"})
	
	ch := make(chan prometheus.Metric, 1)
	
	// Test collecting a metric
	m.collect(ch, 42.0, []string{"0"})
	
	select {
	case metric := <-ch:
		if metric == nil {
			t.Fatal("collected metric should not be nil")
		}
	default:
		t.Fatal("no metric was collected")
	}
}

func TestPackageConstants(t *testing.T) {
	if Name == "" {
		t.Error("Name should not be empty")
	}
	if Version == "" {
		t.Error("Version should not be empty")
	}
	
	expectedName := "node_hardware_exporter"
	if Name != expectedName {
		t.Errorf("expected Name to be %q, got %q", expectedName, Name)
	}
	
	expectedVersion := "1.0.0"
	if Version != expectedVersion {
		t.Errorf("expected Version to be %q, got %q", expectedVersion, Version)
	}
} 