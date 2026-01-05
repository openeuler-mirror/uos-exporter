package metrics

import (
	"context"
	"net"
	"os"
	"sync"
	"testing"

	log "github.com/sirupsen/logrus"
)

var (
	ipv4Addr, ipv6Addr, ipv4AddrGoogle, ipv6AddrGoogle []net.IPAddr
)


// TODO: implement functions
