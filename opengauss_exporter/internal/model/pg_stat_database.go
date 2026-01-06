package model

import "time"

// PgStatDatabase 表示从 pg_stat_database 视图中采集到的单个数据库的统计信息
type PgStatDatabase struct {
	// 数据库标识
	DatID   int64  // 数据库OID
	DatName string // 数据库名称

	// 连接统计
	NumBackends int64 // 当前连接到此数据库的后端数量

	// 事务统计
	XactCommit   int64 // 已提交的事务数量
	XactRollback int64 // 已回滚的事务数量

	// 磁盘I/O统计
	BlksRead int64 // 从磁盘读取的磁盘块数量
	BlksHit  int64 // 缓冲区命中的磁盘块数量

	// 元组操作统计
	TupReturned int64 // 此数据库中查询返回的行数
	TupFetched  int64 // 此数据库中查询提取的行数
	TupInserted int64 // 此数据库中查询插入的行数
	TupUpdated  int64 // 此数据库中查询更新的行数
	TupDeleted  int64 // 此数据库中查询删除的行数

	// 冲突统计
	Conflicts int64 // 此数据库中由于恢复冲突而取消的查询数量

	// 临时文件统计
	TempFiles int64 // 此数据库中查询创建的临时文件数量
	TempBytes int64 // 此数据库中查询创建的临时文件的总大小

	// 死锁统计
	Deadlocks int64 // 此数据库中检测到的死锁数量

	// 统计重置时间
	StatsReset *time.Time // 统计数据最后重置的时间
}

// PgStatDatabaseCollection 表示所有数据库的统计信息集合
type PgStatDatabaseCollection struct {
	Databases map[string]*PgStatDatabase // 以数据库名为key的统计信息映射
}

// NewPgStatDatabase 创建一个新的 PgStatDatabase 实例

// TODO: implement functions
