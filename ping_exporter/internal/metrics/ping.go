package metrics

import (
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/digineo/go-ping"
	mon "github.com/digineo/go-ping/monitor"
	log "github.com/sirupsen/logrus"
)

var (
	targetFlag     = []string{"8.8.8.8", "1.1.1.1", "github.com"}
	pingInterval   = "5s"
	pingTimeout    = "4s"
	pingSize       = uint16(56)
	historySize    = 10
	dnsRefresh     = "1m"
	dnsNameServer  = ""
	disableIPv6    = true
	disableIPv4    = false
	desiredTargets = &targets{}
)


// TODO: implement functions
