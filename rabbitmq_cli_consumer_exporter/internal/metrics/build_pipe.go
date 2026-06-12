package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"rabbitmq_cli_consumer_exporter/pkg/utils"
	"strings"

	"github.com/bketelsen/logr"
)

type PipeBuilder struct {
	Builder
	log          logr.Logger
	outputWriter io.Writer
	errorWriter  io.Writer
	cmd          string
	args         []string
	capture      bool
}

// SetLogger is part of Builder.

// TODO: implement
