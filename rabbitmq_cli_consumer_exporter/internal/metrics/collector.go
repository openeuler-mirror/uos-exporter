package metrics

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bketelsen/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/streadway/amqp"

	"github.com/sirupsen/logrus"
)

const (
	namespace = "rabbitmq_cli_consumer"
)

// RabbitmqCliConsumer implements prometheus.Collector interface and stores required info to collect data.
type RabbitmqCliConsumer struct {
	ProcessCounter  *prometheus.CounterVec
	ProcessDuration prometheus.Histogram
	MessageDuration prometheus.Histogram
	Config          *RabbitmqCliConsumerConfig
}


// TODO: implement
