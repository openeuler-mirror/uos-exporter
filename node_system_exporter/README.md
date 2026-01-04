# Node System Exporter

Node System Exporter 是一个专门用于监控系统基础指标的 Prometheus exporter，包括 CPU、内存、系统负载和系统信息等核心系统指标。

## 功能特性

- **CPU 监控**: CPU 使用率、频率、核心状态、热节流等
- **CPU 频率监控**: CPU 频率信息、调频调节器状态
- **CPU 漏洞监控**: CPU 安全漏洞信息和缓解措施
- **内存监控**: 内存使用情况、缓存、交换空间等详细信息
- **内存分配器监控**: Buddy 分配器状态
- **KSM 守护进程监控**: 内核同页合并功能状态
- **NUMA 内存监控**: NUMA 节点内存分布和统计
- **虚拟内存监控**: 页面错误、交换等虚拟内存统计
- **系统负载**: 1 分钟、5 分钟、15 分钟负载平均值
- **系统统计**: 中断、上下文切换、进程创建、软中断等
- **系统信息**: 系统名称、版本、架构等基本信息

## 监控指标

### CPU 指标

- `node_cpu_seconds_total`: CPU 在各种模式下的时间
- `node_cpu_guest_seconds_total`: CPU 在虚拟机中的时间
- `node_cpu_core_throttles_total`: CPU 核心节流次数
- `node_cpu_package_throttles_total`: CPU 包节流次数
- `node_cpu_isolated`: CPU 核心隔离状态
- `node_cpu_online`: CPU 核心在线状态

### CPU 频率指标

- `node_cpu_frequency_hertz`: 当前 CPU 频率
- `node_cpu_frequency_min_hertz`: 最小 CPU 频率
- `node_cpu_frequency_max_hertz`: 最大 CPU 频率
- `node_cpu_scaling_frequency_hertz`: 当前调频频率
- `node_cpu_scaling_frequency_min_hertz`: 最小调频频率
- `node_cpu_scaling_frequency_max_hertz`: 最大调频频率
- `node_cpu_scaling_governor`: CPU 调频调节器状态

### CPU 漏洞指标

- `node_cpu_vulnerabilities_info`: CPU 安全漏洞详细信息

### 内存指标

- `node_memory_*_bytes`: 各种内存使用情况（Active、Cached、Buffers 等）
- `node_memory_HugePages_*`: 大页内存信息

### 内存分配器指标

- `node_buddyinfo_blocks`: 按大小分类的空闲内存块数量

### KSM 守护进程指标

- `node_ksmd_full_scans_total`: KSM 全扫描次数
- `node_ksmd_merge_across_nodes`: 跨节点合并状态
- `node_ksmd_pages_shared`: 共享页面数
- `node_ksmd_pages_sharing`: 共享中的页面数
- `node_ksmd_pages_to_scan`: 待扫描页面数
- `node_ksmd_pages_unshared`: 未共享页面数
- `node_ksmd_pages_volatile`: 易失性页面数
- `node_ksmd_run`: KSM 运行状态
- `node_ksmd_sleep_seconds`: KSM 睡眠时间

### NUMA 内存指标

- `node_memory_numa_*`: NUMA 节点内存统计信息

### 虚拟内存指标

- `node_vmstat_*`: 虚拟内存统计（页面错误、交换操作等）

### 系统负载指标

- `node_load1`: 1 分钟负载平均值
- `node_load5`: 5 分钟负载平均值
- `node_load15`: 15 分钟负载平均值

### 系统统计指标

- `node_intr_total`: 中断总数
- `node_context_switches_total`: 上下文切换总数
- `node_forks_total`: 进程创建总数
- `node_boot_time_seconds`: 系统启动时间
- `node_procs_running`: 运行中的进程数
- `node_procs_blocked`: 阻塞的进程数
- `node_softirqs_total`: 软中断统计

### 系统信息指标

- `node_uname_info`: 系统信息（系统名、版本、架构等）

## 配置

默认配置文件位置：`/etc/uos-exporter/export.yaml`

```yaml
address: "0.0.0.0"
port: 9120
metricsPath: "/metrics"
log:
  level: "info"
  log_path: "/var/log/uos-exporter/node_system_exporter.log"
```

## 构建和运行

```bash
# 构建
make build

# 运行
./node_system_exporter

# 或指定配置文件
./node_system_exporter -c /path/to/config.yaml
```

## 访问指标

启动后访问 `http://localhost:9120/metrics` 查看所有监控指标。
