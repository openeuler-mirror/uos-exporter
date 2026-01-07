package metrics

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/gosnmp/gosnmp"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewGoSNMP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockLogger := logrus.NewEntry(logrus.New())

	tests := []struct {
		name        string
		target      string
		srcAddress  string
		debug       bool
		wantWrapper *GoSNMPWrapper
		wantErr     bool
		expectedErr error
	}{
		{
			name:       "success with default port",
			target:     "192.168.1.1",
			srcAddress: "0.0.0.0",
			debug:      false,
			wantWrapper: &GoSNMPWrapper{
				c: &gosnmp.GoSNMP{
					Transport: "udp",
					Target:    "192.168.1.1",
					Port:      161,
					LocalAddr: "0.0.0.0",
				},
				logger: mockLogger,
			},
			wantErr: false,
		},
		{
			name:       "success with tcp transport",
			target:     "tcp://192.168.1.1",
			srcAddress: "0.0.0.0",
			debug:      false,
			wantWrapper: &GoSNMPWrapper{
				c: &gosnmp.GoSNMP{
					Transport: "tcp",
					Target:    "192.168.1.1",
					Port:      161,
					LocalAddr: "0.0.0.0",
				},
				logger: mockLogger,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewGoSNMP(mockLogger, tt.target, tt.srcAddress, tt.debug)

			if (err != nil) != tt.wantErr {
				t.Errorf("NewGoSNMP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if err.Error() != tt.expectedErr.Error() {
					t.Errorf("NewGoSNMP() error = %v, expectedErr %v", err, tt.expectedErr)
				}
				return
			}

			if got.c.Target != tt.wantWrapper.c.Target ||
				got.c.Port != tt.wantWrapper.c.Port ||
				got.c.Transport != tt.wantWrapper.c.Transport ||
				got.c.LocalAddr != tt.wantWrapper.c.LocalAddr {
				t.Errorf("NewGoSNMP() = %+v, want %+v", got, tt.wantWrapper)
			}
		})
	}
}

func TestParseTransport(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantType string
		wantAddr string
	}{
		{
			name:     "with transport prefix",
			target:   "tcp://127.0.0.1:8080",
			wantType: "tcp",
			wantAddr: "127.0.0.1:8080",
		},
		{
			name:     "without transport prefix",
			target:   "127.0.0.1:8080",
			wantType: "udp",
			wantAddr: "127.0.0.1:8080",
		},
		{
			name:     "empty string",
			target:   "",
			wantType: "udp",
			wantAddr: "",
		},
		{
			name:     "only transport prefix",
			target:   "tcp://",
			wantType: "tcp",
			wantAddr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotType, gotAddr := parseTransport(tt.target)
			if gotType != tt.wantType {
				t.Errorf("parseTransport() gotType = %v, want %v", gotType, tt.wantType)
			}
			if gotAddr != tt.wantAddr {
				t.Errorf("parseTransport() gotAddr = %v, want %v", gotAddr, tt.wantAddr)
			}
		})
	}
}

func TestParsePort(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantPort uint16
		wantErr  bool
	}{
		{
			name:     "valid port",
			target:   "example.com:8080",
			wantPort: 8080,
			wantErr:  false,
		},
		{
			name:     "default port when no port specified",
			target:   "example.com",
			wantPort: 161,
			wantErr:  false,
		},
		{
			name:     "invalid port format",
			target:   "example.com:abc",
			wantPort: 0,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPort, err := parsePort(tt.target)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantPort, gotPort)
		})
	}
}

func TestCreateGoSNMP(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name       string
		transport  string
		target     string
		port       uint16
		srcAddress string
		logger     *logrus.Entry
		debug      bool
	}{
		{
			name:       "basic case without debug",
			transport:  "udp",
			target:     "127.0.0.1",
			port:       161,
			srcAddress: "",
			logger:     nil,
			debug:      false,
		},
		{
			name:       "case with debug enabled",
			transport:  "tcp",
			target:     "localhost",
			port:       162,
			srcAddress: "0.0.0.0",
			logger:     logrus.NewEntry(logrus.New()),
			debug:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := createGoSNMP(tt.transport, tt.target, tt.port, tt.srcAddress, tt.logger, tt.debug)

			if got.Transport != tt.transport {
				t.Errorf("Transport = %v, want %v", got.Transport, tt.transport)
			}
			if got.Target != tt.target {
				t.Errorf("Target = %v, want %v", got.Target, tt.target)
			}
			if got.Port != tt.port {
				t.Errorf("Port = %v, want %v", got.Port, tt.port)
			}
			if got.LocalAddr != tt.srcAddress {
				t.Errorf("LocalAddr = %v, want %v", got.LocalAddr, tt.srcAddress)
			}
		})
	}
}
