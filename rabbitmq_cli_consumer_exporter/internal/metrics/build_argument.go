package metrics

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"rabbitmq_cli_consumer_exporter/pkg/utils"
	"strings"

	"github.com/bketelsen/logr"
)

type ArgumentBuilder struct {
	Builder
	Compressed   bool
	WithMetadata bool
	log          logr.Logger
	outputWriter io.Writer
	errorWriter  io.Writer
	cmd          string
	args         []string
	capture      bool
}

// SetLogger is part of Builder.
func (b *ArgumentBuilder) SetLogger(l logr.Logger) {
	b.log = l
}

func (b *ArgumentBuilder) SetOutputWriter(w io.Writer) {
	b.outputWriter = w
}

func (b *ArgumentBuilder) SetErrorWriter(w io.Writer) {
	b.errorWriter = w
}

func (b *ArgumentBuilder) SetCommand(cmd string) {
	var args []string

	if split := strings.Split(cmd, " "); len(split) > 1 {
		cmd, args = split[0], split[1:]
	}

	b.cmd = cmd
	b.args = args
}

func (b *ArgumentBuilder) SetCaptureOutput(capture bool) {
	b.capture = capture
}

// func (b *ArgumentBuilder) GetCommand(p Properties, d Info, body []byte) (*exec.Cmd, error) {
// 	var err error
// 	payload := body
// 	if b.WithMetadata {
// 		payload, err = json.Marshal(&struct {
// 			Properties   Properties `json:"properties"`
// 			DeliveryInfo Info       `json:"delivery_info"`
// 			Body         string     `json:"body"`
// 		}{

// 			Properties:   p,
// 			DeliveryInfo: d,
// 			Body:         string(body),
// 		})
// 		if err != nil {
// 			return nil, fmt.Errorf("failed to marshall payload: %v", err)
// 		}
// 	}

// 	buf, err := b.payloadBuffer(payload)
// 	if err != nil {
// 		return nil, err
// 	}

// 	cmd := exec.Command(b.cmd, append(b.args, buf.String())...)
// 	cmd.Env = os.Environ()

// 	if b.capture {
// 		cmd.Stdout = b.outputWriter
// 		cmd.Stderr = b.errorWriter
// 	}

// 	return cmd, nil
// }

func (b *ArgumentBuilder) GetCommand(p Properties, d Info, body []byte) (*exec.Cmd, error) {
	createPayload := func() ([]byte, error) {
		if !b.WithMetadata {
			return body, nil
		}
		return json.Marshal(&struct {
			Properties   Properties `json:"properties"`
			DeliveryInfo Info       `json:"delivery_info"`
			Body         string     `json:"body"`
		}{
			Properties:   p,
			DeliveryInfo: d,
			Body:         string(body),
		})
	}

	setupCmd := func(payload []byte) (*exec.Cmd, error) {
		buf, err := b.payloadBuffer(payload)
		if err != nil {
			return nil, err
		}

		cmd := utils.GetCommand(b.cmd, append(b.args, buf.String())...)
		cmd.Env = os.Environ()

		if b.capture {
			cmd.Stdout = b.outputWriter
			cmd.Stderr = b.errorWriter
		}
		return cmd, nil
	}

	payload, err := createPayload()
	if err != nil {
		return nil, fmt.Errorf("failed to marshall payload: %w", err)
	}

	return setupCmd(payload)
}

func (b *ArgumentBuilder) payloadBuffer(payload []byte) (*bytes.Buffer, error) {
	buf := &bytes.Buffer{}
	enc := base64.NewEncoder(base64.StdEncoding, buf)
	defer enc.Close()

	w, err := b.createWriter(enc)
	if err != nil {
		return nil, err
	}

	if err := b.writePayload(w, payload); err != nil {
		return nil, err
	}

	return buf, nil
}

func (b *ArgumentBuilder) createWriter(enc io.Writer) (io.Writer, error) {
	if !b.Compressed {
		return enc, nil
	}

	comp, err := zlib.NewWriterLevel(enc, zlib.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to create zlib handler: %w", err)
	}
	b.log.Info("Compressed message")
	return comp, nil
}

func (b *ArgumentBuilder) writePayload(w io.Writer, payload []byte) error {
	if _, err := w.Write(payload); err != nil {
		return fmt.Errorf("failed to write payload: %w", err)
	}

	if closer, ok := w.(io.Closer); ok {
		return closer.Close()
	}
	return nil
}
