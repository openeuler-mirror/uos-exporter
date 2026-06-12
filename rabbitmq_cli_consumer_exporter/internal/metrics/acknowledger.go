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

// TODO: implement
