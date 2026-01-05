package icmp

import (
	"encoding/binary"
	"fmt"
	"math"
	"net"
	"time"

	"network_exporter/pkg/common"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// https://hechao.li/2018/09/27/How-Is-Ping-Deduplexed/
const (
	protocolICMP     = 1  // Internet Control Message
	protocolIPv6ICMP = 58 // ICMP for IPv6
)

// Icmp 执行真实的ICMP测试

// TODO: implement functions
