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


// TODO: implement functions
