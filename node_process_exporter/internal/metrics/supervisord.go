package metrics

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"time"

	"node_process_exporter/internal/exporter"
	"github.com/mattn/go-xmlrpc"
	"github.com/prometheus/client_golang/prometheus"
)

func init() {
	exporter.Register(NewSupervisordCollector())
}

type supervisordCollector struct {
	*baseMetrics
	upDesc         *prometheus.Desc
	stateDesc      *prometheus.Desc
	exitStatusDesc *prometheus.Desc
	startTimeDesc  *prometheus.Desc
	supervisordURL string
	xrpc          *xmlrpc.Client
	logger        *slog.Logger
}

func NewSupervisordCollector() *supervisordCollector {
	const subsystem = "supervisord"
	labelNames := []string{"name", "group"}
	
	// 默认supervisord URL
	supervisordURL := "http://localhost:9001/RPC2"
	
	logger := slog.Default()
	logger.Warn("This collector is deprecated and will be removed in the next major version release.")

	var xrpc *xmlrpc.Client
	if u, err := url.Parse(supervisordURL); err == nil && u.Scheme == "unix" {
		// Fake the URI scheme as http, since net/http.*Transport.roundTrip will complain
		// about a non-http(s) transport.
		xrpc = xmlrpc.NewClient("http://unix/RPC2")
		xrpc.HttpClient.Transport = &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				d := net.Dialer{Timeout: 10 * time.Second}
				return d.DialContext(ctx, "unix", u.Path)
			},
		}
	} else {
		xrpc = xmlrpc.NewClient(supervisordURL)
	}

	return &supervisordCollector{
		baseMetrics: NewMetrics("node_supervisord_collect_errors_total", "Number of errors that occurred during supervisord collection", []string{}),
		upDesc: prometheus.NewDesc(
			"node_supervisord_up",
			"Process Up",
			labelNames,
			nil,
		),
		stateDesc: prometheus.NewDesc(
			"node_supervisord_state",
			"Process State",
			labelNames,
			nil,
		),
		exitStatusDesc: prometheus.NewDesc(
			"node_supervisord_exit_status",
			"Process Exit Status",
			labelNames,
			nil,
		),
		startTimeDesc: prometheus.NewDesc(
			"node_supervisord_start_time_seconds",
			"Process start time",
			labelNames,
			nil,
		),
		supervisordURL: supervisordURL,
		xrpc:          xrpc,
		logger:        logger,
	}
}

func (c *supervisordCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.Update(ch); err != nil {
		c.logger.Error("Error updating supervisord metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(c.baseMetrics.desc, prometheus.CounterValue, 1)
	}
}

func (c *supervisordCollector) isRunning(state int) bool {
	// http://supervisord.org/subprocess.html#process-states
	const (
		// STOPPED  = 0
		STARTING = 10
		RUNNING  = 20
		// BACKOFF  = 30
		STOPPING = 40
		// EXITED   = 100
		// FATAL    = 200
		// UNKNOWN  = 1000
	)
	switch state {
	case STARTING, RUNNING, STOPPING:
		return true
	}
	return false
}

func (c *supervisordCollector) Update(ch chan<- prometheus.Metric) error {
	var info struct {
		Name          string `xmlrpc:"name"`
		Group         string `xmlrpc:"group"`
		Start         int    `xmlrpc:"start"`
		Stop          int    `xmlrpc:"stop"`
		Now           int    `xmlrpc:"now"`
		State         int    `xmlrpc:"state"`
		StateName     string `xmlrpc:"statename"`
		SpawnErr      string `xmlrpc:"spanerr"`
		ExitStatus    int    `xmlrpc:"exitstatus"`
		StdoutLogfile string `xmlrcp:"stdout_logfile"`
		StderrLogfile string `xmlrcp:"stderr_logfile"`
		PID           int    `xmlrpc:"pid"`
	}

	res, err := c.xrpc.Call("supervisor.getAllProcessInfo")
	if err != nil {
		return fmt.Errorf("unable to call supervisord: %w", err)
	}

	for _, p := range res.(xmlrpc.Array) {
		for k, v := range p.(xmlrpc.Struct) {
			switch k {
			case "name":
				info.Name = v.(string)
			case "group":
				info.Group = v.(string)
			case "start":
				info.Start = v.(int)
			case "stop":
				info.Stop = v.(int)
			case "now":
				info.Now = v.(int)
			case "state":
				info.State = v.(int)
			case "statename":
				info.StateName = v.(string)
			case "exitstatus":
				info.ExitStatus = v.(int)
			case "pid":
				info.PID = v.(int)
			}
		}
		labels := []string{info.Name, info.Group}

		ch <- prometheus.MustNewConstMetric(c.stateDesc, prometheus.GaugeValue, float64(info.State), labels...)
		ch <- prometheus.MustNewConstMetric(c.exitStatusDesc, prometheus.GaugeValue, float64(info.ExitStatus), labels...)

		if c.isRunning(info.State) {
			ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 1, labels...)
			ch <- prometheus.MustNewConstMetric(c.startTimeDesc, prometheus.CounterValue, float64(info.Start), labels...)
		} else {
			ch <- prometheus.MustNewConstMetric(c.upDesc, prometheus.GaugeValue, 0, labels...)
		}
		c.logger.Debug("process info", "group", info.Group, "name", info.Name, "state", info.StateName, "pid", info.PID)
	}

	return nil
} 