package metrics

import (
	"strconv"
	"strings"
	"time"
)

// RedisInfo 是整个 Redis INFO 命令输出的结构化表示
type RedisInfo struct {
	Server      ServerInfo
	Clients     ClientsInfo
	Memory      MemoryInfo
	Persistence PersistenceInfo
	Stats       StatsInfo
	Replication ReplicationInfo
	CPU         CPUInfo
	Cluster     ClusterInfo
	Keyspace    map[string]KeyspaceDB // 如 db0, db1...
	Modules     []string
}

// -------------------
// # Server Section
// -------------------
type ServerInfo struct {
	RedisVersion     string
	GitSha1          string
	GitDirty         string
	BuildID          string
	Mode             string
	UptimeInSeconds  int64
	UptimeInDays     int
	PID              int
	TCPPort          int
	UptimeSinceFork  int64
	OS               string
	ArchBits         int
	MultiplexingAPI  string
	GCCVersion       string
	RunID            string
	RedisExecutable  string
	RedisCommandLine string
	LuaEngine        string
}

// -------------------
// # Clients Section
// -------------------
type ClientsInfo struct {
	ConnectedClients        int
	ClientLongestOutputList int
	ClientBiggestInputBuf   int
	BlockedClients          int
	MaxInputBuffer          int
	MaxOutputBuffer         int
	MaxOutputListLength     int
}

// -------------------
// # Memory Section
// -------------------
type MemoryInfo struct {
	UsedMemory           float64
	UsedMemoryHuman      string
	UsedMemoryRss        float64
	UsedMemoryPeak       float64
	UsedMemoryLua        float64
	UsedMemoryOverhead   float64
	UsedMemoryStartup    float64
	UsedMemoryDataset    float64
	UsedMemoryFragmented float64
	TotalSystemMemory    float64
	UsedHugePages        float64
	MaxMemory            float64
	MaxMemoryPolicy      string
	MaxMemorySlab        float64
	MemFragmentation     float64
	MemAllocator         string
}

// -------------------
// # Persistence Section
// -------------------
type PersistenceInfo struct {
	RdbChangesSinceLastSave  int
	RdbBgsaveInProgress      int
	RdbLastSaveTime          time.Time
	RdbLastBgsaveStatus      string
	RdbLastBgsaveTimeSec     int
	RdbCurrentBgsaveTimeSec  int
	AofEnabled               int
	AofRewriteInProgress     int
	AofRewriteScheduled      int
	AofLastRewriteTimeSec    int
	AofCurrentRewriteTimeSec int
	AofLastBgrewriteStatus   string
	AofLastWriteStatus       string
	AofPendingRewrite        int
	AofBufferLength          int
	AofPendingBioFsync       int
	AofDelayedFsync          int
}

// -------------------
// # Stats Section
// -------------------
type StatsInfo struct {
	TotalConnectionsReceived int64
	TotalCommandsProcessed   int64
	InstantaneousOpsPerSec   int
	RejectedConnections      int
	EvictedKeys              int
	KeyspaceMisses           int64
	KeyspaceHitRate          float64
	HashMaxZiplistValue      int
	HashMaxZiplistEntries    int
	PubsubChannels           int
	PubsubPatterns           int
	LatestForkUsec           int64
	MigrateCachedSockets     int
	SlaveExpiresTrackedKeys  int
	ActiveDefragHits         int64
	ActiveDefragMisses       int64
	ActiveDefragKeyHits      int
	ActiveDefragKeyMisses    int
}

// -------------------
// # Replication Section
// -------------------
type ReplicationInfo struct {
	Role                       string
	ConnectedSlaves            int
	MasterReplOffset           int64
	ReplBacklogActive          int
	ReplBacklogSize            int64
	ReplBacklogFirstByteOffset int64
	ReplBacklogHistlen         int64
	MinSlavesGoodSlaves        int
	MinSlavesScoutReportIntr   int
}

// -------------------
// # CPU Section
// -------------------
type CPUInfo struct {
	UsedCPUSys          float64
	UsedCPUUser         float64
	UsedCPUSysChildren  float64
	UsedCPUUserChildren float64
}

// -------------------
// # Cluster Section
// -------------------
type ClusterInfo struct {
	ClusterEnabled               int
	ClusterNodeCount             int
	ClusterMyEpoch               int
	ClusterSlotsAssigned         int
	ClusterSlotsOk               int
	ClusterSlotsPfail            int
	ClusterSlotsFail             int
	ClusterKnownNodes            int
	ClusterSize                  int
	ClusterCurrentEpoch          int
	ClusterStatsMessagesSent     int64
	ClusterStatsMessagesReceived int64
}

// -------------------
// # Keyspace Section (per database)
// -------------------
type KeyspaceDB struct {
	Keys    int
	Expires int
	AvgTTL  time.Duration
}

func parseRedisInfo(info string) (RedisInfo, error) {
	var result RedisInfo
	result.Keyspace = make(map[string]KeyspaceDB)

	lines := strings.Split(info, "\n")
	var currentSection string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否是 section 标题
		if strings.HasPrefix(line, "# ") {
			sectionTitle := strings.TrimSpace(line[2:])
			currentSection = sectionTitle
			continue
		}

		// 解析 key:value 行
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := parts[0]
		value := parts[1]

		switch currentSection {
		case "Server":
			parseServerInfo(key, value, &result.Server)
		case "Clients":
			parseClientsInfo(key, value, &result.Clients)
		case "Memory":
			parseMemoryInfo(key, value, &result.Memory)
		case "Persistence":
			parsePersistenceInfo(key, value, &result.Persistence)
		case "Stats":
			parseStatsInfo(key, value, &result.Stats)
		case "Replication":
			parseReplicationInfo(key, value, &result.Replication)
		case "CPU":
			parseCPUInfo(key, value, &result.CPU)
		case "Cluster":
			parseClusterInfo(key, value, &result.Cluster)
		case "Keyspace":
			db := parseKeyspaceLine(key, value)
			result.Keyspace[key] = db
		case "Modules":
			result.Modules = append(result.Modules, key)
		}
	}

	return result, nil
}

// -------------------
// 各 Section 的解析函数
// -------------------

func parseServerInfo(key, value string, s *ServerInfo) {
	switch key {
	case "redis_version":
		s.RedisVersion = value
	case "git_sha1":
		s.GitSha1 = value
	case "git_dirty":
		s.GitDirty = value
	case "build_id":
		s.BuildID = value
	case "redis_mode":
		s.Mode = value
	case "uptime_in_seconds":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			s.UptimeInSeconds = i
		}
	case "uptime_in_days":
		if i, err := strconv.Atoi(value); err == nil {
			s.UptimeInDays = i
		}
	case "process_id":
		if i, err := strconv.Atoi(value); err == nil {
			s.PID = i
		}
	case "tcp_port":
		if i, err := strconv.Atoi(value); err == nil {
			s.TCPPort = i
		}
	case "uptime_since_fork":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			s.UptimeSinceFork = i
		}
	case "os":
		s.OS = value
	case "arch_bits":
		if i, err := strconv.Atoi(value); err == nil {
			s.ArchBits = i
		}
	case "multiplexing_api":
		s.MultiplexingAPI = value
	case "gcc_version":
		s.GCCVersion = value
	case "run_id":
		s.RunID = value
	case "executable":
		s.RedisExecutable = value
	case "cmdline":
		s.RedisCommandLine = value
	case "lua_engine":
		s.LuaEngine = value
	}
}

func parseClientsInfo(key, value string, c *ClientsInfo) {
	switch key {
	case "connected_clients":
		if i, err := strconv.Atoi(value); err == nil {
			c.ConnectedClients = i
		}
	case "client_longest_output_list":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClientLongestOutputList = i
		}
	case "client_biggest_input_buf":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClientBiggestInputBuf = i
		}
	case "blocked_clients":
		if i, err := strconv.Atoi(value); err == nil {
			c.BlockedClients = i
		}
	case "max_input_buffer":
		if i, err := strconv.Atoi(value); err == nil {
			c.MaxInputBuffer = i
		}
	case "max_output_buffer":
		if i, err := strconv.Atoi(value); err == nil {
			c.MaxOutputBuffer = i
		}
	case "max_output_listpack_length":
		if i, err := strconv.Atoi(value); err == nil {
			c.MaxOutputListLength = i
		}
	}
}

func parseMemoryInfo(key, value string, m *MemoryInfo) {
	switch key {
	case "used_memory":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedMemory = f
		}
	case "used_memory_human":
		m.UsedMemoryHuman = value
	case "used_memory_rss":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedMemoryRss = f
		}
	case "used_memory_peak":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedMemoryPeak = f
		}
	case "used_memory_lua":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedMemoryLua = f
		}
	case "used_memory_overhead":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedMemoryOverhead = f
		}
	case "used_memory_startup":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedMemoryStartup = f
		}
	case "used_memory_dataset":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedMemoryDataset = f
		}
	case "used_memory_fragmented":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedMemoryFragmented = f
		}
	case "total_system_memory":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.TotalSystemMemory = f
		}
	case "used_huge_pages":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.UsedHugePages = f
		}
	case "maxmemory":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.MaxMemory = f
		}
	case "maxmemory_policy":
		m.MaxMemoryPolicy = value
	case "maxmemory_slave":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.MaxMemorySlab = f
		}
	case "mem_fragmentation_ratio":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			m.MemFragmentation = f
		}
	case "mem_allocator":
		m.MemAllocator = value
	}
}

func parsePersistenceInfo(key, value string, p *PersistenceInfo) {
	switch key {
	case "rdb_changes_since_last_save":
		if i, err := strconv.Atoi(value); err == nil {
			p.RdbChangesSinceLastSave = i
		}
	case "rdb_bgsave_in_progress":
		if i, err := strconv.Atoi(value); err == nil {
			p.RdbBgsaveInProgress = i
		}
	case "rdb_last_save_time":
		if t, err := strconv.ParseInt(value, 10, 64); err == nil {
			p.RdbLastSaveTime = time.Unix(t, 0)
		}
	case "rdb_last_bgsave_status":
		p.RdbLastBgsaveStatus = value
	case "rdb_last_bgsave_time_sec":
		if i, err := strconv.Atoi(value); err == nil {
			p.RdbLastBgsaveTimeSec = i
		}
	case "rdb_current_bgsave_time_sec":
		if i, err := strconv.Atoi(value); err == nil {
			p.RdbCurrentBgsaveTimeSec = i
		}
	case "aof_enabled":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofEnabled = i
		}
	case "aof_rewrite_in_progress":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofRewriteInProgress = i
		}
	case "aof_rewrite_scheduled":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofRewriteScheduled = i
		}
	case "aof_last_rewrite_time_sec":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofLastRewriteTimeSec = i
		}
	case "aof_current_rewrite_time_sec":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofCurrentRewriteTimeSec = i
		}
	case "aof_last_bgrewrite_status":
		p.AofLastBgrewriteStatus = value
	case "aof_last_write_status":
		p.AofLastWriteStatus = value
	case "aof_pending_rewrite":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofPendingRewrite = i
		}
	case "aof_buffer_length":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofBufferLength = i
		}
	case "aof_pending_bio_fsync":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofPendingBioFsync = i
		}
	case "aof_delayed_fsync":
		if i, err := strconv.Atoi(value); err == nil {
			p.AofDelayedFsync = i
		}
	}
}

func parseStatsInfo(key, value string, s *StatsInfo) {
	switch key {
	case "total_connections_received":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			s.TotalConnectionsReceived = i
		}
	case "total_commands_processed":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			s.TotalCommandsProcessed = i
		}
	case "instantaneous_ops_per_sec":
		if i, err := strconv.Atoi(value); err == nil {
			s.InstantaneousOpsPerSec = i
		}
	case "rejected_connections":
		if i, err := strconv.Atoi(value); err == nil {
			s.RejectedConnections = i
		}
	case "evicted_keys":
		if i, err := strconv.Atoi(value); err == nil {
			s.EvictedKeys = i
		}
	case "keyspace_misses":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			s.KeyspaceMisses = i
		}
	case "keyspace_hit_ratio":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			s.KeyspaceHitRate = f
		}
	case "hash_max_ziplist_value":
		if i, err := strconv.Atoi(value); err == nil {
			s.HashMaxZiplistValue = i
		}
	case "hash_max_ziplist_entries":
		if i, err := strconv.Atoi(value); err == nil {
			s.HashMaxZiplistEntries = i
		}
	case "pubsub_channels":
		if i, err := strconv.Atoi(value); err == nil {
			s.PubsubChannels = i
		}
	case "pubsub_patterns":
		if i, err := strconv.Atoi(value); err == nil {
			s.PubsubPatterns = i
		}
	case "latest_fork_usec":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			s.LatestForkUsec = i
		}
	case "migrate_cached_sockets":
		if i, err := strconv.Atoi(value); err == nil {
			s.MigrateCachedSockets = i
		}
	case "slave_expires_tracked_keys":
		if i, err := strconv.Atoi(value); err == nil {
			s.SlaveExpiresTrackedKeys = i
		}
	case "active_defrag_hits":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			s.ActiveDefragHits = i
		}
	case "active_defrag_misses":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			s.ActiveDefragMisses = i
		}
	case "active_defrag_key_hits":
		if i, err := strconv.Atoi(value); err == nil {
			s.ActiveDefragKeyHits = i
		}
	case "active_defrag_key_misses":
		if i, err := strconv.Atoi(value); err == nil {
			s.ActiveDefragKeyMisses = i
		}
	}
}

func parseReplicationInfo(key, value string, r *ReplicationInfo) {
	switch key {
	case "role":
		r.Role = value
	case "connected_slaves":
		if i, err := strconv.Atoi(value); err == nil {
			r.ConnectedSlaves = i
		}
	case "master_repl_offset":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			r.MasterReplOffset = i
		}
	case "repl_backlog_active":
		if i, err := strconv.Atoi(value); err == nil {
			r.ReplBacklogActive = i
		}
	case "repl_backlog_size":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			r.ReplBacklogSize = i
		}
	case "repl_backlog_first_byte_offset":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			r.ReplBacklogFirstByteOffset = i
		}
	case "repl_backlog_histlen":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			r.ReplBacklogHistlen = i
		}
	case "min-slaves-good-slaves":
		if i, err := strconv.Atoi(value); err == nil {
			r.MinSlavesGoodSlaves = i
		}
	case "min-slaves-scout-report-intr":
		if i, err := strconv.Atoi(value); err == nil {
			r.MinSlavesScoutReportIntr = i
		}
	}
}

func parseCPUInfo(key, value string, c *CPUInfo) {
	switch key {
	case "used_cpu_sys":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			c.UsedCPUSys = f
		}
	case "used_cpu_user":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			c.UsedCPUUser = f
		}
	case "used_cpu_sys_children":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			c.UsedCPUSysChildren = f
		}
	case "used_cpu_user_children":
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			c.UsedCPUUserChildren = f
		}
	}
}

func parseClusterInfo(key, value string, c *ClusterInfo) {
	switch key {
	case "cluster_enabled":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterEnabled = i
		}
	case "cluster_node_count":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterNodeCount = i
		}
	case "cluster_my_epoch":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterMyEpoch = i
		}
	case "cluster_slots_assigned":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterSlotsAssigned = i
		}
	case "cluster_slots_ok":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterSlotsOk = i
		}
	case "cluster_slots_pfail":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterSlotsPfail = i
		}
	case "cluster_slots_fail":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterSlotsFail = i
		}
	case "cluster_known_nodes":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterKnownNodes = i
		}
	case "cluster_size":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterSize = i
		}
	case "cluster_current_epoch":
		if i, err := strconv.Atoi(value); err == nil {
			c.ClusterCurrentEpoch = i
		}
	case "cluster_stats_messages_sent":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			c.ClusterStatsMessagesSent = i
		}
	case "cluster_stats_messages_received":
		if i, err := strconv.ParseInt(value, 10, 64); err == nil {
			c.ClusterStatsMessagesReceived = i
		}
	}
}

func parseKeyspaceLine(key, value string) KeyspaceDB {
	db, err := parseKeyspace(value)
	if err != nil {
		return KeyspaceDB{}
	}
	return db
}

func parseKeyspace(value string) (KeyspaceDB, error) {
	var db KeyspaceDB

	parts := strings.Split(value, ",")
	for _, part := range parts {
		kv := strings.Split(part, "=")
		if len(kv) != 2 {
			continue
		}
		k := kv[0]
		v := kv[1]

		switch k {
		case "keys":
			i, _ := strconv.Atoi(v)
			db.Keys = i
		case "expires":
			i, _ := strconv.Atoi(v)
			db.Expires = i
		case "avg_ttl":
			i, _ := strconv.Atoi(v)
			db.AvgTTL = time.Duration(i) * time.Second
		}
	}

	return db, nil
}
