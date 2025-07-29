package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"testing"
)

// Helper function to extract JSON from log output that includes Go log prefix
func extractJSONFromLogOutput(output string) (map[string]interface{}, error) {
	var logEntry map[string]interface{}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return nil, fmt.Errorf("no log output")
	}

	line := lines[len(lines)-1]
	jsonStart := strings.Index(line, "{")
	if jsonStart == -1 {
		return nil, fmt.Errorf("no JSON found in log output: %s", line)
	}
	jsonPart := line[jsonStart:]

	err := json.Unmarshal([]byte(jsonPart), &logEntry)
	return logEntry, err
}

func TestDebug(t *testing.T) {
	// Capture log output
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	// Set DEBUG level to ensure debug output is produced
	originalLevel := defaultLogger.level
	SetLevel(DEBUG)
	defer SetLevel(originalLevel)

	fields := map[string]interface{}{
		"field1": "value1",
		"field2": 42,
	}

	Debug("test debug message", fields)

	output := buf.String()
	if output == "" {
		t.Error("Expected debug output, got empty string")
	}

	// Verify it contains structured JSON
	logEntry, err := extractJSONFromLogOutput(output)
	if err != nil {
		t.Errorf("Expected valid JSON log entry, got error: %v", err)
		return
	}

	if logEntry["level"] != "DEBUG" {
		t.Errorf("Expected level DEBUG, got %v", logEntry["level"])
	}

	if logEntry["message"] != "test debug message" {
		t.Errorf("Expected message 'test debug message', got %v", logEntry["message"])
	}

	if logEntry["fields"] != nil {
		fields := logEntry["fields"].(map[string]interface{})
		if fields["field1"] != "value1" {
			t.Errorf("Expected field field1=value1, got %v", fields["field1"])
		}
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fields := map[string]interface{}{
		"user_id": "12345",
		"action":  "login",
	}

	Info("user logged in", fields)

	output := buf.String()
	if output == "" {
		t.Error("Expected info output, got empty string")
	}

	// Verify JSON structure
	logEntry, err := extractJSONFromLogOutput(output)
	if err != nil {
		t.Errorf("Expected valid JSON log entry, got error: %v", err)
		return
	}

	if logEntry["level"] != "INFO" {
		t.Errorf("Expected level INFO, got %v", logEntry["level"])
	}
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fields := map[string]interface{}{
		"error_code": 4001,
		"retry":      true,
	}

	Warn("rate limit approaching", fields)

	output := buf.String()
	if output == "" {
		t.Error("Expected warn output, got empty string")
	}

	// Verify JSON structure
	logEntry, err := extractJSONFromLogOutput(output)
	if err != nil {
		t.Errorf("Expected valid JSON log entry, got error: %v", err)
		return
	}

	if logEntry["level"] != "WARN" {
		t.Errorf("Expected level WARN, got %v", logEntry["level"])
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fields := map[string]interface{}{
		"error":       "database connection failed",
		"retry_count": 3,
	}

	Error("critical system error", fields)

	output := buf.String()
	if output == "" {
		t.Error("Expected error output, got empty string")
	}

	// Verify JSON structure
	logEntry, err := extractJSONFromLogOutput(output)
	if err != nil {
		t.Errorf("Expected valid JSON log entry, got error: %v", err)
		return
	}

	if logEntry["level"] != "ERROR" {
		t.Errorf("Expected level ERROR, got %v", logEntry["level"])
	}
}

func TestLogWithoutFields(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	Info("message without fields")

	output := buf.String()
	if output == "" {
		t.Error("Expected output, got empty string")
	}

	// Should still be valid JSON
	_, err := extractJSONFromLogOutput(output)
	if err != nil {
		t.Errorf("Expected valid JSON log entry, got error: %v", err)
	}
}

func TestLogWithEmptyFields(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fields := make(map[string]interface{})
	Info("message with empty fields", fields)

	output := buf.String()
	if output == "" {
		t.Error("Expected output, got empty string")
	}

	// Should still be valid JSON
	_, err := extractJSONFromLogOutput(output)
	if err != nil {
		t.Errorf("Expected valid JSON log entry, got error: %v", err)
	}
}

func TestLogFieldTypes(t *testing.T) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fields := map[string]interface{}{
		"string_field": "test",
		"int_field":    42,
		"float_field":  3.14,
		"bool_field":   true,
		"nil_field":    nil,
	}

	Info("testing different field types", fields)

	output := buf.String()
	if output == "" {
		t.Error("Expected output, got empty string")
	}

	// Should handle all field types without error
	_, err := extractJSONFromLogOutput(output)
	if err != nil {
		t.Errorf("Expected valid JSON log entry with mixed field types, got error: %v", err)
	}
}

// Benchmark logging functions
func BenchmarkDebug(b *testing.B) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fields := map[string]interface{}{
		"field1": "value1",
		"field2": 42,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Debug("benchmark debug message", fields)
	}
}

func BenchmarkInfo(b *testing.B) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	fields := map[string]interface{}{
		"user_id": "12345",
		"action":  "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark info message", fields)
	}
}

func BenchmarkInfoWithoutFields(b *testing.B) {
	var buf bytes.Buffer
	log.SetOutput(&buf)
	defer log.SetOutput(os.Stderr)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Info("benchmark message without fields")
	}
}
