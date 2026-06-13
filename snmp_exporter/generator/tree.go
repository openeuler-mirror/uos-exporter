package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	m "snmp_exporter/internal/metrics"

	"github.com/sirupsen/logrus"
)

// These types have one following the other.
// We need to check indexes and sequences have them
// in the right order, so the exporter can handle them.
var combinedTypes = map[string]string{
	"InetAddress":            "InetAddressType",
	"InetAddressMissingSize": "InetAddressType",
	"LldpPortId":             "LldpPortIdSubtype",
}

// Helper to walk MIB nodes.

// TODO: implement functions
