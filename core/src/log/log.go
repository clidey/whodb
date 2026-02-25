/*
 * Copyright 2026 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/clidey/whodb/core/src/env"
	"github.com/sirupsen/logrus"
)

const (
	ServiceName   = "whodb"
	ISO8601Format = time.RFC3339
)

// Fields is an alias for logrus.Fields used in structured logging.
type Fields logrus.Fields

var logger *logrus.Logger

// logFile holds the opened log file when WHODB_LOG_FILE is set.
var logFile *os.File

// accessLogger is a separate logrus instance for HTTP access logs.
// nil unless WHODB_ACCESS_LOG_FILE is set.
var accessLogger *logrus.Logger
var accessLogFile *os.File

// ConditionalEntry wraps logrus.Entry to downgrade unsupported-operation errors
// from Error to Debug level. All other methods are promoted from *logrus.Entry.
type ConditionalEntry struct {
	*logrus.Entry
}

func Debugf(format string, args ...any) { logger.Debugf(format, args...) }
func Debug(args ...any)                 { logger.Debug(args...) }
func Infof(format string, args ...any)  { logger.Infof(format, args...) }
func Info(args ...any)                  { logger.Info(args...) }
func Warnf(format string, args ...any)  { logger.Warnf(format, args...) }
func Warn(args ...any)                  { logger.Warn(args...) }
func Errorf(format string, args ...any) { logger.Errorf(format, args...) }
func Error(args ...any)                 { logger.Error(args...) }
func Fatalf(format string, args ...any) { logger.Fatalf(format, args...) }
func Fatal(args ...any)                 { logger.Fatal(args...) }

// Alwaysf logs a formatted message at Info level regardless of the configured log level.
func Alwaysf(format string, args ...any) {
	prev := logger.GetLevel()
	logger.SetLevel(logrus.InfoLevel)
	logger.Infof(format, args...)
	logger.SetLevel(prev)
}

// Always logs a message at Info level regardless of the configured log level.
func Always(args ...any) {
	prev := logger.GetLevel()
	logger.SetLevel(logrus.InfoLevel)
	logger.Info(args...)
	logger.SetLevel(prev)
}

// WithError creates a ConditionalEntry with an error field.
func WithError(err error) *ConditionalEntry {
	return &ConditionalEntry{Entry: logger.WithError(err)}
}

// WithField creates a ConditionalEntry with a single field.
func WithField(key string, value any) *ConditionalEntry {
	return &ConditionalEntry{Entry: logger.WithField(key, value)}
}

// WithFields creates a ConditionalEntry with multiple fields.
func WithFields(fields Fields) *ConditionalEntry {
	return &ConditionalEntry{Entry: logger.WithFields(logrus.Fields(fields))}
}

// Error downgrades to Debug when the attached error is errors.ErrUnsupported.
func (e *ConditionalEntry) Error(args ...any) {
	if isUnsupportedOperation(e.Entry.Data["error"]) {
		e.Entry.Debug(args...)
		return
	}
	e.Entry.Error(args...)
}

// Errorf downgrades to Debug when the attached error is errors.ErrUnsupported.
func (e *ConditionalEntry) Errorf(format string, args ...any) {
	if isUnsupportedOperation(e.Entry.Data["error"]) {
		e.Entry.Debug(args...)
		return
	}
	e.Entry.Errorf(format, args...)
}

// Chaining methods return *ConditionalEntry to preserve the Error override.
func (e *ConditionalEntry) WithField(key string, value any) *ConditionalEntry {
	return &ConditionalEntry{Entry: e.Entry.WithField(key, value)}
}

func (e *ConditionalEntry) WithFields(fields map[string]any) *ConditionalEntry {
	return &ConditionalEntry{Entry: e.Entry.WithFields(fields)}
}

func (e *ConditionalEntry) WithError(err error) *ConditionalEntry {
	return &ConditionalEntry{Entry: e.Entry.WithError(err)}
}

func isUnsupportedOperation(err any) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(error); ok {
		return errors.Is(e, errors.ErrUnsupported)
	}
	return false
}

// toLogrusLevel converts a log level string to a logrus.Level.
// Accepts everything logrus accepts (debug, info, warn/warning, error, fatal, panic)
// plus "none"/"off"/"disabled" to silence all output.
func toLogrusLevel(level string) logrus.Level {
	switch strings.ToLower(level) {
	case "", "info":
		return logrus.InfoLevel
	case "none", "off", "disabled":
		return logrus.PanicLevel
	default:
		parsed, err := logrus.ParseLevel(level)
		if err != nil {
			return logrus.InfoLevel
		}
		return parsed
	}
}

type serviceHook struct{}

func (h *serviceHook) Levels() []logrus.Level { return logrus.AllLevels }

func (h *serviceHook) Fire(entry *logrus.Entry) error {
	entry.Data["service"] = ServiceName
	return nil
}

func init() {
	logger = logrus.New()
	logger.SetLevel(toLogrusLevel(env.LogLevel))
	configureFormatter()
	logger.AddHook(&serviceHook{})

	if logFilePath := resolveLogPath(env.LogFile, env.DefaultLogFile); logFilePath != "" {
		logFile = openLogFile(logFilePath)
		logger.SetOutput(logFile)
	}

	if accessLogPath := resolveLogPath(env.AccessLogFile, env.DefaultAccessLogFile); accessLogPath != "" {
		accessLogFile = openLogFile(accessLogPath)
		accessLogger = logrus.New()
		accessLogger.SetOutput(accessLogFile)
		accessLogger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: ISO8601Format,
			FullTimestamp:   true,
		})
	}
}

// resolveLogPath maps the env var value to an actual file path.
// "default" (case-insensitive) returns the provided default path;
// empty stays empty (no file logging); anything else is used as-is.
func resolveLogPath(value string, defaultPath string) string {
	if strings.EqualFold(value, "default") {
		return defaultPath
	}
	return value
}

// openLogFile creates the parent directory and opens the file for appending.
// Exits the process if the directory or file cannot be opened — if file logging
// was explicitly configured, running without it is a misconfiguration.
func openLogFile(path string) *os.File {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		fmt.Fprintf(os.Stderr, "whodb: failed to create log directory %s: %v\n", dir, err)
		os.Exit(1)
	}
	if info, err := os.Lstat(path); err == nil && info.Mode()&os.ModeSymlink != 0 {
		fmt.Fprintf(os.Stderr, "whodb: refusing to open log file %s: path is a symlink\n", path)
		os.Exit(1)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		fmt.Fprintf(os.Stderr, "whodb: failed to open log file %s: %v\n", path, err)
		os.Exit(1)
	}
	return f
}

func configureFormatter() {
	if getLogFormat() == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: ISO8601Format,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: ISO8601Format,
			FullTimestamp:   true,
		})
	}
}

func getLogFormat() string {
	if strings.EqualFold(env.LogFormat, "json") {
		return "json"
	}
	return "text"
}

// GetLevel returns the current log level as a string.
func GetLevel() string {
	return logger.GetLevel().String()
}

// SetLogLevel changes the active log level at runtime.
func SetLogLevel(level string) {
	logger.SetLevel(toLogrusLevel(level))
}

// DisableOutput redirects all log output to io.Discard.
// Used by the TUI where stdout is reserved for terminal rendering.
func DisableOutput() {
	if logger != nil {
		logger.SetOutput(io.Discard)
	}
}

// CloseLogFile closes the log file and access log file if they were opened.
func CloseLogFile() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
	if accessLogFile != nil {
		accessLogFile.Close()
		accessLogFile = nil
		accessLogger = nil
	}
}

// LogAccess writes an HTTP access log entry to the dedicated access log file.
// This is a no-op unless WHODB_ACCESS_LOG_FILE is set.
func LogAccess(method, path string, status int, duration time.Duration, host, remoteAddr string) {
	if accessLogger == nil {
		return
	}
	accessLogger.WithFields(logrus.Fields{
		"method":      method,
		"path":        path,
		"status":      status,
		"duration":    duration,
		"host":        host,
		"remote_addr": remoteAddr,
	}).Info("access")
}
