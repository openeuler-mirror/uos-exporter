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

func NewDefaultCollector(subsystem string, timestamps bool) DefaultCollector {
	return DefaultCollector{
		subsystem,
		&clock.SystemClock{},
		timestamps,
	}
}

func (c *DefaultCollector) GetSubsystem() string {
	return c.subsystem
}

// check that all the given paths exist and are executable files
func CheckExecutables(paths ...string) error {
	for _, path := range paths {
		fileInfo, err := os.Stat(path)
		if err != nil || os.IsNotExist(err) {
			return errors.Errorf("'%s' does not exist", path)
		}
		if fileInfo.IsDir() {
			return errors.Errorf("'%s' is a directory", path)
		}
		if (fileInfo.Mode() & 0111) == 0 {
			return errors.Errorf("'%s' is not executable", path)
		}
	}
	return nil
}
