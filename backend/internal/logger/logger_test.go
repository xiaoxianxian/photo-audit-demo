package logger

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestLogger_Info(t *testing.T) {
	var buf bytes.Buffer
	l := New("test_component")
	l.writer = &buf

	l.Info("test message")

	output := buf.String()
	if !strings.Contains(output, "test message") {
		t.Errorf("expected output to contain 'test message', got: %s", output)
	}

	var entry Entry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if entry.Level != "INFO" {
		t.Errorf("expected level INFO, got %s", entry.Level)
	}

	if entry.Component != "test_component" {
		t.Errorf("expected component test_component, got %s", entry.Component)
	}
}

func TestLogger_Error(t *testing.T) {
	var buf bytes.Buffer
	l := New("test_component")
	l.writer = &buf

	l.Error("error message")

	output := buf.String()
	if !strings.Contains(output, "error message") {
		t.Errorf("expected output to contain 'error message', got: %s", output)
	}

	var entry Entry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if entry.Level != "ERROR" {
		t.Errorf("expected level ERROR, got %s", entry.Level)
	}
}

func TestLogger_Warn(t *testing.T) {
	var buf bytes.Buffer
	l := New("test_component")
	l.writer = &buf

	l.Warn("warning message")

	output := buf.String()
	if !strings.Contains(output, "warning message") {
		t.Errorf("expected output to contain 'warning message', got: %s", output)
	}

	var entry Entry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if entry.Level != "WARN" {
		t.Errorf("expected level WARN, got %s", entry.Level)
	}
}

func TestLogger_Debug(t *testing.T) {
	var buf bytes.Buffer
	l := New("test_component")
	l.writer = &buf
	l.level = LevelDebug // Enable debug level

	l.Debug("debug message")

	output := buf.String()
	if !strings.Contains(output, "debug message") {
		t.Errorf("expected output to contain 'debug message', got: %s", output)
	}

	var entry Entry
	if err := json.Unmarshal([]byte(output), &entry); err != nil {
		t.Fatalf("failed to parse JSON output: %v", err)
	}

	if entry.Level != "DEBUG" {
		t.Errorf("expected level DEBUG, got %s", entry.Level)
	}
}

func TestLogger_LevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	l := New("test_component")
	l.writer = &buf
	l.level = LevelInfo // Only INFO and above

	l.Debug("should not appear")
	l.Info("info message")
	l.Warn("warn message")

	output := buf.String()
	lines := strings.Split(strings.TrimSpace(output), "\n")

	if len(lines) != 2 {
		t.Errorf("expected 2 log lines, got %d", len(lines))
	}

	if !strings.Contains(lines[0], "info message") {
		t.Errorf("expected first line to contain 'info message', got: %s", lines[0])
	}

	if !strings.Contains(lines[1], "warn message") {
		t.Errorf("expected second line to contain 'warn message', got: %s", lines[1])
	}
}

func TestLevel_String(t *testing.T) {
	tests := []struct {
		level    Level
		expected string
	}{
		{LevelDebug, "DEBUG"},
		{LevelInfo, "INFO"},
		{LevelWarn, "WARN"},
		{LevelError, "ERROR"},
		{Level(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.level.String(); got != tt.expected {
				t.Errorf("Level(%d).String() = %s, want %s", tt.level, got, tt.expected)
			}
		})
	}
}

func TestEntry_JSON_Marshal(t *testing.T) {
	entry := Entry{
		Timestamp: "2024-01-01T00:00:00Z",
		Level:     "INFO",
		Msg:       "test message",
		Component: "test",
		Extra: map[string]interface{}{
			"key": "value",
			"number": 42,
		},
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("failed to marshal entry: %v", err)
	}

	var parsed Entry
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("failed to unmarshal entry: %v", err)
	}

	if parsed.Msg != entry.Msg {
		t.Errorf("expected msg %s, got %s", entry.Msg, parsed.Msg)
	}

	if parsed.Extra["key"] != "value" {
		t.Errorf("expected extra[key] = value, got %v", parsed.Extra["key"])
	}
}

func TestNewWithConfig_NoLogFile(t *testing.T) {
	cfg := LogConfig{
		Component: "test",
		Level:     LevelInfo,
		LogFile:   "",
	}

	l := NewWithConfig(cfg)

	if l.component != "test" {
		t.Errorf("expected component 'test', got '%s'", l.component)
	}

	if l.level != LevelInfo {
		t.Errorf("expected level LevelInfo, got %v", l.level)
	}
}
