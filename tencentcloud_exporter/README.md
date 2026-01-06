# Tencentcloud Exporter

腾讯云监控指标导出器，用于 Prometheus 监控系统收集腾讯云各项产品的监控指标。

## 功能特性

- 支持腾讯云 CDB 云数据库 MySQL 实例监控指标收集
- 使用腾讯云监控 API 自动发现并收集指标
- 支持通过配置文件自定义收集参数
- 支持速率限制，避免 API 调用过于频繁

## 安装与使用

### 编译

```bash
make build
```

### 配置

在`config/export.yaml`中配置：

```yaml
address: "0.0.0.0" # 监听地址
port: 9112 # 监听端口
metricsPath: "/metrics" # 指标路径
log:
  level: "info" # 日志级别
  log_path: "./tencentcloud_exporter.log" # 日志文件路径
credential:
  access_key: "您的腾讯云API密钥ID"
  secret_key: "您的腾讯云API密钥"
  region: "ap-guangzhou" # 腾讯云地区
  role: "" # 可选：用于跨账号访问的角色名称
rate_limit: 15 # API调用速率限制

# 启用CDB产品监控
products:
  - namespace: QCE/CDB
    all_metrics: true # 收集所有指标
    all_instances: true # 收集所有实例
    # 可选：添加额外标签
    extra_labels:
      environment: "production"
```

**注意**：`role`字段是新增的配置项，用于支持跨账号访问腾讯云资源的场景。如果您不需要跨账号访问，可以保持为空字符串或者完全省略此字段。旧项目不需要此字段。

### 运行

```bash
./tencentcloud_exporter --config=./config/export.yaml
```

### Prometheus 配置

在 Prometheus 配置中添加如下内容：

```yaml
scrape_configs:
  - job_name: "tencentcloud_cdb"
    static_configs:
      - targets: ["localhost:9112"]
    scrape_interval: 60s
```

## 支持的指标

当前支持腾讯云 CDB 云数据库 MySQL 产品的以下指标：

- 实例状态
- CPU 使用率
- 内存使用率
- 磁盘使用率
- 连接数
- 查询数
- QPS 和 TPS
- 慢查询

## 开发

如果需要添加更多腾讯云产品的监控指标，请参考`internal/metrics/cdb.go`的实现方式。

## 许可证

MIT
