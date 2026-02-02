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
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

var (
	debugFile    *os.File
	debugFileMu  sync.Mutex
	debugEnabled = os.Getenv("WHODB_DEBUG_FILE") == "true"
)

// getDebugLogPath returns the platform-appropriate debug log path
func getDebugLogPath() string {
	var logDir string

	switch runtime.GOOS {
	case "darwin":
		home, _ := os.UserHomeDir()
		logDir = filepath.Join(home, "Library", "Logs", "WhoDB")
	case "windows":
		logDir = filepath.Join(os.Getenv("LOCALAPPDATA"), "WhoDB", "Logs")
	default: // linux and others
		home, _ := os.UserHomeDir()
		logDir = filepath.Join(home, ".local", "share", "whodb", "logs")
	}

	return filepath.Join(logDir, "debug.log")
}

// initDebugFile creates the debug log file if not already open
func initDebugFile() error {
	if debugFile != nil {
		return nil
	}

	logPath := getDebugLogPath()
	logDir := filepath.Dir(logPath)

	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open debug log file: %w", err)
	}

	debugFile = f

	// Write startup marker
	fmt.Fprintf(debugFile, "\n\n========== WhoDB Debug Log Started: %s ==========\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(debugFile, "GOOS=%s GOARCH=%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(debugFile, "Log path: %s\n\n", logPath)

	return nil
}

// DebugFile writes a debug message to the file log.
// This always writes regardless of log level, but only if WHODB_DEBUG_FILE=true.
func DebugFile(format string, args ...any) {
	if !debugEnabled {
		return
	}

	debugFileMu.Lock()
	defer debugFileMu.Unlock()

	if err := initDebugFile(); err != nil {
		// Can't log the error anywhere useful, just return
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(debugFile, "[%s] %s\n", timestamp, msg)
	debugFile.Sync() // Flush immediately for debugging
}

// DebugFileAlways writes to debug file regardless of WHODB_DEBUG_FILE setting.
// Use sparingly - mainly for critical debugging paths.
func DebugFileAlways(format string, args ...any) {
	debugFileMu.Lock()
	defer debugFileMu.Unlock()

	if err := initDebugFile(); err != nil {
		return
	}

	timestamp := time.Now().Format("15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(debugFile, "[%s] %s\n", timestamp, msg)
	debugFile.Sync()
}

// GetDebugLogPath returns the path where debug logs are written
func GetDebugLogPath() string {
	return getDebugLogPath()
}

// CloseDebugFile closes the debug log file
func CloseDebugFile() {
	debugFileMu.Lock()
	defer debugFileMu.Unlock()

	if debugFile != nil {
		debugFile.Close()
		debugFile = nil
	}
}
