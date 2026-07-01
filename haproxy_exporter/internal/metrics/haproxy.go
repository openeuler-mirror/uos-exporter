package metrics

import (
	"encoding/csv"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"haproxy_exporter/config"
	"haproxy_exporter/internal/exporter"
	"haproxy_exporter/internal/haproxy"
	"io"
	"strconv"
	"time"
)

const (
	pxname = iota
	svname
	qcur
	qmax
	scur
	smax
	slim
	stot
	bin
	bout
	dreq
	dresp
	ereq
	econ
	eresp
	wretr
	wredis
	status
	weight
	act
	bck
	chkfail
	chkdown
	lastchg
	downtime
	qlimit
	pid
	iid
	sid
	throttle
	lbtot
	tracked
	type1
	rate
	rate_lim
	rate_max
	check_status
	check_code
	check_duration
	hrsp_1xx
	hrsp_2xx
	hrsp_3xx
	hrsp_4xx
	hrsp_5xx
	hrsp_other
	hanafail
	req_rate
	req_rate_max
	req_tot
	cli_abrt
	srv_abrt
	comp_in
	comp_out
	comp_byp
	comp_rsp
	lastsess
	last_chk
	last_agt
	qtime
	ctime
	rtime
	ttime
	agent_status
	agent_code
	agent_duration
	check_desc
	agent_desc
	check_rise
	check_fall
	check_health
	agent_rise
	agent_fall
	agent_health
	addr
	cookie
	mode
	algo
	conn_rate
	conn_rate_max
	conn_tot
	intercepted
	dcon
	dses
	wrew
	connect
	reuse
	cache_lookups
	cache_hits
	srv_icur
	src_ilim
	qtime_max
	ctime_max
	rtime_max
	ttime_max
	eint
	idle_conn_cur
	safe_conn_cur
	used_conn_cur
	need_conn_est
	uweight
	agg_server_status
	agg_server_check_status
	agg_check_status
	srid
	sess_other
	h1sess
	h2sess
	h3sess
	req_other
	h1req
	h2req
	h3req
	proto
	nn
	ssl_sess
	ssl_reused_sess
	ssl_failed_handshake
	h2_headers_rcvd
	h2_data_rcvd
	h2_settings_rcvd
	h2_rst_stream_rcvd
	h2_goaway_rcvd
	h2_detected_conn_protocol_errors
	h2_detected_strm_protocol_errors
	h2_rst_stream_resp
	h2_goaway_resp
	h2_open_connections
	h2_backend_open_streams
	h2_total_connections
	h2_backend_total_streams
	h1_open_connections
	h1_open_streams
	h1_total_connections
	h1_total_streams
	h1_bytes_in
	h1_bytes_out
	h1_spliced_bytes_in
	h1_spliced_bytes_out
)

const (
	minimumCsvFieldCount = 33

	pxnameField        = 0
	svnameField        = 1
	statusField        = 17
	typeField          = 32
	checkDurationField = 38
	qtimeMsField       = 58
	ctimeMsField       = 59
	rtimeMsField       = 60
	ttimeMsField       = 61

	excludedServerStates = ""
	showStatCmd          = "show stat\n"
	showInfoCmd          = "show info\n"
)

func init() {
	exporter.Register(
		NewHaproxy())
}

type Haproxy struct {
	haStatus
	haType
	haCheckDuration
	haQtimeMs
	haCtimeMs
	haRtimeMs
	haTtimeMs
	//haRtimeMs
}

func NewHaproxy() *Haproxy {
	return &Haproxy{
		haStatus:        *newHaStatus(),
		haType:          *newHaType(),
		haCheckDuration: *newHaCheckDuration(),
		haQtimeMs:       *newHaQtimeMs(),
		haCtimeMs:       *newHaCtimeMs(),
		haRtimeMs:       *newHaRtimeMs(),
		haTtimeMs:       *newHaTtimeMs(),
	}
}

func (qd *Haproxy) Collect(ch chan<- prometheus.Metric) {
	logrus.Info("Start collecting haproxy metrics")
	logrus.Info("get haproxy stats")
	timeout := 5 * time.Second
	fecth := haproxy.FetchHTTP(*config.ScrapeUrl, false, false, timeout)
	stat, err := fecth()
	if err != nil {
		logrus.Error(err)
		return
	}
	defer stat.Close()
	reader := csv.NewReader(stat)
	firstLineFlag := true
	for {
		if firstLineFlag {
			firstLineFlag = false
			continue
		}

		record, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		if len(record) < minimumCsvFieldCount {
			continue
		}
		if record[statusField] == excludedServerStates {
			continue
		}
		qd.haStatus.Collect(ch,
			1,
			[]string{record[pxnameField],
				record[svnameField]})
		typeValue, err := strconv.Atoi(record[typeField])
		if err != nil {
			logrus.Error(err)
		} else {
			qd.haType.Collect(ch,
				float64(typeValue),
				[]string{record[pxnameField],
					record[svnameField]})
		}

		checkDuration, err := strconv.Atoi(record[checkDurationField])
		if err != nil {
			logrus.Error(err)
		} else {
			qd.haCheckDuration.Collect(ch,
				float64(checkDuration),
				[]string{record[pxnameField],
					record[svnameField]})
		}

		qtimeMs, err := strconv.Atoi(record[qtimeMsField])
		if err != nil {
			logrus.Error(err)
		} else {
			qd.haQtimeMs.Collect(ch,
				float64(qtimeMs),
				[]string{record[pxnameField],
					record[svnameField]})
		}
		ctimeMs, err := strconv.Atoi(record[ctimeMsField])
		if err != nil {
			logrus.Error(err)
		}
		if err != nil {
			logrus.Error(err)
		} else {
			qd.haCtimeMs.Collect(ch,
				float64(ctimeMs),
				[]string{record[pxnameField],
					record[svnameField]})
		}
		rtimeMs, err := strconv.Atoi(record[rtimeMsField])
		if err != nil {
			logrus.Error(err)
		}
		if err != nil {
			logrus.Error(err)
		} else {
			qd.haRtimeMs.Collect(ch,
				float64(rtimeMs),
				[]string{record[pxnameField],
					record[svnameField]})
		}
		ttimeMs, err := strconv.Atoi(record[ttimeMsField])
		if err != nil {
			logrus.Error(err)
		}
		if err != nil {
			logrus.Error(err)
		} else {
			qd.haTtimeMs.Collect(ch,
				float64(ttimeMs),
				[]string{record[pxnameField],
					record[svnameField]})
		}
	}
}

type haStatus struct {
	*baseMetrics
}

func newHaStatus() *haStatus {
	return &haStatus{
		NewMetrics(
			"haproxy_status",
			"haproxy status ,open 1 up 2 other 0",
			[]string{"pxname",
				"svnname"})}
}

func (qd *haStatus) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type haType struct {
	*baseMetrics
}

func newHaType() *haType {
	return &haType{
		NewMetrics(
			"haproxy_type",
			"haproxy type",
			[]string{"pxname",
				"svnname"})}
}

func (qd *haType) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type haCheckDuration struct {
	*baseMetrics
}

func newHaCheckDuration() *haCheckDuration {
	return &haCheckDuration{
		NewMetrics(
			"haproxy_check_duration",
			"haproxy check duration",
			[]string{"pxname",
				"svnname"})}
}

func (qd *haCheckDuration) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type haQtimeMs struct {
	*baseMetrics
}

func newHaQtimeMs() *haQtimeMs {
	return &haQtimeMs{
		NewMetrics(
			"haproxy_qtime_ms",
			"haproxy qtime ms",
			[]string{"pxname",
				"svnname"})}
}

func (qd *haQtimeMs) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type haCtimeMs struct {
	*baseMetrics
}

func newHaCtimeMs() *haCtimeMs {
	return &haCtimeMs{
		NewMetrics(
			"haproxy_ctime_ms",
			"haproxy ctime ms",
			[]string{"pxname",
				"svnname"})}
}
func (qd *haCtimeMs) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type haRtimeMs struct {
	*baseMetrics
}

func newHaRtimeMs() *haRtimeMs {
	return &haRtimeMs{
		NewMetrics(
			"haproxy_rtime_ms",
			"haproxy rtime ms",
			[]string{"pxname",
				"svnname"})}
}

func (qd *haRtimeMs) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}

type haTtimeMs struct {
	*baseMetrics
}

func newHaTtimeMs() *haTtimeMs {
	return &haTtimeMs{
		NewMetrics(
			"haproxy_ttime_ms",
			"haproxy ttime ms",
			[]string{"pxname",
				"svnname"})}
}

func (qd *haTtimeMs) Collect(ch chan<- prometheus.Metric,
	value float64,
	labels []string) {
	qd.collect(ch,
		value,
		labels)
}
