package metrics

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type TestDelivery struct {
	mock.Mock
}

func (t *TestDelivery) Ack() error {
	argstT := t.Called()

	return argstT.Error(0)
}

func (t *TestDelivery) Nack(requeue bool) error {
	argsT := t.Called(requeue)

	return argsT.Error(0)
}

func (t *TestDelivery) Reject(requeue bool) error {
	argsT := t.Called(requeue)

	return argsT.Error(0)
}

func (t *TestDelivery) Body() []byte {
	argsT := t.Called()

	return argsT.Get(0).([]byte)
}

func (t *TestDelivery) Properties() Properties {
	argsT := t.Called()

	return argsT.Get(0).(Properties)
}

func (t *TestDelivery) Info() Info {
	argsT := t.Called()

	return argsT.Get(0).(Info)
}

var defaultTests = []struct {
	name      string
	onFailure int
	method    string
	args      []interface{}
}{
	{"ack", 0, "Ack", []interface{}{}},
	{"reject", 3, "Reject", []interface{}{false}},
	{"rejectRequeue", 4, "Reject", []interface{}{true}},
	{"nack", 5, "Nack", []interface{}{false}},
	{"nackRequeue", 6, "Nack", []interface{}{true}},
	{"undefined", 0, "Nack", []interface{}{true}},
}

func TestDefault(t *testing.T) {
	for code, test := range defaultTests {
		t.Run(test.name, func(t *testing.T) {
			d := new(TestDelivery)
			d.On(test.method, test.args...).Return(nil)
			a := AcknowledgerNew(false, test.onFailure)
			a.Ack(d, code)
		})
	}
}

var strictTests = []struct {
	name   string
	code   int
	method string
	args   []interface{}
	err    error
}{
	{"ack", 0, "Ack", []interface{}{}, nil},
	{"reject", 3, "Reject", []interface{}{false}, nil},
	{"rejectRequeue", 4, "Reject", []interface{}{true}, nil},
	{"nack", 5, "Nack", []interface{}{false}, nil},
	{"nackRequeue", 6, "Nack", []interface{}{true}, nil},
	{"undefined", 42, "Nack", []interface{}{true}, errors.New("unexpected exit code 42")},
}

func TestStrict(t *testing.T) {
	for _, test := range strictTests {
		t.Run(test.name, func(t *testing.T) {
			d := new(TestDelivery)
			d.On(test.method, test.args...).Return(nil)
			a := AcknowledgerNew(true, 0)
			assert.Equal(t, test.err, a.Ack(d, test.code))
		})
	}
}
