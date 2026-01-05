package metrics

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetricName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"metric.name", "metric_name"},
		{"metric_name", "metric_name"},
	}

	for _, tt := range tests {
		result := metricName(tt.input)
		if result != tt.expected {
			t.Errorf("metricName(%s) = %s; expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestParseNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
		err      error
	}{
		{"123.45", 123.45, nil},
		{"invalid", 0, errors.New("invalid number")},
	}

	for _, tt := range tests {
		result, err := parseNumber(tt.input)
		if err != nil && tt.err == nil {
			t.Errorf("parseNumber(%s) returned unexpected error: %v", tt.input, err)
		}
		if result != tt.expected {
			t.Errorf("parseNumber(%s) = %f; expected %f", tt.input, result, tt.expected)
		}
	}
}

func TestHandleResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/success" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("success"))
		} else if r.URL.Path == "/error" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("error"))
		}
	}))
	defer ts.Close()

	tests := []struct {
		uri      string
		expected []byte
		err      error
	}{
		{ts.URL + "/success", []byte("success"), nil},
		{ts.URL + "/error", nil, errors.New("status 500 Internal Server Error (500): error")},
	}

	for _, tt := range tests {
		result, err := handleResponse(tt.uri)
		if err != nil && tt.err == nil {
			t.Errorf("handleResponse(%s) returned unexpected error: %v", tt.uri, err)
		}
		if string(result) != string(tt.expected) {
			t.Errorf("handleResponse(%s) = %s; expected %s", tt.uri, result, tt.expected)
		}
	}
}

func TestParseKeyValueResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("key1 123.45\nkey2 678.90"))
	}))
	defer ts.Close()

	e := &AsyncMetricsExporter{}
	results, err := e.parseKeyValueResponse(ts.URL)
	if err != nil {
		t.Errorf("parseKeyValueResponse returned unexpected error: %v", err)
	}

	expected := []lineResult{
		{"key1", 123.45},
		{"key2", 678.90},
	}

	if len(results) != len(expected) {
		t.Errorf("parseKeyValueResponse returned %d results; expected %d", len(results), len(expected))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("parseKeyValueResponse result[%d] = %v; expected %v", i, result, expected[i])
		}
	}
}

func TestParseDiskResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("disk1 100.0 200.0\ndisk2 300.0 400.0"))
	}))
	defer ts.Close()

	e := &DiskMetricsExporter{}
	results, err := e.parseDiskResponse(ts.URL)
	if err != nil {
		t.Errorf("parseDiskResponse returned unexpected error: %v", err)
	}

	expected := []diskResult{
		{"disk1", 100.0, 200.0},
		{"disk2", 300.0, 400.0},
	}

	if len(results) != len(expected) {
		t.Errorf("parseDiskResponse returned %d results; expected %d", len(results), len(expected))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("parseDiskResponse result[%d] = %v; expected %v", i, result, expected[i])
		}
	}
}

func TestParseEventsURIResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("event1 123.45\nevent2 678.90"))
	}))
	defer ts.Close()

	e := &EventsURIExporter{}
	result, err := e.parseEventsURIResponse(ts.URL)
	if err != nil {
		t.Errorf("parseEventsURIResponse returned unexpected error: %v", err)
	}

	if result == nil {
		t.Errorf("parseEventsURIResponse returned nil result")
	}
}

func TestParseMetricsURIResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("metric1 123.45\nmetric2 678.90"))
	}))
	defer ts.Close()

	e := &MetricsURIExporter{}
	result, err := e.parseMetricsURIResponse(ts.URL)
	if err != nil {
		t.Errorf("parseMetricsURIResponse returned unexpected error: %v", err)
	}

	if result == nil {
		t.Errorf("parseMetricsURIResponse returned nil result")
	}
}

func TestParsePartsResponse(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("db1 table1 100 10 1000\ndb2 table2 200 20 2000"))
	}))
	defer ts.Close()

	e := &PartsURIExporter{}
	results, err := e.parsePartsResponse(ts.URL)
	if err != nil {
		t.Errorf("parsePartsResponse returned unexpected error: %v", err)
	}

	expected := []partsResult{
		{"db1", "table1", 100, 10, 1000},
		{"db2", "table2", 200, 20, 2000},
	}

	if len(results) != len(expected) {
		t.Errorf("parsePartsResponse returned %d results; expected %d", len(results), len(expected))
	}

	for i, result := range results {
		if result != expected[i] {
			t.Errorf("parsePartsResponse result[%d] = %v; expected %v", i, result, expected[i])
		}
	}
}
// Part 2 commit for clickhouse_exporter/internal/metrics/utils_test.go
