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

package source

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"time"

	coreaudit "github.com/clidey/whodb/core/src/audit"
)

var (
	driversMu sync.RWMutex
	drivers   = map[string]SourceConnector{}

	typesMu       sync.RWMutex
	registeredIDs []string
	registered    = map[string]TypeSpec{}
)

// RegisterDriver registers one source connector under a runtime driver id.
func RegisterDriver(id string, connector SourceConnector) {
	if id == "" || connector == nil {
		return
	}

	driversMu.Lock()
	defer driversMu.Unlock()
	drivers[id] = connector
}

// Open opens a source session through the registered runtime driver.
func Open(ctx context.Context, spec TypeSpec, credentials *Credentials) (SourceSession, error) {
	start := time.Now()
	driversMu.RLock()
	driver, ok := drivers[spec.DriverID]
	driversMu.RUnlock()
	if !ok {
		err := fmt.Errorf("unsupported source driver: %s", spec.DriverID)
		AuditScopeFromCredentials(spec, credentials).record(ctx, "source.open_session", start, err, map[string]any{
			"driver_id": spec.DriverID,
		})
		return nil, err
	}
	session, err := driver.Open(ctx, spec, credentials)
	details := map[string]any{
		"driver_id":             spec.DriverID,
		"has_credentials_id":    credentials != nil && credentials.ID != nil && strings.TrimSpace(*credentials.ID) != "",
		"credential_value_keys": len(credentialsValueKeys(credentials)),
	}
	if credentials != nil {
		details["source_type"] = strings.TrimSpace(credentials.SourceType)
	}
	AuditScopeFromCredentials(spec, credentials).record(ctx, "source.open_session", start, err, details)
	return session, err
}

// Invalidate clears cached runtime state for one source type and credential
// set when the owning driver supports lifecycle invalidation.
func Invalidate(ctx context.Context, spec TypeSpec, credentials *Credentials) error {
	start := time.Now()
	driversMu.RLock()
	driver, ok := drivers[spec.DriverID]
	driversMu.RUnlock()
	if !ok {
		err := fmt.Errorf("unsupported source driver: %s", spec.DriverID)
		AuditScopeFromCredentials(spec, credentials).record(ctx, "source.invalidate_session", start, err, map[string]any{
			"driver_id": spec.DriverID,
		})
		return err
	}

	invalidator, ok := driver.(SessionInvalidator)
	if !ok {
		AuditScopeFromCredentials(spec, credentials).record(ctx, "source.invalidate_session", start, nil, map[string]any{
			"driver_id":   spec.DriverID,
			"supported":   false,
			"source_type": strings.TrimSpace(spec.ID),
		})
		return nil
	}

	err := invalidator.Invalidate(ctx, spec, credentials)
	AuditScopeFromCredentials(spec, credentials).record(ctx, "source.invalidate_session", start, err, map[string]any{
		"driver_id": spec.DriverID,
		"supported": true,
	})
	return err
}

// Shutdown releases cached runtime state for every registered source driver
// that exposes process-wide shutdown behavior.
func Shutdown(ctx context.Context) error {
	start := time.Now()
	driversMu.RLock()
	driverList := make([]SourceConnector, 0, len(drivers))
	for _, driver := range drivers {
		driverList = append(driverList, driver)
	}
	driversMu.RUnlock()

	var shutdownErr error
	for _, driver := range driverList {
		shutdowner, ok := driver.(DriverShutdowner)
		if !ok {
			continue
		}
		shutdownErr = errors.Join(shutdownErr, shutdowner.Shutdown(ctx))
	}

	coreaudit.RecordWithContext(ctx, coreaudit.AuditEvent{
		Timestamp: start,
		Action:    "source.shutdown_drivers",
		Resource: coreaudit.Resource{
			ID:   "all",
			Type: "source_driver",
			Name: "all",
		},
		Details: map[string]any{
			"driver_count": len(driverList),
		},
		Error: func() string {
			if shutdownErr == nil {
				return ""
			}
			return shutdownErr.Error()
		}(),
	})

	return shutdownErr
}

func credentialsValueKeys(credentials *Credentials) []string {
	if credentials == nil || len(credentials.Values) == 0 {
		return nil
	}

	keys := make([]string, 0, len(credentials.Values))
	for key := range credentials.Values {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

// RegisterType registers or replaces one source type spec by id.
func RegisterType(spec TypeSpec) {
	if spec.ID == "" {
		return
	}

	typesMu.Lock()
	defer typesMu.Unlock()

	key := strings.ToLower(spec.ID)
	if _, exists := registered[key]; !exists {
		registeredIDs = append(registeredIDs, key)
	}
	registered[key] = cloneTypeSpec(spec)
}

// RegisteredTypes returns registered source type specs in registration order.
func RegisteredTypes() []TypeSpec {
	typesMu.RLock()
	defer typesMu.RUnlock()

	specs := make([]TypeSpec, 0, len(registeredIDs))
	for _, key := range registeredIDs {
		spec, ok := registered[key]
		if !ok {
			continue
		}
		specs = append(specs, cloneTypeSpec(spec))
	}
	return specs
}

// FindType resolves one registered source type by id using a case-insensitive match.
func FindType(id string) (TypeSpec, bool) {
	typesMu.RLock()
	defer typesMu.RUnlock()

	spec, ok := registered[strings.ToLower(id)]
	if !ok {
		return TypeSpec{}, false
	}
	return cloneTypeSpec(spec), true
}

func cloneTypeSpec(spec TypeSpec) TypeSpec {
	cloned := spec
	cloned.ConnectionFields = slices.Clone(spec.ConnectionFields)
	cloned.Contract = Contract{
		Model:             spec.Contract.Model,
		Surfaces:          slices.Clone(spec.Contract.Surfaces),
		RootActions:       slices.Clone(spec.Contract.RootActions),
		BrowsePath:        slices.Clone(spec.Contract.BrowsePath),
		DefaultObjectKind: spec.Contract.DefaultObjectKind,
		GraphScopeKind:    spec.Contract.GraphScopeKind,
		ObjectTypes:       cloneObjectTypes(spec.Contract.ObjectTypes),
	}
	cloned.DiscoveryPrefill = cloneDiscoveryPrefill(spec.DiscoveryPrefill)
	cloned.SSLModes = cloneSSLModes(spec.SSLModes)
	return cloned
}

func cloneObjectTypes(objectTypes []ObjectType) []ObjectType {
	cloned := make([]ObjectType, 0, len(objectTypes))
	for _, objectType := range objectTypes {
		cloned = append(cloned, ObjectType{
			Kind:          objectType.Kind,
			DataShape:     objectType.DataShape,
			Actions:       slices.Clone(objectType.Actions),
			Views:         slices.Clone(objectType.Views),
			SingularLabel: objectType.SingularLabel,
			PluralLabel:   objectType.PluralLabel,
		})
	}
	return cloned
}

func cloneDiscoveryPrefill(prefill DiscoveryPrefill) DiscoveryPrefill {
	cloned := DiscoveryPrefill{
		AdvancedDefaults: make([]DiscoveryAdvancedDefault, 0, len(prefill.AdvancedDefaults)),
	}
	for _, item := range prefill.AdvancedDefaults {
		cloned.AdvancedDefaults = append(cloned.AdvancedDefaults, DiscoveryAdvancedDefault{
			Key:           item.Key,
			Value:         item.Value,
			MetadataKey:   item.MetadataKey,
			DefaultValue:  item.DefaultValue,
			ProviderTypes: slices.Clone(item.ProviderTypes),
			Conditions:    slices.Clone(item.Conditions),
		})
	}
	return cloned
}

func cloneSSLModes(modes []SSLModeInfo) []SSLModeInfo {
	cloned := make([]SSLModeInfo, 0, len(modes))
	for _, mode := range modes {
		cloned = append(cloned, SSLModeInfo{
			Value:       mode.Value,
			Label:       mode.Label,
			Description: mode.Description,
			Aliases:     slices.Clone(mode.Aliases),
		})
	}
	return cloned
}
