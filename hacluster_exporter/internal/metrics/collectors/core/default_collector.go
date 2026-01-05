package core

import (
	"os"

	"hacluster_exporter/internal/clock"

	"github.com/pkg/errors"
)

const NAMESPACE = "hacluster"

type SubsystemCollector interface {
	GetSubsystem() string
}

type DefaultCollector struct {
	subsystem  string
	Clock      clock.Clock
	timestamps bool
}


// TODO: implement functions
