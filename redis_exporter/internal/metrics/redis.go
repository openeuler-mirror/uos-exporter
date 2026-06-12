package metrics

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"redis_exporter/internal/exporter"

	redis "github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

var (
	configFile = flag.String(
		"config.file",
		"/etc/uos-exporter/redis-exporter.yaml",
		"Path to config file",
	)
	defaultAddress = flag.String(
		"redis.address",
		"localhost:6379",
		"Default address for all Redis instances (can be overridden in config)",
	)
	defaultPassword = flag.String(
		"redis.password",
		"",
		"Default password for all Redis instances (can be overridden in config)",
	)
	defaultDB = flag.Int(
		"redis.db",
		0,
		"Default database index for all Redis instances (can be overridden in config)",
	)
)

func init() {
	exporter.Register(
		NewRedisExporter(),
	)
}

type RedisExporter struct {
	infoCollector    *infoCollector
	slowlogCollector *slowlogCollector
	clientCollector  *clientCollector
	keyCollector     *keyCollector
}

func NewRedisExporter() *RedisExporter {
	flag.Parse()

	cfg, err := LoadConfig(
		*configFile,
		*defaultAddress,
		*defaultPassword,
		*defaultDB,
	)
	if err != nil {
		logrus.Fatalf("Error loading config: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisInstances[0].Addr,
		Password: cfg.RedisInstances[0].Password,
		DB:       cfg.RedisInstances[0].DB,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil
	}

	patterns := []string{"user:*", "session:*", "cache:*"}
	return &RedisExporter{
		infoCollector:    newInfoCollector(client),
		slowlogCollector: newSlowlogCollector(client),
		clientCollector:  NewClientCollector(client),
		keyCollector:     NewKeyCollector(client, patterns, 0, 100, true, false),
	}
}

// Collect collects all Redis metrics.
func (e *RedisExporter) Collect(ch chan<- prometheus.Metric) {
	e.infoCollector.Collect(ch)
	e.slowlogCollector.Collect(ch)
	e.clientCollector.Collect(ch)
	e.keyCollector.Collect(ch)
}

type RedisInstance struct {
	Name     string `yaml:"name"`
	Addr     string `yaml:"addr"`
	Password string `yaml:"password,omitempty"`
	DB       int    `yaml:"db,omitempty"`
}

type Config struct {
	RedisInstances []RedisInstance `yaml:"redis_instances"`
}

func LoadConfig(
	configFile string,
	defaultAddress string,
	defaultPassword string,
	defaultDB int,
) (*Config, error) {
	cleanPath := filepath.Clean(configFile)
	if !strings.HasPrefix(cleanPath, "/etc/uos-exporter/") {
		return nil, fmt.Errorf("config path must be under %s", "/etc/uos-exporter/")
	}
	data, err := ioutil.ReadFile(cleanPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return nil, err
	}

	for i := range cfg.RedisInstances {
		instance := &cfg.RedisInstances[i]
		if instance.Addr == "" && defaultAddress != "" {
			instance.Addr = defaultAddress
		}
		if instance.Password == "" && defaultPassword != "" {
			instance.Password = defaultPassword
		}
		if instance.DB == 0 {
			instance.DB = defaultDB
		}
	}

	return &cfg, nil
}
