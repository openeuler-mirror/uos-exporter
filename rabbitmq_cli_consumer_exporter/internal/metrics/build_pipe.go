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
func (b *PipeBuilder) SetLogger(l logr.Logger) {
	b.log = l
}

func (b *PipeBuilder) SetOutputWriter(w io.Writer) {
	b.outputWriter = w
}

func (b *PipeBuilder) SetErrorWriter(w io.Writer) {
	b.errorWriter = w
}

func (b *PipeBuilder) SetCommand(cmd string) {
	var args []string

	if split := strings.Split(cmd, " "); len(split) > 1 {
		cmd, args = split[0], split[1:]
	}

	b.cmd = cmd
	b.args = args
}

func (b *PipeBuilder) SetCaptureOutput(capture bool) {
	b.capture = capture
}

// GetCommand is part of Builder.
func (b *PipeBuilder) GetCommand(p Properties, d Info, body []byte) (*exec.Cmd, error) {
	createMeta := func() ([]byte, error) {
		return json.Marshal(&struct {
			Properties   Properties `json:"properties"`
			DeliveryInfo Info       `json:"delivery_info"`
		}{
			Properties:   p,
			DeliveryInfo: d,
		})
	}

	setupCmd := func(meta []byte, pipe *os.File) (*exec.Cmd, error) {
		// 安全地构建命令，防止命令注入
		cmd := utils.GetCommand(b.cmd)
		cmd.Args = append([]string{b.cmd}, b.args...)
		cmd.Env = os.Environ()
		cmd.Stdin = bytes.NewBuffer(body)
		cmd.ExtraFiles = []*os.File{pipe}

		if b.capture {
			cmd.Stdout = b.outputWriter
			cmd.Stderr = b.errorWriter
		}
		return cmd, nil
	}

	meta, err := createMeta()
	if err != nil {
		return nil, fmt.Errorf("failed to marshall metadata: %w", err)
	}

	r, w, err := os.Pipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create pipe: %w", err)
	}
	defer w.Close()

	if _, err := w.Write(meta); err != nil {
		return nil, fmt.Errorf("failed to write metadata: %w", err)
	}

	return setupCmd(meta, r)
}
