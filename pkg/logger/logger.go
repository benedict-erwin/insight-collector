package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

// orderedJSONWriter ensures consistent field ordering in JSON output
type orderedJSONWriter struct {
	output io.Writer
}

// Write processes the log data and ensures proper field ordering
func (w *orderedJSONWriter) Write(p []byte) (n int, err error) {
	// Parse the JSON data
	var logData map[string]interface{}
	if err := json.Unmarshal(p, &logData); err != nil {
		// If parsing fails, write as-is
		return w.output.Write(p)
	}

	// Build JSON manually to guarantee field order
	var jsonParts []string
	
	// Field order: time, level, scope, message, then others
	fieldOrder := []string{"time", "level", "scope", "message"}
	processedFields := make(map[string]bool)
	
	// Add fields in desired order
	for _, field := range fieldOrder {
		if value, exists := logData[field]; exists {
			jsonValue, _ := json.Marshal(value)
			jsonParts = append(jsonParts, fmt.Sprintf(`"%s":%s`, field, jsonValue))
			processedFields[field] = true
		}
	}
	
	// Add remaining fields
	for key, value := range logData {
		if !processedFields[key] {
			jsonValue, _ := json.Marshal(value)
			jsonParts = append(jsonParts, fmt.Sprintf(`"%s":%s`, key, jsonValue))
		}
	}

	// Build final JSON
	orderedJSON := "{" + strings.Join(jsonParts, ",") + "}\n"
	return w.output.Write([]byte(orderedJSON))
}

// init initializes default logger for early initialization
func init() {
	zerolog.TimeFieldFormat = time.RFC3339
	log = zerolog.New(os.Stdout).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.DefaultContextLogger = &log
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().In(time.UTC)
	}
	log.Info().Msg("Logger initialized with default settings in pkg/logger")
}

// Init configures the logger with timezone settings
func Init(timezone, environment string) {
	// Set timezone for logging
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
		log.Warn().Err(err).Str("timezone", timezone).Msg("Invalid timezone, using UTC")
	}

	// Set zerolog global settings
	zerolog.TimestampFieldName = "time"
	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.LevelFieldName = "level"
	zerolog.MessageFieldName = "message"

	// Choose writer based on environment
	var writer io.Writer
	if environment == "prod" {
		// Production: direct output for performance
		writer = os.Stdout
		log.Info().Str("environment", environment).Msg("Using direct JSON output for production")
	} else {
		// Development/staging: ordered output for readability
		writer = &orderedJSONWriter{output: os.Stdout}
		log.Info().Str("environment", environment).Msg("Using ordered JSON output for development")
	}

	// Init logger with appropriate writer
	log = zerolog.New(writer).
		With().
		Timestamp().
		Logger().
		Level(zerolog.InfoLevel)
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.DefaultContextLogger = &log

	zerolog.TimestampFunc = func() time.Time {
		return time.Now().In(loc)
	}
	log.Info().Str("timezone", loc.String()).Str("environment", environment).Msg("Logger reconfigured")
}

// Log returns a log event
func Log() *zerolog.Event {
	return log.Log()
}

// Debug returns an debug level log event
func Debug() *zerolog.Event {
	return log.Debug()
}

// Info returns an info level log event
func Info() *zerolog.Event {
	return log.Info()
}

// Warn returns a warning level log event
func Warn() *zerolog.Event {
	return log.Warn()
}

// Error returns an error level log event
func Error() *zerolog.Event {
	return log.Error()
}

// Fatal returns a fatal level log event
func Fatal() *zerolog.Event {
	return log.Fatal()
}

// Panic returns a panic level log event
func Panic() *zerolog.Event {
	return log.Panic()
}

// ScopedLogger represents a logger with predefined scope
type ScopedLogger struct {
	logger zerolog.Logger
	scope  string
}

// WithScope creates a new scoped logger instance with predefined scope
func WithScope(scope string) *ScopedLogger {
	scopedLogger := log.With().Str("scope", scope).Logger()
	return &ScopedLogger{
		logger: scopedLogger,
		scope:  scope,
	}
}

// Log returns a log level log event with scope
func (s *ScopedLogger) Log() *zerolog.Event {
	return s.logger.Log()
}

// Debug returns a debug level log event with scope
func (s *ScopedLogger) Debug() *zerolog.Event {
	return s.logger.Debug()
}

// Info returns an info level log event with scope
func (s *ScopedLogger) Info() *zerolog.Event {
	return s.logger.Info()
}

// Warn returns a warning level log event with scope
func (s *ScopedLogger) Warn() *zerolog.Event {
	return s.logger.Warn()
}

// Error returns an error level log event with scope
func (s *ScopedLogger) Error() *zerolog.Event {
	return s.logger.Error()
}

// Fatal returns a fatal level log event with scope
func (s *ScopedLogger) Fatal() *zerolog.Event {
	return s.logger.Fatal()
}

// Panic returns a panic level log event with scope
func (s *ScopedLogger) Panic() *zerolog.Event {
	return s.logger.Panic()
}

// GetScope returns the current scope name
func (s *ScopedLogger) GetScope() string {
	return s.scope
}
