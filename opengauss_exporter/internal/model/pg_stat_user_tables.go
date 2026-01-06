package model

import "time"

// PgStatUserTable 表示从 pg_stat_user_tables 视图中采集到的单个用户表的统计信息
type PgStatUserTable struct {
	// 表标识信息
	SchemaName string // 模式名
	TableName  string // 表名
	RelID      int64  // 表OID

	// 扫描统计
	SeqScan     int64 // 顺序扫描次数
	SeqTupRead  int64 // 顺序扫描读取的元组数
	IdxScan     int64 // 索引扫描次数
	IdxTupFetch int64 // 索引扫描获取的元组数

	// 元组变更统计
	NTupIns    int64 // 插入的元组数
	NTupUpd    int64 // 更新的元组数
	NTupDel    int64 // 删除的元组数
	NTupHotUpd int64 // HOT更新的元组数

	// 元组状态统计
	NLiveTup         int64 // 活跃元组数
	NDeadTup         int64 // 死亡元组数
	NModSinceAnalyze int64 // 自上次ANALYZE以来修改的元组数

	// VACUUM统计
	VacuumCount     int64      // 手动VACUUM次数
	AutovacuumCount int64      // 自动VACUUM次数
	LastVacuum      *time.Time // 最后一次手动VACUUM时间
	LastAutovacuum  *time.Time // 最后一次自动VACUUM时间

	// ANALYZE统计
	AnalyzeCount     int64      // 手动ANALYZE次数
	AutoanalyzeCount int64      // 自动ANALYZE次数
	LastAnalyze      *time.Time // 最后一次手动ANALYZE时间
	LastAutoanalyze  *time.Time // 最后一次自动ANALYZE时间
}

// PgStatUserTablesCollection 表示所有用户表的统计信息集合
type PgStatUserTablesCollection struct {
	Tables map[string]*PgStatUserTable // 以"schema.table"为key的统计信息映射
}

// NewPgStatUserTable 创建一个新的 PgStatUserTable 实例

// TODO: implement functions
