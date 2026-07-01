package mysql

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	_ "github.com/go-sql-driver/mysql"
)

const (
	MySQL   = "mysql"
	MariaDB = "mariadb"

	// 连接池设置
	maxOpenConns = 1
	maxIdleConns = 1

	versionQuery = "SELECT VERSION()" // 比 @@version 更标准
)

var (
	// 预编译正则表达式
	versionRegex = regexp.MustCompile(`^(\d+\.\d+\.\d+)`)
)

type Instance struct {
	Db                *sql.DB
	flavor            string
	version           *semver.Version // 改为指针类型
	versionMajorMinor float64
}

// NewInstance 创建并初始化一个新的MySQL/MariaDB实例
func NewInstance(dsn string) (*Instance, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 配置连接池
	db.SetMaxOpenConns(maxOpenConns)
	db.SetMaxIdleConns(maxIdleConns)

	// 立即验证连接
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database ping failed: %w", err)
	}

	// 获取版本信息
	version, versionStr, err := getVersion(db)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to get database version: %w", err)
	}

	// 计算主次版本号
	majorMinor, err := strconv.ParseFloat(
		fmt.Sprintf("%d.%d", version.Major(), version.Minor()), 64)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("failed to parse version: %w", err)
	}

	// 确定数据库类型
	flavor := MySQL
	if strings.Contains(strings.ToLower(versionStr), "mariadb") {
		flavor = MariaDB
	}

	return &Instance{
		Db:                db,
		flavor:            flavor,
		version:           version,
		versionMajorMinor: majorMinor,
	}, nil
}

// getDB 返回数据库连接
func (i *Instance) GetDB() *sql.DB {
	return i.Db
}

// Close 关闭数据库连接
func (i *Instance) Close() error {
	if i.Db == nil {
		return nil
	}
	return i.Db.Close()
}

// Ping 检查连接是否有效
func (i *Instance) Ping() error {
	if err := i.Db.Ping(); err != nil {
		// 关闭无效连接
		if cerr := i.Close(); cerr != nil {
			return fmt.Errorf("ping failed: %v, close also failed: %w", err, cerr)
		}
		return fmt.Errorf("ping failed and connection closed: %w", err)
	}
	return nil
}

// GetVersion 获取数据库版本
func (i *Instance) GetVersion() *semver.Version {
	return i.version
}

// GetFlavor 获取数据库类型
func (i *Instance) GetFlavor() string {
	return i.flavor
}

// getVersion 获取并解析数据库版本
func getVersion(db *sql.DB) (*semver.Version, string, error) {
	var versionStr string
	if err := db.QueryRow(versionQuery).Scan(&versionStr); err != nil {
		return nil, "", fmt.Errorf("version query failed: %w", err)
	}

	matches := versionRegex.FindStringSubmatch(versionStr)
	if len(matches) < 2 {
		return nil, versionStr, fmt.Errorf("invalid version format: %q", versionStr)
	}

	version, err := semver.NewVersion(matches[1])
	if err != nil {
		return nil, versionStr, fmt.Errorf("failed to parse version %q: %w", matches[1], err)
	}

	return version, versionStr, nil
}
