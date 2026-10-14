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

package cmd

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/cli/internal/platform"
)

// workspaceTestServer answers the workspace lookups and counts every request,
// so a test can assert that a command path made none.
func workspaceTestServer(t *testing.T, version string) (*platform.Client, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		var request struct {
			Query string `json:"query"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.Contains(request.Query, "CLIPlatformVersion"):
			_, _ = w.Write([]byte(`{"data":{"Version":"` + version + `"}}`))
		case strings.Contains(request.Query, "MyOrganizations"):
			_, _ = w.Write([]byte(`{"data":{"MyOrganizations":[{"id":"org-1","name":"Clidey","slug":"clidey"}]}}`))
		case strings.Contains(request.Query, "Projects"):
			_, _ = w.Write([]byte(`{"data":{"Projects":[{"id":"proj-1","orgId":"org-1","name":"Source","slug":"source"},{"id":"proj-2","orgId":"org-1","name":"Target","slug":"target"}]}}`))
		default:
			t.Errorf("unexpected query: %s", request.Query)
		}
	}))
	t.Cleanup(server.Close)
	client, err := platform.NewClient(server.URL, "token")
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	return client, &requests
}

func TestResolvePlatformProjectUsesSavedDefaultWithoutLookups(t *testing.T) {
	client, requests := workspaceTestServer(t, "1.2.3")
	session := &platformSession{
		Host:   config.PlatformHost{URL: "https://app.whodb.com", DefaultOrgID: "org-1", DefaultOrgName: "Clidey", DefaultProjectID: "proj-1", DefaultProjectName: "Source"},
		Client: client,
	}

	org, project, err := resolvePlatformProject(context.Background(), session, "", "")
	if err != nil {
		t.Fatalf("resolve default: %v", err)
	}
	if org.ID != "org-1" || org.Name != "Clidey" || project.ID != "proj-1" || project.Name != "Source" {
		t.Fatalf("unexpected default workspace: org=%+v project=%+v", org, project)
	}
	if requests.Load() != 0 {
		t.Fatalf("saved default must not list organizations or projects, made %d requests", requests.Load())
	}

	// Naming a project still resolves it against the server and carries slugs.
	org, project, err = resolvePlatformProject(context.Background(), session, "", "target")
	if err != nil {
		t.Fatalf("resolve named project: %v", err)
	}
	if project.ID != "proj-2" || project.Slug != "target" || org.Slug != "clidey" || requests.Load() != 2 {
		t.Fatalf("expected a detailed lookup (2 requests), got project=%+v org=%+v requests=%d", project, org, requests.Load())
	}

	// The detailed variant always lists, so slug-dependent callers are safe.
	before := requests.Load()
	if _, project, err := resolvePlatformProjectDetailed(context.Background(), session, "", ""); err != nil || project.Slug != "source" {
		t.Fatalf("detailed resolve: project=%+v err=%v", project, err)
	}
	if requests.Load() != before+2 {
		t.Fatalf("detailed resolve must list organizations and projects, made %d requests", requests.Load()-before)
	}

	// A default without names falls back to the lookup.
	session.Host.DefaultProjectName = ""
	before = requests.Load()
	if _, _, err := resolvePlatformProject(context.Background(), session, "", ""); err != nil {
		t.Fatalf("resolve unnamed default: %v", err)
	}
	if requests.Load() == before {
		t.Fatal("a default without a saved name must be resolved against the server")
	}
}

func TestResolveDefaultOrganizationSkipsLookup(t *testing.T) {
	client, requests := workspaceTestServer(t, "1.2.3")
	session := &platformSession{Host: config.PlatformHost{DefaultOrgID: "org-1", DefaultOrgName: "Clidey"}, Client: client}
	org, err := resolveDefaultOrganization(context.Background(), session, "")
	if err != nil || org.ID != "org-1" || requests.Load() != 0 {
		t.Fatalf("expected the saved organization without a request, got %+v err=%v requests=%d", org, err, requests.Load())
	}
	if _, err := resolveDefaultOrganization(context.Background(), session, "clidey"); err != nil || requests.Load() != 1 {
		t.Fatalf("a named organization must be looked up, err=%v requests=%d", err, requests.Load())
	}
}

func TestResolveCachedPlatformManifestChecksVersionOnlyAfterInterval(t *testing.T) {
	raw, _ := json.Marshal(platform.PlatformManifest{PlatformVersion: "1.2.3", ManifestProtocolVersion: "1"})
	fresh := func(checkedAt time.Time) *config.PlatformManifestCache {
		return &config.PlatformManifestCache{PlatformVersion: "1.2.3", ManifestProtocolVersion: "1", CheckedAt: checkedAt.UTC().Format(time.RFC3339), Raw: raw}
	}

	client, requests := workspaceTestServer(t, "1.2.3")
	cache := fresh(time.Now())
	if manifest, verified := resolveCachedPlatformManifest(context.Background(), client, cache); manifest == nil || verified || requests.Load() != 0 {
		t.Fatalf("a recently checked cache must be used without a request: manifest=%v verified=%v requests=%d", manifest != nil, verified, requests.Load())
	}

	stale := fresh(time.Now().Add(-platformManifestCheckInterval - time.Minute))
	manifest, verified := resolveCachedPlatformManifest(context.Background(), client, stale)
	if manifest == nil || !verified || requests.Load() != 1 {
		t.Fatalf("a stale check must query the version once: manifest=%v verified=%v requests=%d", manifest != nil, verified, requests.Load())
	}
	if checked, err := time.Parse(time.RFC3339, stale.CheckedAt); err != nil || time.Since(checked) > time.Minute {
		t.Fatalf("expected CheckedAt to be refreshed, got %q", stale.CheckedAt)
	}

	// A cache with no check time (written before the field existed) is checked.
	legacy := &config.PlatformManifestCache{PlatformVersion: "1.2.3", Raw: raw}
	if _, verified := resolveCachedPlatformManifest(context.Background(), client, legacy); !verified || requests.Load() != 2 {
		t.Fatalf("legacy cache must be verified, verified=%v requests=%d", verified, requests.Load())
	}

	upgraded, _ := workspaceTestServer(t, "2.0.0")
	if manifest, _ := resolveCachedPlatformManifest(context.Background(), upgraded, fresh(time.Now().Add(-time.Hour))); manifest != nil {
		t.Fatal("a changed platform version must invalidate the cached manifest")
	}
}
