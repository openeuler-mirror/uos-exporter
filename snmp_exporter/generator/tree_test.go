package main

import (
	"reflect"
	"regexp"
	"testing"

	"snmp_exporter/internal/metrics"

	"github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var (
	fields = logrus.Fields{}
	logger = logrus.WithFields(fields)
)


// TODO: implement functions
