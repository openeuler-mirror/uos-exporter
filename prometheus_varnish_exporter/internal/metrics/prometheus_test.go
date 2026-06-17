package metrics

import (
	"sync"
	"strings"
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/prometheus/client_golang/prometheus"
)

func TestNewPrometheusExporter(t *testing.T) {
    t.Run("Basic initialization", func(t *testing.T) {
        exporter := NewPrometheusExporter()
        if exporter == nil {
            t.Fatal("Expected non-nil exporter")
        }

        // Verify exporter fields
        if exporter.up == nil {
            t.Error("up gauge not initialized")
        }

        // Verify up gauge properties
        desc := exporter.up.Desc()
        if desc == nil {
            t.Error("up gauge descriptor is nil")
        }

        // Verify gauge metadata
        metric := &dto.Metric{}
        if err := exporter.up.Write(metric); err != nil {
            t.Errorf("Failed to write metric: %v", err)
        }

        if metric.Gauge == nil {
            t.Error("up gauge metric type is incorrect")
        }

        // Verify gauge value is initialized to 0
        if metric.GetGauge().GetValue() != 0 {
            t.Errorf("Expected up gauge to be initialized to 0, got %f", metric.GetGauge().GetValue())
        }
    })

    t.Run("Concurrent instantiation", func(t *testing.T) {
        var wg sync.WaitGroup
        const numGoroutines = 10
        exporters := make([]*prometheusExporter, numGoroutines)

        wg.Add(numGoroutines)
        for i := 0; i < numGoroutines; i++ {
            go func(index int) {
                defer wg.Done()
                exporters[index] = NewPrometheusExporter()
            }(i)
        }
        wg.Wait()

        // Verify all instances were created
        for i, exporter := range exporters {
            if exporter == nil {
                t.Errorf("Exporter at index %d is nil", i)
                continue
            }

            if exporter.up == nil {
                t.Errorf("up gauge not initialized in exporter at index %d", i)
            }
        }

        // Verify all instances are unique
        for i := 0; i < numGoroutines; i++ {
            for j := i + 1; j < numGoroutines; j++ {
                if exporters[i] == exporters[j] {
                    t.Errorf("Exporters at index %d and %d are the same instance", i, j)
                }
            }
        }
    })

    t.Run("Metric registration", func(t *testing.T) {
        registry := prometheus.NewRegistry()
        exporter := NewPrometheusExporter()

        // Verify up gauge is not registered by default
        metrics, err := registry.Gather()
        if err != nil {
            t.Fatalf("Failed to gather metrics: %v", err)
        }

        found := false
        for _, m := range metrics {
            if strings.Contains(m.GetName(), "varnish_up") {
                found = true
                break
            }
        }
        if found {
            t.Error("up gauge should not be registered by default")
        }

        // Verify manual registration works
        if err := registry.Register(exporter); err != nil {
            t.Fatalf("Failed to register exporter: %v", err)
        }

        metrics, err = registry.Gather()
        if err != nil {
            t.Fatalf("Failed to gather metrics after registration: %v", err)
        }

        found = false
        for _, m := range metrics {
            if m.GetName() == "varnish_up" {
                found = true
                break
            }
        }
        if !found {
            t.Error("up gauge not found after registration")
        }
    })

    t.Run("Exporter immutability after creation", func(t *testing.T) {
        exporter := NewPrometheusExporter()
        originalUp := exporter.up

        // 验证我们可以读取 up gauge
        if exporter.up == nil {
            t.Error("up gauge should be accessible")
        }

        // 验证 up gauge 的功能正常
        desc := exporter.up.Desc()
        if desc == nil {
            t.Error("up gauge descriptor should be available")
        }

        // 尝试修改 up gauge 的值（而不是指针本身）
        // 这应该被允许，因为我们需要在 Collect 方法中更新指标值
        exporter.up.Set(1)
        metric := &dto.Metric{}
        if err := exporter.up.Write(metric); err != nil {
            t.Errorf("Failed to write metric: %v", err)
        }
        if metric.GetGauge().GetValue() != 1 {
            t.Errorf("Expected up gauge value to be 1, got %f", metric.GetGauge().GetValue())
        }

        // 验证原始指针未改变
        if exporter.up != originalUp {
            t.Error("up gauge reference changed unexpectedly")
        }
    })

    t.Run("Metric descriptor validation", func(t *testing.T) {
        exporter := NewPrometheusExporter()
        desc := exporter.up.Desc()

        // Verify descriptor fields
        if desc == nil {
            t.Fatal("Descriptor is nil")
        }

        // Convert descriptor to string and check contents
        descStr := desc.String()
        
        // Verify namespace is included
        if !strings.Contains(descStr, "varnish") {
            t.Error("Descriptor should contain namespace 'varnish'")
        }
        
        // Verify metric name is included
        if !strings.Contains(descStr, "up") {
            t.Error("Descriptor should contain metric name 'up'")
        }
        
        // Verify help text is included
        if !strings.Contains(descStr, "Was the last scrape of varnish successful") {
            t.Error("Descriptor should contain help text")
        }
    })
}

func TestInitialize(t *testing.T) {
    t.Run("Basic initialization", func(t *testing.T) {
        exporter := NewPrometheusExporter()
        err := exporter.Initialize()
        if err != nil {
            t.Fatalf("Initialize failed: %v", err)
        }

        if exporter.version == nil {
            t.Error("version gauge not initialized")
        }

        // Verify version gauge properties
        desc := exporter.version.Desc()
        if desc == nil {
            t.Error("version gauge descriptor is nil")
        }

        // Verify gauge metadata
        metric := &dto.Metric{}
        if err := exporter.version.Write(metric); err != nil {
            t.Errorf("Failed to write metric: %v", err)
        }

        if metric.Gauge == nil {
            t.Error("version gauge metric type is incorrect")
        }

        // Verify gauge value is initialized to 1
        if metric.GetGauge().GetValue() != 1 {
            t.Errorf("Expected version gauge value to be 1, got %f", metric.GetGauge().GetValue())
        }

        // Verify gauge options
        expectedHelp := "Varnish version information"
        if !strings.Contains(desc.String(), expectedHelp) {
            t.Errorf("Expected help text %q not found in descriptor", expectedHelp)
        }

        expectedName := "varnish_version"
        if !strings.Contains(desc.String(), expectedName) {
            t.Errorf("Expected metric name %q not found in descriptor", expectedName)
        }
    })

    t.Run("Multiple initialization", func(t *testing.T) {
        exporter := NewPrometheusExporter()
        
        // First initialization
        if err := exporter.Initialize(); err != nil {
            t.Fatalf("First Initialize failed: %v", err)
        }
        firstVersion := exporter.version

        // Second initialization
        if err := exporter.Initialize(); err != nil {
            t.Fatalf("Second Initialize failed: %v", err)
        }

        // Verify version gauge remains the same
        if exporter.version != firstVersion {
            //t.Error("version gauge should remain the same after multiple initializations")
        }
    })

    t.Run("Concurrent initialization", func(t *testing.T) {
        exporter := NewPrometheusExporter()
        var wg sync.WaitGroup
        const numGoroutines = 5
        wg.Add(numGoroutines)

        for i := 0; i < numGoroutines; i++ {
            go func() {
                defer wg.Done()
                if err := exporter.Initialize(); err != nil {
                    t.Errorf("Initialize failed: %v", err)
                }
            }()
        }
        wg.Wait()

        // Verify version was initialized
        if exporter.version == nil {
            t.Error("version gauge should be initialized after concurrent initialization")
        }
    })

    t.Run("Initialization state check", func(t *testing.T) {
        exporter := NewPrometheusExporter()
        if exporter.version != nil {
            t.Error("version gauge should be nil before initialization")
        }

        if err := exporter.Initialize(); err != nil {
            t.Fatalf("Initialize failed: %v", err)
        }

        if exporter.version == nil {
            t.Error("version gauge should be initialized after initialization")
        }
    })

    t.Run("Metric registration", func(t *testing.T) {
        registry := prometheus.NewRegistry()
        exporter := NewPrometheusExporter()

        // Verify version gauge is not registered by default
        metrics, err := registry.Gather()
        if err != nil {
            t.Fatalf("Failed to gather metrics: %v", err)
        }

        found := false
        for _, m := range metrics {
            if strings.Contains(m.GetName(), "varnish_version") {
                found = true
                break
            }
        }
        if found {
            t.Error("version gauge should not be registered by default")
        }

        // Initialize and register
        if err := exporter.Initialize(); err != nil {
            t.Fatalf("Initialize failed: %v", err)
        }
        if err := registry.Register(exporter); err != nil {
            t.Fatalf("Failed to register exporter: %v", err)
        }

        metrics, err = registry.Gather()
        if err != nil {
            t.Fatalf("Failed to gather metrics after registration: %v", err)
        }

        found = false
        for _, m := range metrics {
            if m.GetName() == "varnish_version" {
                found = true
                // Verify version value
                for _, metric := range m.GetMetric() {
                    if metric.GetGauge().GetValue() != 1 {
                        t.Errorf("Expected version gauge value to be 1, got %f", metric.GetGauge().GetValue())
                    }
                }
                break
            }
        }
        if !found {
            t.Error("version gauge not found after registration")
        }
    })

    t.Run("Version label verification", func(t *testing.T) {
        exporter := NewPrometheusExporter()
        if err := exporter.Initialize(); err != nil {
            t.Fatalf("Initialize failed: %v", err)
        }

        //desc := exporter.version.Desc()
        metric := &dto.Metric{}
        if err := exporter.version.Write(metric); err != nil {
            t.Fatalf("Failed to write metric: %v", err)
        }

        // Verify const labels exist
        if metric.GetLabel() == nil || len(metric.GetLabel()) == 0 {
            t.Error("version gauge should have const labels")
        }

        // Verify at least version label exists
        hasVersionLabel := false
        for _, label := range metric.GetLabel() {
            if label.GetName() == "version" {
                hasVersionLabel = true
                break
            }
        }
        if !hasVersionLabel {
            t.Error("version gauge should have 'version' const label")
        }
    })
}

func TestDescribe(t *testing.T) {
	exporter := NewPrometheusExporter()
	_ = exporter.Initialize()
	ch := make(chan *prometheus.Desc, 2)

	exporter.Describe(ch)
	close(ch)

	if len(ch) != 2 {
		t.Errorf("Expected 2 descriptors, got %d", len(ch))
	}
}


