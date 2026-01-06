package model

// PgStatUserIndex 表示从 pg_stat_user_indexes 视图中采集到的单个用户索引的统计信息
type PgStatUserIndex struct {
	// 索引标识信息
	SchemaName string // 模式名
	TableName  string // 表名
	IndexName  string // 索引名
	IndexID    int64  // 索引OID

	// 索引使用统计
	IdxScan     int64 // 索引扫描次数
	IdxTupRead  int64 // 索引扫描返回的索引条目数
	IdxTupFetch int64 // 通过索引扫描获取的活跃表行数
}

// PgStatUserIndexesCollection 表示所有用户索引的统计信息集合
type PgStatUserIndexesCollection struct {
	Indexes map[string]*PgStatUserIndex // 以"schema.table.index"为key的统计信息映射
}

// NewPgStatUserIndex 创建一个新的 PgStatUserIndex 实例
func NewPgStatUserIndex() *PgStatUserIndex {
	return &PgStatUserIndex{}
}

// NewPgStatUserIndexesCollection 创建一个新的 PgStatUserIndexesCollection 实例
func NewPgStatUserIndexesCollection() *PgStatUserIndexesCollection {
	return &PgStatUserIndexesCollection{
		Indexes: make(map[string]*PgStatUserIndex),
	}
}

// GetIndexKey 生成索引的唯一标识key
func (i *PgStatUserIndex) GetIndexKey() string {
	return i.SchemaName + "." + i.TableName + "." + i.IndexName
}
