package metrics

import (
	"fmt"
	"testing"

	"github.com/magiconair/properties/assert"
	"github.com/streadway/amqp"
	"github.com/stretchr/testify/mock"
)

var ackTests = []struct {
	name   string
	method string
	tag    uint64
	args   []interface{}
	err    error
	call   func(d Delivery) error
}{
	{
		"ack",
		"Ack",
		3,
		[]interface{}{true},
		nil,
		func(d Delivery) error { return d.Ack() },
	},
	{
		"ackError",
		"Ack",
		7,
		[]interface{}{true},
		fmt.Errorf("ack"),
		func(d Delivery) error { return d.Ack() },
	},
	{
		"nack",
		"Nack",
		11,
		[]interface{}{true, false},
		nil,
		func(d Delivery) error { return d.Nack(false) },
	},
	{
		"nackRequeue",
		"Nack",
		17,
		[]interface{}{true, true},
		nil,
		func(d Delivery) error { return d.Nack(true) },
	},
	{
		"nackError",
		"Nack",
		19,
		[]interface{}{true, true},
		fmt.Errorf("nack"),
		func(d Delivery) error { return d.Nack(true) },
	},
	{
		"reject",
		"Reject",
		23,
		[]interface{}{true},
		nil,
		func(d Delivery) error { return d.Reject(true) },
	},
	{
		"rejectRequeue",
		"Reject",
		29,
		[]interface{}{true},
		nil,
		func(d Delivery) error { return d.Reject(true) },
	},
	{
		"rejectError",
		"Reject",
		31,
		[]interface{}{true},
		fmt.Errorf("reject"),
		func(d Delivery) error { return d.Reject(true) },
	},
}


// TODO: implement functions
