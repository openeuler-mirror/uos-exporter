package sbd

import (
	"fmt"
	"hacluster_exporter/internal/metrics/collectors/core"
	"hacluster_exporter/pkg/utils"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const subsystem = "sbd"

const SBD_STATUS_UNHEALTHY = "unhealthy"
const SBD_STATUS_HEALTHY = "healthy"

type sbdMetrics struct {
	sbdDevicesDesc  *prometheus.Desc
	sbdTimeoutsDesc *prometheus.Desc
}

// NewCollector create a new sbd collector
func NewCollector(sbdPath string, sbdConfigPath string, timestamps bool) (*SbdCollector, error) {
	err := checkArguments(sbdPath, sbdConfigPath)
	if err != nil {
		return nil, errors.Wrapf(err, "could not initialize '%s' collector", subsystem)
	}

	c := &SbdCollector{
		core.NewDefaultCollector(subsystem, timestamps),
		sbdPath,
		sbdConfigPath,
		sbdMetrics{
			sbdDevicesDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "devices"),
				"SBD devices; one line per device",
				[]string{"device", "status"},
				nil,
			),
			sbdTimeoutsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "timeouts"),
				"SBD timeouts for each device and type",
				[]string{"device", "type"},
				nil,
			),
		},
	}

	return c, nil
}

func checkArguments(sbdPath string, sbdConfigPath string) error {
	if err := core.CheckExecutables(sbdPath); err != nil {
		return err
	}
	if _, err := os.Stat(sbdConfigPath); os.IsNotExist(err) {
		return errors.Errorf("'%s' does not exist", sbdConfigPath)
	}
	return nil
}

type SbdCollector struct {
	core.DefaultCollector
	sbdPath       string
	sbdConfigPath string
	metrics       sbdMetrics
}

func (c *SbdCollector) CollectWithError(ch chan<- prometheus.Metric) error {
	logrus.Debug("Collecting pacemaker metrics...")

	sbdConfiguration, err := readSdbFile(c.sbdConfigPath)
	if err != nil {
		return err
	}

	sbdDevices := getSbdDevices(sbdConfiguration)

	sbdStatuses := c.getSbdDeviceStatuses(sbdDevices)
	for sbdDev, sbdStatus := range sbdStatuses {
		var statusValue float64 = 0
		if sbdStatus == SBD_STATUS_HEALTHY {
			statusValue = 1
		}
		ch <- prometheus.MustNewConstMetric(
			c.metrics.sbdDevicesDesc,
			prometheus.GaugeValue,
			statusValue,
			sbdDev, sbdStatus,
		)
	}

	sbdWatchdogs, sbdMsgWaits := c.getSbdTimeouts(sbdDevices)
	for sbdDev, sbdWatchdog := range sbdWatchdogs {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.sbdTimeoutsDesc,
			prometheus.GaugeValue,
			sbdWatchdog,
			sbdDev, "watchdog",
		)
	}

	for sbdDev, sbdMsgWait := range sbdMsgWaits {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.sbdTimeoutsDesc,
			prometheus.GaugeValue,
			sbdMsgWait,
			sbdDev, "msgwait",
		)
	}

	return nil
}

func (c *SbdCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.sbdDevicesDesc
	ch <- c.metrics.sbdTimeoutsDesc
}

func (c *SbdCollector) Collect(ch chan<- prometheus.Metric) {
	err := c.CollectWithError(ch)
	if err != nil {
		logrus.Warnf("%s collector scrape failed: %v", c.GetSubsystem(), err)
	}
}

func readSdbFile(sbdConfigPath string) ([]byte, error) {
	cleanSbdConfigPath := filepath.Clean(sbdConfigPath)
	configDir := "/etc/sysconfig"
	if !strings.HasPrefix(cleanSbdConfigPath, configDir) {
		return nil, fmt.Errorf("sbd config file must be located within %s", configDir)
	}
	sbdConfFile, err := os.Open(cleanSbdConfigPath)
	if err != nil {
		return nil, fmt.Errorf("could not open sbd config file %s", err)
	}

	defer sbdConfFile.Close()
	sbdConfigRaw, err := io.ReadAll(sbdConfFile)

	if err != nil {
		return nil, fmt.Errorf("could not read sbd config file %s", err)
	}
	return sbdConfigRaw, nil
}

// retrieve a list of sbd devices from the config file contents
func getSbdDevices(sbdConfigRaw []byte) []string {
	// The following regex matches lines like SBD_DEVICE="/dev/foo" or SBD_DEVICE=/dev/foo;/dev/bar
	// It captures all the colon separated device names, without double quotes, into a capture group
	// It allows for free indentation, trailing spaces and end of lines, and it will ignore commented lines
	// Unbalanced double quotes are not checked and they will still produce a match
	// If multiple matching lines are present, only the first will be used
	// The single device name pattern is `[\w-/]+`, which is pretty relaxed
	regex := regexp.MustCompile(`(?m)^\s*SBD_DEVICE="?((?:[\w-/]+;?\s?)+)"?\s*$`)
	sbdDevicesLine := regex.FindStringSubmatch(string(sbdConfigRaw))

	// if SBD_DEVICE line could not be found, return 0 devices
	if sbdDevicesLine == nil {
		return nil
	}

	// split the first capture group, e.g. `/dev/foo;/dev/bar`; the 0th element is always the whole line
	sbdDevices := strings.Split(strings.TrimRight(sbdDevicesLine[1], ";"), ";")
	for i, _ := range sbdDevices {
		sbdDevices[i] = strings.TrimSpace(sbdDevices[i])
	}

	return sbdDevices
}

// this function takes a list of sbd devices and returns
// a map of SBD device names with 1 if healthy, 0 if not
func (c *SbdCollector) getSbdDeviceStatuses(sbdDevices []string) map[string]string {
	sbdStatuses := make(map[string]string)
	for _, sbdDev := range sbdDevices {
		_, err := utils.RunCommand(c.sbdPath, "-d", sbdDev, "dump")
		// in case of error the device is not healthy
		if err != nil {
			sbdStatuses[sbdDev] = SBD_STATUS_UNHEALTHY
		} else {
			sbdStatuses[sbdDev] = SBD_STATUS_HEALTHY
		}
	}

	return sbdStatuses
}

// for each sbd device, extract the watchdog and msgwait timeout via regex
func (c *SbdCollector) getSbdTimeouts(sbdDevices []string) (map[string]float64, map[string]float64) {
	sbdWatchdogs := make(map[string]float64)
	sbdMsgWaits := make(map[string]float64)
	for _, sbdDev := range sbdDevices {
		sbdDump, _ := utils.RunCommand(c.sbdPath, "-d", sbdDev, "dump")

		regexW := regexp.MustCompile(`Timeout \(msgwait\)  *: \d+`)
		regex := regexp.MustCompile(`Timeout \(watchdog\)  *: \d+`)

		msgWaitLine := regexW.FindStringSubmatch(string(sbdDump))
		watchdogLine := regex.FindStringSubmatch(string(sbdDump))

		if watchdogLine == nil || msgWaitLine == nil {
			continue
		}

		// get the timeout from the line
		regexNumber := regexp.MustCompile(`\d+`)
		watchdogTimeout := regexNumber.FindString(string(watchdogLine[0]))
		msgWaitTimeout := regexNumber.FindString(string(msgWaitLine[0]))

		// map the timeout to the device
		if s, err := strconv.ParseFloat(watchdogTimeout, 64); err == nil {
			sbdWatchdogs[sbdDev] = s
		}

		// map the timeout to the device
		if s, err := strconv.ParseFloat(msgWaitTimeout, 64); err == nil {
			sbdMsgWaits[sbdDev] = s
		}

	}
	return sbdWatchdogs, sbdMsgWaits
}
// Part 2 commit for hacluster_exporter/internal/metrics/collectors/sbd/sbd.go
