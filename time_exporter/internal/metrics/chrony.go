package metrics

import (
	"math"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	chronysubsystem = "chrony"
)

type ChronyConfig struct {
	ChronyAddress string `yaml:"chrony_address"`
	ChronyPort    int    `yaml:"chrony_port"`
}

type ChronyMetrics struct {
	Tracking TrackingMetrics            `deepcopier:"field:Tracking"`
	Sources  map[uint16]*SourcesMetrics `deepcopier:"field:Sources"`
}

type TrackingMetrics struct {
	LI          uint8
	ClockSource string
	Correction  float64
	Stratum     int
	Precision   int
	RootDelay   float64
	RootDisp    float64
	RefID       string
	RefTime     string
	Offset      float64
	Frequency   float64
}

type SourcesMetrics struct {
	Addr string
	// sources response
	SourcesConfigured   bool
	SourcesAuthPossible bool
	SourcesAuthentic    bool
	SourcesReachable    bool
	SourcesBroadcast    bool

	SourcesSelection uint8
	SourcesCondition string

	SourcesLeap      int
	SourcesStratum   int
	SourcesPrecision int

	SourcesRootDelay float64
	SourcesRootDisp  float64

	SourcesReach   uint8
	SourcesUnreach uint8

	SourcesRefID   string
	SourcesRefTime string

	Sourcesoffset  float64
	SourcesHeadway int
	SourcesJitter  float64
	SourcesXleave  float64

	SourcesHPoll int
	SourcesPPoll int
	SourcesHMode int
	SourcesPMode int

	// sourcestats
	SourcestatsNSamples       uint32
	SourcestatsNRuns          uint32
	SourcestatsSpanSeconds    uint32
	SourcestatsStdDev         float64
	SourcestatsResidFreq      float64
	SourcestatsSkewFreq       float64
	SourcestatsEstOffset      float64
	SourcestatsEstOffsetError float64
}

type ChronyCollector struct {
	config  ChronyConfig
	metrics map[string]*MetricDesc
}

func NewChronyCollector(cfg *ChronyConfig) *ChronyCollector {
	return &ChronyCollector{
		config: *cfg,
		metrics: map[string]*MetricDesc{
			// tracking metrics
			"trackingLI": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_li"),
					"Chrony tracking LI",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingClockSource": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_clock_source"),
					"Chrony tracking clock source",
					[]string{"source"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingCorrection": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_correction"),
					"Chrony tracking correction",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingStratum": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_stratum"),
					"Chrony tracking stratum",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingPrecision": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_precision"),
					"Chrony tracking precision",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingRootDelay": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_root_delay"),
					"Chrony tracking root delay",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingRootDisp": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_root_disp"),
					"Chrony tracking root dispersion",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingRefID": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_ref_id"),
					"Chrony tracking ref id",
					[]string{"ref_id"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingRefTime": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_ref_time"),
					"Chrony tracking ref time",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingOffset": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_offset"),
					"Chrony tracking offset",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"trackingFrequency": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "tracking_frequency"),
					"Chrony tracking frequency",
					nil,
					nil,
				),
				Type: prometheus.GaugeValue,
			},

			// sources (peer) metrics
			"sources_configured": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_configured"),
					"Configured flag for source (1|0).",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_auth_possible": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_auth_possible"),
					"Auth possible flag for source (1|0).",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_authentic": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_authentic"),
					"Authentic flag for source (1|0).",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_reachable": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_reachable"),
					"Reachable flag for source (1|0).",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_broadcast": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_broadcast"),
					"Broadcast flag for source (1|0).",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_selection": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_selection"),
					"Selection state of the source.",
					append([]string{"addr"}, "desc"),
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_leap": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_leap"),
					"Leap value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_stratum": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_stratum"),
					"Stratum of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_precision": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_precision"),
					"Precision of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_root_delay": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_root_delay"),
					"Root delay of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_root_disp": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_root_disp"),
					"Root dispersion of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_ref_id": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_ref_id"),
					"Reference ID of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_ref_time": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_ref_time"),
					"Reference time of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_reach": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_reach"),
					"Reach value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_unreach": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_unreach"),
					"Unreach value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_hmode": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_hmode"),
					"HMode value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_pmode": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_pmode"),
					"PMode value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_hpoll": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_hpoll"),
					"HPoll value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_ppoll": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_ppoll"),
					"PPoll value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_offset": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_offset"),
					"Offset value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_headway": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_headway"),
					"Headway value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sources_jitter": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_jitter"),
					"Jitter value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},

			"sources_xleave": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sources_xleave"),
					"Xleave value of the source.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},

			// sourcestats metrics
			"sourcestats_nsamples": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sourcestats_nsamples"),
					"Number of sample points in measurement set.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sourcestats_nruns": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sourcestats_nruns"),
					"Number of measurement runs.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sourcestats_span_seconds": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sourcestats_span_seconds"),
					"Span of measurement set.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sourcestats_std_dev_seconds": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sourcestats_std_dev_seconds"),
					"Standard deviation of measurement set.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sourcestats_resid_freq_ppm": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sourcestats_resid_freq_ppm"),
					"Residual frequency of measurement set.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sourcestats_skew_freq_ppm": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sourcestats_skew_freq_ppm"),
					"Skew frequency of measurement set.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sourcestats_est_offset_seconds": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sourcestats_est_offset_seconds"),
					"Estimated offset of measurement set.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
			"sourcestats_est_offset_error_seconds": {
				Desc: prometheus.NewDesc(
					prometheus.BuildFQName(namespace, chronysubsystem, "sourcestats_est_offset_error_seconds"),
					"Estimated offset error of measurement set.",
					[]string{"addr"},
					nil,
				),
				Type: prometheus.GaugeValue,
			},
		},
	}
}

func (c *ChronyCollector) Update(ch chan<- prometheus.Metric) error {
	metrics, err := c.collectChronyMetrics()
	if err != nil {
		return errors.Wrap(err, "chrony metrics collection failed")
	}

	// 发送tracking指标
	c.sendTrackingMetrics(metrics.Tracking, ch)

	// 发送sources指标
	for _, source := range metrics.Sources {
		c.sendSourceMetrics(source, ch)
	}

	return nil
}

func (c *ChronyCollector) execChronyc(command string) (string, error) {
	cmd := exec.Command("chronyc", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.Wrapf(err, "chronyc %s failed: %s", command, string(output))
	}
	return string(output), nil
}

// 主收集逻辑
func (c *ChronyCollector) collectChronyMetrics() (*ChronyMetrics, error) {
	metrics := &ChronyMetrics{
		Sources: make(map[uint16]*SourcesMetrics),
	}

	// 收集tracking数据
	if err := c.collectTracking(metrics); err != nil {
		return nil, err
	}

	// 收集sources数据
	if err := c.collectSources(metrics); err != nil {
		return nil, err
	}

	return metrics, nil
}

// 解析tracking信息（示例格式：chronyc -c tracking）
func (c *ChronyCollector) collectTracking(metrics *ChronyMetrics) error {
	output, err := c.execChronyc("tracking")
	if err != nil {
		return err
	}

	fields := strings.Split(output, ",")
	if len(fields) < 14 {
		return errors.New("invalid tracking data format")
	}

	metrics.Tracking = TrackingMetrics{
		RefID:       fields[0],
		Stratum:     parseInt(fields[2]),
		RefTime:     fields[3],
		Correction:  parseFloat(fields[4]),
		LI:          parseLeapStatus(fields[13]),
		Offset:      parseFloat(fields[5]),
		Frequency:   parseFloat(fields[7]),
		RootDelay:   parseFloat(fields[10]),
		RootDisp:    parseFloat(fields[11]),
		ClockSource: "ntp",
		Precision:   -int(math.Log2(parseFloat(fields[4]))),
	}
	return nil
}
func safeIntToUint8(size int) uint8 {
	if size < 0 {
		return 0
	}
	if size > math.MaxUint8 {
		return math.MaxUint8
	}
	return uint8(size)
}

func safeIntToUint16(size int) uint16 {
	if size < 0 {
		return 0
	}
	if size > math.MaxUint16 {
		return math.MaxUint16
	}
	return uint16(size)
}

func safeIntToUint32(size int) uint32 {
	if size < 0 {
		return 0
	}
	if size > math.MaxUint32 {
		return math.MaxUint32
	}
	return uint32(size)
}

func (c *ChronyCollector) collectSources(metrics *ChronyMetrics) error {
	// 1. 获取sources基础信息
	output, err := c.execChronyc("sources")
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	for idx, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 10 {
			continue
		}

		source := &SourcesMetrics{}
		source.Addr = fields[2]
		source.SourcesStratum = parseInt(fields[3])
		source.SourcesHPoll = parseInt(fields[4])
		source.SourcesReach = safeIntToUint8(parseInt(fields[5]))
		source.SourcesHeadway = parseInt(fields[6])
		source.Sourcesoffset = parseFloat(fields[7])
		source.SourcesJitter = parseFloat(fields[9])

		// 状态符号解析（第1、2列）
		status1 := fields[0]
		status2 := fields[1]
		source.SourcesBroadcast = status1 == "~"
		source.SourcesConfigured = status1 == "^" || status1 == "=" || status1 == "#" || status1 == "x" || status1 == "~"
		switch status2 {
		case "*":
			source.SourcesSelection = 1
			source.SourcesAuthentic = true
			source.SourcesCondition = "selected"
		case "+":
			source.SourcesSelection = 2
			source.SourcesCondition = "candidate"
		case "-":
			source.SourcesSelection = 3
			source.SourcesCondition = "excluded"
		case "?":
			source.SourcesSelection = 4
			source.SourcesCondition = "unreachable"
		default:
			source.SourcesSelection = 0
			source.SourcesCondition = "unknown"
		}
		metrics.Sources[safeIntToUint16(idx)] = source // 用idx作为key，或hash addr
	}

	// 2. 为每个源补充 sourcestats
	for _, s := range metrics.Sources {
		_ = c.collectSourceStats(s)
	}

	// 3. 为每个源补充 ntpdata
	for _, s := range metrics.Sources {
		_ = c.collectNtpData(s)
	}
	return nil
}

// 调用 chronyc -c sourcestats <addr> 并填充字段
func (c *ChronyCollector) collectSourceStats(s *SourcesMetrics) error {
	output, err := c.execChronyc("sourcestats " + s.Addr)
	if err != nil {
		return err
	}
	fields := strings.Split(output, ",")
	if len(fields) < 8 {
		return nil
	}
	s.SourcestatsNSamples = safeIntToUint32(parseInt(fields[0]))
	s.SourcestatsNRuns = safeIntToUint32(parseInt(fields[1]))
	s.SourcestatsSpanSeconds = safeIntToUint32(parseInt(fields[2]))
	s.SourcestatsResidFreq = parseFloat(fields[3])
	s.SourcestatsSkewFreq = parseFloat(fields[4])
	s.SourcestatsEstOffset = parseFloat(fields[5])
	s.SourcestatsEstOffsetError = parseFloat(fields[6])
	s.SourcestatsStdDev = parseFloat(fields[7])
	return nil
}

// 调用 chronyc -c ntpdata <addr> 并填充字段
func (c *ChronyCollector) collectNtpData(s *SourcesMetrics) error {
	output, err := c.execChronyc("ntpdata " + s.Addr)
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	for _, l := range lines {
		parts := strings.SplitN(l, ":", 2)
		if len(parts) != 2 {
			continue
		}
		k, v := strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		switch k {
		case "Leap status":
			s.SourcesLeap = leapStringToCode(v)
		case "Precision":
			s.SourcesPrecision = parseInt(v)
		case "Root delay":
			s.SourcesRootDelay = parseFloat(strings.Fields(v)[0]) // 截取数值部分
		case "Root dispersion":
			s.SourcesRootDisp = parseFloat(strings.Fields(v)[0])
		case "Reference ID":
			s.SourcesRefID = v
		case "Reference time":
			s.SourcesRefTime = v
		case "Peer polling":
			s.SourcesPPoll = parseInt(v)
		case "Host polling":
			s.SourcesHPoll = parseInt(v)
		case "Mode":
			_, hmodeCode, pmodeCode := modeStringToCodes(v)
			s.SourcesHMode, s.SourcesPMode = hmodeCode, pmodeCode // 如需映射可自定义函数
		}
	}
	return nil
}

func leapStringToCode(s string) int {
	switch strings.ToLower(s) {
	case "normal":
		return 0
	case "insertsecond":
		return 1
	case "deletesecond":
		return 2
	default:
		return -1
	}
}
func modeStringToCodes(s string) (int, int, int) {
	switch strings.ToLower(s) {
	case "server":
		return 4, 4, 4 // NTP模式号4为server，按需调整返回值结构
	case "client":
		return 3, 3, 3
	default:
		return -1, -1, -1
	}
}

// 指标发送辅助方法
func (c *ChronyCollector) sendTrackingMetrics(t TrackingMetrics, ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingLI"].Desc,
		prometheus.GaugeValue,
		float64(t.LI),
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingClockSource"].Desc,
		prometheus.GaugeValue,
		1,
		t.ClockSource,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingCorrection"].Desc,
		prometheus.GaugeValue,
		t.Correction,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingStratum"].Desc,
		prometheus.GaugeValue,
		float64(t.Stratum),
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingPrecision"].Desc,
		prometheus.GaugeValue,
		float64(t.Precision),
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingRootDelay"].Desc,
		prometheus.GaugeValue,
		t.RootDelay,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingRootDisp"].Desc,
		prometheus.GaugeValue,
		t.RootDisp,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingRefID"].Desc,
		prometheus.GaugeValue,
		1,
		t.RefID,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingRefTime"].Desc,
		prometheus.GaugeValue,
		parseFloat(t.RefTime),
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingFrequency"].Desc,
		prometheus.GaugeValue,
		t.Frequency,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["trackingOffset"].Desc,
		prometheus.GaugeValue,
		t.Offset,
	)
}

func (c *ChronyCollector) sendSourceMetrics(s *SourcesMetrics, ch chan<- prometheus.Metric) {
	addr := s.Addr

	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_configured"].Desc,
		prometheus.GaugeValue,
		boolToFloat(s.SourcesConfigured),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_auth_possible"].Desc,
		prometheus.GaugeValue,
		boolToFloat(s.SourcesAuthPossible),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_authentic"].Desc,
		prometheus.GaugeValue,
		boolToFloat(s.SourcesAuthentic),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_reachable"].Desc,
		prometheus.GaugeValue,
		boolToFloat(s.SourcesReachable),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_broadcast"].Desc,
		prometheus.GaugeValue,
		boolToFloat(s.SourcesBroadcast),
		addr,
	)

	// selection/condition desc
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_selection"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesSelection),
		addr,
		s.SourcesCondition,
	)

	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_leap"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesLeap),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_stratum"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesStratum),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_precision"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesPrecision),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_root_delay"].Desc,
		prometheus.GaugeValue, s.SourcesRootDelay,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_root_disp"].Desc,
		prometheus.GaugeValue, s.SourcesRootDisp,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_ref_id"].Desc,
		prometheus.GaugeValue,
		1.0,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_ref_time"].Desc,
		prometheus.GaugeValue,
		1.0,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_reach"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesReach),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_unreach"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesUnreach),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_hmode"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesHMode),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_pmode"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesPMode),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_hpoll"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesHPoll),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_ppoll"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesPPoll),
		addr,
	)

	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_offset"].Desc,
		prometheus.GaugeValue,
		s.Sourcesoffset,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_headway"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcesHeadway),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_jitter"].Desc,
		prometheus.GaugeValue,
		s.SourcesJitter,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sources_xleave"].Desc,
		prometheus.GaugeValue,
		s.SourcesXleave,
		addr,
	)

	// sourcestats
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sourcestats_nsamples"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcestatsNSamples),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sourcestats_nruns"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcestatsNRuns),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sourcestats_span_seconds"].Desc,
		prometheus.GaugeValue,
		float64(s.SourcestatsSpanSeconds),
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sourcestats_std_dev_seconds"].Desc,
		prometheus.GaugeValue,
		s.SourcestatsStdDev,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sourcestats_resid_freq_ppm"].Desc,
		prometheus.GaugeValue,
		s.SourcestatsResidFreq,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sourcestats_skew_freq_ppm"].Desc,
		prometheus.GaugeValue,
		s.SourcestatsSkewFreq,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sourcestats_est_offset_seconds"].Desc,
		prometheus.GaugeValue,
		s.SourcestatsEstOffset,
		addr,
	)
	ch <- prometheus.MustNewConstMetric(
		c.metrics["sourcestats_est_offset_error_seconds"].Desc,
		prometheus.GaugeValue,
		s.SourcestatsEstOffsetError,
		addr,
	)
}

func boolToFloat(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func parseFloat(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}

func parseInt(s string) int {
	v, _ := strconv.Atoi(strings.TrimSpace(s))
	return v
}

func parseLeapStatus(s string) uint8 {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "normal":
		return 0
	case "leapsecond":
		return 1
	case "notsynchronized":
		return 2
	default:
		logrus.Warnf("Unknown leap status: %s", s)
		return 3
	}
}
// Part 2 commit for time_exporter/internal/metrics/chrony.go
