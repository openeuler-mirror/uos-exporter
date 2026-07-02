package metrics

import (
	// "flag"
	"time"
	"strings"
	"testing"

	"github.com/hashicorp/consul/api"
)

func TestGetDefault(t *testing.T) {
    tests := []struct {
        name         string
        value        string
        defaultValue string
        expected     string
    }{
        {"non-empty value", "actual", "default", "actual"},
        {"empty value", "", "default", "default"},
        {"both empty", "", "", ""},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            if got := GetDefaultValue(tt.value, tt.defaultValue); got != tt.expected {
                t.Errorf("getDefault() = %v, want %v", got, tt.expected)
            }
        })
    }
}

func TestCreateConsulClient(t *testing.T) {
    tests := []struct {
        name        string
        config      *api.Config
        wantErr     bool
        errContains string
    }{
        {
            name:    "nil config",
            config:  nil,
            wantErr: true,
            errContains: "cannot be nil",
        },
        {
            name:    "valid config",
            config:  &api.Config{Address: "localhost:8500"},
            wantErr: false,
        },
    }

    for _, tt := range tests {
		
		t.Errorf("tt = %v", tt)

        t.Run(tt.name, func(t *testing.T) {
            got, err := createConsulClient(tt.config)
            if (err != nil) != tt.wantErr {
                t.Errorf("createConsulClient() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if tt.wantErr && !strings.Contains(err.Error(), tt.errContains) {
                t.Errorf("createConsulClient() error = %v, should contain %v", err, tt.errContains)
            }
            if !tt.wantErr && got == nil {
                t.Error("createConsulClient() returned nil client on success")
            }
        })
    }
}

func TestInitConsulConfig(t *testing.T) {
    // Test cases
    tests := []struct {
        name        string
        cfg         *Config
        wantErr     bool
        errContains string
    }{
        {
            name:        "nil consul config",
            cfg:         &Config{Consul: nil},
            wantErr:     true,
            errContains: "cannot be nil",
        },
        {
            name: "valid config with defaults",
            cfg: &Config{
                Consul: &ConsulConfig{
                    Address:    "",
                    Datacenter: "",
                    Scheme:     "",
                },
            },
            wantErr: false,
        },
        {
            name: "custom config values",
            cfg: &Config{
                Consul: &ConsulConfig{
                    Address:    "custom-address:8500",
                    Datacenter: "custom-dc",
                    Scheme:     "https",
                    Token:      "test-token",
                },
            },
            wantErr: false,
        },
        {
            name: "empty address after defaults",
            cfg: &Config{
                Consul: &ConsulConfig{
                    Address:    "", // Will be set to default
                },
            },
            wantErr: false, // Because default will be applied
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := initConsulConfig(tt.cfg)
            
            if (err != nil) != tt.wantErr {
                t.Errorf("initConsulConfig() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            
            if tt.wantErr {
                if !strings.Contains(err.Error(), tt.errContains) {
                    t.Errorf("initConsulConfig() error = %v, should contain %v", err, tt.errContains)
                }
                return
            }
            
            // Verify default values when empty
            if tt.cfg.Consul.Address == "" && got.Address != "localhost:8500" {
                t.Errorf("expected default address, got %v", got.Address)
            }
            
            if tt.cfg.Consul.Datacenter == "" && got.Datacenter != "dc1" {
                t.Errorf("expected default datacenter, got %v", got.Datacenter)
            }
            
            if tt.cfg.Consul.Scheme == "" && got.Scheme != "http" {
                t.Errorf("expected default scheme, got %v", got.Scheme)
            }
            
            // Verify transport settings
            if got.Transport == nil {
                t.Error("expected transport configuration, got nil")
            }
        })
    }
}

func TestCreateTransportConfig(t *testing.T) {

    transport := createTransportConfig()
    
    if transport.MaxIdleConns != 10 {
        t.Errorf("expected MaxIdleConns = 10, got %d", transport.MaxIdleConns)
    }
    
    if transport.IdleConnTimeout != 30*time.Second {
        t.Errorf("expected IdleConnTimeout = 30s, got %v", transport.IdleConnTimeout)
    }
    
    if transport.DisableCompression != false {
        t.Error("expected DisableCompression = false")
    }
    
    if transport.TLSHandshakeTimeout != 10*time.Second {
        t.Errorf("expected TLSHandshakeTimeout = 10s, got %v", transport.TLSHandshakeTimeout)
    }
}
