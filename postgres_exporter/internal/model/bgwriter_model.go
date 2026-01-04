package model

// PostgreSQLBGWriter 表示后台写入器的统计信息
type PostgreSQLBGWriter struct {
	Database             string  // 数据库名
	CheckpointsScheduled int64   // scheduled checkpoints
	CheckpointsRequested int64   // requested checkpoints
	CheckpointWriteTime  float64 // checkpoint write time (ms)
	CheckpointSyncTime   float64 // checkpoint sync time (ms)
	BuffersCheckpoint    int64   // buffers written during checkpoints
	BuffersClean         int64   // buffers written by background writer
	MaxWaitTime          float64 // max wait time before writing dirty buffer (ms)
	AvgWaitTime          float64 // average wait time before writing dirty buffer
	BuffersBackend       int64   // buffers written directly by backend
	BuffersAllocated     int64   // total buffers allocated
	StatsReset           string  // 最后一次重置时间
}

// PostgreSQLBGWriterStats 包含后台写入器的全局状态
type PostgreSQLBGWriterStats struct {
	BGWriter *PostgreSQLBGWriter
}
