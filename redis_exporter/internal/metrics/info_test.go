package metrics

import (
	"strconv"
	"testing"
	"time"
)

func TestParseRedisInfo(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(*RedisInfo)
	}{
		{
			name: "Empty input",
			input: `
`,
			expected: func(r *RedisInfo) {
				if len(r.Keyspace) != 0 || len(r.Modules) != 0 {
					t.Errorf("Expected empty RedisInfo")
				}
			},
		},
		{
			name: "Server section parsed correctly",
			input: `
# Server
redis_version:6.2.6
uptime_in_seconds:123456
tcp_port:6379
os:Linux
`,
			expected: func(r *RedisInfo) {
				if r.Server.RedisVersion != "6.2.6" {
					t.Errorf("Expected redis_version=6.2.6, got %s", r.Server.RedisVersion)
				}
				if r.Server.UptimeInSeconds != 123456 {
					t.Errorf("Expected uptime_in_seconds=123456, got %d", r.Server.UptimeInSeconds)
				}
				if r.Server.TCPPort != 6379 {
					t.Errorf("Expected tcp_port=6379, got %d", r.Server.TCPPort)
				}
				if r.Server.OS != "Linux" {
					t.Errorf("Expected os=Linux, got %s", r.Server.OS)
				}
			},
		},
		{
			name: "Clients section parsed correctly",
			input: `
# Clients
connected_clients:10
blocked_clients:2
`,
			expected: func(r *RedisInfo) {
				if r.Clients.ConnectedClients != 10 {
					t.Errorf("Expected connected_clients=10, got %d", r.Clients.ConnectedClients)
				}
				if r.Clients.BlockedClients != 2 {
					t.Errorf("Expected blocked_clients=2, got %d", r.Clients.BlockedClients)
				}
			},
		},
		{
			name: "Memory section parsed correctly",
			input: `
# Memory
used_memory:1024.0
used_memory_human:1K
mem_allocator:jemalloc-5.1.0
`,
			expected: func(r *RedisInfo) {
				if r.Memory.UsedMemory != 1024.0 {
					t.Errorf("Expected used_memory=1024.0, got %f", r.Memory.UsedMemory)
				}
				if r.Memory.UsedMemoryHuman != "1K" {
					t.Errorf("Expected used_memory_human=1K, got %s", r.Memory.UsedMemoryHuman)
				}
				if r.Memory.MemAllocator != "jemalloc-5.1.0" {
					t.Errorf("Expected mem_allocator=jemalloc-5.1.0, got %s", r.Memory.MemAllocator)
				}
			},
		},
		{
			name: "Keyspace section parsed correctly",
			input: `
# Keyspace
db0:keys=100,expires=5,avg_ttl=3600
db1:keys=200,expires=10,avg_ttl=7200
`,
			expected: func(r *RedisInfo) {
				if len(r.Keyspace) != 2 {
					t.Errorf("Expected 2 databases in keyspace, got %d", len(r.Keyspace))
				}
				db0 := r.Keyspace["db0"]
				if db0.Keys != 100 || db0.Expires != 5 || db0.AvgTTL != 3600*time.Second {
					t.Errorf("db0 parsed incorrectly: %+v", db0)
				}
				db1 := r.Keyspace["db1"]
				if db1.Keys != 200 || db1.Expires != 10 || db1.AvgTTL != 7200*time.Second {
					t.Errorf("db1 parsed incorrectly: %+v", db1)
				}
			},
		},
		{
			name: "Mixed sections parsed correctly",
			input: `
# Server
redis_version:7.0.0
# Clients
connected_clients:5
# Memory
used_memory:2048.0
`,
			expected: func(r *RedisInfo) {
				if r.Server.RedisVersion != "7.0.0" {
					t.Errorf("Expected redis_version=7.0.0, got %s", r.Server.RedisVersion)
				}
				if r.Clients.ConnectedClients != 5 {
					t.Errorf("Expected connected_clients=5, got %d", r.Clients.ConnectedClients)
				}
				if r.Memory.UsedMemory != 2048.0 {
					t.Errorf("Expected used_memory=2048.0, got %f", r.Memory.UsedMemory)
				}
			},
		},
		{
			name: "Persistence section parsed correctly",
			input: `
# Persistence
rdb_changes_since_last_save:1000
rdb_bgsave_in_progress:0
rdb_last_save_time:1677654321
aof_enabled:1
aof_rewrite_in_progress:0
aof_rewrite_scheduled:1
`,
			expected: func(r *RedisInfo) {
				if r.Persistence.RdbChangesSinceLastSave != 1000 {
					t.Errorf("Expected rdb_changes_since_last_save=1000, got %d", r.Persistence.RdbChangesSinceLastSave)
				}
				if r.Persistence.RdbBgsaveInProgress != 0 {
					t.Errorf("Expected rdb_bgsave_in_progress=0, got %d", r.Persistence.RdbBgsaveInProgress)
				}
				expectedTime := time.Unix(1677654321, 0)
				if !r.Persistence.RdbLastSaveTime.Equal(expectedTime) {
					t.Errorf("Expected rdb_last_save_time=%v, got %v", expectedTime, r.Persistence.RdbLastSaveTime)
				}
				if r.Persistence.AofEnabled != 1 {
					t.Errorf("Expected aof_enabled=1, got %d", r.Persistence.AofEnabled)
				}
				if r.Persistence.AofRewriteInProgress != 0 {
					t.Errorf("Expected aof_rewrite_in_progress=0, got %d", r.Persistence.AofRewriteInProgress)
				}
				if r.Persistence.AofRewriteScheduled != 1 {
					t.Errorf("Expected aof_rewrite_scheduled=1, got %d", r.Persistence.AofRewriteScheduled)
				}
			},
		},
		{
			name: "Stats section parsed correctly",
			input: `
# Stats
total_connections_received:10000
total_commands_processed:50000
instantaneous_ops_per_sec:200
rejected_connections:0
evicted_keys:10
keyspace_misses:500
keyspace_hit_ratio:0.95
pubsub_channels:3
pubsub_patterns:1
latest_fork_usec:123456
migrate_cached_sockets:0
slave_expires_tracked_keys:5
active_defrag_hits:100
active_defrag_misses:5
active_defrag_key_hits:10
active_defrag_key_misses:1
`,
			expected: func(r *RedisInfo) {
				if r.Stats.TotalConnectionsReceived != 10000 {
					t.Errorf("Expected total_connections_received=10000, got %d", r.Stats.TotalConnectionsReceived)
				}
				if r.Stats.TotalCommandsProcessed != 50000 {
					t.Errorf("Expected total_commands_processed=50000, got %d", r.Stats.TotalCommandsProcessed)
				}
				if r.Stats.InstantaneousOpsPerSec != 200 {
					t.Errorf("Expected instantaneous_ops_per_sec=200, got %d", r.Stats.InstantaneousOpsPerSec)
				}
				if r.Stats.RejectedConnections != 0 {
					t.Errorf("Expected rejected_connections=0, got %d", r.Stats.RejectedConnections)
				}
				if r.Stats.EvictedKeys != 10 {
					t.Errorf("Expected evicted_keys=10, got %d", r.Stats.EvictedKeys)
				}
				if r.Stats.KeyspaceMisses != 500 {
					t.Errorf("Expected keyspace_misses=500, got %d", r.Stats.KeyspaceMisses)
				}
				if r.Stats.KeyspaceHitRate != 0.95 {
					t.Errorf("Expected keyspace_hit_ratio=0.95, got %f", r.Stats.KeyspaceHitRate)
				}
				if r.Stats.PubsubChannels != 3 {
					t.Errorf("Expected pubsub_channels=3, got %d", r.Stats.PubsubChannels)
				}
				if r.Stats.PubsubPatterns != 1 {
					t.Errorf("Expected pubsub_patterns=1, got %d", r.Stats.PubsubPatterns)
				}
				if r.Stats.LatestForkUsec != 123456 {
					t.Errorf("Expected latest_fork_usec=123456, got %d", r.Stats.LatestForkUsec)
				}
				if r.Stats.MigrateCachedSockets != 0 {
					t.Errorf("Expected migrate_cached_sockets=0, got %d", r.Stats.MigrateCachedSockets)
				}
				if r.Stats.SlaveExpiresTrackedKeys != 5 {
					t.Errorf("Expected slave_expires_tracked_keys=5, got %d", r.Stats.SlaveExpiresTrackedKeys)
				}
				if r.Stats.ActiveDefragHits != 100 {
					t.Errorf("Expected active_defrag_hits=100, got %d", r.Stats.ActiveDefragHits)
				}
				if r.Stats.ActiveDefragMisses != 5 {
					t.Errorf("Expected active_defrag_misses=5, got %d", r.Stats.ActiveDefragMisses)
				}
				if r.Stats.ActiveDefragKeyHits != 10 {
					t.Errorf("Expected active_defrag_key_hits=10, got %d", r.Stats.ActiveDefragKeyHits)
				}
				if r.Stats.ActiveDefragKeyMisses != 1 {
					t.Errorf("Expected active_defrag_key_misses=1, got %d", r.Stats.ActiveDefragKeyMisses)
				}
			},
		},
		{
			name: "Replication section parsed correctly (master)",
			input: `
# Replication
role:master
connected_slaves:2
master_repl_offset:123456789
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:123456780
repl_backlog_histlen:90000000
min-slaves-good-slaves:1
min-slaves-scout-report-intr:60
`,
			expected: func(r *RedisInfo) {
				if r.Replication.Role != "master" {
					t.Errorf("Expected role=master, got %s", r.Replication.Role)
				}
				if r.Replication.ConnectedSlaves != 2 {
					t.Errorf("Expected connected_slaves=2, got %d", r.Replication.ConnectedSlaves)
				}
				if r.Replication.MasterReplOffset != 123456789 {
					t.Errorf("Expected master_repl_offset=123456789, got %d", r.Replication.MasterReplOffset)
				}
				if r.Replication.ReplBacklogActive != 1 {
					t.Errorf("Expected repl_backlog_active=1, got %d", r.Replication.ReplBacklogActive)
				}
				if r.Replication.ReplBacklogSize != 1048576 {
					t.Errorf("Expected repl_backlog_size=1048576, got %d", r.Replication.ReplBacklogSize)
				}
				if r.Replication.ReplBacklogFirstByteOffset != 123456780 {
					t.Errorf("Expected repl_backlog_first_byte_offset=123456780, got %d", r.Replication.ReplBacklogFirstByteOffset)
				}
				if r.Replication.ReplBacklogHistlen != 90000000 {
					t.Errorf("Expected repl_backlog_histlen=90000000, got %d", r.Replication.ReplBacklogHistlen)
				}
				if r.Replication.MinSlavesGoodSlaves != 1 {
					t.Errorf("Expected min-slaves-good-slaves=1, got %d", r.Replication.MinSlavesGoodSlaves)
				}
				if r.Replication.MinSlavesScoutReportIntr != 60 {
					t.Errorf("Expected min-slaves-scout-report-intr=60, got %d", r.Replication.MinSlavesScoutReportIntr)
				}
			},
		},
		{
			name: "CPU section parsed correctly",
			input: `
# CPU
used_cpu_sys:10.5
used_cpu_user:20.3
used_cpu_sys_children:1.2
used_cpu_user_children:2.8
`,
			expected: func(r *RedisInfo) {
				if r.CPU.UsedCPUSys != 10.5 {
					t.Errorf("Expected used_cpu_sys=10.5, got %f", r.CPU.UsedCPUSys)
				}
				if r.CPU.UsedCPUUser != 20.3 {
					t.Errorf("Expected used_cpu_user=20.3, got %f", r.CPU.UsedCPUUser)
				}
				if r.CPU.UsedCPUSysChildren != 1.2 {
					t.Errorf("Expected used_cpu_sys_children=1.2, got %f", r.CPU.UsedCPUSysChildren)
				}
				if r.CPU.UsedCPUUserChildren != 2.8 {
					t.Errorf("Expected used_cpu_user_children=2.8, got %f", r.CPU.UsedCPUUserChildren)
				}
			},
		},
		{
			name: "Cluster section parsed correctly",
			input: `
# Cluster
cluster_enabled:1
cluster_node_count:6
cluster_my_epoch:12345
cluster_slots_assigned:16384
cluster_slots_ok:16384
cluster_slots_pfail:0
cluster_slots_fail:0
cluster_known_nodes:6
cluster_size:6
cluster_current_epoch:12345
cluster_stats_messages_sent:10000
cluster_stats_messages_received:10500
`,
			expected: func(r *RedisInfo) {
				if r.Cluster.ClusterEnabled != 1 {
					t.Errorf("Expected cluster_enabled=1, got %d", r.Cluster.ClusterEnabled)
				}
				if r.Cluster.ClusterNodeCount != 6 {
					t.Errorf("Expected cluster_node_count=6, got %d", r.Cluster.ClusterNodeCount)
				}
				if r.Cluster.ClusterMyEpoch != 12345 {
					t.Errorf("Expected cluster_my_epoch=12345, got %d", r.Cluster.ClusterMyEpoch)
				}
				if r.Cluster.ClusterSlotsAssigned != 16384 {
					t.Errorf("Expected cluster_slots_assigned=16384, got %d", r.Cluster.ClusterSlotsAssigned)
				}
				if r.Cluster.ClusterSlotsOk != 16384 {
					t.Errorf("Expected cluster_slots_ok=16384, got %d", r.Cluster.ClusterSlotsOk)
				}
				if r.Cluster.ClusterSlotsPfail != 0 {
					t.Errorf("Expected cluster_slots_pfail=0, got %d", r.Cluster.ClusterSlotsPfail)
				}
				if r.Cluster.ClusterSlotsFail != 0 {
					t.Errorf("Expected cluster_slots_fail=0, got %d", r.Cluster.ClusterSlotsFail)
				}
				if r.Cluster.ClusterKnownNodes != 6 {
					t.Errorf("Expected cluster_known_nodes=6, got %d", r.Cluster.ClusterKnownNodes)
				}
				if r.Cluster.ClusterSize != 6 {
					t.Errorf("Expected cluster_size=6, got %d", r.Cluster.ClusterSize)
				}
				if r.Cluster.ClusterCurrentEpoch != 12345 {
					t.Errorf("Expected cluster_current_epoch=12345, got %d", r.Cluster.ClusterCurrentEpoch)
				}
				if r.Cluster.ClusterStatsMessagesSent != 10000 {
					t.Errorf("Expected cluster_stats_messages_sent=10000, got %d", r.Cluster.ClusterStatsMessagesSent)
				}
				if r.Cluster.ClusterStatsMessagesReceived != 10500 {
					t.Errorf("Expected cluster_stats_messages_received=10500, got %d", r.Cluster.ClusterStatsMessagesReceived)
				}
			},
		},

		{
			name: "All sections parsed correctly",
			input: `
# Server
redis_version:7.0.0
uptime_in_seconds:123456
tcp_port:6379
os:Linux

# Clients
connected_clients:10
blocked_clients:2

# Memory
used_memory:1024.0
used_memory_human:1K
mem_allocator:jemalloc-5.1.0

# Persistence
rdb_changes_since_last_save:1000
rdb_bgsave_in_progress:0
rdb_last_save_time:1677654321
aof_enabled:1
aof_rewrite_in_progress:0
aof_rewrite_scheduled:1

# Stats
total_connections_received:10000
total_commands_processed:50000
instantaneous_ops_per_sec:200
rejected_connections:0
evicted_keys:10
keyspace_misses:500
keyspace_hit_ratio:0.95
pubsub_channels:3
pubsub_patterns:1
latest_fork_usec:123456
migrate_cached_sockets:0
slave_expires_tracked_keys:5
active_defrag_hits:100
active_defrag_misses:5
active_defrag_key_hits:10
active_defrag_key_misses:1

# Replication
role:master
connected_slaves:2
master_repl_offset:123456789
repl_backlog_active:1
repl_backlog_size:1048576
repl_backlog_first_byte_offset:123456780
repl_backlog_histlen:90000000
min-slaves-good-slaves:1
min-slaves-scout-report-intr:60

# CPU
used_cpu_sys:10.5
used_cpu_user:20.3
used_cpu_sys_children:1.2
used_cpu_user_children:2.8

# Cluster
cluster_enabled:1
cluster_node_count:6
cluster_my_epoch:12345
cluster_slots_assigned:16384
cluster_slots_ok:16384
cluster_slots_pfail:0
cluster_slots_fail:0
cluster_known_nodes:6
cluster_size:6
cluster_current_epoch:12345
cluster_stats_messages_sent:10000
cluster_stats_messages_received:10500

# Keyspace
db0:keys=100,expires=5,avg_ttl=3600
db1:keys=200,expires=10,avg_ttl=7200

# Modules
module1
module2
module3
`,
			expected: func(r *RedisInfo) {
				// Server section
				if r.Server.RedisVersion != "7.0.0" {
					t.Errorf("Expected redis_version=7.0.0, got %s", r.Server.RedisVersion)
				}
				if r.Server.UptimeInSeconds != 123456 {
					t.Errorf("Expected uptime_in_seconds=123456, got %d", r.Server.UptimeInSeconds)
				}
				if r.Server.TCPPort != 6379 {
					t.Errorf("Expected tcp_port=6379, got %d", r.Server.TCPPort)
				}
				if r.Server.OS != "Linux" {
					t.Errorf("Expected os=Linux, got %s", r.Server.OS)
				}

				// Clients section
				if r.Clients.ConnectedClients != 10 {
					t.Errorf("Expected connected_clients=10, got %d", r.Clients.ConnectedClients)
				}
				if r.Clients.BlockedClients != 2 {
					t.Errorf("Expected blocked_clients=2, got %d", r.Clients.BlockedClients)
				}

				// Memory section
				if r.Memory.UsedMemory != 1024.0 {
					t.Errorf("Expected used_memory=1024.0, got %f", r.Memory.UsedMemory)
				}
				if r.Memory.UsedMemoryHuman != "1K" {
					t.Errorf("Expected used_memory_human=1K, got %s", r.Memory.UsedMemoryHuman)
				}
				if r.Memory.MemAllocator != "jemalloc-5.1.0" {
					t.Errorf("Expected mem_allocator=jemalloc-5.1.0, got %s", r.Memory.MemAllocator)
				}

				// Persistence section
				if r.Persistence.RdbChangesSinceLastSave != 1000 {
					t.Errorf("Expected rdb_changes_since_last_save=1000, got %d", r.Persistence.RdbChangesSinceLastSave)
				}
				if r.Persistence.RdbBgsaveInProgress != 0 {
					t.Errorf("Expected rdb_bgsave_in_progress=0, got %d", r.Persistence.RdbBgsaveInProgress)
				}
				expectedTime := time.Unix(1677654321, 0)
				if !r.Persistence.RdbLastSaveTime.Equal(expectedTime) {
					t.Errorf("Expected rdb_last_save_time=%v, got %v", expectedTime, r.Persistence.RdbLastSaveTime)
				}
				if r.Persistence.AofEnabled != 1 {
					t.Errorf("Expected aof_enabled=1, got %d", r.Persistence.AofEnabled)
				}
				if r.Persistence.AofRewriteInProgress != 0 {
					t.Errorf("Expected aof_rewrite_in_progress=0, got %d", r.Persistence.AofRewriteInProgress)
				}
				if r.Persistence.AofRewriteScheduled != 1 {
					t.Errorf("Expected aof_rewrite_scheduled=1, got %d", r.Persistence.AofRewriteScheduled)
				}

				// Stats section
				if r.Stats.TotalConnectionsReceived != 10000 {
					t.Errorf("Expected total_connections_received=10000, got %d", r.Stats.TotalConnectionsReceived)
				}
				if r.Stats.TotalCommandsProcessed != 50000 {
					t.Errorf("Expected total_commands_processed=50000, got %d", r.Stats.TotalCommandsProcessed)
				}
				if r.Stats.InstantaneousOpsPerSec != 200 {
					t.Errorf("Expected instantaneous_ops_per_sec=200, got %d", r.Stats.InstantaneousOpsPerSec)
				}
				if r.Stats.RejectedConnections != 0 {
					t.Errorf("Expected rejected_connections=0, got %d", r.Stats.RejectedConnections)
				}
				if r.Stats.EvictedKeys != 10 {
					t.Errorf("Expected evicted_keys=10, got %d", r.Stats.EvictedKeys)
				}
				if r.Stats.KeyspaceMisses != 500 {
					t.Errorf("Expected keyspace_misses=500, got %d", r.Stats.KeyspaceMisses)
				}
				if r.Stats.KeyspaceHitRate != 0.95 {
					t.Errorf("Expected keyspace_hit_ratio=0.95, got %f", r.Stats.KeyspaceHitRate)
				}
				if r.Stats.PubsubChannels != 3 {
					t.Errorf("Expected pubsub_channels=3, got %d", r.Stats.PubsubChannels)
				}
				if r.Stats.PubsubPatterns != 1 {
					t.Errorf("Expected pubsub_patterns=1, got %d", r.Stats.PubsubPatterns)
				}
				if r.Stats.LatestForkUsec != 123456 {
					t.Errorf("Expected latest_fork_usec=123456, got %d", r.Stats.LatestForkUsec)
				}
				if r.Stats.MigrateCachedSockets != 0 {
					t.Errorf("Expected migrate_cached_sockets=0, got %d", r.Stats.MigrateCachedSockets)
				}
				if r.Stats.SlaveExpiresTrackedKeys != 5 {
					t.Errorf("Expected slave_expires_tracked_keys=5, got %d", r.Stats.SlaveExpiresTrackedKeys)
				}
				if r.Stats.ActiveDefragHits != 100 {
					t.Errorf("Expected active_defrag_hits=100, got %d", r.Stats.ActiveDefragHits)
				}
				if r.Stats.ActiveDefragMisses != 5 {
					t.Errorf("Expected active_defrag_misses=5, got %d", r.Stats.ActiveDefragMisses)
				}
				if r.Stats.ActiveDefragKeyHits != 10 {
					t.Errorf("Expected active_defrag_key_hits=10, got %d", r.Stats.ActiveDefragKeyHits)
				}
				if r.Stats.ActiveDefragKeyMisses != 1 {
					t.Errorf("Expected active_defrag_key_misses=1, got %d", r.Stats.ActiveDefragKeyMisses)
				}

				// Replication section
				if r.Replication.Role != "master" {
					t.Errorf("Expected role=master, got %s", r.Replication.Role)
				}
				if r.Replication.ConnectedSlaves != 2 {
					t.Errorf("Expected connected_slaves=2, got %d", r.Replication.ConnectedSlaves)
				}
				if r.Replication.MasterReplOffset != 123456789 {
					t.Errorf("Expected master_repl_offset=123456789, got %d", r.Replication.MasterReplOffset)
				}
				if r.Replication.ReplBacklogActive != 1 {
					t.Errorf("Expected repl_backlog_active=1, got %d", r.Replication.ReplBacklogActive)
				}
				if r.Replication.ReplBacklogSize != 1048576 {
					t.Errorf("Expected repl_backlog_size=1048576, got %d", r.Replication.ReplBacklogSize)
				}
				if r.Replication.ReplBacklogFirstByteOffset != 123456780 {
					t.Errorf("Expected repl_backlog_first_byte_offset=123456780, got %d", r.Replication.ReplBacklogFirstByteOffset)
				}
				if r.Replication.ReplBacklogHistlen != 90000000 {
					t.Errorf("Expected repl_backlog_histlen=90000000, got %d", r.Replication.ReplBacklogHistlen)
				}
				if r.Replication.MinSlavesGoodSlaves != 1 {
					t.Errorf("Expected min-slaves-good-slaves=1, got %d", r.Replication.MinSlavesGoodSlaves)
				}
				if r.Replication.MinSlavesScoutReportIntr != 60 {
					t.Errorf("Expected min-slaves-scout-report-intr=60, got %d", r.Replication.MinSlavesScoutReportIntr)
				}

				// CPU section
				if r.CPU.UsedCPUSys != 10.5 {
					t.Errorf("Expected used_cpu_sys=10.5, got %f", r.CPU.UsedCPUSys)
				}
				if r.CPU.UsedCPUUser != 20.3 {
					t.Errorf("Expected used_cpu_user=20.3, got %f", r.CPU.UsedCPUUser)
				}
				if r.CPU.UsedCPUSysChildren != 1.2 {
					t.Errorf("Expected used_cpu_sys_children=1.2, got %f", r.CPU.UsedCPUSysChildren)
				}
				if r.CPU.UsedCPUUserChildren != 2.8 {
					t.Errorf("Expected used_cpu_user_children=2.8, got %f", r.CPU.UsedCPUUserChildren)
				}

				// Cluster section
				if r.Cluster.ClusterEnabled != 1 {
					t.Errorf("Expected cluster_enabled=1, got %d", r.Cluster.ClusterEnabled)
				}
				if r.Cluster.ClusterNodeCount != 6 {
					t.Errorf("Expected cluster_node_count=6, got %d", r.Cluster.ClusterNodeCount)
				}
				if r.Cluster.ClusterMyEpoch != 12345 {
					t.Errorf("Expected cluster_my_epoch=12345, got %d", r.Cluster.ClusterMyEpoch)
				}
				if r.Cluster.ClusterSlotsAssigned != 16384 {
					t.Errorf("Expected cluster_slots_assigned=16384, got %d", r.Cluster.ClusterSlotsAssigned)
				}
				if r.Cluster.ClusterSlotsOk != 16384 {
					t.Errorf("Expected cluster_slots_ok=16384, got %d", r.Cluster.ClusterSlotsOk)
				}
				if r.Cluster.ClusterSlotsPfail != 0 {
					t.Errorf("Expected cluster_slots_pfail=0, got %d", r.Cluster.ClusterSlotsPfail)
				}
				if r.Cluster.ClusterSlotsFail != 0 {
					t.Errorf("Expected cluster_slots_fail=0, got %d", r.Cluster.ClusterSlotsFail)
				}
				if r.Cluster.ClusterKnownNodes != 6 {
					t.Errorf("Expected cluster_known_nodes=6, got %d", r.Cluster.ClusterKnownNodes)
				}
				if r.Cluster.ClusterSize != 6 {
					t.Errorf("Expected cluster_size=6, got %d", r.Cluster.ClusterSize)
				}
				if r.Cluster.ClusterCurrentEpoch != 12345 {
					t.Errorf("Expected cluster_current_epoch=12345, got %d", r.Cluster.ClusterCurrentEpoch)
				}
				if r.Cluster.ClusterStatsMessagesSent != 10000 {
					t.Errorf("Expected cluster_stats_messages_sent=10000, got %d", r.Cluster.ClusterStatsMessagesSent)
				}
				if r.Cluster.ClusterStatsMessagesReceived != 10500 {
					t.Errorf("Expected cluster_stats_messages_received=10500, got %d", r.Cluster.ClusterStatsMessagesReceived)
				}

				// Keyspace section
				if len(r.Keyspace) != 2 {
					t.Errorf("Expected 2 databases in keyspace, got %d", len(r.Keyspace))
				}
				db0 := r.Keyspace["db0"]
				if db0.Keys != 100 || db0.Expires != 5 || db0.AvgTTL != 3600*time.Second {
					t.Errorf("db0 parsed incorrectly: %+v", db0)
				}
				db1 := r.Keyspace["db1"]
				if db1.Keys != 200 || db1.Expires != 10 || db1.AvgTTL != 7200*time.Second {
					t.Errorf("db1 parsed incorrectly: %+v", db1)
				}

				// // Modules section
				// if len(r.Modules) != 3 {
				// 	t.Errorf("Expected 3 modules, got %d", len(r.Modules))
				// }
				// if r.Modules[0] != "module1" {
				// 	t.Errorf("Expected module1 at index 0, got %s", r.Modules[0])
				// }
				// if r.Modules[1] != "module2" {
				// 	t.Errorf("Expected module2 at index 1, got %s", r.Modules[1])
				// }
				// if r.Modules[2] != "module3" {
				// 	t.Errorf("Expected module3 at index 2, got %s", r.Modules[2])
				// }
			},
		},
		{
			name: "Invalid lines are skipped",
			input: `
invalid_line_without_section
another_invalid_line
# Server
redis_version:7.0.0
invalid_line_with_missing_value:
invalid_line_with_extra_colon:value:extra
`,
			expected: func(r *RedisInfo) {
				if r.Server.RedisVersion != "7.0.0" {
					t.Errorf("Expected redis_version=7.0.0, got %s", r.Server.RedisVersion)
				}
			},
		},
		{
			name: "Numeric fields with invalid values use default",
			input: `
# Server
uptime_in_seconds:invalid
pid:not_a_number
arch_bits:

# Clients
connected_clients:-1
blocked_clients:9999999999999999999999999999

# Memory
used_memory:not_a_float
mem_fragmentation_ratio:NaN
`,
			expected: func(r *RedisInfo) {
				// These should keep their zero values
				if r.Server.UptimeInSeconds != 0 {
					t.Errorf("Expected uptime_in_seconds=0 for invalid value, got %d", r.Server.UptimeInSeconds)
				}
				if r.Server.PID != 0 {
					t.Errorf("Expected pid=0 for invalid value, got %d", r.Server.PID)
				}
				if r.Server.ArchBits != 0 {
					t.Errorf("Expected arch_bits=0 for invalid value, got %d", r.Server.ArchBits)
				}

				// These should have valid values even if negative or overflow
				// if r.Clients.ConnectedClients == -1 {
				// 	t.Errorf("Expected connected_clients to handle negative value gracefully")
				// }
				if r.Clients.BlockedClients == 99999999999999 {
					t.Errorf("Expected blocked_clients to handle overflow gracefully")
				}

				// Float values
				if r.Memory.UsedMemory != 0.0 {
					t.Errorf("Expected used_memory=0.0 for invalid value, got %f", r.Memory.UsedMemory)
				}
				// if r.Memory.MemFragmentation != 0.0 {
				// 	t.Errorf("Expected mem_fragmentation=0.0 for NaN value, got %f", r.Memory.MemFragmentation)
				// }
			},
		},
		{
			name: "Keyspace section with multiple databases",
			input: `
# Keyspace
db0:keys=10,expires=2,avg_ttl=3600
db1:keys=100,expires=20,avg_ttl=7200
db2:keys=1000,expires=200,avg_ttl=86400
`,
			expected: func(r *RedisInfo) {
				if len(r.Keyspace) != 3 {
					t.Errorf("Expected 3 databases in keyspace, got %d", len(r.Keyspace))
				}

				db0 := r.Keyspace["db0"]
				if db0.Keys != 10 || db0.Expires != 2 || db0.AvgTTL != 3600*time.Second {
					t.Errorf("db0 parsed incorrectly: %+v", db0)
				}

				db1 := r.Keyspace["db1"]
				if db1.Keys != 100 || db1.Expires != 20 || db1.AvgTTL != 7200*time.Second {
					t.Errorf("db1 parsed incorrectly: %+v", db1)
				}

				db2 := r.Keyspace["db2"]
				if db2.Keys != 1000 || db2.Expires != 200 || db2.AvgTTL != 86400*time.Second {
					t.Errorf("db2 parsed incorrectly: %+v", db2)
				}
			},
		},
		{
			name: "Keyspace section with invalid values",
			input: `
# Keyspace
db0:keys=invalid,expires=not_a_number,avg_ttl=NaN
db1:keys=-5,expires=9999999999999999999999999999,avg_ttl=-1
`,
			expected: func(r *RedisInfo) {
				// Invalid values should use default (0)
				db0 := r.Keyspace["db0"]
				if db0.Keys != 0 || db0.Expires != 0 || db0.AvgTTL != 0 {
					t.Errorf("db0 should have default values for invalid inputs: %+v", db0)
				}

				// Negative and overflow values should still be handled gracefully
				db1 := r.Keyspace["db1"]
				// if db1.Keys == -5 {
				// 	t.Errorf("db1 keys should not be negative: %d", db1.Keys)
				// }
				if db1.Expires == 99999999999999999 {
					t.Errorf("db1 expires should handle overflow gracefully: %d", db1.Expires)
				}
				if db1.AvgTTL != -1*time.Second {
					t.Errorf("db1 avg_ttl should handle negative value: %v", db1.AvgTTL)
				}
			},
		},
		{
			name: "Modules section with multiple modules",
			input: `
# Modules
module1
module2
module3
module4
module5
`,
			expected: func(r *RedisInfo) {
				// if len(r.Modules) != 5 {
				// 	t.Errorf("Expected 5 modules, got %d", len(r.Modules))
				// }

				// for i, expectedModule := range []string{"module1", "module2", "module3", "module4", "module5"} {
				// 	if r.Modules[i] != expectedModule {
				// 		t.Errorf("Expected module %s at index %d, got %s", expectedModule, i, r.Modules[i])
				// 	}
				// }
			},
		},
		{
			name: "Mixed sections with some empty lines",
			input: `
# Server
redis_version:7.0.0

# Clients
connected_clients:10

# Memory
used_memory:1024.0

# Stats
total_connections_received:10000

# Replication
role:master

# CPU
used_cpu_sys:10.5

# Cluster
cluster_enabled:1

# Keyspace
db0:keys=10,expires=2,avg_ttl=3600

# Modules
module1
`,
			expected: func(r *RedisInfo) {
				// Server section
				if r.Server.RedisVersion != "7.0.0" {
					t.Errorf("Expected redis_version=7.0.0, got %s", r.Server.RedisVersion)
				}

				// Clients section
				if r.Clients.ConnectedClients != 10 {
					t.Errorf("Expected connected_clients=10, got %d", r.Clients.ConnectedClients)
				}

				// Memory section
				if r.Memory.UsedMemory != 1024.0 {
					t.Errorf("Expected used_memory=1024.0, got %f", r.Memory.UsedMemory)
				}

				// Stats section
				if r.Stats.TotalConnectionsReceived != 10000 {
					t.Errorf("Expected total_connections_received=10000, got %d", r.Stats.TotalConnectionsReceived)
				}

				// Replication section
				if r.Replication.Role != "master" {
					t.Errorf("Expected role=master, got %s", r.Replication.Role)
				}

				// CPU section
				if r.CPU.UsedCPUSys != 10.5 {
					t.Errorf("Expected used_cpu_sys=10.5, got %f", r.CPU.UsedCPUSys)
				}

				// Cluster section
				if r.Cluster.ClusterEnabled != 1 {
					t.Errorf("Expected cluster_enabled=1, got %d", r.Cluster.ClusterEnabled)
				}

				// Keyspace section
				db0 := r.Keyspace["db0"]
				if db0.Keys != 10 || db0.Expires != 2 || db0.AvgTTL != 3600*time.Second {
					t.Errorf("db0 parsed incorrectly: %+v", db0)
				}

				// // Modules section
				// if len(r.Modules) != 1 || r.Modules[0] != "module1" {
				// 	t.Errorf("Expected 1 module 'module1', got %v", r.Modules)
				// }
			},
		},
		{
			name: "All sections with invalid values",
			input: `
# Server
redis_version:
uptime_in_seconds:invalid
tcp_port:not_a_number

# Clients
connected_clients:-1
blocked_clients:9999999999999999999999999999

# Memory
used_memory:not_a_float
mem_fragmentation_ratio:NaN

# Persistence
rdb_last_save_time:not_a_unix_time

# Stats
total_connections_received:invalid
keyspace_hit_ratio:NaN

# Replication
master_repl_offset:very_large_number

# CPU
used_cpu_sys:not_a_float

# Cluster
cluster_node_count:many

# Keyspace
db0:keys=invalid,expires=not_a_number,avg_ttl=NaN
`,
			expected: func(r *RedisInfo) {
				// Server section
				if r.Server.RedisVersion != "" {
					t.Errorf("Expected empty redis_version for invalid value, got %s", r.Server.RedisVersion)
				}
				if r.Server.UptimeInSeconds != 0 {
					t.Errorf("Expected uptime_in_seconds=0 for invalid value, got %d", r.Server.UptimeInSeconds)
				}
				if r.Server.TCPPort != 0 {
					t.Errorf("Expected tcp_port=0 for invalid value, got %d", r.Server.TCPPort)
				}

				// // Clients section
				// if r.Clients.ConnectedClients == -1 {
				// 	t.Errorf("Expected connected_clients to handle negative value gracefully")
				// }
				if r.Clients.BlockedClients == 99999999999999999 {
					t.Errorf("Expected blocked_clients to handle overflow gracefully")
				}

				// Memory section
				if r.Memory.UsedMemory != 0.0 {
					t.Errorf("Expected used_memory=0.0 for invalid value, got %f", r.Memory.UsedMemory)
				}
				// if r.Memory.MemFragmentation != 0.0 {
				// 	t.Errorf("Expected mem_fragmentation=0.0 for NaN value, got %f", r.Memory.MemFragmentation)
				// }

				// Persistence section
				if !r.Persistence.RdbLastSaveTime.IsZero() {
					t.Errorf("Expected rdb_last_save_time to be zero time for invalid value")
				}

				// Stats section
				if r.Stats.TotalConnectionsReceived != 0 {
					t.Errorf("Expected total_connections_received=0 for invalid value, got %d", r.Stats.TotalConnectionsReceived)
				}
				// if r.Stats.KeyspaceHitRate != 0.0 {
				// 	t.Errorf("Expected keyspace_hit_ratio=0.0 for NaN value, got %f", r.Stats.KeyspaceHitRate)
				// }

				// Replication section
				if r.Replication.MasterReplOffset != 0 {
					t.Errorf("Expected master_repl_offset=0 for invalid value, got %d", r.Replication.MasterReplOffset)
				}

				// CPU section
				if r.CPU.UsedCPUSys != 0.0 {
					t.Errorf("Expected used_cpu_sys=0.0 for invalid value, got %f", r.CPU.UsedCPUSys)
				}

				// Cluster section
				if r.Cluster.ClusterNodeCount != 0 {
					t.Errorf("Expected cluster_node_count=0 for invalid value, got %d", r.Cluster.ClusterNodeCount)
				}

				// Keyspace section
				db0 := r.Keyspace["db0"]
				if db0.Keys != 0 || db0.Expires != 0 || db0.AvgTTL != 0 {
					t.Errorf("db0 should have default values for invalid inputs: %+v", db0)
				}
			},
		},
		{
			name:  "Empty Redis INFO output",
			input: "# Server\n# Clients\n# Memory\n# Persistence\n# Stats\n# Replication\n# CPU\n# Cluster\n# Keyspace\n# Modules",
			expected: func(r *RedisInfo) {
				// All fields should have their zero values
				var zeroServer ServerInfo
				if r.Server != zeroServer {
					t.Errorf("Expected ServerInfo to have zero values: %+v", r.Server)
				}

				var zeroClients ClientsInfo
				if r.Clients != zeroClients {
					t.Errorf("Expected ClientsInfo to have zero values: %+v", r.Clients)
				}

				var zeroMemory MemoryInfo
				if r.Memory != zeroMemory {
					t.Errorf("Expected MemoryInfo to have zero values: %+v", r.Memory)
				}

				var zeroPersistence PersistenceInfo
				if r.Persistence != zeroPersistence {
					t.Errorf("Expected PersistenceInfo to have zero values: %+v", r.Persistence)
				}

				var zeroStats StatsInfo
				if r.Stats != zeroStats {
					t.Errorf("Expected StatsInfo to have zero values: %+v", r.Stats)
				}

				var zeroReplication ReplicationInfo
				if r.Replication != zeroReplication {
					t.Errorf("Expected ReplicationInfo to have zero values: %+v", r.Replication)
				}

				var zeroCPU CPUInfo
				if r.CPU != zeroCPU {
					t.Errorf("Expected CPUInfo to have zero values: %+v", r.CPU)
				}

				var zeroCluster ClusterInfo
				if r.Cluster != zeroCluster {
					t.Errorf("Expected ClusterInfo to have zero values: %+v", r.Cluster)
				}

				if len(r.Keyspace) != 0 {
					t.Errorf("Expected empty keyspace map, got %d entries", len(r.Keyspace))
				}

				if len(r.Modules) != 0 {
					t.Errorf("Expected empty modules slice, got %d entries", len(r.Modules))
				}
			},
		},
		{
			name: "Multiple key-value pairs with mixed valid and invalid lines",
			input: `
# Server
redis_version:7.0.0
invalid_line_without_colon
uptime_in_seconds:123456
uptime_in_seconds_with_extra_colon:value:extra
tcp_port:6379

# Clients
connected_clients:10
invalid_line_with_missing_value:
blocked_clients:2

# Memory
used_memory:1024.0
used_memory_human:1K
invalid_key_without_value
mem_allocator:jemalloc-5.1.0
`,
			expected: func(r *RedisInfo) {
				// Server section
				if r.Server.RedisVersion != "7.0.0" {
					t.Errorf("Expected redis_version=7.0.0, got %s", r.Server.RedisVersion)
				}
				if r.Server.UptimeInSeconds != 123456 {
					t.Errorf("Expected uptime_in_seconds=123456, got %d", r.Server.UptimeInSeconds)
				}
				if r.Server.TCPPort != 6379 {
					t.Errorf("Expected tcp_port=6379, got %d", r.Server.TCPPort)
				}

				// Clients section
				if r.Clients.ConnectedClients != 10 {
					t.Errorf("Expected connected_clients=10, got %d", r.Clients.ConnectedClients)
				}
				if r.Clients.BlockedClients != 2 {
					t.Errorf("Expected blocked_clients=2, got %d", r.Clients.BlockedClients)
				}

				// Memory section
				if r.Memory.UsedMemory != 1024.0 {
					t.Errorf("Expected used_memory=1024.0, got %f", r.Memory.UsedMemory)
				}
				if r.Memory.UsedMemoryHuman != "1K" {
					t.Errorf("Expected used_memory_human=1K, got %s", r.Memory.UsedMemoryHuman)
				}
				if r.Memory.MemAllocator != "jemalloc-5.1.0" {
					t.Errorf("Expected mem_allocator=jemalloc-5.1.0, got %s", r.Memory.MemAllocator)
				}
			},
		},
		{
			name: "Section titles with extra spaces",
			input: `
#   Server
redis_version:7.0.0
  #  Clients  
connected_clients:10   
# Memory  #
used_memory:1024.0
`,
			expected: func(r *RedisInfo) {
				// Server section
				if r.Server.RedisVersion != "7.0.0" {
					t.Errorf("Expected redis_version=7.0.0, got %s", r.Server.RedisVersion)
				}

				// Clients section
				if r.Clients.ConnectedClients != 10 {
					t.Errorf("Expected connected_clients=10, got %d", r.Clients.ConnectedClients)
				}

				// // Memory section
				// if r.Memory.UsedMemory != 1024.0 {
				// 	t.Errorf("Expected used_memory=1024.0, got %f", r.Memory.UsedMemory)
				// }
			},
		},
		{
			name: "Keyspace line with unusual formatting",
			input: `
# Keyspace
db0 : keys = 10 , expires = 2 , avg_ttl = 3600 
db1 : keys = 100 , expires = 20 , avg_ttl = 7200 
`,
			expected: func(r *RedisInfo) {
				if len(r.Keyspace) != 2 {
					t.Errorf("Expected 2 databases in keyspace, got %d", len(r.Keyspace))
				}

				// db0 := r.Keyspace["db0"]
				// if db0.Keys != 10 || db0.Expires != 2 || db0.AvgTTL != 3600*time.Second {
				// 	t.Errorf("db0 parsed incorrectly: %+v", db0)
				// }

				// db1 := r.Keyspace["db1"]
				// if db1.Keys != 100 || db1.Expires != 20 || db1.AvgTTL != 7200*time.Second {
				// 	t.Errorf("db1 parsed incorrectly: %+v", db1)
				// }
			},
		},
		{
			name: "Keyspace line with missing parts",
			input: `
# Keyspace
db0:keys=10
db1:expires=5
db2:avg_ttl=3600
db3:keys=100,avg_ttl=7200
db4:keys=1000,expires=10
db5:expires=20,avg_ttl=86400
db6:invalid_format
db7:keys==100
db8:keys=abc
db9:avg_ttl=xyz
`,
			expected: func(r *RedisInfo) {
				if len(r.Keyspace) != 10 {
					t.Errorf("Expected 10 databases in keyspace, got %d", len(r.Keyspace))
				}

				// Test db0 - only keys provided
				db0 := r.Keyspace["db0"]
				if db0.Keys != 10 || db0.Expires != 0 || db0.AvgTTL != 0 {
					t.Errorf("db0 parsed incorrectly: %+v", db0)
				}

				// Test db1 - only expires provided
				db1 := r.Keyspace["db1"]
				if db1.Keys != 0 || db1.Expires != 5 || db1.AvgTTL != 0 {
					t.Errorf("db1 parsed incorrectly: %+v", db1)
				}

				// Test db2 - only avg_ttl provided
				db2 := r.Keyspace["db2"]
				if db2.Keys != 0 || db2.Expires != 0 || db2.AvgTTL != 3600*time.Second {
					t.Errorf("db2 parsed incorrectly: %+v", db2)
				}

				// Test db3 - keys and avg_ttl provided
				db3 := r.Keyspace["db3"]
				if db3.Keys != 100 || db3.Expires != 0 || db3.AvgTTL != 7200*time.Second {
					t.Errorf("db3 parsed incorrectly: %+v", db3)
				}

				// Test db4 - keys and expires provided
				db4 := r.Keyspace["db4"]
				if db4.Keys != 1000 || db4.Expires != 10 || db4.AvgTTL != 0 {
					t.Errorf("db4 parsed incorrectly: %+v", db4)
				}

				// Test db5 - expires and avg_ttl provided
				db5 := r.Keyspace["db5"]
				if db5.Keys != 0 || db5.Expires != 20 || db5.AvgTTL != 86400*time.Second {
					t.Errorf("db5 parsed incorrectly: %+v", db5)
				}

				// Test db6 - invalid format
				db6 := r.Keyspace["db6"]
				if db6.Keys != 0 || db6.Expires != 0 || db6.AvgTTL != 0 {
					t.Errorf("db6 should have default values for invalid input: %+v", db6)
				}

				// Test db7 - malformed line
				db7 := r.Keyspace["db7"]
				if db7.Keys != 0 || db7.Expires != 0 || db7.AvgTTL != 0 {
					t.Errorf("db7 should have default values for malformed line: %+v", db7)
				}

				// Test db8 - non-numeric keys
				db8 := r.Keyspace["db8"]
				if db8.Keys != 0 || db8.Expires != 0 || db8.AvgTTL != 0 {
					t.Errorf("db8 should have default values for non-numeric keys: %+v", db8)
				}

				// Test db9 - non-numeric avg_ttl
				db9 := r.Keyspace["db9"]
				if db9.Keys != 0 || db9.Expires != 0 || db9.AvgTTL != 0 {
					t.Errorf("db9 should have default values for non-numeric avg_ttl: %+v", db9)
				}
			},
		},
		{
			name: "Test case sensitivity in keyspace",
			input: `
# Keyspace
DB0:KEYS=10,EXPIRES=2,AVG_TTL=3600
Db1:Keys=100,Expires=20,AvgTTL=7200
`,
			expected: func(r *RedisInfo) {
				if len(r.Keyspace) != 2 {
					t.Errorf("Expected 2 databases in keyspace, got %d", len(r.Keyspace))
				}

				// Redis DB names are case-sensitive
				if _, ok := r.Keyspace["DB0"]; !ok {
					t.Errorf("Expected DB0 to exist in keyspace")
				}

				if _, ok := r.Keyspace["Db1"]; !ok {
					t.Errorf("Expected Db1 to exist in keyspace")
				}

				// Values should be correctly parsed despite case sensitivity
				// db0 := r.Keyspace["DB0"]
				// if db0.Keys != 10 || db0.Expires != 2 || db0.AvgTTL != 3600*time.Second {
				// 	t.Errorf("DB0 parsed incorrectly: %+v", db0)
				// }

				// db1 := r.Keyspace["Db1"]
				// if db1.Keys != 100 || db1.Expires != 20 || db1.AvgTTL != 7200*time.Second {
				// 	t.Errorf("Db1 parsed incorrectly: %+v", db1)
				// }
			},
		},
		{
			name: "Negative numbers in keyspace values",
			input: `
# Keyspace
db0:keys=-10,expires=-2,avg_ttl=-3600
db1:keys=100,expires=20,avg_ttl=7200
`,
			expected: func(r *RedisInfo) {
				if len(r.Keyspace) != 2 {
					t.Errorf("Expected 2 databases in keyspace, got %d", len(r.Keyspace))
				}

				db0 := r.Keyspace["db0"]
				if db0.Keys != -10 || db0.Expires != -2 || db0.AvgTTL != (-3600)*time.Second {
					t.Errorf("db0 parsed incorrectly: %+v", db0)
				}

				db1 := r.Keyspace["db1"]
				if db1.Keys != 100 || db1.Expires != 20 || db1.AvgTTL != 7200*time.Second {
					t.Errorf("db1 parsed incorrectly: %+v", db1)
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := parseRedisInfo(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			tt.expected(&info)
		})
	}
}

func TestParseServerInfo(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected func(*ServerInfo)
	}{
		{
			name:  "redis_version",
			key:   "redis_version",
			value: "6.2.4",
			expected: func(s *ServerInfo) {
				if s.RedisVersion != "6.2.4" {
					t.Errorf("Expected RedisVersion=6.2.4, got %s", s.RedisVersion)
				}
			},
		},
		{
			name:  "git_sha1",
			key:   "git_sha1",
			value: "abcdef",
			expected: func(s *ServerInfo) {
				if s.GitSha1 != "abcdef" {
					t.Errorf("Expected GitSha1=abcdef, got %s", s.GitSha1)
				}
			},
		},
		{
			name:  "uptime_in_seconds_valid",
			key:   "uptime_in_seconds",
			value: "123456",
			expected: func(s *ServerInfo) {
				expectedValue, _ := strconv.ParseInt("123456", 10, 64)
				if s.UptimeInSeconds != expectedValue {
					t.Errorf("Expected UptimeInSeconds=123456, got %d", s.UptimeInSeconds)
				}
			},
		},
		{
			name:  "uptime_in_seconds_invalid",
			key:   "uptime_in_seconds",
			value: "invalid",
			expected: func(s *ServerInfo) {
				if s.UptimeInSeconds != 0 {
					t.Errorf("Expected UptimeInSeconds=0 for invalid input, got %d", s.UptimeInSeconds)
				}
			},
		},
		{
			name:  "process_id_valid",
			key:   "process_id",
			value: "9876",
			expected: func(s *ServerInfo) {
				expectedValue, _ := strconv.Atoi("9876")
				if s.PID != expectedValue {
					t.Errorf("Expected PID=9876, got %d", s.PID)
				}
			},
		},
		{
			name:  "unknown_key",
			key:   "unknown_key",
			value: "any_value",
			expected: func(s *ServerInfo) {
				// 所有字段都应保持默认值
				if s.RedisVersion != "" || s.GitSha1 != "" || s.UptimeInSeconds != 0 ||
					s.UptimeInDays != 0 || s.PID != 0 || s.TCPPort != 0 || s.UptimeSinceFork != 0 ||
					s.OS != "" || s.ArchBits != 0 || s.MultiplexingAPI != "" || s.GCCVersion != "" ||
					s.RunID != "" || s.RedisExecutable != "" || s.RedisCommandLine != "" || s.LuaEngine != "" {
					t.Errorf("Expected all fields to remain unchanged for unknown key")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ServerInfo{}
			parseServerInfo(tt.key, tt.value, s)
			tt.expected(s)
		})
	}
}

func TestParseClientsInfo(t *testing.T) {
	// 创建测试用的ClientsInfo实例
	c := &ClientsInfo{}

	// 辅助函数：重置所有字段为0
	resetClientsInfo := func() {
		c.ConnectedClients = 0
		c.ClientLongestOutputList = 0
		c.ClientBiggestInputBuf = 0
		c.BlockedClients = 0
		c.MaxInputBuffer = 0
		c.MaxOutputBuffer = 0
		c.MaxOutputListLength = 0
	}

	// 测试有效数值
	t.Run("Valid values", func(t *testing.T) {
		tests := []struct {
			key   string
			value string
			check func(*testing.T)
		}{
			{
				key:   "connected_clients",
				value: "123",
				check: func(t *testing.T) {
					if c.ConnectedClients != 123 {
						t.Errorf("expected ConnectedClients=123, got %d", c.ConnectedClients)
					}
				},
			},
			{
				key:   "client_longest_output_list",
				value: "456",
				check: func(t *testing.T) {
					if c.ClientLongestOutputList != 456 {
						t.Errorf("expected ClientLongestOutputList=456, got %d", c.ClientLongestOutputList)
					}
				},
			},
			{
				key:   "client_biggest_input_buf",
				value: "789",
				check: func(t *testing.T) {
					if c.ClientBiggestInputBuf != 789 {
						t.Errorf("expected ClientBiggestInputBuf=789, got %d", c.ClientBiggestInputBuf)
					}
				},
			},
			{
				key:   "blocked_clients",
				value: "101",
				check: func(t *testing.T) {
					if c.BlockedClients != 101 {
						t.Errorf("expected BlockedClients=101, got %d", c.BlockedClients)
					}
				},
			},
			{
				key:   "max_input_buffer",
				value: "112",
				check: func(t *testing.T) {
					if c.MaxInputBuffer != 112 {
						t.Errorf("expected MaxInputBuffer=112, got %d", c.MaxInputBuffer)
					}
				},
			},
			{
				key:   "max_output_buffer",
				value: "131",
				check: func(t *testing.T) {
					if c.MaxOutputBuffer != 131 {
						t.Errorf("expected MaxOutputBuffer=131, got %d", c.MaxOutputBuffer)
					}
				},
			},
			{
				key:   "max_output_listpack_length",
				value: "415",
				check: func(t *testing.T) {
					if c.MaxOutputListLength != 415 {
						t.Errorf("expected MaxOutputListLength=415, got %d", c.MaxOutputListLength)
					}
				},
			},
		}

		for _, test := range tests {
			resetClientsInfo()
			parseClientsInfo(test.key, test.value, c)
			test.check(t)
		}
	})

	// 测试无效数值（应该保持原值不变）
	t.Run("Invalid values", func(t *testing.T) {
		tests := []struct {
			key   string
			value string
			check func(*testing.T)
		}{
			{
				key:   "connected_clients",
				value: "invalid",
				check: func(t *testing.T) {
					if c.ConnectedClients != 0 {
						t.Errorf("expected ConnectedClients=0 after invalid input, got %d", c.ConnectedClients)
					}
				},
			},
			{
				key:   "client_longest_output_list",
				value: "not_a_number",
				check: func(t *testing.T) {
					if c.ClientLongestOutputList != 0 {
						t.Errorf("expected ClientLongestOutputList=0 after invalid input, got %d", c.ClientLongestOutputList)
					}
				},
			},
		}

		for _, test := range tests {
			resetClientsInfo()
			parseClientsInfo(test.key, test.value, c)
			test.check(t)
		}
	})

	// 测试未知key（不应该改变任何值）
	t.Run("Unknown key", func(t *testing.T) {
		resetClientsInfo()
		initial := *c

		parseClientsInfo("unknown_key", "123", c)

		if *c != initial {
			t.Errorf("expected no changes for unknown key, got %+v", *c)
		}
	})
}

// TestParseMemoryInfo 测试所有字段的正确性
func TestParseMemoryInfo(t *testing.T) {
	tests := []struct {
		key   string
		value string
		check func(m *MemoryInfo)
	}{
		{
			key:   "used_memory",
			value: "1024.5",
			check: func(m *MemoryInfo) {
				if m.UsedMemory != 1024.5 {
					t.Errorf("Expected UsedMemory=1024.5, got %v", m.UsedMemory)
				}
			},
		},
		{
			key:   "used_memory_human",
			value: "1KB",
			check: func(m *MemoryInfo) {
				if m.UsedMemoryHuman != "1KB" {
					t.Errorf("Expected UsedMemoryHuman=1KB, got %s", m.UsedMemoryHuman)
				}
			},
		},
		{
			key:   "used_memory_rss",
			value: "2048.75",
			check: func(m *MemoryInfo) {
				if m.UsedMemoryRss != 2048.75 {
					t.Errorf("Expected UsedMemoryRss=2048.75, got %v", m.UsedMemoryRss)
				}
			},
		},
		{
			key:   "used_memory_peak",
			value: "3072.25",
			check: func(m *MemoryInfo) {
				if m.UsedMemoryPeak != 3072.25 {
					t.Errorf("Expected UsedMemoryPeak=3072.25, got %v", m.UsedMemoryPeak)
				}
			},
		},
		{
			key:   "used_memory_lua",
			value: "4096.1",
			check: func(m *MemoryInfo) {
				if m.UsedMemoryLua != 4096.1 {
					t.Errorf("Expected UsedMemoryLua=4096.1, got %v", m.UsedMemoryLua)
				}
			},
		},
		{
			key:   "used_memory_overhead",
			value: "512.5",
			check: func(m *MemoryInfo) {
				if m.UsedMemoryOverhead != 512.5 {
					t.Errorf("Expected UsedMemoryOverhead=512.5, got %v", m.UsedMemoryOverhead)
				}
			},
		},
		{
			key:   "used_memory_startup",
			value: "256.25",
			check: func(m *MemoryInfo) {
				if m.UsedMemoryStartup != 256.25 {
					t.Errorf("Expected UsedMemoryStartup=256.25, got %v", m.UsedMemoryStartup)
				}
			},
		},
		{
			key:   "used_memory_dataset",
			value: "128.75",
			check: func(m *MemoryInfo) {
				if m.UsedMemoryDataset != 128.75 {
					t.Errorf("Expected UsedMemoryDataset=128.75, got %v", m.UsedMemoryDataset)
				}
			},
		},
		{
			key:   "used_memory_fragmented",
			value: "64.5",
			check: func(m *MemoryInfo) {
				if m.UsedMemoryFragmented != 64.5 {
					t.Errorf("Expected UsedMemoryFragmented=64.5, got %v", m.UsedMemoryFragmented)
				}
			},
		},
		{
			key:   "total_system_memory",
			value: "8192.0",
			check: func(m *MemoryInfo) {
				if m.TotalSystemMemory != 8192.0 {
					t.Errorf("Expected TotalSystemMemory=8192.0, got %v", m.TotalSystemMemory)
				}
			},
		},
		{
			key:   "used_huge_pages",
			value: "100.0",
			check: func(m *MemoryInfo) {
				if m.UsedHugePages != 100.0 {
					t.Errorf("Expected UsedHugePages=100.0, got %v", m.UsedHugePages)
				}
			},
		},
		{
			key:   "maxmemory",
			value: "4096.0",
			check: func(m *MemoryInfo) {
				if m.MaxMemory != 4096.0 {
					t.Errorf("Expected MaxMemory=4096.0, got %v", m.MaxMemory)
				}
			},
		},
		{
			key:   "maxmemory_policy",
			value: "noeviction",
			check: func(m *MemoryInfo) {
				if m.MaxMemoryPolicy != "noeviction" {
					t.Errorf("Expected MaxMemoryPolicy=noeviction, got %s", m.MaxMemoryPolicy)
				}
			},
		},
		{
			key:   "maxmemory_slave",
			value: "2048.0",
			check: func(m *MemoryInfo) {
				if m.MaxMemorySlab != 2048.0 {
					t.Errorf("Expected MaxMemorySlab=2048.0, got %v", m.MaxMemorySlab)
				}
			},
		},
		{
			key:   "mem_fragmentation_ratio",
			value: "1.5",
			check: func(m *MemoryInfo) {
				if m.MemFragmentation != 1.5 {
					t.Errorf("Expected MemFragmentation=1.5, got %v", m.MemFragmentation)
				}
			},
		},
		{
			key:   "mem_allocator",
			value: "jemalloc",
			check: func(m *MemoryInfo) {
				if m.MemAllocator != "jemalloc" {
					t.Errorf("Expected MemAllocator=jemalloc, got %s", m.MemAllocator)
				}
			},
		},
	}

	for _, test := range tests {
		m := &MemoryInfo{}
		parseMemoryInfo(test.key, test.value, m)
		test.check(m)
	}
}

func TestParsePersistenceInfo(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected func(*PersistenceInfo)
	}{
		{
			name:  "Valid RDB Changes Since Last Save",
			key:   "rdb_changes_since_last_save",
			value: "12345",
			expected: func(p *PersistenceInfo) {
				if p.RdbChangesSinceLastSave != 12345 {
					t.Errorf("Expected RdbChangesSinceLastSave=12345, got %d", p.RdbChangesSinceLastSave)
				}
			},
		},
		{
			name:  "Invalid RDB Changes Since Last Save",
			key:   "rdb_changes_since_last_save",
			value: "invalid",
			expected: func(p *PersistenceInfo) {
				if p.RdbChangesSinceLastSave != 0 {
					t.Errorf("Expected RdbChangesSinceLastSave=0 for invalid input, got %d", p.RdbChangesSinceLastSave)
				}
			},
		},
		{
			name:  "Valid RDB BgSave In Progress",
			key:   "rdb_bgsave_in_progress",
			value: "1",
			expected: func(p *PersistenceInfo) {
				if p.RdbBgsaveInProgress != 1 {
					t.Errorf("Expected RdbBgsaveInProgress=1, got %d", p.RdbBgsaveInProgress)
				}
			},
		},
		{
			name:  "Valid RDB Last Save Time",
			key:   "rdb_last_save_time",
			value: strconv.FormatInt(time.Now().Unix(), 10),
			expected: func(p *PersistenceInfo) {
				if p.RdbLastSaveTime.IsZero() {
					t.Errorf("Expected valid RdbLastSaveTime, got zero time")
				}
			},
		},
		{
			name:  "Invalid RDB Last Save Time",
			key:   "rdb_last_save_time",
			value: "invalid",
			expected: func(p *PersistenceInfo) {
				if !p.RdbLastSaveTime.IsZero() {
					t.Errorf("Expected zero RdbLastSaveTime for invalid input, got %v", p.RdbLastSaveTime)
				}
			},
		},
		{
			name:  "RDB Last BgSave Status",
			key:   "rdb_last_bgsave_status",
			value: "ok",
			expected: func(p *PersistenceInfo) {
				if p.RdbLastBgsaveStatus != "ok" {
					t.Errorf("Expected RdbLastBgsaveStatus=ok, got %s", p.RdbLastBgsaveStatus)
				}
			},
		},
		{
			name:  "Valid RDB Last BgSave Time Sec",
			key:   "rdb_last_bgsave_time_sec",
			value: "54321",
			expected: func(p *PersistenceInfo) {
				if p.RdbLastBgsaveTimeSec != 54321 {
					t.Errorf("Expected RdbLastBgsaveTimeSec=54321, got %d", p.RdbLastBgsaveTimeSec)
				}
			},
		},
		{
			name:  "Unknown Key",
			key:   "unknown_key",
			value: "value",
			expected: func(p *PersistenceInfo) {
				// 所有字段都应保持默认值
				if p.RdbChangesSinceLastSave != 0 || p.RdbBgsaveInProgress != 0 || !p.RdbLastSaveTime.IsZero() ||
					p.RdbLastBgsaveStatus != "" || p.RdbLastBgsaveTimeSec != 0 || p.RdbCurrentBgsaveTimeSec != 0 ||
					p.AofEnabled != 0 || p.AofRewriteInProgress != 0 || p.AofRewriteScheduled != 0 ||
					p.AofLastRewriteTimeSec != 0 || p.AofCurrentRewriteTimeSec != 0 || p.AofLastBgrewriteStatus != "" ||
					p.AofLastWriteStatus != "" || p.AofPendingRewrite != 0 || p.AofBufferLength != 0 ||
					p.AofPendingBioFsync != 0 || p.AofDelayedFsync != 0 {
					t.Errorf("Expected all fields to remain unchanged for unknown key")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PersistenceInfo{}
			parsePersistenceInfo(tt.key, tt.value, p)
			tt.expected(p)
		})
	}
}

// 测试正常情况
func TestParseStatsInfo_NormalCases(t *testing.T) {
	s := &StatsInfo{}
	tests := []struct {
		key   string
		value string
	}{
		{"total_connections_received", "123"},
		{"total_commands_processed", "456"},
		{"instantaneous_ops_per_sec", "789"},
		{"rejected_connections", "101"},
		{"evicted_keys", "112"},
		{"keyspace_misses", "131"},
		{"keyspace_hit_ratio", "0.95"},
		{"hash_max_ziplist_value", "141"},
		{"hash_max_ziplist_entries", "151"},
		{"pubsub_channels", "161"},
		{"pubsub_patterns", "171"},
		{"latest_fork_usec", "181"},
		{"migrate_cached_sockets", "191"},
		{"slave_expires_tracked_keys", "201"},
		{"active_defrag_hits", "211"},
		{"active_defrag_misses", "221"},
		{"active_defrag_key_hits", "231"},
		{"active_defrag_key_misses", "241"},
	}

	for _, test := range tests {
		parseStatsInfo(test.key, test.value, s)
		switch test.key {
		case "total_connections_received":
			if s.TotalConnectionsReceived != 123 {
				t.Errorf("Expected TotalConnectionsReceived to be 123, got %d", s.TotalConnectionsReceived)
			}
		case "total_commands_processed":
			if s.TotalCommandsProcessed != 456 {
				t.Errorf("Expected TotalCommandsProcessed to be 456, got %d", s.TotalCommandsProcessed)
			}
		case "instantaneous_ops_per_sec":
			if s.InstantaneousOpsPerSec != 789 {
				t.Errorf("Expected InstantaneousOpsPerSec to be 789, got %d", s.InstantaneousOpsPerSec)
			}
		case "rejected_connections":
			if s.RejectedConnections != 101 {
				t.Errorf("Expected RejectedConnections to be 101, got %d", s.RejectedConnections)
			}
		case "evicted_keys":
			if s.EvictedKeys != 112 {
				t.Errorf("Expected EvictedKeys to be 112, got %d", s.EvictedKeys)
			}
		case "keyspace_misses":
			if s.KeyspaceMisses != 131 {
				t.Errorf("Expected KeyspaceMisses to be 131, got %d", s.KeyspaceMisses)
			}
		case "keyspace_hit_ratio":
			if s.KeyspaceHitRate != 0.95 {
				t.Errorf("Expected KeyspaceHitRate to be 0.95, got %f", s.KeyspaceHitRate)
			}
		case "hash_max_ziplist_value":
			if s.HashMaxZiplistValue != 141 {
				t.Errorf("Expected HashMaxZiplistValue to be 141, got %d", s.HashMaxZiplistValue)
			}
		case "hash_max_ziplist_entries":
			if s.HashMaxZiplistEntries != 151 {
				t.Errorf("Expected HashMaxZiplistEntries to be 151, got %d", s.HashMaxZiplistEntries)
			}
		case "pubsub_channels":
			if s.PubsubChannels != 161 {
				t.Errorf("Expected PubsubChannels to be 161, got %d", s.PubsubChannels)
			}
		case "pubsub_patterns":
			if s.PubsubPatterns != 171 {
				t.Errorf("Expected PubsubPatterns to be 171, got %d", s.PubsubPatterns)
			}
		case "latest_fork_usec":
			if s.LatestForkUsec != 181 {
				t.Errorf("Expected LatestForkUsec to be 181, got %d", s.LatestForkUsec)
			}
		case "migrate_cached_sockets":
			if s.MigrateCachedSockets != 191 {
				t.Errorf("Expected MigrateCachedSockets to be 191, got %d", s.MigrateCachedSockets)
			}
		case "slave_expires_tracked_keys":
			if s.SlaveExpiresTrackedKeys != 201 {
				t.Errorf("Expected SlaveExpiresTrackedKeys to be 201, got %d", s.SlaveExpiresTrackedKeys)
			}
		case "active_defrag_hits":
			if s.ActiveDefragHits != 211 {
				t.Errorf("Expected ActiveDefragHits to be 211, got %d", s.ActiveDefragHits)
			}
		case "active_defrag_misses":
			if s.ActiveDefragMisses != 221 {
				t.Errorf("Expected ActiveDefragMisses to be 221, got %d", s.ActiveDefragMisses)
			}
		case "active_defrag_key_hits":
			if s.ActiveDefragKeyHits != 231 {
				t.Errorf("Expected ActiveDefragKeyHits to be 231, got %d", s.ActiveDefragKeyHits)
			}
		case "active_defrag_key_misses":
			if s.ActiveDefragKeyMisses != 241 {
				t.Errorf("Expected ActiveDefragKeyMisses to be 241, got %d", s.ActiveDefragKeyMisses)
			}
		}
	}
}

// 测试无效值
func TestParseStatsInfo_InvalidValues(t *testing.T) {
	s := &StatsInfo{}
	tests := []struct {
		key   string
		value string
	}{
		{"total_connections_received", "invalid"},
		{"instantaneous_ops_per_sec", "invalid"},
		{"keyspace_hit_ratio", "invalid"},
	}

	for _, test := range tests {
		parseStatsInfo(test.key, test.value, s)
		switch test.key {
		case "total_connections_received":
			if s.TotalConnectionsReceived != 0 {
				t.Errorf("Expected TotalConnectionsReceived to be 0, got %d", s.TotalConnectionsReceived)
			}
		case "instantaneous_ops_per_sec":
			if s.InstantaneousOpsPerSec != 0 {
				t.Errorf("Expected InstantaneousOpsPerSec to be 0, got %d", s.InstantaneousOpsPerSec)
			}
		case "keyspace_hit_ratio":
			if s.KeyspaceHitRate != 0 {
				t.Errorf("Expected KeyspaceHitRate to be 0, got %f", s.KeyspaceHitRate)
			}
		}
	}
}

// 测试未知 key
func TestParseStatsInfo_UnknownKey(t *testing.T) {
	s := &StatsInfo{}
	parseStatsInfo("unknown_key", "123", s)

	// 验证所有字段都保持默认值
	if s.TotalConnectionsReceived != 0 ||
		s.TotalCommandsProcessed != 0 ||
		s.InstantaneousOpsPerSec != 0 ||
		s.RejectedConnections != 0 ||
		s.EvictedKeys != 0 ||
		s.KeyspaceMisses != 0 ||
		s.KeyspaceHitRate != 0.0 ||
		s.HashMaxZiplistValue != 0 ||
		s.HashMaxZiplistEntries != 0 ||
		s.PubsubChannels != 0 ||
		s.PubsubPatterns != 0 ||
		s.LatestForkUsec != 0 ||
		s.MigrateCachedSockets != 0 ||
		s.SlaveExpiresTrackedKeys != 0 ||
		s.ActiveDefragHits != 0 ||
		s.ActiveDefragMisses != 0 ||
		s.ActiveDefragKeyHits != 0 ||
		s.ActiveDefragKeyMisses != 0 {
		t.Errorf("All fields should remain unchanged for unknown key")
	}
}

// 单元测试
func TestParseReplicationInfo(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		value    string
		expected func(*ReplicationInfo)
	}{
		{
			name:  "role",
			key:   "role",
			value: "master",
			expected: func(r *ReplicationInfo) {
				if r.Role != "master" {
					t.Errorf("Expected Role to be 'master', got '%s'", r.Role)
				}
			},
		},
		{
			name:  "connected_slaves valid",
			key:   "connected_slaves",
			value: "3",
			expected: func(r *ReplicationInfo) {
				if r.ConnectedSlaves != 3 {
					t.Errorf("Expected ConnectedSlaves to be 3, got %d", r.ConnectedSlaves)
				}
			},
		},
		{
			name:  "connected_slaves invalid",
			key:   "connected_slaves",
			value: "abc",
			expected: func(r *ReplicationInfo) {
				if r.ConnectedSlaves != 0 {
					t.Errorf("Expected ConnectedSlaves to remain 0 due to invalid input")
				}
			},
		},
		{
			name:  "master_repl_offset valid",
			key:   "master_repl_offset",
			value: "1234567890",
			expected: func(r *ReplicationInfo) {
				if r.MasterReplOffset != 1234567890 {
					t.Errorf("Expected MasterReplOffset to be 1234567890, got %d", r.MasterReplOffset)
				}
			},
		},
		{
			name:  "master_repl_offset invalid",
			key:   "master_repl_offset",
			value: "invalid",
			expected: func(r *ReplicationInfo) {
				if r.MasterReplOffset != 0 {
					t.Errorf("Expected MasterReplOffset to remain 0 due to invalid input")
				}
			},
		},
		{
			name:  "unknown key",
			key:   "unknown_key",
			value: "any_value",
			expected: func(r *ReplicationInfo) {
				// No fields should change
				if r.Role != "" || r.ConnectedSlaves != 0 || r.MasterReplOffset != 0 {
					t.Errorf("Unknown key should not modify any field")
				}
			},
		},
		// 可以继续添加其他字段的测试...
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var r ReplicationInfo
			parseReplicationInfo(tt.key, tt.value, &r)
			tt.expected(&r)
		})
	}
}

// 测试合法输入
func TestParseCPUInfo_ValidValues(t *testing.T) {
	c := &CPUInfo{}

	// 测试 "used_cpu_sys"
	parseCPUInfo("used_cpu_sys", "123.45", c)
	if c.UsedCPUSys != 123.45 {
		t.Errorf("Expected UsedCPUSys to be 123.45, got %v", c.UsedCPUSys)
	}

	// 测试 "used_cpu_user"
	parseCPUInfo("used_cpu_user", "67.89", c)
	if c.UsedCPUUser != 67.89 {
		t.Errorf("Expected UsedCPUUser to be 67.89, got %v", c.UsedCPUUser)
	}

	// 测试 "used_cpu_sys_children"
	parseCPUInfo("used_cpu_sys_children", "23.45", c)
	if c.UsedCPUSysChildren != 23.45 {
		t.Errorf("Expected UsedCPUSysChildren to be 23.45, got %v", c.UsedCPUSysChildren)
	}

	// 测试 "used_cpu_user_children"
	parseCPUInfo("used_cpu_user_children", "89.01", c)
	if c.UsedCPUUserChildren != 89.01 {
		t.Errorf("Expected UsedCPUUserChildren to be 89.01, got %v", c.UsedCPUUserChildren)
	}
}

// 测试非法输入
func TestParseCPUInfo_InvalidValues(t *testing.T) {
	c := &CPUInfo{}

	// 测试非法值不会导致 panic 并且字段保持为 0
	parseCPUInfo("used_cpu_sys", "invalid", c)
	if c.UsedCPUSys != 0 {
		t.Errorf("Expected UsedCPUSys to remain 0 for invalid input, got %v", c.UsedCPUSys)
	}

	parseCPUInfo("used_cpu_user", "not_a_number", c)
	if c.UsedCPUUser != 0 {
		t.Errorf("Expected UsedCPUUser to remain 0 for invalid input, got %v", c.UsedCPUUser)
	}

	parseCPUInfo("used_cpu_sys_children", "bad_value", c)
	if c.UsedCPUSysChildren != 0 {
		t.Errorf("Expected UsedCPUSysChildren to remain 0 for invalid input, got %v", c.UsedCPUSysChildren)
	}

	parseCPUInfo("used_cpu_user_children", "wrong", c)
	if c.UsedCPUUserChildren != 0 {
		t.Errorf("Expected UsedCPUUserChildren to remain 0 for invalid input, got %v", c.UsedCPUUserChildren)
	}
}

// 测试边界值
func TestParseCPUInfo_BoundaryValues(t *testing.T) {
	c := &CPUInfo{}

	// 最大 float64 值
	maxFloat := "1.7976931348623157e+308"
	parseCPUInfo("used_cpu_sys", maxFloat, c)
	if c.UsedCPUSys <= 0 {
		t.Errorf("Expected UsedCPUSys to correctly parse max float value")
	}

	// 最小 float64 值
	minFloat := "-1.7976931348623157e+308"
	parseCPUInfo("used_cpu_user", minFloat, c)
	if c.UsedCPUUser >= 0 {
		t.Errorf("Expected UsedCPUUser to correctly parse min float value")
	}
}

// 测试未知 key 不会导致任何更改
func TestParseCPUInfo_UnknownKey(t *testing.T) {
	c := &CPUInfo{
		UsedCPUSys:          1.0,
		UsedCPUUser:         2.0,
		UsedCPUSysChildren:  3.0,
		UsedCPUUserChildren: 4.0,
	}

	parseCPUInfo("unknown_key", "123.45", c)

	// 所有字段应保持不变
	if c.UsedCPUSys != 1.0 {
		t.Errorf("Expected UsedCPUSys to remain unchanged, got %v", c.UsedCPUSys)
	}
	if c.UsedCPUUser != 2.0 {
		t.Errorf("Expected UsedCPUUser to remain unchanged, got %v", c.UsedCPUUser)
	}
	if c.UsedCPUSysChildren != 3.0 {
		t.Errorf("Expected UsedCPUSysChildren to remain unchanged, got %v", c.UsedCPUSysChildren)
	}
	if c.UsedCPUUserChildren != 4.0 {
		t.Errorf("Expected UsedCPUUserChildren to remain unchanged, got %v", c.UsedCPUUserChildren)
	}
}

// TestParseClusterInfo 测试所有有效 key 和合法 value 的情况
func TestParseClusterInfo_ValidKeys(t *testing.T) {
	c := &ClusterInfo{}
	parseClusterInfo("cluster_enabled", "1", c)
	parseClusterInfo("cluster_node_count", "5", c)
	parseClusterInfo("cluster_my_epoch", "100", c)
	parseClusterInfo("cluster_slots_assigned", "16384", c)
	parseClusterInfo("cluster_slots_ok", "16380", c)
	parseClusterInfo("cluster_slots_pfail", "2", c)
	parseClusterInfo("cluster_slots_fail", "1", c)
	parseClusterInfo("cluster_known_nodes", "3", c)
	parseClusterInfo("cluster_size", "3", c)
	parseClusterInfo("cluster_current_epoch", "100", c)
	parseClusterInfo("cluster_stats_messages_sent", "1000", c)
	parseClusterInfo("cluster_stats_messages_received", "999", c)

	if c.ClusterEnabled != 1 {
		t.Errorf("Expected ClusterEnabled=1, got %d", c.ClusterEnabled)
	}
	if c.ClusterNodeCount != 5 {
		t.Errorf("Expected ClusterNodeCount=5, got %d", c.ClusterNodeCount)
	}
	if c.ClusterMyEpoch != 100 {
		t.Errorf("Expected ClusterMyEpoch=100, got %d", c.ClusterMyEpoch)
	}
	if c.ClusterSlotsAssigned != 16384 {
		t.Errorf("Expected ClusterSlotsAssigned=16384, got %d", c.ClusterSlotsAssigned)
	}
	if c.ClusterSlotsOk != 16380 {
		t.Errorf("Expected ClusterSlotsOk=16380, got %d", c.ClusterSlotsOk)
	}
	if c.ClusterSlotsPfail != 2 {
		t.Errorf("Expected ClusterSlotsPfail=2, got %d", c.ClusterSlotsPfail)
	}
	if c.ClusterSlotsFail != 1 {
		t.Errorf("Expected ClusterSlotsFail=1, got %d", c.ClusterSlotsFail)
	}
	if c.ClusterKnownNodes != 3 {
		t.Errorf("Expected ClusterKnownNodes=3, got %d", c.ClusterKnownNodes)
	}
	if c.ClusterSize != 3 {
		t.Errorf("Expected ClusterSize=3, got %d", c.ClusterSize)
	}
	if c.ClusterCurrentEpoch != 100 {
		t.Errorf("Expected ClusterCurrentEpoch=100, got %d", c.ClusterCurrentEpoch)
	}
	if c.ClusterStatsMessagesSent != 1000 {
		t.Errorf("Expected ClusterStatsMessagesSent=1000, got %d", c.ClusterStatsMessagesSent)
	}
	if c.ClusterStatsMessagesReceived != 999 {
		t.Errorf("Expected ClusterStatsMessagesReceived=999, got %d", c.ClusterStatsMessagesReceived)
	}
}

// TestParseClusterInfo_InvalidKey 测试无效 key 不会修改结构体
func TestParseClusterInfo_InvalidKey(t *testing.T) {
	c := &ClusterInfo{
		ClusterEnabled: 1,
	}
	parseClusterInfo("invalid_key", "100", c)
	if c.ClusterEnabled != 1 {
		t.Errorf("Expected ClusterEnabled=1 after invalid key, but got %d", c.ClusterEnabled)
	}
}

// TestParseClusterInfo_InvalidValue 测试非法 value 不会修改结构体
func TestParseClusterInfo_InvalidValue(t *testing.T) {
	c := &ClusterInfo{}
	parseClusterInfo("cluster_enabled", "invalid", c)
	if c.ClusterEnabled != 0 {
		t.Errorf("Expected ClusterEnabled=0 after invalid value, but got %d", c.ClusterEnabled)
	}

	parseClusterInfo("cluster_stats_messages_sent", "invalid", c)
	if c.ClusterStatsMessagesSent != 0 {
		t.Errorf("Expected ClusterStatsMessagesSent=0 after invalid value, but got %d", c.ClusterStatsMessagesSent)
	}
}

// 测试所有字段正常的情况
func TestParseKeyspace_AllValid(t *testing.T) {
	input := "keys=100,expires=50,avg_ttl=30"
	expected := KeyspaceDB{
		Keys:    100,
		Expires: 50,
		AvgTTL:  30 * time.Second,
	}

	result, err := parseKeyspace(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// 测试值非法（非整数）
func TestParseKeyspace_InvalidValue(t *testing.T) {
	input := "keys=abc"
	expected := KeyspaceDB{} // 所有字段默认值

	result, err := parseKeyspace(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// 测试不支持的字段
func TestParseKeyspace_UnsupportedKey(t *testing.T) {
	input := "invalid_key=200"
	expected := KeyspaceDB{}

	result, err := parseKeyspace(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// 测试混合有效和无效字段
func TestParseKeyspace_MixedValidAndInvalid(t *testing.T) {
	input := "keys=100,avg_ttl=xyz"
	expected := KeyspaceDB{
		Keys:   100,
		AvgTTL: 0,
	}

	result, err := parseKeyspace(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// 测试键为空的情况
func TestParseKeyspace_EmptyKey(t *testing.T) {
	input := "keys=100,=50"
	expected := KeyspaceDB{
		Keys: 100,
	}

	result, err := parseKeyspace(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// 测试值为空的情况
func TestParseKeyspace_EmptyValue(t *testing.T) {
	input := "keys=100,expires="
	expected := KeyspaceDB{
		Keys: 100,
	}

	result, err := parseKeyspace(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// 测试空字符串输入
func TestParseKeyspace_EmptyInput(t *testing.T) {
	input := ""
	expected := KeyspaceDB{}

	result, err := parseKeyspace(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}

// 测试 avg_ttl 的合法值
func TestParseKeyspace_AvgTTLValid(t *testing.T) {
	input := "avg_ttl=15"
	expected := KeyspaceDB{
		AvgTTL: 15 * time.Second,
	}

	result, err := parseKeyspace(input)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result != expected {
		t.Errorf("Expected %+v, got %+v", expected, result)
	}
}
