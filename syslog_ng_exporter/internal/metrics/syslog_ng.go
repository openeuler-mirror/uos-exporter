package metrics

import (
	"bufio"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"syslog_ng_exporter/config"
	"syslog_ng_exporter/internal/exporter"
	"syslog_ng_exporter/pkg/utils"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	namespace = "syslog_ng" // For Prometheus metrics.
)

var (
	socketPath = "/var/lib/syslog-ng/syslog-ng.ctl" // 默认路径
)

func init() {
	exporter.Register(NewSyslogNGSourceProcessed("syslog_ng_source_messages_processed_total",
		"Number of messages processed by this source.",
		[]string{"type", "id", "source"}))

	exporter.Register(NewSyslogNGDestProcessed("syslog_ng_destination_messages_processed_total",
		"Number of messages processed by this destination.",
		[]string{"type", "id", "destination"}))

	exporter.Register(NewSyslogNGDestDropped("syslog_ng_destination_messages_dropped_total",
		"Number of messages dropped by this destination due to store overflow.",
		[]string{"type", "id", "destination"}))

	exporter.Register(NewSyslogNGDestStored("syslog_ng_destination_messages_stored_total",
		"Number of messages currently stored for this destination.",
		[]string{"type", "id", "destination"}))

	exporter.Register(NewSyslogNGDestWritten("syslog_ng_destination_messages_written_total",
		"Number of messages successfully written by this destination.",
		[]string{"type", "id", "destination"}))

	exporter.Register(NewSyslogNGDestMemory("syslog_ng_destination_bytes_stored_total",
		"Bytes of memory currently used to store messages for this destination.",
		[]string{"type", "id", "destination"}))

	exporter.Register(NewSyslogNGUp("syslog_ng_up",
		"Reads 1 if the syslog-ng server could be reached, else 0.",
		[]string{}))
}

// Stat 结构体用于解析 syslog-ng 提供的统计信息
type Stat struct {
	objectType string
	id         string
	instance   string
	state      string
	metric     string
	value      float64
}

// SyslogNGCollector 是所有 syslog-ng 指标收集器的基础结构
type SyslogNGCollector struct {
	*baseMetrics
	mutex    sync.Mutex
	logger   log.Logger
	sockPath string
}

// SyslogNGSourceProcessed 源处理的消息数量
type SyslogNGSourceProcessed struct {
	*SyslogNGCollector
}

// SyslogNGDestProcessed 目标处理的消息数量
type SyslogNGDestProcessed struct {
	*SyslogNGCollector
}

// SyslogNGDestDropped 目标丢弃的消息数量
type SyslogNGDestDropped struct {
	*SyslogNGCollector
}

// SyslogNGDestStored 目标存储的消息数量
type SyslogNGDestStored struct {
	*SyslogNGCollector
}

// SyslogNGDestWritten 目标成功写入的消息数量
type SyslogNGDestWritten struct {
	*SyslogNGCollector
}

// SyslogNGDestMemory 目标使用的内存量
type SyslogNGDestMemory struct {
	*SyslogNGCollector
}

// SyslogNGUp syslog-ng 是否可访问
type SyslogNGUp struct {
	*SyslogNGCollector
}

// 创建一个函数来获取 socketPath，从命令行参数或配置文件中读取
func GetSocketPath() string {
	// 从命令行参数获取
	if config.SocketPath != nil && *config.SocketPath != "" {
		logrus.Debugf("Using socket path from command line: %s", *config.SocketPath)
		return *config.SocketPath
	}

	// 尝试从 CommonConfig 中获取（如果 Server 已创建）
	configPath := socketPath

	// 检查默认配置文件
	cfgPath := "/etc/uos-exporter/syslog-ng-exporter.yaml"
	if utils.FileExists(cfgPath) {
		logrus.Debugf("Found default config file at %s, attempting to read socket path", cfgPath)
		// 这里可以尝试读取配置文件获取 socket path
	}

	// 使用默认路径
	logrus.Debugf("Using default socket path: %s", configPath)
	return configPath
}

// NewSyslogNGCollector 创建一个新的 SyslogNGCollector
func NewSyslogNGCollector(fqname, help string, labels []string) *SyslogNGCollector {
	path := GetSocketPath()
	logrus.Debugf("Creating SyslogNGCollector with socket path: %s", path)
	return &SyslogNGCollector{
		baseMetrics: NewMetrics(fqname, help, labels),
		sockPath:    path,
		logger:      log.NewNopLogger(),
	}
}

// NewSyslogNGSourceProcessed 创建一个新的 SyslogNGSourceProcessed 收集器
func NewSyslogNGSourceProcessed(fqname, help string, labels []string) *SyslogNGSourceProcessed {
	return &SyslogNGSourceProcessed{NewSyslogNGCollector(fqname, help, labels)}
}

// NewSyslogNGDestProcessed 创建一个新的 SyslogNGDestProcessed 收集器
func NewSyslogNGDestProcessed(fqname, help string, labels []string) *SyslogNGDestProcessed {
	return &SyslogNGDestProcessed{NewSyslogNGCollector(fqname, help, labels)}
}

// NewSyslogNGDestDropped 创建一个新的 SyslogNGDestDropped 收集器
func NewSyslogNGDestDropped(fqname, help string, labels []string) *SyslogNGDestDropped {
	return &SyslogNGDestDropped{NewSyslogNGCollector(fqname, help, labels)}
}

// NewSyslogNGDestStored 创建一个新的 SyslogNGDestStored 收集器
func NewSyslogNGDestStored(fqname, help string, labels []string) *SyslogNGDestStored {
	return &SyslogNGDestStored{NewSyslogNGCollector(fqname, help, labels)}
}

// NewSyslogNGDestWritten 创建一个新的 SyslogNGDestWritten 收集器
func NewSyslogNGDestWritten(fqname, help string, labels []string) *SyslogNGDestWritten {
	return &SyslogNGDestWritten{NewSyslogNGCollector(fqname, help, labels)}
}

// NewSyslogNGDestMemory 创建一个新的 SyslogNGDestMemory 收集器
func NewSyslogNGDestMemory(fqname, help string, labels []string) *SyslogNGDestMemory {
	return &SyslogNGDestMemory{NewSyslogNGCollector(fqname, help, labels)}
}

// NewSyslogNGUp 创建一个新的 SyslogNGUp 收集器
func NewSyslogNGUp(fqname, help string, labels []string) *SyslogNGUp {
	return &SyslogNGUp{NewSyslogNGCollector(fqname, help, labels)}
}

// parseStatLine 解析一行统计数据
func parseStatLine(line string) (Stat, error) {
	part := strings.SplitN(strings.TrimSpace(line), ";", 6)
	if len(part) < 6 {
		return Stat{}, fmt.Errorf("insufficient parts: %d < 6", len(part))
	}

	if len(part[0]) < 4 {
		return Stat{}, fmt.Errorf("invalid name: %s", part[0])
	}

	val, err := strconv.ParseFloat(part[5], 64)
	if err != nil {
		return Stat{}, err
	}

	stat := Stat{part[0], part[1], part[2], part[3], part[4], val}

	return stat, nil
}

// collectStats 收集 syslog-ng 的统计信息
func (c *SyslogNGCollector) collectStats() ([]Stat, bool, error) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	conn, err := net.Dial("unix", c.sockPath)
	if err != nil {
		_ = level.Error(c.logger).Log("msg", "Error connecting to syslog-ng", "err", err)
		return nil, false, fmt.Errorf("Error connecting to syslog-ng: %v", err)
	}

	defer conn.Close()

	err = conn.SetDeadline(time.Now().Add(time.Second))
	if err != nil {
		_ = level.Error(c.logger).Log("msg", "Failed to set conn deadline", "err", err)
		return nil, false, fmt.Errorf("Failed to set conn deadline: %v", err)
	}

	_, err = conn.Write([]byte("STATS\n"))
	if err != nil {
		_ = level.Error(c.logger).Log("msg", "Error writing to control socket", "err", err)
		return nil, false, fmt.Errorf("Error writing to control socket: %v", err)
	}

	buff := bufio.NewReader(conn)

	_, err = buff.ReadString('\n')
	if err != nil {
		_ = level.Error(c.logger).Log("msg", "Error reading header from control socket", "err", err)
		return nil, false, fmt.Errorf("Error reading header from control socket: %v", err)
	}

	var stats []Stat

	for {
		line, err := buff.ReadString('\n')

		if err != nil || line[0] == '.' {
			_ = level.Debug(c.logger).Log("msg", "Reached end of STATS output")
			break
		}

		stat, err := parseStatLine(line)
		if err != nil {
			_ = level.Debug(c.logger).Log("msg", "Skipping STATS line", "err", err)
			continue
		}

		_ = level.Debug(c.logger).Log("msg", "Successfully parsed STATS line", "stat", fmt.Sprintf("%v", stat))
		stats = append(stats, stat)
	}

	return stats, true, nil
}

// Describe 实现 SyslogNGCollector 的 Describe 方法
func (c *SyslogNGCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.baseMetrics.desc
}

// Describe 实现 SyslogNGSourceProcessed 的 Describe 方法
func (s *SyslogNGSourceProcessed) Describe(ch chan<- *prometheus.Desc) {
	ch <- s.baseMetrics.desc
}

// Describe 实现 SyslogNGDestProcessed 的 Describe 方法
func (d *SyslogNGDestProcessed) Describe(ch chan<- *prometheus.Desc) {
	ch <- d.baseMetrics.desc
}

// Describe 实现 SyslogNGDestDropped 的 Describe 方法
func (d *SyslogNGDestDropped) Describe(ch chan<- *prometheus.Desc) {
	ch <- d.baseMetrics.desc
}

// Describe 实现 SyslogNGDestStored 的 Describe 方法
func (d *SyslogNGDestStored) Describe(ch chan<- *prometheus.Desc) {
	ch <- d.baseMetrics.desc
}

// Describe 实现 SyslogNGDestWritten 的 Describe 方法
func (d *SyslogNGDestWritten) Describe(ch chan<- *prometheus.Desc) {
	ch <- d.baseMetrics.desc
}

// Describe 实现 SyslogNGDestMemory 的 Describe 方法
func (d *SyslogNGDestMemory) Describe(ch chan<- *prometheus.Desc) {
	ch <- d.baseMetrics.desc
}

// Describe 实现 SyslogNGUp 的 Describe 方法
func (u *SyslogNGUp) Describe(ch chan<- *prometheus.Desc) {
	ch <- u.baseMetrics.desc
}

// Collect 实现 SyslogNGSourceProcessed 的 Collect 方法
func (s *SyslogNGSourceProcessed) Collect(ch chan<- prometheus.Metric) {
	stats, _, err := s.collectStats()
	if err != nil {
		_ = level.Error(s.logger).Log("msg", "Error collecting syslog-ng stats", "err", err)
		return
	}

	for _, stat := range stats {
		if stat.objectType[0:4] == "src." && stat.metric == "processed" {
			s.baseMetrics.collect(ch, stat.value, []string{stat.objectType, stat.id, stat.instance})
		}
	}
}

// Collect 实现 SyslogNGDestProcessed 的 Collect 方法
func (d *SyslogNGDestProcessed) Collect(ch chan<- prometheus.Metric) {
	stats, _, err := d.collectStats()
	if err != nil {
		_ = level.Error(d.logger).Log("msg", "Error collecting syslog-ng stats", "err", err)
		return
	}

	for _, stat := range stats {
		if stat.objectType[0:4] == "dst." && stat.metric == "processed" {
			d.baseMetrics.collect(ch, stat.value, []string{stat.objectType, stat.id, stat.instance})
		}
	}
}

// Collect 实现 SyslogNGDestDropped 的 Collect 方法
func (d *SyslogNGDestDropped) Collect(ch chan<- prometheus.Metric) {
	stats, _, err := d.collectStats()
	if err != nil {
		_ = level.Error(d.logger).Log("msg", "Error collecting syslog-ng stats", "err", err)
		return
	}

	for _, stat := range stats {
		if stat.objectType[0:4] == "dst." && stat.metric == "dropped" {
			d.baseMetrics.collect(ch, stat.value, []string{stat.objectType, stat.id, stat.instance})
		}
	}
}

// Collect 实现 SyslogNGDestStored 的 Collect 方法
func (d *SyslogNGDestStored) Collect(ch chan<- prometheus.Metric) {
	stats, _, err := d.collectStats()
	if err != nil {
		_ = level.Error(d.logger).Log("msg", "Error collecting syslog-ng stats", "err", err)
		return
	}

	for _, stat := range stats {
		if stat.objectType[0:4] == "dst." && (stat.metric == "stored" || stat.metric == "queued") {
			d.baseMetrics.collect(ch, stat.value, []string{stat.objectType, stat.id, stat.instance})
		}
	}
}

// Collect 实现 SyslogNGDestWritten 的 Collect 方法
func (d *SyslogNGDestWritten) Collect(ch chan<- prometheus.Metric) {
	stats, _, err := d.collectStats()
	if err != nil {
		_ = level.Error(d.logger).Log("msg", "Error collecting syslog-ng stats", "err", err)
		return
	}

	for _, stat := range stats {
		if stat.objectType[0:4] == "dst." && stat.metric == "written" {
			d.baseMetrics.collect(ch, stat.value, []string{stat.objectType, stat.id, stat.instance})
		}
	}
}

// Collect 实现 SyslogNGDestMemory 的 Collect 方法
func (d *SyslogNGDestMemory) Collect(ch chan<- prometheus.Metric) {
	stats, _, err := d.collectStats()
	if err != nil {
		_ = level.Error(d.logger).Log("msg", "Error collecting syslog-ng stats", "err", err)
		return
	}

	for _, stat := range stats {
		if stat.objectType[0:4] == "dst." && stat.metric == "memory_usage" {
			d.baseMetrics.collect(ch, stat.value, []string{stat.objectType, stat.id, stat.instance})
		}
	}
}

// Collect 实现 SyslogNGUp 的 Collect 方法
func (u *SyslogNGUp) Collect(ch chan<- prometheus.Metric) {
	_, up, err := u.collectStats()
	if err != nil {
		u.baseMetrics.collect(ch, 0, []string{})
		return
	}

	if up {
		u.baseMetrics.collect(ch, 1, []string{})
	} else {
		u.baseMetrics.collect(ch, 0, []string{})
	}
}
// Part 2 commit for syslog_ng_exporter/internal/metrics/syslog_ng.go
