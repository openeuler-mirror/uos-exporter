package metrics

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestGetSmbdMetricsNotRunningProcess(t *testing.T) {

	metrics := GetSmbdMetrics([]PsUtilPidData{}, false)

	if len(metrics) != 19 {
		t.Errorf("Got %d lines but expected %d", len(metrics), 7)
	}

	if metrics[0].Name != "smbd_unique_process_id_count" {
		t.Errorf("The metric at index '0' name '%s' is not expected", metrics[0].Name)
	}

	if metrics[0].Value != 0 {
		t.Errorf("Found '%f' processes, but 0 expected", metrics[0].Value)
	}

	if metricArrContainsItemWithName(metrics, "smbd_cpu_usage_percentage") == false {
		t.Errorf("Can not find a metric named 'smbd_cpu_usage_percentage'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_cpu_usage_percentage") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_cpu_usage_percentage'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_virtual_memory_usage_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_virtual_memory_usage_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_virtual_memory_usage_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_virtual_memory_usage_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_cpu_usage_percentage") == false {
		t.Errorf("Can not find a metric named 'smbd_cpu_usage_percentage'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_cpu_usage_percentage") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_cpu_usage_percentage'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_io_counter_read_count") == false {
		t.Errorf("Can not find a metric named 'smbd_io_counter_read_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_io_counter_read_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_io_counter_read_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_io_counter_write_count") == false {
		t.Errorf("Can not find a metric named 'smbd_io_counter_write_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_io_counter_write_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_io_counter_write_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_io_counter_read_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_io_counter_read_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_io_counter_read_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_io_counter_read_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_io_counter_write_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_io_counter_write_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_io_counter_write_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_io_counter_write_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_open_file_count") == false {
		t.Errorf("Can not find a metric named 'smbd_open_file_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_open_file_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_open_file_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_thread_count") == false {
		t.Errorf("Can not find a metric named 'smbd_thread_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_thread_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_thread_count'")
	}
}

func TestGetSmbdMetricsRunningProcessNoPids(t *testing.T) {

	pidData := GetTestPsUtilPidData()
	metrics := GetSmbdMetrics(pidData, true)

	if len(metrics) < 1 {
		t.Errorf("Got less then one metric")
	}

	if metrics[0].Name != "smbd_unique_process_id_count" {
		t.Errorf("The metric at index '0' name '%s' is not expected", metrics[0].Name)
	}

	if metrics[0].Value != 2 {
		t.Errorf("Found '%f' processes, but at two expected", metrics[0].Value)
	}

	expectedMetricCount := 10
	if len(metrics) != expectedMetricCount {
		t.Errorf("Got '%d' metrics but expected '%d'", len(metrics), expectedMetricCount)
	}

	if metricArrContainsItemWithName(metrics, "smbd_cpu_usage_percentage") == true {
		t.Errorf("Can find a metric named 'smbd_cpu_usage_percentage' but should not")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_cpu_usage_percentage") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_cpu_usage_percentage'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_thread_count") == true {
		t.Errorf("Can find a metric named 'smbd_thread_count' but should not")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_thread_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_thread_count'")
	}
}

func TestGetSmbdMetricsRunningProcess(t *testing.T) {

	pidData := GetTestPsUtilPidData()
	metrics := GetSmbdMetrics(pidData, false)

	if len(metrics) < 1 {
		t.Errorf("Got less then one metric")
	}

	if metrics[0].Name != "smbd_unique_process_id_count" {
		t.Errorf("The metric at index '0' name '%s' is not expected", metrics[0].Name)
	}

	if metrics[0].Value != 2 {
		t.Errorf("Found '%f' processes, but at two expected", metrics[0].Value)
	}

	numUnqueMetrics := 9
	numSumMetrics := numUnqueMetrics
	expectedMetricCount := 1 + (int(metrics[0].Value) * numUnqueMetrics) + numSumMetrics
	if len(metrics) != expectedMetricCount {
		t.Errorf("Got '%d' metrics but expected '%d'", len(metrics), expectedMetricCount)
	}

	if metricArrContainsItemWithName(metrics, "smbd_cpu_usage_percentage") == false {
		t.Errorf("Can not find a metric named 'smbd_cpu_usage_percentage'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_cpu_usage_percentage") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_cpu_usage_percentage'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_virtual_memory_usage_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_virtual_memory_usage_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_virtual_memory_usage_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_virtual_memory_usage_bytes'")
	}

	if metricArrCountItemWithName(metrics, "smbd_virtual_memory_usage_bytes") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_virtual_memory_usage_bytes' is not exported as often as expected")
	}

	if metricArrCountItemWithName(metrics, "smbd_cpu_usage_percentage") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_cpu_usage_percentage' is not exported as often as expected")
	}

	if metricArrSumItemWithName(metrics, "smbd_virtual_memory_usage_bytes") !=
		metricArrGetValueithName(metrics, "smbd_sum_virtual_memory_usage_bytes") {

		t.Errorf("The metrics 'smbd_virtual_memory_usage_bytes' sum is not equal 'smbd_sum_virtual_memory_usage_bytes'")
	}

	if metricArrSumItemWithName(metrics, "smbd_cpu_usage_percentage") !=
		metricArrGetValueithName(metrics, "smbd_sum_cpu_usage_percentage") {

		t.Errorf("The metrics 'smbd_cpu_usage_percentage' sum is not equal 'smbd_sum_cpu_usage_percentage'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_virtual_memory_usage_percent") == false {
		t.Errorf("Can not find a metric named 'smbd_virtual_memory_usage_percent'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_virtual_memory_usage_percent") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_virtual_memory_usage_percent'")
	}

	if metricArrCountItemWithName(metrics, "smbd_virtual_memory_usage_percent") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_virtual_memory_usage_percent' is not exported as often as expected")
	}

	if metricArrSumItemWithName(metrics, "smbd_virtual_memory_usage_percent") !=
		metricArrGetValueithName(metrics, "smbd_sum_virtual_memory_usage_percent") {

		t.Errorf("The metrics 'smbd_virtual_memory_usage_percent' sum is not equal 'smbd_sum_virtual_memory_usage_percent'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_io_counter_read_count") == false {
		t.Errorf("Can not find a metric named 'smbd_io_counter_read_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_io_counter_read_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_io_counter_read_count'")
	}

	if metricArrCountItemWithName(metrics, "smbd_io_counter_read_count") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_io_counter_read_count' is not exported as often as expected")
	}

	if metricArrSumItemWithName(metrics, "smbd_io_counter_read_count") !=
		metricArrGetValueithName(metrics, "smbd_sum_io_counter_read_count") {

		t.Errorf("The metrics 'smbd_io_counter_read_count' (%f) sum is not equal 'smbd_sum_io_counter_read_count' (%f)",
			metricArrSumItemWithName(metrics, "smbd_io_counter_read_count"),
			metricArrGetValueithName(metrics, "smbd_sum_io_counter_read_count"))
	}

	if metricArrContainsItemWithName(metrics, "smbd_io_counter_write_count") == false {
		t.Errorf("Can not find a metric named 'smbd_io_counter_write_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_io_counter_write_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_io_counter_write_count'")
	}

	if metricArrCountItemWithName(metrics, "smbd_io_counter_write_count") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_io_counter_write_count' is not exported as often as expected")
	}

	if metricArrSumItemWithName(metrics, "smbd_io_counter_write_count") !=
		metricArrGetValueithName(metrics, "smbd_sum_io_counter_write_count") {

		t.Errorf("The metrics 'smbd_io_counter_write_count' (%f) sum is not equal 'smbd_sum_io_counter_write_count' (%f)",
			metricArrSumItemWithName(metrics, "smbd_io_counter_write_count"),
			metricArrGetValueithName(metrics, "smbd_sum_io_counter_write_count"))
	}

	if metricArrContainsItemWithName(metrics, "smbd_io_counter_read_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_io_counter_read_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_io_counter_read_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_io_counter_read_bytes'")
	}

	if metricArrCountItemWithName(metrics, "smbd_io_counter_read_bytes") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_io_counter_read_bytes' is not exported as often as expected")
	}

	if metricArrSumItemWithName(metrics, "smbd_io_counter_read_bytes") !=
		metricArrGetValueithName(metrics, "smbd_sum_io_counter_read_bytes") {

		t.Errorf("The metrics 'smbd_io_counter_read_bytes' (%f) sum is not equal 'smbd_sum_io_counter_read_bytes' (%f)",
			metricArrSumItemWithName(metrics, "smbd_io_counter_read_bytes"),
			metricArrGetValueithName(metrics, "smbd_sum_io_counter_read_bytes"))
	}

	if metricArrContainsItemWithName(metrics, "smbd_io_counter_write_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_io_counter_write_bytes'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_io_counter_write_bytes") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_io_counter_write_bytes'")
	}

	if metricArrCountItemWithName(metrics, "smbd_io_counter_write_bytes") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_io_counter_write_bytes' is not exported as often as expected")
	}

	if metricArrSumItemWithName(metrics, "smbd_io_counter_write_bytes") !=
		metricArrGetValueithName(metrics, "smbd_sum_io_counter_write_bytes") {

		t.Errorf("The metrics 'smbd_io_counter_write_bytes' (%f) sum is not equal 'smbd_sum_io_counter_write_bytes' (%f)",
			metricArrSumItemWithName(metrics, "smbd_io_counter_write_bytes"),
			metricArrGetValueithName(metrics, "smbd_sum_io_counter_write_bytes"))
	}

	if metricArrContainsItemWithName(metrics, "smbd_open_file_count") == false {
		t.Errorf("Can not find a metric named 'smbd_open_file_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_open_file_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_open_file_count'")
	}

	if metricArrCountItemWithName(metrics, "smbd_open_file_count") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_open_file_count' is not exported as often as expected")
	}

	if metricArrSumItemWithName(metrics, "smbd_open_file_count") !=
		metricArrGetValueithName(metrics, "smbd_sum_open_file_count") {

		t.Errorf("The metrics 'smbd_open_file_count' (%f) sum is not equal 'smbd_sum_open_file_count' (%f)",
			metricArrSumItemWithName(metrics, "smbd_open_file_count"),
			metricArrGetValueithName(metrics, "smbd_sum_open_file_count"))
	}

	if metricArrContainsItemWithName(metrics, "smbd_thread_count") == false {
		t.Errorf("Can not find a metric named 'smbd_thread_count'")
	}

	if metricArrContainsItemWithName(metrics, "smbd_sum_thread_count") == false {
		t.Errorf("Can not find a metric named 'smbd_sum_thread_count'")
	}

	if metricArrCountItemWithName(metrics, "smbd_thread_count") != int(metrics[0].Value) {
		t.Errorf("The metric 'smbd_thread_count' is not exported as often as expected")
	}

	if metricArrSumItemWithName(metrics, "smbd_thread_count") !=
		metricArrGetValueithName(metrics, "smbd_sum_thread_count") {

		t.Errorf("The metrics 'smbd_thread_count' (%f) sum is not equal 'smbd_sum_thread_count' (%f)",
			metricArrSumItemWithName(metrics, "smbd_thread_count"),
			metricArrGetValueithName(metrics, "smbd_sum_thread_count"))
	}

}

func metricArrContainsItemWithName(arr []SmbStatisticsNumeric, name string) bool {
	for _, item := range arr {
		if item.Name == name {
			return true
		}
	}

	return false
}

func metricArrCountItemWithName(arr []SmbStatisticsNumeric, name string) int {
	ret := 0
	for _, item := range arr {
		if item.Name == name {
			ret++
		}
	}

	return ret
}

func metricArrSumItemWithName(arr []SmbStatisticsNumeric, name string) float64 {
	ret := float64(0)
	for _, item := range arr {
		if item.Name == name {
			ret += item.Value
		}
	}

	return ret
}

func metricArrGetValueithName(arr []SmbStatisticsNumeric, name string) float64 {
	ret := float64(0)
	for _, item := range arr {
		if item.Name == name {
			return item.Value
		}
	}

	return ret
}

func TestGetSmbStatisticsNoLockData(t *testing.T) {
	locks := GetLockData(LockDataNoData)
	shares := GetShareData(ShareDataOneLine)
	processes := GetProcessData(ProcessDataOneLine)

	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 15 {
		t.Errorf("The number of return values %d was not expected", len(ret))
	}

	if ret[0].Name != "individual_user_count" || ret[0].Value != 1.0 {
		t.Errorf("The individual_user_count does not match as expected")
	}
}

func TestGetSmbStatisticsClusterData(t *testing.T) {
	locks := GetLockData(LockDataCluster)
	shares := GetShareData(ShareDataCluster)
	processes := GetProcessData(ProcessDataCluster)

	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 33 {
		t.Errorf("The number of return values %d was not expected", len(ret))
	}

	if ret[0].Name != "individual_user_count" || ret[0].Value != 2.0 {
		t.Errorf("The individual_user_count does not match as expected")
	}

	if ret[3].Name != "client_count" || ret[3].Value != 4.0 {
		t.Errorf("The client_count does not match as expected")
	}

	if ret[4].Name != "cluster_node_count" || ret[4].Value != 3.0 {
		t.Errorf("The cluster_node_count does not match as expected")
	}
}

func getNewStatisticGenSettings() StatisticsGeneratorSettings {
	return StatisticsGeneratorSettings{}
}

func TestGetSmbStatisticsEmptyData(t *testing.T) {
	locks := GetLockData(LockData0Line)
	shares := GetShareData(ShareData0Line)
	processes := GetProcessData(ProcessData0Lines)

	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 15 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	for _, field := range ret[0:5] {
		if field.Value != 0 {
			t.Errorf("The value is not 0 when reading only empty tables")
		}
	}

	if ret[5].Name != "server_information" {
		t.Errorf("The Name \"%s\" is not expected", ret[5].Name)
	}

	if ret[5].Value != 1 {
		t.Errorf("The Value %f is not expected", ret[5].Value)
	}

	if len(ret[5].Labels) != 1 {
		t.Errorf("There are more labels than expected")
	}

	value, found := ret[5].Labels["version"]
	if !found {
		t.Errorf("No label with key \"version\" found")
	}

	if value != "" {
		t.Errorf("The SambaVersion \"%s\" is not expected", value)
	}

	value, found = ret[12].Labels["client"]
	if !found {
		t.Errorf("No label with key \"client\" found")
	}

	if value != "" {
		t.Errorf("The client \"%s\" is not expected", value)
	}
}

func TestGetSmbStatisticsEmptyRespomse(t *testing.T) {
	locks := GetLockData(LockDataEmpty)
	shares := GetShareData(ShareDataEmpty)
	processes := GetProcessData(ProcessDataEmpty)

	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 15 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	for _, field := range ret[0:5] {
		if field.Value != 0 {
			t.Errorf("The value is not 0 when reading only empty tables")
		}
	}

	if ret[5].Name != "server_information" {
		t.Errorf("The Name \"%s\" is not expected", ret[5].Name)
	}

	if ret[5].Value != 1 {
		t.Errorf("The Value %f is not expected", ret[5].Value)
	}

	if len(ret[5].Labels) != 1 {
		t.Errorf("There are more labels than expected")
	}

	value, found := ret[5].Labels["version"]
	if !found {
		t.Errorf("No label with key \"version\" found")
	}

	if value != "" {
		t.Errorf("The SambaVersion \"%s\" is not expected", value)
	}

	value, found = ret[12].Labels["client"]
	if !found {
		t.Errorf("No label with key \"client\" found")
	}

	if value != "" {
		t.Errorf("The client \"%s\" is not expected", value)
	}
}

func TestGetSmbStatisticsEmptyResponseLabels(t *testing.T) {
	locks := GetLockData(LockData0Line)
	shares := GetShareData(ShareData0Line)
	processes := GetProcessData(ProcessData0Lines)

	ret := GetSmbStatistics(locks, processes, shares, parms)
	if len(ret) != 15 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[6].Name != "locks_per_share_count" {
		t.Errorf("The Name \"%s\" is not expected", ret[5].Name)
	}

	if ret[6].Labels["share"] != "" {
		t.Errorf("The Labels[\"share\"] %s is not expected", ret[5].Labels["share"])
	}
}

func TestGetSmbStatistics(t *testing.T) {
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)

	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 33 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[0].Name != "individual_user_count" {
		t.Errorf("The individual_user_count is not at expecgted place")
	}

	if ret[0].Value != 1 {
		t.Errorf("The individual_user_count is not the expected value")
	}

	if ret[1].Name != "locked_file_count" {
		t.Errorf("The locked_file_count is not at expecgted place")
	}

	if ret[1].Value != float64(len(locks)) {
		t.Errorf("The locked_file_count is not the expected value")
	}

	if ret[4].Name != "pid_count" {
		t.Errorf("The pid_count is not at expecgted place")
	}

	if ret[4].Value != 4 {
		t.Errorf("The pid_count is not the expected value")
	}

	if ret[2].Name != "share_count" {
		t.Errorf("The share_count is not at expecgted place")
	}

	if ret[2].Value != 4 {
		t.Errorf("The share_count is not the expected value")
	}

	if ret[3].Name != "client_count" {
		t.Errorf("The client_countis not at expecgted place")
	}

	if ret[3].Value != 4 {
		t.Errorf("The client_count is not the expected value")
	}

	if ret[5].Name != "server_information" {
		t.Errorf("The Name \"%s\" is not expected", ret[5].Name)
	}

	if ret[5].Value != 1 {
		t.Errorf("The Value %f is not expected", ret[5].Value)
	}

	if len(ret[5].Labels) != 1 {
		t.Errorf("There are more labels than expected")
	}

	value, found := ret[5].Labels["version"]
	if !found {
		t.Errorf("No label with key \"version\" found")
	}

	if value != "4.11.6-Ubuntu" {
		t.Errorf("The SambaVersion \"%s\" is not expected", value)
	}

	value, found = ret[10].Labels["protocol_version"]
	if !found {
		t.Errorf("No label with key \"protocol_version\" found")
	}

	if value != "SMB3_11" {
		t.Errorf("The Protocol Version \"%s\" is not expected", value)
	}

	if ret[11].Value != 4 {
		t.Errorf("The value %f is not expected", ret[11].Value)
	}

	value, found = ret[11].Labels["signing"]
	if !found {
		t.Errorf("No label with key \"signing\" found")
	}

	if value != "partial(AES-128-CMAC)" {
		t.Errorf("The signing \"%s\" is not expected", value)
	}

	if ret[12].Value != 4 {
		t.Errorf("The value %f is not expected", ret[12].Value)
	}

	value, found = ret[12].Labels["encryption"]
	if !found {
		t.Errorf("No label with key \"signing\" found")
	}

	if value != "-" {
		t.Errorf("The encryption \"%s\" is not expected", value)
	}

	if ret[23].Name != "client_connected_at" {
		t.Errorf("The name %s is not expected", ret[23].Name)
	}

	value, found = ret[23].Labels["client"]
	if !found {
		t.Errorf("No label with key \"client\" found")
	}

	if !strings.HasPrefix(value, "192.168.1.") {
		t.Errorf("The value %s is not expected", value)
	}

	if ret[31].Name != "lock_created_at" {
		t.Errorf("The name %s is not expected", ret[23].Name)
	}

	value, found = ret[31].Labels["user"]
	if !found {
		t.Errorf("No label with key \"client\" found")
	}

	if !strings.HasPrefix(value, "1080") {
		t.Errorf("The value %s is not expected", value)
	}

	if ret[32].Name != "lock_created_since_seconds" {
		t.Errorf("The name %s is not expected", ret[32].Name)
	}

	value, found = ret[32].Labels["user"]
	if !found {
		t.Errorf("No label with key \"client\" found")
	}

	if value != "1080" {
		t.Errorf("The value %s is not expected", value)
	}

	value, found = ret[32].Labels["share"]
	if !found {
		t.Errorf("No label with key \"client\" found")
	}

	if !strings.HasPrefix(value, "/usr/share") {
		t.Errorf("The value %s is not expected", value)
	}

	if ret[32].Value <= 0 {
		t.Errorf("The 'lock_created_since_seconds' is '%f', it's expected grater then '0'", ret[32].Value)
	}
}

func TestGetSmbStatisticsNotExportEncryption(t *testing.T) {
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          true,
		Do_not_expose_pid:           false,
		Do_not_expose_share_details: false,
	}
	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 25 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[21].Name != "client_connected_at" {
		t.Errorf("The name %s is not expected", ret[20].Name)
	}

	value, found := ret[20].Labels["client"]
	if !found {
		t.Errorf("No label with key \"client\" found")
	}

	if !strings.HasPrefix(value, "192.168.1.") {
		t.Errorf("The value %s is not expected", value)
	}

}

func TestGetSmbStatisticsNotExportClient(t *testing.T) {
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    true,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           false,
		Do_not_expose_share_details: false,
	}

	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 30 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[11].Name != "process_per_client_count" {
		t.Errorf("The name %s is not expected", ret[11].Name)
	}
}

func TestGetSmbStatisticsNotExportUser(t *testing.T) {
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)

	parms = Parmeters{
		Test:                        true,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        true,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           false,
		Do_not_expose_share_details: false,
	}
	ret := GetSmbStatistics(locks, processes, shares, parms)
	// fmt.Print(ret)
	if len(ret) != 21 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[12].Name != "encryption_method_count" {
		t.Errorf("The name %s is not expected", ret[12].Name)
	}

	// value, found := ret[18].Labels["client"]
	// if !found {
	// 	t.Errorf("No label with key \"client\" found")
	// }

	// if !strings.HasPrefix(value, "192.168.1.") {
	// 	t.Errorf("The value %s is not expected", value)
	// }
}

func TestGetSmbStatisticsNotExportShare(t *testing.T) {
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           false,
		Do_not_expose_share_details: true,
	}
	ret := GetSmbStatistics(locks, processes, shares, parms)
	if len(ret) != 21 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[8].Name != "encryption_method_count" {
		t.Errorf("The name %s is not expected", ret[8].Name)
	}

	value, found := ret[20].Labels["client"]
	if !found {
		t.Errorf("No label with key \"client\" found")
	}

	if !strings.HasPrefix(value, "192.168.1.") {
		t.Errorf("The value %s is not expected", value)
	}
}

func TestGetSmbStatisticsNotExportShareAndUser(t *testing.T) {
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        true,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           false,
		Do_not_expose_share_details: true,
	}
	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 9 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[8].Name != "encryption_method_count" {
		t.Errorf("The name %s is not expected", ret[8].Name)
	}

	// value, found := ret[5].Labels["client"]
	// if !found {
	// 	t.Errorf("No label with key \"client\" found")
	// }

	// if !strings.HasPrefix(value, "192.168.1.") {
	// 	t.Errorf("The value %s is not expected", value)
	// }

}

func TestGetSmbStatisticsAllNotExportFlags(t *testing.T) {
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    true,
		Do_not_expose_client:        true,
		Do_not_expose_user:          true,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: true,
	}
	ret := GetSmbStatistics(locks, processes, shares, parms)

	if len(ret) != 6 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[5].Name != "server_information" {
		t.Errorf("The name %s is not expected", ret[5].Name)
	}
}

func TestGetSmbStatisticsNameWithSpaces(t *testing.T) {
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4LinesWithSpacesInName)
	processes := GetProcessData(ProcessData4Lines)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           false,
		Do_not_expose_share_details: false,
	}
	ret := GetSmbStatistics(locks, processes, shares, parms)
	fmt.Print(ret)
	if len(ret) != 29 {
		t.Errorf("The number of resturn values %d was not expected", len(ret))
	}

	if ret[5].Name != "server_information" {
		t.Errorf("The name %s is not expected", ret[9].Name)
	}

	if ret[12].Name != "encryption_method_count" {
		t.Errorf("The name '%s' is not the expected 'encryption_method_count'", ret[12].Name)
	}

	if ret[12].Value != 4.0 {
		t.Errorf("The value '%f' is not the expected '4.0'", ret[12].Value)
	}
}

func TestStringArrContains(t *testing.T) {
	arr := []string{"a", "b", "c"}

	if strArrContains(arr, "a") == false {
		t.Errorf("strArrContains returns false but should true")
	}

	if strArrContains(arr, "z") == true {
		t.Errorf("strArrContains returns true but should false")
	}
}

func TestIntArrContains(t *testing.T) {
	arr := []int{1, 2, 3}

	if intArrContains(arr, 2) == false {
		t.Errorf("strArrContains returns false but should true")
	}

	if intArrContains(arr, 100) == true {
		t.Errorf("strArrContains returns true but should false")
	}
}

func TestLockArrContains(t *testing.T) {

	entry1 := lockCreationEntry{1, time.Now(), "/media/data"}
	entry2 := lockCreationEntry{1, time.Now(), "/media/projects"}
	entry3 := lockCreationEntry{2, time.Now(), "/media/projects"}
	entry4 := lockCreationEntry{2, time.Now(), "/home/user"}
	arr := []lockCreationEntry{
		entry1,
		entry2,
		entry3,
	}

	if lockArrContainsEntry(arr, entry2) == false {
		t.Errorf("lockArrContainsEntry returns false but should true")
	}

	if lockArrContainsEntry(arr, entry4) == true {
		t.Errorf("lockArrContainsEntry returns true but should false")
	}
}
