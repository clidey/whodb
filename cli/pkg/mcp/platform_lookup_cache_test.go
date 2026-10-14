// Copyright 2026 Clidey, Inc.
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

package mcp

import (
	"context"
	"testing"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// countingPlatformClient records how often the bookkeeping lookups hit the
// server and lets a test change the organization list between calls.
type countingPlatformClient struct {
	fakePlatformClient
	orgs          []platformapi.Organization
	orgCalls      int
	projectCalls  int
	manifestCalls int
}

func (c *countingPlatformClient) Organizations(context.Context) ([]platformapi.Organization, error) {
	c.orgCalls++
	return c.orgs, nil
}

func (c *countingPlatformClient) Projects(ctx context.Context, orgID string) ([]platformapi.Project, error) {
	c.projectCalls++
	return c.fakePlatformClient.Projects(ctx, orgID)
}

func (c *countingPlatformClient) PlatformManifest(ctx context.Context) (*platformapi.PlatformManifest, error) {
	c.manifestCalls++
	return c.fakePlatformClient.PlatformManifest(ctx)
}

func newCountingSession(account string, orgs ...platformapi.Organization) (*platformToolSession, *countingPlatformClient) {
	if orgs == nil {
		orgs = []platformapi.Organization{
			{ID: "org-1", Name: "Clidey", Slug: "clidey"},
			{ID: "org-2", Name: "Acme", Slug: "acme"},
		}
	}
	client := &countingPlatformClient{orgs: orgs}
	return &platformToolSession{Host: config.PlatformHost{URL: "https://app.whodb.com", AccountID: account}, Client: client}, client
}

func TestPlatformLookupCacheReusesWorkspaceListsPerAccount(t *testing.T) {
	platformLookups.reset()
	ctx := context.Background()
	session, client := newCountingSession("account-a")

	for range 3 {
		if _, _, err := resolvePlatformToolProject(ctx, session, "acme", "customer"); err != nil {
			t.Fatalf("resolve: %v", err)
		}
	}
	if client.orgCalls != 1 || client.projectCalls != 1 {
		t.Fatalf("expected one fetch each, got orgs=%d projects=%d", client.orgCalls, client.projectCalls)
	}

	// Another account on the same host has its own list: it sees its own
	// organization and does not see account-a's.
	other, otherClient := newCountingSession("account-b", platformapi.Organization{ID: "org-9", Name: "Globex", Slug: "globex"})
	if _, err := resolvePlatformToolOrg(ctx, other, "globex"); err != nil {
		t.Fatalf("resolve for second account: %v", err)
	}
	if _, err := resolvePlatformToolOrg(ctx, other, "acme"); err == nil {
		t.Fatal("second account resolved an organization that belongs to the first account's list")
	}
	if otherClient.orgCalls != 2 {
		t.Fatalf("expected the miss to refetch once for the second account, got %d calls", otherClient.orgCalls)
	}
}

func TestPlatformLookupCacheRefetchesOnceOnMiss(t *testing.T) {
	platformLookups.reset()
	ctx := context.Background()
	session, client := newCountingSession("account-a")

	if _, err := resolvePlatformToolOrg(ctx, session, "acme"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	client.orgs = append(client.orgs, platformapi.Organization{ID: "org-3", Name: "New Org", Slug: "new-org"})
	org, err := resolvePlatformToolOrg(ctx, session, "new-org")
	if err != nil {
		t.Fatalf("resolve newly created organization: %v", err)
	}
	if org.ID != "org-3" || client.orgCalls != 2 {
		t.Fatalf("expected org-3 after one refetch, got %+v with %d calls", org, client.orgCalls)
	}

	// A genuinely unknown name costs exactly one refetch, not a loop.
	if _, err := resolvePlatformToolOrg(ctx, session, "missing"); err == nil || client.orgCalls != 3 {
		t.Fatalf("expected not-found after one refetch, err=%v calls=%d", err, client.orgCalls)
	}
	// Ambiguity and missing input are not misses and must not refetch.
	if _, err := matchPlatformToolOrg(session, []platformapi.Organization{{ID: "a", Name: "Same"}, {ID: "b", Name: "Same"}}, "Same"); err == nil {
		t.Fatal("expected an ambiguity error")
	} else if _, isMiss := err.(platformLookupMiss); isMiss {
		t.Fatal("ambiguity must not be treated as a cache miss")
	}
}

func TestPlatformLookupCacheInvalidateDropsAccountLists(t *testing.T) {
	platformLookups.reset()
	ctx := context.Background()
	session, client := newCountingSession("account-a")

	if _, _, err := resolvePlatformToolProject(ctx, session, "acme", "customer"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	platformLookups.invalidate(session.Host.URL, session.Host.AccountID)
	if _, _, err := resolvePlatformToolProject(ctx, session, "acme", "customer"); err != nil {
		t.Fatalf("resolve after invalidate: %v", err)
	}
	if client.orgCalls != 2 || client.projectCalls != 2 {
		t.Fatalf("expected refetch after invalidate, got orgs=%d projects=%d", client.orgCalls, client.projectCalls)
	}
}

func TestPlatformLookupCacheExpires(t *testing.T) {
	ctx := context.Background()
	cache := newPlatformLookupCache()
	now := time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)
	cache.now = func() time.Time { return now }
	_, client := newCountingSession("account-a")

	for range 2 {
		if _, _, err := cache.organizations(ctx, client, "https://app.whodb.com", "account-a", false); err != nil {
			t.Fatalf("organizations: %v", err)
		}
		if _, err := cache.manifest(ctx, "https://app.whodb.com", client.PlatformManifest); err != nil {
			t.Fatalf("manifest: %v", err)
		}
	}
	if client.orgCalls != 1 || client.manifestCalls != 1 {
		t.Fatalf("expected one fetch each within the TTL, got orgs=%d manifest=%d", client.orgCalls, client.manifestCalls)
	}

	now = now.Add(platformWorkspaceCacheTTL + time.Second)
	if _, _, err := cache.organizations(ctx, client, "https://app.whodb.com", "account-a", false); err != nil {
		t.Fatalf("organizations: %v", err)
	}
	if _, err := cache.manifest(ctx, "https://app.whodb.com", client.PlatformManifest); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if client.orgCalls != 2 || client.manifestCalls != 1 {
		t.Fatalf("expected the workspace list to expire before the manifest, got orgs=%d manifest=%d", client.orgCalls, client.manifestCalls)
	}

	now = now.Add(platformManifestCacheTTL)
	if _, err := cache.manifest(ctx, "https://app.whodb.com", client.PlatformManifest); err != nil {
		t.Fatalf("manifest: %v", err)
	}
	if client.manifestCalls != 2 {
		t.Fatalf("expected the manifest to expire, got %d fetches", client.manifestCalls)
	}
}

func TestPlatformLookupCacheBypassesSessionsWithoutAccount(t *testing.T) {
	platformLookups.reset()
	ctx := context.Background()
	session, client := newCountingSession("")

	for range 2 {
		if _, err := resolvePlatformToolOrg(ctx, session, "acme"); err != nil {
			t.Fatalf("resolve: %v", err)
		}
	}
	if client.orgCalls != 2 {
		t.Fatalf("expected no caching without an account id, got %d calls", client.orgCalls)
	}
}
