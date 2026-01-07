package collector

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDHCPDInfo_Read(t *testing.T) {
	// 创建临时测试文件
	content := `lease 192.168.1.100 {
  starts 6 2024/03/15 10:00:00;
  ends 6 2024/03/15 22:00:00;
  hardware ethernet 00:11:22:33:44:55;
  client-hostname "test-host-1";
}
lease 192.168.1.101 {
  starts 6 2024/03/15 09:00:00;
  ends 6 2024/03/15 09:30:00;
  hardware ethernet 00:11:22:33:44:66;
  client-hostname "test-host-2";
}
lease 192.168.1.102 {
  starts 6 2024/03/15 10:30:00;
  ends 6 2024/03/15 22:30:00;
  hardware ethernet 00:11:22:33:44:77;
  client-hostname "test-host-3";
}`

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "dhcpd.leases")
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}

	// 创建 DHCPDInfo 实例
	info, err := NewDHCPDInfo(tmpFile)
	if err != nil {
		t.Fatalf("创建 DHCPDInfo 失败: %v", err)
	}

	// 读取并解析文件
	if err := info.Read(); err != nil {
		t.Fatalf("读取租约文件失败: %v", err)
	}

	// 验证总租约数
	if got := info.GetTotalLeases(); got != 3 {
		t.Errorf("总租约数不匹配, 期望: 3, 实际: %d", got)
	}

	// 设置一个固定的当前时间用于测试
	now := time.Date(2024, 3, 15, 10, 15, 0, 0, time.UTC)
	timeNow := func() time.Time { return now }

	// 验证活跃租约
	activeLeases := info.GetActiveLeases()
	if len(activeLeases) != 2 {
		t.Errorf("活跃租约数不匹配, 期望: 2, 实际: %d", len(activeLeases))
	}

	// 验证第一个活跃租约的详细信息
	if len(activeLeases) > 0 {
		lease := activeLeases[0]
		if lease.IP != "192.168.1.100" {
			t.Errorf("IP 地址不匹配, 期望: 192.168.1.100, 实际: %s", lease.IP)
		}
		if lease.HardwareAddress != "00:11:22:33:44:55" {
			t.Errorf("MAC 地址不匹配, 期望: 00:11:22:33:44:55, 实际: %s", lease.HardwareAddress)
		}
		if lease.Hostname != "test-host-1" {
			t.Errorf("主机名不匹配, 期望: test-host-1, 实际: %s", lease.Hostname)
		}
	}

	// 验证有效和过期租约数量
	if got := info.GetValidLeases(); got != 2 {
		t.Errorf("有效租约数不匹配, 期望: 2, 实际: %d", got)
	}
	if got := info.GetExpiredLeases(); got != 1 {
		t.Errorf("过期租约数不匹配, 期望: 1, 实际: %d", got)
	}

	// 验证文件修改时间
	if info.GetModTime().IsZero() {
		t.Error("文件修改时间未设置")
	}
}
