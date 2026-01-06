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

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-h" {
		fmt.Println("LDAP Connection Diagnostic Tool")
		fmt.Println("Usage: go run ldap_diag.go [config_file]")
		fmt.Println("Default config: ../config/exporter.yaml")
		return
	}

	configFile := "../config/exporter.yaml"
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}

	fmt.Println("=== LDAP Connection Diagnostic ===")
	fmt.Printf("Config file: %s\n", configFile)

	// 加载配置
	data, err := os.ReadFile(configFile)
	if err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config file: %v", err)
	}

	fmt.Printf("LDAP Host: %s\n", cfg.LDAP.Host)
	fmt.Printf("LDAP Port: %s\n", cfg.LDAP.Port)
	fmt.Printf("Bind DN: %s\n", cfg.LDAP.BindDN)
	fmt.Printf("Bind Password: %s\n", "***")

	// 测试连接
	fmt.Println("\n=== Testing Connection ===")
	connStr := fmt.Sprintf("ldap://%s:%s", cfg.LDAP.Host, cfg.LDAP.Port)
	fmt.Printf("Connection string: %s\n", connStr)

	l, err := ldap.DialURL(connStr)
	if err != nil {
		log.Fatalf("Failed to connect to LDAP server: %v", err)
	}
	defer l.Close()
	fmt.Println("✓ Connection successful")

	// 测试匿名绑定
	fmt.Println("\n=== Testing Anonymous Bind ===")
	err = l.UnauthenticatedBind("")
	if err != nil {
		fmt.Printf("✗ Anonymous bind failed: %v\n", err)
	} else {
		fmt.Println("✓ Anonymous bind successful")
	}

	// 重新连接用于认证测试
	l.Close()
	l, err = ldap.DialURL(connStr)
	if err != nil {
		log.Fatalf("Failed to reconnect to LDAP server: %v", err)
	}
	defer l.Close()

	// 测试配置的绑定
	fmt.Println("\n=== Testing Configured Bind ===")
	if cfg.LDAP.BindDN != "" {
		err = l.Bind(cfg.LDAP.BindDN, cfg.LDAP.BindPass)
		if err != nil {
			fmt.Printf("✗ Bind failed: %v\n", err)
			fmt.Println("\nTroubleshooting suggestions:")
			fmt.Println("1. Check if bind_dn is correct")
			fmt.Println("2. Check if bind_password is correct")
			fmt.Println("3. Try anonymous bind (set bind_dn to empty string)")
			fmt.Println("4. Check LDAP server logs")
			fmt.Println("5. Verify LDAP server allows this user to bind")

			// 尝试一些常见的替代方案
			fmt.Println("\n=== Trying Alternative Bind DNs ===")
			alternatives := []string{
				"cn=admin,cn=config",
				"cn=Manager,dc=example,dc=com",
				"cn=admin,dc=example,dc=org",
				"cn=admin,dc=example,dc=com",
				"uid=admin,ou=people,dc=example,dc=com",
				"", // 匿名绑定
			}

			for _, altDN := range alternatives {
				if altDN == "" {
					fmt.Println("Trying: Anonymous bind")
					if err := l.UnauthenticatedBind(""); err != nil {
						fmt.Printf("  ✗ Failed: %v\n", err)
					} else {
						fmt.Println("  ✓ Success with anonymous bind")
						break
					}
				} else {
					fmt.Printf("Trying: %s\n", altDN)
					if err := l.Bind(altDN, cfg.LDAP.BindPass); err != nil {
						fmt.Printf("  ✗ Failed: %v\n", err)
					} else {
						fmt.Printf("  ✓ Success with: %s\n", altDN)
						break
					}
				}
			}
		} else {
			fmt.Println("✓ Bind successful")
		}
	} else {
		fmt.Println("No bind_dn configured, using anonymous bind")
		err = l.UnauthenticatedBind("")
		if err != nil {
			fmt.Printf("✗ Anonymous bind failed: %v\n", err)
		} else {
			fmt.Println("✓ Anonymous bind successful")
		}
	}

	// 测试查询Monitor
	fmt.Println("\n=== Testing Monitor Query ===")
	searchRequest := ldap.NewSearchRequest(
		"cn=Monitor",
		ldap.ScopeBaseObject, ldap.NeverDerefAliases, 1, 0, false,
		"(objectClass=*)",
		[]string{"dn"},
		nil,
	)

	sr, err := l.Search(searchRequest)
	if err != nil {
		fmt.Printf("✗ Monitor query failed: %v\n", err)
		fmt.Println("Suggestions:")
		fmt.Println("1. Check if cn=Monitor exists")
		fmt.Println("2. Check if user has read permission on cn=Monitor")
		fmt.Println("3. Try alternative monitor DNs")

		// 尝试其他可能的监控入口点
		alternatives := []string{
			"cn=config",
			"",
			"dc=example,dc=com",
			"cn=admin,dc=example,dc=com",
		}

		for _, baseDN := range alternatives {
			fmt.Printf("Trying base DN: '%s'\n", baseDN)
			req := ldap.NewSearchRequest(
				baseDN,
				ldap.ScopeBaseObject, ldap.NeverDerefAliases, 1, 0, false,
				"(objectClass=*)",
				[]string{"dn"},
				nil,
			)
			if sr, err := l.Search(req); err != nil {
				fmt.Printf("  ✗ Failed: %v\n", err)
			} else {
				fmt.Printf("  ✓ Success, found %d entries\n", len(sr.Entries))
				if len(sr.Entries) > 0 {
					fmt.Printf("  First entry DN: %s\n", sr.Entries[0].DN)
				}
			}
		}
	} else {
		fmt.Printf("✓ Monitor query successful, found %d entries\n", len(sr.Entries))
		if len(sr.Entries) > 0 {
			fmt.Printf("Monitor DN: %s\n", sr.Entries[0].DN)
		}
	}

	fmt.Println("\n=== Diagnostic Complete ===")
	fmt.Println("\nQuick fixes to try:")
	fmt.Println("1. If anonymous bind works, set bind_dn to empty string in config")
	fmt.Println("2. If alternative bind DN works, update your config file")
	fmt.Println("3. Check LDAP server documentation for correct admin credentials")
	fmt.Println("4. Verify LDAP service is running and accessible")
}
