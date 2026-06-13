# OpenLDAP Exporter

OpenLDAP Prometheus Exporter - 从OpenLDAP服务器收集监控指标。

## 快速开始

### 1. 在Fedora上安装OpenLDAP

我们提供了完整的自动化安装脚本：

```bash
# 进入脚本目录
cd scripts

# 运行自动安装脚本（需要sudo权限）
bash install_openldap_fedora.sh
```

脚本将自动：
- 安装OpenLDAP服务器和客户端
- 配置基本的LDAP服务（域名：dc=example,dc=com）
- 启用cn=Monitor监控功能
- 创建管理员用户（cn=admin,dc=example,dc=com，密码：admin123）
- 配置防火墙规则
- 验证安装和配置

### 2. 手动安装指南

如果你希望手动安装，请参考详细的安装指南：
- [Fedora OpenLDAP 安装和配置指南](docs/fedora_openldap_setup.md)

### 3. 测试LDAP连接和Monitor查询

安装完成后，使用测试脚本验证配置：

```bash
# 测试LDAP连接和cn=Monitor查询
cd scripts
bash test_monitor.sh
```

### 4. 启动OpenLDAP Exporter

```bash
# 使用自动生成的配置文件
./openldap_exporter --config config/exporter_ldap_ready.yaml

# 或使用默认配置文件
./openldap_exporter --config config/exporter.yaml
```

## 配置

### 配置文件

exporter支持通过YAML配置文件进行配置。默认配置文件路径为 `/etc/uos-exporter/openldap-exporter.yaml`。

#### 配置示例

```yaml
address: "0.0.0.0"
port: 9006
metricsPath: "/metrics"
log:
  level: "info"
  log_path: "/var/log/uos-exporter/openldap-exporter.log"

# OpenLDAP 客户端配置
ldap:
  host: "127.0.0.1"          # LDAP服务器地址
  port: "389"                 # LDAP服务器端口
  bind_dn: "cn=admin,dc=example,dc=com"  # 绑定DN（留空使用匿名绑定）
  bind_password: "admin123"   # 绑定密码
```

#### 配置参数说明

##### 服务器配置
- `address`: HTTP服务器绑定地址
- `port`: HTTP服务器端口
- `metricsPath`: metrics端点路径

##### 日志配置
- `log.level`: 日志级别 (debug, info, warn, error)
- `log.log_path`: 日志文件路径

##### LDAP配置
- `ldap.host`: OpenLDAP服务器主机地址
- `ldap.port`: OpenLDAP服务器端口 (默认: 389)
- `ldap.bind_dn`: 用于连接的LDAP绑定DN（留空使用匿名绑定）
- `ldap.bind_password`: 绑定密码

### 使用方法

```bash
# 使用默认配置文件
./openldap_exporter

# 指定配置文件
./openldap_exporter --config /path/to/config.yaml

# 查看帮助
./openldap_exporter --help
```

## 诊断工具

### LDAP连接诊断

我们提供了专门的诊断工具来排查LDAP连接问题：

```bash
cd tools
go run ldap_diag.go [config_file]
```

诊断工具将测试：
- LDAP服务器连接
- 匿名绑定
- 认证绑定
- cn=Monitor查询
- 尝试不同的绑定DN

### Monitor查询测试

快速测试cn=Monitor查询功能：

```bash
cd scripts
bash test_monitor.sh

# 或指定自定义参数
LDAP_HOST=192.168.1.100 LDAP_PORT=389 bash test_monitor.sh
```

## 故障排除

### 常见错误和解决方案

#### 1. LDAP Result Code 49 "Invalid Credentials"

**问题**: 绑定凭据无效

**解决方案**:
1. 检查bind_dn和bind_password是否正确
2. 尝试使用匿名绑定（将bind_dn设为空字符串）
3. 运行诊断工具：`cd tools && go run ldap_diag.go`

#### 2. LDAP Result Code 32 "No Such Object"

**问题**: cn=Monitor不存在或无访问权限

**解决方案**:
1. 确保OpenLDAP启用了Monitor模块
2. 检查Monitor配置是否正确
3. 验证用户是否有Monitor访问权限
4. 参考安装指南重新配置Monitor

#### 3. 连接超时或连接被拒绝

**问题**: 无法连接到LDAP服务器

**解决方案**:
1. 检查LDAP服务是否运行：`sudo systemctl status slapd`
2. 检查端口是否监听：`sudo ss -tlnp | grep :389`
3. 检查防火墙设置：`sudo firewall-cmd --list-services`
4. 验证主机和端口配置

## 指标

exporter收集以下OpenLDAP监控指标：

### 连接指标
- `openldap_exporter_connections_current`: 当前LDAP连接数
- `openldap_exporter_connections_total`: 总LDAP连接数
- `openldap_exporter_connections_idle`: 空闲LDAP连接数
- `openldap_exporter_connections_max`: 最大允许LDAP连接数

### 操作指标
- `openldap_exporter_operations_initiated_total`: 发起的LDAP操作总数
- `openldap_exporter_operations_completed_total`: 完成的LDAP操作总数
- `openldap_exporter_operations_bind_total`: LDAP绑定操作总数
- `openldap_exporter_operations_search_total`: LDAP搜索操作总数
- 等等...

### 其他指标
- `openldap_exporter_thread_count`: 活跃线程数
- `openldap_exporter_database_entries`: 数据库条目数
- `openldap_exporter_cache_size`: 缓存大小

## 系统要求

- Go 1.19+
- 对OpenLDAP服务器的cn=Monitor的读取权限
- OpenLDAP 2.4+ （推荐）

## 部署

### 开发环境

1. 编译程序:
```bash
go build -o openldap_exporter
```

2. 安装OpenLDAP (Fedora):
```bash
cd scripts
bash install_openldap_fedora.sh
```

3. 启动服务:
```bash
./openldap_exporter --config config/exporter_ldap_ready.yaml
```

### 生产环境

1. 创建配置文件:
```bash
sudo mkdir -p /etc/uos-exporter
sudo cp config/exporter.yaml /etc/uos-exporter/openldap-exporter.yaml
```

2. 使用systemd服务:
```bash
sudo cp uos-openldap-exporter.service /etc/systemd/system/
sudo systemctl enable uos-openldap-exporter
sudo systemctl start uos-openldap-exporter
```

## 文档

- [Fedora OpenLDAP 安装指南](docs/fedora_openldap_setup.md) - 详细的手动安装步骤
- [脚本说明](scripts/) - 自动化安装和测试脚本
- [诊断工具](tools/) - LDAP连接诊断工具

## 支持的操作系统

- ✅ Fedora (已测试)
- ✅ CentOS/RHEL (兼容)
- ✅ Ubuntu/Debian (需要适配包管理器命令)
- ✅ 其他Linux发行版 (手动安装)

## 贡献

欢迎提交Issue和Pull Request来改进这个项目！