package model

// PgDatabaseSize 表示单个数据库的大小信息
type PgDatabaseSize struct {
	DatName string // 数据库名称
	Size    int64  // 数据库大小（字节）
}

// PgTableSize 表示单个表的大小信息
type PgTableSize struct {
	SchemaName string // 模式名
	TableName  string // 表名
	Size       int64  // 表大小（字节）
	TotalSize  int64  // 包含索引的总大小（字节）
}

// PgTablespaceSize 表示单个表空间的大小信息
type PgTablespaceSize struct {
	TablespaceName string // 表空间名称
	Size           int64  // 表空间大小（字节）
}

// PgSizeStats 表示数据库大小统计信息集合
type PgSizeStats struct {
	// 数据库大小统计
	DatabaseSizes map[string]*PgDatabaseSize // 以数据库名为key

	// 表大小统计（只统计用户表）
	TableSizes map[string]*PgTableSize // 以"schema.table"为key

	// 表空间大小统计
	TablespaceSizes map[string]*PgTablespaceSize // 以表空间名为key

	// 总体统计
	TotalDatabaseSize int64 // 所有数据库总大小
	TotalTableSize    int64 // 所有表总大小
}

// NewPgSizeStats 创建一个新的 PgSizeStats 实例
func NewPgSizeStats() *PgSizeStats {
	return &PgSizeStats{
		DatabaseSizes:   make(map[string]*PgDatabaseSize),
		TableSizes:      make(map[string]*PgTableSize),
		TablespaceSizes: make(map[string]*PgTablespaceSize),
	}
}

// GetTableKey 生成表的唯一标识key
func (t *PgTableSize) GetTableKey() string {
	return t.SchemaName + "." + t.TableName
}
