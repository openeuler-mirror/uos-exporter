package metrics

import (
	"strings"
	"testing"
	"os"
	"errors"

	"openvpn_exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestOpenVPNCollector_NewOpenVPNCollector 测试创建OpenVPN收集器

// TODO: implement functions
