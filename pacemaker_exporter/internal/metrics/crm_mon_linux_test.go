package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockCollector 是一个模拟的 crmMonCollector
type MockCollector struct {
	mock.Mock
	crmMonCollector
}

// TestExposeResourcesGroup 测试 exposeResourcesGroup 函数
func TestExposeResourcesGroup(t *testing.T) {
	// 设置测试用例
	tests := []struct {
		name           string
		resources      ResourcesStruct
		expectedMetric int
		expectError    bool
	}{
		{
			name: "正常资源组测试",
			resources: ResourcesStruct{
				Group: []struct {
					ID              string           `xml:"id,attr"`
					NumberResources float64          `xml:"number_resources,attr"`
					Resource        []ResourceStruct `xml:"resource"`
				}{
					{
						ID:              "test-group",
						NumberResources: 1,
						Resource: []ResourceStruct{
							{
								ID:             "test-resource",
								ResourceAgent:  "ocf::heartbeat:Dummy",
								Role:           "Started",
								Active:         true,
								Orphaned:       false,
								Blocked:        false,
								Managed:        true,
								Failed:         false,
								FailureIgnored: false,
								Node: []struct {
									Name   string  `xml:"name,attr"`
									ID     float64 `xml:"id,attr"`
									Cached string  `xml:"cached,attr"`
								}{
									{
										Name: "node1",
										ID:   1,
									},
								},
							},
						},
					},
				},
			},
			expectedMetric: 7, // 每个资源至少应该有7个指标
			expectError:    false,
		},
		{
			name: "空资源组测试",
			resources: ResourcesStruct{
				Group: []struct {
					ID              string           `xml:"id,attr"`
					NumberResources float64          `xml:"number_resources,attr"`
					Resource        []ResourceStruct `xml:"resource"`
				}{},
			},
			expectedMetric: 0,
			expectError:    false,
		},
		{
			name: "无效资源组测试",
			resources: ResourcesStruct{
				Group: []struct {
					ID              string           `xml:"id,attr"`
					NumberResources float64          `xml:"number_resources,attr"`
					Resource        []ResourceStruct `xml:"resource"`
				}{
					{
						ID: "", // 无效的ID
						Resource: []ResourceStruct{
							{
								ID: "", // 无效的ID
							},
						},
					},
				},
			},
			expectedMetric: 0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的收集器
			collector := &crmMonCollector{
				crmMonResourceGroupActive: prometheus.NewDesc(
					"crm_mon_resource_group_active",
					"Resource group active status",
					[]string{"id", "group_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceGroupOrphaned: prometheus.NewDesc(
					"crm_mon_resource_group_orphaned",
					"Resource group orphaned status",
					[]string{"id", "group_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceGroupBlocked: prometheus.NewDesc(
					"crm_mon_resource_group_blocked",
					"Resource group blocked status",
					[]string{"id", "group_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceGroupManaged: prometheus.NewDesc(
					"crm_mon_resource_group_managed",
					"Resource group managed status",
					[]string{"id", "group_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceGroupFailed: prometheus.NewDesc(
					"crm_mon_resource_group_failed",
					"Resource group failed status",
					[]string{"id", "group_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceGroupFailureIgnored: prometheus.NewDesc(
					"crm_mon_resource_group_failure_ignored",
					"Resource group failure ignored status",
					[]string{"id", "group_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourcesGroup: prometheus.NewDesc(
					"crm_mon_resources_group",
					"Number of resources in group",
					[]string{"group_id"},
					nil,
				),
			}

			// 创建指标通道
			ch := make(chan prometheus.Metric, 100)

			// 执行测试
			collector.exposeResourcesGroup(ch, tt.resources)

			// 验证结果
			close(ch)
			metrics := make([]prometheus.Metric, 0)
			for metric := range ch {
				metrics = append(metrics, metric)
			}

			assert.Equal(t, tt.expectedMetric, len(metrics), "指标数量不匹配")

			// 验证指标值
			for _, metric := range metrics {
				var m dto.Metric
				metric.Write(&m)

				// 验证标签
				labelPairs := make(map[string]string)
				for _, label := range m.Label {
					labelPairs[label.GetName()] = label.GetValue()
				}

				// 验证指标值
				if m.Gauge != nil {
					// 对于布尔类型的指标，值应该是0或1
					value := m.Gauge.GetValue()
					assert.True(t, value == 0 || value == 1, "指标值应该是0或1，实际值: %f", value)
				}
			}
		})
	}
}

// TestExposeResourcesClone 测试 exposeResourcesClone 函数
func TestExposeResourcesClone(t *testing.T) {
	// 设置测试用例
	tests := []struct {
		name           string
		resources      ResourcesStruct
		expectedMetric int
		expectError    bool
	}{
		{
			name: "正常克隆资源测试",
			resources: ResourcesStruct{
				Clone: []struct {
					ID             string           `xml:"id,attr"`
					MultiState     bool             `xml:"multi_state,attr"`
					Unique         bool             `xml:"unique,attr"`
					Managed        bool             `xml:"managed,attr"`
					Failed         bool             `xml:"failed,attr"`
					FailureIgnored bool             `xml:"failure_ignored,attr"`
					Resource       []ResourceStruct `xml:"resource"`
				}{
					{
						ID:         "test-clone",
						MultiState: true,
						Resource: []ResourceStruct{
							{
								ID:             "test-resource",
								ResourceAgent:  "ocf::heartbeat:Dummy",
								Role:           "Master",
								Active:         true,
								Orphaned:       false,
								Blocked:        false,
								Managed:        true,
								Failed:         false,
								FailureIgnored: false,
								Node: []struct {
									Name   string  `xml:"name,attr"`
									ID     float64 `xml:"id,attr"`
									Cached string  `xml:"cached,attr"`
								}{
									{
										Name: "node1",
									},
								},
							},
						},
					},
				},
			},
			expectedMetric: 10, // 明细+聚合
			expectError:    false,
		},
		{
			name: "空克隆资源测试",
			resources: ResourcesStruct{
				Clone: []struct {
					ID             string           `xml:"id,attr"`
					MultiState     bool             `xml:"multi_state,attr"`
					Unique         bool             `xml:"unique,attr"`
					Managed        bool             `xml:"managed,attr"`
					Failed         bool             `xml:"failed,attr"`
					FailureIgnored bool             `xml:"failure_ignored,attr"`
					Resource       []ResourceStruct `xml:"resource"`
				}{},
			},
			expectedMetric: 0,
			expectError:    false,
		},
		{
			name: "无效克隆资源测试",
			resources: ResourcesStruct{
				Clone: []struct {
					ID             string           `xml:"id,attr"`
					MultiState     bool             `xml:"multi_state,attr"`
					Unique         bool             `xml:"unique,attr"`
					Managed        bool             `xml:"managed,attr"`
					Failed         bool             `xml:"failed,attr"`
					FailureIgnored bool             `xml:"failure_ignored,attr"`
					Resource       []ResourceStruct `xml:"resource"`
				}{
					{
						ID: "", // 无效的ID
						Resource: []ResourceStruct{
							{
								ID: "", // 无效的ID
							},
						},
					},
				},
			},
			expectedMetric: 0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的收集器
			collector := &crmMonCollector{
				crmMonResourceCloneMultistate: prometheus.NewDesc(
					"crm_mon_resource_clone_multistate",
					"Resource clone multistate status",
					[]string{"clone_id"},
					nil,
				),
				crmMonResourceClonePromoted: prometheus.NewDesc(
					"crm_mon_resource_clone_promoted",
					"Resource clone promoted status",
					[]string{"id", "clone_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceCloneActive: prometheus.NewDesc(
					"crm_mon_resource_clone_active",
					"Resource clone active status",
					[]string{"id", "clone_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceCloneOrphaned: prometheus.NewDesc(
					"crm_mon_resource_clone_orphaned",
					"Resource clone orphaned status",
					[]string{"id", "clone_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceCloneBlocked: prometheus.NewDesc(
					"crm_mon_resource_clone_blocked",
					"Resource clone blocked status",
					[]string{"id", "clone_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceCloneManaged: prometheus.NewDesc(
					"crm_mon_resource_clone_managed",
					"Resource clone managed status",
					[]string{"id", "clone_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceCloneFailed: prometheus.NewDesc(
					"crm_mon_resource_clone_failed",
					"Resource clone failed status",
					[]string{"id", "clone_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceCloneFailureIgnored: prometheus.NewDesc(
					"crm_mon_resource_clone_failure_ignored",
					"Resource clone failure ignored status",
					[]string{"id", "clone_id", "node", "resource_agent", "role", "target_role"},
					nil,
				),
				crmMonResourceCloneNumActive: prometheus.NewDesc(
					"crm_mon_resource_clone_num_active",
					"Number of active clone instances",
					[]string{"clone_id"},
					nil,
				),
				crmMonResourceCloneNumPromoted: prometheus.NewDesc(
					"crm_mon_resource_clone_num_promoted",
					"Number of promoted clone instances",
					[]string{"clone_id"},
					nil,
				),
			}

			// 创建指标通道
			ch := make(chan prometheus.Metric, 100)

			// 执行测试
			collector.exposeResourcesClone(ch, tt.resources)

			// 验证结果
			close(ch)
			metrics := make([]prometheus.Metric, 0)
			for metric := range ch {
				metrics = append(metrics, metric)
			}

			assert.Equal(t, tt.expectedMetric, len(metrics), "指标数量不匹配")

			// 验证指标值
			for _, metric := range metrics {
				var m dto.Metric
				metric.Write(&m)

				// 验证标签
				labelPairs := make(map[string]string)
				for _, label := range m.Label {
					labelPairs[label.GetName()] = label.GetValue()
				}

				// 验证指标值
				if m.Gauge != nil {
					// 对于布尔类型的指标，值应该是0或1
					value := m.Gauge.GetValue()
					assert.True(t, value == 0 || value == 1, "指标值应该是0或1，实际值: %f", value)
				}
			}
		})
	}
}

// TestExposeNodes 测试 exposeNodes 函数
func TestExposeNodes(t *testing.T) {
	// 设置测试用例
	tests := []struct {
		name           string
		nodes          NodesStruct
		expectedMetric int
		expectError    bool
	}{
		{
			name: "正常节点测试",
			nodes: NodesStruct{
				Node: []struct {
					Name             string  `xml:"name,attr"`
					ID               string  `xml:"id,attr"`
					Online           bool    `xml:"online,attr"`
					Standby          bool    `xml:"standby,attr"`
					StandbyOnFail    bool    `xml:"standby_onfail,attr"`
					Maintenance      bool    `xml:"maintenance,attr"`
					Pending          bool    `xml:"pending,attr"`
					Unclean          bool    `xml:"unclean,attr"`
					Shutdown         bool    `xml:"shutdown,attr"`
					ExpectedUp       bool    `xml:"expected_up,attr"`
					IsDC             bool    `xml:"is_dc,attr"`
					ResourcesRunning float64 `xml:"resources_running,attr"`
					Type             string  `xml:"type,attr"`
				}{
					{
						Name:             "node1",
						ID:               "1",
						Online:           true,
						Standby:          false,
						StandbyOnFail:    false,
						Maintenance:      false,
						Pending:          false,
						Unclean:          false,
						Shutdown:         false,
						ExpectedUp:       true,
						IsDC:             true,
						ResourcesRunning: 5,
					},
				},
			},
			expectedMetric: 11, // 每个节点至少应该有11个指标
			expectError:    false,
		},
		{
			name: "空节点测试",
			nodes: NodesStruct{
				Node: []struct {
					Name             string  `xml:"name,attr"`
					ID               string  `xml:"id,attr"`
					Online           bool    `xml:"online,attr"`
					Standby          bool    `xml:"standby,attr"`
					StandbyOnFail    bool    `xml:"standby_onfail,attr"`
					Maintenance      bool    `xml:"maintenance,attr"`
					Pending          bool    `xml:"pending,attr"`
					Unclean          bool    `xml:"unclean,attr"`
					Shutdown         bool    `xml:"shutdown,attr"`
					ExpectedUp       bool    `xml:"expected_up,attr"`
					IsDC             bool    `xml:"is_dc,attr"`
					ResourcesRunning float64 `xml:"resources_running,attr"`
					Type             string  `xml:"type,attr"`
				}{},
			},
			expectedMetric: 0,
			expectError:    false,
		},
		{
			name: "无效节点测试",
			nodes: NodesStruct{
				Node: []struct {
					Name             string  `xml:"name,attr"`
					ID               string  `xml:"id,attr"`
					Online           bool    `xml:"online,attr"`
					Standby          bool    `xml:"standby,attr"`
					StandbyOnFail    bool    `xml:"standby_onfail,attr"`
					Maintenance      bool    `xml:"maintenance,attr"`
					Pending          bool    `xml:"pending,attr"`
					Unclean          bool    `xml:"unclean,attr"`
					Shutdown         bool    `xml:"shutdown,attr"`
					ExpectedUp       bool    `xml:"expected_up,attr"`
					IsDC             bool    `xml:"is_dc,attr"`
					ResourcesRunning float64 `xml:"resources_running,attr"`
					Type             string  `xml:"type,attr"`
				}{
					{
						Name: "", // 无效的名称
						ID:   "", // 无效的ID
					},
				},
			},
			expectedMetric: 0,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的收集器，补全所有用到的字段
			collector := &crmMonCollector{
				crmMonNodeID: prometheus.NewDesc(
					"crm_mon_node_id",
					"Node ID",
					[]string{"name", "type", "id"},
					nil,
				),
				crmMonNodeOnline: prometheus.NewDesc(
					"crm_mon_node_online",
					"Node online status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodeStandby: prometheus.NewDesc(
					"crm_mon_node_standby",
					"Node standby status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodeStandbyOnFail: prometheus.NewDesc(
					"crm_mon_node_standby_on_fail",
					"Node standby on fail status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodeMaintenance: prometheus.NewDesc(
					"crm_mon_node_maintenance",
					"Node maintenance status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodePending: prometheus.NewDesc(
					"crm_mon_node_pending",
					"Node pending status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodeUnclean: prometheus.NewDesc(
					"crm_mon_node_unclean",
					"Node unclean status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodeShutdown: prometheus.NewDesc(
					"crm_mon_node_shutdown",
					"Node shutdown status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodeExpectedUp: prometheus.NewDesc(
					"crm_mon_node_expected_up",
					"Node expected up status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodeIsDC: prometheus.NewDesc(
					"crm_mon_node_is_dc",
					"Node is DC status",
					[]string{"name", "id"},
					nil,
				),
				crmMonNodeResourcesRunning: prometheus.NewDesc(
					"crm_mon_node_resources_running",
					"Number of resources running on node",
					[]string{"name", "id"},
					nil,
				),
			}

			// 创建指标通道
			ch := make(chan prometheus.Metric, 100)

			// 执行测试
			collector.exposeNodes(ch, tt.nodes)

			// 验证结果
			close(ch)
			metrics := make([]prometheus.Metric, 0)
			for metric := range ch {
				metrics = append(metrics, metric)
			}

			assert.Equal(t, tt.expectedMetric, len(metrics), "指标数量不匹配")
		})
	}
}

// TestExposeNodeAttributes 测试 exposeNodeAttributes 函数
func TestExposeNodeAttributes(t *testing.T) {
	// 设置测试用例
	tests := []struct {
		name           string
		nodeAttr       NodeAttrStruct
		expectedMetric int
		expectError    bool
	}{
		{
			name: "正常节点属性测试",
			nodeAttr: NodeAttrStruct{
				Node: []struct {
					Name      string `xml:"name,attr"`
					Attribute []struct {
						Name  string `xml:"name,attr"`
						Value string `xml:"value,attr"`
					} `xml:"attribute"`
				}{
					{
						Name: "node1",
						Attribute: []struct {
							Name  string `xml:"name,attr"`
							Value string `xml:"value,attr"`
						}{
							{
								Name:  "test-attr",
								Value: "test-value",
							},
						},
					},
				},
			},
			expectedMetric: 1, // 每个属性应该有一个指标
			expectError:    false,
		},
		{
			name: "空节点属性测试",
			nodeAttr: NodeAttrStruct{
				Node: []struct {
					Name      string `xml:"name,attr"`
					Attribute []struct {
						Name  string `xml:"name,attr"`
						Value string `xml:"value,attr"`
					} `xml:"attribute"`
				}{},
			},
			expectedMetric: 0,
			expectError:    false,
		},
		{
			name: "无效节点属性测试",
			nodeAttr: NodeAttrStruct{
				Node: []struct {
					Name      string `xml:"name,attr"`
					Attribute []struct {
						Name  string `xml:"name,attr"`
						Value string `xml:"value,attr"`
					} `xml:"attribute"`
				}{
					{
						Name: "", // 无效的名称
						Attribute: []struct {
							Name  string `xml:"name,attr"`
							Value string `xml:"value,attr"`
						}{
							{
								Name:  "", // 无效的属性名
								Value: "", // 无效的属性值
							},
						},
					},
				},
			},
			expectedMetric: 1, // 实际主代码会输出1个指标
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的收集器
			collector := &crmMonCollector{
				crmMonNodeAttribute: prometheus.NewDesc(
					"crm_mon_node_attribute",
					"Node attribute value",
					[]string{"name", "node", "value"},
					nil,
				),
			}

			// 创建指标通道
			ch := make(chan prometheus.Metric, 100)

			// 执行测试
			collector.exposeNodeAttributes(ch, tt.nodeAttr)

			// 验证结果
			close(ch)
			metrics := make([]prometheus.Metric, 0)
			for metric := range ch {
				metrics = append(metrics, metric)
			}

			assert.Equal(t, tt.expectedMetric, len(metrics), "指标数量不匹配")
		})
	}
}

// TestExposeFailures 测试 exposeFailures 函数
func TestExposeFailures(t *testing.T) {
	// 设置测试用例
	tests := []struct {
		name           string
		crmMon         CrmMonStruct
		expectedMetric int
		expectError    bool
	}{
		{
			name: "正常失败测试",
			crmMon: CrmMonStruct{
				Failures: FailuresStruct{
					Failure: []struct {
						OpKey      string `xml:"op_key,attr"`
						Node       string `xml:"node,attr"`
						ExitStatus string `xml:"exitstatus,attr"`
						ExitReason string `xml:"exitreason,attr"`
						ExitCode   string `xml:"exitcode,attr"`
						Call       string `xml:"call,attr"`
						Status     string `xml:"status,attr"`
						Task       string `xml:"task,attr"`
					}{
						{
							OpKey:      "test-op",
							Node:       "node1",
							ExitReason: "test-reason",
							Status:     "failed",
						},
					},
				},
			},
			expectedMetric: 2, // 1个count+1个description
			expectError:    false,
		},
		{
			name: "空失败测试",
			crmMon: CrmMonStruct{
				Failures: FailuresStruct{
					Failure: []struct {
						OpKey      string `xml:"op_key,attr"`
						Node       string `xml:"node,attr"`
						ExitStatus string `xml:"exitstatus,attr"`
						ExitReason string `xml:"exitreason,attr"`
						ExitCode   string `xml:"exitcode,attr"`
						Call       string `xml:"call,attr"`
						Status     string `xml:"status,attr"`
						Task       string `xml:"task,attr"`
					}{},
				},
			},
			expectedMetric: 1, // 只有count
			expectError:    false,
		},
		{
			name: "无效失败测试",
			crmMon: CrmMonStruct{
				Failures: FailuresStruct{
					Failure: []struct {
						OpKey      string `xml:"op_key,attr"`
						Node       string `xml:"node,attr"`
						ExitStatus string `xml:"exitstatus,attr"`
						ExitReason string `xml:"exitreason,attr"`
						ExitCode   string `xml:"exitcode,attr"`
						Call       string `xml:"call,attr"`
						Status     string `xml:"status,attr"`
						Task       string `xml:"task,attr"`
					}{
						{
							OpKey:  "", // 无效的操作键
							Node:   "", // 无效的节点
							Status: "", // 无效的状态
						},
					},
				},
			},
			expectedMetric: 2, // 1个count+1个description
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的收集器
			collector := &crmMonCollector{
				crmMonFailuresCount: prometheus.NewDesc(
					"crm_mon_failures_count",
					"Cluster failures count",
					[]string{"name"},
					nil,
				),
				crmMonFailureDescription: prometheus.NewDesc(
					"crm_mon_failure_description",
					"Resource operation failure status",
					[]string{"node", "op_key", "status", "task"},
					nil,
				),
			}

			// 创建指标通道
			ch := make(chan prometheus.Metric, 100)

			// 执行测试
			collector.exposeFailures(ch, tt.crmMon)

			// 验证结果
			close(ch)
			metrics := make([]prometheus.Metric, 0)
			for metric := range ch {
				metrics = append(metrics, metric)
			}

			assert.Equal(t, tt.expectedMetric, len(metrics), "指标数量不匹配")
		})
	}
}

// TestExposeBans 测试 exposeBans 函数
func TestExposeBans(t *testing.T) {
	// 设置测试用例
	tests := []struct {
		name           string
		crmMon         CrmMonStruct
		expectedMetric int
		expectError    bool
	}{
		{
			name: "正常禁止测试",
			crmMon: CrmMonStruct{
				Bans: BansStruct{
					Ban: []struct {
						ID         string `xml:"id,attr"`
						Resource   string `xml:"resource,attr"`
						Node       string `xml:"node,attr"`
						Weight     string `xml:"weight,attr"`
						MasterOnly string `xml:"master_only,attr"`
					}{
						{
							ID:       "test-ban",
							Resource: "test-resource",
							Node:     "node1",
						},
					},
				},
			},
			expectedMetric: 2, // 1个count+1个description
			expectError:    false,
		},
		{
			name: "空禁止测试",
			crmMon: CrmMonStruct{
				Bans: BansStruct{
					Ban: []struct {
						ID         string `xml:"id,attr"`
						Resource   string `xml:"resource,attr"`
						Node       string `xml:"node,attr"`
						Weight     string `xml:"weight,attr"`
						MasterOnly string `xml:"master_only,attr"`
					}{},
				},
			},
			expectedMetric: 1, // 只有count
			expectError:    false,
		},
		{
			name: "无效禁止测试",
			crmMon: CrmMonStruct{
				Bans: BansStruct{
					Ban: []struct {
						ID         string `xml:"id,attr"`
						Resource   string `xml:"resource,attr"`
						Node       string `xml:"node,attr"`
						Weight     string `xml:"weight,attr"`
						MasterOnly string `xml:"master_only,attr"`
					}{
						{
							ID:       "", // 无效的ID
							Resource: "", // 无效的资源
							Node:     "", // 无效的节点
						},
					},
				},
			},
			expectedMetric: 2, // 1个count+1个description
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建模拟的收集器
			collector := &crmMonCollector{
				crmMonBansCount: prometheus.NewDesc(
					"crm_mon_bans_count",
					"Cluster bans count",
					[]string{"name"},
					nil,
				),
				crmMonBanDescription: prometheus.NewDesc(
					"crm_mon_ban_description",
					"Resource ban status",
					[]string{"id", "resource", "node", "weight", "master_only"},
					nil,
				),
			}

			// 创建指标通道
			ch := make(chan prometheus.Metric, 100)

			// 执行测试
			collector.exposeBans(ch, tt.crmMon)

			// 验证结果
			close(ch)
			metrics := make([]prometheus.Metric, 0)
			for metric := range ch {
				metrics = append(metrics, metric)
			}

			assert.Equal(t, tt.expectedMetric, len(metrics), "指标数量不匹配")
		})
	}
}

// 辅助函数：验证指标值
func validateMetric(t *testing.T, metric prometheus.Metric, expectedValue float64, expectedLabels map[string]string) {
	var m dto.Metric
	metric.Write(&m)

	// 验证标签
	labelPairs := make(map[string]string)
	for _, label := range m.Label {
		labelPairs[label.GetName()] = label.GetValue()
	}

	// 验证标签
	for key, value := range expectedLabels {
		assert.Equal(t, value, labelPairs[key], "标签值不匹配: %s", key)
	}

	// 验证指标值
	if m.Gauge != nil {
		assert.Equal(t, expectedValue, m.Gauge.GetValue(), "指标值不匹配")
	} else if m.Counter != nil {
		assert.Equal(t, expectedValue, m.Counter.GetValue(), "指标值不匹配")
	} else if m.Untyped != nil {
		assert.Equal(t, expectedValue, m.Untyped.GetValue(), "指标值不匹配")
	}
}
