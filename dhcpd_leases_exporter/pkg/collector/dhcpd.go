package collector

import (
	"bufio"
	"net"
	"os"
	"regexp"
	"strings"
	"time"
	// "github.com/sirupsen/logrus"
)

// Lease 表示一个 DHCP 租约
type Lease struct {
	IP              string
	HardwareAddress string
	Hostname        string
	StartTime       time.Time
	EndTime         time.Time
	Abandoned       bool
	IsNew           bool
	IsRenewed       bool
	VendorClass     string
	BindingState    string
}

// PoolStats 表示地址池统计信息
type PoolStats struct {
	UsageRatio         float64
	AvailableAddresses int
	TotalAddresses     int
}

// SubnetInfo 表示子网信息
type SubnetInfo struct {
	RangeStart string
	RangeEnd   string
	Network    *net.IPNet
}

// ServerInfo 表示 DHCP 服务器信息
type ServerInfo struct {
	Version string
	Uptime  time.Duration
}

// DHCPDInfo 表示 DHCP 服务器信息
type DHCPDInfo struct {
	filePath   string
	leases     []*Lease
	poolStats  map[string]*PoolStats
	subnets    map[string]*SubnetInfo
	serverInfo *ServerInfo
	modTime    time.Time
}

var (
	reStartTime = regexp.MustCompile(`starts \d+ ([^;]+);`)
	reEndTime   = regexp.MustCompile(`ends \d+ ([^;]+);`)
	reHostname  = regexp.MustCompile(`client-hostname "([^"]+)";`)
	reHWAddr    = regexp.MustCompile(`hardware ethernet ([^;]+);`)
	reAbandoned = regexp.MustCompile(`abandoned;`)
	reBinding   = regexp.MustCompile(`binding state ([^;]+);`)
	reVendor    = regexp.MustCompile(`vendor-class-identifier "([^"]+)";`)
)

// NewDHCPDInfo 创建新的 DHCPDInfo 实例
func NewDHCPDInfo(filePath string) *DHCPDInfo {
	return &DHCPDInfo{
		filePath:  filePath,
		leases:    make([]*Lease, 0),
		poolStats: make(map[string]*PoolStats),
		subnets:   make(map[string]*SubnetInfo),
		serverInfo: &ServerInfo{
			Version: "unknown",
		},
	}
}

// Read 读取并解析 DHCP 租约文件
func (d *DHCPDInfo) Read() error {
	file, err := os.Open(d.filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// 获取文件修改时间
	fileInfo, err := file.Stat()
	if err != nil {
		return err
	}
	d.modTime = fileInfo.ModTime()

	// 清空旧数据
	d.leases = make([]*Lease, 0)

	scanner := bufio.NewScanner(file)
	var currentLease *Lease
	var leaseLines []string

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "lease" {
			// 开始新的租约
			if currentLease != nil && len(leaseLines) > 0 {
				d.parseLease(currentLease, leaseLines)
			}
			currentLease = &Lease{}
			leaseLines = make([]string, 0)
			continue
		}

		if currentLease != nil {
			leaseLines = append(leaseLines, line)
			if line == "}" {
				d.parseLease(currentLease, leaseLines)
				currentLease = nil
				leaseLines = nil
			}
		}
	}

	// 处理最后一个租约
	if currentLease != nil && len(leaseLines) > 0 {
		d.parseLease(currentLease, leaseLines)
	}

	// 更新地址池统计信息
	d.updatePoolStats()

	return scanner.Err()
}

// parseLease 解析单个租约信息
func (d *DHCPDInfo) parseLease(lease *Lease, lines []string) {
	leaseText := strings.Join(lines, " ")

	// 解析 IP 地址
	if matches := regexp.MustCompile(`lease ([^ ]+) {`).FindStringSubmatch(leaseText); len(matches) > 1 {
		lease.IP = matches[1]
	}

	// 解析开始时间
	if matches := reStartTime.FindStringSubmatch(leaseText); len(matches) > 1 {
		if t, err := time.Parse("2006/01/02 15:04:05", matches[1]); err == nil {
			lease.StartTime = t
		}
	}

	// 解析结束时间
	if matches := reEndTime.FindStringSubmatch(leaseText); len(matches) > 1 {
		if t, err := time.Parse("2006/01/02 15:04:05", matches[1]); err == nil {
			lease.EndTime = t
		}
	}

	// 解析主机名
	if matches := reHostname.FindStringSubmatch(leaseText); len(matches) > 1 {
		lease.Hostname = matches[1]
	}

	// 解析硬件地址
	if matches := reHWAddr.FindStringSubmatch(leaseText); len(matches) > 1 {
		lease.HardwareAddress = matches[1]
	}

	// 检查是否被废弃
	lease.Abandoned = reAbandoned.MatchString(leaseText)

	// 解析绑定状态
	if matches := reBinding.FindStringSubmatch(leaseText); len(matches) > 1 {
		lease.BindingState = matches[1]
		lease.IsNew = matches[1] == "active"
		lease.IsRenewed = strings.Contains(leaseText, "rewind binding state")
	}

	// 解析厂商类标识
	if matches := reVendor.FindStringSubmatch(leaseText); len(matches) > 1 {
		lease.VendorClass = matches[1]
	}

	d.leases = append(d.leases, lease)
}

// updatePoolStats 更新地址池统计信息
func (d *DHCPDInfo) updatePoolStats() {
	// 按子网分组统计
	subnetStats := make(map[string]struct {
		used  int
		total int
	})

	for _, lease := range d.leases {
		subnet := d.GetSubnetForIP(lease.IP)
		stats := subnetStats[subnet]
		if !lease.EndTime.Before(time.Now()) {
			stats.used++
		}
		stats.total++
		subnetStats[subnet] = stats
	}

	// 更新池统计信息
	for subnet, stats := range subnetStats {
		d.poolStats[subnet] = &PoolStats{
			UsageRatio:         float64(stats.used) / float64(stats.total),
			AvailableAddresses: stats.total - stats.used,
			TotalAddresses:     stats.total,
		}
	}
}

// GetValidLeases 返回有效租约数量
func (d *DHCPDInfo) GetValidLeases() int {
	count := 0
	for _, lease := range d.leases {
		if !lease.EndTime.Before(time.Now()) && !lease.Abandoned {
			count++
		}
	}
	return count
}

// GetExpiredLeases 返回过期租约数量
func (d *DHCPDInfo) GetExpiredLeases() int {
	count := 0
	for _, lease := range d.leases {
		if lease.EndTime.Before(time.Now()) {
			count++
		}
	}
	return count
}

// GetTotalLeases 返回总租约数量
func (d *DHCPDInfo) GetTotalLeases() int {
	return len(d.leases)
}

// GetActiveLeases 返回活跃租约列表
func (d *DHCPDInfo) GetActiveLeases() []*Lease {
	active := make([]*Lease, 0)
	for _, lease := range d.leases {
		if !lease.EndTime.Before(time.Now()) {
			active = append(active, lease)
		}
	}
	return active
}

// GetModTime 返回文件修改时间
func (d *DHCPDInfo) GetModTime() time.Time {
	return d.modTime
}

// GetPoolStats 返回地址池统计信息
func (d *DHCPDInfo) GetPoolStats() map[string]*PoolStats {
	return d.poolStats
}

// GetSubnetForIP 返回 IP 地址所属的子网
func (d *DHCPDInfo) GetSubnetForIP(ipStr string) string {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return "unknown"
	}

	for subnet, info := range d.subnets {
		if info.Network.Contains(ip) {
			return subnet
		}
	}

	// 如果找不到匹配的子网，返回 CIDR 格式的网段
	if ip4 := ip.To4(); ip4 != nil {
		return ip4.Mask(net.CIDRMask(24, 32)).String() + "/24"
	}
	return "unknown"
}

// GetSubnetInfo 返回子网信息
func (d *DHCPDInfo) GetSubnetInfo(subnet string) *SubnetInfo {
	return d.subnets[subnet]
}

// GetServerInfo 返回服务器信息
func (d *DHCPDInfo) GetServerInfo() *ServerInfo {
	return d.serverInfo
}

// AddSubnet 添加子网信息
func (d *DHCPDInfo) AddSubnet(subnet string, rangeStart, rangeEnd string) error {
	_, network, err := net.ParseCIDR(subnet)
	if err != nil {
		return err
	}

	d.subnets[subnet] = &SubnetInfo{
		RangeStart: rangeStart,
		RangeEnd:   rangeEnd,
		Network:    network,
	}
	return nil
}

// SetServerInfo 设置服务器信息
func (d *DHCPDInfo) SetServerInfo(version string, uptime time.Duration) {
	d.serverInfo.Version = version
	d.serverInfo.Uptime = uptime
}
