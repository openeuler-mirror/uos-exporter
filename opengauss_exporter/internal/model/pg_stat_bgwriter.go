package model

import "time"

// PgStatBgwriter 表示从 pg_stat_bgwriter 视图中采集到的后台写进程统计信息
type PgStatBgwriter struct {
	// 检查点统计
	CheckpointsTimed    int64   // 定时检查点数量
	CheckpointsReq      int64   // 请求检查点数量
	CheckpointWriteTime float64 // 检查点写入时间（毫秒）
	CheckpointSyncTime  float64 // 检查点同步时间（毫秒）

	// 缓冲区统计
	BuffersCheckpoint   int64 // 检查点期间写入的缓冲区数量
	BuffersClean        int64 // 后台写进程写入的缓冲区数量
	MaxwrittenClean     int64 // 后台写进程因达到最大写入次数而停止的次数
	BuffersBackend      int64 // 后端进程直接写入的缓冲区数量
	BuffersBackendFsync int64 // 后端进程执行的fsync调用次数
	BuffersAlloc        int64 // 分配的缓冲区数量

	// 统计重置时间
	StatsReset *time.Time // 统计数据最后重置的时间
}

// NewPgStatBgwriter 创建一个新的 PgStatBgwriter 实例

// TODO: implement functions
