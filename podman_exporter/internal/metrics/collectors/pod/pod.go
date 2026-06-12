package pod

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"podman_exporter/pkg/utils"
	"regexp"
	"time"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const (
	// Prometheus namespace
	namespace = "podman"
	subsystem = "pod"
)

// podmanIDPattern matches valid podman IDs (alphanumeric, typically hex)
var podmanIDPattern = regexp.MustCompile(`^[a-fA-F0-9]+$`)

// validatePodmanID validates that an ID is safe to use in command arguments
func validatePodmanID(id string) bool {
	if id == "" {
		return false
	}
	// Podman IDs are typically 12 or 64 character hex strings
	// Allow 1-128 characters to be flexible but still safe
	if len(id) > 128 {
		return false
	}
	return podmanIDPattern.MatchString(id)
}

// Collector 定义Pod收集器结构
type Collector struct {
	parser  Parser
	logger  *logrus.Logger
	timeout time.Duration

	// 指标描述符
	stateDesc      *prometheus.Desc
	infoDesc       *prometheus.Desc
	containersDesc *prometheus.Desc
	createdDesc    *prometheus.Desc
}

// NewCollector 创建一个新的Pod收集器实例
func NewCollector(logger *logrus.Logger, timeout time.Duration) *Collector {
	return &Collector{
		parser:  NewParser(),
		logger:  logger,
		timeout: timeout,

		// Pod状态指标 - 包含id标签
		stateDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "state"),
			"Pods current state (-1=unknown,0=created,1=error,2=exited,3=paused,4=running,5=degraded,6=stopped)",
			[]string{"id"},
			nil,
		),

		// Pod信息指标 - 包含id、infra_id、name标签
		infoDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "info"),
			"Pod information",
			[]string{"id", "infra_id", "name"},
			nil,
		),

		// Pod中容器数量指标 - 包含id标签
		containersDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "containers"),
			"Number of containers in a pod",
			[]string{"id"},
			nil,
		),

		// Pod创建时间指标 - 包含id标签
		createdDesc: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "created_seconds"),
			"Pods creation time in unixtime",
			[]string{"id"},
			nil,
		),
	}
}

// Describe 实现prometheus.Collector接口，发送所有可能的描述符到channel
func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.stateDesc
	ch <- c.infoDesc
	ch <- c.containersDesc
	ch <- c.createdDesc
}

// Collect 实现prometheus.Collector接口，收集并发送指标数据
func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if err := c.CollectWithError(ch); err != nil {
		c.logger.WithError(err).Error("Failed to collect pod metrics")
	}
}

// CollectWithError 收集指标数据，返回错误而不是记录日志
func (c *Collector) CollectWithError(ch chan<- prometheus.Metric) error {
	// 获取Pod状态信息
	status, err := c.getPodStatus()
	if err != nil {
		return errors.Wrap(err, "failed to get pod status")
	}

	// 收集各种指标
	c.collectStateMetrics(ch, status)
	c.collectInfoMetrics(ch, status)
	c.collectContainersMetrics(ch, status)
	c.collectCreatedMetrics(ch, status)

	return nil
}

// getPodStatus 获取Pod状态信息
func (c *Collector) getPodStatus() (*Status, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	// 执行podman pod ls命令获取JSON格式输出
	cmd := exec.CommandContext(ctx, "podman", "pod", "ls", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute podman pod ls command")
	}

	// 解析输出
	status, err := c.parser.Parse(output)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse pod output")
	}

	// 如果需要，获取每个pod的详细信息来填充infra_id和容器数量
	if err := c.enrichPodDetails(status); err != nil {
		c.logger.WithError(err).Warn("Failed to enrich pod details")
	}

	c.logger.WithField("pods_count", len(status.Pods)).Debug("Successfully parsed pod data")
	return status, nil
}

// enrichPodDetails 通过podman pod inspect命令获取Pod的详细信息
func (c *Collector) enrichPodDetails(status *Status) error {
	for i := range status.Pods {
		pod := &status.Pods[i]

		// 执行podman pod inspect命令获取详细信息
		ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
		// 校验pod.ID的名称是否合法
		if !validatePodmanID(pod.ID) {
			c.logger.WithField("pod_id", pod.ID).Warn("Invalid pod ID, skipping inspect")
			cancel()
			continue
		}

		cmd := utils.GetCommandCtx(ctx, "podman", "pod", "inspect", pod.ID)
		output, err := cmd.Output()
		cancel()

		if err != nil {
			c.logger.WithError(err).WithField("pod_id", pod.ID).Debug("Failed to inspect pod")
			continue
		}

		// 解析inspect输出来获取infra_id和容器数量
		if err := c.parseInspectOutput(output, pod); err != nil {
			c.logger.WithError(err).WithField("pod_id", pod.ID).Debug("Failed to parse pod inspect output")
		}
	}

	return nil
}

// parseInspectOutput 解析podman pod inspect的输出
func (c *Collector) parseInspectOutput(output []byte, pod *Pod) error {
	var data map[string]interface{}

	if err := json.Unmarshal(output, &data); err != nil {
		return err
	}

	// 获取InfraContainerID
	if infraID, ok := data["InfraContainerID"].(string); ok {
		pod.InfraID = c.truncateID(infraID)
	}

	// 获取容器数量
	if containers, ok := data["Containers"].([]interface{}); ok {
		pod.Containers = len(containers)
	}

	// 获取创建时间
	if created, ok := data["Created"].(string); ok {
		if t, err := time.Parse(time.RFC3339, created); err == nil {
			pod.Created = t
		}
	}

	return nil
}

// collectStateMetrics 收集Pod状态指标
func (c *Collector) collectStateMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, pod := range status.Pods {
		ch <- prometheus.MustNewConstMetric(
			c.stateDesc,
			prometheus.GaugeValue,
			float64(pod.StateValue),
			c.truncateID(pod.ID),
		)
	}
}

// collectInfoMetrics 收集Pod信息指标
func (c *Collector) collectInfoMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, pod := range status.Pods {
		ch <- prometheus.MustNewConstMetric(
			c.infoDesc,
			prometheus.GaugeValue,
			1,
			c.truncateID(pod.ID),
			c.truncateID(pod.InfraID),
			pod.Name,
		)
	}
}

// collectContainersMetrics 收集Pod中容器数量指标
func (c *Collector) collectContainersMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, pod := range status.Pods {
		ch <- prometheus.MustNewConstMetric(
			c.containersDesc,
			prometheus.GaugeValue,
			float64(pod.Containers),
			c.truncateID(pod.ID),
		)
	}
}

// collectCreatedMetrics 收集Pod创建时间指标
func (c *Collector) collectCreatedMetrics(ch chan<- prometheus.Metric, status *Status) {
	for _, pod := range status.Pods {
		var createdSeconds float64
		if !pod.Created.IsZero() {
			createdSeconds = float64(pod.Created.Unix())
		}

		ch <- prometheus.MustNewConstMetric(
			c.createdDesc,
			prometheus.GaugeValue,
			createdSeconds,
			c.truncateID(pod.ID),
		)
	}
}

// truncateID 截断ID到前12位，与其他收集器保持一致
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
