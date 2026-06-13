package metrics

import (
	"bytes"
	"encoding/json"
	"errors"
	"keepalived_container_exporter/pkg/utils"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

// KeepalivedHostCollectorHost implements Collector for when Keepalived and Keepalived Exporter are both on a same host.
type KeepalivedHostCollectorHost struct {
	pidPath string
	version *version.Version
	useJSON bool

	SIGJSON  syscall.Signal
	SIGDATA  syscall.Signal
	SIGSTATS syscall.Signal
}

// NewKeepalivedHostCollectorHost is creating new instance of KeepalivedHostCollectorHost.

// TODO: implement functions
