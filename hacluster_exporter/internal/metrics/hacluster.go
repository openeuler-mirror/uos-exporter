package metrics

import (
	"fmt"
	"hacluster_exporter/internal/exporter"
	"hacluster_exporter/internal/metrics/collectors/core"
	"hacluster_exporter/internal/metrics/collectors/corosync"
	"hacluster_exporter/internal/metrics/collectors/drbd"
	"hacluster_exporter/internal/metrics/collectors/pacemaker"
	"hacluster_exporter/internal/metrics/collectors/sbd"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var config Config

func init() {
	err := LoadConfigFromFile("/etc/uos-exporter/hacluster-exporter.yaml")
	if err != nil {
		logrus.Errorf("Error loading config file: %v", err)
		os.Exit(1)
	}

	logger := logrus.New()
	HAClusterCollector := NewHAClusterCollector(logger)
	ScrapeCollector := core.NewScapreCollector(HAClusterCollector, logger)
	exporter.Register(ScrapeCollector)
}

type Config struct {
}

func LoadConfigFromFile(path string) error {

	cleanPath := filepath.Clean(path)
	// 限制文件扩展名
	ext := filepath.Ext(cleanPath)
	if ext != ".yaml" && ext != ".yml" && ext != "" {
		return fmt.Errorf("invalid file extension: only .yaml or .yml files are allowed")
	}
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		return fmt.Errorf("config file must be located within %s", configDir)
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}
	return nil
}

type HAClusterCollector struct {
	corosyncCollector  *corosync.CorosyncCollector
	PacemakerCollector *pacemaker.PacemakerCollector
	sbdCollector       *sbd.SbdCollector
	DrdbCollector      *drbd.DrbdCollector

	logger *logrus.Logger
}

func NewHAClusterCollector(logger *logrus.Logger) *HAClusterCollector {
	collector := &HAClusterCollector{
		logger: logger,
	}

	corosyncCollector, err := corosync.NewCollector(
		"/usr/sbin/corosync-cfgtool",
		"/usr/sbin/corosync-quorumtool",
		true)
	if err != nil {
		logrus.Errorf("Error initializing Corosync collector: %v", err)
	} else {
		collector.corosyncCollector = corosyncCollector
	}

	sbdCollector, err := sbd.NewCollector(
		"/usr/sbin/sbd",
		"/etc/sysconfig/sbd",
		true)
	if err != nil {
		logrus.Errorf("Error initializing SBD collector: %v", err)
	} else {
		collector.sbdCollector = sbdCollector
	}

	pacemakerCollector, err := pacemaker.NewCollector(
		"/usr/sbin/crm_mon",
		"/usr/sbin/cibadmin",
		true)
	if err != nil {
		logrus.Errorf("Error initializing Pacemaker collector: %v", err)
	} else {
		collector.PacemakerCollector = pacemakerCollector
	}

	drbdCollector, err := drbd.NewCollector(
		"/sbin/drbdsetup",
		"/var/run/drbd/splitbrain",
		true)
	if err != nil {
		logrus.Errorf("Error initializing DRBD collector: %v", err)
	} else {
		collector.DrdbCollector = drbdCollector
	}

	return collector
}

func (c *HAClusterCollector) CollectWithError(ch chan<- prometheus.Metric) error {
	var errs []error

	if c.corosyncCollector != nil {
		if err := c.corosyncCollector.CollectWithError(ch); err != nil {
			errs = append(errs, errors.Wrap(err, "corosync collector error"))
		}
	}

	if c.sbdCollector != nil {
		if err := c.sbdCollector.CollectWithError(ch); err != nil {
			errs = append(errs, errors.Wrap(err, "sbd collector error"))
		}
	}

	if c.PacemakerCollector != nil {
		if err := c.PacemakerCollector.CollectWithError(ch); err != nil {
			errs = append(errs, errors.Wrap(err, "pacemaker collector error"))
		}
	}

	if c.DrdbCollector != nil {
		if err := c.DrdbCollector.CollectWithError(ch); err != nil {
			errs = append(errs, errors.Wrap(err, "drbd collector error"))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("collection errors: %v", errs)
	}
	return nil
}

func (c *HAClusterCollector) GetSubsystem() string {
	return "hacluster"
}

func (c *HAClusterCollector) Describe(ch chan<- *prometheus.Desc) {
	c.corosyncCollector.Describe(ch)
	c.PacemakerCollector.Describe(ch)
	c.sbdCollector.Describe(ch)
	c.DrdbCollector.Describe(ch)
}

func (c *HAClusterCollector) Collect(ch chan<- prometheus.Metric) {
	c.corosyncCollector.Collect(ch)
	c.PacemakerCollector.Collect(ch)
	c.sbdCollector.Collect(ch)
	c.DrdbCollector.Collect(ch)
}
