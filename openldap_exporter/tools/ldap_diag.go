package main

import (
	"fmt"
	"log"
	"os"

	"github.com/go-ldap/ldap/v3"
	"gopkg.in/yaml.v2"
)

// 简化的配置结构，避免导入依赖
type Config struct {
	LDAP LDAPConfig `yaml:"ldap"`
}

type LDAPConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	BindDN   string `yaml:"bind_dn"`
	BindPass string `yaml:"bind_password"`
}


// TODO: implement functions
