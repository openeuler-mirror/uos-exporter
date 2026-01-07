package metrics

import (
	"errors"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/gosnmp/gosnmp"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/sirupsen/logrus"
)

var (
	fields = logrus.Fields{"test": "test"}
	logger = logrus.WithFields(fields)
)


// TODO: implement functions
