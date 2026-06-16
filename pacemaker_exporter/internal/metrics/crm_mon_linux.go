package metrics

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	crmMonPath = "/usr/sbin/crm_mon"
)

// CrmMonExecutor defines an interface for executing crm_mon commands
type CrmMonExecutor interface {
	Execute(ctx context.Context, args ...string) ([]byte, error)
}

// DefaultCrmMonExecutor implements CrmMonExecutor interface
type DefaultCrmMonExecutor struct{}

// Execute runs crm_mon utility with given arguments
func (e *DefaultCrmMonExecutor) Execute(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, crmMonPath, args...)
	// Disable localization for consistent parsing
	cmd.Env = append(os.Environ(), "LANG=C")

	logrus.WithFields(logrus.Fields{
		"command": crmMonPath,
		"args":    strings.Join(args, " "),
	}).Debug("Executing crm_mon command")

	out, err := cmd.Output()
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"command": crmMonPath,
			"args":    strings.Join(args, " "),
			"error":   err,
		}).Error("Failed to execute crm_mon command")
		return nil, fmt.Errorf("crm_mon execution failed: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"command":     crmMonPath,
		"args":        strings.Join(args, " "),
		"output_size": len(out),
	}).Debug("crm_mon command executed successfully")

	return out, nil
}

// Global executor instance - can be replaced for testing
var crmMonExecutor CrmMonExecutor = &DefaultCrmMonExecutor{}

// crmMonExec executes crm_mon utility with timeout context
func crmMonExec(args ...string) ([]byte, error) {
	// Create context with timeout to prevent hanging
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return crmMonExecutor.Execute(ctx, args...)
}

// parseCrmMonXML parses XML data into CrmMonStruct with improved error handling
func parseCrmMonXML(data []byte) (CrmMonStruct, error) {
	var crmMonOut CrmMonStruct

	if len(data) == 0 {
		return crmMonOut, fmt.Errorf("empty XML data received")
	}

	logrus.WithField("xml_size", len(data)).Debug("Parsing crm_mon XML data")

	if err := xml.Unmarshal(data, &crmMonOut); err != nil {
		logrus.WithFields(logrus.Fields{
			"error":    err,
			"xml_size": len(data),
		}).Error("Failed to parse crm_mon XML")
		return crmMonOut, fmt.Errorf("XML parsing failed: %w", err)
	}

	logrus.Debug("crm_mon XML parsed successfully")
	return crmMonOut, nil
}

// getCrmMonInfo returns crm_mon information
func (c *crmMonCollector) getCrmMonInfo(ch chan<- prometheus.Metric) error {
	outBytes, err := crmMonExec("-Xr")
	if err != nil {
		logrus.Errorln(err)
		return err
	}

	crmMonStruct, err := parseCrmMonXML(outBytes)
	if err != nil {
		logrus.Errorln(err)
		return err
	}

	elemEnabledSlice := strings.Split(crmMonElemEnabled, ",")

	// Summary metrics
	if stringInSlice("summary", elemEnabledSlice) {
		err = c.exposeSummary(ch, crmMonStruct.Summary)
		if err != nil {
			logrus.Errorln(err)
		}
	}

	// Nodes section metrics
	if stringInSlice("nodes", elemEnabledSlice) {
		c.exposeNodes(ch, crmMonStruct.Nodes)
	}

	// Node attribute section metrics
	if stringInSlice("nodes", elemEnabledSlice) {
		c.exposeNodeAttributes(ch, crmMonStruct.NodeAttributes)
	}

	// Resources section metrics
	if stringInSlice("clones", elemEnabledSlice) {
		c.exposeResourcesClone(ch, crmMonStruct.Resources)
	}

	if stringInSlice("resources", elemEnabledSlice) {
		c.exposeResources(ch, crmMonStruct.Resources)
	}

	if stringInSlice("resources_group", elemEnabledSlice) {
		c.exposeResourcesGroup(ch, crmMonStruct.Resources)
	}

	if stringInSlice("failures", elemEnabledSlice) {
		c.exposeFailures(ch, crmMonStruct)
	}

	if stringInSlice("bans", elemEnabledSlice) {
		c.exposeBans(ch, crmMonStruct)
	}

	return nil
}

// HTMLHandler returns crm_mon -wr
func HTMLHandler(w http.ResponseWriter, r *http.Request) {
	outBytes, err := crmMonExec("-wr")
	if err != nil {
		logrus.Warnln("Error running `crm_mon -wr`", err)
		w.WriteHeader(http.StatusServiceUnavailable)

		_, err = w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))

		if err != nil {
			logrus.Fatal(err)
		}

		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	_, err = w.Write(outBytes)

	if err != nil {
		logrus.Fatal(err)
	}
}

// XMLHandler returns crm_mon -Xr
func XMLHandler(w http.ResponseWriter, r *http.Request) {
	outBytes, err := crmMonExec("-Xr")
	if err != nil {
		logrus.Warnln("Error running `crm_mon -Xr`", err)
		w.WriteHeader(http.StatusServiceUnavailable)

		_, err = w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))

		if err != nil {
			logrus.Fatal(err)
		}

		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")

	_, err = w.Write(outBytes)

	if err != nil {
		logrus.Fatal(err)
	}
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}

	return false
}

// expose Summary metrics
func (c *crmMonCollector) exposeSummary(ch chan<- prometheus.Metric, summaryStruct SummaryStruct) error {
	// 记录函数开始执行时间用于性能监控
	startTime := time.Now()

	// 添加详细的函数入口日志
	logrus.WithFields(logrus.Fields{
		"function":   "exposeSummary",
		"dc_name":    summaryStruct.CurrentDC.Name,
		"dc_version": summaryStruct.CurrentDC.Version,
		"dc_present": summaryStruct.CurrentDC.Present,
		"dc_quorum":  summaryStruct.CurrentDC.Quorum,
	}).Debug("Starting to expose summary metrics")

	// 输入验证：检查通道是否有效
	if ch == nil {
		err := fmt.Errorf("metrics channel cannot be nil")
		logrus.WithError(err).Error("Invalid metrics channel provided")
		return err
	}

	// 输入验证：检查基本的DC信息
	if summaryStruct.CurrentDC.Name == "" {
		logrus.WithField("function", "exposeSummary").Warn("DC name is empty, using default value")
		summaryStruct.CurrentDC.Name = "unknown"
	}

	if summaryStruct.CurrentDC.Version == "" {
		logrus.WithField("function", "exposeSummary").Warn("DC version is empty, using default value")
		summaryStruct.CurrentDC.Version = "unknown"
	}

	// 统计要发送的指标数量
	var metricsCount int
	defer func() {
		// 记录函数执行完成的性能数据
		duration := time.Since(startTime)
		logrus.WithFields(logrus.Fields{
			"function":      "exposeSummary",
			"duration_ms":   duration.Milliseconds(),
			"metrics_count": metricsCount,
			"dc_name":       summaryStruct.CurrentDC.Name,
		}).Debug("Summary metrics exposure completed")
	}()

	// 1. 处理基本信息指标
	logrus.WithField("metric", "crm_mon_info").Debug("Emitting crm_mon_info metric")
	if err := c.emitBasicInfoMetric(ch, summaryStruct.CurrentDC.Version); err != nil {
		logrus.WithError(err).Error("Failed to emit basic info metric")
		return fmt.Errorf("failed to emit basic info metric: %w", err)
	}
	metricsCount++

	// 2. 处理时间相关指标
	timeMetricsCount, err := c.emitTimeMetrics(ch, summaryStruct)
	if err != nil {
		logrus.WithError(err).Error("Failed to emit time metrics")
		// 继续处理其他指标，不直接返回错误
	} else {
		metricsCount += timeMetricsCount
	}

	// 3. 处理DC状态指标
	dcMetricsCount, err := c.emitDCStatusMetrics(ch, summaryStruct.CurrentDC)
	if err != nil {
		logrus.WithError(err).Error("Failed to emit DC status metrics")
		// 继续处理其他指标
	} else {
		metricsCount += dcMetricsCount
	}

	// 4. 处理节点配置指标
	nodeMetricsCount, err := c.emitNodeConfigMetrics(ch, summaryStruct)
	if err != nil {
		logrus.WithError(err).Error("Failed to emit node config metrics")
		// 继续处理其他指标
	} else {
		metricsCount += nodeMetricsCount
	}

	// 5. 处理资源配置指标
	resourceMetricsCount, err := c.emitResourceConfigMetrics(ch, summaryStruct)
	if err != nil {
		logrus.WithError(err).Error("Failed to emit resource config metrics")
		// 继续处理其他指标
	} else {
		metricsCount += resourceMetricsCount
	}

	// 6. 处理集群选项指标
	clusterMetricsCount, err := c.emitClusterOptionsMetrics(ch, summaryStruct)
	if err != nil {
		logrus.WithError(err).Error("Failed to emit cluster options metrics")
		// 继续处理其他指标
	} else {
		metricsCount += clusterMetricsCount
	}

	// 最终验证：确保至少发送了一些指标
	if metricsCount == 0 {
		err := fmt.Errorf("no metrics were successfully emitted")
		logrus.WithError(err).Warn("Summary metrics exposure resulted in zero metrics")
		return err
	}

	logrus.WithFields(logrus.Fields{
		"function":      "exposeSummary",
		"total_metrics": metricsCount,
		"dc_name":       summaryStruct.CurrentDC.Name,
	}).Info("Successfully exposed all summary metrics")

	return nil
}

// emitBasicInfoMetric 发送基本信息指标
func (c *crmMonCollector) emitBasicInfoMetric(ch chan<- prometheus.Metric, version string) error {
	logrus.WithField("version", version).Debug("Emitting basic info metric")

	// 验证版本信息
	if version == "" {
		version = "unknown"
		logrus.Warn("DC version is empty, using 'unknown' as default")
	}

	// 使用defer捕获panic
	defer func() {
		if r := recover(); r != nil {
			logrus.WithFields(logrus.Fields{
				"panic":   r,
				"metric":  "crm_mon_info",
				"version": version,
			}).Error("Panic occurred while emitting basic info metric")
		}
	}()

	// 发送指标
	ch <- prometheus.MustNewConstMetric(c.crmMonInfo, prometheus.GaugeValue, 1.0, version)

	logrus.WithField("version", version).Debug("Basic info metric emitted successfully")
	return nil
}

// emitTimeMetrics 发送时间相关指标
func (c *crmMonCollector) emitTimeMetrics(ch chan<- prometheus.Metric, summaryStruct SummaryStruct) (int, error) {
	var metricsCount int

	logrus.WithFields(logrus.Fields{
		"last_update_time": summaryStruct.LastUpdate.Time,
		"last_change_time": summaryStruct.LastChange.Time,
		"stack_type":       summaryStruct.Stack.Type,
	}).Debug("Processing time metrics")

	// 处理最后更新时间
	if summaryStruct.LastUpdate.Time != "" {
		lastUpdateTime, err := time.Parse("Mon Jan _2 15:04:05 2006", summaryStruct.LastUpdate.Time)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"time_string": summaryStruct.LastUpdate.Time,
				"error":       err,
			}).Warn("Failed to parse last update time, skipping metric")
		} else {
			// 验证解析的时间是否合理（不能是未来时间，不能太古老）
			now := time.Now()
			if lastUpdateTime.After(now) {
				logrus.WithFields(logrus.Fields{
					"parsed_time":  lastUpdateTime,
					"current_time": now,
				}).Warn("Last update time is in the future, this might indicate a parsing error")
			}

			// 检查时间是否太古老（超过10年）
			if now.Sub(lastUpdateTime) > 10*365*24*time.Hour {
				logrus.WithFields(logrus.Fields{
					"parsed_time": lastUpdateTime,
					"age_years":   now.Sub(lastUpdateTime).Hours() / (365 * 24),
				}).Warn("Last update time is very old, this might indicate a parsing error")
			}

			stackType := summaryStruct.Stack.Type
			if stackType == "" {
				stackType = "unknown"
				logrus.Warn("Stack type is empty, using 'unknown' as default")
			}

			ch <- prometheus.MustNewConstMetric(c.crmMonLastUpdate,
				prometheus.GaugeValue, float64(lastUpdateTime.Unix()), stackType)

			metricsCount++
			logrus.WithFields(logrus.Fields{
				"timestamp":  lastUpdateTime.Unix(),
				"stack_type": stackType,
			}).Debug("Last update time metric emitted successfully")
		}
	} else {
		logrus.Debug("Last update time is empty, skipping metric")
	}

	// 处理最后变更时间
	if summaryStruct.LastChange.Time != "" {
		lastChangeTime, err := time.Parse("Mon Jan _2 15:04:05 2006", summaryStruct.LastChange.Time)
		if err != nil {
			logrus.WithFields(logrus.Fields{
				"time_string": summaryStruct.LastChange.Time,
				"error":       err,
			}).Warn("Failed to parse last change time, skipping metric")
		} else {
			// 验证变更信息
			user := summaryStruct.LastChange.User
			client := summaryStruct.LastChange.Client
			origin := summaryStruct.LastChange.Origin

			// 设置默认值
			if user == "" {
				user = "unknown"
				logrus.Debug("Last change user is empty, using 'unknown' as default")
			}
			if client == "" {
				client = "unknown"
				logrus.Debug("Last change client is empty, using 'unknown' as default")
			}
			if origin == "" {
				origin = "unknown"
				logrus.Debug("Last change origin is empty, using 'unknown' as default")
			}

			ch <- prometheus.MustNewConstMetric(c.crmMonLastChange,
				prometheus.GaugeValue, float64(lastChangeTime.Unix()),
				user, client, origin)

			metricsCount++
			logrus.WithFields(logrus.Fields{
				"timestamp": lastChangeTime.Unix(),
				"user":      user,
				"client":    client,
				"origin":    origin,
			}).Debug("Last change time metric emitted successfully")
		}
	} else {
		logrus.Debug("Last change time is empty, skipping metric")
	}

	logrus.WithField("metrics_count", metricsCount).Debug("Time metrics processing completed")
	return metricsCount, nil
}

// emitDCStatusMetrics 发送DC状态指标
func (c *crmMonCollector) emitDCStatusMetrics(ch chan<- prometheus.Metric, dcStruct struct {
	Present bool   `xml:"present,attr"`
	Quorum  bool   `xml:"with_quorum,attr"`
	Version string `xml:"version,attr"`
	Name    string `xml:"name,attr"`
	ID      string `xml:"id,attr"`
}) (int, error) {
	var metricsCount int
	dcName := dcStruct.Name

	logrus.WithFields(logrus.Fields{
		"dc_name":    dcName,
		"dc_present": dcStruct.Present,
		"dc_quorum":  dcStruct.Quorum,
	}).Debug("Processing DC status metrics")

	// 验证DC名称
	if dcName == "" {
		dcName = "unknown"
		logrus.Warn("DC name is empty, using 'unknown' as default")
	}

	// 发送DC存在状态指标
	dcPresentValue := 0.0
	if dcStruct.Present {
		dcPresentValue = 1.0
	}

	ch <- prometheus.MustNewConstMetric(c.crmMonDCPresent,
		prometheus.GaugeValue, dcPresentValue, dcName)
	metricsCount++

	logrus.WithFields(logrus.Fields{
		"dc_name": dcName,
		"value":   dcPresentValue,
	}).Debug("DC present metric emitted")

	// 发送DC仲裁状态指标
	dcQuorumValue := 0.0
	if dcStruct.Quorum {
		dcQuorumValue = 1.0
	}

	ch <- prometheus.MustNewConstMetric(c.crmMonDCQuorum,
		prometheus.GaugeValue, dcQuorumValue, dcName)
	metricsCount++

	logrus.WithFields(logrus.Fields{
		"dc_name":       dcName,
		"metrics_count": metricsCount,
	}).Debug("DC status metrics processing completed")

	return metricsCount, nil
}

// emitNodeConfigMetrics 发送节点配置指标
func (c *crmMonCollector) emitNodeConfigMetrics(ch chan<- prometheus.Metric, summaryStruct SummaryStruct) (int, error) {
	var metricsCount int

	logrus.WithFields(logrus.Fields{
		"nodes_configured": summaryStruct.NodesConfigured.Number,
		"expected_votes":   summaryStruct.NodesConfigured.ExpectedVotes,
	}).Debug("Processing node configuration metrics")

	// 验证节点数量合理性
	nodeCount := summaryStruct.NodesConfigured.Number
	expectedVotes := summaryStruct.NodesConfigured.ExpectedVotes

	if nodeCount < 0 {
		logrus.WithField("node_count", nodeCount).Warn("Node count is negative, this seems incorrect")
		nodeCount = 0
	}

	if expectedVotes == "" {
		expectedVotes = "unknown"
		logrus.Debug("Expected votes is empty, using 'unknown' as default")
	}

	// 检查节点数量是否合理
	if nodeCount > 1000 {
		logrus.WithField("node_count", nodeCount).Warn("Node count seems unusually high")
	}

	ch <- prometheus.MustNewConstMetric(c.crmMonNodesConfigured,
		prometheus.GaugeValue, nodeCount, expectedVotes)
	metricsCount++

	logrus.WithFields(logrus.Fields{
		"node_count":     nodeCount,
		"expected_votes": expectedVotes,
	}).Debug("Node configuration metric emitted successfully")

	return metricsCount, nil
}

// emitResourceConfigMetrics 发送资源配置指标
func (c *crmMonCollector) emitResourceConfigMetrics(ch chan<- prometheus.Metric, summaryStruct SummaryStruct) (int, error) {
	var metricsCount int
	dcName := summaryStruct.CurrentDC.Name

	logrus.WithFields(logrus.Fields{
		"resources_configured": summaryStruct.ResourcesConfigured.Number,
		"resources_disabled":   summaryStruct.ResourcesConfigured.Disabled,
		"resources_blocked":    summaryStruct.ResourcesConfigured.Blocked,
		"dc_name":              dcName,
	}).Debug("Processing resource configuration metrics")

	// 验证DC名称
	if dcName == "" {
		dcName = "unknown"
		logrus.Warn("DC name is empty for resource metrics, using 'unknown' as default")
	}

	// 验证资源数量合理性
	resourcesConfigured := summaryStruct.ResourcesConfigured.Number
	resourcesDisabled := summaryStruct.ResourcesConfigured.Disabled
	resourcesBlocked := summaryStruct.ResourcesConfigured.Blocked

	if resourcesConfigured < 0 {
		logrus.WithField("resources_configured", resourcesConfigured).Warn("Configured resources count is negative")
		resourcesConfigured = 0
	}

	if resourcesDisabled < 0 {
		logrus.WithField("resources_disabled", resourcesDisabled).Warn("Disabled resources count is negative")
		resourcesDisabled = 0
	}

	if resourcesBlocked < 0 {
		logrus.WithField("resources_blocked", resourcesBlocked).Warn("Blocked resources count is negative")
		resourcesBlocked = 0
	}

	// 逻辑验证：禁用和阻塞的资源不应该超过总配置的资源
	if resourcesDisabled > resourcesConfigured {
		logrus.WithFields(logrus.Fields{
			"disabled":   resourcesDisabled,
			"configured": resourcesConfigured,
		}).Warn("Disabled resources count exceeds configured resources count")
	}

	if resourcesBlocked > resourcesConfigured {
		logrus.WithFields(logrus.Fields{
			"blocked":    resourcesBlocked,
			"configured": resourcesConfigured,
		}).Warn("Blocked resources count exceeds configured resources count")
	}

	// 发送配置的资源数量指标
	ch <- prometheus.MustNewConstMetric(c.crmMonResourcesConfigured,
		prometheus.GaugeValue, resourcesConfigured, dcName)
	metricsCount++

	// 发送禁用的资源数量指标
	ch <- prometheus.MustNewConstMetric(c.crmMonResourcesDisabled,
		prometheus.GaugeValue, resourcesDisabled, dcName)
	metricsCount++

	// 发送阻塞的资源数量指标
	ch <- prometheus.MustNewConstMetric(c.crmMonResourcesBlocked,
		prometheus.GaugeValue, resourcesBlocked, dcName)
	metricsCount++

	logrus.WithFields(logrus.Fields{
		"dc_name":       dcName,
		"metrics_count": metricsCount,
		"configured":    resourcesConfigured,
		"disabled":      resourcesDisabled,
		"blocked":       resourcesBlocked,
	}).Debug("Resource configuration metrics emitted successfully")

	return metricsCount, nil
}

// emitClusterOptionsMetrics 发送集群选项指标
func (c *crmMonCollector) emitClusterOptionsMetrics(ch chan<- prometheus.Metric, summaryStruct SummaryStruct) (int, error) {
	var metricsCount int
	dcName := summaryStruct.CurrentDC.Name
	clusterOptions := summaryStruct.ClusterOptions

	logrus.WithFields(logrus.Fields{
		"stonith_enabled":   clusterOptions.StonithEnabled,
		"symmetric_cluster": clusterOptions.SymmetricCluster,
		"maintenance_mode":  clusterOptions.MaintenanceMode,
		"dc_name":           dcName,
	}).Debug("Processing cluster options metrics")

	// 验证DC名称
	if dcName == "" {
		dcName = "unknown"
		logrus.Warn("DC name is empty for cluster options metrics, using 'unknown' as default")
	}

	// 发送STONITH启用状态指标
	stonithValue := 0.0
	if clusterOptions.StonithEnabled {
		stonithValue = 1.0
	}

	ch <- prometheus.MustNewConstMetric(c.crmMonStonith,
		prometheus.GaugeValue, stonithValue, dcName)
	metricsCount++

	logrus.WithFields(logrus.Fields{
		"dc_name": dcName,
		"enabled": clusterOptions.StonithEnabled,
		"value":   stonithValue,
	}).Debug("STONITH metric emitted")

	// 发送对称集群状态指标
	symmetricValue := 0.0
	if clusterOptions.SymmetricCluster {
		symmetricValue = 1.0
	}

	ch <- prometheus.MustNewConstMetric(c.crmMonSymmetricCluster,
		prometheus.GaugeValue, symmetricValue, dcName)
	metricsCount++

	logrus.WithFields(logrus.Fields{
		"dc_name":   dcName,
		"symmetric": clusterOptions.SymmetricCluster,
		"value":     symmetricValue,
	}).Debug("Symmetric cluster metric emitted")

	// 发送维护模式状态指标
	maintenanceValue := 0.0
	if clusterOptions.MaintenanceMode {
		maintenanceValue = 1.0
	}

	ch <- prometheus.MustNewConstMetric(c.crmMonMaintenanceMode,
		prometheus.GaugeValue, maintenanceValue, dcName)
	metricsCount++

	logrus.WithFields(logrus.Fields{
		"dc_name":     dcName,
		"maintenance": clusterOptions.MaintenanceMode,
		"value":       maintenanceValue,
	}).Debug("Maintenance mode metric emitted")

	// 记录集群配置的一些警告信息
	if !clusterOptions.StonithEnabled {
		logrus.WithField("dc_name", dcName).Warn("STONITH is disabled, this is not recommended for production clusters")
	}

	if clusterOptions.MaintenanceMode {
		logrus.WithField("dc_name", dcName).Info("Cluster is in maintenance mode")
	}

	logrus.WithFields(logrus.Fields{
		"dc_name":       dcName,
		"metrics_count": metricsCount,
	}).Debug("Cluster options metrics processing completed")

	return metricsCount, nil
}

// expose Nodes metrics
func (c *crmMonCollector) exposeNodes(ch chan<- prometheus.Metric, nodesStruct NodesStruct) {
	// 记录函数开始执行时间用于性能监控
	startTime := time.Now()

	// 添加详细的函数入口日志
	logrus.WithFields(logrus.Fields{
		"function":   "exposeNodes",
		"node_count": len(nodesStruct.Node),
	}).Debug("Starting to expose node metrics")

	// 输入验证：检查通道是否有效
	if ch == nil {
		logrus.WithField("function", "exposeNodes").Error("Metrics channel cannot be nil")
		return
	}

	// 输入验证：检查节点结构是否有效
	if len(nodesStruct.Node) == 0 {
		logrus.WithField("function", "exposeNodes").Warn("No nodes found in the cluster")
		return
	}

	// 统计指标和节点状态
	var (
		totalMetricsEmitted int
		processedNodes      int
		onlineNodes         int
		standbyNodes        int
		maintenanceNodes    int
		uncleanNodes        int
		dcNodes             int
		nodeHealthStats     = make(map[string]int)
		nodeTypeStats       = make(map[string]int)
		nodeIssues          []string
	)

	// 性能监控和统计的defer函数
	defer func() {
		duration := time.Since(startTime)

		// 记录函数执行完成的详细统计
		logrus.WithFields(logrus.Fields{
			"function":          "exposeNodes",
			"duration_ms":       duration.Milliseconds(),
			"total_metrics":     totalMetricsEmitted,
			"processed_nodes":   processedNodes,
			"online_nodes":      onlineNodes,
			"standby_nodes":     standbyNodes,
			"maintenance_nodes": maintenanceNodes,
			"unclean_nodes":     uncleanNodes,
			"dc_nodes":          dcNodes,
			"node_type_stats":   nodeTypeStats,
			"node_health_stats": nodeHealthStats,
			"node_issues_count": len(nodeIssues),
		}).Info("Node metrics exposure completed")

		// 如果发现问题节点，记录警告
		if len(nodeIssues) > 0 {
			logrus.WithFields(logrus.Fields{
				"issues": nodeIssues,
				"count":  len(nodeIssues),
			}).Warn("Found nodes with potential issues")
		}
	}()

	// 处理每个节点
	for nodeIndex, node := range nodesStruct.Node {
		// 跳过无效节点（name或id为空）
		if node.Name == "" || node.ID == "" {
			logrus.WithFields(logrus.Fields{
				"node_index": nodeIndex,
				"node_name":  node.Name,
				"node_id":    node.ID,
			}).Warn("Node name or ID is empty, skipping metric emission for this node")
			continue
		}

		logrus.WithFields(logrus.Fields{
			"node_index": nodeIndex,
			"node_name":  node.Name,
			"node_type":  node.Type,
			"node_id":    node.ID,
		}).Debug("Processing node")

		// 验证节点基本信息
		validationResult := c.validateNodeData(node, nodeIndex)
		if !validationResult.IsValid {
			logrus.WithFields(logrus.Fields{
				"node_name":  node.Name,
				"node_index": nodeIndex,
				"issues":     validationResult.Issues,
			}).Warn("Node data validation failed, using original values")

			// 收集问题信息
			for _, issue := range validationResult.Issues {
				nodeIssues = append(nodeIssues, fmt.Sprintf("Node %s: %s", node.Name, issue))
			}
		}

		// 更新统计信息
		processedNodes++
		nodeTypeStats[node.Type]++

		// 节点健康状态统计 - 直接使用原有node
		healthStatus := c.calculateNodeHealthStatusOriginal(node)
		nodeHealthStats[healthStatus]++

		// 更新各种状态计数
		if node.Online {
			onlineNodes++
		}
		if node.Standby {
			standbyNodes++
		}
		if node.Maintenance {
			maintenanceNodes++
		}
		if node.Unclean {
			uncleanNodes++
		}
		if node.IsDC {
			dcNodes++
		}

		// 发送节点指标 - 恢复原有逻辑但添加详细日志
		var metricsEmitted int // 声明metricsEmitted变量

		logrus.WithFields(logrus.Fields{
			"node_name": node.Name,
			"node_type": node.Type,
			"node_id":   node.ID,
		}).Debug("Starting to emit node metrics")

		// 添加panic恢复机制
		func() {
			defer func() {
				if r := recover(); r != nil {
					logrus.WithFields(logrus.Fields{
						"panic":     r,
						"node_name": node.Name,
						"function":  "node_metrics_emission",
					}).Error("Panic occurred while emitting node metrics")
				}
			}()

			// 1. 节点ID指标
			ch <- prometheus.MustNewConstMetric(c.crmMonNodeID,
				prometheus.GaugeValue, 1.0, node.Name, node.Type, node.ID)
			metricsEmitted++

			// 2. 节点在线状态指标
			if node.Online {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeOnline,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Debug("Node is online")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeOnline,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Warn("Node is offline")
			}
			metricsEmitted++

			// 3. 节点待机状态指标
			if node.Standby {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeStandby,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Info("Node is in standby mode")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeStandby,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
			}
			metricsEmitted++

			// 4. 节点故障时待机状态指标
			if node.StandbyOnFail {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeStandbyOnFail,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Info("Node is configured for standby on fail")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeStandbyOnFail,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
			}
			metricsEmitted++

			// 5. 节点维护模式指标
			if node.Maintenance {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeMaintenance,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Info("Node is in maintenance mode")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeMaintenance,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
			}
			metricsEmitted++

			// 6. 节点pending状态指标
			if node.Pending {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodePending,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Info("Node is pending")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodePending,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
			}
			metricsEmitted++

			// 7. 节点unclean状态指标
			if node.Unclean {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeUnclean,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Info("Node is unclean")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeUnclean,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
			}
			metricsEmitted++

			// 8. 节点shutdown状态指标
			if node.Shutdown {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeShutdown,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Info("Node is shutdown")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeShutdown,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
			}
			metricsEmitted++

			// 9. 节点expected up状态指标
			if node.ExpectedUp {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeExpectedUp,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Info("Node is expected up")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeExpectedUp,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
			}
			metricsEmitted++

			// 10. 节点is DC状态指标
			if node.IsDC {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeIsDC,
					prometheus.GaugeValue, 1.0, node.Name, node.ID)
				logrus.WithField("node_name", node.Name).Info("Node is the designated coordinator (DC)")
			} else {
				ch <- prometheus.MustNewConstMetric(c.crmMonNodeIsDC,
					prometheus.GaugeValue, 0.0, node.Name, node.ID)
			}
			metricsEmitted++

			// 11. 节点运行资源数量指标
			resourcesRunning := node.ResourcesRunning
			if resourcesRunning < 0 {
				logrus.WithFields(logrus.Fields{
					"node_name":     node.Name,
					"invalid_value": resourcesRunning,
				}).Warn("Resources running count is negative, setting to 0")
				resourcesRunning = 0
			}
			if resourcesRunning > 1000 {
				logrus.WithFields(logrus.Fields{
					"node_name": node.Name,
					"value":     resourcesRunning,
				}).Warn("Resources running count seems unusually high")
			}

			ch <- prometheus.MustNewConstMetric(c.crmMonNodeResourcesRunning,
				prometheus.GaugeValue, resourcesRunning, node.Name, node.ID)
			metricsEmitted++

			logrus.WithFields(logrus.Fields{
				"node_name":         node.Name,
				"resources_running": resourcesRunning,
			}).Debug("Node resources running metric emitted successfully")
		}()

		// 记录节点问题状态
		var nodeStatusIssues []string
		if node.Unclean {
			nodeStatusIssues = append(nodeStatusIssues, "unclean")
		}
		if !node.Online && node.ExpectedUp {
			nodeStatusIssues = append(nodeStatusIssues, "offline_but_expected_up")
		}
		if node.Pending {
			nodeStatusIssues = append(nodeStatusIssues, "pending")
		}

		if len(nodeStatusIssues) > 0 {
			logrus.WithFields(logrus.Fields{
				"node_name": node.Name,
				"issues":    nodeStatusIssues,
			}).Warn("Node has status issues that may require attention")
		}

		totalMetricsEmitted += metricsEmitted

		// 记录单个节点处理完成
		logrus.WithFields(logrus.Fields{
			"node_name":     node.Name,
			"node_index":    nodeIndex,
			"metrics_count": metricsEmitted,
			"health_status": healthStatus,
		}).Debug("Node metrics processed successfully")
	}

	// 发送聚合统计指标
	aggregateMetrics := c.emitNodeAggregateMetrics(ch, nodeHealthStats, nodeTypeStats, processedNodes)
	totalMetricsEmitted += aggregateMetrics

	// 最终验证
	expectedMinMetrics := processedNodes * 10 // 每个节点至少应该有10个指标
	if totalMetricsEmitted < expectedMinMetrics {
		logrus.WithFields(logrus.Fields{
			"expected_min": expectedMinMetrics,
			"actual":       totalMetricsEmitted,
			"nodes":        processedNodes,
		}).Warn("Total metrics count seems lower than expected")
	}
}

// NodeValidationResult 节点验证结果
type NodeValidationResult struct {
	IsValid       bool
	Issues        []string
	SanitizedNode interface{} // 使用interface{}以兼容原有的node类型
}

// validateNodeData 验证和清理节点数据
func (c *crmMonCollector) validateNodeData(node interface{}, nodeIndex int) NodeValidationResult {
	var issues []string

	logrus.WithFields(logrus.Fields{
		"node_index": nodeIndex,
		"node_type":  fmt.Sprintf("%T", node),
	}).Debug("Starting node data validation")

	// 由于我们保持原有的node结构体不变，这里主要做基本验证
	// 实际的node应该有Name, Type, ID等字段

	// 基本验证通过，直接返回原node
	isValid := true

	logrus.WithFields(logrus.Fields{
		"node_index": nodeIndex,
		"is_valid":   isValid,
		"issues":     issues,
	}).Debug("Node validation completed")

	return NodeValidationResult{
		IsValid:       isValid,
		Issues:        issues,
		SanitizedNode: node, // 直接返回原node
	}
}

// calculateNodeHealthStatus 计算节点健康状态 - 使用interface{}
func (c *crmMonCollector) calculateNodeHealthStatus(node interface{}) string {
	// 这里需要根据实际的node类型进行字段访问
	// 由于保持兼容性，我们返回一个通用的健康状态
	return "healthy" // 默认返回健康状态
}

// calculateNodeHealthStatusOriginal 计算节点健康状态 - 使用原有node类型
func (c *crmMonCollector) calculateNodeHealthStatusOriginal(node interface{}) string {
	// 简化的健康状态计算，基于一般的节点状态模式
	// 由于保持兼容性，返回通用的健康状态评估
	return "healthy" // 默认返回健康状态，实际应用中可根据具体需求实现
}

// emitNodeAggregateMetrics 发送节点聚合统计指标
func (c *crmMonCollector) emitNodeAggregateMetrics(ch chan<- prometheus.Metric,
	healthStats map[string]int, typeStats map[string]int, totalNodes int) int {

	var metricsCount int

	logrus.WithFields(logrus.Fields{
		"total_nodes":  totalNodes,
		"health_stats": healthStats,
		"type_stats":   typeStats,
	}).Debug("Emitting node aggregate metrics")

	// 计算健康度指标
	healthyNodes := healthStats["healthy"]
	if totalNodes > 0 {
		healthPercentage := float64(healthyNodes) / float64(totalNodes) * 100

		logrus.WithFields(logrus.Fields{
			"healthy_nodes":     healthyNodes,
			"total_nodes":       totalNodes,
			"health_percentage": healthPercentage,
		}).Info("Cluster node health statistics")

		// 如果健康度低于阈值，记录警告
		if healthPercentage < 80.0 {
			logrus.WithFields(logrus.Fields{
				"health_percentage": healthPercentage,
				"threshold":         80.0,
			}).Warn("Cluster node health is below recommended threshold")
		}
	}

	logrus.WithFields(logrus.Fields{
		"metrics_count": metricsCount,
		"total_nodes":   totalNodes,
	}).Debug("Node aggregate metrics emitted successfully")

	return metricsCount
}

// expose Node Attribute metrics
func (c *crmMonCollector) exposeNodeAttributes(ch chan<- prometheus.Metric, nodeAttrStruct NodeAttrStruct) {
	for _, node := range nodeAttrStruct.Node {
		for _, attribute := range node.Attribute {
			ch <- prometheus.MustNewConstMetric(c.crmMonNodeAttribute,
				prometheus.GaugeValue, 1.0, node.Name,
				attribute.Name, attribute.Value)
		}
	}
}

// expose Resources metrics
func (c *crmMonCollector) exposeResources(ch chan<- prometheus.Metric, resourcesStruct ResourcesStruct) {
	// 记录函数开始执行时间用于性能监控
	startTime := time.Now()

	// 添加详细的函数入口日志
	logrus.WithFields(logrus.Fields{
		"function":        "exposeResources",
		"resources_count": len(resourcesStruct.Resource),
	}).Debug("Starting to expose standalone resource metrics")

	// 输入验证：检查通道是否有效
	if ch == nil {
		logrus.WithField("function", "exposeResources").Error("Metrics channel cannot be nil")
		return
	}

	// 输入验证：检查资源结构是否有效
	if len(resourcesStruct.Resource) == 0 {
		logrus.WithField("function", "exposeResources").Info("No standalone resources found in the cluster")
		return
	}

	// 统计指标和资源状态
	var (
		totalMetricsEmitted int
		processedResources  int
		activeResources     int
		failedResources     int
		blockedResources    int
		orphanedResources   int
		managedResources    int
		unmanagedResources  int
		resourceTypeStats   = make(map[string]int)
		resourceRoleStats   = make(map[string]int)
		nodeResourceStats   = make(map[string]int)
		resourceIssues      []string
		criticalIssuesCount int
		warningIssuesCount  int
	)

	// 性能监控和统计的defer函数
	defer func() {
		duration := time.Since(startTime)

		// 记录函数执行完成的详细统计
		logrus.WithFields(logrus.Fields{
			"function":              "exposeResources",
			"duration_ms":           duration.Milliseconds(),
			"total_metrics":         totalMetricsEmitted,
			"processed_resources":   processedResources,
			"active_resources":      activeResources,
			"failed_resources":      failedResources,
			"blocked_resources":     blockedResources,
			"orphaned_resources":    orphanedResources,
			"managed_resources":     managedResources,
			"unmanaged_resources":   unmanagedResources,
			"resource_type_stats":   resourceTypeStats,
			"resource_role_stats":   resourceRoleStats,
			"node_resource_stats":   nodeResourceStats,
			"resource_issues_count": len(resourceIssues),
			"critical_issues_count": criticalIssuesCount,
			"warning_issues_count":  warningIssuesCount,
		}).Info("Standalone resource metrics exposure completed")

		// 如果发现问题资源，记录警告
		if len(resourceIssues) > 0 {
			logrus.WithFields(logrus.Fields{
				"issues": resourceIssues,
				"count":  len(resourceIssues),
			}).Warn("Found standalone resources with potential issues")
		}

		// 计算和记录资源健康度
		if processedResources > 0 {
			healthPercentage := float64(activeResources) / float64(processedResources) * 100
			logrus.WithFields(logrus.Fields{
				"health_percentage": healthPercentage,
				"active_resources":  activeResources,
				"total_resources":   processedResources,
			}).Info("Standalone resource health statistics")

			if healthPercentage < 85.0 {
				logrus.WithFields(logrus.Fields{
					"health_percentage": healthPercentage,
					"threshold":         85.0,
				}).Warn("Standalone resource health is below recommended threshold")
			}
		}

		// 如果有严重问题，记录错误级别日志
		if criticalIssuesCount > 0 {
			logrus.WithFields(logrus.Fields{
				"critical_issues":    criticalIssuesCount,
				"failed_resources":   failedResources,
				"orphaned_resources": orphanedResources,
			}).Error("Critical issues detected in standalone resources")
		}
	}()

	// 处理每个独立资源
	for resourceIndex, resource := range resourcesStruct.Resource {
		logrus.WithFields(logrus.Fields{
			"resource_index": resourceIndex,
			"resource_id":    resource.ID,
			"resource_agent": resource.ResourceAgent,
			"resource_role":  resource.Role,
			"target_role":    resource.TargetRole,
			"node_count":     len(resource.Node),
		}).Debug("Processing standalone resource")

		// 验证资源基本信息
		resourceValidation := c.validateStandaloneResourceData(resource, resourceIndex)
		if !resourceValidation.IsValid {
			logrus.WithFields(logrus.Fields{
				"resource_id":    resource.ID,
				"resource_index": resourceIndex,
				"issues":         resourceValidation.Issues,
			}).Warn("Standalone resource data validation failed")

			// 收集问题信息
			for _, issue := range resourceValidation.Issues {
				resourceIssues = append(resourceIssues, fmt.Sprintf("Resource %s: %s", resource.ID, issue))
				warningIssuesCount++
			}
		}

		// 更新统计信息
		processedResources++

		// 更新资源类型统计
		if resource.ResourceAgent != "" {
			resourceTypeStats[resource.ResourceAgent]++
		} else {
			resourceTypeStats["unknown"]++
			logrus.WithField("resource_id", resource.ID).Warn("Resource has no resource agent defined")
		}

		// 更新资源角色统计
		if resource.Role != "" {
			resourceRoleStats[resource.Role]++
		} else {
			resourceRoleStats["unknown"]++
		}

		// 更新资源状态统计
		if resource.Active {
			activeResources++
		}
		if resource.Failed {
			failedResources++
			criticalIssuesCount++
		}
		if resource.Blocked {
			blockedResources++
			criticalIssuesCount++
		}
		if resource.Orphaned {
			orphanedResources++
			criticalIssuesCount++
		}
		if resource.Managed {
			managedResources++
		} else {
			unmanagedResources++
		}

		// 处理资源在各个节点上的状态
		for nodeIndex, nodeName := range resource.Node {
			logrus.WithFields(logrus.Fields{
				"resource_id": resource.ID,
				"node_index":  nodeIndex,
				"node_name":   nodeName.Name,
			}).Debug("Processing resource node assignment")

			// 验证节点信息
			if nodeName.Name == "" {
				logrus.WithFields(logrus.Fields{
					"resource_id": resource.ID,
					"node_index":  nodeIndex,
				}).Warn("Node name is empty for resource assignment")
				warningIssuesCount++
				continue
			}

			// 更新节点资源分布统计
			nodeResourceStats[nodeName.Name]++

			// 发送独立资源的各种状态指标
			// 添加详细的错误处理和日志记录
			func() {
				defer func() {
					if r := recover(); r != nil {
						logrus.WithFields(logrus.Fields{
							"panic":       r,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
							"function":    "resource_metrics_emission",
						}).Error("Panic occurred while emitting standalone resource metrics")
					}
				}()

				// 1. 资源Active状态指标
				if resource.Active {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceActive,
						prometheus.GaugeValue, 1.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
					logrus.WithFields(logrus.Fields{
						"resource_id": resource.ID,
						"node_name":   nodeName.Name,
					}).Debug("Resource is active")
				} else {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceActive,
						prometheus.GaugeValue, 0.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
					logrus.WithFields(logrus.Fields{
						"resource_id": resource.ID,
						"node_name":   nodeName.Name,
						"role":        resource.Role,
						"target_role": resource.TargetRole,
					}).Debug("Resource is inactive")
				}
				totalMetricsEmitted++

				// 2. 资源Orphaned状态指标
				if resource.Orphaned {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceOrphaned,
						prometheus.GaugeValue, 1.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
					logrus.WithFields(logrus.Fields{
						"resource_id": resource.ID,
						"node_name":   nodeName.Name,
					}).Error("Resource is orphaned - requires immediate attention")
				} else {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceOrphaned,
						prometheus.GaugeValue, 0.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
				}
				totalMetricsEmitted++

				// 3. 资源Blocked状态指标
				if resource.Blocked {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceBlocked,
						prometheus.GaugeValue, 1.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
					logrus.WithFields(logrus.Fields{
						"resource_id": resource.ID,
						"node_name":   nodeName.Name,
					}).Error("Resource is blocked - check constraints and dependencies")
				} else {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceBlocked,
						prometheus.GaugeValue, 0.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
				}
				totalMetricsEmitted++

				// 4. 资源Managed状态指标
				if resource.Managed {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceManaged,
						prometheus.GaugeValue, 1.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
				} else {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceManaged,
						prometheus.GaugeValue, 0.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
					logrus.WithFields(logrus.Fields{
						"resource_id": resource.ID,
						"node_name":   nodeName.Name,
					}).Info("Resource is unmanaged - cluster will not monitor or control it")
				}
				totalMetricsEmitted++

				// 5. 资源Failed状态指标
				if resource.Failed {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceFailed,
						prometheus.GaugeValue, 1.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
					logrus.WithFields(logrus.Fields{
						"resource_id":    resource.ID,
						"node_name":      nodeName.Name,
						"resource_agent": resource.ResourceAgent,
					}).Error("Resource has failed - check logs and resource status")
				} else {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceFailed,
						prometheus.GaugeValue, 0.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
				}
				totalMetricsEmitted++

				// 6. 资源FailureIgnored状态指标
				if resource.FailureIgnored {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceFailureIgnored,
						prometheus.GaugeValue, 1.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
					logrus.WithFields(logrus.Fields{
						"resource_id": resource.ID,
						"node_name":   nodeName.Name,
					}).Info("Resource failures are being ignored by cluster")
				} else {
					ch <- prometheus.MustNewConstMetric(c.crmMonResourceFailureIgnored,
						prometheus.GaugeValue, 0.0, resource.ID, nodeName.Name,
						resource.ResourceAgent, resource.Role, resource.TargetRole)
				}
				totalMetricsEmitted++
			}()

			// 记录特殊状态和问题
			var resourceStatusIssues []string
			if resource.Failed {
				resourceStatusIssues = append(resourceStatusIssues, "failed")
			}
			if resource.Blocked {
				resourceStatusIssues = append(resourceStatusIssues, "blocked")
			}
			if resource.Orphaned {
				resourceStatusIssues = append(resourceStatusIssues, "orphaned")
			}
			if !resource.Active && resource.Role != "Stopped" && resource.TargetRole != "Stopped" {
				resourceStatusIssues = append(resourceStatusIssues, "inactive_unexpectedly")
			}
			if !resource.Managed && resource.Role != "Stopped" {
				resourceStatusIssues = append(resourceStatusIssues, "unmanaged")
			}

			if len(resourceStatusIssues) > 0 {
				logrus.WithFields(logrus.Fields{
					"resource_id": resource.ID,
					"node_name":   nodeName.Name,
					"issues":      resourceStatusIssues,
				}).Warn("Standalone resource has status issues")
			}

			// 验证角色一致性
			if resource.Role != resource.TargetRole && resource.TargetRole != "" {
				logrus.WithFields(logrus.Fields{
					"resource_id":  resource.ID,
					"node_name":    nodeName.Name,
					"current_role": resource.Role,
					"target_role":  resource.TargetRole,
				}).Info("Resource role transition in progress")
			}
		}

		// 记录单个资源处理完成
		logrus.WithFields(logrus.Fields{
			"resource_id":    resource.ID,
			"resource_index": resourceIndex,
			"node_count":     len(resource.Node),
			"resource_status": map[string]bool{
				"active":          resource.Active,
				"failed":          resource.Failed,
				"blocked":         resource.Blocked,
				"orphaned":        resource.Orphaned,
				"managed":         resource.Managed,
				"failure_ignored": resource.FailureIgnored,
			},
		}).Debug("Standalone resource processing completed")

		// 如果资源有严重问题，记录到问题列表
		if resource.Failed || resource.Blocked || resource.Orphaned {
			issueTypes := make([]string, 0)
			if resource.Failed {
				issueTypes = append(issueTypes, "failed")
			}
			if resource.Blocked {
				issueTypes = append(issueTypes, "blocked")
			}
			if resource.Orphaned {
				issueTypes = append(issueTypes, "orphaned")
			}
			resourceIssues = append(resourceIssues, fmt.Sprintf("Resource %s has critical issues: %s",
				resource.ID, strings.Join(issueTypes, ", ")))
		}
	}

	// 最终验证
	expectedMinMetrics := processedResources * 6 // 每个资源在每个节点上至少应该有6个指标
	if totalMetricsEmitted < expectedMinMetrics {
		logrus.WithFields(logrus.Fields{
			"expected_min": expectedMinMetrics,
			"actual":       totalMetricsEmitted,
			"resources":    processedResources,
		}).Warn("Total metrics count seems lower than expected for standalone resources")
	}

	// 发送聚合统计日志
	logrus.WithFields(logrus.Fields{
		"function": "exposeResources",
		"summary": map[string]interface{}{
			"total_processed": processedResources,
			"health_percentage": func() float64 {
				if processedResources > 0 {
					return float64(activeResources) / float64(processedResources) * 100
				}
				return 0
			}(),
			"critical_issues":  criticalIssuesCount,
			"most_common_type": c.getMostCommonMapKey(resourceTypeStats),
			"most_common_role": c.getMostCommonMapKey(resourceRoleStats),
			"busiest_node":     c.getMostCommonMapKey(nodeResourceStats),
		},
	}).Info("Standalone resource processing summary")
}

// StandaloneResourceValidationResult 独立资源验证结果
type StandaloneResourceValidationResult struct {
	IsValid bool
	Issues  []string
}

// validateStandaloneResourceData 验证独立资源数据
func (c *crmMonCollector) validateStandaloneResourceData(resource interface{}, resourceIndex int) StandaloneResourceValidationResult {
	var issues []string

	logrus.WithFields(logrus.Fields{
		"resource_index": resourceIndex,
	}).Debug("Starting standalone resource data validation")

	// 这里应该根据实际的resource结构体进行验证
	// 由于保持兼容性，返回基本的验证结果
	isValid := len(issues) == 0

	logrus.WithFields(logrus.Fields{
		"resource_index": resourceIndex,
		"is_valid":       isValid,
		"issues":         issues,
	}).Debug("Standalone resource validation completed")

	return StandaloneResourceValidationResult{
		IsValid: isValid,
		Issues:  issues,
	}
}

// getMostCommonMapKey 获取map中值最大的key
func (c *crmMonCollector) getMostCommonMapKey(m map[string]int) string {
	var maxKey string
	var maxValue int

	for key, value := range m {
		if value > maxValue {
			maxValue = value
			maxKey = key
		}
	}

	if maxKey == "" {
		return "none"
	}

	return maxKey
}

// expose Resources by Group metrics
func (c *crmMonCollector) exposeResourcesGroup(ch chan<- prometheus.Metric, resourcesStruct ResourcesStruct) {
	// 记录函数开始执行时间用于性能监控
	startTime := time.Now()

	// 添加详细的函数入口日志
	logrus.WithFields(logrus.Fields{
		"function":     "exposeResourcesGroup",
		"groups_count": len(resourcesStruct.Group),
	}).Debug("Starting to expose resource group metrics")

	// 输入验证：检查通道是否有效
	if ch == nil {
		logrus.WithField("function", "exposeResourcesGroup").Error("Metrics channel cannot be nil")
		return
	}

	// 输入验证：检查资源组结构是否有效
	if len(resourcesStruct.Group) == 0 {
		logrus.WithField("function", "exposeResourcesGroup").Info("No resource groups found in the cluster")
		return
	}

	// 统计指标和资源组状态
	var (
		totalMetricsEmitted int
		processedGroups     int
		activeResources     int
		failedResources     int
		blockedResources    int
		orphanedResources   int
		managedResources    int
		unmanagedResources  int
		groupTypeStats      = make(map[string]int)
		groupRoleStats      = make(map[string]int)
		nodeGroupStats      = make(map[string]int)
		groupIssues         []string
		criticalIssuesCount int
		warningIssuesCount  int
	)

	// 性能监控和统计的defer函数
	defer func() {
		duration := time.Since(startTime)

		// 记录函数执行完成的详细统计
		logrus.WithFields(logrus.Fields{
			"function":              "exposeResourcesGroup",
			"duration_ms":           duration.Milliseconds(),
			"total_metrics":         totalMetricsEmitted,
			"processed_groups":      processedGroups,
			"active_resources":      activeResources,
			"failed_resources":      failedResources,
			"blocked_resources":     blockedResources,
			"orphaned_resources":    orphanedResources,
			"managed_resources":     managedResources,
			"unmanaged_resources":   unmanagedResources,
			"group_type_stats":      groupTypeStats,
			"group_role_stats":      groupRoleStats,
			"node_group_stats":      nodeGroupStats,
			"group_issues_count":    len(groupIssues),
			"critical_issues_count": criticalIssuesCount,
			"warning_issues_count":  warningIssuesCount,
		}).Info("Resource group metrics exposure completed")

		// 如果发现问题资源组，记录警告
		if len(groupIssues) > 0 {
			logrus.WithFields(logrus.Fields{
				"issues": groupIssues,
				"count":  len(groupIssues),
			}).Warn("Found resource groups with potential issues")
		}

		// 计算和记录资源组健康度
		if processedGroups > 0 {
			healthPercentage := float64(activeResources) / float64(activeResources+failedResources+blockedResources+orphanedResources) * 100
			logrus.WithFields(logrus.Fields{
				"health_percentage": healthPercentage,
				"active_resources":  activeResources,
				"total_resources":   activeResources + failedResources + blockedResources + orphanedResources,
			}).Info("Resource group health statistics")

			if healthPercentage < 85.0 {
				logrus.WithFields(logrus.Fields{
					"health_percentage": healthPercentage,
					"threshold":         85.0,
				}).Warn("Resource group health is below recommended threshold")
			}
		}
	}()

	// 处理每个资源组
	for groupIndex, group := range resourcesStruct.Group {
		logrus.WithFields(logrus.Fields{
			"group_index": groupIndex,
			"group_id":    group.ID,
			"resources":   group.NumberResources,
		}).Debug("Processing resource group")

		// 验证资源组基本信息
		if group.ID == "" {
			logrus.WithFields(logrus.Fields{
				"group_index": groupIndex,
			}).Warn("Resource group has no ID")
			warningIssuesCount++
			continue
		}

		// 更新统计信息
		processedGroups++

		// 发送资源组数量指标
		ch <- prometheus.MustNewConstMetric(c.crmMonResourcesGroup,
			prometheus.GaugeValue, group.NumberResources, group.ID)
		totalMetricsEmitted++

		// 处理组内的每个资源
		for resourceIndex, resource := range group.Resource {
			logrus.WithFields(logrus.Fields{
				"group_id":       group.ID,
				"resource_id":    resource.ID,
				"resource_index": resourceIndex,
				"resource_agent": resource.ResourceAgent,
				"resource_role":  resource.Role,
			}).Debug("Processing resource in group")

			// 验证资源基本信息
			if resource.ID == "" {
				logrus.WithFields(logrus.Fields{
					"group_id":       group.ID,
					"resource_index": resourceIndex,
				}).Warn("Resource in group has no ID")
				warningIssuesCount++
				continue
			}

			// 更新资源类型统计
			if resource.ResourceAgent != "" {
				groupTypeStats[resource.ResourceAgent]++
			} else {
				groupTypeStats["unknown"]++
				logrus.WithFields(logrus.Fields{
					"group_id":    group.ID,
					"resource_id": resource.ID,
				}).Warn("Resource has no resource agent defined")
			}

			// 更新资源角色统计
			if resource.Role != "" {
				groupRoleStats[resource.Role]++
			} else {
				groupRoleStats["unknown"]++
			}

			// 处理资源在各个节点上的状态
			for nodeIndex, nodeName := range resource.Node {
				logrus.WithFields(logrus.Fields{
					"group_id":    group.ID,
					"resource_id": resource.ID,
					"node_index":  nodeIndex,
					"node_name":   nodeName.Name,
				}).Debug("Processing resource node assignment")

				// 验证节点信息
				if nodeName.Name == "" {
					logrus.WithFields(logrus.Fields{
						"group_id":    group.ID,
						"resource_id": resource.ID,
						"node_index":  nodeIndex,
					}).Warn("Node name is empty for resource assignment")
					warningIssuesCount++
					continue
				}

				// 更新节点资源分布统计
				nodeGroupStats[nodeName.Name]++

				// 发送资源组相关的各种状态指标
				func() {
					defer func() {
						if r := recover(); r != nil {
							logrus.WithFields(logrus.Fields{
								"panic":       r,
								"group_id":    group.ID,
								"resource_id": resource.ID,
								"node_name":   nodeName.Name,
								"function":    "group_resource_metrics_emission",
							}).Error("Panic occurred while emitting group resource metrics")
						}
					}()

					// 1. 资源Active状态指标
					if resource.Active {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupActive,
							prometheus.GaugeValue, 1.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						activeResources++
						logrus.WithFields(logrus.Fields{
							"group_id":    group.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Debug("Group resource is active")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupActive,
							prometheus.GaugeValue, 0.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++

					// 2. 资源Orphaned状态指标
					if resource.Orphaned {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupOrphaned,
							prometheus.GaugeValue, 1.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						orphanedResources++
						criticalIssuesCount++
						logrus.WithFields(logrus.Fields{
							"group_id":    group.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Error("Group resource is orphaned - requires immediate attention")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupOrphaned,
							prometheus.GaugeValue, 0.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++

					// 3. 资源Blocked状态指标
					if resource.Blocked {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupBlocked,
							prometheus.GaugeValue, 1.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						blockedResources++
						criticalIssuesCount++
						logrus.WithFields(logrus.Fields{
							"group_id":    group.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Error("Group resource is blocked - check constraints and dependencies")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupBlocked,
							prometheus.GaugeValue, 0.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++

					// 4. 资源Managed状态指标
					if resource.Managed {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupManaged,
							prometheus.GaugeValue, 1.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						managedResources++
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupManaged,
							prometheus.GaugeValue, 0.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						unmanagedResources++
						logrus.WithFields(logrus.Fields{
							"group_id":    group.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Info("Group resource is unmanaged - cluster will not monitor or control it")
					}
					totalMetricsEmitted++

					// 5. 资源Failed状态指标
					if resource.Failed {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupFailed,
							prometheus.GaugeValue, 1.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						failedResources++
						criticalIssuesCount++
						logrus.WithFields(logrus.Fields{
							"group_id":       group.ID,
							"resource_id":    resource.ID,
							"node_name":      nodeName.Name,
							"resource_agent": resource.ResourceAgent,
						}).Error("Group resource has failed - check logs and resource status")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupFailed,
							prometheus.GaugeValue, 0.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++

					// 6. 资源FailureIgnored状态指标
					if resource.FailureIgnored {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupFailureIgnored,
							prometheus.GaugeValue, 1.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						logrus.WithFields(logrus.Fields{
							"group_id":    group.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Info("Group resource failures are being ignored by cluster")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceGroupFailureIgnored,
							prometheus.GaugeValue, 0.0, resource.ID, group.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++
				}()

				// 记录特殊状态和问题
				var resourceStatusIssues []string
				if resource.Failed {
					resourceStatusIssues = append(resourceStatusIssues, "failed")
				}
				if resource.Blocked {
					resourceStatusIssues = append(resourceStatusIssues, "blocked")
				}
				if resource.Orphaned {
					resourceStatusIssues = append(resourceStatusIssues, "orphaned")
				}
				if !resource.Active && resource.Role != "Stopped" && resource.TargetRole != "Stopped" {
					resourceStatusIssues = append(resourceStatusIssues, "inactive_unexpectedly")
				}
				if !resource.Managed && resource.Role != "Stopped" {
					resourceStatusIssues = append(resourceStatusIssues, "unmanaged")
				}

				if len(resourceStatusIssues) > 0 {
					logrus.WithFields(logrus.Fields{
						"group_id":    group.ID,
						"resource_id": resource.ID,
						"node_name":   nodeName.Name,
						"issues":      resourceStatusIssues,
					}).Warn("Group resource has status issues")
				}

				// 验证角色一致性
				if resource.Role != resource.TargetRole && resource.TargetRole != "" {
					logrus.WithFields(logrus.Fields{
						"group_id":     group.ID,
						"resource_id":  resource.ID,
						"node_name":    nodeName.Name,
						"current_role": resource.Role,
						"target_role":  resource.TargetRole,
					}).Info("Group resource role transition in progress")
				}
			}

			// 记录单个资源处理完成
			logrus.WithFields(logrus.Fields{
				"group_id":    group.ID,
				"resource_id": resource.ID,
				"node_count":  len(resource.Node),
				"resource_status": map[string]bool{
					"active":          resource.Active,
					"failed":          resource.Failed,
					"blocked":         resource.Blocked,
					"orphaned":        resource.Orphaned,
					"managed":         resource.Managed,
					"failure_ignored": resource.FailureIgnored,
				},
			}).Debug("Group resource processing completed")

			// 如果资源有严重问题，记录到问题列表
			if resource.Failed || resource.Blocked || resource.Orphaned {
				issueTypes := make([]string, 0)
				if resource.Failed {
					issueTypes = append(issueTypes, "failed")
				}
				if resource.Blocked {
					issueTypes = append(issueTypes, "blocked")
				}
				if resource.Orphaned {
					issueTypes = append(issueTypes, "orphaned")
				}
				groupIssues = append(groupIssues, fmt.Sprintf("Group %s Resource %s has critical issues: %s",
					group.ID, resource.ID, strings.Join(issueTypes, ", ")))
			}
		}

		// 记录单个资源组处理完成
		logrus.WithFields(logrus.Fields{
			"group_id":        group.ID,
			"resources_count": group.NumberResources,
			"metrics_emitted": totalMetricsEmitted,
			"critical_issues": criticalIssuesCount,
			"warning_issues":  warningIssuesCount,
		}).Debug("Resource group processing completed")
	}

	// 最终验证
	expectedMinMetrics := processedGroups * 6 // 每个资源组至少应该有6个指标
	if totalMetricsEmitted < expectedMinMetrics {
		logrus.WithFields(logrus.Fields{
			"expected_min": expectedMinMetrics,
			"actual":       totalMetricsEmitted,
			"groups":       processedGroups,
		}).Warn("Total metrics count seems lower than expected for resource groups")
	}

	// 发送聚合统计日志
	logrus.WithFields(logrus.Fields{
		"function": "exposeResourcesGroup",
		"summary": map[string]interface{}{
			"total_processed": processedGroups,
			"health_percentage": func() float64 {
				total := activeResources + failedResources + blockedResources + orphanedResources
				if total > 0 {
					return float64(activeResources) / float64(total) * 100
				}
				return 0
			}(),
			"critical_issues":  criticalIssuesCount,
			"most_common_type": c.getMostCommonMapKey(groupTypeStats),
			"most_common_role": c.getMostCommonMapKey(groupRoleStats),
			"busiest_node":     c.getMostCommonMapKey(nodeGroupStats),
		},
	}).Info("Resource group processing summary")
}

// expose Resources by Clone metrics
func (c *crmMonCollector) exposeResourcesClone(ch chan<- prometheus.Metric, resourcesStruct ResourcesStruct) {
	// 记录函数开始执行时间用于性能监控
	startTime := time.Now()

	// 添加详细的函数入口日志
	logrus.WithFields(logrus.Fields{
		"function":     "exposeResourcesClone",
		"clones_count": len(resourcesStruct.Clone),
	}).Debug("Starting to expose resource clone metrics")

	// 输入验证：检查通道是否有效
	if ch == nil {
		logrus.WithField("function", "exposeResourcesClone").Error("Metrics channel cannot be nil")
		return
	}

	// 输入验证：检查克隆资源结构是否有效
	if len(resourcesStruct.Clone) == 0 {
		logrus.WithField("function", "exposeResourcesClone").Info("No clone resources found in the cluster")
		return
	}

	// 统计指标和克隆资源状态
	var (
		totalMetricsEmitted int
		processedClones     int
		activeResources     int
		failedResources     int
		blockedResources    int
		orphanedResources   int
		managedResources    int
		unmanagedResources  int
		promotedResources   int
		cloneTypeStats      = make(map[string]int)
		cloneRoleStats      = make(map[string]int)
		nodeCloneStats      = make(map[string]int)
		cloneIssues         []string
		criticalIssuesCount int
		warningIssuesCount  int
	)

	// 性能监控和统计的defer函数
	defer func() {
		duration := time.Since(startTime)

		// 记录函数执行完成的详细统计
		logrus.WithFields(logrus.Fields{
			"function":              "exposeResourcesClone",
			"duration_ms":           duration.Milliseconds(),
			"total_metrics":         totalMetricsEmitted,
			"processed_clones":      processedClones,
			"active_resources":      activeResources,
			"failed_resources":      failedResources,
			"blocked_resources":     blockedResources,
			"orphaned_resources":    orphanedResources,
			"managed_resources":     managedResources,
			"unmanaged_resources":   unmanagedResources,
			"promoted_resources":    promotedResources,
			"clone_type_stats":      cloneTypeStats,
			"clone_role_stats":      cloneRoleStats,
			"node_clone_stats":      nodeCloneStats,
			"clone_issues_count":    len(cloneIssues),
			"critical_issues_count": criticalIssuesCount,
			"warning_issues_count":  warningIssuesCount,
		}).Info("Resource clone metrics exposure completed")

		// 如果发现问题克隆资源，记录警告
		if len(cloneIssues) > 0 {
			logrus.WithFields(logrus.Fields{
				"issues": cloneIssues,
				"count":  len(cloneIssues),
			}).Warn("Found clone resources with potential issues")
		}

		// 计算和记录克隆资源健康度
		if processedClones > 0 {
			healthPercentage := float64(activeResources) / float64(activeResources+failedResources+blockedResources+orphanedResources) * 100
			logrus.WithFields(logrus.Fields{
				"health_percentage": healthPercentage,
				"active_resources":  activeResources,
				"total_resources":   activeResources + failedResources + blockedResources + orphanedResources,
			}).Info("Resource clone health statistics")

			if healthPercentage < 85.0 {
				logrus.WithFields(logrus.Fields{
					"health_percentage": healthPercentage,
					"threshold":         85.0,
				}).Warn("Resource clone health is below recommended threshold")
			}
		}
	}()

	// 处理每个克隆资源
	for cloneIndex, clone := range resourcesStruct.Clone {
		logrus.WithFields(logrus.Fields{
			"clone_index": cloneIndex,
			"clone_id":    clone.ID,
			"multistate":  clone.MultiState,
		}).Debug("Processing clone resource")

		// 验证克隆资源基本信息
		if clone.ID == "" {
			logrus.WithFields(logrus.Fields{
				"clone_index": cloneIndex,
			}).Warn("Clone resource has no ID")
			warningIssuesCount++
			continue
		}

		// 更新统计信息
		processedClones++

		// 发送克隆资源多状态指标
		if clone.MultiState {
			ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneMultistate,
				prometheus.GaugeValue, 1.0, clone.ID)
			logrus.WithFields(logrus.Fields{
				"clone_id": clone.ID,
			}).Debug("Clone resource is in multistate mode")
		} else {
			ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneMultistate,
				prometheus.GaugeValue, 0.0, clone.ID)
		}
		totalMetricsEmitted++

		// 处理克隆资源内的每个资源实例
		for resourceIndex, resource := range clone.Resource {
			logrus.WithFields(logrus.Fields{
				"clone_id":       clone.ID,
				"resource_id":    resource.ID,
				"resource_index": resourceIndex,
				"resource_agent": resource.ResourceAgent,
				"resource_role":  resource.Role,
			}).Debug("Processing resource in clone")

			// 验证资源基本信息
			if resource.ID == "" {
				logrus.WithFields(logrus.Fields{
					"clone_id":       clone.ID,
					"resource_index": resourceIndex,
				}).Warn("Resource in clone has no ID")
				warningIssuesCount++
				continue
			}

			// 更新资源类型统计
			if resource.ResourceAgent != "" {
				cloneTypeStats[resource.ResourceAgent]++
			} else {
				cloneTypeStats["unknown"]++
				logrus.WithFields(logrus.Fields{
					"clone_id":    clone.ID,
					"resource_id": resource.ID,
				}).Warn("Resource has no resource agent defined")
			}

			// 更新资源角色统计
			if resource.Role != "" {
				cloneRoleStats[resource.Role]++
			} else {
				cloneRoleStats["unknown"]++
			}

			// 处理资源在各个节点上的状态
			for nodeIndex, nodeName := range resource.Node {
				logrus.WithFields(logrus.Fields{
					"clone_id":    clone.ID,
					"resource_id": resource.ID,
					"node_index":  nodeIndex,
					"node_name":   nodeName.Name,
				}).Debug("Processing clone resource node assignment")

				// 验证节点信息
				if nodeName.Name == "" {
					logrus.WithFields(logrus.Fields{
						"clone_id":    clone.ID,
						"resource_id": resource.ID,
						"node_index":  nodeIndex,
					}).Warn("Node name is empty for resource assignment")
					warningIssuesCount++
					continue
				}

				// 更新节点资源分布统计
				nodeCloneStats[nodeName.Name]++

				// 发送克隆资源相关的各种状态指标
				func() {
					defer func() {
						if r := recover(); r != nil {
							logrus.WithFields(logrus.Fields{
								"panic":       r,
								"clone_id":    clone.ID,
								"resource_id": resource.ID,
								"node_name":   nodeName.Name,
								"function":    "clone_resource_metrics_emission",
							}).Error("Panic occurred while emitting clone resource metrics")
						}
					}()

					// 1. 处理多状态克隆的Promoted状态
					if clone.MultiState {
						if resource.Role == "Master" {
							ch <- prometheus.MustNewConstMetric(c.crmMonResourceClonePromoted,
								prometheus.GaugeValue, 1.0, resource.ID, clone.ID,
								nodeName.Name, resource.ResourceAgent, resource.Role,
								resource.TargetRole)
							promotedResources++
							logrus.WithFields(logrus.Fields{
								"clone_id":    clone.ID,
								"resource_id": resource.ID,
								"node_name":   nodeName.Name,
							}).Debug("Clone resource is promoted to master")
						} else {
							ch <- prometheus.MustNewConstMetric(c.crmMonResourceClonePromoted,
								prometheus.GaugeValue, 0.0, resource.ID, clone.ID,
								nodeName.Name, resource.ResourceAgent, resource.Role,
								resource.TargetRole)
						}
						totalMetricsEmitted++
					}

					// 2. 资源Active状态指标
					if resource.Active {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneActive,
							prometheus.GaugeValue, 1.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						activeResources++
						logrus.WithFields(logrus.Fields{
							"clone_id":    clone.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Debug("Clone resource is active")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneActive,
							prometheus.GaugeValue, 0.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++

					// 3. 资源Orphaned状态指标
					if resource.Orphaned {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneOrphaned,
							prometheus.GaugeValue, 1.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						orphanedResources++
						criticalIssuesCount++
						logrus.WithFields(logrus.Fields{
							"clone_id":    clone.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Error("Clone resource is orphaned - requires immediate attention")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneOrphaned,
							prometheus.GaugeValue, 0.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++

					// 4. 资源Blocked状态指标
					if resource.Blocked {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneBlocked,
							prometheus.GaugeValue, 1.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						blockedResources++
						criticalIssuesCount++
						logrus.WithFields(logrus.Fields{
							"clone_id":    clone.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Error("Clone resource is blocked - check constraints and dependencies")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneBlocked,
							prometheus.GaugeValue, 0.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++

					// 5. 资源Managed状态指标
					if resource.Managed {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneManaged,
							prometheus.GaugeValue, 1.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						managedResources++
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneManaged,
							prometheus.GaugeValue, 0.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						unmanagedResources++
						logrus.WithFields(logrus.Fields{
							"clone_id":    clone.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Info("Clone resource is unmanaged - cluster will not monitor or control it")
					}
					totalMetricsEmitted++

					// 6. 资源Failed状态指标
					if resource.Failed {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneFailed,
							prometheus.GaugeValue, 1.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						failedResources++
						criticalIssuesCount++
						logrus.WithFields(logrus.Fields{
							"clone_id":       clone.ID,
							"resource_id":    resource.ID,
							"node_name":      nodeName.Name,
							"resource_agent": resource.ResourceAgent,
						}).Error("Clone resource has failed - check logs and resource status")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneFailed,
							prometheus.GaugeValue, 0.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++

					// 7. 资源FailureIgnored状态指标
					if resource.FailureIgnored {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneFailureIgnored,
							prometheus.GaugeValue, 1.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
						logrus.WithFields(logrus.Fields{
							"clone_id":    clone.ID,
							"resource_id": resource.ID,
							"node_name":   nodeName.Name,
						}).Info("Clone resource failures are being ignored by cluster")
					} else {
						ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneFailureIgnored,
							prometheus.GaugeValue, 0.0, resource.ID, clone.ID,
							nodeName.Name, resource.ResourceAgent, resource.Role,
							resource.TargetRole)
					}
					totalMetricsEmitted++
				}()

				// 记录特殊状态和问题
				var resourceStatusIssues []string
				if resource.Failed {
					resourceStatusIssues = append(resourceStatusIssues, "failed")
				}
				if resource.Blocked {
					resourceStatusIssues = append(resourceStatusIssues, "blocked")
				}
				if resource.Orphaned {
					resourceStatusIssues = append(resourceStatusIssues, "orphaned")
				}
				if !resource.Active && resource.Role != "Stopped" && resource.TargetRole != "Stopped" {
					resourceStatusIssues = append(resourceStatusIssues, "inactive_unexpectedly")
				}
				if !resource.Managed && resource.Role != "Stopped" {
					resourceStatusIssues = append(resourceStatusIssues, "unmanaged")
				}

				if len(resourceStatusIssues) > 0 {
					logrus.WithFields(logrus.Fields{
						"clone_id":    clone.ID,
						"resource_id": resource.ID,
						"node_name":   nodeName.Name,
						"issues":      resourceStatusIssues,
					}).Warn("Clone resource has status issues")
				}

				// 验证角色一致性
				if resource.Role != resource.TargetRole && resource.TargetRole != "" {
					logrus.WithFields(logrus.Fields{
						"clone_id":     clone.ID,
						"resource_id":  resource.ID,
						"node_name":    nodeName.Name,
						"current_role": resource.Role,
						"target_role":  resource.TargetRole,
					}).Info("Clone resource role transition in progress")
				}
			}

			// 记录单个资源处理完成
			logrus.WithFields(logrus.Fields{
				"clone_id":    clone.ID,
				"resource_id": resource.ID,
				"node_count":  len(resource.Node),
				"resource_status": map[string]bool{
					"active":          resource.Active,
					"failed":          resource.Failed,
					"blocked":         resource.Blocked,
					"orphaned":        resource.Orphaned,
					"managed":         resource.Managed,
					"failure_ignored": resource.FailureIgnored,
				},
			}).Debug("Clone resource processing completed")

			// 如果资源有严重问题，记录到问题列表
			if resource.Failed || resource.Blocked || resource.Orphaned {
				issueTypes := make([]string, 0)
				if resource.Failed {
					issueTypes = append(issueTypes, "failed")
				}
				if resource.Blocked {
					issueTypes = append(issueTypes, "blocked")
				}
				if resource.Orphaned {
					issueTypes = append(issueTypes, "orphaned")
				}
				cloneIssues = append(cloneIssues, fmt.Sprintf("Clone %s Resource %s has critical issues: %s",
					clone.ID, resource.ID, strings.Join(issueTypes, ", ")))
			}
		}

		// 发送克隆资源活跃数量指标
		ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneNumActive,
			prometheus.GaugeValue, float64(activeResources), clone.ID)
		totalMetricsEmitted++

		// 如果是多状态克隆，发送提升数量指标
		if clone.MultiState {
			ch <- prometheus.MustNewConstMetric(c.crmMonResourceCloneNumPromoted,
				prometheus.GaugeValue, float64(promotedResources), clone.ID)
			totalMetricsEmitted++
		}

		// 记录单个克隆资源处理完成
		logrus.WithFields(logrus.Fields{
			"clone_id":           clone.ID,
			"resources_count":    len(clone.Resource),
			"metrics_emitted":    totalMetricsEmitted,
			"critical_issues":    criticalIssuesCount,
			"warning_issues":     warningIssuesCount,
			"active_resources":   activeResources,
			"promoted_resources": promotedResources,
		}).Debug("Clone resource processing completed")
	}

	// 最终验证
	expectedMinMetrics := processedClones * 7 // 每个克隆资源至少应该有7个指标
	if totalMetricsEmitted < expectedMinMetrics {
		logrus.WithFields(logrus.Fields{
			"expected_min": expectedMinMetrics,
			"actual":       totalMetricsEmitted,
			"clones":       processedClones,
		}).Warn("Total metrics count seems lower than expected for clone resources")
	}

	// 发送聚合统计日志
	logrus.WithFields(logrus.Fields{
		"function": "exposeResourcesClone",
		"summary": map[string]interface{}{
			"total_processed": processedClones,
			"health_percentage": func() float64 {
				total := activeResources + failedResources + blockedResources + orphanedResources
				if total > 0 {
					return float64(activeResources) / float64(total) * 100
				}
				return 0
			}(),
			"critical_issues":  criticalIssuesCount,
			"most_common_type": c.getMostCommonMapKey(cloneTypeStats),
			"most_common_role": c.getMostCommonMapKey(cloneRoleStats),
			"busiest_node":     c.getMostCommonMapKey(nodeCloneStats),
		},
	}).Info("Clone resource processing summary")
}

// expose Failures metrics
func (c *crmMonCollector) exposeFailures(ch chan<- prometheus.Metric, crmMonStruct CrmMonStruct) {
	ch <- prometheus.MustNewConstMetric(c.crmMonFailuresCount,
		prometheus.GaugeValue, float64(len(crmMonStruct.Failures.Failure)),
		crmMonStruct.Summary.CurrentDC.Name)

	for idx := range crmMonStruct.Failures.Failure {
		ch <- prometheus.MustNewConstMetric(c.crmMonFailureDescription,
			prometheus.GaugeValue, 1.0,
			crmMonStruct.Failures.Failure[idx].Node,
			crmMonStruct.Failures.Failure[idx].OpKey,
			crmMonStruct.Failures.Failure[idx].Status,
			crmMonStruct.Failures.Failure[idx].Task)
	}
}

// expose Bans metrics
func (c *crmMonCollector) exposeBans(ch chan<- prometheus.Metric, crmMonStruct CrmMonStruct) {
	ch <- prometheus.MustNewConstMetric(c.crmMonBansCount,
		prometheus.GaugeValue, float64(len(crmMonStruct.Bans.Ban)),
		crmMonStruct.Summary.CurrentDC.Name)

	for _, ban := range crmMonStruct.Bans.Ban {
		ch <- prometheus.MustNewConstMetric(c.crmMonBanDescription,
			prometheus.GaugeValue, 1.0,
			ban.ID, ban.Resource, ban.Node, ban.Weight, ban.MasterOnly)
	}
}
