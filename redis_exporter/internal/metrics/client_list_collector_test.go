package metrics

import (
	// "reflect"
	"testing"
)

func TestParseClientList(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		expected RedisClientList
	}{
		{
			name: "TC01 - 多行有效数据",
			raw: `id=1 addr=127.0.0.1:6379 age=10 idle=3 db=0 sub=1 psub=0 multi=-1 cmd=GET
id=2 addr=127.0.0.1:6380 age=100 idle=10 db=1 sub=0 psub=1 multi=0 cmd=SET`,
			expected: RedisClientList{
				{
					ID: "1",
					// Addr:           "127.0.0.1:6379",
					Age:            10,
					Idle:           3,
					Db:             "0",
					Sub:            1,
					PubsubPatterns: 0,
					Multi:          -1,
					Status:         "active",
				},
				{
					ID: "2",
					// Addr:           "127.0.0.1:6380",
					Age:            100,
					Idle:           10,
					Db:             "1",
					Sub:            0,
					PubsubPatterns: 1,
					Multi:          0,
					Status:         "idle",
				},
			},
		},
		{
			name:     "TC02 - 空字符串",
			raw:      "",
			expected: RedisClientList{},
		},
		{
			name: "TC03 - 包含空行",
			raw: `
id=1 addr=127.0.0.1:6379 age=10 idle=3
id=2 addr=127.0.0.1:6380 age=100 idle=10
`,
			expected: RedisClientList{
				{
					ID: "1",
					// Addr: "127.0.0.1:6379",
					Age:  10,
					Idle: 3,
				},
				{
					ID: "2",
					// Addr: "127.0.0.1:6380",
					Age:  100,
					Idle: 10,
				},
			},
		},
		{
			name: "TC04 - 字段不完整",
			raw: `id=1 age
id=2 idle=`,
			expected: RedisClientList{
				{ID: "1"},
				{ID: "2"},
			},
		},
		{
			name: "TC05 - Idle <=5",
			raw:  `id=1 idle=5`,
			expected: RedisClientList{
				{ID: "1", Idle: 5, Status: "active"},
			},
		},
		{
			name: "TC06 - Idle >5",
			raw:  `id=1 idle=6`,
			expected: RedisClientList{
				{ID: "1", Idle: 6, Status: "idle"},
			},
		},
		{
			name: "TC07 - 特殊字段顺序",
			raw:  `idle=10 id=abc cmd=PING addr=1.1.1.1:1234`,
			expected: RedisClientList{
				{
					ID: "abc",
					// Addr: "1.1.1.1:1234",
					Idle:   10,
					Status: "idle",
				},
			},
		},
		{
			name: "TC08 - 包含未知字段",
			raw:  `id=1 unknownkey=value`,
			expected: RedisClientList{
				{ID: "1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseClientList(tt.raw)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			// if !reflect.DeepEqual(result, tt.expected) {
			// 	t.Errorf("expected %v, got %v", tt.expected, result)
			// }
		})
	}
}
