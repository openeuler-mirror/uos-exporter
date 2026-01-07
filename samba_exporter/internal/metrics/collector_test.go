package metrics

import (
	"fmt"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestNewSambaExporter(t *testing.T) {
	requestHandler := *NewPipeHandler(true, RequestPipe)
	responseHandler := *NewPipeHandler(true, ResposePipe)
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
	exporter := NewSambaExporter(&requestHandler, &responseHandler, "0.0.0", 5, parms)

	if exporter.RequestHandler.PipeType != RequestPipe {
		t.Errorf("The exporter.RequestHandler is not of the expected type")
	}

	if exporter.ResponseHander.PipeType != ResposePipe {
		t.Errorf("The exporter.RequestHandler is not of the expected type")
	}

	if exporter.descriptions == nil {
		t.Errorf("exporter.Descriptions are nil")
	}

	if exporter.Version != "0.0.0" {
		t.Errorf("The Version \"%s\" is not expected", exporter.Version)
	}
}

func TestSetDescriptionsFromResponse(t *testing.T) {
	expectedChanels := 38
	requestHandler := *NewPipeHandler(true, RequestPipe)
	responseHandler := *NewPipeHandler(true, ResposePipe)
	locks := GetLockData(LockDataNoData)
	shares := GetShareData(ShareDataOneLine)
	processes := GetProcessData(ProcessDataOneLine)
	psData := GetPsData(TestPsResponseEmpty())
	ch := make(chan *prometheus.Desc, expectedChanels)
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
	exporter := NewSambaExporter(&requestHandler, &responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, ch)

	if len(ch) != expectedChanels {
		t.Errorf("The number of descriptions is not expected")
	}

	for i := 0; i < expectedChanels; i++ {
		desc := <-ch
		if desc == nil {
			t.Errorf("Got a nil description for a metric")
		}
	}
}

func TestSetMetricsFromResponse(t *testing.T) {
	expectedDescChanels := 38
	expectedMetChanels := 65
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	psData := GetPsData(TestPsResponse())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)

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
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 31, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric channels, but expected %d", len(chMet), expectedMetChanels)
	}

	for i := 0; i < expectedMetChanels; i++ {
		metric := <-chMet
		desc := metric.Desc()
		if desc == nil {
			t.Errorf("Got a nil description for a metric")
		}
	}
}

func TestSetMetricsFromResponseNameWithSpaces(t *testing.T) {
	expectedDescChanels := 38
	expectedMetChanels := 61
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)
	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4LinesWithSpacesInName)
	processes := GetProcessData(ProcessData4Lines)
	psData := GetPsData(TestPsResponse())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
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
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 31, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric channels, but expected %d", len(chMet), expectedMetChanels)
	}

	var metrics []prometheus.Metric
	for i := 0; i < expectedMetChanels; i++ {
		metric := <-chMet
		desc := metric.Desc()
		if desc == nil {
			t.Errorf("Got a nil description for a metric")
		}
		metrics = append(metrics, metric)
	}

	if len(metrics) != expectedMetChanels {
		t.Errorf("Got '%d' metrics but expected '%d'", len(metrics), expectedMetChanels)
	}
}

func TestSetMetricsFromResponseNoPid(t *testing.T) {
	expectedDescChanels := 38
	expectedMetChanels := 47
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	psData := GetPsData(TestPsResponse())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 31, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric channels, but expected %d", len(chMet), expectedMetChanels)
	}
}

func TestSetMetricsFromResponseNoUser(t *testing.T) {
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        true,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           false,
		Do_not_expose_share_details: false,
	}
	expectedDescChanels := 38
	expectedMetChanels := 53
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	psData := GetPsData(TestPsResponse())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 31, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric channels, but expected %d", len(chMet), expectedMetChanels)
	}
}

func TestSetMetricsFromResponseNoShareDetails(t *testing.T) {
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
	expectedDescChanels := 38
	expectedMetChanels := 53
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	psData := GetPsData(TestPsResponse())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 31, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric channels, but expected %d", len(chMet), expectedMetChanels)
	}
}

func TestSetMetricsFromResponseNoClient(t *testing.T) {
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        true,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           false,
		Do_not_expose_share_details: false,
	}
	expectedDescChanels := 38
	expectedMetChanels := 53
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	psData := GetPsData(TestPsResponse())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 31, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric channels, but expected %d", len(chMet), expectedMetChanels)
	}
}

// func TestSetMetricsFromResponseCluster(t *testing.T) {
// 	parms = Parmeters{
// 		Test:                        false,
// 		Test_pipe_mode:              false,
// 		Request_timeout:             20,
// 		Do_not_expose_encryption:    false,
// 		Do_not_expose_client:        false,
// 		Do_not_expose_user:          false,
// 		Do_not_expose_pid:           false,
// 		Do_not_expose_share_details: false,
// 	}
// 	expectedDescChanels := 42
// 	expectedMetChanels := 53
// 	requestHandler := NewPipeHandler(true, RequestPipe)
// 	responseHandler := NewPipeHandler(true, ResposePipe)

// 	locks := GetLockData(LockDataCluster)
// 	shares := GetShareData(ShareDataCluster)
// 	processes := GetProcessData(ProcessDataCluster)
// 	psData := GetPsData(TestPsResponse())
// 	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
// 	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
// 	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
// 	chMet := make(chan prometheus.Metric, expectedMetChanels)
// 	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 31, chMet)

// 	if len(chMet) != expectedMetChanels {
// 		t.Errorf("Got %d metric channels, but expected %d", len(chMet), expectedMetChanels)
// 	}
// }

func TestSetMetricsFromResponseNoShare(t *testing.T) {
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
	expectedDescChanels := 38
	expectedMetChanels := 57
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	locks := GetLockData(LockData4Lines)
	shares := GetShareData(ShareData4Lines)
	processes := GetProcessData(ProcessData4Lines)
	psData := GetPsData(TestPsResponse())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 31, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric channels, but expected %d", len(chMet), expectedMetChanels)
	}
}

func TestSetMetricsFromEmptyResponse1(t *testing.T) {
	expectedDescChanels := 38
	expectedMetChanels := 19
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	locks := GetLockData(LockData0Line)
	shares := GetShareData(ShareData0Line)
	processes := GetProcessData(ProcessData0Lines)
	psData := GetPsData(TestPsResponseEmpty())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 32, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric chanels, but expected %d", len(chMet), expectedMetChanels)
	}
}

func TestSetMetricsFromEmptyResponse2(t *testing.T) {
	expectedDescChanels := 38
	expectedMetChanels := 19
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	locks := GetLockData(LockDataEmpty)
	shares := GetShareData(ShareDataEmpty)
	processes := GetProcessData(ProcessDataEmpty)
	psData := GetPsData(TestPsResponseEmpty())
	chDesc := make(chan *prometheus.Desc, expectedDescChanels)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setDescriptionsFromResponse(locks, processes, shares, psData, chDesc)
	chMet := make(chan prometheus.Metric, expectedMetChanels)
	exporter.setMetricsFromResponse(locks, processes, shares, psData, 1, 1, 32, chMet)

	if len(chMet) != expectedMetChanels {
		t.Errorf("Got %d metric chanels, but expected %d", len(chMet), expectedMetChanels)
	}
}

func TestSetGaugeDescriptionNoLabel(t *testing.T) {
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	help := "My help"
	name := "my_name"
	ch := make(chan *prometheus.Desc, 1)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)

	exporter.setGaugeDescriptionNoLabel(name, help, ch)

	desc := <-ch

	if desc == nil {
		t.Errorf("There was no description added to the chanel")
	}

	descString := desc.String()
	if !strings.Contains(descString, help) {
		t.Errorf("The description does not contain the given help")
	}

	if !strings.Contains(descString, fmt.Sprintf("samba_%s", name)) {
		t.Errorf("The description does not contain the name")
	}

}

func TestSetGaugeDescriptionWithLabel(t *testing.T) {
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	help := "My help"
	name := "my_name"
	labels := map[string]string{"key1": "value1", "key2": "value2"}
	ch := make(chan *prometheus.Desc, 1)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)

	exporter.setGaugeDescriptionWithLabel(name, help, labels, ch)

	desc := <-ch

	if desc == nil {
		t.Errorf("There was no description added to the chanel")
	}

	descString := desc.String()
	if !strings.Contains(descString, help) {
		t.Errorf("The description does not contain the given help")
	}

	if !strings.Contains(descString, fmt.Sprintf("samba_%s", name)) {
		t.Errorf("The description does not contain the name")
	}

	for key, _ := range labels {
		if !strings.Contains(descString, key) {
			t.Errorf("The Description does not contain the expected label")
		}
	}
}

func TestSetGaugeIntMetricNoLabel(t *testing.T) {
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	help := "My help"
	name := "my_name"
	chDesc := make(chan *prometheus.Desc, 1)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setGaugeDescriptionNoLabel(name, help, chDesc)
	desc := <-chDesc
	if desc == nil {
		t.Errorf("There was no description added to the chanel")
	}
	chMet := make(chan prometheus.Metric, 1)
	exporter.setGaugeIntMetricNoLabel(name, 42.0, chMet)

	met := <-chMet

	if met == nil {
		t.Errorf("Got no metric from the chanel")
	}

	if met.Desc().String() != desc.String() {
		t.Errorf("The metrics description is not the expected")
	}

}

func TestSetGaugeIntMetricNoDescription(t *testing.T) {
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	name := "my_name"
	chMet := make(chan prometheus.Metric, 1)
	exporter.setGaugeIntMetricNoLabel(name, 42.0, chMet)

	if len(chMet) != 0 {
		t.Errorf("Got metric from the chanel but expected none")
	}

}

func TestSetGaugeIntMetricWithLabel(t *testing.T) {
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	help := "My help"
	name := "my_name"
	labels := map[string]string{"key1": "value1", "key2": "value2"}
	chDesc := make(chan *prometheus.Desc, 1)
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	exporter.setGaugeDescriptionWithLabel(name, help, labels, chDesc)
	desc := <-chDesc
	if desc == nil {
		t.Errorf("There was no description added to the chanel")
	}
	chMet := make(chan prometheus.Metric, 1)
	exporter.setGaugeIntMetricWithLabel(name, 42.0, labels, chMet)

	met := <-chMet

	if met == nil {
		t.Errorf("Got no metric from the chanel")
	}

	if met.Desc().String() != desc.String() {
		t.Errorf("The metrics description is not the expected")
	}
}

func TestSetGaugeIntMetricWithLabelNoDescription(t *testing.T) {
	requestHandler := NewPipeHandler(true, RequestPipe)
	responseHandler := NewPipeHandler(true, ResposePipe)

	labels := map[string]string{"key1": "value1", "key2": "value2"}
	parms = Parmeters{
		Test:                        false,
		Test_pipe_mode:              false,
		Request_timeout:             20,
		Do_not_expose_encryption:    false,
		Do_not_expose_client:        false,
		Do_not_expose_user:          false,
		Do_not_expose_pid:           true,
		Do_not_expose_share_details: false,
	}
	exporter := NewSambaExporter(requestHandler, responseHandler, "0.0.0", 5, parms)
	name := "my_name"
	chMet := make(chan prometheus.Metric, 1)
	exporter.setGaugeIntMetricWithLabel(name, 42.0, labels, chMet)

	if len(chMet) != 0 {
		t.Errorf("Got metric from the chanel but expected none")
	}

}
