package model

import (
	"fmt"
	"strings"
	// "github.com/jackc/pgx/v4/stdlib"
)

// InstanceConnection 表示 OpenGauss 实例的连接信息
type InstanceConnection struct {
	Name        string `yaml:"name"`
	Host        string `yaml:"host,omitempty"`
	Port        int    `yaml:"port,omitempty"`
	User        string `yaml:"user,omitempty"`
	Password    string `yaml:"password,omitempty"` // 仅用于连接，不上报为 label
	DBName      string `yaml:"dbname,omitempty"`
	SSLMode     string `yaml:"sslmode,omitempty"`
	SSLRootCert string `yaml:"sslrootcert,omitempty"`
	SSLCert     string `yaml:"sslcert,omitempty"`
	SSLKey      string `yaml:"sslkey,omitempty"`

	// 可选：直接使用 DSN 字符串
	DSN string `yaml:"dsn,omitempty"`
}

// BuildDSN 构建标准 DSN 字符串（优先使用 DSN）
func (c *InstanceConnection) BuildDSN() string {
	if c.DSN != "" {
		return c.DSN
	}

	var dsn strings.Builder
	dsn.WriteString(fmt.Sprintf("host=%s ", c.Host))
	if c.Port > 0 {
		dsn.WriteString(fmt.Sprintf("port=%d ", c.Port))
	}
	if c.User != "" {
		dsn.WriteString(fmt.Sprintf("user=%s ", c.User))
	}
	if c.Password != "" {
		dsn.WriteString(fmt.Sprintf("password=%s ", c.Password)) // 连接用，不上报为 label
	}
	if c.DBName != "" {
		dsn.WriteString(fmt.Sprintf("dbname=%s ", c.DBName))
	}
	if c.SSLMode != "" {
		dsn.WriteString(fmt.Sprintf("sslmode=%s ", c.SSLMode))
	}
	if c.SSLRootCert != "" {
		dsn.WriteString(fmt.Sprintf("sslrootcert=%s ", c.SSLRootCert))
	}
	if c.SSLCert != "" {
		dsn.WriteString(fmt.Sprintf("sslcert=%s ", c.SSLCert))
	}
	if c.SSLKey != "" {
		dsn.WriteString(fmt.Sprintf("sslkey=%s ", c.SSLKey))
	}

	return strings.TrimSpace(dsn.String())
}

func (c *InstanceConnection) BuildLable() string {
	// %s://%s%s?%s
	return fmt.Sprintf("%s:******//%s:%d/%s", c.User, c.Host, c.Port, c.DBName)
	// return fmt.Sprintf("host=%s,port=%d,user=%s,dbname=%s", c.Host, c.Port, c.User, c.DBName)
}
