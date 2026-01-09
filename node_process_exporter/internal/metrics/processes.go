package metrics

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"node_process_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/procfs"
)

func init() {
	exporter.Register(NewProcessesCollector())
}

type ProcessesCollector struct {
	*baseMetrics
	fs           procfs.FS
	threadAlloc  *prometheus.Desc
	threadLimit  *prometheus.Desc
	threadsState *prometheus.Desc
	procsState   *prometheus.Desc
	pidUsed      *prometheus.Desc
	pidMax       *prometheus.Desc
	logger       *slog.Logger
}

func NewProcessesCollector() *ProcessesCollector {
	const subsystem = "processes"
	procPath := "/proc"

	fs, err := procfs.NewFS(procPath)
	if err != nil {
		// 如果无法创建procfs，记录错误但继续初始化
		slog.Error("failed to open procfs", "error", err)
	}

	logger := slog.Default()

	return &ProcessesCollector{
		baseMetrics: NewMetrics("node_processes_collect_errors_total", "Number of errors that occurred during processes collection", []string{}),
		fs:          fs,
		threadAlloc: prometheus.NewDesc(
			"node_processes_threads",
			"Allocated threads in system",
			nil, nil,
		),
		threadLimit: prometheus.NewDesc(
			"node_processes_max_threads",
			"Limit of threads in the system",
			nil, nil,
		),
		threadsState: prometheus.NewDesc(
			"node_processes_threads_state",
			"Number of threads in each state.",
			[]string{"thread_state"}, nil,
		),
		procsState: prometheus.NewDesc(
			"node_processes_state",
			"Number of processes in each state.",
			[]string{"state"}, nil,
		),
		pidUsed: prometheus.NewDesc(
			"node_processes_pids",
			"Number of PIDs", nil, nil,
		),
		pidMax: prometheus.NewDesc(
			"node_processes_max_processes",
			"Number of max PIDs limit", nil, nil,
		),
		logger: logger,
	}
}

func (c *ProcessesCollector) Collect(ch chan<- prometheus.Metric) {
	if err := c.Update(ch); err != nil {
		c.logger.Error("Error updating processes metrics", "error", err)
		ch <- prometheus.MustNewConstMetric(c.baseMetrics.desc, prometheus.CounterValue, 1)
	}
}

func (c *ProcessesCollector) Update(ch chan<- prometheus.Metric) error {
	if c.fs == (procfs.FS{}) {
		return fmt.Errorf("procfs not initialized")
	}

	pids, states, threads, threadStates, err := c.getAllocatedThreads()
	if err != nil {
		return fmt.Errorf("unable to retrieve number of allocated threads: %w", err)
	}

	ch <- prometheus.MustNewConstMetric(c.threadAlloc, prometheus.GaugeValue, float64(threads))

	maxThreads, err := c.readUintFromFile(c.procFilePath("sys/kernel/threads-max"))
	if err != nil {
		return fmt.Errorf("unable to retrieve limit number of threads: %w", err)
	}
	ch <- prometheus.MustNewConstMetric(c.threadLimit, prometheus.GaugeValue, float64(maxThreads))

	for state := range states {
		ch <- prometheus.MustNewConstMetric(c.procsState, prometheus.GaugeValue, float64(states[state]), state)
	}

	for state := range threadStates {
		ch <- prometheus.MustNewConstMetric(c.threadsState, prometheus.GaugeValue, float64(threadStates[state]), state)
	}

	pidM, err := c.readUintFromFile(c.procFilePath("sys/kernel/pid_max"))
	if err != nil {
		return fmt.Errorf("unable to retrieve limit number of maximum pids allowed: %w", err)
	}
	ch <- prometheus.MustNewConstMetric(c.pidUsed, prometheus.GaugeValue, float64(pids))
	ch <- prometheus.MustNewConstMetric(c.pidMax, prometheus.GaugeValue, float64(pidM))

	return nil
}

func (c *ProcessesCollector) getAllocatedThreads() (int, map[string]int32, int, map[string]int32, error) {
	p, err := c.fs.AllProcs()
	if err != nil {
		return 0, nil, 0, nil, fmt.Errorf("unable to list all processes: %w", err)
	}
	pids := 0
	thread := 0
	procStates := make(map[string]int32)
	threadStates := make(map[string]int32)

	for _, pid := range p {
		stat, err := pid.Stat()
		if err != nil {
			// PIDs can vanish between getting the list and getting stats.
			if c.isIgnoredError(err) {
				c.logger.Debug("file not found when retrieving stats for pid", "pid", pid.PID, "err", err)
				continue
			}
			c.logger.Debug("error reading stat for pid", "pid", pid.PID, "err", err)
			return 0, nil, 0, nil, fmt.Errorf("error reading stat for pid %d: %w", pid.PID, err)
		}
		pids++
		procStates[stat.State]++
		thread += stat.NumThreads
		err = c.getThreadStates(pid.PID, stat, threadStates)
		if err != nil {
			return 0, nil, 0, nil, err
		}
	}
	return pids, procStates, thread, threadStates, nil
}

func (c *ProcessesCollector) getThreadStates(pid int, pidStat procfs.ProcStat, threadStates map[string]int32) error {
	fs, err := procfs.NewFS(c.procFilePath(path.Join(strconv.Itoa(pid), "task")))
	if err != nil {
		if c.isIgnoredError(err) {
			c.logger.Debug("file not found when retrieving tasks for pid", "pid", pid, "err", err)
			return nil
		}
		c.logger.Debug("error reading tasks for pid", "pid", pid, "err", err)
		return fmt.Errorf("error reading task for pid %d: %w", pid, err)
	}

	t, err := fs.AllProcs()
	if err != nil {
		if c.isIgnoredError(err) {
			c.logger.Debug("file not found when retrieving tasks for pid", "pid", pid, "err", err)
			return nil
		}
		return fmt.Errorf("unable to list all threads for pid: %d %w", pid, err)
	}

	for _, thread := range t {
		if pid == thread.PID {
			threadStates[pidStat.State]++
			continue
		}
		threadStat, err := thread.Stat()
		if err != nil {
			if c.isIgnoredError(err) {
				c.logger.Debug("file not found when retrieving stats for thread", "pid", pid, "threadId", thread.PID, "err", err)
				continue
			}
			c.logger.Debug("error reading stat for thread", "pid", pid, "threadId", thread.PID, "err", err)
			return fmt.Errorf("error reading stat for pid:%d thread:%d err:%w", pid, thread.PID, err)
		}
		threadStates[threadStat.State]++
	}
	return nil
}

func (c *ProcessesCollector) isIgnoredError(err error) bool {
	if errors.Is(err, os.ErrNotExist) || strings.Contains(err.Error(), syscall.ESRCH.Error()) {
		return true
	}
	return false
}

func (c *ProcessesCollector) procFilePath(name string) string {
	return filepath.Join("/proc", name)
}

func (c *ProcessesCollector) readUintFromFile(path string) (uint64, error) {
	cleanPath := filepath.Clean(path)
	statDir := "/proc"
	if !strings.HasPrefix(cleanPath, statDir) {
		return 0, fmt.Errorf("stat file must be located within %s", statDir)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}
	return value, nil
}
