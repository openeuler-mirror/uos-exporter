package metrics

import (
	"syscall"

	"github.com/hashicorp/go-version"
	"github.com/sirupsen/logrus"
)

var (
	sigNumSupportedVersion = version.Must(version.NewVersion("1.3.8"))
	defaultSignals         = map[string]syscall.Signal{"DATA": syscall.SIGUSR1, "STATS": syscall.SIGUSR2}
)

// HasSigNumSupport checks if Keepalived supports --signum command.

// TODO: implement functions
