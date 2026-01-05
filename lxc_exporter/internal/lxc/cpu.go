package lxc

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	CPUStatFile = "cpu.stat"
)

type CPUStat struct {
	Usage         float64
	User          float64
	System        float64
	ForceIdle     float64
	NrPeriods     float64
	NrThrottled   float64
	ThrottledUsec float64
	Nrbursts      float64
	BurstUsec     float64
}

func (l *Lxc) GetCPUStat(containerName string) (CPUStat, error) {
	if !l.containerExists(containerName) {
		return CPUStat{}, ErrorContainerNotFound
	}
	statContent, err := l.readCPUStatFile(containerName)
	if err != nil {
		return CPUStat{}, err
	}
	return parseCPUStat(statContent)
}
func (l *Lxc) GetCPUStatAll() ([]CPUStat, error) {
	return nil, nil
}

func (l *Lxc) readCPUStatFile(containerName string) ([]byte, error) {
	cpuStatPath, err := l.getCPUStatFilePath(containerName)
	if err != nil {
		return nil, err
	}
	cleanPath := filepath.Clean(cpuStatPath)
	if !strings.HasPrefix(cleanPath, cgroupPath) {
		return nil, fmt.Errorf("config file must be located within %s", cgroupPath)
	}
	return os.ReadFile(cleanPath)
}

func (l *Lxc) getCPUStatFilePath(containerName string) (string, error) {
	cpuStatPath := path.Join(cgroupPath,
		//lxcPrefix,
		l.getContainerPathName(containerName),
		CPUStatFile)
	_, err := os.Stat(cpuStatPath)
	if err != nil {
		return "", err
	}
	return cpuStatPath, nil
}

func parseCPUStat(content []byte) (CPUStat, error) {
	var (
		usage         float64
		user          float64
		system        float64
		forceIdle     float64
		nrPeriods     float64
		nrThrottled   float64
		throttledUsec float64
		nrbursts      float64
		burstUsec     float64
	)
	lines := strings.Split(
		strings.TrimSpace(
			string(content)), "\n")
	if len(lines) < 3 {
		return CPUStat{},
			errors.New("invalid cpu.stat format")
	}
	usage, err := parseCPUStatLine(lines[0])
	if err != nil {
		return CPUStat{}, err
	}
	user, err = parseCPUStatLine(lines[1])
	if err != nil {
		return CPUStat{}, err
	}
	system, err = parseCPUStatLine(lines[2])
	if err != nil {
		return CPUStat{}, err
	}
	if len(lines) == 9 {
		forceIdle, err = parseCPUStatLine(lines[3])
		if err != nil {
			return CPUStat{}, err
		}
		nrPeriods, err = parseCPUStatLine(lines[4])
		if err != nil {
			return CPUStat{}, err
		}
		nrThrottled, err = parseCPUStatLine(lines[5])
		if err != nil {
			return CPUStat{}, err
		}
		throttledUsec, err = parseCPUStatLine(lines[6])
		if err != nil {
			return CPUStat{}, err
		}
		nrbursts, err = parseCPUStatLine(lines[7])
		if err != nil {
			return CPUStat{}, err
		}
		burstUsec, err = parseCPUStatLine(lines[8])
		if err != nil {
			return CPUStat{}, err
		}
		return CPUStat{
			Usage:         usage,
			User:          user,
			System:        system,
			ForceIdle:     forceIdle,
			NrPeriods:     nrPeriods,
			NrThrottled:   nrThrottled,
			ThrottledUsec: throttledUsec,
			Nrbursts:      nrbursts,
			BurstUsec:     burstUsec,
		}, nil
	} else {
		return CPUStat{
			Usage:  usage,
			User:   user,
			System: system,
		}, nil
	}
}

// 解析 cpu.stat 文件中的一行，例如 "usage_usec 123456789"
func parseCPUStatLine(line string) (float64, error) {
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return 0, errors.New("invalid cpu.stat line format")
	}
	return parseFloat64(fields[1])
}
