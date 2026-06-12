package metrics

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewHealthReport(t *testing.T) {
	hr := NewHealthReport()
	assert.NotNil(t, hr)
	assert.Equal(t, "http://localhost:9200", hr.esURL)
	assert.False(t, hr.insecure)
	assert.NotNil(t, hr.jsonParseFailures)
	assert.NotNil(t, hr.totalRepositories)
	assert.NotNil(t, hr.maxShardsInClusterData)
	assert.NotNil(t, hr.maxShardsInClusterFrozen)
	assert.NotNil(t, hr.restartingReplicas)
	assert.NotNil(t, hr.creatingPrimaries)
	assert.NotNil(t, hr.initializingReplicas)
	assert.NotNil(t, hr.unassignedReplicas)
	assert.NotNil(t, hr.startedPrimaries)
	assert.NotNil(t, hr.restartingPrimaries)
	assert.NotNil(t, hr.initializingPrimaries)
	assert.NotNil(t, hr.creatingReplicas)
	assert.NotNil(t, hr.startedReplicas)
	assert.NotNil(t, hr.unassignedPrimaries)
	assert.NotNil(t, hr.slmPolicies)
	assert.NotNil(t, hr.ilmPolicies)
	assert.NotNil(t, hr.ilmStagnatingIndices)
	assert.NotNil(t, hr.status)
	assert.NotNil(t, hr.masterIsStableStatus)
	assert.NotNil(t, hr.repositoryIntegrityStatus)
	assert.NotNil(t, hr.diskStatus)
	assert.NotNil(t, hr.shardsCapacityStatus)
	assert.NotNil(t, hr.shardsAvailabilityStatus)
	assert.NotNil(t, hr.dataStreamLifecycleStatus)
	assert.NotNil(t, hr.slmStatus)
	assert.NotNil(t, hr.ilmStatus)
}

func TestHealthReportFetchAndDecodeHealthReport(t *testing.T) {
	// 创建模拟的健康报告响应
	mockHealthReport := HealthReportResponse{
		ClusterName: "test-cluster",
		Status:      "green",
		Indicators: HealthReportIndicators{
			MasterIsStable: HealthReportMasterIsStable{
				Status:  "green",
				Symptom: "The cluster has a stable master node",
				Details: HealthReportMasterIsStableDetails{
					CurrentMaster: HealthReportMasterIsStableDetailsNode{
						NodeID: "node1",
						Name:   "node-1",
					},
					RecentMasters: []HealthReportMasterIsStableDetailsNode{
						{
							NodeID: "node1",
							Name:   "node-1",
						},
					},
				},
			},
			RepositoryIntegrity: HealthReportRepositoryIntegrity{
				Status:  "green",
				Symptom: "All repositories are verified to be operational.",
				Details: HealthReportRepositoriyIntegrityDetails{
					TotalRepositories: 2,
				},
			},
			Disk: HealthReportDisk{
				Status:  "green",
				Symptom: "All nodes have sufficient disk space.",
				Details: HealthReportDiskDetails{
					IndicesWithReadonlyBlock:     0,
					NodesWithEnoughDiskSpace:     3,
					NodesWithUnknownDiskStatus:   0,
					NodesOverHighWatermark:       0,
					NodesOverFloodStageWatermark: 0,
				},
			},
			ShardsCapacity: HealthReportShardsCapacity{
				Status:  "green",
				Symptom: "The cluster has enough capacity for all primary and replica shards.",
				Details: HealthReportShardsCapacityDetails{
					Data: HealthReportShardsCapacityDetailsMaxShards{
						MaxShardsInCluster: 1000,
					},
					Frozen: HealthReportShardsCapacityDetailsMaxShards{
						MaxShardsInCluster: 500,
					},
				},
			},
			ShardsAvailability: HealthReportShardsAvailability{
				Status:  "green",
				Symptom: "All shards are available.",
				Details: HealthReportShardsAvailabilityDetails{
					RestartingReplicas:    0,
					CreatingPrimaries:     0,
					InitializingReplicas:  0,
					UnassignedReplicas:    0,
					StartedPrimaries:      10,
					RestartingPrimaries:   0,
					InitializingPrimaries: 0,
					CreatingReplicas:      0,
					StartedReplicas:       10,
					UnassignedPrimaries:   0,
				},
			},
			DataStreamLifecycle: HealthReportDataStreamLifecycle{
				Status:  "green",
				Symptom: "No issues found.",
			},
			Slm: HealthReportSlm{
				Status:  "green",
				Symptom: "Snapshot lifecycle management is running.",
				Details: HealthReportSlmDetails{
					SlmStatus: "RUNNING",
					Policies:  3,
				},
			},
			Ilm: HealthReportIlm{
				Status:  "green",
				Symptom: "Index lifecycle management is running.",
				Details: HealthReportIlmDetails{
					Policies:          5,
					StagnatingIndices: 0,
					IlmStatus:         "RUNNING",
				},
			},
		},
	}

	// 创建模拟服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_health_report") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(mockHealthReport)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not found")
		}
	}))
	defer ts.Close()

	// 测试成功的响应
	hr := NewHealthReport()
	hr.esURL = ts.URL
	resp, err := hr.fetchAndDecodeHealthReport()
	assert.NoError(t, err)
	assert.Equal(t, "test-cluster", resp.ClusterName)
	assert.Equal(t, "green", resp.Status)
	assert.Equal(t, 2, resp.Indicators.RepositoryIntegrity.Details.TotalRepositories)
	assert.Equal(t, 1000, resp.Indicators.ShardsCapacity.Details.Data.MaxShardsInCluster)
	assert.Equal(t, 500, resp.Indicators.ShardsCapacity.Details.Frozen.MaxShardsInCluster)
	assert.Equal(t, 10, resp.Indicators.ShardsAvailability.Details.StartedPrimaries)
	assert.Equal(t, 3, resp.Indicators.Slm.Details.Policies)
	assert.Equal(t, 5, resp.Indicators.Ilm.Details.Policies)

	// 测试服务器错误
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error")
	}))
	defer ts2.Close()

	hr.esURL = ts2.URL
	_, err = hr.fetchAndDecodeHealthReport()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP Request failed with code 500")

	// 测试无效的JSON响应
	ts3 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "invalid json data")
	}))
	defer ts3.Close()

	hr.esURL = ts3.URL
	_, err = hr.fetchAndDecodeHealthReport()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid")
}

func TestHealthReportCollect(t *testing.T) {
	// 创建模拟的健康报告响应
	mockHealthReport := HealthReportResponse{
		ClusterName: "test-cluster",
		Status:      "green",
		Indicators: HealthReportIndicators{
			MasterIsStable: HealthReportMasterIsStable{
				Status:  "green",
				Symptom: "The cluster has a stable master node",
				Details: HealthReportMasterIsStableDetails{
					CurrentMaster: HealthReportMasterIsStableDetailsNode{
						NodeID: "node1",
						Name:   "node-1",
					},
					RecentMasters: []HealthReportMasterIsStableDetailsNode{
						{
							NodeID: "node1",
							Name:   "node-1",
						},
					},
				},
			},
			RepositoryIntegrity: HealthReportRepositoryIntegrity{
				Status:  "green",
				Symptom: "All repositories are verified to be operational.",
				Details: HealthReportRepositoriyIntegrityDetails{
					TotalRepositories: 2,
				},
			},
			Disk: HealthReportDisk{
				Status:  "green",
				Symptom: "All nodes have sufficient disk space.",
				Details: HealthReportDiskDetails{
					IndicesWithReadonlyBlock:     0,
					NodesWithEnoughDiskSpace:     3,
					NodesWithUnknownDiskStatus:   0,
					NodesOverHighWatermark:       0,
					NodesOverFloodStageWatermark: 0,
				},
			},
			ShardsCapacity: HealthReportShardsCapacity{
				Status:  "green",
				Symptom: "The cluster has enough capacity for all primary and replica shards.",
				Details: HealthReportShardsCapacityDetails{
					Data: HealthReportShardsCapacityDetailsMaxShards{
						MaxShardsInCluster: 1000,
					},
					Frozen: HealthReportShardsCapacityDetailsMaxShards{
						MaxShardsInCluster: 500,
					},
				},
			},
			ShardsAvailability: HealthReportShardsAvailability{
				Status:  "green",
				Symptom: "All shards are available.",
				Details: HealthReportShardsAvailabilityDetails{
					RestartingReplicas:    0,
					CreatingPrimaries:     0,
					InitializingReplicas:  0,
					UnassignedReplicas:    0,
					StartedPrimaries:      10,
					RestartingPrimaries:   0,
					InitializingPrimaries: 0,
					CreatingReplicas:      0,
					StartedReplicas:       10,
					UnassignedPrimaries:   0,
				},
			},
			DataStreamLifecycle: HealthReportDataStreamLifecycle{
				Status:  "green",
				Symptom: "No issues found.",
			},
			Slm: HealthReportSlm{
				Status:  "green",
				Symptom: "Snapshot lifecycle management is running.",
				Details: HealthReportSlmDetails{
					SlmStatus: "RUNNING",
					Policies:  3,
				},
			},
			Ilm: HealthReportIlm{
				Status:  "green",
				Symptom: "Index lifecycle management is running.",
				Details: HealthReportIlmDetails{
					Policies:          5,
					StagnatingIndices: 0,
					IlmStatus:         "RUNNING",
				},
			},
		},
	}

	// 创建模拟服务器
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/_health_report") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			data, _ := json.Marshal(mockHealthReport)
			w.Write(data)
		} else {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "Not found")
		}
	}))
	defer ts.Close()

	// 设置日志级别
	logrus.SetLevel(logrus.DebugLevel)

	// 测试 Collect 方法
	hr := NewHealthReport()
	hr.esURL = ts.URL

	// 创建注册表
	registry := prometheus.NewRegistry()
	registry.MustRegister(hr)

	// 验证指标
	expected := `
# HELP elasticsearch_health_report_creating_primaries The number of creating primary shards
# TYPE elasticsearch_health_report_creating_primaries gauge
elasticsearch_health_report_creating_primaries{cluster="test-cluster"} 0
# HELP elasticsearch_health_report_creating_replicas The number of creating replica shards
# TYPE elasticsearch_health_report_creating_replicas gauge
elasticsearch_health_report_creating_replicas{cluster="test-cluster"} 0
# HELP elasticsearch_health_report_data_stream_lifecycle_status Data stream lifecycle status
# TYPE elasticsearch_health_report_data_stream_lifecycle_status gauge
elasticsearch_health_report_data_stream_lifecycle_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_data_stream_lifecycle_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_data_stream_lifecycle_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_disk_status Disk status
# TYPE elasticsearch_health_report_disk_status gauge
elasticsearch_health_report_disk_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_disk_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_disk_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_ilm_policies The number of ILM Policies
# TYPE elasticsearch_health_report_ilm_policies gauge
elasticsearch_health_report_ilm_policies{cluster="test-cluster"} 5
# HELP elasticsearch_health_report_ilm_stagnating_indices The number of stagnating indices
# TYPE elasticsearch_health_report_ilm_stagnating_indices gauge
elasticsearch_health_report_ilm_stagnating_indices{cluster="test-cluster"} 0
# HELP elasticsearch_health_report_ilm_status ILM status
# TYPE elasticsearch_health_report_ilm_status gauge
elasticsearch_health_report_ilm_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_ilm_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_ilm_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_initializing_primaries The number of initializing primary shards
# TYPE elasticsearch_health_report_initializing_primaries gauge
elasticsearch_health_report_initializing_primaries{cluster="test-cluster"} 0
# HELP elasticsearch_health_report_initializing_replicas The number of initializing replica shards
# TYPE elasticsearch_health_report_initializing_replicas gauge
elasticsearch_health_report_initializing_replicas{cluster="test-cluster"} 0
# HELP elasticsearch_health_report_json_parse_failures Number of errors while parsing JSON.
# TYPE elasticsearch_health_report_json_parse_failures counter
elasticsearch_health_report_json_parse_failures 0
# HELP elasticsearch_health_report_master_is_stable_status Master is stable status
# TYPE elasticsearch_health_report_master_is_stable_status gauge
elasticsearch_health_report_master_is_stable_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_master_is_stable_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_master_is_stable_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_max_shards_in_cluster_data The maximum shards in data cluster
# TYPE elasticsearch_health_report_max_shards_in_cluster_data gauge
elasticsearch_health_report_max_shards_in_cluster_data{cluster="test-cluster"} 1000
# HELP elasticsearch_health_report_max_shards_in_cluster_frozen The maximum shards in frozen cluster
# TYPE elasticsearch_health_report_max_shards_in_cluster_frozen gauge
elasticsearch_health_report_max_shards_in_cluster_frozen{cluster="test-cluster"} 500
# HELP elasticsearch_health_report_repository_integrity_status Repository integrity status
# TYPE elasticsearch_health_report_repository_integrity_status gauge
elasticsearch_health_report_repository_integrity_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_repository_integrity_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_repository_integrity_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_restarting_primaries The number of restarting primary shards
# TYPE elasticsearch_health_report_restarting_primaries gauge
elasticsearch_health_report_restarting_primaries{cluster="test-cluster"} 0
# HELP elasticsearch_health_report_restarting_replicas The number of restarting replica shards
# TYPE elasticsearch_health_report_restarting_replicas gauge
elasticsearch_health_report_restarting_replicas{cluster="test-cluster"} 0
# HELP elasticsearch_health_report_shards_availability_status Shards availability status
# TYPE elasticsearch_health_report_shards_availability_status gauge
elasticsearch_health_report_shards_availability_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_shards_availability_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_shards_availability_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_shards_capacity_status Shards capacity status
# TYPE elasticsearch_health_report_shards_capacity_status gauge
elasticsearch_health_report_shards_capacity_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_shards_capacity_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_shards_capacity_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_slm_policies The number of SLM policies
# TYPE elasticsearch_health_report_slm_policies gauge
elasticsearch_health_report_slm_policies{cluster="test-cluster"} 3
# HELP elasticsearch_health_report_slm_status SLM status
# TYPE elasticsearch_health_report_slm_status gauge
elasticsearch_health_report_slm_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_slm_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_slm_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_started_primaries The number of started primary shards
# TYPE elasticsearch_health_report_started_primaries gauge
elasticsearch_health_report_started_primaries{cluster="test-cluster"} 10
# HELP elasticsearch_health_report_started_replicas The number of started replica shards
# TYPE elasticsearch_health_report_started_replicas gauge
elasticsearch_health_report_started_replicas{cluster="test-cluster"} 10
# HELP elasticsearch_health_report_status Overall cluster status
# TYPE elasticsearch_health_report_status gauge
elasticsearch_health_report_status{cluster="test-cluster",color="green"} 1
elasticsearch_health_report_status{cluster="test-cluster",color="red"} 0
elasticsearch_health_report_status{cluster="test-cluster",color="yellow"} 0
# HELP elasticsearch_health_report_total_repositories The number of total repositories
# TYPE elasticsearch_health_report_total_repositories gauge
elasticsearch_health_report_total_repositories{cluster="test-cluster"} 2
# HELP elasticsearch_health_report_unassigned_primaries The number of unassigned primary shards
# TYPE elasticsearch_health_report_unassigned_primaries gauge
elasticsearch_health_report_unassigned_primaries{cluster="test-cluster"} 0
# HELP elasticsearch_health_report_unassigned_replicas The number of unassigned replica shards
# TYPE elasticsearch_health_report_unassigned_replicas gauge
elasticsearch_health_report_unassigned_replicas{cluster="test-cluster"} 0
`

	err := testutil.GatherAndCompare(registry, strings.NewReader(expected))
	assert.NoError(t, err)

	// 测试服务器错误时的 Collect
	ts2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Server Error")
	}))
	defer ts2.Close()

	hr2 := NewHealthReport()
	hr2.esURL = ts2.URL

	// 捕获日志输出
	var logOutput strings.Builder
	logrus.SetOutput(io.MultiWriter(&logOutput))
	defer func() {
		logrus.SetOutput(io.Discard)
	}()

	registry2 := prometheus.NewRegistry()
	registry2.MustRegister(hr2)

	// 测试日志输出
	_, err = testutil.GatherAndCount(registry2)
	assert.NoError(t, err)
	assert.Contains(t, logOutput.String(), "Failed to fetch and decode health report")
}

func TestStatusValue(t *testing.T) {
	// 测试绿色状态
	assert.Equal(t, 1.0, statusValue("green", "green"))
	assert.Equal(t, 0.0, statusValue("green", "yellow"))
	assert.Equal(t, 0.0, statusValue("green", "red"))

	// 测试黄色状态
	assert.Equal(t, 0.0, statusValue("yellow", "green"))
	assert.Equal(t, 1.0, statusValue("yellow", "yellow"))
	assert.Equal(t, 0.0, statusValue("yellow", "red"))

	// 测试红色状态
	assert.Equal(t, 0.0, statusValue("red", "green"))
	assert.Equal(t, 0.0, statusValue("red", "yellow"))
	assert.Equal(t, 1.0, statusValue("red", "red"))

	// 测试未知状态
	assert.Equal(t, 0.0, statusValue("unknown", "green"))
	assert.Equal(t, 0.0, statusValue("unknown", "yellow"))
	assert.Equal(t, 0.0, statusValue("unknown", "red"))
} 