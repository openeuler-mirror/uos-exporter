package exporter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"sync"
)

var defaultReg *Registry


// TODO: implement
