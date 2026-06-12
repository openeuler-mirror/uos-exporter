// internal/metrics/connection_pool.go
package metrics

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoDBInstance struct {
	Name          string `yaml:"name"`
	URI           string `yaml:"uri"` // 必填
	Username      string `yaml:"username,omitempty"`
	Password      string `yaml:"password,omitempty"`
	AuthMechanism string `yaml:"auth_mechanism,omitempty"`
	AuthDB        string `yaml:"auth_db,omitempty"`
}

// MongoDBClientPool 管理单个 MongoDB 实例的连接池
type MongoDBClientPool struct {
	client   *mongo.Client
	instance MongoDBInstance
	mu       sync.Mutex
}

func (p *MongoDBClientPool) GetClient() (*mongo.Client, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client == nil || !p.isConnected() {
		err := p.reconnect()
		if err != nil {
			return nil, err
		}
	}
	return p.client, nil
}

// 检查连接是否存活
func (p *MongoDBClientPool) isConnected() bool {
	if p.client == nil {
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return p.client.Ping(ctx, nil) == nil
}

// 建立新连接
func (p *MongoDBClientPool) reconnect() error {
	var clientOptions options.ClientOptions
	if p.instance.Username != "" || p.instance.Password != "" {
		cred := options.Credential{
			Username:      p.instance.Username,
			Password:      p.instance.Password,
			AuthSource:    p.instance.AuthDB,
			AuthMechanism: p.instance.AuthMechanism,
		}
		clientOptions.SetAuth(cred)
	}

	clientOptions.ApplyURI(p.instance.URI)
	client, err := mongo.Connect(context.Background(), &clientOptions)
	if err != nil {
		return err
	}

	if err := client.Ping(context.Background(), nil); err != nil {
		return err
	}

	if p.client != nil {
		_ = p.client.Disconnect(context.Background())
	}
	p.client = client
	return nil
}

// 定期健康检查
func (p *MongoDBClientPool) StartHealthCheck(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				p.mu.Lock()
				if !p.isConnected() {
					logrus.Warnf("MongoDB connection lost for %s. Reconnecting...", p.instance.Name)
					_ = p.reconnect()
				}
				p.mu.Unlock()
			}
		}
	}()
}
