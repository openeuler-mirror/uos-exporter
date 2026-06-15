#!/bin/bash

# 测试 cn=Monitor 查询脚本

echo "=== OpenLDAP Monitor 查询测试 ==="

# 设置默认参数
LDAP_HOST="${LDAP_HOST:-127.0.0.1}"
LDAP_PORT="${LDAP_PORT:-389}"
BIND_DN="${BIND_DN:-cn=admin,dc=example,dc=com}"
BIND_PASS="${BIND_PASS:-admin123}"

echo "连接信息："
echo "  主机: $LDAP_HOST"
echo "  端口: $LDAP_PORT"
echo "  绑定DN: $BIND_DN"
echo ""

# 1. 测试基本连接
echo "1. 测试基本连接..."
if ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -b "" -s base >/dev/null 2>&1; then
    echo "✓ 基本连接成功"
else
    echo "✗ 基本连接失败"
    exit 1
fi

# 2. 测试匿名绑定
echo "2. 测试匿名绑定..."
if ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -b "" -s base "(objectClass=*)" >/dev/null 2>&1; then
    echo "✓ 匿名绑定成功"
    ANONYMOUS_OK=true
else
    echo "✗ 匿名绑定失败"
    ANONYMOUS_OK=false
fi

# 3. 测试认证绑定
echo "3. 测试认证绑定..."
if ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -D "$BIND_DN" -w "$BIND_PASS" -b "dc=example,dc=com" -s base >/dev/null 2>&1; then
    echo "✓ 认证绑定成功"
    AUTH_OK=true
else
    echo "✗ 认证绑定失败"
    AUTH_OK=false
fi

# 4. 测试Monitor查询
echo "4. 测试 cn=Monitor 查询..."

echo "4.1 尝试匿名查询 cn=Monitor..."
if ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -b "cn=Monitor" -s base "(objectClass=*)" >/dev/null 2>&1; then
    echo "✓ 匿名Monitor查询成功"
    MONITOR_ANONYMOUS=true
else
    echo "✗ 匿名Monitor查询失败"
    MONITOR_ANONYMOUS=false
fi

echo "4.2 尝试认证查询 cn=Monitor..."
if [ "$AUTH_OK" = true ]; then
    if ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -D "$BIND_DN" -w "$BIND_PASS" -b "cn=Monitor" -s base "(objectClass=*)" >/dev/null 2>&1; then
        echo "✓ 认证Monitor查询成功"
        MONITOR_AUTH=true
    else
        echo "✗ 认证Monitor查询失败"
        MONITOR_AUTH=false
    fi
else
    echo "- 跳过认证Monitor查询（认证失败）"
    MONITOR_AUTH=false
fi

# 5. 详细Monitor查询
echo ""
echo "5. 详细Monitor信息查询..."

if [ "$MONITOR_AUTH" = true ]; then
    echo "使用认证查询Monitor属性："
    ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -D "$BIND_DN" -w "$BIND_PASS" -b "cn=Monitor" -s base | head -20
elif [ "$MONITOR_ANONYMOUS" = true ]; then
    echo "使用匿名查询Monitor属性："
    ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -b "cn=Monitor" -s base | head -20
else
    echo "无法查询Monitor信息"
fi

# 6. 查询Monitor子条目
echo ""
echo "6. 查询Monitor子条目..."

if [ "$MONITOR_AUTH" = true ]; then
    echo "查询Monitor子条目（认证）："
    ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -D "$BIND_DN" -w "$BIND_PASS" -b "cn=Monitor" -s one | grep "^dn:" | head -10
elif [ "$MONITOR_ANONYMOUS" = true ]; then
    echo "查询Monitor子条目（匿名）："
    ldapsearch -x -H "ldap://$LDAP_HOST:$LDAP_PORT" -b "cn=Monitor" -s one | grep "^dn:" | head -10
fi

# 7. 总结和建议
echo ""
echo "=== 测试总结 ==="
echo "基本连接: $([ $? -eq 0 ] && echo "✓" || echo "✗")"
echo "匿名绑定: $([ "$ANONYMOUS_OK" = true ] && echo "✓" || echo "✗")"
echo "认证绑定: $([ "$AUTH_OK" = true ] && echo "✓" || echo "✗")"
echo "匿名Monitor: $([ "$MONITOR_ANONYMOUS" = true ] && echo "✓" || echo "✗")"
echo "认证Monitor: $([ "$MONITOR_AUTH" = true ] && echo "✓" || echo "✗")"

echo ""
echo "=== openldap_exporter 配置建议 ==="

if [ "$MONITOR_AUTH" = true ]; then
    echo "推荐配置（使用认证）："
    echo "ldap:"
    echo "  host: \"$LDAP_HOST\""
    echo "  port: \"$LDAP_PORT\""
    echo "  bind_dn: \"$BIND_DN\""
    echo "  bind_password: \"$BIND_PASS\""
elif [ "$MONITOR_ANONYMOUS" = true ]; then
    echo "推荐配置（使用匿名绑定）："
    echo "ldap:"
    echo "  host: \"$LDAP_HOST\""
    echo "  port: \"$LDAP_PORT\""
    echo "  bind_dn: \"\""
    echo "  bind_password: \"\""
else
    echo "❌ Monitor查询不可用，请检查："
    echo "1. OpenLDAP是否启用了Monitor模块"
    echo "2. 是否正确配置了cn=Monitor"
    echo "3. 用户是否有Monitor访问权限"
fi

echo ""
echo "手动测试命令："
if [ "$MONITOR_AUTH" = true ]; then
    echo "ldapsearch -x -H \"ldap://$LDAP_HOST:$LDAP_PORT\" -D \"$BIND_DN\" -w \"$BIND_PASS\" -b \"cn=Monitor\" -s base"
elif [ "$MONITOR_ANONYMOUS" = true ]; then
    echo "ldapsearch -x -H \"ldap://$LDAP_HOST:$LDAP_PORT\" -b \"cn=Monitor\" -s base"
fi 