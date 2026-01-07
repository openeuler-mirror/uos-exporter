# DHCP Leases Exporter

一个用于 ISC DHCP 服务器租约信息的 Prometheus 导出器。

## 功能特点

- 从 dhcpd.leases 文件收集指标
- 提供有关有效、过期和总租约的信息
- 暴露每个活跃租约的详细信息（主机名、IP、MAC）
- 通过命令行参数支持自定义配置
- 支持子网配置和统计信息
- 提供收集器性能指标
- 支持测试模式，方便开发和调试

## 指标

导出器提供以下指标：

### 基础统计指标

| 指标名称                    | 描述                                           | 标签 |
| --------------------------- | ---------------------------------------------- | ---- |
| dhcpd_leases_stats_valid    | dhcpd.leases 文件中有效租约的数量              | -    |
| dhcpd_leases_stats_expired  | dhcpd.leases 文件中过期租约的数量              | -    |
| dhcpd_leases_stats_count    | dhcpd.leases 文件中总租约数量                  | -    |
| dhcpd_leases_stats_filetime | dhcpd.leases 文件的最后修改时间（Unix 时间戳） | -    |

### 活跃租约指标

| 指标名称                   | 描述         | 标签              |
| -------------------------- | ------------ | ----------------- |
| dhcpd_leases_active_client | 活跃租约信息 | hostname, ip, mac |

### 收集器性能指标 - Stats

| 指标名称                                        | 描述                                         | 标签 |
| ----------------------------------------------- | -------------------------------------------- | ---- |
| dhcpd_leases_stats_scrapes_total                | Stats 收集器的总抓取次数                     | -    |
| dhcpd_leases_stats_scrape_errors_total          | Stats 收集器的总错误次数                     | -    |
| dhcpd_leases_stats_last_scrape_error            | 最后一次 Stats 抓取是否出错 (1=错误, 0=成功) | -    |
| dhcpd_leases_stats_last_scrape_timestamp        | 最后一次 Stats 抓取的时间戳                  | -    |
| dhcpd_leases_stats_last_scrape_duration_seconds | 最后一次 Stats 抓取的持续时间                | -    |

### 收集器性能指标 - Active

| 指标名称                                         | 描述                                          | 标签 |
| ------------------------------------------------ | --------------------------------------------- | ---- |
| dhcpd_leases_active_scrapes_total                | Active 收集器的总抓取次数                     | -    |
| dhcpd_leases_active_scrape_errors_total          | Active 收集器的总错误次数                     | -    |
| dhcpd_leases_active_last_scrape_error            | 最后一次 Active 抓取是否出错 (1=错误, 0=成功) | -    |
| dhcpd_leases_active_last_scrape_timestamp        | 最后一次 Active 抓取的时间戳                  | -    |
| dhcpd_leases_active_last_scrape_duration_seconds | 最后一次 Active 抓取的持续时间                | -    |

## 安装

### 从源码构建

```bash
# 克隆仓库
git clone https://github.com/yourusername/dhcpd_leases_exporter.git
cd dhcpd_leases_exporter

# 构建
make build
```

### 使用预编译二进制文件

从 [Releases](https://github.com/yourusername/dhcpd_leases_exporter/releases) 页面下载适合您系统的预编译二进制文件。

## 使用方法

```bash
./dhcpd_leases_exporter [flags]
```

### 命令行参数

| 参数                   | 描述                                                         | 默认值                       |
| ---------------------- | ------------------------------------------------------------ | ---------------------------- |
| `--web.listen-address` | 监听地址和端口                                               | `:8090`                      |
| `--web.telemetry-path` | 指标暴露路径                                                 | `/metrics`                   |
| `--dhcpd.leases-file`  | dhcpd.leases 文件路径                                        | `/var/lib/dhcp/dhcpd.leases` |
| `--dhcpd.subnets`      | DHCP 子网配置，格式：subnet1=start1-end1,subnet2=start2-end2 | -                            |
| `--test-mode`          | 创建测试租约文件（用于测试）                                 | `false`                      |
| `--debug`              | 启用调试模式                                                 | `false`                      |
| `--config`             | 配置文件路径                                                 | `/etc/exporter/exporter.yaml` |
| `--help`, `-h`         | 显示帮助信息                                                 | -                            |
| `--version`            | 显示版本信息                                                 | -                            |

### 配置文件

配置文件使用 YAML 格式，示例如下：

```yaml
log:
  level: debug
  logPath: /var/log/exporter.log
  maxSize: 10MB
  maxAge: 168h
address: 0.0.0.0
port: 8090
metricsPath: /metrics
```

### 示例

```bash
#测试前提：安装dhcp-server包，配置：/etc/dhcp/dhcpd.conf，启动服务。
systemctl status dhcpd

# 基本用法
./dhcpd_leases_exporter --web.listen-address=:8090 --dhcpd.leases-file=/var/lib/dhcpd/dhcpd.leases

# 使用配置文件
./dhcpd_leases_exporter --config=/path/to/config.yaml

# 启用测试模式
./dhcpd_leases_exporter --test-mode

# 配置子网
./dhcpd_leases_exporter --dhcpd.subnets="192.168.1.0/24=192.168.1.100-192.168.1.200,192.168.2.0/24=192.168.2.100-192.168.2.200"
```

## 开发

### 构建

```bash
make build
```

### 测试

```bash
make test
```

### Docker

构建 Docker 镜像：

```bash
docker build -t dhcpd-leases-exporter .
```

运行容器：

```bash
docker run -p 8090:8090 -v /var/lib/dhcpd/dhcpd.leases:/var/lib/dhcpd/dhcpd.leases:ro dhcpd-leases-exporter
```

## Prometheus 配置

在 `prometheus.yaml` 中添加以下内容：

```yaml
scrape_configs:
  - job_name: "dhcpd"
    static_configs:
      - targets: ["localhost:8090"]
```

## 监控面板

可以使用以下 Grafana 面板查询来创建 DHCP 租约监控面板：

1. DHCP 租约统计

```
# 有效租约数量
dhcpd_leases_stats_valid

# 过期租约数量
dhcpd_leases_stats_expired

# 总租约数量
dhcpd_leases_stats_count
```

2. 活跃租约表格

```
dhcpd_leases_active_client
```

## 故障排除

如果遇到问题，请尝试以下步骤：

1. 确保 dhcpd.leases 文件存在且可读
2. 使用 `--debug` 参数启用调试模式
3. 检查日志文件中的错误信息
4. 使用 `--test-mode` 参数创建测试租约文件进行测试

## License

MIT License
