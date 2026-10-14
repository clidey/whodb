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
	"slices"
	"strings"
	"sync"
	"time"

	platformapi "github.com/clidey/whodb/cli/internal/platform"
)

// Every tool call rebuilds its session, which used to mean fetching the
// platform manifest and the organization and project lists before the request
// the tool was actually made for. Those lookups are cached per process here.
// Keys always include the host and, for workspace lists, the signed-in
// account, so two hosts or two accounts never share an entry and a hosted pod
// serving many users keeps them apart. The tool's own query is never cached.
const (
	platformManifestCacheTTL  = 5 * time.Minute
	platformWorkspaceCacheTTL = time.Minute
)

type platformLookupEntry[T any] struct {
	value   T
	expires time.Time
}

type platformLookupCache struct {
	mu           sync.Mutex
	now          func() time.Time
	manifests    map[string]platformLookupEntry[*platformapi.PlatformManifest]
	orgs         map[string]platformLookupEntry[[]platformapi.Organization]
	projectLists map[string]platformLookupEntry[[]platformapi.Project]
}

var platformLookups = newPlatformLookupCache()

func newPlatformLookupCache() *platformLookupCache {
	c := &platformLookupCache{now: time.Now}
	c.reset()
	return c
}

// platformLookupMiss marks a resolution failure that a fresher list might fix,
// as opposed to an ambiguous or missing input.
type platformLookupMiss struct{ error }

func platformLookupKey(parts ...string) string {
	return strings.Join(parts, "\x00")
}

// manifest returns the host's platform manifest, fetching it at most once per
// TTL. The manifest is the same for every signed-in user of a host.
func (c *platformLookupCache) manifest(ctx context.Context, host string, fetch func(context.Context) (*platformapi.PlatformManifest, error)) (*platformapi.PlatformManifest, error) {
	c.mu.Lock()
	entry, ok := c.manifests[host]
	fresh := ok && c.now().Before(entry.expires)
	c.mu.Unlock()
	if fresh {
		return entry.value, nil
	}
	manifest, err := fetch(ctx)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweepLocked()
	c.manifests[host] = platformLookupEntry[*platformapi.PlatformManifest]{value: manifest, expires: c.now().Add(platformManifestCacheTTL)}
	return manifest, nil
}

// organizations returns the account's organizations on host and reports
// whether they came from the cache, so a caller can refetch once on a miss.
// Sessions without an account id are not signed in and bypass the cache.
func (c *platformLookupCache) organizations(ctx context.Context, client platformClient, host, account string, refresh bool) ([]platformapi.Organization, bool, error) {
	if account != "" && !refresh {
		c.mu.Lock()
		entry, ok := c.orgs[platformLookupKey(host, account)]
		fresh := ok && c.now().Before(entry.expires)
		c.mu.Unlock()
		if fresh {
			return slices.Clone(entry.value), true, nil
		}
	}
	orgs, err := client.Organizations(ctx)
	if err != nil {
		return nil, false, err
	}
	c.storeOrganizations(host, account, orgs)
	return orgs, false, nil
}

// projects is the organization-scoped counterpart of organizations.
func (c *platformLookupCache) projects(ctx context.Context, client platformClient, host, account, orgID string, refresh bool) ([]platformapi.Project, bool, error) {
	if account != "" && !refresh {
		c.mu.Lock()
		entry, ok := c.projectLists[platformLookupKey(host, account, orgID)]
		fresh := ok && c.now().Before(entry.expires)
		c.mu.Unlock()
		if fresh {
			return slices.Clone(entry.value), true, nil
		}
	}
	projects, err := client.Projects(ctx, orgID)
	if err != nil {
		return nil, false, err
	}
	c.storeProjects(host, account, orgID, projects)
	return projects, false, nil
}

// storeOrganizations records a freshly listed organization set. Listing tools
// call it so the resolvers see what the user was just shown.
func (c *platformLookupCache) storeOrganizations(host, account string, orgs []platformapi.Organization) {
	if account == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweepLocked()
	c.orgs[platformLookupKey(host, account)] = platformLookupEntry[[]platformapi.Organization]{value: slices.Clone(orgs), expires: c.now().Add(platformWorkspaceCacheTTL)}
}

// storeProjects records a freshly listed project set for one organization.
func (c *platformLookupCache) storeProjects(host, account, orgID string, projects []platformapi.Project) {
	if account == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sweepLocked()
	c.projectLists[platformLookupKey(host, account, orgID)] = platformLookupEntry[[]platformapi.Project]{value: slices.Clone(projects), expires: c.now().Add(platformWorkspaceCacheTTL)}
}

// invalidate drops the account's organization and project lists on host. It
// runs after every write so a created, renamed or deleted workspace resolves
// on the next call instead of after the TTL.
func (c *platformLookupCache) invalidate(host, account string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.orgs, platformLookupKey(host, account))
	prefix := platformLookupKey(host, account) + "\x00"
	for key := range c.projectLists {
		if strings.HasPrefix(key, prefix) {
			delete(c.projectLists, key)
		}
	}
}

// sweepLocked removes expired entries so a long-lived hosted process serving
// many accounts does not keep dead keys. The caller holds c.mu.
func (c *platformLookupCache) sweepLocked() {
	now := c.now()
	for key, entry := range c.manifests {
		if !now.Before(entry.expires) {
			delete(c.manifests, key)
		}
	}
	for key, entry := range c.orgs {
		if !now.Before(entry.expires) {
			delete(c.orgs, key)
		}
	}
	for key, entry := range c.projectLists {
		if !now.Before(entry.expires) {
			delete(c.projectLists, key)
		}
	}
}

// reset clears every entry. Tests share one process and many fake hosts.
func (c *platformLookupCache) reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.manifests = map[string]platformLookupEntry[*platformapi.PlatformManifest]{}
	c.orgs = map[string]platformLookupEntry[[]platformapi.Organization]{}
	c.projectLists = map[string]platformLookupEntry[[]platformapi.Project]{}
}

// cachedOrganizationByID finds one organization by id, refetching once when
// the cached list does not contain it. It returns nil when the id is unknown.
func cachedOrganizationByID(ctx context.Context, client platformClient, host, account, id string) (*platformapi.Organization, error) {
	for refresh := false; ; refresh = true {
		orgs, cached, err := platformLookups.organizations(ctx, client, host, account, refresh)
		if err != nil {
			return nil, err
		}
		for i := range orgs {
			if orgs[i].ID == id {
				return &orgs[i], nil
			}
		}
		if !cached {
			return nil, nil
		}
	}
}

// cachedProjectByID is the project counterpart of cachedOrganizationByID.
func cachedProjectByID(ctx context.Context, client platformClient, host, account, orgID, id string) (*platformapi.Project, error) {
	for refresh := false; ; refresh = true {
		projects, cached, err := platformLookups.projects(ctx, client, host, account, orgID, refresh)
		if err != nil {
			return nil, err
		}
		for i := range projects {
			if projects[i].ID == id {
				return &projects[i], nil
			}
		}
		if !cached {
			return nil, nil
		}
	}
}
