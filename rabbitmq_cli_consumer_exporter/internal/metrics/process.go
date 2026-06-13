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
func ProcessNew(b Builder, a Acknowledger, l *logrus.Logger,
	processCounter *prometheus.CounterVec,
	processDuration prometheus.Histogram,
	messageDuration prometheus.Histogram) Processor {
	return &processor{builder: b, ack: a, log: l,
		ProcessCounter:  processCounter,
		ProcessDuration: processDuration,
		MessageDuration: messageDuration}
}

type processor struct {
	Processor
	builder         Builder
	ack             Acknowledger
	log             *logrus.Logger
	ProcessCounter  *prometheus.CounterVec
	ProcessDuration prometheus.Histogram
	MessageDuration prometheus.Histogram
	mu              sync.Mutex
	cmd             *exec.Cmd
}

// Process creates a new exec command using the builder and executes the command. The message gets acknowledged
// according to the commands exit code using the acknowledger.
func (p *processor) Process(d Delivery) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	cmd, err := p.createCommand(d)
	if err != nil {
		_ = d.Nack(true)
		return NewCreateCommandError(err)
	}
	defer func() { p.cmd = nil }()

	exitCode, duration := p.executeCommand(cmd)
	p.recordMetrics(d, exitCode, duration)

	if err := p.ack.Ack(d, exitCode); err != nil {
		return NewAcknowledgmentError(err)
	}

	return nil
}

func (p *processor) createCommand(d Delivery) (*exec.Cmd, error) {
	cmd, err := p.builder.GetCommand(d.Properties(), d.Info(), d.Body())
	p.cmd = cmd
	return cmd, err
}

func (p *processor) executeCommand(cmd *exec.Cmd) (int, time.Duration) {
	start := time.Now()
	exitCode := p.run()
	return exitCode, time.Since(start)
}

func (p *processor) recordMetrics(d Delivery, exitCode int, duration time.Duration) {
	p.log.Info("Process message...")
	p.ProcessCounter.With(prometheus.Labels{"exit_code": strconv.Itoa(exitCode)}).Inc()
	p.ProcessDuration.Observe(duration.Seconds())
	if !d.Properties().Timestamp.IsZero() {
		p.MessageDuration.Observe(time.Since(d.Properties().Timestamp).Seconds())
	}
}

func (p *processor) run() int {
	p.log.Info("Processing message...")
	defer p.log.Info("Processed!")

	var out []byte
	var err error
	capture := p.cmd.Stdout == nil && p.cmd.Stderr == nil

	if capture {
		out, err = p.cmd.CombinedOutput()
	} else {
		err = p.cmd.Run()
	}

	if err != nil {
		p.log.Info("Failed. Check error log for details.")
		p.log.Errorf("Error: %s\n", err)
		if capture {
			p.log.Errorf("Failed: %s", string(out))
		}

		return exitCode(err)
	}

	return 0
}

func exitCode(err error) int {
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
	}

	return 1
}
