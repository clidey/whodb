// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"io"
	"os"

	"github.com/sirupsen/logrus"
)

type Fields logrus.Fields

var Logger *ConditionalLogger
var logLevel string

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
func isLevelEnabled(level string) bool {
	switch logLevel {
	case "none":
		return false
	case "info":
		return level == "info"
	case "warning":
		return level == "info" || level == "warning"
	case "error":
		return level == "info" || level == "warning" || level == "error"
	default:
		return level == "info" // Default to info only
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

func (c *ConditionalLogger) Fatal(args ...any) {
	// Fatal should always execute regardless of logging flag as it terminates the program
	c.Logger.Fatal(args...)
}

func (c *ConditionalLogger) Fatalf(format string, args ...any) {
	// Fatal should always execute regardless of logging flag as it terminates the program
	c.Logger.Fatalf(format, args...)
}

func init() {
	Logger = &ConditionalLogger{Logger: logrus.New()}
	logLevel = getLogLevel()
}

func getLogLevel() string {
	level := os.Getenv("WHODB_LOG_LEVEL")
	switch level {
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
