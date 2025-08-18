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
var enableLogging bool
var noopEntry *logrus.Entry

type ConditionalLogger struct {
	*logrus.Logger
}

func (c *ConditionalLogger) WithError(err error) *logrus.Entry {
	if !enableLogging {
		return noopEntry
	}
	return c.Logger.WithError(err)
}

func (c *ConditionalLogger) WithField(key string, value any) *logrus.Entry {
	if !enableLogging {
		return noopEntry
	}
	return c.Logger.WithField(key, value)
}

func (c *ConditionalLogger) WithFields(fields map[string]any) *logrus.Entry {
	if !enableLogging {
		return noopEntry
	}
	return c.Logger.WithFields(fields)
}

func (c *ConditionalLogger) Error(args ...any) {
	if !enableLogging {
		return
	}
	c.Logger.Error(args...)
}

func (c *ConditionalLogger) Errorf(format string, args ...any) {
	if !enableLogging {
		return
	}
	c.Logger.Errorf(format, args...)
}

func (c *ConditionalLogger) Warn(args ...any) {
	if !enableLogging {
		return
	}
	c.Logger.Warn(args...)
}

func (c *ConditionalLogger) Info(args ...any) {
	if !enableLogging {
		return
	}
	c.Logger.Info(args...)
}

func (c *ConditionalLogger) Infof(format string, args ...any) {
	if !enableLogging {
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

func (c *ConditionalLogger) Println(args ...any) {
	if !enableLogging {
		return
	}
	c.Logger.Println(args...)
}

func init() {
	Logger = &ConditionalLogger{Logger: logrus.New()}
	enableLogging = os.Getenv("WHODB_ENABLE_HTTP_LOGGING") == "true"
	
	// Create a no-op logger that discards all output
	noopLogger := logrus.New()
	noopLogger.SetOutput(io.Discard)
	noopEntry = logrus.NewEntry(noopLogger)
}

func LogFields(fields Fields) *logrus.Entry {
	if !enableLogging {
		return noopEntry
	}
	return Logger.WithFields(logrus.Fields(fields))
}
