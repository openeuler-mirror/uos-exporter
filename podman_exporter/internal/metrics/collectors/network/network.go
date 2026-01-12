package network

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
	subsystem = "network"
)

// Collector 定义网络收集器结构
type Collector struct {
	parser  Parser
	logger  *logrus.Logger
	timeout time.Duration

	// 指标描述符
	infoDesc *prometheus.Desc
}

// NewCollector 创建一个新的网络收集器实例
func NewCollector(logger *logrus.Logger, timeout time.Duration) *Collector {
	return &Collector{
		parser:  NewParser(),
		logger:  logger,
		timeout: timeout,

		// 网络信息指标 - 包含driver、id、interface、labels、name标签
		infoDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "info"),
			"Network information",
			[]string{"driver", "id", "interface", "labels", "name"},
			nil,
		),
	}
}

// Describe 实现prometheus.Collector接口，发送所有可能的描述符到channel
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc
}

// Collect 实现prometheus.Collector接口，收集并发送指标数据
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if err := c.CollectWithError(ch); err != nil {
		c.logger.WithError(err).Error("Failed to collect network metrics")
	}
}

// CollectWithError 收集指标数据，返回错误而不是记录日志
func (c *Collector) CollectWithError(ch chan<- prometheus.Metric) error {
	// 获取网络状态信息
	status, err := c.getNetworkStatus()
	if err != nil {
		return errors.Wrap(err, "failed to get network status")
	}

	// 收集网络信息指标
	c.collectInfoMetrics(ch, status)

	return nil
}

// getNetworkStatus 获取网络状态信息
func (c *Collector) getNetworkStatus() (*Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 执行podman network ls命令获取JSON格式输出
	cmd := exec.CommandContext(ctx, "podman", "network", "ls", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute podman network ls command")
	}

	// 解析输出
	status, err := c.parser.Parse(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse network output")
	}

	c.logger.WithField("networks_count", len(status.Networks)).Debug("Successfully parsed network data")
	return status, nil
}

// collectInfoMetrics 收集网络信息指标
func (c *Collector) collectInfoMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, network := range status.Networks {
		ch <- prometheus.MustNewConstMetric(
			c.infoDesc,
			prometheus.GaugeValue,
			1,
			network.Driver,
			c.truncateID(network.ID),
			network.Interface,
			network.Labels,
			network.Name,
		)
	}
}

// truncateID 截断网络ID到前12位，与其他收集器保持一致
func (c *Collector) truncateID(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

// Name 返回收集器名称
func (c *Collector) Name() string {
	return fmt.Sprintf("%s_%s", namespace, subsystem)
}
