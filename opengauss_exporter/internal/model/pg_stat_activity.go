package model

// PgStatActivity 表示从 pg_stat_activity 视图中采集到的指标信息
type PgStatActivity struct {
	// 按状态分组的连接数
	ActiveConnections            int64
	IdleConnections              int64
	IdleInTransactionConnections int64
	WaitingConnections           int64
	OtherConnections             int64

	// 按数据库分组的连接数
	ConnectionsByDatabase map[string]int64

	// 按用户分组的连接数
	ConnectionsByUser map[string]int64

	// 等待事件统计
	WaitEventStats map[string]int64

	// 长时间运行的查询统计
	LongRunningQueries  int64   // 运行超过5分钟的查询数量
	OldestQueryDuration float64 // 最长运行查询的时间（秒）

	// 总连接数
	TotalConnections int64
}

// NewPgStatActivity 创建一个新的 PgStatActivity 实例

// TODO: implement functions
