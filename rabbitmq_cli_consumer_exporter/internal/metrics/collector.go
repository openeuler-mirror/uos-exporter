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

func NewRabbitmqCliConsumerCollector(config *RabbitmqCliConsumerConfig) *RabbitmqCliConsumer {
	// ProcessCounter is a Prometheus metric describing the total number of processes executed.
	processCounter := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "process_total",
			Help:      "The total number of processes executed.",
		},
		[]string{"exit_code"},
	)

	// ProcessDuration is a Prometheus metric describing the time spent by the consumer to process the message.
	processDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "process_duration_seconds",
			Help:      "The time spent by the consumer to process the message.",
		},
	)

	// MessageDuration is a Prometheus metric describing the time spent from publishing to finished processing the message.
	messageDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "message_duration_seconds",
			Help:      "The time spent from publishing to finished processing the message.",
		},
	)
	kc := &RabbitmqCliConsumer{
		ProcessCounter:  processCounter,
		ProcessDuration: processDuration,
		MessageDuration: messageDuration,
		Config:          config,
	}

	return kc
}

// Collect get metrics and add to prometheus metric channel.
func (k *RabbitmqCliConsumer) Collect(ch chan<- prometheus.Metric) {
	cfg, err := LoadConfiguration(k.Config)
	if err != nil {
		logrus.Errorf("failed to load configuration : %v", err)
	}
	k.ProcessCounter.Collect(ch)
	k.ProcessDuration.Collect(ch)
	k.MessageDuration.Collect(ch)

	logrus.Info("Collecting metrics")
	l, infW, errW, err := NewLogFromConfig(cfg)
	if err != nil {
		logrus.Errorf("failed to load log configuration : %v", err)
	}

	b := CreateBuilder(k.Config.Pipe, cfg.RabbitMq.Compression, k.Config.Include_metadata)
	builder, err := NewBuilder(b, k.Config.Executable, false, l, infW, errW)
	if err != nil {
		logrus.Errorf("failed to create command builder: %v", err)
	}

	ack := NewAcknowledgerFromConfig(cfg)
	p := ProcessNew(builder, ack, logrus.StandardLogger(), k.ProcessCounter, k.ProcessDuration, k.MessageDuration)

	client, err := NewConsumerFromConfig(cfg, p, logrus.StandardLogger())
	if err != nil {
		logrus.Errorf("failed to create consumer config: %v", err)
	}
	defer client.Close()

	errs := make(chan error)

	go func() {
		errs <- consume(client, l)
	}()
}

// Describe outputs metrics descriptions.
func (k *RabbitmqCliConsumer) Describe(ch chan<- *prometheus.Desc) {
	logrus.Info("Get prometheus descriptions")
}

func CreateBuilder(pipe, compression, metadata bool) Builder {
	if pipe {
		return &PipeBuilder{}
	}

	return &ArgumentBuilder{
		Compressed:   compression,
		WithMetadata: metadata,
	}
}

// LoadConfiguration checks the configuration flags, loads the config from file and updates the config according the flags.
func LoadConfiguration(config *RabbitmqCliConsumerConfig) (*Config, error) {
	file := config.Configuration
	url := config.Url
	queue := config.Queue_name

	if file == "" && url == "" && queue == "" && config.Executable == "" {
		logrus.Error("configuration file or url or queue or executable must be specified")
		return nil, fmt.Errorf("configuration file or url or queue or executable must be specified")
	}

	cfg, err := configuration(file)
	if err != nil {
		logrus.Errorf("failed parsing configuration: %s", err)
		return nil, fmt.Errorf("failed parsing configuration: %s", err)
	}

	if len(url) > 0 {
		cfg.RabbitMq.AmqpUrl = url
	}

	if queue != "" {
		cfg.RabbitMq.Queue = queue
	}

	cfg.Logs.NoDateTime = config.No_datetime

	cfg.RabbitMq.Stricfailure = config.Strict_exit_code

	cfg.QueueSettings.Nodeclare = config.No_declare

	return cfg, nil
}

func configuration(file string) (*Config, error) {
	if file == "" {
		return CreateFromString("")
	}

	return LoadAndParse(file)
}

func consume(client *Consumer, l logr.Logger) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := startConsuming(client, ctx)
	sig := setupSignalHandler()

	select {
	case <-sig:
		return handleSignalInterrupt(l, done, cancel)
	case err := <-done:
		return checkConsumeError(err)
	}
}

func startConsuming(client *Consumer, ctx context.Context) chan error {
	done := make(chan error)
	go func() {
		done <- client.Consume(ctx)
	}()
	return done
}

func setupSignalHandler() chan os.Signal {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGTERM)
	return sig
}

func handleSignalInterrupt(l logr.Logger, done chan error, cancel context.CancelFunc) error {
	logrus.Info("Cancel consumption of messages.")
	l.Info("Cancel consumption of messages.")
	cancel()
	return checkConsumeError(<-done)
}

func checkConsumeError(err error) error {
	switch err.(type) {
	case *amqp.Error:
		if strings.Contains(err.Error(), "Exception (320) Reason:") {
			return fmt.Errorf("connection closed: %v", err.(*amqp.Error).Reason)
		}
		return err

	case *AcknowledgmentError:
		return fmt.Errorf("connection closed: %v", err)

	default:
		return err
	}
}
