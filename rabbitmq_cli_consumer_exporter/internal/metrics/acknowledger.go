package metrics

import "fmt"

// Mapping of script exit codes and message acknowledgment.
const (
	exitAck           = 0
	exitReject        = 3
	exitRejectRequeue = 4
	exitNack          = 5
	exitNackRequeue   = 6
)

// Acknowledger does message acknowledgment depending on the scripts exit code.
type Acknowledger interface {
	Ack(d Delivery, code int) error
}

// New creates new Acknowledger using strict or default behaviour.
func AcknowledgerNew(strict bool, onFailure int) Acknowledger {
	if strict {
		return &Strict{}
	}

	return &Default{
		OnFailure: onFailure,
	}
}

// NewFromConfig creates a new Acknowledger from the given configuration.
func NewAcknowledgerFromConfig(cfg *Config) Acknowledger {
	if cfg.RabbitMq.Stricfailure {
		return &Strict{}
	}

	return &Default{cfg.RabbitMq.Onfailure}
}

// Default is an Acknowledger implementation using a configurable default behaviour for script errors.
type Default struct {
	OnFailure int
}

// Ack acknowledges the message on success or negatively acknowledges or rejects the message according to the configured
// on error behaviour.
func (a Default) Ack(d Delivery, code int) error {
	if code == exitAck {
		_ = d.Ack()
		return nil
	}
	switch a.OnFailure {
	case exitReject:
		_ = d.Reject(false)
	case exitRejectRequeue:
		_ = d.Reject(true)
	case exitNack:
		_ = d.Nack(false)
	case exitNackRequeue:
		_ = d.Nack(true)
	default:
		_ = d.Nack(true)
	}
	return nil
}

// Strict is an Acknowledger implementation strictly using the scripts exit code.
type Strict struct{}

// Ack acknowledges the message on success or negatively acknowledges or rejects the message according to the scripts
// exit code. It is an error if the script does not exit with one of the predefined exit codes.
func (a Strict) Ack(d Delivery, code int) error {
	switch code {
	case exitAck:
		_ = d.Ack()
	case exitReject:
		_ = d.Reject(false)
	case exitRejectRequeue:
		_ = d.Reject(true)
	case exitNack:
		_ = d.Nack(false)
	case exitNackRequeue:
		_ = d.Nack(true)
	default:
		_ = d.Nack(true)
		return fmt.Errorf("unexpected exit code %v", code)
	}

	return nil
}
