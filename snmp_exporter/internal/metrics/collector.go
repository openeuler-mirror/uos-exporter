package metrics

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/itchyny/timefmt-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

var (
	// 64-bit float mantissa: https://en.wikipedia.org/wiki/Double-precision_floating-point_format
	float64Mantissa uint64 = 9007199254740992
	wrapCounters           = true
	srcAddress             = ""
)

// Types preceded by an enum with their actual type.
var combinedTypeMapping = map[string]map[int]string{
	"InetAddress": {
		1: "InetAddressIPv4",
		2: "InetAddressIPv6",
	},
	"InetAddressMissingSize": {
		1: "InetAddressIPv4",
		2: "InetAddressIPv6",
	},
	"LldpPortId": {
		1: "DisplayString",
		2: "DisplayString",
		3: "PhysAddress48",
		5: "DisplayString",
		7: "DisplayString",
	},
}


// TODO: implement functions
