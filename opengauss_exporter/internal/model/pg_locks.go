package model

// PgLocksStat 表示从 pg_locks 视图中采集到的锁统计信息
type PgLocksStat struct {
	// 锁类型统计
	LocksByType  map[string]int64 // 按锁类型分组的锁数量
	LocksByMode  map[string]int64 // 按锁模式分组的锁数量
	LocksByState map[string]int64 // 按锁状态分组的锁数量

	// 数据库级锁统计
	LocksByDatabase map[string]int64 // 按数据库分组的锁数量

	// 等待锁统计
	WaitingLocks int64 // 等待中的锁数量
	GrantedLocks int64 // 已授予的锁数量
	TotalLocks   int64 // 总锁数量
}

// NewPgLocksStat 创建一个新的 PgLocksStat 实例

// TODO: implement functions
