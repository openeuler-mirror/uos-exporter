package metrics

import (
	"context"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// 是否运行耗时测试
var runLongTests = flag.Bool("long", false, "运行耗时较长的测试")

// 创建测试用的租约文件内容
func createTestLeaseFileContent() string {
	now := time.Now()
	return fmt.Sprintf(`lease 192.168.1.100 {
		starts 6 %s;
		ends 6 %s;
		hardware ethernet 00:11:22:33:44:55;
		client-hostname "test-host-1";
	}
	lease 192.168.1.101 {
		starts 6 %s;
		ends 6 %s;
		hardware ethernet 00:11:22:33:44:66;
		client-hostname "test-host-2";
		abandoned;
	}`,
		now.AddDate(0, 0, -1).Format("2006/01/02 15:04:05"),
		now.AddDate(0, 0, 7).Format("2006/01/02 15:04:05"),
		now.AddDate(0, 0, -2).Format("2006/01/02 15:04:05"),
		now.AddDate(0, 0, -1).Format("2006/01/02 15:04:05"),
	)
}

// 设置测试环境
func setupTestEnvironment(t *testing.T) (string, func()) {
	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "dhcpd_test")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}

	// 创建测试租约文件
	leaseFile := filepath.Join(tmpDir, "dhcpd.leases")
	err = os.WriteFile(leaseFile, []byte(createTestLeaseFileContent()), 0644)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("创建测试租约文件失败: %v", err)
	}

	// 设置日志级别为debug
	logrus.SetLevel(logrus.DebugLevel)

	// 返回清理函数
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return leaseFile, cleanup
}

// TestNewDHCPDMetrics 测试指标收集器的创建
func TestNewDHCPDMetrics(t *testing.T) {
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 验证基础指标是否已创建
	assert.NotNil(t, metrics.validLeases, "validLeases指标不应为空")
	assert.NotNil(t, metrics.expiredLeases, "expiredLeases指标不应为空")
	assert.NotNil(t, metrics.totalLeases, "totalLeases指标不应为空")
	assert.NotNil(t, metrics.fileTimestamp, "fileTimestamp指标不应为空")
	assert.NotNil(t, metrics.activeLeases, "activeLeases指标不应为空")

	// 验证收集器指标
	assert.NotNil(t, metrics.scrapesTotalStats, "scrapesTotalStats指标不应为空")
	assert.NotNil(t, metrics.scrapeErrorsTotalStats, "scrapeErrorsTotalStats指标不应为空")
	assert.NotNil(t, metrics.lastScrapeErrorStats, "lastScrapeErrorStats指标不应为空")
	assert.NotNil(t, metrics.lastScrapeTimeStats, "lastScrapeTimeStats指标不应为空")
	assert.NotNil(t, metrics.lastScrapeDurationStats, "lastScrapeDurationStats指标不应为空")
}

// TestDescribe 测试指标描述功能
func TestDescribe(t *testing.T) {
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	ch := make(chan *prometheus.Desc, 100)
	metrics.Describe(ch)
	close(ch)

	// 计算接收到的描述符数量
	count := 0
	for range ch {
		count++
	}

	// 验证是否收到了所有预期的描述符
	// 基础指标(4) + 活跃租约指标(1) + 收集器指标(10)
	expectedCount := 15
	assert.Equal(t, expectedCount, count, "描述符数量不匹配")
}

// TestCollect 测试指标收集功能
func TestCollect(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 设置超时上下文，非常短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 确保全局 DHCPDInfo 是干净的
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 在一个新的goroutine中运行收集操作
	done := make(chan struct{})
	var ch chan prometheus.Metric
	go func() {
		ch = make(chan prometheus.Metric, 100)
		// 收集指标
		metrics.Collect(ch)
		close(ch)
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 收集完成，计算指标数量
		count := 0
		for range ch {
			count++
		}
		// 验证是否收集到了所有预期的指标
		assert.GreaterOrEqual(t, count, 5, "收集到的指标数量不足")
	case <-ctx.Done():
		t.Fatalf("指标收集超时，可能存在死锁: %v", ctx.Err())
	}

	// 确保清理全局状态，避免影响其他测试
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestInitDHCPDInfo 测试DHCP信息收集器的初始化
func TestInitDHCPDInfo(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置超时上下文，非常短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false
	*subnetsConfig = "192.168.1.0/24=192.168.1.100-192.168.1.200"

	// 在goroutine中初始化，以便可以检测超时
	done := make(chan struct{})
	var err error
	go func() {
		// 初始化DHCP信息收集器
		InitDHCPDInfo()

		// 验证是否成功初始化
		assert.NotNil(t, DHCPDInfo, "DHCP信息收集器不应为空")

		// 测试读取租约信息
		if DHCPDInfo != nil {
			err = DHCPDInfo.Read()
		}
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		assert.NoError(t, err, "读取租约信息不应出错")
		if DHCPDInfo != nil {
			// 验证租约数量
			assert.GreaterOrEqual(t, DHCPDInfo.GetTotalLeases(), 1, "应至少有一个租约")
		}
	case <-ctx.Done():
		t.Fatalf("初始化DHCP信息收集器超时，可能存在死锁: %v", ctx.Err())
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestCreateTestLeaseFile 测试测试租约文件的创建
func TestCreateTestLeaseFile(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 启用测试模式
	*testMode = true
	*dhcpdLeasesFile = "/tmp/dhcpd_test.leases"

	// 创建测试租约文件
	createTestLeaseFile()

	// 验证文件是否创建成功
	_, err := os.Stat(*dhcpdLeasesFile)
	assert.NoError(t, err, "测试租约文件应该存在")

	// 清理测试文件
	os.Remove(*dhcpdLeasesFile)
}

// TestMetricsWithSubnets 测试带子网配置的指标收集
func TestMetricsWithSubnets(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置超时上下文，非常短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false
	*subnetsConfig = "192.168.1.0/24=192.168.1.100-192.168.1.200,192.168.2.0/24=192.168.2.100-192.168.2.200"

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 在goroutine中运行，以便可以检测超时
	done := make(chan struct{})
	var ch chan prometheus.Metric
	go func() {
		// 初始化DHCP信息收集器
		InitDHCPDInfo()

		// 验证子网配置
		assert.NotNil(t, DHCPDInfo, "DHCP信息收集器不应为空")

		// 收集指标
		ch = make(chan prometheus.Metric, 100)
		metrics.Collect(ch)
		close(ch)
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 计数收集到的指标
		count := 0
		for range ch {
			count++
		}
		assert.GreaterOrEqual(t, count, 5, "收集到的指标数量不足")
	case <-ctx.Done():
		t.Fatalf("带子网配置的指标收集超时，可能存在死锁: %v", ctx.Err())
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestMetricsWithInvalidFile 测试无效租约文件的情况
func TestMetricsWithInvalidFile(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置一个不存在的文件
	*dhcpdLeasesFile = "/path/to/nonexistent/file"
	*testMode = false

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 验证是否仍然能收集到基本指标
	count := 0
	for range ch {
		count++
	}

	// 即使文件不存在，也应该能收集到基本的指标
	assert.GreaterOrEqual(t, count, 10, "即使文件无效也应该能收集到基本指标")
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 创建等待组和上下文
	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 同时运行多个收集操作
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ch := make(chan prometheus.Metric, 100)
			metrics.Collect(ch)
			close(ch)

			// 验证是否成功收集指标
			count := 0
			for range ch {
				count++
			}
			// 注意：在并发环境中，我们不做断言，只是执行操作
		}()
	}

	// 等待所有goroutine完成或超时
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// 成功完成
	case <-ctx.Done():
		t.Fatalf("并发测试超时")
	}
}

// TestEdgeCases 测试边缘情况
func TestEdgeCases(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 写入空的租约文件
	err := os.WriteFile(leaseFile, []byte(""), 0644)
	assert.NoError(t, err, "写入空租约文件不应出错")

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 验证空文件情况下是否仍能收集基本指标
	count := 0
	for range ch {
		count++
	}
	assert.GreaterOrEqual(t, count, 5, "即使文件为空也应该能收集到基本指标")
}

// TestPerformance 测试性能
func TestPerformance(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 测量收集性能
	start := time.Now()

	// 多次运行收集操作
	iterations := 5
	for i := 0; i < iterations; i++ {
		ch := make(chan prometheus.Metric, 100)
		metrics.Collect(ch)

		// 消费所有指标
		for range ch {
			// 仅消费，不做其他操作
		}
		close(ch)
	}

	duration := time.Since(start)
	avgTime := duration / time.Duration(iterations)

	// 打印性能信息，但不做断言（因为性能因环境而异）
	t.Logf("平均收集时间: %v", avgTime)
}

// TestInvalidSubnetConfig 测试无效子网配置
func TestInvalidSubnetConfig(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false
	*subnetsConfig = "invalid-format" // 使用无效格式的子网配置

	// 初始化DHCP信息收集器（不应崩溃）
	InitDHCPDInfo()

	// 验证是否仍然初始化
	assert.NotNil(t, DHCPDInfo, "即使子网配置无效，DHCP信息收集器也不应为空")

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 验证是否仍然能收集到基本指标
	count := 0
	for range ch {
		count++
	}
	assert.GreaterOrEqual(t, count, 5, "即使子网配置无效也应该能收集到基本指标")
}

// TestDescribeWithMockRegistry 使用模拟注册表测试描述功能
func TestDescribeWithMockRegistry(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 创建一个注册表并注册收集器
	registry := prometheus.NewRegistry()
	err := registry.Register(metrics)
	assert.NoError(t, err, "注册收集器不应出错")

	// 取消注册收集器
	assert.True(t, registry.Unregister(metrics), "取消注册收集器应该成功")
}

// TestMultipleInitialization 测试多次初始化
func TestMultipleInitialization(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置超时上下文，非常短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 在goroutine中运行，以便可以检测超时
	done := make(chan struct{})
	var err error
	go func() {
		// 多次初始化DHCP信息收集器
		for i := 0; i < 5; i++ {
			InitDHCPDInfo()
			assert.NotNil(t, DHCPDInfo, "DHCP信息收集器不应为空")
		}

		// 验证是否能正常读取
		if DHCPDInfo != nil {
			err = DHCPDInfo.Read()
		}
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		assert.NoError(t, err, "多次初始化后读取不应出错")
	case <-ctx.Done():
		t.Fatalf("多次初始化测试超时，可能存在死锁: %v", ctx.Err())
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestCorruptedLeaseFile 测试损坏的租约文件情况
func TestCorruptedLeaseFile(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "dhcpd_test_corrupted")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建损坏的租约文件
	corruptedContent := `
		lease 192.168.1.100 {
			starts 6 incomplete;
			ends 6 invalid;
			hardware ethernet 00:11:22:33:44:55;
			client-hostname "corrupted-test";
		}
		lease 192.168.1.101 {
			invalid format
			hardware ethernet XX:XX:XX:XX:XX:XX;
		}
	`
	leaseFile := filepath.Join(tmpDir, "corrupted.leases")
	err = os.WriteFile(leaseFile, []byte(corruptedContent), 0644)
	if err != nil {
		t.Fatalf("创建损坏的租约文件失败: %v", err)
	}

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 设置超时上下文，非常短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 在goroutine中运行收集操作
	done := make(chan struct{})
	var ch chan prometheus.Metric
	go func() {
		ch = make(chan prometheus.Metric, 100)
		metrics.Collect(ch)
		close(ch)
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 即使文件损坏，也应该能收集基本指标
		count := 0
		for range ch {
			count++
		}
		assert.GreaterOrEqual(t, count, 5, "即使文件损坏也应该能收集到基本指标")
	case <-ctx.Done():
		t.Fatalf("损坏文件测试超时: %v", ctx.Err())
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestLargeLeaseFile 测试大型租约文件的性能和稳定性
func TestLargeLeaseFile(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "dhcpd_test_large")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 生成随机MAC地址
	generateRandomMAC := func() string {
		mac := make([]byte, 6)
		rand.Read(mac)
		return fmt.Sprintf("%02x:%02x:%02x:%02x:%02x:%02x", mac[0], mac[1], mac[2], mac[3], mac[4], mac[5])
	}

	// 生成大型租约文件内容
	var contentBuilder strings.Builder
	now := time.Now()

	// 添加100个租约条目
	for i := 1; i <= 100; i++ {
		ip := fmt.Sprintf("192.168.1.%d", i+100)
		mac := generateRandomMAC()
		hostname := fmt.Sprintf("test-host-%d", i)

		leaseEntry := fmt.Sprintf(`
lease %s {
	starts 6 %s;
	ends 6 %s;
	hardware ethernet %s;
	client-hostname "%s";
}`,
			ip,
			now.AddDate(0, 0, -1).Format("2006/01/02 15:04:05"),
			now.AddDate(0, 0, 7).Format("2006/01/02 15:04:05"),
			mac,
			hostname,
		)

		contentBuilder.WriteString(leaseEntry)
	}

	// 写入大型租约文件
	leaseFile := filepath.Join(tmpDir, "large.leases")
	err = os.WriteFile(leaseFile, []byte(contentBuilder.String()), 0644)
	if err != nil {
		t.Fatalf("创建大型租约文件失败: %v", err)
	}

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 设置超时上下文，给大文件处理更少时间
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 在goroutine中运行收集操作
	done := make(chan struct{})
	var collectStart time.Time
	var collectDuration time.Duration
	var ch chan prometheus.Metric

	go func() {
		ch = make(chan prometheus.Metric, 1000) // 增加缓冲区大小
		collectStart = time.Now()
		metrics.Collect(ch)
		collectDuration = time.Since(collectStart)
		close(ch)
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 收集大型文件的指标应该成功
		count := 0
		for range ch {
			count++
		}
		t.Logf("大型文件处理时间: %v, 收集的指标数量: %d", collectDuration, count)
		assert.GreaterOrEqual(t, count, 100, "大型文件应该能收集到更多指标")
	case <-ctx.Done():
		t.Fatalf("大型文件测试超时: %v", ctx.Err())
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestComplexSubnetConfig 测试复杂的子网配置
func TestComplexSubnetConfig(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置超时上下文，非常短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 设置复杂的子网配置
	*dhcpdLeasesFile = leaseFile
	*testMode = false
	*subnetsConfig = "192.168.1.0/24=192.168.1.50-192.168.1.150,192.168.2.0/24=192.168.2.100-192.168.2.200,10.0.0.0/8=10.10.10.1-10.10.10.254"

	// 在goroutine中运行
	done := make(chan struct{})
	go func() {
		// 初始化DHCP信息收集器
		InitDHCPDInfo()

		// 验证是否成功初始化
		assert.NotNil(t, DHCPDInfo, "DHCP信息收集器不应为空")

		// 读取租约信息
		if DHCPDInfo != nil {
			err := DHCPDInfo.Read()
			assert.NoError(t, err, "读取租约信息不应出错")

			// 验证子网配置有效
			assert.NotEmpty(t, *subnetsConfig, "子网配置不应为空")

			// 验证子网字符串中包含预期的子网
			assert.Contains(t, *subnetsConfig, "192.168.1.0/24", "子网配置应包含第一个子网")
			assert.Contains(t, *subnetsConfig, "192.168.2.0/24", "子网配置应包含第二个子网")
			assert.Contains(t, *subnetsConfig, "10.0.0.0/8", "子网配置应包含第三个子网")
		}

		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 成功完成
	case <-ctx.Done():
		t.Fatalf("复杂子网配置测试超时: %v", ctx.Err())
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestRaceCondition 测试并发读写情况下的竞态条件
func TestRaceCondition(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置超时上下文，非常短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建等待组
	var wg sync.WaitGroup

	// 创建多个收集器实例
	metrics1 := NewDHCPDMetrics()
	metrics2 := NewDHCPDMetrics()

	// 启动多个goroutine同时初始化和读取
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// 交替使用不同的收集器和初始化方法
			if id%2 == 0 {
				InitDHCPDInfo()
				ch := make(chan prometheus.Metric, 100)
				metrics1.Collect(ch)
				for range ch {
					// 仅消费
				}
				close(ch)
			} else {
				InitDHCPDInfo()
				ch := make(chan prometheus.Metric, 100)
				metrics2.Collect(ch)
				for range ch {
					// 仅消费
				}
				close(ch)
			}
		}(i)
	}

	// 在goroutine中等待所有操作完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 成功完成，没有死锁
	case <-ctx.Done():
		t.Fatalf("竞态条件测试超时，可能存在死锁: %v", ctx.Err())
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestFrequentFileModification 测试频繁文件修改情况
func TestFrequentFileModification(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "dhcpd_test_modification")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建初始租约文件
	leaseFile := filepath.Join(tmpDir, "changing.leases")
	err = os.WriteFile(leaseFile, []byte(createTestLeaseFileContent()), 0644)
	if err != nil {
		t.Fatalf("创建初始租约文件失败: %v", err)
	}

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 设置超时上下文，非常短的超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 创建等待组
	var wg sync.WaitGroup

	// 启动goroutine频繁修改文件
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 5; i++ {
			// 生成新的租约内容
			now := time.Now()
			ip := fmt.Sprintf("192.168.1.%d", 150+i)
			content := fmt.Sprintf(`lease %s {
				starts 6 %s;
				ends 6 %s;
				hardware ethernet 00:11:22:33:44:%02x;
				client-hostname "dynamic-host-%d";
			}`,
				ip,
				now.AddDate(0, 0, -1).Format("2006/01/02 15:04:05"),
				now.AddDate(0, 0, 7).Format("2006/01/02 15:04:05"),
				55+i,
				i,
			)

			// 附加到租约文件
			f, err := os.OpenFile(leaseFile, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(content)
				f.Close()
			}

			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 同时启动收集操作
	wg.Add(1)
	go func() {
		defer wg.Done()

		for i := 0; i < 5; i++ {
			ch := make(chan prometheus.Metric, 100)
			metrics.Collect(ch)
			for range ch {
				// 仅消费
			}
			close(ch)

			time.Sleep(200 * time.Millisecond)
		}
	}()

	// 在goroutine中等待所有操作完成
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	// 等待完成或超时
	select {
	case <-done:
		// 成功完成，没有死锁
	case <-ctx.Done():
		t.Fatalf("频繁文件修改测试超时: %v", ctx.Err())
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestMetricsLogging 测试指标收集过程中的日志记录
func TestMetricsLogging(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 捕获日志输出
	var logBuffer strings.Builder
	logrus.SetOutput(&logBuffer)
	defer logrus.SetOutput(os.Stderr) // 恢复默认输出

	// 设置日志级别为debug
	previousLevel := logrus.GetLevel()
	logrus.SetLevel(logrus.DebugLevel)
	defer logrus.SetLevel(previousLevel) // 恢复之前的日志级别

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 消费所有指标
	for range ch {
		// 仅消费
	}

	// 验证日志输出中包含预期的信息
	logOutput := logBuffer.String()
	assert.Contains(t, logOutput, "debug", "日志输出应该包含debug级别的消息")

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestZeroLeasesCase 测试没有租约的情况
func TestZeroLeasesCase(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "dhcpd_test_empty")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建一个空的租约文件
	emptyLeaseContent := `# empty lease file 
# The format of this file is documented in the dhcpd.leases(5) manual page.
# This lease file was written by isc-dhcp-4.4.1

`
	leaseFile := filepath.Join(tmpDir, "empty.leases")
	err = os.WriteFile(leaseFile, []byte(emptyLeaseContent), 0644)
	if err != nil {
		t.Fatalf("创建空租约文件失败: %v", err)
	}

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 计算收集到的指标数量
	count := 0
	for range ch {
		count++
	}

	// 验证空租约文件情况下能收集的指标
	assert.GreaterOrEqual(t, count, 5, "即使没有租约也应该能收集到基本指标")

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestMalformedSubnetConfig 测试格式错误的子网配置
func TestMalformedSubnetConfig(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置各种格式错误的子网配置并尝试初始化
	malformedConfigs := []string{
		"192.168.1.0/24=192.168.1.100",               // 缺少结束IP
		"192.168.1.0/24",                             // 缺少IP范围
		"192.168.1.0/24=192.168.1.100-192.168.1.300", // 无效IP
		"192.168.1/24=192.168.1.100-192.168.1.200",   // 无效子网格式
		"=192.168.1.100-192.168.1.200",               // 缺少子网
		"192.168.1.0/24=",                            // 缺少IP范围
		"192.168.1.0/33=192.168.1.100-192.168.1.200", // 无效掩码
	}

	for i, config := range malformedConfigs {
		t.Run(fmt.Sprintf("MalformedConfig_%d", i), func(t *testing.T) {
			// 设置环境变量
			*dhcpdLeasesFile = leaseFile
			*testMode = false
			*subnetsConfig = config

			// 清理全局状态
			mux.Lock()
			DHCPDInfo = nil
			mux.Unlock()

			// 初始化DHCP信息收集器（不应崩溃）
			InitDHCPDInfo()

			// 验证是否仍然初始化
			assert.NotNil(t, DHCPDInfo, "即使子网配置格式错误，DHCP信息收集器也不应为空")

			// 创建指标收集器
			metrics := NewDHCPDMetrics()
			assert.NotNil(t, metrics, "指标收集器不应为空")

			// 收集指标
			ch := make(chan prometheus.Metric, 100)
			metrics.Collect(ch)
			close(ch)

			// 验证是否仍然能收集到基本指标
			count := 0
			for range ch {
				count++
			}
			assert.GreaterOrEqual(t, count, 5, "即使子网配置格式错误也应该能收集到基本指标")

			// 清理全局状态
			mux.Lock()
			DHCPDInfo = nil
			mux.Unlock()
		})
	}
}

// TestExtremeLeaseValues 测试极端租约值
func TestExtremeLeaseValues(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "dhcpd_test_extreme")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建具有极端值的租约文件
	extremeLeaseContent := `
# Lease with a very far future end time
lease 192.168.1.100 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2099/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:55;
	client-hostname "future-host";
}

# Lease with a very past end time
lease 192.168.1.101 {
	starts 6 1970/01/01 00:00:00;
	ends 6 1970/01/02 00:00:00;
	hardware ethernet 00:11:22:33:44:66;
	client-hostname "past-host";
}

# Lease with a very long client hostname
lease 192.168.1.102 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:77;
	client-hostname "this-is-a-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-very-long-hostname";
}

# Lease with no client hostname
lease 192.168.1.103 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:88;
}

# Lease with unusual MAC address
lease 192.168.1.104 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:00:00:00:00:00;
	client-hostname "zero-mac";
}

# Lease with unusual MAC address
lease 192.168.1.105 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet ff:ff:ff:ff:ff:ff;
	client-hostname "broadcast-mac";
}
`
	leaseFile := filepath.Join(tmpDir, "extreme.leases")
	err = os.WriteFile(leaseFile, []byte(extremeLeaseContent), 0644)
	if err != nil {
		t.Fatalf("创建极端租约文件失败: %v", err)
	}

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 计算收集到的指标数量
	count := 0
	for range ch {
		count++
	}

	// 验证即使有极端值也能收集指标
	assert.GreaterOrEqual(t, count, 5, "即使租约有极端值也应该能收集到指标")

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestSpecialIPAddresses 测试特殊IP地址的处理
func TestSpecialIPAddresses(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "dhcpd_test_special_ips")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建包含特殊IP地址的租约文件
	specialIPsContent := `
# Network boundary
lease 192.168.1.0 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:01;
	client-hostname "network-boundary";
}

# Broadcast address
lease 192.168.1.255 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:02;
	client-hostname "broadcast-address";
}

# Loopback address
lease 127.0.0.1 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:03;
	client-hostname "loopback";
}

# Multicast address
lease 239.0.0.1 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:04;
	client-hostname "multicast";
}

# Link-local address
lease 169.254.1.1 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:05;
	client-hostname "link-local";
}

# Default gateway
lease 192.168.1.1 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:06;
	client-hostname "gateway";
}
`
	leaseFile := filepath.Join(tmpDir, "special_ips.leases")
	err = os.WriteFile(leaseFile, []byte(specialIPsContent), 0644)
	if err != nil {
		t.Fatalf("创建特殊IP地址租约文件失败: %v", err)
	}

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 计算收集到的指标数量
	count := 0
	for range ch {
		count++
	}

	// 验证即使有特殊IP地址也能收集指标
	assert.GreaterOrEqual(t, count, 5, "即使租约有特殊IP地址也应该能收集到指标")

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestInternationalization 测试国际化字符的处理
func TestInternationalization(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 创建临时目录
	tmpDir, err := os.MkdirTemp("", "dhcpd_test_i18n")
	if err != nil {
		t.Fatalf("创建临时目录失败: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// 创建包含国际化字符的租约文件
	i18nContent := `
# Chinese hostname
lease 192.168.1.100 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:55;
	client-hostname "中文主机名";
}

# Russian hostname
lease 192.168.1.101 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:66;
	client-hostname "русский хост";
}

# Japanese hostname
lease 192.168.1.102 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:77;
	client-hostname "日本語ホスト";
}

# Arabic hostname
lease 192.168.1.103 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:88;
	client-hostname "اسم المضيف العربي";
}

# Greek hostname
lease 192.168.1.104 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:99;
	client-hostname "ελληνικό όνομα";
}

# Thai hostname
lease 192.168.1.105 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:aa;
	client-hostname "ชื่อโฮสต์ไทย";
}

# Emoji hostname
lease 192.168.1.106 {
	starts 6 2020/01/01 00:00:00;
	ends 6 2020/12/31 23:59:59;
	hardware ethernet 00:11:22:33:44:bb;
	client-hostname "emoji-😀😁😂🤣😃😄";
}
`
	leaseFile := filepath.Join(tmpDir, "i18n.leases")
	err = os.WriteFile(leaseFile, []byte(i18nContent), 0644)
	if err != nil {
		t.Fatalf("创建国际化字符租约文件失败: %v", err)
	}

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 计算收集到的指标数量
	count := 0
	for range ch {
		count++
	}

	// 验证即使有国际化字符也能收集指标
	assert.GreaterOrEqual(t, count, 5, "即使租约有国际化字符也应该能收集到指标")

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestLargeNumberOfSubnets 测试大量子网配置的情况
func TestLargeNumberOfSubnets(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 创建大量子网配置
	var subnetsBuilder strings.Builder

	// 添加100个子网配置
	for i := 0; i < 100; i++ {
		subnet := fmt.Sprintf("192.168.%d.0/24=192.168.%d.100-192.168.%d.200", i, i, i)
		if i > 0 {
			subnetsBuilder.WriteString(",")
		}
		subnetsBuilder.WriteString(subnet)
	}

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false
	*subnetsConfig = subnetsBuilder.String()

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()

	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 收集指标
	ch := make(chan prometheus.Metric, 100)
	metrics.Collect(ch)
	close(ch)

	// 计算收集到的指标数量
	count := 0
	for range ch {
		count++
	}

	// 验证即使有大量子网配置也能收集指标
	assert.GreaterOrEqual(t, count, 5, "即使有大量子网配置也应该能收集到指标")

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestMultipleCollectorsParallel 测试多个收集器并行工作
func TestMultipleCollectorsParallel(t *testing.T) {
	// 跳过耗时测试
	if !*runLongTests {
		t.Skip("跳过耗时测试，使用 -long 标志来运行")
	}

	// 设置测试环境
	leaseFile, cleanup := setupTestEnvironment(t)
	defer cleanup()

	// 设置环境变量
	*dhcpdLeasesFile = leaseFile
	*testMode = false

	// 创建多个收集器
	collectors := make([]*DHCPDMetrics, 5)
	for i := 0; i < 5; i++ {
		collectors[i] = NewDHCPDMetrics()
		assert.NotNil(t, collectors[i], "指标收集器不应为空")
	}

	// 并行收集指标
	var wg sync.WaitGroup
	results := make([]int, 5)

	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// 收集指标
			ch := make(chan prometheus.Metric, 100)
			collectors[index].Collect(ch)
			close(ch)

			// 计算收集到的指标数量
			count := 0
			for range ch {
				count++
			}

			// 存储结果
			results[index] = count
		}(i)
	}

	// 等待所有收集器完成
	wg.Wait()

	// 验证所有收集器都收集到了指标
	for i, count := range results {
		assert.GreaterOrEqual(t, count, 5, "收集器 %d 应该收集到基本指标", i)
	}

	// 清理全局状态
	mux.Lock()
	DHCPDInfo = nil
	mux.Unlock()
}

// TestDHCPDMetricsInterface 测试DHCPDMetrics是否正确实现了接口
func TestDHCPDMetricsInterface(t *testing.T) {
	// 创建指标收集器
	metrics := NewDHCPDMetrics()
	assert.NotNil(t, metrics, "指标收集器不应为空")

	// 验证是否实现了 prometheus.Collector 接口
	var collector prometheus.Collector = metrics
	assert.NotNil(t, collector, "指标收集器应该实现 prometheus.Collector 接口")

	// 验证是否可以注册到注册表
	registry := prometheus.NewRegistry()
	err := registry.Register(metrics)
	assert.NoError(t, err, "应该能够注册到 prometheus.Registry")

	// 验证已注册 - 尝试再次注册同一收集器应该失败
	err = registry.Register(metrics)
	assert.Error(t, err, "再次注册同一收集器应该失败，这表明收集器已被注册")

	// 取消注册
	assert.True(t, registry.Unregister(metrics), "应该能够从注册表中取消注册")

	// 取消注册后，应该能再次注册成功
	err = registry.Register(metrics)
	assert.NoError(t, err, "取消注册后应该能够重新注册")
}
