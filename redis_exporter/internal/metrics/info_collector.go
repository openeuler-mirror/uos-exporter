package metrics

import (
	"context"
	"fmt"

	redis "github.com/go-redis/redis/v8"
	"github.com/prometheus/client_golang/prometheus"
)

func newInfoCollector(client *redis.Client) *infoCollector {
	return &infoCollector{
		client:  client,
		metrics: newInfoMetrics(),
	}
}

type infoMetrics struct {
	// Up
	upMetric *baseMetrics

	// Server 部分
	serverUptimeInSecondsMetric  *baseMetrics
	serverUptimeInDaysMetric     *baseMetrics
	serverPIDMetric              *baseMetrics
	serverTCPPortMetric          *baseMetrics
	serverUptimeSinceForkMetric  *baseMetrics
	serverOSMetric               *baseMetrics
	serverArchBitsMetric         *baseMetrics
	serverMultiplexingAPIMetric  *baseMetrics
	serverAtomicAPIMetric        *baseMetrics
	serverGCCVersionMetric       *baseMetrics
	serverProcessIDMetric        *baseMetrics
	serverRunIDMetric            *baseMetrics
	serverRedisExecutableMetric  *baseMetrics
	serverRedisCommandLineMetric *baseMetrics
	serverLuaEngineMetric        *baseMetrics

	// Clients 部分
	connectedClientsMetric        *baseMetrics
	clientLongestOutputListMetric *baseMetrics
	clientBiggestInputBufMetric   *baseMetrics
	blockedClientsMetric          *baseMetrics
	maxInputBufferMetric          *baseMetrics
	maxOutputBufferMetric         *baseMetrics
	maxOutputListpackLengthMetric *baseMetrics

	// Memory 部分
	usedMemoryMetric            *baseMetrics
	usedMemoryHumanMetric       *baseMetrics // string 类型需特殊处理
	usedMemoryRssMetric         *baseMetrics
	usedMemoryPeakMetric        *baseMetrics
	usedMemoryLuaMetric         *baseMetrics
	usedMemoryOverheadMetric    *baseMetrics
	usedMemoryStartupMetric     *baseMetrics
	usedMemoryDatasetMetric     *baseMetrics
	usedMemoryFragmentedMetric  *baseMetrics
	totalSystemMemoryMetric     *baseMetrics
	usedHugePagesMetric         *baseMetrics
	maxMemoryMetric             *baseMetrics
	maxMemoryPolicyMetric       *baseMetrics // string 类型需特殊处理
	maxMemorySlabMetric         *baseMetrics
	memFragmentationRatioMetric *baseMetrics
	memAllocatorMetric          *baseMetrics // string 类型需特殊处理

	// Persistence 部分
	rdbChangesSinceLastSaveMetric  *baseMetrics
	rdbBgsaveInProgressMetric      *baseMetrics
	rdbLastSaveTimeMetric          *baseMetrics
	rdbLastBgsaveStatusMetric      *baseMetrics // string 类型需特殊处理
	rdbLastBgsaveTimeSecMetric     *baseMetrics
	rdbCurrentBgsaveTimeSecMetric  *baseMetrics
	aofEnabledMetric               *baseMetrics
	aofRewriteInProgressMetric     *baseMetrics
	aofRewriteScheduledMetric      *baseMetrics
	aofLastRewriteTimeSecMetric    *baseMetrics
	aofCurrentRewriteTimeSecMetric *baseMetrics
	aofLastBgrewriteStatusMetric   *baseMetrics // string 类型需特殊处理
	aofLastWriteStatusMetric       *baseMetrics // string 类型需特殊处理
	aofPendingRewriteMetric        *baseMetrics
	aofBufferLengthMetric          *baseMetrics
	aofPendingBioFsyncMetric       *baseMetrics
	aofDelayedFsyncMetric          *baseMetrics

	// Stats 部分
	totalConnectionsReceivedMetric *baseMetrics
	totalCommandsProcessedMetric   *baseMetrics
	instantaneousOpsPerSecMetric   *baseMetrics
	rejectedConnectionsMetric      *baseMetrics
	evictedKeysMetric              *baseMetrics
	keyspaceMissesMetric           *baseMetrics
	keysapceHitRateMetric          *baseMetrics
	hashMaxZiplistValueMetric      *baseMetrics
	hashMaxZiplistEntriesMetric    *baseMetrics
	pubsubChannelsMetric           *baseMetrics
	pubsubPatternsMetric           *baseMetrics
	latestForkUsecMetric           *baseMetrics
	migrateCachedSocketsMetric     *baseMetrics
	slaveExpiresTrackedKeysMetric  *baseMetrics
	activeDefragHitsMetric         *baseMetrics
	activeDefragMissesMetric       *baseMetrics
	activeDefragKeyHitsMetric      *baseMetrics
	activeDefragKeyMissesMetric    *baseMetrics

	// Replication 部分
	replicationRoleMetric            *baseMetrics // string 类型需特殊处理
	connectedSlavesMetric            *baseMetrics
	masterReplOffsetMetric           *baseMetrics
	replBacklogActiveMetric          *baseMetrics
	replBacklogSizeMetric            *baseMetrics
	replBacklogFirstByteOffsetMetric *baseMetrics
	replBacklogHistlenMetric         *baseMetrics
	minSlavesGoodSlavesMetric        *baseMetrics
	minSlavesScoutReportIntrMetric   *baseMetrics

	// CPU 部分
	usedCpuSysMetric          *baseMetrics
	usedCpuUserMetric         *baseMetrics
	usedCpuSysChildrenMetric  *baseMetrics
	usedCpuUserChildrenMetric *baseMetrics

	// Cluster 部分
	clusterEnabledMetric               *baseMetrics
	clusterNodeCountMetric             *baseMetrics
	clusterMyEpochMetric               *baseMetrics
	clusterSlotsAssignedMetric         *baseMetrics
	clusterSlotsOkMetric               *baseMetrics
	clusterSlotsPfailMetric            *baseMetrics
	clusterSlotsFailMetric             *baseMetrics
	clusterKnownNodesMetric            *baseMetrics
	clusterSizeMetric                  *baseMetrics
	clusterCurrentEpochMetric          *baseMetrics
	clusterStatsMessagesSentMetric     *baseMetrics
	clusterStatsMessagesReceivedMetric *baseMetrics

	// Keyspace 部分（带 label）
	keyspaceKeysMetric    *baseMetrics
	keyspaceExpiresMetric *baseMetrics
	keyspaceAvgTTLMetric  *baseMetrics

	// Modules 部分（label: module）
	moduleLoadedMetric *baseMetrics
}

type infoCollector struct {
	client  *redis.Client
	metrics *infoMetrics
}

// func (e *RedisExporter) registerMetrics() {
func newInfoMetrics() *infoMetrics {
	return &infoMetrics{
		upMetric: NewMetrics(
			"redis_up",
			"Whether the Redis instance is up (1) or down (0).",
			nil,
		),

		// -------------------
		// Server Section
		// -------------------
		serverUptimeInSecondsMetric: NewMetrics(
			"redis_server_uptime_in_seconds",
			"The number of seconds since Redis server started.",
			nil,
		),
		serverUptimeInDaysMetric: NewMetrics(
			"redis_server_uptime_in_days",
			"The number of days since Redis server started.",
			nil,
		),
		serverPIDMetric: NewMetrics(
			"redis_server_process_id",
			"The process ID of Redis server.",
			nil,
		),
		serverTCPPortMetric: NewMetrics(
			"redis_server_tcp_port",
			"The TCP port on which Redis listens.",
			nil,
		),
		serverUptimeSinceForkMetric: NewMetrics(
			"redis_server_uptime_since_fork",
			"The number of seconds since the last fork() operation.",
			nil,
		),
		serverOSMetric: NewMetrics(
			"redis_server_os",
			"The operating system Redis is running on.",
			nil,
		),
		serverArchBitsMetric: NewMetrics(
			"redis_server_arch_bits",
			"The architecture bit width (32 or 64 bits).",
			nil,
		),
		serverMultiplexingAPIMetric: NewMetrics(
			"redis_server_multiplexing_api",
			"The I/O multiplexing API used by Redis.",
			nil,
		),
		serverAtomicAPIMetric: NewMetrics(
			"redis_server_atomic_api",
			"The atomic API used by Redis.",
			nil,
		),
		serverGCCVersionMetric: NewMetrics(
			"redis_server_gcc_version",
			"The version of the GCC compiler used to compile Redis.",
			nil,
		),
		serverProcessIDMetric: NewMetrics(
			"redis_server_process_id_string",
			"The process ID as a string (deprecated, use redis_server_process_id).",
			nil,
		),
		serverRunIDMetric: NewMetrics(
			"redis_server_run_id",
			"A unique run ID for this Redis instanc",
			nil,
		),
		serverRedisExecutableMetric: NewMetrics(
			"redis_server_executable_path",
			"The full path to the Redis executabl",
			nil,
		),
		serverRedisCommandLineMetric: NewMetrics(
			"redis_server_command_line",
			"The command line arguments passed to Redis at startup.",
			nil,
		),
		serverLuaEngineMetric: NewMetrics(
			"redis_server_lua_engine",
			"The version of the Lua engine used by Redis.",
			nil,
		),

		// -------------------
		// Clients Section
		// -------------------
		connectedClientsMetric: NewMetrics(
			"redis_connected_clients",
			"The number of client connections (excluding slaves).",
			nil,
		),
		clientLongestOutputListMetric: NewMetrics(
			"redis_client_longest_output_list",
			"The length of the longest client output list.",
			nil,
		),
		clientBiggestInputBufMetric: NewMetrics(
			"redis_client_biggest_input_buf",
			"The size of the largest input buffer among clients.",
			nil,
		),
		blockedClientsMetric: NewMetrics(
			"redis_blocked_clients",
			"The number of clients waiting on a blocking call.",
			nil,
		),
		maxInputBufferMetric: NewMetrics(
			"redis_client_max_input_buffer",
			"The maximum input buffer size allowed per client.",
			nil,
		),
		maxOutputBufferMetric: NewMetrics(
			"redis_client_max_output_buffer",
			"The maximum output buffer size allowed per client.",
			nil,
		),
		maxOutputListpackLengthMetric: NewMetrics(
			"redis_client_max_output_listpack_length",
			"The maximum length of listpacks in output buffers.",
			nil,
		),

		// -------------------
		// Memory Section
		// -------------------
		usedMemoryMetric: NewMetrics(
			"redis_memory_used_bytes",
			"The total number of bytes allocated by Redis using its allocator.",
			nil,
		),
		usedMemoryHumanMetric: NewMetrics(
			"redis_memory_used_human",
			"The total memory used in human-readable format (not numeric).",
			nil,
		),
		usedMemoryRssMetric: NewMetrics(
			"redis_memory_used_rss_bytes",
			"The number of bytes that Redis allocated as seen by the OS.",
			nil,
		),
		usedMemoryPeakMetric: NewMetrics(
			"redis_memory_peak_bytes",
			"The peak memory used by Redis.",
			nil,
		),
		usedMemoryLuaMetric: NewMetrics(
			"redis_memory_lua_bytes",
			"The number of bytes used by Lua scripts.",
			nil,
		),
		usedMemoryOverheadMetric: NewMetrics(
			"redis_memory_overhead_bytes",
			"The amount of overhead memory used by Redis.",
			nil,
		),
		usedMemoryStartupMetric: NewMetrics(
			"redis_memory_startup_bytes",
			"The initial memory used at startup.",
			nil,
		),
		usedMemoryDatasetMetric: NewMetrics(
			"redis_memory_dataset_bytes",
			"The size of the dataset in memory.",
			nil,
		),
		usedMemoryFragmentedMetric: NewMetrics(
			"redis_memory_fragmented_bytes",
			"The fragmented memory not yet reclaimed by the system.",
			nil,
		),
		totalSystemMemoryMetric: NewMetrics(
			"redis_system_total_memory_bytes",
			"The total memory available in the system.",
			nil,
		),
		usedHugePagesMetric: NewMetrics(
			"redis_memory_used_huge_pages_bytes",
			"The memory used by Redis via Huge Pages.",
			nil,
		),
		maxMemoryMetric: NewMetrics(
			"redis_maxmemory_bytes",
			"The maximum memory Redis can use before evicting keys.",
			nil,
		),
		maxMemoryPolicyMetric: NewMetrics(
			"redis_maxmemory_policy",
			"The eviction policy when maxmemory limit is reached.",
			nil,
		),
		maxMemorySlabMetric: NewMetrics(
			"redis_maxmemory_slave_bytes",
			"The amount of memory used by slave connections.",
			nil,
		),
		memFragmentationRatioMetric: NewMetrics(
			"redis_memory_fragmentation_ratio",
			"The ratio of used_memory_rss / used_memory.",
			nil,
		),
		memAllocatorMetric: NewMetrics(
			"redis_memory_allocator",
			"The memory allocator used by Redis (g. jemalloc).",
			nil,
		),

		// -------------------
		// Persistence Section
		// -------------------
		rdbChangesSinceLastSaveMetric: NewMetrics(
			"redis_persistence_rdb_changes_since_last_save",
			"The number of changes since the last RDB sav",
			nil,
		),
		rdbBgsaveInProgressMetric: NewMetrics(
			"redis_persistence_rdb_bgsave_in_progress",
			"Flag indicating if a background save is in progress.",
			nil,
		),
		rdbLastSaveTimeMetric: NewMetrics(
			"redis_persistence_rdb_last_save_timestamp",
			"The Unix timestamp of the last successful RDB sav",
			nil,
		),
		rdbLastBgsaveStatusMetric: NewMetrics(
			"redis_persistence_rdb_last_bgsave_status",
			"The status of the last RDB background save (ok/err).",
			nil,
		),
		rdbLastBgsaveTimeSecMetric: NewMetrics(
			"redis_persistence_rdb_last_bgsave_time_sec",
			"The duration of the last RDB background save in seconds.",
			nil,
		),
		rdbCurrentBgsaveTimeSecMetric: NewMetrics(
			"redis_persistence_rdb_current_bgsave_time_sec",
			"The current duration of an ongoing RDB background save in seconds.",
			nil,
		),
		aofEnabledMetric: NewMetrics(
			"redis_persistence_aof_enabled",
			"Flag indicating whether AOF mode is enabled.",
			nil,
		),
		aofRewriteInProgressMetric: NewMetrics(
			"redis_persistence_aof_rewrite_in_progress",
			"Flag indicating whether an AOF rewrite is in progress.",
			nil,
		),
		aofRewriteScheduledMetric: NewMetrics(
			"redis_persistence_aof_rewrite_scheduled",
			"Flag indicating whether an AOF rewrite is scheduled.",
			nil,
		),
		aofLastRewriteTimeSecMetric: NewMetrics(
			"redis_persistence_aof_last_rewrite_time_sec",
			"The duration of the last AOF rewrite in seconds.",
			nil,
		),
		aofCurrentRewriteTimeSecMetric: NewMetrics(
			"redis_persistence_aof_current_rewrite_time_sec",
			"The current duration of an ongoing AOF rewrite in seconds.",
			nil,
		),
		aofLastBgrewriteStatusMetric: NewMetrics(
			"redis_persistence_aof_last_bgrewrite_status",
			"The status of the last AOF background rewrite (ok/err).",
			nil,
		),
		aofLastWriteStatusMetric: NewMetrics(
			"redis_persistence_aof_last_write_status",
			"The result of the last write to the AOF file (ok/err).",
			nil,
		),
		aofPendingRewriteMetric: NewMetrics(
			"redis_persistence_aof_pending_rewrite",
			"Flag indicating whether a rewrite is pending.",
			nil,
		),
		aofBufferLengthMetric: NewMetrics(
			"redis_persistence_aof_buffer_length",
			"The current size of the AOF buffer.",
			nil,
		),
		aofPendingBioFsyncMetric: NewMetrics(
			"redis_persistence_aof_pending_bio_fsync",
			"The number of pending fsync operations in AOF.",
			nil,
		),
		aofDelayedFsyncMetric: NewMetrics(
			"redis_persistence_aof_delayed_fsync",
			"The number of delayed fsync operations in AOF.",
			nil,
		),

		// -------------------
		// Stats Section
		// -------------------
		totalConnectionsReceivedMetric: NewMetrics(
			"redis_total_connections_received",
			"The total number of connections accepted by the server.",
			nil,
		),
		totalCommandsProcessedMetric: NewMetrics(
			"redis_total_commands_processed",
			"The total number of commands processed by the server.",
			nil,
		),
		instantaneousOpsPerSecMetric: NewMetrics(
			"redis_instantaneous_ops_per_sec",
			"The number of commands processed per second.",
			nil,
		),
		rejectedConnectionsMetric: NewMetrics(
			"redis_rejected_connections",
			"The number of rejected connections due to maxclients limit.",
			nil,
		),
		evictedKeysMetric: NewMetrics(
			"redis_evicted_keys",
			"The number of evicted keys due to maxmemory limit.",
			nil,
		),
		keyspaceMissesMetric: NewMetrics(
			"redis_keyspace_misses",
			"The number of failed lookup in the main dictionary.",
			nil,
		),
		keysapceHitRateMetric: NewMetrics(
			"redis_keyspace_hit_rate",
			"The cache hit rate for key lookups.",
			nil,
		),
		hashMaxZiplistValueMetric: NewMetrics(
			"redis_hash_max_ziplist_value",
			"The maximum size of hash value stored in ziplist.",
			nil,
		),
		hashMaxZiplistEntriesMetric: NewMetrics(
			"redis_hash_max_ziplist_entries",
			"The maximum number of entries in hash ziplist.",
			nil,
		),
		pubsubChannelsMetric: NewMetrics(
			"redis_pubsub_channels",
			"The number of active Pub/Sub channels.",
			nil,
		),
		pubsubPatternsMetric: NewMetrics(
			"redis_pubsub_patterns",
			"The number of active Pub/Sub pattern subscriptions.",
			nil,
		),
		latestForkUsecMetric: NewMetrics(
			"redis_latest_fork_usec",
			"The duration of the latest fork() operation in microseconds.",
			nil,
		),
		migrateCachedSocketsMetric: NewMetrics(
			"redis_migrate_cached_sockets",
			"The number of cached sockets used for MIGRATE command.",
			nil,
		),
		slaveExpiresTrackedKeysMetric: NewMetrics(
			"redis_slave_expires_tracked_keys",
			"The number of keys tracked for expiration in replication.",
			nil,
		),
		activeDefragHitsMetric: NewMetrics(
			"redis_active_defrag_hits",
			"The number of allocations successfully defragmented.",
			nil,
		),
		activeDefragMissesMetric: NewMetrics(
			"redis_active_defrag_misses",
			"The number of allocations skipped during defragmentation.",
			nil,
		),
		activeDefragKeyHitsMetric: NewMetrics(
			"redis_active_defrag_key_hits",
			"The number of keys successfully defragmented.",
			nil,
		),
		activeDefragKeyMissesMetric: NewMetrics(
			"redis_active_defrag_key_misses",
			"The number of keys skipped during defragmentation.",
			nil,
		),

		// -------------------
		// Replication Section
		// -------------------
		replicationRoleMetric: NewMetrics(
			"redis_replication_role",
			"The role of the instance ('master' or 'slave').",
			nil,
		),
		connectedSlavesMetric: NewMetrics(
			"redis_connected_slaves",
			"The number of connected replicas/slaves.",
			nil,
		),
		masterReplOffsetMetric: NewMetrics(
			"redis_master_repl_offset",
			"The master replication offset.",
			nil,
		),
		replBacklogActiveMetric: NewMetrics(
			"redis_repl_backlog_active",
			"Flag indicating if the replication backlog is activ",
			nil,
		),
		replBacklogSizeMetric: NewMetrics(
			"redis_repl_backlog_size",
			"The size of the replication backlog buffer.",
			nil,
		),
		replBacklogFirstByteOffsetMetric: NewMetrics(
			"redis_repl_backlog_first_byte_offset",
			"The offset of the first byte in the replication backlog.",
			nil,
		),
		replBacklogHistlenMetric: NewMetrics(
			"redis_repl_backlog_histlen",
			"The length of data in the replication backlog buffer.",
			nil,
		),
		minSlavesGoodSlavesMetric: NewMetrics(
			"redis_min_slaves_good_slaves",
			"The number of slaves meeting min-slaves-to-write requirements.",
			nil,
		),
		minSlavesScoutReportIntrMetric: NewMetrics(
			"redis_min_slaves_scout_report_interval",
			"The interval between checks for min-slaves requirements.",
			nil,
		),

		// -------------------
		// CPU Section
		// -------------------
		usedCpuSysMetric: NewMetrics(
			"redis_cpu_sys_seconds_total",
			"The system CPU time consumed by Redis (in seconds).",
			nil,
		),
		usedCpuUserMetric: NewMetrics(
			"redis_cpu_user_seconds_total",
			"The user CPU time consumed by Redis (in seconds).",
			nil,
		),
		usedCpuSysChildrenMetric: NewMetrics(
			"redis_cpu_sys_children_seconds_total",
			"The system CPU time used by Redis background processes.",
			nil,
		),
		usedCpuUserChildrenMetric: NewMetrics(
			"redis_cpu_user_children_seconds_total",
			"The user CPU time used by Redis background processes.",
			nil,
		),

		// -------------------
		// Cluster Section
		// -------------------
		clusterEnabledMetric: NewMetrics(
			"redis_cluster_enabled",
			"Flag indicating whether cluster mode is enabled.",
			nil,
		),
		clusterNodeCountMetric: NewMetrics(
			"redis_cluster_node_count",
			"The number of nodes in the Redis cluster.",
			nil,
		),
		clusterMyEpochMetric: NewMetrics(
			"redis_cluster_my_epoch",
			"The epoch of the current node in the Redis cluster.",
			nil,
		),
		clusterSlotsAssignedMetric: NewMetrics(
			"redis_cluster_slots_assigned",
			"The number of slots assigned to the Redis cluster.",
			nil,
		),
		clusterSlotsOkMetric: NewMetrics(
			"redis_cluster_slots_ok",
			"The number of slots in OK stat",
			nil,
		),
		clusterSlotsPfailMetric: NewMetrics(
			"redis_cluster_slots_pfail",
			"The number of slots in PFAIL stat",
			nil,
		),
		clusterSlotsFailMetric: NewMetrics(
			"redis_cluster_slots_fail",
			"The number of slots in FAIL stat",
			nil,
		),
		clusterKnownNodesMetric: NewMetrics(
			"redis_cluster_known_nodes",
			"The number of nodes known to the Redis cluster.",
			nil,
		),
		clusterSizeMetric: NewMetrics(
			"redis_cluster_size",
			"The number of master nodes in the Redis cluster.",
			nil,
		),
		clusterCurrentEpochMetric: NewMetrics(
			"redis_cluster_current_epoch",
			"The current epoch in the Redis cluster.",
			nil,
		),
		clusterStatsMessagesSentMetric: NewMetrics(
			"redis_cluster_stats_messages_sent_total",
			"The total number of messages sent in the Redis cluster.",
			nil,
		),
		clusterStatsMessagesReceivedMetric: NewMetrics(
			"redis_cluster_stats_messages_received_total",
			"The total number of messages received in the Redis cluster.",
			nil,
		),

		// -------------------
		// Keyspace Section (with label {db="db0"})
		// -------------------
		keyspaceKeysMetric: NewMetrics(
			"redis_keyspace_keys",
			"The number of keys in a Redis databas",
			[]string{"db"},
		),
		keyspaceExpiresMetric: NewMetrics(
			"redis_keyspace_expires",
			"The number of keys with an expire set in a Redis databas",
			[]string{"db"},
		),
		keyspaceAvgTTLMetric: NewMetrics(
			"redis_keyspace_avg_ttl_seconds",
			"The average TTL of keys in a Redis database (in seconds).",
			[]string{"db"},
		),

		// -------------------
		// Modules Section (with label {module="name"})
		// -------------------
		moduleLoadedMetric: NewMetrics(
			"redis_module_loaded",
			"Flag indicating whether a module is loaded (1) or not (0).",
			[]string{"module"},
		),
	}
}

func (c *infoCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()
	infoStr, err := c.client.Info(ctx).Result()

	var up float64 = 1
	if err != nil {
		up = 0
	}
	c.metrics.upMetric.collect(
		ch,
		up,
		nil,
	)

	if up == 0 {
		return
	}

	redisInfo, err := parseRedisInfo(infoStr)
	if err != nil {
		fmt.Println("Error parsing Redis info:", err)
		return
	}

	// -------------------
	// Server Section
	// -------------------
	c.metrics.serverUptimeInSecondsMetric.collect(
		ch,
		float64(redisInfo.Server.UptimeInSeconds),
		nil,
	)
	c.metrics.serverUptimeInDaysMetric.collect(
		ch,
		float64(redisInfo.Server.UptimeInDays),
		nil,
	)
	c.metrics.serverPIDMetric.collect(
		ch,
		float64(redisInfo.Server.PID),
		nil,
	)
	c.metrics.serverTCPPortMetric.collect(
		ch,
		float64(redisInfo.Server.TCPPort),
		nil,
	)
	c.metrics.serverUptimeSinceForkMetric.collect(
		ch,
		float64(redisInfo.Server.UptimeSinceFork),
		nil,
	)
	c.metrics.serverOSMetric.collect(
		ch,
		float64(0),
		nil,
	) // string 类型，可选做 info metric
	c.metrics.serverArchBitsMetric.collect(
		ch,
		float64(redisInfo.Server.ArchBits),
		nil,
	)
	c.metrics.serverMultiplexingAPIMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.serverAtomicAPIMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.serverGCCVersionMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.serverProcessIDMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.serverRunIDMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.serverRedisExecutableMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.serverRedisCommandLineMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.serverLuaEngineMetric.collect(
		ch,
		float64(0),
		nil,
	)

	// -------------------
	// Clients Section
	// -------------------
	c.metrics.connectedClientsMetric.collect(
		ch,
		float64(redisInfo.Clients.ConnectedClients),
		nil,
	)
	c.metrics.clientLongestOutputListMetric.collect(
		ch,
		float64(redisInfo.Clients.ClientLongestOutputList),
		nil,
	)
	c.metrics.clientBiggestInputBufMetric.collect(
		ch,
		float64(redisInfo.Clients.ClientBiggestInputBuf),
		nil,
	)
	c.metrics.blockedClientsMetric.collect(
		ch,
		float64(redisInfo.Clients.BlockedClients),
		nil,
	)
	c.metrics.maxInputBufferMetric.collect(
		ch,
		float64(redisInfo.Clients.MaxInputBuffer),
		nil,
	)
	c.metrics.maxOutputBufferMetric.collect(
		ch,
		float64(redisInfo.Clients.MaxOutputBuffer),
		nil,
	)
	c.metrics.maxOutputListpackLengthMetric.collect(
		ch,
		float64(redisInfo.Clients.MaxOutputListLength),
		nil,
	)

	// -------------------
	// Memory Section
	// -------------------
	c.metrics.usedMemoryMetric.collect(
		ch,
		redisInfo.Memory.UsedMemory,
		nil,
	)
	c.metrics.usedMemoryHumanMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.usedMemoryRssMetric.collect(
		ch,
		redisInfo.Memory.UsedMemoryRss,
		nil,
	)
	c.metrics.usedMemoryPeakMetric.collect(
		ch,
		redisInfo.Memory.UsedMemoryPeak,
		nil,
	)
	c.metrics.usedMemoryLuaMetric.collect(
		ch,
		redisInfo.Memory.UsedMemoryLua,
		nil,
	)
	c.metrics.usedMemoryOverheadMetric.collect(
		ch,
		redisInfo.Memory.UsedMemoryOverhead,
		nil,
	)
	c.metrics.usedMemoryStartupMetric.collect(
		ch,
		redisInfo.Memory.UsedMemoryStartup,
		nil,
	)
	c.metrics.usedMemoryDatasetMetric.collect(
		ch,
		redisInfo.Memory.UsedMemoryDataset,
		nil,
	)
	c.metrics.usedMemoryFragmentedMetric.collect(
		ch,
		redisInfo.Memory.UsedMemoryFragmented,
		nil,
	)
	c.metrics.totalSystemMemoryMetric.collect(
		ch,
		redisInfo.Memory.TotalSystemMemory,
		nil,
	)
	c.metrics.usedHugePagesMetric.collect(
		ch,
		redisInfo.Memory.UsedHugePages,
		nil,
	)
	c.metrics.maxMemoryMetric.collect(
		ch,
		redisInfo.Memory.MaxMemory,
		nil,
	)
	c.metrics.maxMemoryPolicyMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.maxMemorySlabMetric.collect(
		ch,
		redisInfo.Memory.MaxMemorySlab,
		nil,
	)
	c.metrics.memFragmentationRatioMetric.collect(
		ch,
		redisInfo.Memory.MemFragmentation,
		nil,
	)
	c.metrics.memAllocatorMetric.collect(
		ch,
		float64(0),
		nil,
	)

	// -------------------
	// Persistence Section
	// -------------------
	c.metrics.rdbChangesSinceLastSaveMetric.collect(
		ch,
		float64(redisInfo.Persistence.RdbChangesSinceLastSave),
		nil,
	)
	c.metrics.rdbBgsaveInProgressMetric.collect(
		ch,
		float64(redisInfo.Persistence.RdbBgsaveInProgress),
		nil,
	)
	c.metrics.rdbLastSaveTimeMetric.collect(
		ch,
		float64(redisInfo.Persistence.RdbLastSaveTime.Unix()),
		nil,
	)
	c.metrics.rdbLastBgsaveStatusMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.rdbLastBgsaveTimeSecMetric.collect(
		ch,
		float64(redisInfo.Persistence.RdbLastBgsaveTimeSec),
		nil,
	)
	c.metrics.rdbCurrentBgsaveTimeSecMetric.collect(
		ch,
		float64(redisInfo.Persistence.RdbCurrentBgsaveTimeSec),
		nil,
	)
	c.metrics.aofEnabledMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofEnabled),
		nil,
	)
	c.metrics.aofRewriteInProgressMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofRewriteInProgress),
		nil,
	)
	c.metrics.aofRewriteScheduledMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofRewriteScheduled),
		nil,
	)
	c.metrics.aofLastRewriteTimeSecMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofLastRewriteTimeSec),
		nil,
	)
	c.metrics.aofCurrentRewriteTimeSecMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofCurrentRewriteTimeSec),
		nil,
	)
	c.metrics.aofLastBgrewriteStatusMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.aofLastWriteStatusMetric.collect(
		ch,
		float64(0),
		nil,
	)
	c.metrics.aofPendingRewriteMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofPendingRewrite),
		nil,
	)
	c.metrics.aofBufferLengthMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofBufferLength),
		nil,
	)
	c.metrics.aofPendingBioFsyncMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofPendingBioFsync),
		nil,
	)
	c.metrics.aofDelayedFsyncMetric.collect(
		ch,
		float64(redisInfo.Persistence.AofDelayedFsync),
		nil,
	)

	// -------------------
	// Stats Section
	// -------------------
	c.metrics.totalConnectionsReceivedMetric.collect(
		ch,
		float64(redisInfo.Stats.TotalConnectionsReceived),
		nil,
	)
	c.metrics.totalCommandsProcessedMetric.collect(
		ch,
		float64(redisInfo.Stats.TotalCommandsProcessed),
		nil,
	)
	c.metrics.instantaneousOpsPerSecMetric.collect(
		ch,
		float64(redisInfo.Stats.InstantaneousOpsPerSec),
		nil,
	)
	c.metrics.rejectedConnectionsMetric.collect(
		ch,
		float64(redisInfo.Stats.RejectedConnections),
		nil,
	)
	c.metrics.evictedKeysMetric.collect(
		ch,
		float64(redisInfo.Stats.EvictedKeys),
		nil,
	)
	c.metrics.keyspaceMissesMetric.collect(
		ch,
		float64(redisInfo.Stats.KeyspaceMisses),
		nil,
	)
	c.metrics.keysapceHitRateMetric.collect(
		ch,
		redisInfo.Stats.KeyspaceHitRate,
		nil,
	)
	c.metrics.hashMaxZiplistValueMetric.collect(
		ch,
		float64(redisInfo.Stats.HashMaxZiplistValue),
		nil,
	)
	c.metrics.hashMaxZiplistEntriesMetric.collect(
		ch,
		float64(redisInfo.Stats.HashMaxZiplistEntries),
		nil,
	)
	c.metrics.pubsubChannelsMetric.collect(
		ch,
		float64(redisInfo.Stats.PubsubChannels),
		nil,
	)
	c.metrics.pubsubPatternsMetric.collect(
		ch,
		float64(redisInfo.Stats.PubsubPatterns),
		nil,
	)
	c.metrics.latestForkUsecMetric.collect(
		ch,
		float64(redisInfo.Stats.LatestForkUsec),
		nil,
	)
	c.metrics.migrateCachedSocketsMetric.collect(
		ch,
		float64(redisInfo.Stats.MigrateCachedSockets),
		nil,
	)
	c.metrics.slaveExpiresTrackedKeysMetric.collect(
		ch,
		float64(redisInfo.Stats.SlaveExpiresTrackedKeys),
		nil,
	)
	c.metrics.activeDefragHitsMetric.collect(
		ch,
		float64(redisInfo.Stats.ActiveDefragHits),
		nil,
	)
	c.metrics.activeDefragMissesMetric.collect(
		ch,
		float64(redisInfo.Stats.ActiveDefragMisses),
		nil,
	)
	c.metrics.activeDefragKeyHitsMetric.collect(
		ch,
		float64(redisInfo.Stats.ActiveDefragKeyHits),
		nil,
	)
	c.metrics.activeDefragKeyMissesMetric.collect(
		ch,
		float64(redisInfo.Stats.ActiveDefragKeyMisses),
		nil,
	)

	// -------------------
	// Replication Section
	// -------------------
	c.metrics.replicationRoleMetric.collect(
		ch,
		float64(0),
		nil,
	) // 字符串类型，可选做 info metric
	c.metrics.connectedSlavesMetric.collect(
		ch,
		float64(redisInfo.Replication.ConnectedSlaves),
		nil,
	)
	c.metrics.masterReplOffsetMetric.collect(
		ch,
		float64(redisInfo.Replication.MasterReplOffset),
		nil,
	)
	c.metrics.replBacklogActiveMetric.collect(
		ch,
		float64(redisInfo.Replication.ReplBacklogActive),
		nil,
	)
	c.metrics.replBacklogSizeMetric.collect(
		ch,
		float64(redisInfo.Replication.ReplBacklogSize),
		nil,
	)
	c.metrics.replBacklogFirstByteOffsetMetric.collect(
		ch,
		float64(redisInfo.Replication.ReplBacklogFirstByteOffset),
		nil,
	)
	c.metrics.replBacklogHistlenMetric.collect(
		ch,
		float64(redisInfo.Replication.ReplBacklogHistlen),
		nil,
	)
	c.metrics.minSlavesGoodSlavesMetric.collect(
		ch,
		float64(redisInfo.Replication.MinSlavesGoodSlaves),
		nil,
	)
	c.metrics.minSlavesScoutReportIntrMetric.collect(
		ch,
		float64(redisInfo.Replication.MinSlavesScoutReportIntr),
		nil,
	)

	// -------------------
	// CPU Section
	// -------------------
	c.metrics.usedCpuSysMetric.collect(
		ch,
		redisInfo.CPU.UsedCPUSys,
		nil,
	)
	c.metrics.usedCpuUserMetric.collect(
		ch,
		redisInfo.CPU.UsedCPUUser,
		nil,
	)
	c.metrics.usedCpuSysChildrenMetric.collect(
		ch,
		redisInfo.CPU.UsedCPUSysChildren,
		nil,
	)
	c.metrics.usedCpuUserChildrenMetric.collect(
		ch,
		redisInfo.CPU.UsedCPUUserChildren,
		nil,
	)

	// -------------------
	// Cluster Section
	// -------------------
	c.metrics.clusterEnabledMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterEnabled),
		nil,
	)
	c.metrics.clusterNodeCountMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterNodeCount),
		nil,
	)
	c.metrics.clusterMyEpochMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterMyEpoch),
		nil,
	)
	c.metrics.clusterSlotsAssignedMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterSlotsAssigned),
		nil,
	)
	c.metrics.clusterSlotsOkMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterSlotsOk),
		nil,
	)
	c.metrics.clusterSlotsPfailMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterSlotsPfail),
		nil,
	)
	c.metrics.clusterSlotsFailMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterSlotsFail),
		nil,
	)
	c.metrics.clusterKnownNodesMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterKnownNodes),
		nil,
	)
	c.metrics.clusterSizeMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterSize),
		nil,
	)
	c.metrics.clusterCurrentEpochMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterCurrentEpoch),
		nil,
	)
	c.metrics.clusterStatsMessagesSentMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterStatsMessagesSent),
		nil,
	)
	c.metrics.clusterStatsMessagesReceivedMetric.collect(
		ch,
		float64(redisInfo.Cluster.ClusterStatsMessagesReceived),
		nil,
	)

	// -------------------
	// Keyspace Section
	// -------------------
	for db, ks := range redisInfo.Keyspace {
		c.metrics.keyspaceKeysMetric.collect(
			ch,
			float64(ks.Keys),
			[]string{db},
		)
		c.metrics.keyspaceExpiresMetric.collect(
			ch,
			float64(ks.Expires),
			[]string{db},
		)
		c.metrics.keyspaceAvgTTLMetric.collect(
			ch,
			ks.AvgTTL.Seconds(),
			[]string{db},
		)
	}

	// -------------------
	// Modules Section
	// -------------------
	for _, module := range redisInfo.Modules {
		c.metrics.moduleLoadedMetric.collect(
			ch,
			1,
			[]string{module},
		)
	}
}
