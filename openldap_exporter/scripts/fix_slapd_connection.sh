#!/bin/bash

# OpenLDAP 连接问题故障排除脚本

echo "=== OpenLDAP 连接问题诊断和修复 ==="

# 1. 检查服务状态
echo "1. 检查 slapd 服务状态..."
if systemctl is-active --quiet slapd; then
    echo "✓ slapd 服务正在运行"
    sudo systemctl status slapd --no-pager
else
    echo "✗ slapd 服务未运行"
    echo "尝试启动 slapd 服务..."
    sudo systemctl start slapd
    sleep 3
    if systemctl is-active --quiet slapd; then
        echo "✓ slapd 服务启动成功"
    else
        echo "✗ slapd 服务启动失败"
        echo "查看启动错误："
        sudo systemctl status slapd --no-pager
        sudo journalctl -u slapd --no-pager -n 20
    fi
fi

# 2. 检查进程
echo ""
echo "2. 检查 slapd 进程..."
if pgrep -x slapd > /dev/null; then
    echo "✓ slapd 进程存在："
    ps aux | grep slapd | grep -v grep
else
    echo "✗ slapd 进程不存在"
fi

# 3. 检查端口监听
echo ""
echo "3. 检查端口监听..."
if sudo ss -tlnp | grep -q :389; then
    echo "✓ 端口 389 正在监听："
    sudo ss -tlnp | grep :389
else
    echo "✗ 端口 389 未监听"
fi

# 4. 检查Unix socket
echo ""
echo "4. 检查 Unix socket..."
if [ -S /var/run/slapd/ldapi ]; then
    echo "✓ Unix socket 存在: /var/run/slapd/ldapi"
else
    echo "✗ Unix socket 不存在: /var/run/slapd/ldapi"
    echo "检查 /var/run/slapd/ 目录："
    ls -la /var/run/slapd/ 2>/dev/null || echo "目录不存在"
fi

# 5. 检查配置目录权限
echo ""
echo "5. 检查配置目录权限..."
if [ -d /etc/openldap/slapd.d ]; then
    echo "配置目录权限："
    ls -la /etc/openldap/slapd.d/
    echo ""
    echo "配置目录所有者："
    stat -c "%U:%G" /etc/openldap/slapd.d
    
    # 检查是否需要修复权限
    OWNER=$(stat -c "%U" /etc/openldap/slapd.d)
    if [ "$OWNER" != "ldap" ]; then
        echo "⚠️  配置目录所有者不是 ldap，需要修复"
        echo "执行: sudo chown -R ldap:ldap /etc/openldap/slapd.d"
    fi
else
    echo "✗ 配置目录不存在: /etc/openldap/slapd.d"
fi

# 6. 检查数据目录
echo ""
echo "6. 检查数据目录..."
if [ -d /var/lib/ldap ]; then
    echo "数据目录权限："
    ls -la /var/lib/ldap/
    echo ""
    echo "数据目录所有者："
    stat -c "%U:%G" /var/lib/ldap
    
    # 检查是否需要修复权限
    OWNER=$(stat -c "%U" /var/lib/ldap)
    if [ "$OWNER" != "ldap" ]; then
        echo "⚠️  数据目录所有者不是 ldap，需要修复"
        echo "执行: sudo chown -R ldap:ldap /var/lib/ldap"
    fi
else
    echo "✗ 数据目录不存在: /var/lib/ldap"
fi

# 7. 查看最近的日志
echo ""
echo "7. 查看最近的系统日志..."
echo "slapd 相关日志："
sudo journalctl -u slapd --no-pager -n 10

# 8. 测试基本连接
echo ""
echo "8. 测试基本连接..."

echo "8.1 测试 TCP 连接..."
if timeout 5 bash -c "</dev/tcp/127.0.0.1/389" 2>/dev/null; then
    echo "✓ TCP 连接到 127.0.0.1:389 成功"
else
    echo "✗ TCP 连接到 127.0.0.1:389 失败"
fi

echo "8.2 测试 ldapi 连接..."
if ldapsearch -Y EXTERNAL -H ldapi:/// -b "" -s base >/dev/null 2>&1; then
    echo "✓ ldapi 连接成功"
else
    echo "✗ ldapi 连接失败"
fi

echo "8.3 测试 LDAP 连接..."
if ldapsearch -x -H ldap://127.0.0.1:389 -b "" -s base >/dev/null 2>&1; then
    echo "✓ LDAP 连接成功"
else
    echo "✗ LDAP 连接失败"
fi

# 9. 提供修复建议
echo ""
echo "=== 修复建议 ==="

# 检查是否需要重新初始化
if ! systemctl is-active --quiet slapd || ! pgrep -x slapd > /dev/null; then
    echo "🔧 步骤1: 重新启动 slapd 服务"
    echo "sudo systemctl stop slapd"
    echo "sudo systemctl start slapd"
    echo "sudo systemctl status slapd"
fi

# 检查权限问题
if [ -d /etc/openldap/slapd.d ] && [ "$(stat -c "%U" /etc/openldap/slapd.d)" != "ldap" ]; then
    echo ""
    echo "🔧 步骤2: 修复配置目录权限"
    echo "sudo chown -R ldap:ldap /etc/openldap/slapd.d"
fi

if [ -d /var/lib/ldap ] && [ "$(stat -c "%U" /var/lib/ldap)" != "ldap" ]; then
    echo ""
    echo "🔧 步骤3: 修复数据目录权限"
    echo "sudo chown -R ldap:ldap /var/lib/ldap"
fi

# 提供重新安装选项
echo ""
echo "🔧 步骤4: 如果上述步骤无法解决问题，尝试重新安装："
echo "sudo systemctl stop slapd"
echo "sudo dnf remove -y openldap-servers"
echo "sudo rm -rf /etc/openldap/slapd.d/*"
echo "sudo rm -rf /var/lib/ldap/*"
echo "sudo dnf install -y openldap-servers"
echo "sudo systemctl start slapd"

# 提供手动配置选项
echo ""
echo "🔧 步骤5: 手动配置 (如果自动脚本失败)："
echo "1. 确保服务运行: sudo systemctl status slapd"
echo "2. 测试连接: ldapsearch -Y EXTERNAL -H ldapi:/// -b '' -s base"
echo "3. 手动执行配置文件"

echo ""
echo "=== 诊断完成 ==="
echo ""
echo "如果问题仍然存在，请："
echo "1. 查看完整日志: sudo journalctl -u slapd -f"
echo "2. 检查 SELinux: sudo getenforce"
echo "3. 运行此脚本获取详细信息: bash scripts/fix_slapd_connection.sh" 