# Node Storage Exporter

Node Storage Exporter 是一个专门用于监控存储相关指标的 Prometheus exporter，包括磁盘 I/O、文件系统、存储设备等核心存储指标。

## 功能特性

- **磁盘 I/O 监控**: 磁盘读写统计、I/O 时间、队列深度等
- **文件系统监控**: 文件系统使用情况、挂载点状态、可用空间等
- **XFS 文件系统监控**: XFS 专用统计信息和性能指标
- **BTRFS 文件系统监控**: BTRFS 设备和分配统计信息

## 监控指标

### 磁盘 I/O 指标

- `node_disk_reads_completed_total`: 成功完成的读操作总数
- `node_disk_reads_merged_total`: 合并的读操作总数
- `node_disk_read_bytes_total`: 成功读取的字节总数
- `node_disk_read_time_seconds_total`: 所有读操作花费的总时间
- `node_disk_writes_completed_total`: 成功完成的写操作总数
- `node_disk_writes_merged_total`: 合并的写操作总数
- `node_disk_written_bytes_total`: 成功写入的字节总数
- `node_disk_write_time_seconds_total`: 所有写操作花费的总时间
- `node_disk_io_now`: 当前正在进行的 I/O 操作数
- `node_disk_io_time_seconds_total`: 执行 I/O 操作的总时间
- `node_disk_io_time_weighted_seconds_total`: 加权 I/O 操作时间

### 文件系统指标

- `node_filesystem_size_bytes`: 文件系统总大小（字节）
- `node_filesystem_free_bytes`: 文件系统可用空间（字节）
- `node_filesystem_avail_bytes`: 非 root 用户可用空间（字节）
- `node_filesystem_files`: 文件系统总文件节点数
- `node_filesystem_files_free`: 文件系统可用文件节点数
- `node_filesystem_readonly`: 文件系统只读状态
- `node_filesystem_device_error`: 获取设备统计信息时是否发生错误

### XFS 文件系统指标

- `node_xfs_extent_allocation_*`: XFS 扩展分配统计
- `node_xfs_allocation_btree_*`: XFS 分配 B-tree 统计
- `node_xfs_block_mapping_*`: XFS 块映射统计
- `node_xfs_block_map_btree_*`: XFS 块映射 B-tree 统计
- `node_xfs_directory_operation_*`: XFS 目录操作统计
- `node_xfs_inode_operation_*`: XFS inode 操作统计

### BTRFS 文件系统指标

- `node_btrfs_device_size_bytes`: BTRFS 设备大小
- `node_btrfs_allocation_total_bytes`: BTRFS 总分配空间
- `node_btrfs_allocation_used_bytes`: BTRFS 已使用分配空间

## 配置

默认配置文件位置：`/etc/uos-exporter/node-storage-exporter.yaml`

```yaml
address: "0.0.0.0"
port: 9121
metricsPath: "/metrics"
log:
  level: "info"
  log_path: "/var/log/uos-exporter/node_storage_exporter.log"
```

## 构建和运行

```bash
# 构建
make build

# 运行
./node_storage_exporter

# 或指定配置文件
./node_storage_exporter -c /path/to/config.yaml
```

## 访问指标

启动后访问 `http://localhost:9121/metrics` 查看所有监控指标。

## 支持的存储类型

- **通用磁盘设备**: 支持所有块设备的 I/O 统计
- **文件系统**: 支持所有挂载的文件系统监控
- **XFS**: 专门的 XFS 文件系统性能监控
- **BTRFS**: BTRFS 文件系统设备和分配监控

## 过滤规则

### 忽略的设备

默认忽略以下设备模式：

- RAM 磁盘: `ram*`, `zram*`
- 循环设备: `loop*`
- 软盘设备: `fd*`
- 分区设备: `*d[a-z][0-9]+`, `nvme*p[0-9]+`

### 忽略的挂载点

默认忽略以下挂载点：

- 系统目录: `/dev`, `/proc`, `/sys`
- 容器目录: `/var/lib/docker/*`, `/var/lib/containers/storage/*`

### 忽略的文件系统类型

默认忽略以下文件系统类型：

- 虚拟文件系统: `proc`, `sysfs`, `tmpfs`
- 特殊文件系统: `devpts`, `cgroup*`, `overlay`
