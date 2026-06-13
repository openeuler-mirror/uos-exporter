# Fedora OpenLDAP 安装和配置指南

## 1. 安装 OpenLDAP 服务器和客户端

```bash
# 安装 OpenLDAP 服务器、客户端和工具
sudo dnf install -y openldap-servers openldap-clients openldap

# 安装额外的工具
sudo dnf install -y openldap-devel
```

## 2. 启动和启用 OpenLDAP 服务

```bash
# 启动 slapd 服务
sudo systemctl start slapd

# 设置开机自启
sudo systemctl enable slapd

# 检查服务状态
sudo systemctl status slapd
```

## 3. 配置基本的 LDAP 设置

### 3.1 设置管理员密码

```bash
# 生成密码哈希
sudo slappasswd
# 输入密码，例如：admin123
# 会得到类似这样的哈希：{SSHA}xxxxxxxxxx
```

### 3.2 创建基础配置文件

创建 `/tmp/db.ldif` 文件：

```ldif
dn: olcDatabase=mdb,cn=config
changetype: modify
replace: olcSuffix
olcSuffix: dc=example,dc=com

dn: olcDatabase=mdb,cn=config
changetype: modify
replace: olcRootDN
olcRootDN: cn=admin,dc=example,dc=com

dn: olcDatabase=mdb,cn=config
changetype: modify
replace: olcRootPW
olcRootPW: {SSHA}替换为上面生成的密码哈希
```

### 3.3 创建监控配置文件

创建 `/tmp/monitor.ldif` 文件：

```ldif
dn: cn=module,cn=config
objectClass: olcModuleList
cn: module
olcModulepath: /usr/lib64/openldap
olcModuleload: back_monitor.la

dn: olcDatabase=monitor,cn=config
objectClass: olcDatabaseConfig
objectClass: olcMonitorConfig
olcDatabase: monitor
olcSuffix: cn=Monitor
olcRootDN: cn=monitor,cn=Monitor
olcMonitorContext: cn=Monitor
olcAccess: to dn.subtree="cn=Monitor" 
  by dn="cn=admin,dc=example,dc=com" read 
  by * none
```

### 3.4 应用配置

```bash
# 应用基础配置
sudo ldapmodify -Y EXTERNAL -H ldapi:/// -f /tmp/db.ldif

# 应用监控配置
sudo ldapmodify -Y EXTERNAL -H ldapi:/// -f /tmp/monitor.ldif
```

## 4. 创建基础目录结构

创建 `/tmp/base.ldif` 文件：

```ldif
dn: dc=example,dc=com
objectClass: top
objectClass: dcObject
objectClass: organization
o: Example Organization
dc: example

dn: cn=admin,dc=example,dc=com
objectClass: organizationalRole
cn: admin
description: LDAP administrator
```

应用基础结构：

```bash
# 使用管理员凭据添加基础结构
ldapadd -x -D "cn=admin,dc=example,dc=com" -W -f /tmp/base.ldif
# 输入之前设置的管理员密码
```

## 5. 配置监控访问权限

创建 `/tmp/monitor_access.ldif` 文件：

```ldif
dn: olcDatabase={1}monitor,cn=config
changetype: modify
replace: olcAccess
olcAccess: {0}to * by dn.base="gidNumber=0+uidNumber=0,cn=peercred,cn=external,cn=auth" read by dn.base="cn=admin,dc=example,dc=com" read by * none
```

应用访问权限：

```bash
sudo ldapmodify -Y EXTERNAL -H ldapi:/// -f /tmp/monitor_access.ldif
```

## 6. 验证安装和配置

### 6.1 检查服务状态

```bash
# 检查 LDAP 服务端口
sudo netstat -tlnp | grep :389

# 或者使用 ss 命令
sudo ss -tlnp | grep :389
```

### 6.2 测试基本连接

```bash
# 测试匿名连接
ldapsearch -x -H ldap://localhost -b "" -s base

# 测试管理员认证
ldapsearch -x -H ldap://localhost -D "cn=admin,dc=example,dc=com" -W -b "dc=example,dc=com"
```

### 6.3 测试监控查询

```bash
# 查询监控信息（匿名）
ldapsearch -x -H ldap://localhost -b "cn=Monitor" -s base

# 查询监控信息（认证）
ldapsearch -x -H ldap://localhost -D "cn=admin,dc=example,dc=com" -W -b "cn=Monitor" -s base

# 查询具体监控指标
ldapsearch -x -H ldap://localhost -D "cn=admin,dc=example,dc=com" -W -b "cn=Monitor" "(objectClass=*)"
```

## 7. 防火墙配置

如果启用了防火墙，需要打开 LDAP 端口：

```bash
# 允许 LDAP 端口
sudo firewall-cmd --permanent --add-service=ldap
sudo firewall-cmd --reload

# 或者直接打开端口
sudo firewall-cmd --permanent --add-port=389/tcp
sudo firewall-cmd --reload
```

## 8. 故障排除

### 8.1 查看日志

```bash
# 查看 systemd 日志
sudo journalctl -u slapd -f

# 查看系统日志
sudo tail -f /var/log/messages
```

### 8.2 配置调试模式

编辑 `/etc/sysconfig/slapd`：

```bash
# 添加调试选项
SLAPD_OPTIONS="-d 256"
```

重启服务：

```bash
sudo systemctl restart slapd
```

### 8.3 常见问题

1. **权限错误**：确保 `/var/lib/ldap` 目录的所有者是 `ldap`
   ```bash
   sudo chown -R ldap:ldap /var/lib/ldap
   ```

2. **配置错误**：检查配置语法
   ```bash
   sudo slaptest -f /etc/openldap/slapd.conf
   ```

3. **端口冲突**：确保端口 389 没有被其他服务占用
   ```bash
   sudo lsof -i :389
   ```

## 9. 配置 openldap_exporter

更新 openldap_exporter 配置文件：

```yaml
address: "0.0.0.0"
port: 9006
metricsPath: "/metrics"
log:
  level: "info"
  log_path: "/var/log/uos-exporter/openldap-exporter.log"

ldap:
  host: "127.0.0.1"
  port: "389"                    # 标准 LDAP 端口
  bind_dn: "cn=admin,dc=example,dc=com"  # 或留空使用匿名绑定
  bind_password: "admin123"      # 管理员密码
```

## 10. 测试 exporter

```bash
# 使用诊断工具测试连接
cd openldap_exporter/tools
go run ldap_diag.go

# 启动 exporter
cd ../
./openldap_exporter --config config/exporter.yaml
```

## 附录：完整的监控配置 LDIF

如果需要更完整的监控配置，可以使用以下 LDIF：

```ldif
dn: olcDatabase={2}monitor,cn=config
objectClass: olcDatabaseConfig
objectClass: olcMonitorConfig
olcDatabase: {2}monitor
olcSuffix: cn=Monitor
olcAddContentAcl: FALSE
olcLastMod: TRUE
olcMaxDerefDepth: 15
olcReadOnly: FALSE
olcRootDN: cn=admin,dc=example,dc=com
olcMonitorContext: cn=Monitor
olcAccess: {0}to dn.subtree="cn=Monitor" by dn.base="cn=admin,dc=example,dc=com" read by * none
```

这个配置将允许通过 cn=Monitor 查询 OpenLDAP 的运行时统计信息，包括连接数、操作统计、缓存信息等，这些正是 openldap_exporter 需要的监控指标。 