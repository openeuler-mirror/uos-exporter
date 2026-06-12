// internal/metrics/client_pool_manager.go
package metrics

import (
	"fmt"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
)

type MongoDBClientPoolManager struct {
	pools map[string]*MongoDBClientPool
	mu    sync.RWMutex
}

func NewMongoDBClientPoolManager() *MongoDBClientPoolManager {
	return &MongoDBClientPoolManager{
		pools: make(map[string]*MongoDBClientPool),
	}
}

func (m *MongoDBClientPoolManager) GetPool(instanceName string) (*MongoDBClientPool, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	pool, ok := m.pools[instanceName]
	return pool, ok
}

func (m *MongoDBClientPoolManager) AddPool(instance MongoDBInstance) {
	m.mu.Lock()
	defer m.mu.Unlock()
	pool := &MongoDBClientPool{
		instance: instance,
	}
	pool.StartHealthCheck(10 * time.Second)
	m.pools[instance.Name] = pool
}

func (m *MongoDBClientPoolManager) GetClient(instanceName string) (*mongo.Client, error) {
	pool, ok := m.GetPool(instanceName)
	if !ok {
		return nil, fmt.Errorf("pool not found for instance: %s", instanceName)
	}
	return pool.GetClient()
}
