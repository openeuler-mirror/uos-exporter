package model

import "time"

// PgStatReplication 表示从 pg_stat_replication 视图中采集到的复制统计信息
type PgStatReplication struct {
	// 复制标识信息
	ApplicationName string // 应用名称
	ClientAddr      string // 客户端地址
	ClientHostname  string // 客户端主机名
	State           string // 复制状态

	// 复制滞后统计
	WriteLag  float64 // 写入滞后（秒）
	FlushLag  float64 // 刷新滞后（秒）
	ReplayLag float64 // 重放滞后（秒）

	// 复制进度
	SentLsn   string // 已发送的LSN
	WriteLsn  string // 已写入的LSN
	FlushLsn  string // 已刷新的LSN
	ReplayLsn string // 已重放的LSN

	// 统计信息
	BackendStart *time.Time // 后端启动时间
}

// PgStatReplicationCollection 表示所有复制连接的统计信息集合
type PgStatReplicationCollection struct {
	Replications  map[string]*PgStatReplication // 以"application_name:client_addr"为key
	TotalReplicas int64                         // 总复制连接数
}

// NewPgStatReplication 创建一个新的 PgStatReplication 实例
func NewPgStatReplication() *PgStatReplication {
	return &PgStatReplication{}
}

// NewPgStatReplicationCollection 创建一个新的 PgStatReplicationCollection 实例
func NewPgStatReplicationCollection() *PgStatReplicationCollection {
	return &PgStatReplicationCollection{
		Replications: make(map[string]*PgStatReplication),
	}
}

// GetReplicationKey 生成复制连接的唯一标识key
func (r *PgStatReplication) GetReplicationKey() string {
	return r.ApplicationName + ":" + r.ClientAddr
}
