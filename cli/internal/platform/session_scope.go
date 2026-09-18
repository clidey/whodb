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

package platform

import (
	"fmt"
	"os"
	"strings"
)

const (
	// SessionHostEnv overrides the hosted platform host for the current process.
	SessionHostEnv = "WHODB_PLATFORM_SESSION_HOST"
	// SessionOrgEnv overrides the hosted organization for the current process.
	SessionOrgEnv = "WHODB_PLATFORM_SESSION_ORG"
	// SessionProjectEnv overrides the hosted project for the current process.
	SessionProjectEnv = "WHODB_PLATFORM_SESSION_PROJECT"
)

// SessionScope describes process-local hosted platform workspace overrides.
type SessionScope struct {
	Host    string
	Org     string
	Project string
}

// SessionScopeFromEnvironment loads process-local hosted platform workspace overrides.
func SessionScopeFromEnvironment() SessionScope {
	return SessionScope{
		Host:    strings.TrimSpace(os.Getenv(SessionHostEnv)),
		Org:     strings.TrimSpace(os.Getenv(SessionOrgEnv)),
		Project: strings.TrimSpace(os.Getenv(SessionProjectEnv)),
	}
}

// HasWorkspace reports whether the scope selects an organization and project.
func (s SessionScope) HasWorkspace() bool {
	return strings.TrimSpace(s.Org) != "" && strings.TrimSpace(s.Project) != ""
}

// Validate rejects partial workspace overrides that could target an unintended project.
func (s SessionScope) Validate() error {
	hasOrg := strings.TrimSpace(s.Org) != ""
	hasProject := strings.TrimSpace(s.Project) != ""
	if hasOrg != hasProject {
		return fmt.Errorf("%s and %s must be set together", SessionOrgEnv, SessionProjectEnv)
	}
	return nil
}
