package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"snmp_exporter/internal/metrics"
)

var (
	cannotFindModuleRE = regexp.MustCompile(`Cannot find module \((.+)\): (.+)`)
)


// TODO: implement functions
