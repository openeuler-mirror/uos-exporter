package metrics

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	"mongodb_exporter/internal/exporter"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Instances []MongoDBInstance `yaml:"instances"`
}

var (
	configFile = flag.String(
		"config.file",
		"/etc/uos-exporter/mongodb-exporter.yaml",
		"Path to config file",
	)
	defaultURI = flag.String(
		"mongodb.uri",
		"mongodb://localhost:27017",
		"Default URI for all instances (can be overridden in config)",
	)
	defaultUsername = flag.String(
		"mongodb.username",
		"",
		"Default username for all instances (can be overridden in config)",
	)
	defaultPassword = flag.String(
		"mongodb.password",
		"",
		"Default password for all instances (can be overridden in config)",
	)
	defaultAuthDB = flag.String(
		"mongodb.authdb",
		"admin",
		"Authentication database",
	)
	defaultAuthMechanism = flag.String(
		"mongodb.authMechanism",
		"SCRAM-SHA-256",
		"Authentication mechanism",
	)

	// TLS/SSL 参数可选添加
	tlsInsecureSkipVerify = flag.Bool(
		"mongodb.tlsInsecureSkipVerify",
		false,
		"Skip TLS verification for MongoDB connections",
	)
)

func LoadConfig(
	configFile string,
	defaultURI,
	defaultUsername,
	defaultPassword,
	defaultAuthDB,
	defaultAuthMechanism string,
) (*Config, error) {
	cleanPath := filepath.Clean(configFile)
	configDir := "/etc/uos-exporter"
	if !strings.HasPrefix(cleanPath, configDir) {
		return nil, fmt.Errorf("config file must be located within %s", configDir)
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

	for i := range cfg.Instances {
		instance := &cfg.Instances[i]

		// 填充默认值（优先级：YAML > CLI 默认值）
		if instance.URI == "" && defaultURI != "" {
			instance.URI = defaultURI
		}
		if instance.Username == "" && defaultUsername != "" {
			instance.Username = defaultUsername
		}
		if instance.Password == "" && defaultPassword != "" {
			instance.Password = defaultPassword
		}
		if instance.AuthDB == "" && defaultAuthDB != "" {
			instance.AuthDB = defaultAuthDB
		}
		if instance.AuthMechanism == "" && defaultAuthMechanism != "" {
			instance.AuthMechanism = defaultAuthMechanism
		}
	}

	return &cfg, nil
}

func init() {
	exporter.Register(
		NewMongoDBExporter(),
	)
}

type MongoDBExporter struct {
	clientPools                     *MongoDBClientPoolManager
	MongoDBInfocollector            *MongoDBInfocollector
	MongoDBStatuscollector          *MongoDBDBStatscollector
	MongoDBcollectionStatscollector *MongoDBcollectionStatscollector
	MongoDBIndexStatscollector      *MongoDBIndexStatscollector
	MongoDBSlowQuerycollector       *MongoDBSlowQuerycollector
	MongoDBReplSetcollector         *MongoDBReplSetcollector
	MongoDBCachecollector           *MongoDBCachecollector
	MongoDBJumboChunkscollector     *MongoDBJumboChunkscollector
	MongoDBBalancercollector        *MongoDBBalancercollector
	MongoDBMigrationCollector       *MongoDBMigrationCollector
	MongoDBOplogCollector           *MongoDBOplogCollector
	MongoDBShardServerCollector     *MongoDBShardServerCollector
	MongoDBShardCollectionCollector *MongoDBShardCollectionCollector
	MongoDBConfigServerCollector    *MongoDBConfigServerCollector
	MongoDBZoneCollector            *MongoDBZoneCollector
	MongoDBAutoDiscoveryCollector   *MongoDBAutoDiscoveryCollector
	MongoDBMetadataCollector        *MongoDBMetadataCollector
}

// 连接 MongoDB 实例

func connectToMongo(instance MongoDBInstance) (*mongo.Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	clientOptions := options.Client().ApplyURI(instance.URI)

	// 添加认证信息（如果存在）
	if instance.Username != "" || instance.Password != "" {
		cred := options.Credential{
			Username:      instance.Username,
			Password:      instance.Password,
			AuthSource:    instance.AuthDB,
			AuthMechanism: instance.AuthMechanism,
		}
		clientOptions.SetAuth(cred)
	}

	// 可选：TLS 设置
	// clientOptions.SetTLSConfig(...)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logrus.Printf(
			"Failed to connect to MongoDB instance %s at %s: %v",
			instance.Name,
			instance.URI, err,
		)
		return nil, err
	}

	if err := client.Ping(ctx, nil); err != nil {
		logrus.Printf(
			"Ping failed for MongoDB instance %s: %v",
			instance.Name,
			err,
		)
		return nil, err
	}

	return client, nil
}

func NewMongoDBExporter() *MongoDBExporter {
	flag.Parse()

	cfg, err := LoadConfig(
		*configFile,
		*defaultURI,
		*defaultUsername,
		*defaultPassword,
		*defaultAuthDB,
		*defaultAuthMechanism,
	)
	if err != nil {
		logrus.Fatalf("Error loading config: %v", err)
	}
	manager := NewMongoDBClientPoolManager()
	instance := cfg.Instances[0]
	// client, err := connectToMongo(instance)
	// if err != nil {
	// 	logrus.Fatalf("Failed to connect to MongoDB: %v", err)
	// }
	manager.AddPool(instance)
	pool, _ := manager.GetPool(instance.Name)
	m := &MongoDBExporter{
		clientPools: manager,
		MongoDBInfocollector: NewMongoDBInfocollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBStatuscollector: NewMongoDBDBStatscollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBcollectionStatscollector: NewMongoDBcollectionStatscollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBIndexStatscollector: NewMongoDBIndexStatscollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBSlowQuerycollector: NewMongoDBSlowQuerycollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBReplSetcollector: NewMongoDBReplSetcollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBCachecollector: NewMongoDBCachecollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBJumboChunkscollector: NewMongoDBJumboChunkscollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBBalancercollector: NewMongoDBBalancercollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBMigrationCollector: NewMongoDBMigrationCollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBOplogCollector: NewMongoDBOplogCollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBShardServerCollector: NewMongoDBShardServerCollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBShardCollectionCollector: NewMongoDBShardCollectionCollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBConfigServerCollector: NewMongoDBConfigServerCollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBZoneCollector: NewMongoDBZoneCollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBAutoDiscoveryCollector: NewMongoDBAutoDiscoveryCollector(
			pool,
			instance.Name,
			instance.URI,
		),
		MongoDBMetadataCollector: NewMongoDBMetadataCollector(
			pool,
			instance.Name,
			instance.URI,
		),
	}
	return m
}

func (m *MongoDBExporter) Collect(ch chan<- prometheus.Metric) {
	m.MongoDBInfocollector.collect(ch)
	m.MongoDBStatuscollector.collect(ch)
	m.MongoDBcollectionStatscollector.collect(ch)
	m.MongoDBIndexStatscollector.collect(ch)
	m.MongoDBSlowQuerycollector.collect(ch)
	m.MongoDBReplSetcollector.collect(ch)
	m.MongoDBCachecollector.collect(ch)
	m.MongoDBJumboChunkscollector.collect(ch)
	m.MongoDBBalancercollector.collect(ch)
	m.MongoDBMigrationCollector.collect(ch)
	m.MongoDBOplogCollector.collect(ch)
	m.MongoDBShardServerCollector.collect(ch)
	m.MongoDBShardCollectionCollector.collect(ch)
	m.MongoDBConfigServerCollector.collect(ch)
	m.MongoDBZoneCollector.collect(ch)
	m.MongoDBAutoDiscoveryCollector.collect(ch)
	m.MongoDBMetadataCollector.collect(ch)
}
