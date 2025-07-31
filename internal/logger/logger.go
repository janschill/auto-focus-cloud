package logger

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

type Logger struct {
	level LogLevel
}

type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

var defaultLogger = &Logger{level: INFO}

func New(level LogLevel) *Logger {
	return &Logger{level: level}
}

func SetLevel(level LogLevel) {
	defaultLogger.level = level
}

func (l *Logger) log(level LogLevel, message string, fields map[string]interface{}) {
	if level < l.level {
		return
	}

	// Sanitize sensitive data
	sanitizedFields := sanitizeFields(fields)

	entry := LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		Fields:    sanitizedFields,
	}

	jsonBytes, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	log.Println(string(jsonBytes))
}

func (l *Logger) Debug(message string, fields ...map[string]interface{}) {
	l.log(DEBUG, message, mergeFields(fields...))
}

func (l *Logger) Info(message string, fields ...map[string]interface{}) {
	l.log(INFO, message, mergeFields(fields...))
}

func (l *Logger) Warn(message string, fields ...map[string]interface{}) {
	l.log(WARN, message, mergeFields(fields...))
}

func (l *Logger) Error(message string, fields ...map[string]interface{}) {
	l.log(ERROR, message, mergeFields(fields...))
}

// Package-level convenience functions
func Debug(message string, fields ...map[string]interface{}) {
	defaultLogger.Debug(message, fields...)
}

func Info(message string, fields ...map[string]interface{}) {
	defaultLogger.Info(message, fields...)
}

func Warn(message string, fields ...map[string]interface{}) {
	defaultLogger.Warn(message, fields...)
}

func Error(message string, fields ...map[string]interface{}) {
	defaultLogger.Error(message, fields...)
}

// Helper functions
func mergeFields(fieldMaps ...map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for _, fields := range fieldMaps {
		for k, v := range fields {
			result[k] = v
		}
	}
	return result
}

func sanitizeFields(fields map[string]interface{}) map[string]interface{} {
	if fields == nil {
		return nil
	}

	sanitized := make(map[string]interface{})
	sensitiveKeys := []string{
		"key", "token", "secret", "password", "api_key", "stripe_key",
		"webhook_secret", "signature", "authorization", "auth",
	}

	for k, v := range fields {
		keyLower := strings.ToLower(k)

		// Check if key contains sensitive terms
		isSensitive := false
		for _, sensitive := range sensitiveKeys {
			if strings.Contains(keyLower, sensitive) {
				isSensitive = true
				break
			}
		}

		if isSensitive {
			// Redact sensitive values
			if str, ok := v.(string); ok && len(str) > 0 {
				if len(str) <= 8 {
					sanitized[k] = "[REDACTED]"
				} else {
					// Show first 3 and last 3 characters
					sanitized[k] = str[:3] + "..." + str[len(str)-3:]
				}
			} else {
				sanitized[k] = "[REDACTED]"
			}
		} else {
			sanitized[k] = v
		}
	}

	return sanitized
}

// Initialize logger based on environment
func init() {
	// During tests, reduce log noise by setting higher log level
	if os.Getenv("GO_ENV") == "test" || strings.Contains(os.Args[0], ".test") {
		SetLevel(WARN) // Only show WARN and ERROR during tests
		return
	}
	
	logLevel := os.Getenv("LOG_LEVEL")
	switch strings.ToUpper(logLevel) {
	case "DEBUG":
		SetLevel(DEBUG)
	case "INFO":
		SetLevel(INFO)
	case "WARN":
		SetLevel(WARN)
	case "ERROR":
		SetLevel(ERROR)
	default:
		SetLevel(INFO) // Default to INFO
	}
}
