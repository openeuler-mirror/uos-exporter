package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewConstMetric(t *testing.T) {
	tests := []struct {
		name        string
		metricName  string
		valueType   prometheus.ValueType
		labelValues []string
	}{
		{
			name:        "Test advertisements received",
			metricName:  "advertisements_received_total",
			valueType:   prometheus.CounterValue,
			labelValues: []string{"iname", "intf", "vrid"},
		},
		{
			name:        "Test advertisements sent",
			metricName:  "advertisements_sent_total",
			valueType:   prometheus.CounterValue,
			labelValues: []string{"iname", "intf", "vrid"},
		},
		// 添加所有其他指标的测试用例
		{
			name:        "Test script status",
			metricName:  "script_status",
			valueType:   prometheus.GaugeValue,
			labelValues: []string{"name"},
		},
		{
			name:        "Test script state",
			metricName:  "script_state",
			valueType:   prometheus.GaugeValue,
			labelValues: []string{"name"},
		},
	}

	k := &KeepalivedCollector{}
	k.fillMetrics()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := make(chan prometheus.Metric, 1)
			k.newConstMetric(pm, tt.metricName, tt.valueType, 10, tt.labelValues...)

			select {
			case m := <-pm:
				if m == nil {
					t.Errorf("Failed to create metric for %s", tt.metricName)
				}
			default:
				t.Errorf("No metric received for %s", tt.metricName)
			}
		})
	}
}

func TestFillMetrics(t *testing.T) {
	t.Parallel()

	k := &KeepalivedCollector{}
	k.fillMetrics()

	excpectedMetrics := map[string]*prometheus.Desc{
		"keepalived_up":                                   prometheus.NewDesc("keepalived_up", "Status", nil, nil),
		"keepalived_vrrp_state":                           prometheus.NewDesc("keepalived_vrrp_state", "State of vrrp", []string{"iname", "intf", "vrid", "ip_address"}, nil),
		"keepalived_vrrp_excluded_state":                  prometheus.NewDesc("keepalived_vrrp_excluded_state", "State of vrrp with excluded VIP", []string{"iname", "intf", "vrid", "ip_address"}, nil),
		"keepalived_exporter_check_script_status":         prometheus.NewDesc("keepalived_exporter_check_script_status", "Check Script status for each VIP", []string{"iname", "intf", "vrid", "ip_address"}, nil),
		"keepalived_gratuitous_arp_delay_total":           prometheus.NewDesc("keepalived_gratuitous_arp_delay_total", "Gratuitous ARP delay", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_advertisements_received_total":        prometheus.NewDesc("keepalived_advertisements_received_total", "Advertisements received", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_advertisements_sent_total":            prometheus.NewDesc("keepalived_advertisements_sent_total", "Advertisements sent", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_become_master_total":                  prometheus.NewDesc("keepalived_become_master_total", "Became master", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_release_master_total":                 prometheus.NewDesc("keepalived_release_master_total", "Released master", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_packet_length_errors_total":           prometheus.NewDesc("keepalived_packet_length_errors_total", "Packet length errors", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_advertisements_interval_errors_total": prometheus.NewDesc("keepalived_advertisements_interval_errors_total", "Advertisement interval errors", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_ip_ttl_errors_total":                  prometheus.NewDesc("keepalived_ip_ttl_errors_total", "TTL errors", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_invalid_type_received_total":          prometheus.NewDesc("keepalived_invalid_type_received_total", "Invalid type errors", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_address_list_errors_total":            prometheus.NewDesc("keepalived_address_list_errors_total", "Address list errors", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_authentication_invalid_total":         prometheus.NewDesc("keepalived_authentication_invalid_total", "Authentication invalid", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_authentication_mismatch_total":        prometheus.NewDesc("keepalived_authentication_mismatch_total", "Authentication mismatch", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_authentication_failure_total":         prometheus.NewDesc("keepalived_authentication_failure_total", "Authentication failure", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_priority_zero_received_total":         prometheus.NewDesc("keepalived_priority_zero_received_total", "Priority zero received", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_priority_zero_sent_total":             prometheus.NewDesc("keepalived_priority_zero_sent_total", "Priority zero sent", []string{"iname", "intf", "vrid"}, nil),
		"keepalived_script_status":                        prometheus.NewDesc("keepalived_script_status", "Tracker Script Status", []string{"name"}, nil),
		"keepalived_script_state":                         prometheus.NewDesc("keepalived_script_state", "Tracker Script State", []string{"name"}, nil),
	}

	if len(k.metrics) != len(excpectedMetrics) {
		t.Fail()
	}

	for metric, desc := range k.metrics {
		if excpectedDesc, ok := excpectedMetrics[metric]; ok {
			if excpectedDesc.String() != desc.String() {
				t.Fail()
			}
		} else {
			t.Fail()
		}
	}
}
