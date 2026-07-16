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
	"testing"
	"time"

	"github.com/posthog/posthog-go"

	"github.com/clidey/whodb/core/src/common/datadir"
)

func TestSendHeartbeatUsesStableInstallID(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	client := &fakePosthogClient{}
	storeClient(client)
	cfg = Config{Edition: "ce", AppVersion: "1.2.3", Source: "backend"}

	sendHeartbeat("install-1", time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC))
	sendHeartbeat("install-1", time.Date(2026, 8, 20, 10, 0, 0, 0, time.UTC))

	if len(client.messages) != 2 {
		t.Fatalf("expected two heartbeats, got %d", len(client.messages))
	}
	first := client.messages[0].(posthog.Capture)
	second := client.messages[1].(posthog.Capture)
	if first.DistinctId != "install-1" || second.DistinctId != "install-1" {
		t.Fatalf("expected the stable install id across days, got %s / %s", first.DistinctId, second.DistinctId)
	}
}

func TestSendHeartbeatEmitsAnonymousMinimalEvent(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	client := &fakePosthogClient{}
	storeClient(client)
	// Heartbeat must send even when consent-gated analytics is disabled.
	enabled.Store(false)
	cfg = Config{Edition: "ce", AppVersion: "1.2.3", Source: "backend"}

	sendHeartbeat("install-1", time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC))

	if len(client.messages) != 1 {
		t.Fatalf("expected one heartbeat to be enqueued, got %d", len(client.messages))
	}
	capture, ok := client.messages[0].(posthog.Capture)
	if !ok {
		t.Fatalf("expected capture message, got %T", client.messages[0])
	}
	if capture.Event != heartbeatEvent {
		t.Fatalf("expected %s event, got %s", heartbeatEvent, capture.Event)
	}
	if capture.Properties["$process_person_profile"] != false {
		t.Fatalf("expected anonymous event with no person profile")
	}
	if capture.Properties["$ip"] != "" {
		t.Fatalf("expected sender ip to be discarded")
	}
	if capture.Properties["build_edition"] != "ce" || capture.Properties["source"] != "backend" {
		t.Fatalf("expected edition and source to be stamped")
	}
	if capture.Properties["dev_build"] != false {
		t.Fatalf("expected stamped version to mark dev_build=false")
	}
	for _, forbidden := range []string{"$host", "domain", "user_agent", "path", "error_message"} {
		if _, exists := capture.Properties[forbidden]; exists {
			t.Fatalf("heartbeat must not carry %s", forbidden)
		}
	}
}

func TestStartHeartbeatRespectsKillSwitch(t *testing.T) {
	t.Cleanup(resetAnalyticsState)
	t.Setenv(HeartbeatDisabledEnv, "true")

	client := &fakePosthogClient{}
	storeClient(client)
	cfg = Config{Edition: "ce", AppVersion: "1.2.3", Source: "backend"}

	// The kill switch is checked before any config access, so empty options are safe.
	StartHeartbeat(datadir.Options{})

	if len(client.messages) != 0 {
		t.Fatalf("expected no heartbeat when %s=true", HeartbeatDisabledEnv)
	}
}

func TestStartHeartbeatSkipsCI(t *testing.T) {
	t.Cleanup(resetAnalyticsState)
	t.Setenv(HeartbeatDisabledEnv, "")
	t.Setenv("CI", "true")

	client := &fakePosthogClient{}
	storeClient(client)
	cfg = Config{Edition: "ce", AppVersion: "1.2.3", Source: "backend"}

	StartHeartbeat(datadir.Options{})

	if len(client.messages) != 0 {
		t.Fatalf("expected no heartbeat in CI")
	}
}

func TestSendHeartbeatTagsDevBuilds(t *testing.T) {
	t.Cleanup(resetAnalyticsState)

	client := &fakePosthogClient{}
	storeClient(client)
	cfg = Config{Edition: "ce", AppVersion: "", Source: "backend"}

	sendHeartbeat("install-1", time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC))

	capture := client.messages[0].(posthog.Capture)
	if capture.Properties["app_version"] != "dev" || capture.Properties["dev_build"] != true {
		t.Fatalf("expected unstamped version to be tagged as dev build, got %v / %v",
			capture.Properties["app_version"], capture.Properties["dev_build"])
	}
}
