package metrics

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"openvpn_exporter/config"
	"openvpn_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// RegisterOpenVPNCollector 注册OpenVPN收集器

// TODO: implement functions
