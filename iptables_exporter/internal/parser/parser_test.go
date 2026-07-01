// 在parser_test.go文件中
package parser

import (
	"strings"
	"testing"
)

func TestParseIptablesSave(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		want     Tables
		wantErr  bool
		checkErr func(t *testing.T, err error)
	}{
		{
			name: "完整解析测试",
			input: `*filter
:INPUT ACCEPT [123:45678]
:FORWARD DROP [0:0]
:OUTPUT ACCEPT [789:12345]
[0:0] -A INPUT -s 192.168.1.200/32 -j DROP
[10:20] -A OUTPUT -p udp --sport 53 -j DROP
COMMIT
*nat
:PREROUTING ACCEPT [0:0]
[0:0] -A PREROUTING -p tcp --dport 80 -j REDIRECT --to-port 8080
COMMIT`,
			want: Tables{
				"filter": &Table{
					Name: "filter",
					Chains: map[string]Chain{
						"INPUT": {
							Name:    "INPUT",
							Policy:  "ACCEPT",
							Packets: 123,
							Bytes:   45678,
							Rules: []Rule{
								{Packets: 0, Bytes: 0, RuleSpec: "tcp --dport 80 -j ACCEPT"},
							},
						},
						"FORWARD": {
							Name:    "FORWARD",
							Policy:  "DROP",
							Packets: 0,
							Bytes:   0,
						},
						"OUTPUT": {
							Name:    "OUTPUT",
							Policy:  "ACCEPT",
							Packets: 789,
							Bytes:   12345,
							Rules: []Rule{
								{Packets: 10, Bytes: 20, RuleSpec: "udp --sport 53 -j DROP"},
							},
						},
					},
				},
				"nat": &Table{
					Name: "nat",
					Chains: map[string]Chain{
						"PREROUTING": {
							Name:    "PREROUTING",
							Policy:  "ACCEPT",
							Packets: 0,
							Bytes:   0,
							Rules: []Rule{
								{Packets: 0, Bytes: 0, RuleSpec: "tcp --dport 80 -j REDIRECT --to-port 8080"},
							},
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseIptablesSave(strings.NewReader(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseIptablesSave() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// 验证表结构
			if len(got) != len(tt.want) {
				t.Fatalf("表数量不符，got %d, want %d", len(got), len(tt.want))
			}

			for tableName, wantTable := range tt.want {
				gotTable, ok := got[tableName]
				if !ok {
					t.Errorf("缺少表: %s", tableName)
					continue
				}

				// 验证链
				if len(gotTable.Chains) != len(wantTable.Chains) {
					t.Errorf("表 %s 链数量不符，got %d, want %d",
						tableName, len(gotTable.Chains), len(wantTable.Chains))
				}

				for chainName, wantChain := range wantTable.Chains {
					gotChain, ok := gotTable.Chains[chainName]
					if !ok {
						t.Errorf("缺少链: %s", chainName)
						continue
					}

					// 验证链基础信息
					if gotChain.Policy != wantChain.Policy ||
						gotChain.Packets != wantChain.Packets ||
						gotChain.Bytes != wantChain.Bytes {
						t.Errorf("链 %s 信息不符，got %+v, want %+v",
							chainName, gotChain, wantChain)
					}

					// 验证规则
					if len(gotChain.Rules) != len(wantChain.Rules) {
						t.Errorf("链 %s 规则数量不符，got %d, want %d",
							chainName, len(gotChain.Rules), len(wantChain.Rules))
						continue
					}

					for i, wantRule := range wantChain.Rules {
						gotRule := gotChain.Rules[i]
						// 验证基础值
						if gotRule.Packets != wantRule.Packets ||
							gotRule.Bytes != wantRule.Bytes {
							t.Errorf("规则计数器不符，got %+v, want %+v",
								gotRule, wantRule)
						}
					}
				}
			}
		})
	}
}

func TestParseCounters(t *testing.T) {
	tests := []struct {
		input string
		pkts  uint64
		bytes uint64
		ok    bool
	}{
		{"[123:456]", 123, 456, true},
		{"invalid", 0, 0, false},
		{"[100]", 0, 0, false},
	}

	for _, tt := range tests {
		pkts, bytes, ok := parseCounters(tt.input)
		if ok != tt.ok || pkts != tt.pkts || bytes != tt.bytes {
			t.Errorf("parseCounters(%q) = (%d, %d, %v), want (%d, %d, %v)",
				tt.input, pkts, bytes, ok, tt.pkts, tt.bytes, tt.ok)
		}
	}
}

func TestHandleSpecialCases(t *testing.T) {
	t.Run("重复规则检测", func(t *testing.T) {
		input := `*filter
:INPUT ACCEPT [0:0]
[0:0] -A INPUT -s 192.168.1.1 -j DROP
[0:0] -A INPUT -s 192.168.1.1 -j DROP
COMMIT`

		tables, err := ParseIptablesSave(strings.NewReader(input))
		if err != nil {
			t.Fatal(err)
		}

		rules := tables["filter"].Chains["INPUT"].Rules
		if len(rules) != 2 {
			t.Fatal("应该保留重复规则")
		}

	})

	t.Run("无效行处理", func(t *testing.T) {
		input := `*filter
INVALID LINE
COMMIT`

		_, err := ParseIptablesSave(strings.NewReader(input))
		if err == nil {
			t.Error("应该返回解析错误")
		}
	})
}
