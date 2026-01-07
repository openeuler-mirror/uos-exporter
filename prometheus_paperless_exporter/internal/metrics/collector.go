package metrics

import (
    "fmt"
    "time"
    "errors"
    "github.com/hansmi/paperhooks/pkg/client"
    "github.com/prometheus/client_golang/prometheus"
)

// CollectorFactory is responsible for creating prometheus.Collector instances
// with configurable options.
type CollectorFactory struct {
    client               *client.Client
    timeout              time.Duration
    enableRemoteNetwork  bool
}

// NewCollectorFactory creates a new CollectorFactory instance with the given parameters.
func NewCollectorFactory(
    cl *client.Client,
    timeout time.Duration,
    enableRemoteNetwork bool,
) *CollectorFactory {
    return &CollectorFactory{
        client:              cl,
        timeout:            timeout,
        enableRemoteNetwork: enableRemoteNetwork,
    }
}

// validateParameters checks if the factory parameters are valid.
func (f *CollectorFactory) validateParameters() error {
    if f.client == nil {
        return errors.New("client cannot be nil")
    }
    
    if f.timeout < 0 {
        return errors.New("timeout cannot be negative")
    }
    
    return nil
}

// createBaseCollectors creates the default set of collectors that are always enabled.
func (f *CollectorFactory) createBaseCollectors() []multiCollectorMember {
    return []multiCollectorMember{
        NewTagCollector(f.client),
		NewCorrespondentCollector(f.client),
        NewDocumentTypeCollector(f.client),
		NewStoragePathCollector(f.client),
        NewTaskCollector(f.client),
        NewLogCollector(f.client),
        NewGroupCollector(f.client),
        NewUserCollector(f.client),
        NewDocumentCollector(f.client),
		NewStatusCollector(f.client),
        NewStatisticsCollector(f.client),
        NewRemoteVersionCollector(f.client),
    }
}

// createOptionalCollectors creates collectors that are conditionally enabled.
func (f *CollectorFactory) createOptionalCollectors() []multiCollectorMember {
    var collectors []multiCollectorMember
    
    if f.enableRemoteNetwork {
        collectors = append(collectors, NewRemoteVersionCollector(f.client))
    }
    
    return collectors
}

// combineCollectors merges base and optional collectors into a single slice.
func (f *CollectorFactory) combineCollectors() []multiCollectorMember {
    baseCollectors := f.createBaseCollectors()
    optionalCollectors := f.createOptionalCollectors()
    
    return append(baseCollectors, optionalCollectors...)
}

// Build constructs and configures the final prometheus.Collector instance.
func (f *CollectorFactory) Build() (prometheus.Collector, error) {
    if err := f.validateParameters(); err != nil {
        return nil, fmt.Errorf("invalid collector parameters: %w", err)
    }
    
    allCollectors := f.combineCollectors()
    
    collector := newMultiCollector(allCollectors...)
    collector.timeout = f.timeout
    
    return collector, nil
}

// NewCollector creates a new prometheus.Collector instance with the given configuration.
// This is the original function preserved for backward compatibility.
func NewCollector(
    cl *client.Client,
    timeout time.Duration,
    enableRemoteNetwork bool,
) prometheus.Collector {
    factory := NewCollectorFactory(cl, timeout, enableRemoteNetwork)
    
    collector, err := factory.Build()
    if err != nil {
        // In the original implementation, errors were not handled,
        // so we panic here to maintain the same behavior.
        panic(fmt.Sprintf("failed to create collector: %v", err))
    }
    
    return collector
}
