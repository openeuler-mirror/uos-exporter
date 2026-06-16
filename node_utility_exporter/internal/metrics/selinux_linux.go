package metrics

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"node_utility_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	subsystemName          = "selinux"
	selinuxConfigPath      = "/etc/selinux/config"
	selinuxEnforceFilePath = "/sys/fs/selinux/enforce"
	statusRefreshInterval  = 5 * time.Minute
	defaultTimeout         = 2 * time.Second
)

var (
	ErrSELinuxNotSupported = errors.New("SELinux not supported on this system")
	ErrStatusReadFailed    = errors.New("failed to read SELinux status")
	ErrConfigReadFailed    = errors.New("failed to read SELinux configuration")
)

type SELinuxStatus int

const (
	StatusUnknown SELinuxStatus = iota
	StatusDisabled
	StatusEnabled
)

type SELinuxMode int

const (
	ModeUnknown SELinuxMode = iota
	ModeDisabled
	ModePermissive
	ModeEnforcing
)

type SELinuxState struct {
	Status      SELinuxStatus
	ConfigMode  SELinuxMode
	CurrentMode SELinuxMode
	LastUpdated time.Time
}

type selinuxCollector struct {
	configModeDesc  *prometheus.Desc
	currentModeDesc *prometheus.Desc
	enabledDesc     *prometheus.Desc
	logger          *slog.Logger
	state           SELinuxState
	stateMutex      sync.RWMutex
	lastChecked     time.Time
}


// TODO: implement functions
