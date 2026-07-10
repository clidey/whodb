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

package database

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/source"
	"github.com/clidey/whodb/core/src/sourcecatalog"
)

type sourceQueryStreamWriterAdapter struct {
	writer engine.QueryStreamWriter
}

func (w *sourceQueryStreamWriterAdapter) WriteColumns(columns []source.Column) error {
	return w.writer.WriteColumns(columns)
}

func (w *sourceQueryStreamWriterAdapter) WriteRow(row []string) error {
	return w.writer.WriteRow(row)
}

func (m *Manager) currentSourceSession(ctx context.Context) (source.TypeSpec, source.SourceSession, error) {
	if m.currentConnection == nil {
		return source.TypeSpec{}, nil, fmt.Errorf("not connected to any database")
	}

	return m.openSourceSession(ctx, m.currentConnection)
}

func (m *Manager) openSourceSession(ctx context.Context, conn *Connection) (source.TypeSpec, source.SourceSession, error) {
	spec, ok := sourcecatalog.Find(conn.Type)
	if !ok {
		return source.TypeSpec{}, nil, fmt.Errorf("unsupported source type")
	}

	credentials, err := m.sourceCredentials(conn, spec)
	if err != nil {
		return source.TypeSpec{}, nil, err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	session, err := source.Open(ctx, spec, credentials)
	if err != nil {
		return source.TypeSpec{}, nil, err
	}

	return spec, session, nil
}

func (m *Manager) sourceCredentials(conn *Connection, spec source.TypeSpec) (*source.Credentials, error) {
	values := map[string]string{}

	for _, field := range spec.ConnectionFields {
		switch field.CredentialField {
		case source.CredentialFieldHostname:
			if value := strings.TrimSpace(conn.Host); value != "" {
				values[field.Key] = value
			}
		case source.CredentialFieldUsername:
			if value := strings.TrimSpace(conn.Username); value != "" {
				values[field.Key] = value
			}
		case source.CredentialFieldPassword:
			if conn.Password != "" {
				values[field.Key] = conn.Password
			}
		case source.CredentialFieldDatabase:
			if value := strings.TrimSpace(conn.Database); value != "" {
				values[field.Key] = value
			}
		case source.CredentialFieldAdvanced:
			if strings.EqualFold(field.Key, "Port") && conn.Port > 0 {
				values[field.Key] = strconv.Itoa(conn.Port)
				continue
			}

			candidates := []string{field.Key}
			if field.AdvancedKey != "" {
				candidates = append([]string{field.AdvancedKey}, candidates...)
			}
			for _, candidate := range candidates {
				if value := strings.TrimSpace(conn.Advanced[candidate]); value != "" {
					values[field.Key] = value
					break
				}
			}
		}
	}

	if conn.Port > 0 {
		if _, ok := values["Port"]; !ok {
			values["Port"] = strconv.Itoa(conn.Port)
		}
	}

	for key, value := range conn.Advanced {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if _, exists := values[key]; exists {
			continue
		}
		values[key] = value
	}

	return &source.Credentials{
		SourceType: spec.ID,
		Values:     values,
		IsProfile:  conn.IsProfile,
	}, nil
}

func (m *Manager) namespaceKind(spec source.TypeSpec) (source.ObjectKind, bool) {
	defaultIndex := slices.Index(spec.Contract.BrowsePath, spec.Contract.DefaultObjectKind)
	if defaultIndex <= 0 {
		return "", false
	}

	return spec.Contract.BrowsePath[defaultIndex-1], true
}

func (m *Manager) scopeKind(spec source.TypeSpec) (source.ObjectKind, bool) {
	if spec.Contract.GraphScopeKind != nil {
		return *spec.Contract.GraphScopeKind, true
	}

	return m.namespaceKind(spec)
}

func (m *Manager) sourcePathForKind(spec source.TypeSpec, targetKind source.ObjectKind, namespace string, objectName string) ([]string, error) {
	targetIndex := slices.Index(spec.Contract.BrowsePath, targetKind)
	if targetIndex < 0 {
		return nil, fmt.Errorf("%s is not supported for source type %s", targetKind, spec.ID)
	}

	path := make([]string, 0, targetIndex+1)
	for _, kind := range spec.Contract.BrowsePath[:targetIndex] {
		value, err := m.sourceValueForKind(spec, kind, namespace)
		if err != nil {
			return nil, err
		}
		path = append(path, value)
	}

	if targetKind == spec.Contract.DefaultObjectKind {
		if strings.TrimSpace(objectName) == "" {
			return nil, fmt.Errorf("%s name is required", targetKind)
		}
		path = append(path, objectName)
		return path, nil
	}

	value, err := m.sourceValueForKind(spec, targetKind, namespace)
	if err != nil {
		return nil, err
	}
	path = append(path, value)
	return path, nil
}

func (m *Manager) sourceValueForKind(spec source.TypeSpec, kind source.ObjectKind, namespace string) (string, error) {
	namespaceKind, hasNamespaceKind := m.namespaceKind(spec)
	if hasNamespaceKind && kind == namespaceKind {
		if value := strings.TrimSpace(namespace); value != "" {
			return value, nil
		}
	}

	if m.currentConnection == nil {
		return "", fmt.Errorf("not connected to any database")
	}

	switch kind {
	case source.ObjectKindDatabase:
		if value := strings.TrimSpace(m.currentConnection.Database); value != "" {
			return value, nil
		}
	case source.ObjectKindSchema:
		if value := strings.TrimSpace(m.currentConnection.Schema); value != "" {
			return value, nil
		}
	}

	if hasNamespaceKind && kind == namespaceKind {
		return "", fmt.Errorf("%s is required for %s", kind, spec.Label)
	}

	return "", fmt.Errorf("cannot resolve %s path for %s", kind, spec.Label)
}

func (m *Manager) storageUnitParentRef(spec source.TypeSpec, namespace string) (*source.ObjectRef, error) {
	defaultIndex := slices.Index(spec.Contract.BrowsePath, spec.Contract.DefaultObjectKind)
	if defaultIndex <= 0 {
		return nil, nil
	}

	kind := spec.Contract.BrowsePath[defaultIndex-1]
	path, err := m.sourcePathForKind(spec, kind, namespace, "")
	if err != nil {
		return nil, err
	}
	ref := source.NewObjectRef(kind, path)
	return new(ref), nil
}

func (m *Manager) storageUnitRef(spec source.TypeSpec, namespace string, storageUnit string) (source.ObjectRef, error) {
	path, err := m.sourcePathForKind(spec, spec.Contract.DefaultObjectKind, namespace, storageUnit)
	if err != nil {
		return source.ObjectRef{}, err
	}

	return source.NewObjectRef(spec.Contract.DefaultObjectKind, path), nil
}

func (m *Manager) sourceScopeRef(spec source.TypeSpec, namespace string) (*source.ObjectRef, error) {
	if strings.TrimSpace(namespace) == "" {
		return nil, nil
	}

	kind, ok := m.scopeKind(spec)
	if !ok {
		return nil, nil
	}

	path, err := m.sourcePathForKind(spec, kind, namespace, "")
	if err != nil {
		return nil, err
	}

	ref := source.NewObjectRef(kind, path)
	return new(ref), nil
}

func (m *Manager) listNamespaceObjects(ctx context.Context, spec source.TypeSpec, session source.SourceSession) ([]source.Object, error) {
	kind, ok := m.namespaceKind(spec)
	if !ok {
		return []source.Object{}, nil
	}

	browser, ok := session.(source.SourceBrowser)
	if !ok {
		return nil, fmt.Errorf("browsing is not supported for %s", spec.Label)
	}

	index := slices.Index(spec.Contract.BrowsePath, kind)
	if index <= 0 {
		return browser.ListObjects(ctx, nil, []source.ObjectKind{kind})
	}

	parentKind := spec.Contract.BrowsePath[index-1]
	parentPath, err := m.sourcePathForKind(spec, parentKind, "", "")
	if err != nil {
		return nil, err
	}

	parentRef := source.NewObjectRef(parentKind, parentPath)
	return browser.ListObjects(ctx, new(parentRef), []source.ObjectKind{kind})
}

func (m *Manager) listStorageUnitObjects(ctx context.Context, spec source.TypeSpec, session source.SourceSession, namespace string) ([]source.Object, error) {
	browser, ok := session.(source.SourceBrowser)
	if !ok {
		return nil, fmt.Errorf("browsing is not supported for %s", spec.Label)
	}

	parent, err := m.storageUnitParentRef(spec, namespace)
	if err != nil {
		return nil, err
	}

	return browser.ListObjects(ctx, parent, nil)
}

func storageUnitsFromSourceObjects(objects []source.Object) []engine.StorageUnit {
	units := make([]engine.StorageUnit, 0, len(objects))
	for _, object := range objects {
		units = append(units, engine.StorageUnit{
			Name:       object.Name,
			Attributes: object.Metadata,
		})
	}
	return units
}

func cliChatMessagesFromSource(messages []*source.ChatMessage) []*ChatMessage {
	converted := make([]*ChatMessage, 0, len(messages))
	for _, message := range messages {
		converted = append(converted, &ChatMessage{
			Type:                 message.Type,
			Result:               message.Result,
			Text:                 message.Text,
			RequiresConfirmation: message.RequiresConfirmation,
		})
	}
	return converted
}
