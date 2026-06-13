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
func ConsumerNew(conn Connection, ch Channel, p Processor, l *logrus.Logger) *Consumer {
	return &Consumer{
		Connection: conn,
		Channel:    ch,
		Processor:  p,
		Log:        l,
	}
}

// NewFromConfig creates a new consumer instance. The setup of the amqp connection and channel is done according to the
// configuration.
func NewConsumerFromConfig(cfg ConsumerConfig, p Processor, l *logrus.Logger) (*Consumer, error) {
	l.Info("Connecting RabbitMQ...")
	conn, err := amqp.Dial(cfg.AmqpUrl())
	if nil != err {
		return nil, fmt.Errorf("failed connecting RabbitMQ: %v", err)
	}
	l.Info("Connected.")

	l.Info("Opening channel...")
	ch, err := conn.Channel()
	if nil != err {
		return nil, fmt.Errorf("failed to open a channel: %v", err)
	}
	l.Info("Done.")

	if err := Setup(cfg, ch, l); err != nil {
		return nil, err
	}

	return &Consumer{
		Connection: conn,
		Channel:    ch,
		Queue:      cfg.QueueName(),
		Tag:        cfg.ConsumerTag(),
		Processor:  p,
		Log:        l,
	}, nil
}

// Consume subscribes itself to the message queue and starts consuming messages.
func (c *Consumer) Consume(ctx context.Context) error {
	msgs, err := c.startConsuming()
	if err != nil {
		return err
	}

	remoteClose := c.setupChannelCloseNotify()
	done := make(chan error)
	go c.consume(msgs, done)

	return c.handleConsumeEvents(ctx, remoteClose, done)
}

func (c *Consumer) startConsuming() (<-chan amqp.Delivery, error) {
	c.Log.Info("Registering consumer...")
	msgs, err := c.Channel.Consume(c.Queue, c.Tag, false, false, false, false, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to register a consumer: %s", err)
	}
	c.Log.Info("Succeeded registering consumer. Waiting for messages...")
	return msgs, nil
}

func (c *Consumer) setupChannelCloseNotify() chan *amqp.Error {
	remoteClose := make(chan *amqp.Error)
	c.Channel.NotifyClose(remoteClose)
	return remoteClose
}

func (c *Consumer) handleConsumeEvents(ctx context.Context, remoteClose chan *amqp.Error, done chan error) error {
	select {
	case err := <-remoteClose:
		return err

	case <-ctx.Done():
		c.canceled = true
		if err := c.Channel.Cancel(c.Tag, false); err != nil {
			return err
		}
		return <-done

	case err := <-done:
		return err
	}
}

func (c *Consumer) consume(msgs <-chan amqp.Delivery, done chan error) {
	for m := range msgs {
		d := New(m)
		if c.canceled {
			_ = d.Nack(true)
			continue
		}
		if err := c.checkError(c.Processor.Process(d)); err != nil {
			done <- err
			return
		}
	}
	done <- nil
}

func (c *Consumer) checkError(err error) error {
	switch err.(type) {
	case *CreateCommandError:
		c.Log.Error(err)
		return nil

	default:
		return err
	}
}

// Close tears the connection down, taking the channel with it.
func (c *Consumer) Close() error {
	if c.Connection == nil {
		return nil
	}
	return c.Connection.Close()
}

// Connection describes the part of amqp.Connection required by this code base.
type Connection interface {
	io.Closer
	Channel() (*amqp.Channel, error)
}

// Channel describes the part of amqp.Channel required by this code base.
type Channel interface {
	io.Closer
	Cancel(consumer string, noWait bool) error
	Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
	ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error
	NotifyClose(receiver chan *amqp.Error) chan *amqp.Error
	Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
	Qos(prefetchCount, prefetchSize int, global bool) error
	QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error
	QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
}

// Config defines the interface to present configurations to the consumer.
type ConsumerConfig interface {
	AmqpUrl() string
	ConsumerTag() string
	DeadLetterExchange() string
	DeadLetterRoutingKey() string
	ExchangeIsAutoDelete() bool
	ExchangeIsDurable() bool
	ExchangeName() string
	ExchangeType() string
	HasDeadLetterExchange() bool
	HasDeadLetterRouting() bool
	HasExchange() bool
	HasMessageTTL() bool
	HasPriority() bool
	MessageTTL() int32
	MustDeclareQueue() bool
	PrefetchCount() int
	PrefetchIsGlobal() bool
	Priority() int32
	QueueName() string
	RoutingKeys() []string
	QueueIsDurable() bool
	QueueIsExclusive() bool
	QueueIsAutoDelete() bool
	QueueIsNoWait() bool
}

// Setup configures queues, exchanges and bindings in between according to the configuration.
func Setup(cfg ConsumerConfig, ch Channel, l *logrus.Logger) error {
	if err := setupQoS(cfg, ch, l); err != nil {
		return err
	}

	if cfg.MustDeclareQueue() {
		if err := declareQueue(cfg, ch, l); err != nil {
			return err
		}
	}

	// Empty Exchange name means default, no need to declare
	if cfg.HasExchange() {
		if err := declareExchange(cfg, ch, l); err != nil {
			return err
		}
	}

	return nil
}

func setupQoS(cfg ConsumerConfig, ch Channel, l *logrus.Logger) error {
	l.Info("Setting QoS... ")
	if err := ch.Qos(cfg.PrefetchCount(), 0, cfg.PrefetchIsGlobal()); err != nil {
		return fmt.Errorf("failed to set QoS: %v", err)
	}
	l.Info("Succeeded setting QoS.")
	return nil
}

func declareQueue(cfg ConsumerConfig, ch Channel, l *logrus.Logger) error {
	l.Infof("Declaring queue \"%s\"...", cfg.QueueName())

	_, err := ch.QueueDeclare(
		cfg.QueueName(),
		cfg.QueueIsDurable(),
		cfg.QueueIsAutoDelete(),
		cfg.QueueIsExclusive(),
		cfg.QueueIsNoWait(),
		queueArgs(cfg),
	)

	if err == nil {
		return nil
	}

	if isConflictError(err) {
		l.Error("Queue already declared with conflicting settings. You might want to use --no-declare.")
	}
	return fmt.Errorf("failed to declare queue: %v", err)
}

func isConflictError(err error) bool {
	amqpErr, ok := err.(*amqp.Error)
	return ok && amqpErr.Code == 406
}

func declareExchange(cfg ConsumerConfig, ch Channel, l *logrus.Logger) error {
	if err := declareExchangeCore(cfg, ch, l); err != nil {
		return err
	}
	return bindQueuesToExchange(cfg, ch, l)
}

func declareExchangeCore(cfg ConsumerConfig, ch Channel, l *logrus.Logger) error {
	l.Infof("Declaring exchange \"%s\"...", cfg.ExchangeName())
	err := ch.ExchangeDeclare(
		cfg.ExchangeName(),
		cfg.ExchangeType(),
		cfg.ExchangeIsDurable(),
		cfg.ExchangeIsAutoDelete(),
		false,
		false,
		amqp.Table{},
	)
	if err != nil {
		return fmt.Errorf("failed to declare exchange: %v", err)
	}
	return nil
}

func bindQueuesToExchange(cfg ConsumerConfig, ch Channel, l *logrus.Logger) error {
	l.Infof("Binding queue \"%s\" to exchange \"%s\"...", cfg.QueueName(), cfg.ExchangeName())
	for _, routingKey := range cfg.RoutingKeys() {
		if err := ch.QueueBind(
			cfg.QueueName(),
			routingKey,
			cfg.ExchangeName(),
			false,
			nil,
		); err != nil {
			return fmt.Errorf("failed to bind queue to exchange: %v", err)
		}
	}
	return nil
}

func queueArgs(cfg ConsumerConfig) amqp.Table {

	args := make(amqp.Table)

	if cfg.HasMessageTTL() {
		args["x-message-ttl"] = cfg.MessageTTL()
	}

	if cfg.HasDeadLetterExchange() {
		args["x-dead-letter-exchange"] = cfg.DeadLetterExchange()

		if cfg.HasDeadLetterRouting() {
			args["x-dead-letter-routing-key"] = cfg.DeadLetterRoutingKey()
		}
	}

	if cfg.HasPriority() {
		args["x-max-priority"] = cfg.Priority()
	}

	return args
}
