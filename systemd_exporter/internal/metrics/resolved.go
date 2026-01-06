package metrics

import (
	"fmt"

	"github.com/godbus/dbus/v5"
	"github.com/prometheus/client_golang/prometheus"
	"systemd_exporter/internal/exporter"
)

const subsystem = "resolved"

func init() {
	exporter.Register(NewResolvedCollector("systemd_resolved_current_transactions", "Resolved Current Transactions", []string{}))
	exporter.Register(NewResolvedCollector("systemd_resolved_transactions_total", "Resolved Total Transactions", []string{}))
	exporter.Register(NewResolvedCollector("systemd_resolved_current_cache_size", "Resolved Current Cache Size", []string{}))
	exporter.Register(NewResolvedCollector("systemd_resolved_cache_hits_total", "Resolved Total Cache Hits", []string{}))
	exporter.Register(NewResolvedCollector("systemd_resolved_cache_misses_total", "Resolved Total Cache Misses", []string{}))
	exporter.Register(NewResolvedCollector("systemd_resolved_dnssec_secure_total", "Resolved Total number of DNSSEC Verdicts Secure", []string{}))
	exporter.Register(NewResolvedCollector("systemd_resolved_dnssec_insecure_total", "Resolved Total number of DNSSEC Verdicts Insecure", []string{}))
	exporter.Register(NewResolvedCollector("systemd_resolved_dnssec_bogus_total", "Resolved Total number of DNSSEC Verdicts Bogus", []string{}))
	exporter.Register(NewResolvedCollector("systemd_resolved_dnssec_indeterminate_total", "Resolved Total number of DNSSEC Verdicts Indeterminate", []string{}))
}

type ResolvedCollector struct {
	*baseMetrics
}

func NewResolvedCollector(fqname, help string, labels []string) *ResolvedCollector {
	return &ResolvedCollector{NewMetrics(fqname, help, labels)}
}

// Describe 实现 prometheus.Collector 接口
func (rc *ResolvedCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- rc.baseMetrics.desc
}

func (rc *ResolvedCollector) Collect(ch chan<- prometheus.Metric) {
	conn, err := dbus.ConnectSystemBus()
	if err != nil {
		fmt.Printf("Error connecting to system bus: %v\n", err)
		return
	}

	defer conn.Close()

	obj := conn.Object("org.freedesktop.resolve1", "/org/freedesktop/resolve1")

	cacheStats, err := parseProperty(obj, "org.freedesktop.resolve1.Manager.CacheStatistics")
	if err != nil {
		fmt.Printf("Error getting cache statistics: %v\n", err)
		return
	}

	// 根据metric名称区分不同的指标，并分别设置值
	switch rc.desc.String() {
	case "systemd_resolved_current_cache_size":
		rc.baseMetrics.collect(ch, float64(cacheStats[0]), []string{})
	case "systemd_resolved_cache_hits_total":
		rc.baseMetrics.collect(ch, float64(cacheStats[1]), []string{})
	case "systemd_resolved_cache_misses_total":
		rc.baseMetrics.collect(ch, float64(cacheStats[2]), []string{})
	}

	transactionStats, err := parseProperty(obj, "org.freedesktop.resolve1.Manager.TransactionStatistics")
	if err != nil {
		fmt.Printf("Error getting transaction statistics: %v\n", err)
		return
	}

	switch rc.desc.String() {
	case "systemd_resolved_current_transactions":
		rc.baseMetrics.collect(ch, float64(transactionStats[0]), []string{})
	case "systemd_resolved_transactions_total":
		rc.baseMetrics.collect(ch, float64(transactionStats[1]), []string{})
	}

	dnssecStats, err := parseProperty(obj, "org.freedesktop.resolve1.Manager.DNSSECStatistics")
	if err != nil {
		fmt.Printf("Error getting DNSSEC statistics: %v\n", err)
		return
	}

	switch rc.desc.String() {
	case "systemd_resolved_dnssec_secure_total":
		rc.baseMetrics.collect(ch, float64(dnssecStats[0]), []string{})
	case "systemd_resolved_dnssec_insecure_total":
		rc.baseMetrics.collect(ch, float64(dnssecStats[1]), []string{})
	case "systemd_resolved_dnssec_bogus_total":
		rc.baseMetrics.collect(ch, float64(dnssecStats[2]), []string{})
	case "systemd_resolved_dnssec_indeterminate_total":
		rc.baseMetrics.collect(ch, float64(dnssecStats[3]), []string{})
	}
}

func parseProperty(object dbus.BusObject, path string) (ret []float64, err error) {
	variant, err := object.GetProperty(path)
	if err != nil {
		return nil, err
	}
	for _, v := range variant.Value().([]interface{}) {
		i := v.(uint64)
		ret = append(ret, float64(i))
	}
	return ret, err
} 