package drbd

import (
	"fmt"
	"hacluster_exporter/internal/metrics/collectors/core"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDrbdCollector_CollectWithError(t *testing.T) {
	tests := []struct {
		name               string
		drbdSetupPath      string
		drbdSplitBrainPath string
		mockJSON           string
		expectedMetrics    map[string]float64
		expectedError      string
		setupSplitBrain    func(t *testing.T, path string)
	}{
		{
			name:          "invalid drbd setup path",
			drbdSetupPath: "/nonexistent/drbd-setup",
			expectedError: "drbdsetup command failed",
		},
		{
			name: "successful parse with single resource",
			mockJSON: `[
				{
					"name": "r0",
					"role": "Primary",
					"devices": [
						{
							"volume": 0,
							"written": 1024,
							"read": 512,
							"al-writes": 10,
							"bm-writes": 5,
							"upper-pending": 0,
							"lower-pending": 0,
							"quorum": true,
							"disk-state": "UpToDate"
						}
					],
					"connections": [
						{
							"peer-node-id": 1,
							"peer-role": "Secondary",
							"peer_devices": [
								{
									"volume": 0,
									"received": 2048,
									"sent": 1024,
									"pending": 0,
									"unacked": 0,
									"peer-disk-state": "UpToDate",
									"percent-in-sync": 100.0
								}
							]
						}
					]
				}
			]`,
			expectedMetrics: map[string]float64{
				"hacluster_drbd_resources{disk_state=\"uptodate\",resource=\"r0\",role=\"Primary\",volume=\"0\"}":                                  1,
				"hacluster_drbd_written{resource=\"r0\",volume=\"0\"}":                                                                             1024,
				"hacluster_drbd_read{resource=\"r0\",volume=\"0\"}":                                                                                512,
				"hacluster_drbd_al_writes{resource=\"r0\",volume=\"0\"}":                                                                           10,
				"hacluster_drbd_bm_writes{resource=\"r0\",volume=\"0\"}":                                                                           5,
				"hacluster_drbd_upper_pending{resource=\"r0\",volume=\"0\"}":                                                                       0,
				"hacluster_drbd_lower_pending{resource=\"r0\",volume=\"0\"}":                                                                       0,
				"hacluster_drbd_quorum{resource=\"r0\",volume=\"0\"}":                                                                              1,
				"hacluster_drbd_connections{peer_disk_state=\"uptodate\",peer_node_id=\"1\",peer_role=\"Secondary\",resource=\"r0\",volume=\"0\"}": 1,
				"hacluster_drbd_connections_sync{peer_node_id=\"1\",resource=\"r0\",volume=\"0\"}":                                                 100.0,
				"hacluster_drbd_connections_received{peer_node_id=\"1\",resource=\"r0\",volume=\"0\"}":                                             2048,
				"hacluster_drbd_connections_sent{peer_node_id=\"1\",resource=\"r0\",volume=\"0\"}":                                                 1024,
				"hacluster_drbd_connections_pending{peer_node_id=\"1\",resource=\"r0\",volume=\"0\"}":                                              0,
				"hacluster_drbd_connections_unacked{peer_node_id=\"1\",resource=\"r0\",volume=\"0\"}":                                              0,
			},
		},
		{
			name: "successful parse with multiple resources",
			mockJSON: `[
				{
					"name": "r0",
					"role": "Primary",
					"devices": [
						{
							"volume": 0,
							"written": 1024,
							"read": 512,
							"quorum": true,
							"disk-state": "UpToDate"
						}
					],
					"connections": [
						{
							"peer-node-id": 1,
							"peer-role": "Secondary",
							"peer_devices": [
								{
									"volume": 0,
									"received": 2048,
									"sent": 1024,
									"pending": 0,
									"unacked": 0,
									"peer-disk-state": "UpToDate",
									"percent-in-sync": 100.0
								}
							]
						}
					]
				},
				{
					"name": "r1",
					"role": "Secondary",
					"devices": [
						{
							"volume": 0,
							"written": 2048,
							"read": 1024,
							"quorum": true,
							"disk-state": "UpToDate"
						}
					],
					"connections": [
						{
							"peer-node-id": 2,
							"peer-role": "Primary",
							"peer_devices": [
								{
									"volume": 0,
									"received": 1024,
									"sent": 2048,
									"pending": 0,
									"unacked": 0,
									"peer-disk-state": "UpToDate",
									"percent-in-sync": 100.0
								}
							]
						}
					]
				}
			]`,
			expectedMetrics: map[string]float64{
				"hacluster_drbd_resources{disk_state=\"uptodate\",resource=\"r0\",role=\"Primary\",volume=\"0\"}":                                  1,
				"hacluster_drbd_resources{disk_state=\"uptodate\",resource=\"r1\",role=\"Secondary\",volume=\"0\"}":                                1,
				"hacluster_drbd_written{resource=\"r0\",volume=\"0\"}":                                                                             1024,
				"hacluster_drbd_written{resource=\"r1\",volume=\"0\"}":                                                                             2048,
				"hacluster_drbd_read{resource=\"r0\",volume=\"0\"}":                                                                                512,
				"hacluster_drbd_read{resource=\"r1\",volume=\"0\"}":                                                                                1024,
				"hacluster_drbd_quorum{resource=\"r0\",volume=\"0\"}":                                                                              1,
				"hacluster_drbd_quorum{resource=\"r1\",volume=\"0\"}":                                                                              1,
				"hacluster_drbd_connections{peer_disk_state=\"uptodate\",peer_node_id=\"1\",peer_role=\"Secondary\",resource=\"r0\",volume=\"0\"}": 1,
				"hacluster_drbd_connections{peer_disk_state=\"uptodate\",peer_node_id=\"2\",peer_role=\"Primary\",resource=\"r1\",volume=\"0\"}":   1,
			},
		},
		{
			name:               "test split brain detection",
			mockJSON:           "[]",
			drbdSplitBrainPath: "/tmp/drbd-split-brain",
			setupSplitBrain: func(t *testing.T, path string) {
				err := os.MkdirAll(path, 0755)
				require.NoError(t, err)

				// 创建分裂脑检测文件，格式：drbd-split-brain-detected-<resource>-<volume>
				splitBrainFile := filepath.Join(path, "drbd-split-brain-detected-r0-0")
				err = os.WriteFile(splitBrainFile, []byte(""), 0644)
				require.NoError(t, err)
			},
			expectedMetrics: map[string]float64{
				"hacluster_drbd_split_brain{resource=\"r0\"}": 1,
			},
		},
		{
			name:          "invalid JSON",
			mockJSON:      "invalid json content",
			expectedError: "status parsing failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置临时目录
			tmpDir := t.TempDir()

			// 如果有模拟JSON，创建模拟的drbd-setup脚本
			var drbdSetupPath string
			if tt.mockJSON != "" {
				drbdSetupPath = createMockDrbdSetup(t, tmpDir, tt.mockJSON)
			} else {
				drbdSetupPath = tt.drbdSetupPath
			}

			// 如果需要设置分裂脑检测
			splitBrainPath := tt.drbdSplitBrainPath
			if splitBrainPath == "" {
				splitBrainPath = filepath.Join(tmpDir, "split-brain")
			}
			if tt.setupSplitBrain != nil {
				tt.setupSplitBrain(t, splitBrainPath)
			}

			// 创建收集器
			var collector *DrbdCollector
			var err error
			if tt.name == "invalid drbd setup path" {
				collector = &DrbdCollector{
					DefaultCollector: core.NewDefaultCollector(subsystem, false),
					drbdsetupPath:    tt.drbdSetupPath,
					metrics:          drbdMetrics{},
				}
			} else {
				collector, err = NewCollector(drbdSetupPath, splitBrainPath, false)
				require.NoError(t, err)
			}

			// 创建指标通道
			ch := make(chan prometheus.Metric)
			done := make(chan struct{})

			go func() {
				err := collector.CollectWithError(ch)
				if tt.expectedError != "" {
					assert.Error(t, err)
					assert.Contains(t, err.Error(), tt.expectedError)
				} else {
					assert.NoError(t, err)
				}
				close(ch)
				close(done)
			}()

			// 收集并验证指标
			if tt.expectedMetrics != nil {
				actualMetrics := make(map[string]float64)
				for m := range ch {
					metric := &dto.Metric{}
					err := m.(prometheus.Metric).Write(metric)
					require.NoError(t, err)

					// 获取指标名称
					desc := m.Desc().String()
					fqNameStart := strings.Index(desc, "fqName: \"") + 9
					fqNameEnd := strings.Index(desc[fqNameStart:], "\"")
					metricName := desc[fqNameStart : fqNameStart+fqNameEnd]

					// 构建标签字符串
					var labelPairs []string
					if metric.Label != nil {
						for _, label := range metric.Label {
							labelPairs = append(labelPairs, fmt.Sprintf("%s=\"%s\"", *label.Name, *label.Value))
						}
						sort.Strings(labelPairs)
					}

					// 构建完整的指标名称（包含标签）
					if len(labelPairs) > 0 {
						metricName = fmt.Sprintf("%s{%s}", metricName, strings.Join(labelPairs, ","))
					}

					// 获取指标值
					var value float64
					if metric.Gauge != nil {
						value = metric.Gauge.GetValue()
					} else if metric.Counter != nil {
						value = metric.Counter.GetValue()
					} else if metric.Untyped != nil {
						value = metric.Untyped.GetValue()
					}

					actualMetrics[metricName] = value
				}

				// 打印实际收集到的指标，用于调试
				t.Logf("Actual metrics collected: %+v", actualMetrics)

				for expectedMetric, expectedValue := range tt.expectedMetrics {
					actualValue, exists := actualMetrics[expectedMetric]
					assert.True(t, exists, "Expected metric %s not found", expectedMetric)
					if exists {
						assert.Equal(t, expectedValue, actualValue, "Metric value mismatch for %s", expectedMetric)
					}
				}
			}

			<-done
		})
	}
}

func createMockDrbdSetup(t *testing.T, tmpDir string, jsonOutput string) string {
	// 将单引号替换为转义的单引号，以防止 shell 解释错误
	jsonOutput = strings.ReplaceAll(jsonOutput, "'", "'\\''")
	content := fmt.Sprintf(`#!/bin/bash
if [ "$1" = "status" ] && [ "$2" = "--json" ]; then
    printf '%s\n' '%s'
    exit 0
fi
exit 1`, jsonOutput, jsonOutput)

	scriptPath := filepath.Join(tmpDir, "mock-drbd-setup")
	err := os.WriteFile(scriptPath, []byte(content), 0755)
	require.NoError(t, err)
	return scriptPath
}

// toFloat64 converts a prometheus metric to float64
func toFloat64(m prometheus.Metric) float64 {
	var v float64
	metric := &dto.Metric{}
	err := m.Write(metric)
	if err != nil {
		panic(err)
	}
	if metric.Gauge != nil {
		v = metric.Gauge.GetValue()
	} else if metric.Counter != nil {
		v = metric.Counter.GetValue()
	} else if metric.Untyped != nil {
		v = metric.Untyped.GetValue()
	} else {
		panic("unknown metric type")
	}
	return v
}

// getMetricName extracts the metric name and labels from prometheus.Desc string
func getMetricName(desc string) string {
	// 从 Desc 字符串中提取指标名称和标签
	// 示例格式：
	// Desc{fqName: "drbd_resources", help: "The DRBD resources", constLabels: {}, variableLabels: [resource role volume disk_state]}

	// 提取 fqName
	fqNameStart := strings.Index(desc, "fqName: \"") + 9
	if fqNameStart < 9 {
		return desc
	}
	fqNameEnd := strings.Index(desc[fqNameStart:], "\"")
	if fqNameEnd < 0 {
		return desc
	}
	metricName := desc[fqNameStart : fqNameStart+fqNameEnd]

	// 提取变量标签名称
	varLabelsStart := strings.Index(desc, "variableLabels: [") + 16
	if varLabelsStart < 16 {
		return metricName
	}
	varLabelsEnd := strings.Index(desc[varLabelsStart:], "]")
	if varLabelsEnd < 0 {
		return metricName
	}
	varLabels := strings.Split(desc[varLabelsStart:varLabelsStart+varLabelsEnd], " ")

	// 提取常量标签值
	constLabelsStart := strings.Index(desc, "constLabels: {") + 13
	if constLabelsStart < 13 {
		return metricName
	}
	constLabelsEnd := strings.Index(desc[constLabelsStart:], "}")
	if constLabelsEnd < 0 {
		return metricName
	}
	constLabels := desc[constLabelsStart : constLabelsStart+constLabelsEnd]

	// 解析标签
	labelMap := make(map[string]string)
	if constLabels != "" {
		pairs := strings.Split(constLabels, ", ")
		for _, pair := range pairs {
			if pair != "" {
				kv := strings.Split(pair, "=")
				if len(kv) == 2 {
					labelMap[kv[0]] = strings.Trim(kv[1], "\"")
				}
			}
		}
	}

	// 构建标签字符串
	var labelPairs []string
	for _, label := range varLabels {
		if label != "" {
			if val, ok := labelMap[label]; ok {
				labelPairs = append(labelPairs, fmt.Sprintf("%s=\"%s\"", label, val))
			}
		}
	}

	if len(labelPairs) > 0 {
		return fmt.Sprintf("%s{%s}", metricName, strings.Join(labelPairs, ","))
	}

	return metricName
}
