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

package analytics

import (
	"os"
	"runtime"
	"time"

	"github.com/google/uuid"
	"github.com/posthog/posthog-go"

	"github.com/clidey/whodb/core/src/common/config"
	"github.com/clidey/whodb/core/src/common/datadir"
	"github.com/clidey/whodb/core/src/log"
)

const (
	heartbeatEvent    = "telemetry.heartbeat"
	heartbeatInterval = 24 * time.Hour

	// HeartbeatDisabledEnv disables the anonymous install heartbeat when set
	// to "true". It does not affect the consent-gated product analytics.
	HeartbeatDisabledEnv = "WHODB_HEARTBEAT_DISABLED"

	telemetryConfigSection = "telemetry"
)

// telemetryConfig is the persisted telemetry section of the unified config file.
type telemetryConfig struct {
	InstallID string `json:"installId"`
}

// heartbeatDisabled reports whether the install heartbeat is switched off.
// CI runs are always skipped: each CI job is a fresh container with a fresh
// install id and would pollute install counts.
func heartbeatDisabled() bool {
	return os.Getenv(HeartbeatDisabledEnv) == "true" || os.Getenv("CI") == "true"
}

// installID returns the persisted anonymous install identifier, creating and
// persisting a new one on first run. It is a random UUID with no relation to
// the machine or user; deleting the config file resets it.
func installID(opts datadir.Options) string {
	var section telemetryConfig
	if err := config.ReadSection(telemetryConfigSection, &section, opts); err == nil && section.InstallID != "" {
		return section.InstallID
	}

	section.InstallID = uuid.NewString()
	if err := config.WriteSection(telemetryConfigSection, section, opts); err != nil {
		log.WithError(err).Debug("Telemetry: unable to persist install id")
	}
	return section.InstallID
}

// sendHeartbeat enqueues one anonymous heartbeat event using the stable
// install id, so unique installs, retention, and version adoption are
// measurable over time. It intentionally bypasses the user-controlled
// Enabled() gate and the request-metadata enrichment in buildProperties: the
// heartbeat carries no request context, no person profile, and no IP, and is
// governed solely by HeartbeatDisabledEnv.
func sendHeartbeat(id string, now time.Time) {
	c := loadClient()
	if c == nil {
		return
	}

	// Release artifacts have the version stamped via ldflags; an empty version
	// means a source/dev build (go run, local compile). Tag rather than skip:
	// some people genuinely run from source, so keep them countable but
	// filterable.
	version := cfg.AppVersion
	if version == "" {
		version = "dev"
	}

	props := posthog.NewProperties().
		Set("build_edition", cfg.Edition).
		Set("app_version", version).
		Set("dev_build", version == "dev").
		Set("os", runtime.GOOS).
		Set("arch", runtime.GOARCH).
		Set("source", cfg.Source).
		Set("$lib", libraryName).
		// Anonymous event: no person profile is created or updated.
		Set("$process_person_profile", false).
		// Discard the sender IP at ingestion; no geolocation is stored.
		Set("$ip", "")

	message := posthog.Capture{
		Event:      heartbeatEvent,
		DistinctId: id,
		Timestamp:  now.UTC(),
		Properties: props,
	}

	if err := c.Enqueue(message); err != nil {
		log.WithError(err).Debug("Telemetry: failed to enqueue heartbeat")
	}
}

// StartHeartbeat emits one anonymous install heartbeat immediately and then
// every 24 hours for the lifetime of the process. It is a no-op when
// WHODB_HEARTBEAT_DISABLED=true or when the PostHog client is not configured.
// Call once after Initialize.
func StartHeartbeat(opts datadir.Options) {
	if heartbeatDisabled() || !Configured() {
		return
	}

	id := installID(opts)
	sendHeartbeat(id, time.Now())

	go func() {
		ticker := time.NewTicker(heartbeatInterval)
		defer ticker.Stop()
		for now := range ticker.C {
			sendHeartbeat(id, now)
		}
	}()
}
