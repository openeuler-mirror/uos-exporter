package metrics

import (
	"fmt"
	"log/slog"
	"reflect"
	"node_service_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

func init() {
	exporter.Register(NewZoneinfoCollectorWrapper())
}

const zoneinfoSubsystem = "zoneinfo"

// ZoneinfoCollectorWrapper wraps the old collector to work with new framework
type ZoneinfoCollectorWrapper struct {
	collector *ZoneinfoCollector
}

func NewZoneinfoCollectorWrapper() *ZoneinfoCollectorWrapper {
	collector, err := NewZoneinfoCollector(nil)
	if err != nil {
		return nil
	}
	return &ZoneinfoCollectorWrapper{
		collector: collector,
	}
}

func (z *ZoneinfoCollectorWrapper) Collect(ch chan<- prometheus.Metric) {
	if z.collector != nil {
		if err := z.collector.Collect(ch); err != nil {
			fmt.Printf("Error collecting metrics: %v\n", err)
		}
	}
}

// ZoneinfoCollector collects zone information metrics
type ZoneinfoCollector struct {
	gaugeMetricDescs   map[string]*prometheus.Desc
	counterMetricDescs map[string]*prometheus.Desc
	logger             *slog.Logger
	fs                 procfs.FS
}

// NewZoneinfoCollector creates a new zoneinfo collector
func NewZoneinfoCollector(logger *slog.Logger) (*ZoneinfoCollector, error) {
	if logger == nil {
		logger = slog.Default()
	}

	fs, err := procfs.NewFS("/proc")
	if err != nil {
		return nil, fmt.Errorf("failed to open procfs: %w", err)
	}

	return &ZoneinfoCollector{
		gaugeMetricDescs:   createGaugeMetricDescriptions(),
		counterMetricDescs: createCounterMetricDescriptions(),
		logger:             logger,
		fs:                 fs,
	}, nil
}

// Collect implements the Collector interface
func (c *ZoneinfoCollector) Collect(ch chan<- prometheus.Metric) error {
	if c == nil {
		return fmt.Errorf("ZoneinfoCollector is nil")
	}

	metrics, err := c.fs.Zoneinfo()
	if err != nil {
		return fmt.Errorf("couldn't get zoneinfo: %w", err)
	}

	for _, metric := range metrics {
		node := metric.Node
		zone := metric.Zone
		metricStruct := reflect.ValueOf(metric)
		typeOfMetricStruct := metricStruct.Type()

		for i := 0; i < metricStruct.NumField(); i++ {
			value := reflect.Indirect(metricStruct.Field(i))
			if value.Kind() != reflect.Int64 {
				continue
			}
			metricName := typeOfMetricStruct.Field(i).Name
			desc, ok := c.gaugeMetricDescs[metricName]
			metricType := prometheus.GaugeValue
			if !ok {
				desc = c.counterMetricDescs[metricName]
				metricType = prometheus.CounterValue
			}
			if desc != nil {
				ch <- prometheus.MustNewConstMetric(desc, metricType,
					float64(reflect.Indirect(metricStruct.Field(i)).Int()),
					node, zone)
			}
		}

		for i, value := range metric.Protection {
			if value == nil {
				continue
			}
			metricName := fmt.Sprintf("protection_%d", i)
			desc, ok := c.gaugeMetricDescs[metricName]
			if !ok {
				desc = prometheus.NewDesc(
					prometheus.BuildFQName(namespace, zoneinfoSubsystem, metricName),
					fmt.Sprintf("Protection array %d field", i),
					[]string{"node", "zone"}, nil)
				c.gaugeMetricDescs[metricName] = desc
			}
			ch <- prometheus.MustNewConstMetric(desc, prometheus.GaugeValue,
				float64(*value), node, zone)
		}
	}

	return nil
}

// createGaugeMetricDescriptions creates gauge metric descriptions
func createGaugeMetricDescriptions() map[string]*prometheus.Desc {
	return map[string]*prometheus.Desc{
		"NrFreePages": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_free_pages"),
			"Total number of free pages in the zone",
			[]string{"node", "zone"}, nil),
		"Min": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "min_pages"),
			"Zone watermark pages_min",
			[]string{"node", "zone"}, nil),
		"Low": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "low_pages"),
			"Zone watermark pages_low",
			[]string{"node", "zone"}, nil),
		"High": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "high_pages"),
			"Zone watermark pages_high",
			[]string{"node", "zone"}, nil),
		"Scanned": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "scanned_pages"),
			"Pages scanned since last reclaim",
			[]string{"node", "zone"}, nil),
		"Spanned": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "spanned_pages"),
			"Total pages spanned by the zone, including holes",
			[]string{"node", "zone"}, nil),
		"Present": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "present_pages"),
			"Physical pages existing within the zone",
			[]string{"node", "zone"}, nil),
		"Managed": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "managed_pages"),
			"Present pages managed by the buddy system",
			[]string{"node", "zone"}, nil),
		"NrActiveAnon": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_active_anon_pages"),
			"Number of anonymous pages recently more used",
			[]string{"node", "zone"}, nil),
		"NrInactiveAnon": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_inactive_anon_pages"),
			"Number of anonymous pages recently less used",
			[]string{"node", "zone"}, nil),
		"NrIsolatedAnon": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_isolated_anon_pages"),
			"Temporary isolated pages from anon lru",
			[]string{"node", "zone"}, nil),
		"NrAnonPages": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_anon_pages"),
			"Number of anonymous pages currently used by the system",
			[]string{"node", "zone"}, nil),
		"NrAnonTransparentHugepages": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_anon_transparent_hugepages"),
			"Number of anonymous transparent huge pages currently used by the system",
			[]string{"node", "zone"}, nil),
		"NrActiveFile": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_active_file_pages"),
			"Number of active pages with file-backing",
			[]string{"node", "zone"}, nil),
		"NrInactiveFile": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_inactive_file_pages"),
			"Number of inactive pages with file-backing",
			[]string{"node", "zone"}, nil),
		"NrIsolatedFile": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_isolated_file_pages"),
			"Temporary isolated pages from file lru",
			[]string{"node", "zone"}, nil),
		"NrFilePages": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_file_pages"),
			"Number of file pages",
			[]string{"node", "zone"}, nil),
		"NrSlabReclaimable": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_slab_reclaimable_pages"),
			"Number of reclaimable slab pages",
			[]string{"node", "zone"}, nil),
		"NrSlabUnreclaimable": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_slab_unreclaimable_pages"),
			"Number of unreclaimable slab pages",
			[]string{"node", "zone"}, nil),
		"NrMlockStack": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_mlock_stack_pages"),
			"mlock()ed pages found and moved off LRU",
			[]string{"node", "zone"}, nil),
		"NrKernelStack": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_kernel_stacks"),
			"Number of kernel stacks",
			[]string{"node", "zone"}, nil),
		"NrMapped": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_mapped_pages"),
			"Number of mapped pages",
			[]string{"node", "zone"}, nil),
		"NrDirty": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_dirty_pages"),
			"Number of dirty pages",
			[]string{"node", "zone"}, nil),
		"NrWriteback": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_writeback_pages"),
			"Number of writeback pages",
			[]string{"node", "zone"}, nil),
		"NrUnevictable": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_unevictable_pages"),
			"Number of unevictable pages",
			[]string{"node", "zone"}, nil),
		"NrShmem": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_shmem_pages"),
			"Number of shmem pages (included tmpfs/GEM pages)",
			[]string{"node", "zone"}, nil),
	}
}

// createCounterMetricDescriptions creates counter metric descriptions
func createCounterMetricDescriptions() map[string]*prometheus.Desc {
	return map[string]*prometheus.Desc{
		"NrDirtied": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_dirtied_total"),
			"Page dirtyings since bootup",
			[]string{"node", "zone"}, nil),
		"NrWritten": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "nr_written_total"),
			"Page writings since bootup",
			[]string{"node", "zone"}, nil),
		"NumaHit": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "numa_hit_total"),
			"Allocated in intended node",
			[]string{"node", "zone"}, nil),
		"NumaMiss": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "numa_miss_total"),
			"Allocated in non intended node",
			[]string{"node", "zone"}, nil),
		"NumaForeign": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "numa_foreign_total"),
			"Was intended here, hit elsewhere",
			[]string{"node", "zone"}, nil),
		"NumaInterleave": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "numa_interleave_total"),
			"Interleaver preferred this zone",
			[]string{"node", "zone"}, nil),
		"NumaLocal": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "numa_local_total"),
			"Allocation from local node",
			[]string{"node", "zone"}, nil),
		"NumaOther": prometheus.NewDesc(
			prometheus.BuildFQName(namespace, zoneinfoSubsystem, "numa_other_total"),
			"Allocation from other node",
			[]string{"node", "zone"}, nil),
	}
}
// Part 2 commit for node_service_exporter/internal/metrics/zoneinfo_linux.go
