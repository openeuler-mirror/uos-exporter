package metrics

import (
	"context"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
)

type Consumer struct {
	Connection Connection
	Channel    Channel
	Queue      string
	Tag        string
	Processor  Processor
	Log        *logrus.Logger
	canceled   bool
}

// New creates a new consumer instance. The setup of the amqp connection and channel is expected to be done by the
// calling code.

// TODO: implement functions
