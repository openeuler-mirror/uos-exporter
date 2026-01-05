package metrics

import (
	"os"
	"ping_exporter/internal/exporter"
	"strings"
	"sync"

	mon "github.com/digineo/go-ping/monitor"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

func init() {
	exporter.Register(
		NewPingCollector())
}

type pingCollector struct {
	monitor                 *mon.Monitor
	enableDeprecatedMetrics bool
	rttUnit                 rttUnit

	cfg *Config

	mutex sync.RWMutex

	customLabels *customLabelSet
	metrics      map[string]*mon.Metrics

	// rttDesc    scaledMetrics
	// bestDesc   scaledMetrics
	// worstDesc  scaledMetrics
	// meanDesc   scaledMetrics
	// stddevDesc scaledMetrics
	rttMillis     *prometheus.Desc
	bestMillis    *prometheus.Desc
	worstMillis   *prometheus.Desc
	meanMillis    *prometheus.Desc
	stddevMillis  *prometheus.Desc
	rttSeconds    *prometheus.Desc
	bestSeconds   *prometheus.Desc
	worstSeconds  *prometheus.Desc
	meanSeconds   *prometheus.Desc
	stddevSeconds *prometheus.Desc
	lossDesc      *prometheus.Desc
	progDesc      *prometheus.Desc
}

func NewPingCollector() *pingCollector {
	rttMetricsScale := rttInMills

	setLogLevel("debug")
	log.SetReportCaller(true)

	enableDeprecatedMetrics := false

	// rttMetricsscale值可选  s  ms  both
	if rttMetricsScale = rttUnitFromString("s"); rttMetricsScale == rttInvalid {
		log.Fatal("metrics.rttunit must be `ms` for millis, or `s` for seconds, or `both`")
	}
	log.Infof("rtt units: %#v", rttMetricsScale)

	cfg := Config{}
	addFlagToConfig(&cfg)

	if cfg.Ping.History < 1 {
		log.Fatal("ping.history-size must be greater than 0")
	}

	if cfg.Ping.Size > 65500 {
		log.Fatal("ping.size must be between 0 and 65500")
	}

	if len(cfg.Targets) == 0 {
		log.Fatal("No targets specified")
	}

	resolver := setupResolver(&cfg)

	m, err := startMonitor(&cfg, resolver)
	if err != nil {
		log.Errorln(err)
		os.Exit(2)
	}

	ret := &pingCollector{
		monitor:                 m,
		enableDeprecatedMetrics: enableDeprecatedMetrics,
		rttUnit:                 rttMetricsScale,
		cfg:                     &cfg,
	}
	ret.customLabels = newCustomLabelSet(cfg.Targets)
	ret.createDesc()
	return ret
}

func (p *pingCollector) UpdateConfig(cfg *Config) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.cfg.Targets = cfg.Targets
	p.customLabels = newCustomLabelSet(cfg.Targets)
	p.createDesc()
}

func (p *pingCollector) Describe(ch chan<- *prometheus.Desc) {
	if p.enableDeprecatedMetrics {
		// p.rttDesc.Describe(ch)
		if p.rttUnit == rttInMills || p.rttUnit == rttBoth {
			ch <- p.rttMillis
		}
		if p.rttUnit == rttInSeconds || p.rttUnit == rttBoth {
			ch <- p.rttSeconds
		}
	}
	// p.bestDesc.Describe(ch)
	// p.worstDesc.Describe(ch)
	// p.meanDesc.Describe(ch)
	// p.stddevDesc.Describe(ch)
	ch <- p.bestMillis
	ch <- p.worstMillis
	ch <- p.meanMillis
	ch <- p.stddevMillis
	ch <- p.bestSeconds
	ch <- p.worstSeconds
	ch <- p.meanSeconds
	ch <- p.stddevSeconds
	ch <- p.lossDesc
	ch <- p.progDesc
}

func (p *pingCollector) Collect(ch chan<- prometheus.Metric) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if m := p.monitor.Export(); len(m) > 0 {
		p.metrics = m
	}

	ch <- prometheus.MustNewConstMetric(p.progDesc, prometheus.GaugeValue, 1)

	for target, metrics := range p.metrics {
		l := strings.SplitN(target, " ", 3)

		targetConfig := p.cfg.TargetConfigByAddr(l[0])
		l = append(l, p.customLabels.labelValues(targetConfig)...)

		if metrics.PacketsSent > metrics.PacketsLost {
			if p.enableDeprecatedMetrics {
				// p.rttDesc.Collect(ch, metrics.Best, append(l, "best")...)
				// p.rttDesc.Collect(ch, metrics.Worst, append(l, "worst")...)
				// p.rttDesc.Collect(ch, metrics.Mean, append(l, "mean")...)
				// p.rttDesc.Collect(ch, metrics.StdDev, append(l, "std_dev")...)
				if p.rttUnit == rttInMills || p.rttUnit == rttBoth {
					ch <- prometheus.MustNewConstMetric(p.rttMillis, prometheus.GaugeValue, float64(metrics.Best), append(l, "best")...)
					ch <- prometheus.MustNewConstMetric(p.rttMillis, prometheus.GaugeValue, float64(metrics.Worst), append(l, "worst")...)
					ch <- prometheus.MustNewConstMetric(p.rttMillis, prometheus.GaugeValue, float64(metrics.Mean), append(l, "mean")...)
					ch <- prometheus.MustNewConstMetric(p.rttMillis, prometheus.GaugeValue, float64(metrics.StdDev), append(l, "std_dev")...)
				}
				if p.rttUnit == rttInSeconds || p.rttUnit == rttBoth {
					ch <- prometheus.MustNewConstMetric(p.rttSeconds, prometheus.GaugeValue, float64(metrics.Best), append(l, "best")...)
					ch <- prometheus.MustNewConstMetric(p.rttSeconds, prometheus.GaugeValue, float64(metrics.Worst), append(l, "worst")...)
					ch <- prometheus.MustNewConstMetric(p.rttSeconds, prometheus.GaugeValue, float64(metrics.Mean), append(l, "mean")...)
					ch <- prometheus.MustNewConstMetric(p.rttSeconds, prometheus.GaugeValue, float64(metrics.StdDev), append(l, "std_dev")...)
				}
			}

			if p.rttUnit == rttInMills || p.rttUnit == rttBoth {
				ch <- prometheus.MustNewConstMetric(p.bestMillis, prometheus.GaugeValue, float64(metrics.Best), l...)
				ch <- prometheus.MustNewConstMetric(p.worstMillis, prometheus.GaugeValue, float64(metrics.Worst), l...)
				ch <- prometheus.MustNewConstMetric(p.meanMillis, prometheus.GaugeValue, float64(metrics.Mean), l...)
				ch <- prometheus.MustNewConstMetric(p.stddevMillis, prometheus.GaugeValue, float64(metrics.StdDev), l...)
			}
			if p.rttUnit == rttInSeconds || p.rttUnit == rttBoth {
				ch <- prometheus.MustNewConstMetric(p.bestSeconds, prometheus.GaugeValue, float64(metrics.Best), l...)
				ch <- prometheus.MustNewConstMetric(p.worstSeconds, prometheus.GaugeValue, float64(metrics.Worst), l...)
				ch <- prometheus.MustNewConstMetric(p.meanSeconds, prometheus.GaugeValue, float64(metrics.Mean), l...)
				ch <- prometheus.MustNewConstMetric(p.stddevSeconds, prometheus.GaugeValue, float64(metrics.StdDev), l...)
			}
			// p.bestDesc.Collect(ch, metrics.Best, l...)
			// p.worstDesc.Collect(ch, metrics.Worst, l...)
			// p.meanDesc.Collect(ch, metrics.Mean, l...)
			// p.stddevDesc.Collect(ch, metrics.StdDev, l...)
		}

		loss := float64(metrics.PacketsLost) / float64(metrics.PacketsSent)
		ch <- prometheus.MustNewConstMetric(p.lossDesc, prometheus.GaugeValue, loss, l...)
	}
}

func (p *pingCollector) createDesc() {
	labelNames := []string{"target", "ip", "ip_version"}
	labelNames = append(labelNames, p.customLabels.labelNames()...)

	// p.rttDesc = newScaledDesc("rtt", "Round trip time", p.rttUnit, append(labelNames, "type"))
	// p.bestDesc = newScaledDesc("rtt_best", "Best round trip time", p.rttUnit, labelNames)
	// p.worstDesc = newScaledDesc("rtt_worst", "Worst round trip time", p.rttUnit, labelNames)
	// p.meanDesc = newScaledDesc("rtt_mean", "Mean round trip time", p.rttUnit, labelNames)
	// p.stddevDesc = newScaledDesc("rtt_std_deviation", "Standard deviation", p.rttUnit, labelNames)
	p.rttMillis = newDesc("rtt_ms", "Round trip time in millis (deprecated)", append(labelNames, "type"), nil)
	p.bestMillis = newDesc("rtt_best_ms", "Best round trip time in millis (deprecated)", labelNames, nil)
	p.worstMillis = newDesc("rtt_worst_ms", "Worst round trip time in millis (deprecated)", labelNames, nil)
	p.meanMillis = newDesc("rtt_mean_ms", "Mean round trip time in millis (deprecated)", labelNames, nil)
	p.stddevMillis = newDesc("rtt_std_deviation_ms", "Standard deviation in millis (deprecated)", labelNames, nil)
	p.rttSeconds = newDesc("rtt_seconds", "Round trip time in seconds", append(labelNames, "type"), nil)
	p.bestSeconds = newDesc("rtt_best_seconds", "Best round trip time in seconds", labelNames, nil)
	p.worstSeconds = newDesc("rtt_worst_seconds", "Worst round trip time in seconds", labelNames, nil)
	p.meanSeconds = newDesc("rtt_mean_seconds", "Mean round trip time in seconds", labelNames, nil)
	p.stddevSeconds = newDesc("rtt_std_deviation_seconds", "Standard deviation in seconds", labelNames, nil)
	p.lossDesc = newDesc("loss_ratio", "Packet loss from 0.0 to 1.0", labelNames, nil)
	p.progDesc = newDesc("up", "ping_exporter version", nil, prometheus.Labels{"version": "1.0"})
}

func newDesc(name, help string, variableLabels []string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc("ping_"+name, help, variableLabels, constLabels)
}
