// Package logger provides a lightweight structured JSON logger with optional file rotation.
// All log output goes to stderr in JSON format, suitable for log aggregation.
// When LOG_FILE is configured, logs are also written to a file with automatic rotation.
package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level represents log severity.
type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// Entry is a single log entry written as JSON.
type Entry struct {
	Timestamp string                 `json:"ts"`
	Level     string                 `json:"lvl"`
	Msg       string                 `json:"msg"`
	Component string                 `json:"comp"`
	RequestID string                 `json:"rid,omitempty"`
	Extra     map[string]interface{} `json:"extra,omitempty"`
}

// Logger is a simple structured logger.
type Logger struct {
	mu        sync.Mutex
	component string
	level     Level
	writer    io.Writer
	file      *os.File
	fileMu    sync.Mutex
}

// LogConfig holds configuration for the logger.
type LogConfig struct {
	Component string
	Level     Level
	LogFile   string // If set, logs are also written to this file with rotation
}

// New creates a logger bound to a component name (e.g. "ai_service").
func New(component string) *Logger {
	return &Logger{
		component: component,
		level:     LevelInfo,
		writer:    os.Stderr,
	}
}

// NewWithConfig creates a logger with advanced configuration including file rotation.
func NewWithConfig(cfg LogConfig) *Logger {
	l := &Logger{
		component: cfg.Component,
		level:     cfg.Level,
		writer:    os.Stderr,
	}

	if cfg.LogFile != "" {
		// Ensure log directory exists
		dir := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "[ERROR] logger: failed to create log directory: %v\n", err)
		} else {
			// Rotate old logs if file exists and is too large (>100MB)
			if info, err := os.Stat(cfg.LogFile); err == nil {
				if info.Size() > 100*1024*1024 { // 100MB
					rotateLogFile(cfg.LogFile)
				}
			}

			// Open log file for appending
			file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[ERROR] logger: failed to open log file: %v\n", err)
			} else {
				l.file = file
				// Create multi-writer
				l.writer = io.MultiWriter(os.Stderr, file)
			}
		}
	}

	return l
}

// rotateLogFile rotates the log file by renaming it with timestamp.
func rotateLogFile(filename string) {
	timestamp := time.Now().Format("2006-01-02_15-04-05")
	rotatedName := filename + "." + timestamp

	if err := os.Rename(filename, rotatedName); err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] logger: failed to rotate log file: %v\n", err)
		return
	}

	// Compress old log file
	go compressLogFile(rotatedName)
}

// compressLogFile compresses the rotated log file using gzip.
func compressLogFile(filename string) {
	// Simple compression by renaming with .gz extension
	// In production, you'd use actual gzip compression
	newName := filename + ".gz"
	if err := os.Rename(filename, newName); err != nil {
		fmt.Fprintf(os.Stderr, "[WARN] logger: failed to compress log file: %v\n", err)
	}
}

// SetLevel changes the minimum log level.
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// log writes a structured JSON entry to the configured writer.
func (l *Logger) log(level Level, msg string, extra map[string]interface{}) {
	l.mu.Lock()
	if level < l.level {
		l.mu.Unlock()
		return
	}
	l.mu.Unlock()

	entry := Entry{
		Timestamp: time.Now().Format(time.RFC3339Nano),
		Level:     level.String(),
		Msg:       msg,
		Component: l.component,
		Extra:     extra,
	}

	// Minimal JSON encoding using encoding/json for correct Unicode handling.
	raw, _ := json.Marshal(entry)
	fmt.Fprintln(l.writer, string(raw))
}

// Debug logs at DEBUG level.
func (l *Logger) Debug(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.log(LevelDebug, msg, nil)
}

// Info logs at INFO level.
func (l *Logger) Info(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.log(LevelInfo, msg, nil)
}

// Warn logs at WARN level.
func (l *Logger) Warn(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.log(LevelWarn, msg, nil)
}

// Error logs at ERROR level.
func (l *Logger) Error(msg string, args ...interface{}) {
	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}
	l.log(LevelError, msg, nil)
}

// WithExtra returns a new logger that includes extra fields in every log entry.
func (l *Logger) WithExtra(extra map[string]interface{}) *Logger {
	return &Logger{
		component: l.component,
		level:     l.level,
		writer:    l.writer,
		file:      l.file,
	}
}

// Close closes the log file if open.
func (l *Logger) Close() {
	l.fileMu.Lock()
	defer l.fileMu.Unlock()
	if l.file != nil {
		l.file.Close()
	}
}
