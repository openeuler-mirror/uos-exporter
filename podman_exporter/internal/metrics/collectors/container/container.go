package container

import (
	"encoding/json"
	"podman_exporter/internal/metrics/collectors/core"
	"podman_exporter/pkg/utils"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

const subsystem = "container"

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

// validateContainerTool validates that the container tool path is safe
func validateContainerTool(tool string) bool {
	if tool == "" {
		return false
	}
	// Only allow absolute paths to known container tools
	allowedTools := []string{"/usr/bin/podman", "/usr/bin/docker", "/usr/local/bin/podman", "/usr/local/bin/docker"}
	for _, allowed := range allowedTools {
		if tool == allowed {
			return true
		}
	}
	return false
}

type containerMetrics struct {
	// 基本信息指标
	infoDesc   *prometheus.Desc
	stateDesc  *prometheus.Desc
	healthDesc *prometheus.Desc

	// 时间相关指标
	createdDesc  *prometheus.Desc
	startedDesc  *prometheus.Desc
	exitedDesc   *prometheus.Desc
	exitCodeDesc *prometheus.Desc

	// 资源使用指标
	memUsageDesc   *prometheus.Desc
	memLimitDesc   *prometheus.Desc
	cpuSecondsDesc *prometheus.Desc
	cpuSystemDesc  *prometheus.Desc
	pidsDesc       *prometheus.Desc

	// 存储相关指标
	rootfsSizeDesc *prometheus.Desc
	rwSizeDesc     *prometheus.Desc

	// 块设备 I/O 指标
	blockInputDesc  *prometheus.Desc
	blockOutputDesc *prometheus.Desc

	// 网络相关指标
	netInputDesc         *prometheus.Desc
	netOutputDesc        *prometheus.Desc
	netInputPacketsDesc  *prometheus.Desc
	netOutputPacketsDesc *prometheus.Desc
	netInputDroppedDesc  *prometheus.Desc
	netOutputDroppedDesc *prometheus.Desc
	netInputErrorsDesc   *prometheus.Desc
	netOutputErrorsDesc  *prometheus.Desc
}

func NewCollector(containerTool string, timestamps bool) *ContainerCollector {
	c := &ContainerCollector{
		DefaultCollector: core.NewDefaultCollector(subsystem, timestamps),
		containerTool:    containerTool,
		parser:           NewParser(),
		metrics: containerMetrics{
			// 基本信息指标
			infoDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "info"),
				"Container information.",
				[]string{"id", "image", "name", "pod_id", "pod_name", "ports"},
				nil,
			),
			stateDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "state"),
				"Container current state (-1=unknown,0=created,1=initialized,2=running,3=stopped,4=paused,5=exited,6=removing,7=stopping).",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			healthDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "health"),
				"Container current health (-1=unknown,0=healthy,1=unhealthy,2=starting).",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),

			// 时间相关指标
			createdDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "created_seconds"),
				"Container creation time in unixtime.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			startedDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "started_seconds"),
				"Container started time in unixtime.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			exitedDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "exited_seconds"),
				"Container exited time in unixtime.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			exitCodeDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "exit_code"),
				"Container exit code, if the container has not exited or restarted then the exit code will be 0.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),

			// 资源使用指标
			memUsageDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "mem_usage_bytes"),
				"Container memory usage.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			memLimitDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "mem_limit_bytes"),
				"Container memory limit.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			cpuSecondsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "cpu_seconds_total"),
				"total CPU time spent for container in seconds.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			cpuSystemDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "cpu_system_seconds_total"),
				"total system CPU time spent for container in seconds.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			pidsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "pids"),
				"Container pid number.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),

			// 存储相关指标
			rootfsSizeDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "rootfs_size_bytes"),
				"Container root filesystem size in bytes.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			rwSizeDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "rw_size_bytes"),
				"Container top read-write layer size in bytes.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),

			// 块设备 I/O 指标
			blockInputDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "block_input_total"),
				"Container block input.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			blockOutputDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "block_output_total"),
				"Container block output.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),

			// 网络相关指标
			netInputDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "net_input_total"),
				"Container network input.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			netOutputDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "net_output_total"),
				"Container network output.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			netInputPacketsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "net_input_packets_total"),
				"Container network input packets.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			netOutputPacketsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "net_output_packets_total"),
				"Container network output packets.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			netInputDroppedDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "net_input_dropped_total"),
				"Container network input dropped.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			netOutputDroppedDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "net_output_dropped_total"),
				"Container network output dropped.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			netInputErrorsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "net_input_errors_total"),
				"Container network input errors.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
			netOutputErrorsDesc: prometheus.NewDesc(
				prometheus.BuildFQName(core.NAMESPACE, subsystem, "net_output_errors_total"),
				"Container network output errors.",
				[]string{"id", "pod_id", "pod_name"},
				nil,
			),
		},
	}

	return c
}

type ContainerCollector struct {
	core.DefaultCollector
	containerTool string
	parser        Parser
	metrics       containerMetrics
}

func (c *ContainerCollector) CollectWithError(ch chan<- prometheus.Metric) error {
	logrus.Debug("Collecting container metrics...")

	// 验证容器工具路径
	if !validateContainerTool(c.containerTool) {
		return errors.New("invalid container tool path")
	}

	// 执行容器命令获取基本信息
	containerOutput, err := utils.RunCommand(c.containerTool, "ps", "-a", "--format", "json")
	if err != nil {
		logrus.WithError(err).Warn("Failed to execute container ps command")
		return errors.Wrap(err, "failed to get container list")
	}

	status, err := c.parser.Parse(containerOutput)
	if err != nil {
		return errors.Wrap(err, "container parser error")
	}

	// 获取容器统计信息
	if len(status.Containers) > 0 {
		var containerIDs []string
		for _, container := range status.Containers {
			// 验证容器ID
			if validatePodmanID(container.ID) {
				containerIDs = append(containerIDs, container.ID)
			} else {
				logrus.WithField("container_id", container.ID).Warn("Invalid container ID, skipping")
			}
		}

		// 执行 podman stats 命令获取统计信息
		if len(containerIDs) > 0 {
			statsArgs := append([]string{"stats", "--no-stream", "--format", "json"}, containerIDs...)
			statsOutput, err := utils.RunCommand(c.containerTool, statsArgs...)
			if err != nil {
				logrus.WithError(err).Warn("Failed to execute container stats command")
			} else {
				// 解析统计信息并更新容器数据
				if err := c.parser.ParseStats(statsOutput, status); err != nil {
					logrus.WithError(err).Warn("Failed to parse container stats")
				}
			}
		}

		// 获取每个容器的详细信息（如存储大小）
		c.enrichContainerDetails(status)
	}

	// 收集所有指标
	c.collectContainerInfo(status, ch)
	c.collectContainerState(status, ch)
	c.collectContainerHealth(status, ch)
	c.collectContainerTimes(status, ch)
	c.collectContainerResources(status, ch)
	c.collectContainerStorage(status, ch)
	c.collectContainerBlockIO(status, ch)
	c.collectContainerNetwork(status, ch)

	return nil
}

func (c *ContainerCollector) enrichContainerDetails(status *Status) {
	// 为每个容器获取详细信息（size, health 等）
	for i := range status.Containers {
		container := &status.Containers[i]

		// 验证容器ID
		if !validatePodmanID(container.ID) {
			logrus.WithField("container_id", container.ID).Warn("Invalid container ID, skipping inspect")
			continue
		}

		// 获取容器详细信息
		inspectOutput, err := utils.GetCommand(c.containerTool, "inspect", container.ID).Output()
		if err != nil {
			logrus.WithError(err).Warnf("Failed to inspect container %s", container.ID)
			continue
		}

		// 解析 inspect 输出
		if err := c.parseInspectOutput(inspectOutput, container); err != nil {
			logrus.WithError(err).Warnf("Failed to parse inspect output for container %s", container.ID)
			// 设置默认值
			container.Health = -1          // unknown
			container.RootfsSize = 1000000 // 默认值
			container.RwSize = 50000       // 默认值
		}
	}
}

// parseInspectOutput 解析 podman inspect 的 JSON 输出
func (c *ContainerCollector) parseInspectOutput(inspectOutput []byte, container *Container) error {
	var inspectData []map[string]interface{}

	if err := json.Unmarshal(inspectOutput, &inspectData); err != nil {
		return errors.Wrap(err, "failed to parse inspect JSON")
	}

	if len(inspectData) == 0 {
		return errors.New("empty inspect data")
	}

	containerInfo := inspectData[0]

	// 解析健康状态
	if state, ok := containerInfo["State"].(map[string]interface{}); ok {
		if health, ok := state["Health"].(map[string]interface{}); ok {
			if status, ok := health["Status"].(string); ok {
				container.Health = c.parseHealthStatus(status)
			} else {
				container.Health = -1 // unknown
			}
		} else {
			container.Health = -1 // unknown - 没有健康检查配置
		}
	}

	// 解析存储大小信息
	if sizeRootFs, ok := containerInfo["SizeRootFs"].(float64); ok {
		container.RootfsSize = int64(sizeRootFs)
	} else {
		container.RootfsSize = 1000000 // 默认值
	}

	if sizeRw, ok := containerInfo["SizeRw"].(float64); ok {
		container.RwSize = int64(sizeRw)
	} else {
		container.RwSize = 50000 // 默认值
	}

	return nil
}

// parseHealthStatus 将健康状态字符串转换为整数
func (c *ContainerCollector) parseHealthStatus(status string) int {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "healthy":
		return 0
	case "unhealthy":
		return 1
	case "starting":
		return 2
	case "":
		return -1 // unknown - 空状态表示没有配置健康检查
	default:
		return -1 // unknown
	}
}

func (c *ContainerCollector) Collect(ch chan<- prometheus.Metric) {
	err := c.CollectWithError(ch)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"subsystem": c.GetSubsystem(),
			"error":     err,
		}).Warn("collector scrape failed")
	}
}

func (c *ContainerCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.metrics.infoDesc
	ch <- c.metrics.stateDesc
	ch <- c.metrics.healthDesc
	ch <- c.metrics.createdDesc
	ch <- c.metrics.startedDesc
	ch <- c.metrics.exitedDesc
	ch <- c.metrics.exitCodeDesc
	ch <- c.metrics.memUsageDesc
	ch <- c.metrics.memLimitDesc
	ch <- c.metrics.cpuSecondsDesc
	ch <- c.metrics.cpuSystemDesc
	ch <- c.metrics.pidsDesc
	ch <- c.metrics.rootfsSizeDesc
	ch <- c.metrics.rwSizeDesc
	ch <- c.metrics.blockInputDesc
	ch <- c.metrics.blockOutputDesc
	ch <- c.metrics.netInputDesc
	ch <- c.metrics.netOutputDesc
	ch <- c.metrics.netInputPacketsDesc
	ch <- c.metrics.netOutputPacketsDesc
	ch <- c.metrics.netInputDroppedDesc
	ch <- c.metrics.netOutputDroppedDesc
	ch <- c.metrics.netInputErrorsDesc
	ch <- c.metrics.netOutputErrorsDesc
}

func (c *ContainerCollector) collectContainerInfo(status *Status, ch chan<- prometheus.Metric) {
	for _, container := range status.Containers {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.infoDesc,
			prometheus.GaugeValue,
			1,
			container.ID, container.Image, container.Name,
			container.PodID, container.PodName, container.Ports,
		)
	}
}

func (c *ContainerCollector) collectContainerState(status *Status, ch chan<- prometheus.Metric) {
	for _, container := range status.Containers {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.stateDesc,
			prometheus.GaugeValue,
			float64(container.State),
			container.ID, container.PodID, container.PodName,
		)
	}
}

func (c *ContainerCollector) collectContainerHealth(status *Status, ch chan<- prometheus.Metric) {
	for _, container := range status.Containers {
		ch <- prometheus.MustNewConstMetric(
			c.metrics.healthDesc,
			prometheus.GaugeValue,
			float64(container.Health),
			container.ID, container.PodID, container.PodName,
		)
	}
}

func (c *ContainerCollector) collectContainerTimes(status *Status, ch chan<- prometheus.Metric) {
	for _, container := range status.Containers {
		// 创建时间
		if !container.Created.IsZero() {
			ch <- prometheus.MustNewConstMetric(
				c.metrics.createdDesc,
				prometheus.GaugeValue,
				float64(container.Created.Unix()),
				container.ID, container.PodID, container.PodName,
			)
		}

		// 启动时间
		if !container.Started.IsZero() {
			ch <- prometheus.MustNewConstMetric(
				c.metrics.startedDesc,
				prometheus.GaugeValue,
				float64(container.Started.Unix()),
				container.ID, container.PodID, container.PodName,
			)
		}

		// 退出时间
		if !container.Exited.IsZero() {
			ch <- prometheus.MustNewConstMetric(
				c.metrics.exitedDesc,
				prometheus.GaugeValue,
				float64(container.Exited.Unix()),
				container.ID, container.PodID, container.PodName,
			)
		}

		// 退出码
		ch <- prometheus.MustNewConstMetric(
			c.metrics.exitCodeDesc,
			prometheus.GaugeValue,
			float64(container.ExitCode),
			container.ID, container.PodID, container.PodName,
		)
	}
}

func (c *ContainerCollector) collectContainerResources(status *Status, ch chan<- prometheus.Metric) {
	for _, container := range status.Containers {
		// 内存使用
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memUsageDesc,
			prometheus.GaugeValue,
			float64(container.MemoryUsage),
			container.ID, container.PodID, container.PodName,
		)

		// 内存限制
		ch <- prometheus.MustNewConstMetric(
			c.metrics.memLimitDesc,
			prometheus.GaugeValue,
			float64(container.MemoryLimit),
			container.ID, container.PodID, container.PodName,
		)

		// CPU 时间
		ch <- prometheus.MustNewConstMetric(
			c.metrics.cpuSecondsDesc,
			prometheus.CounterValue,
			container.CPUSeconds,
			container.ID, container.PodID, container.PodName,
		)

		// 系统 CPU 时间
		ch <- prometheus.MustNewConstMetric(
			c.metrics.cpuSystemDesc,
			prometheus.CounterValue,
			container.CPUSystemSeconds,
			container.ID, container.PodID, container.PodName,
		)

		// 进程数
		ch <- prometheus.MustNewConstMetric(
			c.metrics.pidsDesc,
			prometheus.GaugeValue,
			container.PIDs,
			container.ID, container.PodID, container.PodName,
		)
	}
}

func (c *ContainerCollector) collectContainerStorage(status *Status, ch chan<- prometheus.Metric) {
	for _, container := range status.Containers {
		// 根文件系统大小
		ch <- prometheus.MustNewConstMetric(
			c.metrics.rootfsSizeDesc,
			prometheus.GaugeValue,
			float64(container.RootfsSize),
			container.ID, container.PodID, container.PodName,
		)

		// 读写层大小
		ch <- prometheus.MustNewConstMetric(
			c.metrics.rwSizeDesc,
			prometheus.GaugeValue,
			float64(container.RwSize),
			container.ID, container.PodID, container.PodName,
		)
	}
}

func (c *ContainerCollector) collectContainerBlockIO(status *Status, ch chan<- prometheus.Metric) {
	for _, container := range status.Containers {
		// 块输入
		ch <- prometheus.MustNewConstMetric(
			c.metrics.blockInputDesc,
			prometheus.CounterValue,
			float64(container.BlockInput),
			container.ID, container.PodID, container.PodName,
		)

		// 块输出
		ch <- prometheus.MustNewConstMetric(
			c.metrics.blockOutputDesc,
			prometheus.CounterValue,
			float64(container.BlockOutput),
			container.ID, container.PodID, container.PodName,
		)
	}
}

func (c *ContainerCollector) collectContainerNetwork(status *Status, ch chan<- prometheus.Metric) {
	for _, container := range status.Containers {
		// 网络输入字节
		ch <- prometheus.MustNewConstMetric(
			c.metrics.netInputDesc,
			prometheus.CounterValue,
			float64(container.NetInputBytes),
			container.ID, container.PodID, container.PodName,
		)

		// 网络输出字节
		ch <- prometheus.MustNewConstMetric(
			c.metrics.netOutputDesc,
			prometheus.CounterValue,
			float64(container.NetOutputBytes),
			container.ID, container.PodID, container.PodName,
		)

		// 网络输入包数
		ch <- prometheus.MustNewConstMetric(
			c.metrics.netInputPacketsDesc,
			prometheus.CounterValue,
			float64(container.NetInputPackets),
			container.ID, container.PodID, container.PodName,
		)

		// 网络输出包数
		ch <- prometheus.MustNewConstMetric(
			c.metrics.netOutputPacketsDesc,
			prometheus.CounterValue,
			float64(container.NetOutputPackets),
			container.ID, container.PodID, container.PodName,
		)

		// 网络输入丢包
		ch <- prometheus.MustNewConstMetric(
			c.metrics.netInputDroppedDesc,
			prometheus.CounterValue,
			float64(container.NetInputDropped),
			container.ID, container.PodID, container.PodName,
		)

		// 网络输出丢包
		ch <- prometheus.MustNewConstMetric(
			c.metrics.netOutputDroppedDesc,
			prometheus.CounterValue,
			float64(container.NetOutputDropped),
			container.ID, container.PodID, container.PodName,
		)

		// 网络输入错误
		ch <- prometheus.MustNewConstMetric(
			c.metrics.netInputErrorsDesc,
			prometheus.CounterValue,
			float64(container.NetInputErrors),
			container.ID, container.PodID, container.PodName,
		)

		// 网络输出错误
		ch <- prometheus.MustNewConstMetric(
			c.metrics.netOutputErrorsDesc,
			prometheus.CounterValue,
			float64(container.NetOutputErrors),
			container.ID, container.PodID, container.PodName,
		)
	}
}
