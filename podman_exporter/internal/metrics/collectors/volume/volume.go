package volume

import (
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	// Prometheus namespace
	namespace = "podman"
	subsystem = "volume"
)

// Collector 定义存储卷收集器结构
type Collector struct {
	parser  Parser
	logger  *logrus.Logger
	timeout time.Duration

	// 指标描述符
	infoDesc    *prometheus.Desc
	createdDesc *prometheus.Desc
}

// NewCollector 创建一个新的存储卷收集器实例
func NewCollector(logger *logrus.Logger, timeout time.Duration) *Collector {
	return &Collector{
		parser:  NewParser(),
		logger:  logger,
		timeout: timeout,

		// 存储卷信息指标 - 包含driver、mount_point、name标签
		infoDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "info"),
			"Volume information",
			[]string{"driver", "mount_point", "name"},
			nil,
		),

		// 存储卷创建时间指标 - 包含name标签
		createdDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "created_seconds"),
			"Volume creation time in unixtime",
			[]string{"name"},
			nil,
		),
	}
}

// Describe 实现prometheus.Collector接口，发送所有可能的描述符到channel
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc
	ch <- c.createdDesc
}

// Collect 实现prometheus.Collector接口，收集并发送指标数据
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if err := c.CollectWithError(ch); err != nil {
		c.logger.WithError(err).Error("Failed to collect volume metrics")
	}
}

// CollectWithError 收集指标数据，返回错误而不是记录日志
func (c *Collector) CollectWithError(ch chan<- prometheus.Metric) error {
	// 获取存储卷状态信息
	status, err := c.getVolumeStatus()
	if err != nil {
		return errors.Wrap(err, "failed to get volume status")
	}

	// 收集各种指标
	c.collectInfoMetrics(ch, status)
	c.collectCreatedMetrics(ch, status)

	return nil
}

// getVolumeStatus 获取存储卷状态信息
func (c *Collector) getVolumeStatus() (*Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 执行podman volume ls命令获取JSON格式输出
	cmd := exec.CommandContext(ctx, "podman", "volume", "ls", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute podman volume ls command")
	}

	// 解析输出
	status, err := c.parser.Parse(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse volume output")
	}

	c.logger.WithField("volumes_count", len(status.Volumes)).Debug("Successfully parsed volume data")
	return status, nil
}

// collectInfoMetrics 收集存储卷信息指标
func (c *Collector) collectInfoMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, volume := range status.Volumes {
		ch <- prometheus.MustNewConstMetric(
			c.infoDesc,
			prometheus.GaugeValue,
			1,
			volume.Driver,
			volume.MountPoint,
			volume.Name,
		)
	}
}

// collectCreatedMetrics 收集存储卷创建时间指标
func (c *Collector) collectCreatedMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, volume := range status.Volumes {
		var createdSeconds float64
		if !volume.Created.IsZero() {
			createdSeconds = float64(volume.Created.Unix())
		}

		ch <- prometheus.MustNewConstMetric(
			c.createdDesc,
			prometheus.GaugeValue,
			createdSeconds,
			volume.Name,
		)
	}
}

// Name 返回收集器名称
func (c *Collector) Name() string {
	return fmt.Sprintf("%s_%s", namespace, subsystem)
}
// Part 2 commit for podman_exporter/internal/metrics/collectors/volume/volume.go
