package metrics

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"node_utility_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	subsystemName          = "selinux"
	selinuxConfigPath      = "/etc/selinux/config"
	selinuxEnforceFilePath = "/sys/fs/selinux/enforce"
	statusRefreshInterval  = 5 * time.Minute
	defaultTimeout         = 2 * time.Second
)

var (
	ErrSELinuxNotSupported = errors.New("SELinux not supported on this system")
	ErrStatusReadFailed    = errors.New("failed to read SELinux status")
	ErrConfigReadFailed    = errors.New("failed to read SELinux configuration")
)

type SELinuxStatus int

const (
	StatusUnknown SELinuxStatus = iota
	StatusDisabled
	StatusEnabled
)

type SELinuxMode int

const (
	ModeUnknown SELinuxMode = iota
	ModeDisabled
	ModePermissive
	ModeEnforcing
)

type SELinuxState struct {
	Status      SELinuxStatus
	ConfigMode  SELinuxMode
	CurrentMode SELinuxMode
	LastUpdated time.Time
}

type selinuxCollector struct {
	configModeDesc  *prometheus.Desc
	currentModeDesc *prometheus.Desc
	enabledDesc     *prometheus.Desc
	logger          *slog.Logger
	state           SELinuxState
	stateMutex      sync.RWMutex
	lastChecked     time.Time
}

func init() {
	collectorFactory := func() (Collector, error) {
		return NewSelinuxCollector()
	}

	// 注册收集器
	registerSELinuxCollector(collectorFactory)
}

func registerSELinuxCollector(factory func() (Collector, error)) {
	collector, err := factory()
	if err != nil {
		panic(fmt.Sprintf("failed to create SELinux collector: %v", err))
	}

	if metricCollector, ok := collector.(prometheus.Collector); ok {
		exporter.Register(metricCollector)
	} else {
		panic("SELinux collector does not implement prometheus.Collector")
	}
}

func NewSelinuxCollector() (Collector, error) {
	collector := &selinuxCollector{
		lastChecked: time.Now(),
	}

	collector.initializeDescriptors()

	collector.initializeLogger()

	if err := collector.loadSELinuxState(); err != nil {
		collector.logger.Warn("Initial SELinux state load failed", "error", err)
	}

	return collector, nil
}

func (c *selinuxCollector) initializeDescriptors() {
	c.configModeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystemName, "config_mode"),
		"Configured SELinux enforcement mode (0: unknown, 1: disabled, 2: permissive, 3: enforcing)",
		nil, nil,
	)

	c.currentModeDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystemName, "current_mode"),
		"Current SELinux enforcement mode (0: unknown, 1: disabled, 2: permissive, 3: enforcing)",
		nil, nil,
	)

	c.enabledDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, subsystemName, "enabled"),
		"SELinux status (0: unknown, 1: disabled, 2: enabled)",
		nil, nil,
	)
}

func (c *selinuxCollector) initializeLogger() {
	c.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

func (c *selinuxCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.configModeDesc
	ch <- c.currentModeDesc
	ch <- c.enabledDesc
}

func (c *selinuxCollector) Collect(ch chan<- prometheus.Metric) {
	if time.Since(c.lastChecked) > statusRefreshInterval {
		if err := c.refreshSELinuxState(); err != nil {
			c.logger.Error("Failed to refresh SELinux state", "error", err)
		}
	}

	if err := c.Update(ch); err != nil {
		c.logger.Error("Failed to update SELinux metrics", "error", err)
	}
}

func (c *selinuxCollector) Update(ch chan<- prometheus.Metric) error {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()

	ch <- prometheus.MustNewConstMetric(
		c.enabledDesc, prometheus.GaugeValue, float64(c.state.Status),
	)

	ch <- prometheus.MustNewConstMetric(
		c.configModeDesc, prometheus.GaugeValue, float64(c.state.ConfigMode),
	)

	ch <- prometheus.MustNewConstMetric(
		c.currentModeDesc, prometheus.GaugeValue, float64(c.state.CurrentMode),
	)

	return nil
}

func (c *selinuxCollector) loadSELinuxState() error {
	c.logger.Info("Loading SELinux state")

	if !c.isSELinuxSupported() {
		c.setState(SELinuxState{
			Status:      StatusDisabled,
			ConfigMode:  ModeDisabled,
			CurrentMode: ModeDisabled,
		})
		c.logger.Info("SELinux not supported on this system")
		return ErrSELinuxNotSupported
	}

	status, err := c.getSELinuxStatus()
	if err != nil {
		c.logger.Error("Failed to get SELinux status", "error", err)
		return err
	}

	configMode, err := c.getConfigMode()
	if err != nil {
		c.logger.Warn("Failed to get SELinux config mode", "error", err)
	}

	currentMode, err := c.getCurrentMode()
	if err != nil {
		c.logger.Warn("Failed to get current SELinux mode", "error", err)
	}

	c.setState(SELinuxState{
		Status:      status,
		ConfigMode:  configMode,
		CurrentMode: currentMode,
		LastUpdated: time.Now(),
	})

	c.logger.Info("SELinux state loaded",
		"status", status.String(),
		"config_mode", configMode.String(),
		"current_mode", currentMode.String())

	return nil
}

func (c *selinuxCollector) refreshSELinuxState() error {
	c.logger.Debug("Refreshing SELinux state")
	return c.loadSELinuxState()
}

func (c *selinuxCollector) setState(state SELinuxState) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	c.state = state
	c.lastChecked = time.Now()
}

func (c *selinuxCollector) isSELinuxSupported() bool {
	if _, err := os.Stat("/sys/fs/selinux"); os.IsNotExist(err) {
		return false
	}

	if c.checkKernelConfig() {
		return true
	}

	if c.checkSELinuxCommands() {
		return true
	}

	return false
}

func (c *selinuxCollector) checkKernelConfig() bool {
	configPaths := []string{
		"/boot/config",
		"/boot/config-$(uname -r)",
		"/proc/config.gz",
	}

	for _, path := range configPaths {
		if c.checkKernelConfigFile(path) {
			return true
		}
	}

	return false
}

func (c *selinuxCollector) checkKernelConfigFile(path string) bool {
	if strings.Contains(path, "$(uname -r)") {
		unameCmd := exec.Command("uname", "-r")
		output, err := unameCmd.Output()
		if err != nil {
			return false
		}
		kernelVersion := strings.TrimSpace(string(output))
		path = strings.Replace(path, "$(uname -r)", kernelVersion, 1)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	var reader io.Reader
	if strings.HasSuffix(path, ".gz") {
		cmd := exec.Command("zcat", path)
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		reader = bytes.NewReader(output)
	} else {
		cleanPath := filepath.Clean(path)
		if !strings.HasPrefix(cleanPath, "/boot") {
			return false
		}
		file, err := os.Open(path)
		if err != nil {
			return false
		}
		defer file.Close()
		reader = file
	}

	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "CONFIG_SECURITY_SELINUX=") {
			return strings.Contains(line, "y") || strings.Contains(line, "m")
		}
	}

	return false
}

func (c *selinuxCollector) checkSELinuxCommands() bool {
	commands := []string{"sestatus", "getenforce", "setenforce", "selinuxenabled"}

	for _, cmd := range commands {
		if _, err := exec.LookPath(cmd); err == nil {
			return true
		}
	}

	return false
}

func (c *selinuxCollector) getSELinuxStatus() (SELinuxStatus, error) {
	if status, err := c.getStatusFromCommand(); err == nil {
		return status, nil
	}

	if status, err := c.getStatusFromFile(); err == nil {
		return status, nil
	}

	if status, err := c.getStatusFromSestatus(); err == nil {
		return status, nil
	}

	return StatusUnknown, ErrStatusReadFailed
}

func (c *selinuxCollector) getStatusFromCommand() (SELinuxStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "selinuxenabled")
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == 1 {
				return StatusDisabled, nil
			}
		}
		return StatusUnknown, err
	}
	return StatusEnabled, nil
}

func (c *selinuxCollector) getStatusFromFile() (SELinuxStatus, error) {
	if _, err := os.Stat(selinuxEnforceFilePath); err == nil {
		return StatusEnabled, nil
	}
	return StatusDisabled, nil
}

func (c *selinuxCollector) getStatusFromSestatus() (SELinuxStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sestatus")
	output, err := cmd.Output()
	if err != nil {
		return StatusUnknown, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "SELinux status") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				status := strings.TrimSpace(parts[1])
				if strings.EqualFold(status, "enabled") {
					return StatusEnabled, nil
				} else {
					return StatusDisabled, nil
				}
			}
		}
	}

	return StatusUnknown, errors.New("SELinux status not found in sestatus output")
}

func (c *selinuxCollector) getConfigMode() (SELinuxMode, error) {
	if mode, err := c.getConfigModeFromFile(); err == nil {
		return mode, nil
	}

	if mode, err := c.getConfigModeFromSestatus(); err == nil {
		return mode, nil
	}

	if mode, err := c.getConfigModeFromCommand(); err == nil {
		return mode, nil
	}

	return ModeUnknown, ErrConfigReadFailed
}

func (c *selinuxCollector) getConfigModeFromFile() (SELinuxMode, error) {
	file, err := os.Open(selinuxConfigPath)
	if err != nil {
		return ModeUnknown, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "SELINUX=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				mode := strings.TrimSpace(strings.ToLower(parts[1]))
				switch mode {
				case "enforcing":
					return ModeEnforcing, nil
				case "permissive":
					return ModePermissive, nil
				case "disabled":
					return ModeDisabled, nil
				}
			}
		}
	}

	return ModeUnknown, errors.New("SELINUX mode not found in config file")
}

func (c *selinuxCollector) getConfigModeFromSestatus() (SELinuxMode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sestatus")
	output, err := cmd.Output()
	if err != nil {
		return ModeUnknown, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Mode from config file") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				mode := strings.TrimSpace(parts[1])
				return parseModeString(mode)
			}
		}
	}

	return ModeUnknown, errors.New("config mode not found in sestatus output")
}

func (c *selinuxCollector) getConfigModeFromCommand() (SELinuxMode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "getenforce", "-c")
	output, err := cmd.Output()
	if err != nil {
		return ModeUnknown, err
	}

	modeStr := strings.TrimSpace(string(output))
	return parseModeString(modeStr)
}

func (c *selinuxCollector) getCurrentMode() (SELinuxMode, error) {
	if mode, err := c.getCurrentModeFromFile(); err == nil {
		return mode, nil
	}

	if mode, err := c.getCurrentModeFromCommand(); err == nil {
		return mode, nil
	}

	if mode, err := c.getCurrentModeFromSestatus(); err == nil {
		return mode, nil
	}

	return ModeUnknown, errors.New("failed to determine current SELinux mode")
}

func (c *selinuxCollector) getCurrentModeFromFile() (SELinuxMode, error) {
	data, err := os.ReadFile(selinuxEnforceFilePath)
	if err != nil {
		return ModeUnknown, err
	}

	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return ModeUnknown, err
	}

	switch value {
	case 0:
		return ModePermissive, nil
	case 1:
		return ModeEnforcing, nil
	default:
		return ModeUnknown, fmt.Errorf("invalid enforce value: %d", value)
	}
}

func (c *selinuxCollector) getCurrentModeFromCommand() (SELinuxMode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "getenforce")
	output, err := cmd.Output()
	if err != nil {
		return ModeUnknown, err
	}

	modeStr := strings.TrimSpace(string(output))
	return parseModeString(modeStr)
}

func (c *selinuxCollector) getCurrentModeFromSestatus() (SELinuxMode, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sestatus")
	output, err := cmd.Output()
	if err != nil {
		return ModeUnknown, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Current mode") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				mode := strings.TrimSpace(parts[1])
				return parseModeString(mode)
			}
		}
	}

	return ModeUnknown, errors.New("current mode not found in sestatus output")
}

func parseModeString(modeStr string) (SELinuxMode, error) {
	switch strings.ToLower(modeStr) {
	case "enforcing":
		return ModeEnforcing, nil
	case "permissive":
		return ModePermissive, nil
	case "disabled":
		return ModeDisabled, nil
	default:
		return ModeUnknown, fmt.Errorf("unknown mode: %s", modeStr)
	}
}

func (s SELinuxStatus) String() string {
	switch s {
	case StatusDisabled:
		return "disabled"
	case StatusEnabled:
		return "enabled"
	default:
		return "unknown"
	}
}

func (m SELinuxMode) String() string {
	switch m {
	case ModeDisabled:
		return "disabled"
	case ModePermissive:
		return "permissive"
	case ModeEnforcing:
		return "enforcing"
	default:
		return "unknown"
	}
}

func (c *selinuxCollector) GetSELinuxStatus() SELinuxStatus {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state.Status
}

func (c *selinuxCollector) GetConfigMode() SELinuxMode {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state.ConfigMode
}

func (c *selinuxCollector) GetCurrentMode() SELinuxMode {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	return c.state.CurrentMode
}
