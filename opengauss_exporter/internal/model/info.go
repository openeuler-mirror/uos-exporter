package model

// OpenGaussInfo 表示从 OpenGauss 中采集到的基本信息
type OpenGaussInfo struct {
	Up                bool
	Version           string
	UptimeSeconds     float64
	DatabaseCount     int64
	ConnectionCurrent int64
	ConnectionMax     int64
	ActiveBackends    int64
	IdleBackends      int64
	WaitingBackends   int64
}
