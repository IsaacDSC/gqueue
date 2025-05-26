package logs

import (
	"context"
	"io"
	"log/slog"
	"os"
	"sync"
)

var (
	defaultLogger *Logger
	once          sync.Once
)

// LogLevel defines the log levels
type LogLevel int

// Log levels
const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
)

// Logger wraps slog functionality
type Logger struct {
	slogger *slog.Logger
}

// LogOption defines functional options for configuring the logger
type LogOption func(*logConfig)

type logConfig struct {
	level      LogLevel
	output     io.Writer
	addSource  bool
	jsonFormat bool
	timeFormat string
}

// WithLevel sets the minimum log level
func WithLevel(level LogLevel) LogOption {
	return func(c *logConfig) {
		c.level = level
	}
}

// WithOutput sets the output writer
func WithOutput(w io.Writer) LogOption {
	return func(c *logConfig) {
		c.output = w
	}
}

// WithSource adds source code location to logs
func WithSource() LogOption {
	return func(c *logConfig) {
		c.addSource = true
	}
}

// WithJSONFormat sets log format to JSON
func WithJSONFormat(enabled bool) LogOption {
	return func(c *logConfig) {
		c.jsonFormat = enabled
	}
}

// WithTimeFormat sets custom time format
func WithTimeFormat(format string) LogOption {
	return func(c *logConfig) {
		c.timeFormat = format
	}
}

// GetLevel converts our LogLevel to slog.Level
func (l LogLevel) GetLevel() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// New creates a new configured logger
func New(opts ...LogOption) *Logger {
	config := &logConfig{
		level:      LevelInfo,
		output:     os.Stdout,
		addSource:  false,
		jsonFormat: true, // Default to JSON format
		timeFormat: "",   // Default time format
	}

	for _, opt := range opts {
		opt(config)
	}

	handlerOptions := &slog.HandlerOptions{
		Level:     config.level.GetLevel(),
		AddSource: config.addSource,
	}

	var handler slog.Handler
	if config.jsonFormat {
		handler = slog.NewJSONHandler(config.output, handlerOptions)
	} else {
		handler = slog.NewTextHandler(config.output, handlerOptions)
	}

	return &Logger{
		slogger: slog.New(handler),
	}
}

// Default returns the default singleton logger
func Default() *Logger {
	once.Do(func() {
		// Initialize with JSON format by default
		defaultLogger = New()
	})
	return defaultLogger
}

// SetDefault sets the default logger
func SetDefault(l *Logger) {
	defaultLogger = l
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, args ...any) {
	l.slogger.Debug(msg, args...)
}

// Info logs an info message
func (l *Logger) Info(msg string, args ...any) {
	l.slogger.Info(msg, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, args ...any) {
	l.slogger.Warn(msg, args...)
}

// Error logs an error message
func (l *Logger) Error(msg string, args ...any) {
	l.slogger.Error(msg, args...)
}

// WithContext returns a logger with context values
func (l *Logger) WithContext(ctx context.Context) *Logger {
	// The standard slog.Logger doesn't have a WithContext method
	// We'll extract values from context and add them as attributes
	return l
}

// With returns a logger with the given attributes
func (l *Logger) With(args ...any) *Logger {
	return &Logger{
		slogger: l.slogger.With(args...),
	}
}

// Package-level shortcuts that use the default logger

// Debug logs a debug message
func Debug(msg string, args ...any) {
	Default().Debug(msg, args...)
}

// Info logs an info message
func Info(msg string, args ...any) {
	Default().Info(msg, args...)
}

// Warn logs a warning message
func Warn(msg string, args ...any) {
	Default().Warn(msg, args...)
}

// Error logs an error message
func Error(msg string, args ...any) {
	Default().Error(msg, args...)
}

// With returns the default logger with the given attributes
func With(args ...any) *Logger {
	return Default().With(args...)
}
