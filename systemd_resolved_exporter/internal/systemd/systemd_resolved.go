package systemd

import (
	"bufio"
	"github.com/godbus/dbus/v5"
	"github.com/sirupsen/logrus"

	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

const (
	namespace       = "systemd_resolved"
	resolvedCommand = "systemd-resolve"
	resolvedArgs    = "--statistics"
)

var (
	statusLineRegex = regexp.MustCompile(`[a-zA-Z ]+: ?[0-9]+`)
	cachePath       = "org.freedesktop.resolve1.Manager.CacheStatistics"
	transactionPath = "org.freedesktop.resolve1.Manager.TransactionStatistics"
	dnssecPath      = "org.freedesktop.resolve1.Manager.DNSSECStatistics"
)


// TODO: implement functions
