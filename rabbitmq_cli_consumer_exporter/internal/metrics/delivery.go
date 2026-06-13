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

// TODO: implement functions
