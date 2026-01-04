package metrics

import (
	"time"

	"github.com/streadway/amqp"
)

// Info represents the delivery info of an amqp message.
type Info struct {
	MessageCount uint32 `json:"message_count"`
	ConsumerTag  string `json:"consumer_tag"`
	DeliveryTag  uint64 `json:"delivery_tag"`
	Redelivered  bool   `json:"redelivered"`
	Exchange     string `json:"exchange"`
	RoutingKey   string `json:"routing_key"`
}

// NewDeliveryInfo creates a new delivery info struct from the AMQP message.
func NewDeliveryInfo(d amqp.Delivery) Info {
	return Info{
		ConsumerTag:  d.ConsumerTag,
		MessageCount: d.MessageCount,
		DeliveryTag:  d.DeliveryTag,
		Redelivered:  d.Redelivered,
		Exchange:     d.Exchange,
		RoutingKey:   d.RoutingKey,
	}
}

// Properties represents the properties of an AMQP message.
type Properties struct {
	Headers         amqp.Table `json:"application_headers"`
	ContentType     string     `json:"content_type"`
	ContentEncoding string     `json:"content_encoding"`
	DeliveryMode    uint8      `json:"delivery_mode"`
	Priority        uint8      `json:"priority"`
	CorrelationID   string     `json:"correlation_id"`
	ReplyTo         string     `json:"reply_to"`
	Expiration      string     `json:"expiration"`
	MessageID       string     `json:"message_id"`
	Timestamp       time.Time  `json:"timestamp"`
	Type            string     `json:"type"`
	UserID          string     `json:"user_id"`
	AppID           string     `json:"app_id"`
}

// NewProperties creates a new properties struct from the AMQP message.
func NewProperties(d amqp.Delivery) Properties {
	return Properties{
		Headers:         d.Headers,
		ContentType:     d.ContentType,
		ContentEncoding: d.ContentEncoding,
		DeliveryMode:    d.DeliveryMode,
		Priority:        d.Priority,
		CorrelationID:   d.CorrelationId,
		ReplyTo:         d.ReplyTo,
		Expiration:      d.Expiration,
		MessageID:       d.MessageId,
		Timestamp:       d.Timestamp,
		Type:            d.Type,
		AppID:           d.AppId,
		UserID:          d.UserId,
	}
}

// Delivery interface describes interface for messages
type Delivery interface {
	Ack() error
	Nack(requeue bool) error
	Reject(requeue bool) error
	Body() []byte
	Properties() Properties
	Info() Info
}

// New creates a new delivery instance from the given AMQP delivery.
func New(d amqp.Delivery) Delivery {
	return &delivery{d}
}

type delivery struct {
	d amqp.Delivery
}

// Ack acknowledges the message.
func (r delivery) Ack() error {
	return r.d.Ack(true)
}

// Nack negatively acknowledges the message.
func (r delivery) Nack(requeue bool) error {
	return r.d.Nack(true, requeue)
}

// Reject rejects the message.
func (r delivery) Reject(requeue bool) error {
	return r.d.Reject(requeue)
}

// Body returns the message body.
func (r delivery) Body() []byte {
	return r.d.Body
}

// Properties returns the properties struct for the message.
func (r delivery) Properties() Properties {
	return NewProperties(r.d)
}

// Info returns the delivery info struct for the message.
func (r delivery) Info() Info {
	return NewDeliveryInfo(r.d)
}
