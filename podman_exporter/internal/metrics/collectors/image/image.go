package image

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
	subsystem = "image"
)

// Collector 定义镜像收集器结构
type Collector struct {
	parser  Parser
	logger  *logrus.Logger
	timeout time.Duration

	// 指标描述符
	infoDesc    *prometheus.Desc
	sizeDesc    *prometheus.Desc
	createdDesc *prometheus.Desc
}

// NewCollector 创建一个新的镜像收集器实例
func NewCollector(logger *logrus.Logger, timeout time.Duration) *Collector {
	return &Collector{
		parser:  NewParser(),
		logger:  logger,
		timeout: timeout,

		// 镜像信息指标 - 包含ID、repository、tag、digest等标签
		infoDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "info"),
			"Image information",
			[]string{"id", "repository", "tag", "digest", "parent_id"},
			nil,
		),

		// 镜像大小指标
		sizeDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "size"),
			"Image size",
			[]string{"id", "repository", "tag"},
			nil,
		),

		// 镜像创建时间
		createdDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "created_seconds"),
			"Image creation time in seconds since epoch",
			[]string{"id", "repository", "tag"},
			nil,
		),
	}
}

// Describe 实现prometheus.Collector接口，发送所有可能的描述符到channel
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.infoDesc
	ch <- c.sizeDesc
	ch <- c.createdDesc
}

// Collect 实现prometheus.Collector接口，收集并发送指标数据
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if err := c.CollectWithError(ch); err != nil {
		c.logger.WithError(err).Error("Failed to collect image metrics")
	}
}

// CollectWithError 收集指标数据，返回错误而不是记录日志
func (c *Collector) CollectWithError(ch chan<- prometheus.Metric) error {
	// 获取镜像状态信息
	status, err := c.getImageStatus()
	if err != nil {
		return errors.Wrap(err, "failed to get image status")
	}

	// 收集各种指标
	c.collectInfoMetrics(ch, status)
	c.collectSizeMetrics(ch, status)
	c.collectCreatedMetrics(ch, status)

	return nil
}

// getImageStatus 获取镜像状态信息
func (c *Collector) getImageStatus() (*Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 执行podman images命令获取JSON格式输出
	cmd := exec.CommandContext(ctx, "podman", "images", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute podman images command")
	}

	// 解析输出
	status, err := c.parser.Parse(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse image output")
	}

	c.logger.WithField("images_count", len(status.Images)).Debug("Successfully parsed image data")
	return status, nil
}

// collectInfoMetrics 收集镜像信息指标
func (c *Collector) collectInfoMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, image := range status.Images {
		// 为每个镜像创建info指标，值固定为1
		ch <- prometheus.MustNewConstMetric(
			c.infoDesc,
			prometheus.GaugeValue,
			1,
			c.truncateID(image.ID),
			image.Repository,
			image.Tag,
			image.Digest,
			c.truncateID(image.ParentID),
		)
	}
}

// collectSizeMetrics 收集镜像大小指标
func (c *Collector) collectSizeMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, image := range status.Images {
		ch <- prometheus.MustNewConstMetric(
			c.sizeDesc,
			prometheus.GaugeValue,
			float64(image.Size),
			c.truncateID(image.ID),
			image.Repository,
			image.Tag,
		)
	}
}

// collectCreatedMetrics 收集镜像创建时间指标
func (c *Collector) collectCreatedMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, image := range status.Images {
		var createdSeconds float64
		if !image.Created.IsZero() {
			createdSeconds = float64(image.Created.Unix())
		}

		ch <- prometheus.MustNewConstMetric(
			c.createdDesc,
			prometheus.GaugeValue,
			createdSeconds,
			c.truncateID(image.ID),
			image.Repository,
			image.Tag,
		)
	}
}

// truncateID 截断镜像ID到前12位，与容器包保持一致
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
// Part 2 commit for podman_exporter/internal/metrics/collectors/image/image.go
