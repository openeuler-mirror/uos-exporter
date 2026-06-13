package metrics

import (
	"os/exec"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Processor describes the interface used by the consumer to process messages.
type Processor interface {
	Process(Delivery) error
}

// New creates a new processor instance.

// TODO: implement functions
