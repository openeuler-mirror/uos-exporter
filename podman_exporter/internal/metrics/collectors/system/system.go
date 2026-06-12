package system

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	// Prometheus namespace
	namespace = "podman"
	subsystem = "system"
)

// Collector 定义系统收集器结构
type Collector struct {
	parser  Parser
	logger  *logrus.Logger
	timeout time.Duration

	// 指标描述符
	apiVersionDesc     *prometheus.Desc
	buildahVersionDesc *prometheus.Desc
	conmonVersionDesc  *prometheus.Desc
	runtimeVersionDesc *prometheus.Desc
}

// NewCollector 创建一个新的系统收集器实例
func NewCollector(logger *logrus.Logger, timeout time.Duration) *Collector {
	return &Collector{
		parser:  NewParser(),
		logger:  logger,
		timeout: timeout,

		// API版本指标
		apiVersionDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "api_version"),
			"Podman system api version",
			[]string{"version"},
			nil,
		),

		// Buildah版本指标
		buildahVersionDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "buildah_version"),
			"Podman system buildahVer version",
			[]string{"version"},
			nil,
		),

		// Conmon版本指标
		conmonVersionDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "conmon_version"),
			"Podman system conmon version",
			[]string{"version"},
			nil,
		),

		// 运行时版本指标
		runtimeVersionDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "runtime_version"),
			"Podman system runtime version",
			[]string{"version"},
			nil,
		),
	}
}

// Describe 实现prometheus.Collector接口，发送所有可能的描述符到channel
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.apiVersionDesc
	ch <- c.buildahVersionDesc
	ch <- c.conmonVersionDesc
	ch <- c.runtimeVersionDesc
}

// Collect 实现prometheus.Collector接口，收集并发送指标数据
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if err := c.CollectWithError(ch); err != nil {
		c.logger.WithError(err).Error("Failed to collect system metrics")
	}
}

// CollectWithError 收集指标数据，返回错误而不是记录日志
func (c *Collector) CollectWithError(ch chan<- prometheus.Metric) error {
	// 获取系统信息
	status, err := c.getSystemStatus()
	if err != nil {
		return errors.Wrap(err, "failed to get system status")
	}

	// 收集各种指标
	c.collectVersionMetrics(ch, status)

	return nil
}

// getSystemStatus 获取系统状态信息
func (c *Collector) getSystemStatus() (*Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 执行podman system info命令获取JSON格式输出
	cmd := exec.CommandContext(ctx, "podman", "system", "info", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute podman system info command")
	}

	// 解析输出
	status, err := c.parser.Parse(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse system output")
	}

	c.logger.Debug("Successfully parsed system data")
	return status, nil
}

// collectVersionMetrics 收集版本信息指标
func (c *Collector) collectVersionMetrics(ch chan<- prometheus.Metric, status *Status) {
	// API版本
	if status.APIVersion != "" {
		ch <- prometheus.MustNewConstMetric(
			c.apiVersionDesc,
			prometheus.GaugeValue,
			1,
			status.APIVersion,
		)
	}

	// Buildah版本
	if status.BuildahVersion != "" {
		ch <- prometheus.MustNewConstMetric(
			c.buildahVersionDesc,
			prometheus.GaugeValue,
			1,
			status.BuildahVersion,
		)
	}

	// Conmon版本
	if status.ConmonVersion != "" {
		ch <- prometheus.MustNewConstMetric(
			c.conmonVersionDesc,
			prometheus.GaugeValue,
			1,
			c.extractConmonVersion(status.ConmonVersion),
		)
	}

	// 运行时版本
	if status.RuntimeVersion != "" {
		ch <- prometheus.MustNewConstMetric(
			c.runtimeVersionDesc,
			prometheus.GaugeValue,
			1,
			c.extractRuntimeVersion(status.RuntimeVersion),
		)
	}
}

// extractConmonVersion 从conmon版本字符串中提取版本号
func (c *Collector) extractConmonVersion(fullVersion string) string {
	// "conmon version 2.1.10, commit: unknown" -> "2.1.10"
	if strings.HasPrefix(fullVersion, "conmon version ") {
		version := strings.TrimPrefix(fullVersion, "conmon version ")
		if idx := strings.Index(version, ","); idx > 0 {
			return version[:idx]
		}
		return version
	}
	return fullVersion
}

// extractRuntimeVersion 从运行时版本字符串中提取版本号
func (c *Collector) extractRuntimeVersion(fullVersion string) string {
	// "crun version 1.8.7\ncommit: ..." -> "crun version 1.8.7"
	lines := strings.Split(fullVersion, "\n")
	if len(lines) > 0 {
		return lines[0]
	}
	return fullVersion
}

// Name 返回收集器名称
func (c *Collector) Name() string {
	return fmt.Sprintf("%s_%s", namespace, subsystem)
}
