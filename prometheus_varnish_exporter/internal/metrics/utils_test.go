package metrics

import (
	"testing"
)

func TestStartsWith_CaseInsensitive(t *testing.T) {
	testCases := []struct {
		name     string
		str      string
		prefix   string
		expect   bool
	}{
		{"Exact match", "HelloWorld", "hello", true},
		{"Partial uppercase match", "Foobar", "FOO", true},
		{"Full uppercase match", "TEST", "test", true},
		{"No match", "nope", "yes", false},
		{"Empty prefix", "anything", "", true},
		{"Empty string", "", "prefix", false},
		{"Unicode characters", "Gödel", "gö", true},
		{"Long string match", "this is a very long string", "THIS IS", true},
		{"Special characters", "!@#$%^", "!@#", true},
		{"Numbers", "12345", "12", true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := startsWith(tc.str, tc.prefix, CaseInsensitive)
			if result != tc.expect {
				t.Errorf("startsWith(%q, %q, CaseInsensitive) = %v, want %v",
					tc.str, tc.prefix, result, tc.expect)
			}
		})
	}
}

func TestStartsWith_CaseSensitive(t *testing.T) {
	testCases := []struct {
		name     string
		str      string
		prefix   string
		expect   bool
	}{
		{"Exact match", "HelloWorld", "Hello", true},
		{"Case mismatch", "Foobar", "FOO", false},
		{"Full match", "test", "test", true},
		{"Partial match", "testing", "test", true},
		{"No match", "example", "sample", false},
		{"Empty string", "", "prefix", false},
		{"Unicode exact match", "Gödel", "Gö", true},
		{"Unicode case mismatch", "Gödel", "gö", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := startsWith(tc.str, tc.prefix, CaseSensitive)
			if result != tc.expect {
				t.Errorf("startsWith(%q, %q, CaseSensitive) = %v, want %v",
					tc.str, tc.prefix, result, tc.expect)
			}
		})
	}
}

func TestStringProperty_Valid(t *testing.T) {
	data := map[string]interface{}{
		"string": "value",
		"number": 42,
		"bool":   true,
		"nil":    nil,
	}

	tests := []struct {
		name     string
		key      string
		expected string
		wantErr  bool
	}{
		{"Existing string", "string", "value", false},
		{"Non-string value", "number", "", true},
		{"Boolean value", "bool", "", true},
		{"Nil value", "nil", "", false},
		{"Missing key", "missing", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := stringProperty(data, tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("stringProperty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if val != tt.expected {
				t.Errorf("stringProperty() = %v, want %v", val, tt.expected)
			}
		})
	}
}

func TestStartsWithAny(t *testing.T) {
	tests := []struct {
		name     string
		str      string
		prefixes []string
		cs       CaseSensitivity
		expected bool
	}{
		{"Case insensitive match", "HelloWorld", []string{"hello", "test"}, CaseInsensitive, true},
		{"Case sensitive match", "HelloWorld", []string{"Hello", "Test"}, CaseSensitive, true},
		{"No match", "HelloWorld", []string{"test", "demo"}, CaseSensitive, false},
		{"Empty prefixes", "HelloWorld", []string{}, CaseSensitive, false},
		{"Empty string", "", []string{"test"}, CaseSensitive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := startsWithAny(tt.str, tt.prefixes, tt.cs)
			if result != tt.expected {
				t.Errorf("startsWithAny() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCompareChunk(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		cs       CaseSensitivity
		expected bool
	}{
		{"Case sensitive match", "abc", "abc", CaseSensitive, true},
		{"Case sensitive mismatch", "abc", "def", CaseSensitive, false},
		{"Case insensitive match", "ABC", "abc", CaseInsensitive, true},
		{"Case insensitive mismatch", "ABC", "def", CaseInsensitive, false},
		{"Empty strings", "", "", CaseSensitive, true},
		{"Different lengths", "abc", "abcd", CaseSensitive, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareChunk(tt.a, tt.b, tt.cs)
			if result != tt.expected {
				t.Errorf("compareChunk() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestToLower(t *testing.T) {
	tests := []struct {
		input    byte
		expected byte
	}{
		{'A', 'a'},
		{'Z', 'z'},
		{'a', 'a'},
		{'z', 'z'},
		{'0', '0'},
		{'@', '@'},
	}

	for _, tt := range tests {
		t.Run(string(tt.input), func(t *testing.T) {
			result := toLower(tt.input)
			if result != tt.expected {
				t.Errorf("toLower(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}