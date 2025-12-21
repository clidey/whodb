/*
 * Copyright 2025 Clidey, Inc.
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
	"io"
	"os"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ServiceName   = "whodb"
	ISO8601Format = time.RFC3339
)

type Fields logrus.Fields

var Logger *ConditionalLogger
var logLevel string
var logFormat string

type ConditionalLogger struct {
	*logrus.Logger
}

type ConditionalEntry struct {
	*logrus.Entry
}

// Error method that respects log levels
func (e *ConditionalEntry) Error(args ...any) {
	if !isLevelEnabled("error") {
		return
	}
	e.Entry.Error(args...)
}

// Errorf method that respects log levels
func (e *ConditionalEntry) Errorf(format string, args ...any) {
	if !isLevelEnabled("error") {
		return
	}
	e.Entry.Errorf(format, args...)
}

// Debug method that respects log levels
func (e *ConditionalEntry) Debug(args ...any) {
	if !isLevelEnabled("debug") {
		return
	}
	e.Entry.Debug(args...)
}

// Debugf method that respects log levels
func (e *ConditionalEntry) Debugf(format string, args ...any) {
	if !isLevelEnabled("debug") {
		return
	}
	e.Entry.Debugf(format, args...)
}

// Info method that respects log levels
func (e *ConditionalEntry) Info(args ...any) {
	if !isLevelEnabled("info") {
		return
	}
	e.Entry.Info(args...)
}

// Infof method that respects log levels
func (e *ConditionalEntry) Infof(format string, args ...any) {
	if !isLevelEnabled("info") {
		return
	}
	e.Entry.Infof(format, args...)
}

// Warn method that respects log levels
func (e *ConditionalEntry) Warn(args ...any) {
	if !isLevelEnabled("warning") {
		return
	}
	e.Entry.Warn(args...)
}

// Warnf method that respects log levels
func (e *ConditionalEntry) Warnf(format string, args ...any) {
	if !isLevelEnabled("warning") {
		return
	}
	e.Entry.Warnf(format, args...)
}

// WithField method for chaining
func (e *ConditionalEntry) WithField(key string, value any) *ConditionalEntry {
	return &ConditionalEntry{Entry: e.Entry.WithField(key, value)}
}

// WithFields method for chaining
func (e *ConditionalEntry) WithFields(fields map[string]any) *ConditionalEntry {
	return &ConditionalEntry{Entry: e.Entry.WithFields(fields)}
}

// WithError method for chaining
func (e *ConditionalEntry) WithError(err error) *ConditionalEntry {
	return &ConditionalEntry{Entry: e.Entry.WithError(err)}
}

// isLevelEnabled checks if the given level should be logged based on current log level
// Levels from most to least verbose: debug > info > warning > error > none
func isLevelEnabled(level string) bool {
	switch logLevel {
	case "none":
		return false
	case "error":
		return level == "error"
	case "warning":
		return level == "error" || level == "warning"
	case "info":
		return level == "error" || level == "warning" || level == "info"
	case "debug":
		return true // debug enables all levels
	default:
		return level == "error" || level == "warning" || level == "info" // Default to info level (no debug)
	}
}

func (c *ConditionalLogger) WithError(err error) *ConditionalEntry {
	return &ConditionalEntry{Entry: c.Logger.WithError(err)}
}

func (c *ConditionalLogger) WithField(key string, value any) *ConditionalEntry {
	return &ConditionalEntry{Entry: c.Logger.WithField(key, value)}
}

func (c *ConditionalLogger) WithFields(fields map[string]any) *ConditionalEntry {
	return &ConditionalEntry{Entry: c.Logger.WithFields(fields)}
}

func (c *ConditionalLogger) Error(args ...any) {
	if !isLevelEnabled("error") {
		return
	}
	c.Logger.Error(args...)
}

func (c *ConditionalLogger) Errorf(format string, args ...any) {
	if !isLevelEnabled("error") {
		return
	}
	c.Logger.Errorf(format, args...)
}

func (c *ConditionalLogger) Warn(args ...any) {
	if !isLevelEnabled("warning") {
		return
	}
	c.Logger.Warn(args...)
}

func (c *ConditionalLogger) Warnf(format string, args ...any) {
	if !isLevelEnabled("warning") {
		return
	}
	c.Logger.Warnf(format, args...)
}

func (c *ConditionalLogger) Info(args ...any) {
	if !isLevelEnabled("info") {
		return
	}
	c.Logger.Info(args...)
}

func (c *ConditionalLogger) Infof(format string, args ...any) {
	if !isLevelEnabled("info") {
		return
	}
	c.Logger.Infof(format, args...)
}

func (c *ConditionalLogger) Debug(args ...any) {
	if !isLevelEnabled("debug") {
		return
	}
	c.Logger.Debug(args...)
}

func (c *ConditionalLogger) Debugf(format string, args ...any) {
	if !isLevelEnabled("debug") {
		return
	}
	c.Logger.Debugf(format, args...)
}

func (c *ConditionalLogger) Fatal(args ...any) {
	// Fatal should always execute regardless of logging flag as it terminates the program
	c.Logger.Fatal(args...)
}

func (c *ConditionalLogger) Fatalf(format string, args ...any) {
	// Fatal should always execute regardless of logging flag as it terminates the program
	c.Logger.Fatalf(format, args...)
}

// serviceHook adds service name to all log entries
type serviceHook struct{}

func (h *serviceHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (h *serviceHook) Fire(entry *logrus.Entry) error {
	entry.Data["service"] = ServiceName
	return nil
}

func init() {
	Logger = &ConditionalLogger{Logger: logrus.New()}
	// Set logrus to debug level - our wrapper handles filtering
	Logger.Logger.SetLevel(logrus.DebugLevel)
	logLevel = getLogLevel()
	logFormat = getLogFormat()
	configureFormatter()
	// Add service name to all log entries
	Logger.AddHook(&serviceHook{})
}

// configureFormatter sets up JSON or text formatter based on WHODB_LOG_FORMAT
func configureFormatter() {
	if logFormat == "json" {
		Logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: ISO8601Format,
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "timestamp",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "message",
			},
		})
	} else {
		Logger.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: ISO8601Format,
			FullTimestamp:   true,
		})
	}
}

func getLogFormat() string {
	format := os.Getenv("WHODB_LOG_FORMAT")
	switch format {
	case "json", "JSON", "Json":
		return "json"
	default:
		return "text"
	}
}

func getLogLevel() string {
	level := os.Getenv("WHODB_LOG_LEVEL")
	switch level {
	case "debug", "DEBUG", "Debug":
		return "debug"
	case "info", "INFO", "Info":
		return "info"
	case "warning", "WARNING", "Warning", "warn", "WARN", "Warn":
		return "warning"
	case "error", "ERROR", "Error":
		return "error"
	case "none", "NONE", "None", "off", "OFF", "Off", "disabled", "DISABLED", "Disabled":
		return "none"
	default:
		return "info" // Default to info level
	}
}

func LogFields(fields Fields) *ConditionalEntry {
	return Logger.WithFields(logrus.Fields(fields))
}

func SetLogLevel(level string) {
	logLevel = level
}

// DisableOutput redirects all log output to io.Discard, preventing any logs from appearing in stdout/stderr.
// This is useful for TUI applications where stdout is used for terminal rendering.
func DisableOutput() {
	if Logger != nil {
		Logger.SetOutput(io.Discard)
	}
}
