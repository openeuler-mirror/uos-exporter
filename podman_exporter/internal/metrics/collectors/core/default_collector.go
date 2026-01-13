package core

const NAMESPACE = "podman"

type SubsystemCollector interface {
	GetSubsystem() string
}

type DefaultCollector struct {
	subsystem  string
	timestamps bool
}

func NewDefaultCollector(subsystem string, timestamps bool) DefaultCollector {
	return DefaultCollector{
		subsystem,
		timestamps,
	}
}

func (c *DefaultCollector) GetSubsystem() string {
	return c.subsystem
}
