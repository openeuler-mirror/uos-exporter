package metrics

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

const intMax = int(^uint(0) >> 1)

type setupFunc func(*testing.T, *consumeTest) error

type consumeTest struct {
	Name   string
	Setup  setupFunc
	Output string
	Tag    string

	sync        chan bool
	done        chan error
	msgs        chan amqp.Delivery
	ch          *TestChannel
	p           *TestProcessor
	a           *TestAmqpAcknowledger
	dd          []amqp.Delivery
	cancelCount int
}

func newSimpleConsumeTest(name, output string, setup setupFunc) *consumeTest {
	return newConsumeTest(name, output, 1, intMax, setup)
}

func newConsumeTest(name, output string, count uint64, cancelCount int, setup setupFunc) *consumeTest {
	a := new(TestAmqpAcknowledger)
	dd := make([]amqp.Delivery, count)
	for i := uint64(0); i < count; i++ {
		dd[i] = amqp.Delivery{Acknowledger: a, DeliveryTag: i}
	}
	return &consumeTest{
		Name:   name,
		Output: output,
		Setup:  setup,
		Tag:    "ctag",

		sync:        make(chan bool),
		done:        make(chan error),
		msgs:        make(chan amqp.Delivery),
		ch:          new(TestChannel),
		p:           new(TestProcessor),
		a:           a,
		dd:          dd,
		cancelCount: cancelCount,
	}
}

func (ct *consumeTest) Run(t *testing.T) {
	exp := ct.Setup(t, ct)
	c := ConsumerNew(nil, ct.ch, ct.p, logrus.StandardLogger())
	c.Queue = t.Name()
	c.Tag = ct.Tag
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		ct.done <- c.Consume(ctx)
	}()
	go ct.produce(cancel)
	assert.Equal(t, exp, <-ct.done)
	ct.ch.AssertExpectations(t)
	ct.p.AssertExpectations(t)
	ct.a.AssertExpectations(t)
}

func (ct *consumeTest) produce(cancel func()) {
	defer close(ct.msgs)
	if len(ct.dd) == 0 && ct.cancelCount == 0 {
		cancel()
		return
	}
	for i, d := range ct.dd {
		go func() {
			if i >= ct.cancelCount {
				<-ct.sync
				cancel()
				time.Sleep(time.Second)
				ct.sync <- true
				return
			}
		}()
		ct.msgs <- d
	}
}

var consumeTests = []*consumeTest{
	newConsumeTest(
		"happy path",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		3,
		intMax,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", New(ct.dd[0])).Once().Return(nil)
			ct.p.On("Process", New(ct.dd[1])).Once().Return(nil)
			ct.p.On("Process", New(ct.dd[2])).Once().Return(nil)
			return nil
		},
	),
	newSimpleConsumeTest(
		"consume error",
		"INFO Registering consumer... \n",
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(nil, fmt.Errorf("consume error"))
			return fmt.Errorf("failed to register a consumer: consume error")
		},
	),
	newSimpleConsumeTest(
		"process error",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		func(t *testing.T, ct *consumeTest) error {
			err := fmt.Errorf("process error")
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", New(ct.dd[0])).Once().Return(err)
			return err
		},
	),
	newSimpleConsumeTest(
		"create command error",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\nERROR failed to register a consumer: create command error\n",
		func(t *testing.T, ct *consumeTest) error {
			err := NewCreateCommandError(fmt.Errorf("create command error"))
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", New(ct.dd[0])).Once().Return(err)
			return nil
		},
	),
	newSimpleConsumeTest(
		"ack error",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		func(t *testing.T, ct *consumeTest) error {
			err := NewAcknowledgmentError(fmt.Errorf("ack error"))
			ct.ch.On("Consume", t.Name(), "ctag", false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.p.On("Process", New(ct.dd[0])).Once().Return(err)
			return err
		},
	),
}

func TestConsumer_Consume(t *testing.T) {
	for _, test := range consumeTests {
		t.Run(test.Name, test.Run)
	}
}

func TestConsumer_Consume_NotifyClose(t *testing.T) {
	ch := new(TestChannel)
	d := make(chan amqp.Delivery)
	done := make(chan error)

	ch.On("Consume", "", "", false, false, false, false, nilAmqpTable).Once().Return(d, nil)

	c := ConsumerNew(nil, ch, new(TestProcessor), logrus.StandardLogger())

	go func() {
		done <- c.Consume(context.Background())
	}()

	retry := 5
	for !ch.TriggerNotifyClose("server close") && retry > 0 {
		retry--
		if retry == 0 {
			t.Fatal("No notify handler registered.")
		}
		// When called too early, the close handler is not yet registered. Try again later.
		time.Sleep(time.Millisecond)
	}

	assert.Equal(t, &amqp.Error{Reason: "server close", Code: 320}, <-done)
	ch.AssertExpectations(t)
}

func TestConsumer_Close(t *testing.T) {
	t.Run("no connection", func(t *testing.T) {
		c := ConsumerNew(nil, nil, nil, logrus.StandardLogger())
		assert.Nil(t, c.Close())
	})
	t.Run("with connection", func(t *testing.T) {
		conn := new(TestConnection)
		conn.On("Close").Once().Return(nil)
		c := ConsumerNew(conn, nil, nil, logrus.StandardLogger())
		assert.Nil(t, c.Close())
		conn.AssertExpectations(t)
	})
	t.Run("close error", func(t *testing.T) {
		err := fmt.Errorf("close error")
		conn := new(TestConnection)
		conn.On("Close").Once().Return(err)
		c := ConsumerNew(conn, nil, nil, logrus.StandardLogger())
		assert.Equal(t, err, c.Close())
		conn.AssertExpectations(t)
	})
}

func testConsumerCancel(t *testing.T, err error) {
	done := make(chan error)
	ch := new(TestChannel)
	msgs := make(chan amqp.Delivery)
	ch.On("Consume", "queue", t.Name(), false, false, false, false, nilAmqpTable).Once().Return(msgs, nil)
	ch.On("Cancel", t.Name(), false).Once().Return(err).Run(func(_ mock.Arguments) {
		close(msgs)
	})
	ctx, cancel := context.WithCancel(context.Background())
	c := ConsumerNew(nil, ch, nil, logrus.StandardLogger())
	c.Queue = "queue"
	c.Tag = t.Name()
	go func() {
		done <- c.Consume(ctx)
	}()
	cancel()
	assert.Equal(t, err, <-done)
	ch.AssertExpectations(t)
}

var cancelTests = []*consumeTest{
	newConsumeTest(
		"skip remaining",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		3,
		1,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), ct.Tag, false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.ch.On("Cancel", ct.Tag, false).Return(nil)
			ct.p.On("Process", New(ct.dd[0])).Return(nil).Run(func(_ mock.Arguments) {
				ct.sync <- true
				<-ct.sync
			})
			ct.a.On("Nack", uint64(1), true, true).Return(nil)
			ct.a.On("Nack", uint64(2), true, true).Return(nil)
			return nil
		},
	),
	newConsumeTest(
		"no messages",
		"INFO Registering consumer... \nINFO Succeeded registering consumer.\nINFO Waiting for messages...\n",
		0,
		0,
		func(t *testing.T, ct *consumeTest) error {
			ct.ch.On("Consume", t.Name(), ct.Tag, false, false, false, false, nilAmqpTable).
				Once().
				Return(ct.msgs, nil)
			ct.ch.On("Cancel", ct.Tag, false).Return(nil)
			return nil
		},
	),
}

func TestConsumer_Cancel(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		testConsumerCancel(t, nil)
	})
	t.Run("error", func(t *testing.T) {
		testConsumerCancel(t, fmt.Errorf("cancel error"))
	})
	t.Run("notify no block", func(t *testing.T) {
		ch := make(chan bool)
		go func() {
			testConsumerCancel(t, nil)
			ch <- true
		}()
		select {
		case <-ch:
			// Intentionally left blank.
		case <-time.After(5 * time.Second):
			t.Error("Timeout because notify handler is blocking cancel")
		}
	})
	for _, test := range cancelTests {
		t.Run(test.Name, test.Run)
	}
}

type TestConnection struct {
	Connection
	mock.Mock
}

func (t *TestConnection) Close() error {
	argsT := t.Called()

	return argsT.Error(0)
}

func (t *TestConnection) Channel() (*amqp.Channel, error) {
	argsT := t.Called()

	return argsT.Get(0).(*amqp.Channel), argsT.Error(1)
}

type TestChannel struct {
	Channel
	mock.Mock
	notifyClose chan *amqp.Error
}

func (t *TestChannel) ExchangeDeclare(name, kind string, durable, autoDelete, internal, noWait bool, args amqp.Table) error {
	argsT := t.Called(name, kind, durable, autoDelete, internal, noWait, args)

	return argsT.Error(0)
}

func (t *TestChannel) NotifyClose(c chan *amqp.Error) chan *amqp.Error {
	t.notifyClose = c
	return c
}

func (t *TestChannel) TriggerNotifyClose(reason string) bool {
	if t.notifyClose != nil {
		t.notifyClose <- &amqp.Error{
			Reason: reason,
			Code:   320,
		}
		return true
	}
	return false
}

func (t *TestChannel) QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error) {
	argsT := t.Called(name, durable, autoDelete, exclusive, noWait, args)

	return argsT.Get(0).(amqp.Queue), argsT.Error(1)
}

func (t *TestChannel) Qos(prefetchCount, prefetchSize int, global bool) error {
	argsT := t.Called(prefetchCount, prefetchSize, global)

	return argsT.Error(0)
}

func (t *TestChannel) QueueBind(name, key, exchange string, noWait bool, args amqp.Table) error {
	argsT := t.Called(name, key, exchange, noWait, args)

	return argsT.Error(0)
}

func (t *TestChannel) Close() error {
	argsT := t.Called()

	return argsT.Error(0)
}

func (t *TestChannel) Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error) {
	argsT := t.Called(queue, consumer, autoAck, exclusive, noLocal, noWait, args)

	d, ok := argsT.Get(0).(chan amqp.Delivery)
	if !ok {
		d = nil
	}

	return d, argsT.Error(1)
}

func (t *TestChannel) Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error {
	argsT := t.Called(exchange, key, mandatory, immediate, msg)

	return argsT.Error(0)
}
func (t *TestChannel) Cancel(consumer string, noWait bool) error {
	argsT := t.Called(consumer, noWait)

	return argsT.Error(0)
}

type TestProcessor struct {
	Processor
	mock.Mock
}

func (p *TestProcessor) Process(d Delivery) error {
	return p.Called(d).Error(0)
}

func (p *TestProcessor) Cancel() error {
	return p.Called().Error(0)
}

type TestAmqpAcknowledger struct {
	amqp.Acknowledger
	mock.Mock
}

func (a *TestAmqpAcknowledger) Ack(tag uint64, multiple bool) error {
	argsT := a.Called(tag, multiple)

	return argsT.Error(0)
}

func (a *TestAmqpAcknowledger) Nack(tag uint64, multiple bool, requeue bool) error {
	argsT := a.Called(tag, multiple, requeue)

	return argsT.Error(0)
}

func (a *TestAmqpAcknowledger) Reject(tag uint64, requeue bool) error {
	argsT := a.Called(tag, requeue)

	return argsT.Error(0)
}

const (
	autodeleteExchangeConfig  = "autodelete"
	defaultConfig             = "default"
	durableExchangeConfig     = "durable"
	multipleRoutingKeysConfig = "multiple_routing"
	noRoutingKeyConfig        = "no_routing"
	oneEmptyRoutingKeyConfig  = "empty_routing"
	priorityConfig            = "priority"
	qosConfig                 = "qos"
	routingConfig             = "routing"
	simpleExchangeConfig      = "exchange"
	ttlConfig                 = "ttl"
	autoDeleteQueue           = "autodelete_queue"
	durableQueue              = "durable_queue"
	nonDurableQueue           = "non_durable_queue"
	defaultQueueDurability    = "default_queue_durability"
	exclusiveQueue            = "exclusive_queue"
	noWaitQueue               = "nowait_queue"
)

var nilAmqpTable amqp.Table
var emptyAmqpTable = amqp.Table{}

var queueTests = []struct {
	name   string
	config string
	setup  func(*TestChannel)
	err    error
}{
	// Simple queue.
	{
		"simpleQueue",
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Define queue with TTL.
	{
		"queueWithTTL",
		ttlConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "ttlQueue", true, false, false, false, amqp.Table{"x-message-ttl": int32(1200)}).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Define queue with Priority.
	{
		"queueWithPriority",
		priorityConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "priorityWorker", true, false, false, false, amqp.Table{"x-max-priority": int32(42)}).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "priorityExchange", "priorityType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "priorityWorker", "", "priorityExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue with multiple routing keys.
	{
		"queueWithMultipleRoutingKeys",
		multipleRoutingKeysConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "multiRoutingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "multiRoutingExchange", "multiRoutingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "multiRoutingQueue", "foo", "multiRoutingExchange", false, nilAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "multiRoutingQueue", "bar", "multiRoutingExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue with one emtpy routing key.
	{
		"queueWithOneEmptyRoutingKey",
		oneEmptyRoutingKeyConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "emptyRoutingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "emptyRoutingExchange", "emptyRoutingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "emptyRoutingQueue", "", "emptyRoutingExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Define queue without routing key.
	{
		"queueWithoutRoutingKey",
		noRoutingKeyConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "noRoutingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "noRoutingExchange", "noRoutingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "noRoutingQueue", "", "noRoutingExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Set QoS.
	{
		"setQos",
		qosConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 42, 0, true).Return(nil).Once()
			ch.On("QueueDeclare", "qosQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Set QoS fails.
	{
		"setQosFail",
		qosConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 42, 0, true).Return(fmt.Errorf("QoS error")).Once()
		},
		fmt.Errorf("failed to set QoS: QoS error"),
	},
	// Declare queue fails.
	{
		"declareQueueFail",
		defaultConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, fmt.Errorf("queue error")).Once()
		},
		fmt.Errorf("failed to declare queue: queue error"),
	},
	// Declare exchange.
	{
		"declareExchange",
		simpleExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare durable exchange.
	{
		"declareDurableExchange",
		durableExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", true, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare auto delete exchange.
	{
		"declareAutoDeleteExchange",
		autodeleteExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, true, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "queueName", "", "exchangeName", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Declare exchange fails.
	{
		"declareExchangeFail",
		simpleExchangeConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "queueName", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "exchangeName", "exchangeType", false, false, false, false, emptyAmqpTable).Return(fmt.Errorf("declare exchagne error")).Once()
		},
		fmt.Errorf("failed to declare exchange: declare exchagne error"),
	},
	// Bind queue.
	{
		"bindQueue",
		routingConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "routingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "routingExchange", "routingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "routingQueue", "routingKey", "routingExchange", false, nilAmqpTable).Return(nil).Once()
		},
		nil,
	},
	// Bind queue fails.
	{
		"bindQueueFail",
		routingConfig,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "routingQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
			ch.On("ExchangeDeclare", "routingExchange", "routingType", false, false, false, false, emptyAmqpTable).Return(nil).Once()
			ch.On("QueueBind", "routingQueue", "routingKey", "routingExchange", false, nilAmqpTable).Return(fmt.Errorf("queue bind error")).Once()
		},
		fmt.Errorf("failed to bind queue to exchange: queue bind error"),
	},
	// Durable queue
	{
		"durableQueue",
		durableQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "durableQueue", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Non durable queue
	{
		"nonDurableQueue",
		nonDurableQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "nonDurableQueue", false, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Default queue durability
	{
		"defaultQueueDurability",
		defaultQueueDurability,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "defaultQueueDurability", true, false, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// AutoDelete queue
	{
		"autoDeleteQueue",
		autoDeleteQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "autoDeleteQueue", true, true, false, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Exclusive queue
	{
		"exclusiveQueue",
		exclusiveQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "exclusiveQueue", true, false, true, false, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
	// Nowait queue
	{
		"noWaitQueue",
		noWaitQueue,
		func(ch *TestChannel) {
			ch.On("Qos", 3, 0, false).Return(nil).Once()
			ch.On("QueueDeclare", "noWaitQueue", true, false, false, true, emptyAmqpTable).Return(amqp.Queue{}, nil).Once()
		},
		nil,
	},
}

func TestQueueSettings(t *testing.T) {
	for _, test := range queueTests {
		t.Run(test.name, func(t *testing.T) {
			cfg, _ := LoadAndParse(fmt.Sprintf("fixtures/%s.conf", test.config))
			ch := new(TestChannel)
			test.setup(ch)
			assert.Equal(t, test.err, Setup(cfg, ch, logrus.StandardLogger()))
			ch.AssertExpectations(t)
		})
	}
}
