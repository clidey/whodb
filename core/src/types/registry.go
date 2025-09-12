/*
 * Copyright 2025 Clidey, Inc.
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

package types

import (
	"fmt"
	"strings"
	"sync"
)

// TypeCategory represents the category of a data type
type TypeCategory string

const (
	TypeCategoryNumeric  TypeCategory = "numeric"
	TypeCategoryText     TypeCategory = "text"
	TypeCategoryDate     TypeCategory = "date"
	TypeCategoryBoolean  TypeCategory = "boolean"
	TypeCategoryBinary   TypeCategory = "binary"
	TypeCategoryJSON     TypeCategory = "json"
	TypeCategoryGeometry TypeCategory = "geometry"
	TypeCategoryArray    TypeCategory = "array"
	TypeCategoryOther    TypeCategory = "other"
)

// TypeDefinition defines a database type and its conversion functions
type TypeDefinition struct {
	Name     string
	Category TypeCategory
	SQLTypes []string // List of SQL type names that map to this type

	// Conversion functions
	FromString func(string) (any, error)
	ToString   func(any) (string, error)

	// Validation
	Validator func(string) error

	// Frontend hints
	InputType     string // HTML input type: "number", "text", "date", etc.
	Icon          string // Icon identifier for UI
	EditComponent string // Component to use for editing
}

// TypeRegistry manages all type definitions
type TypeRegistry struct {
	mu    sync.RWMutex
	types map[string]*TypeDefinition // key is uppercase SQL type name
}

// NewTypeRegistry creates a new type registry
func NewTypeRegistry() *TypeRegistry {
	return &TypeRegistry{
		types: make(map[string]*TypeDefinition),
	}
}

// RegisterType registers a new type definition
func (r *TypeRegistry) RegisterType(def *TypeDefinition) error {
	if def == nil || def.Name == "" {
		return fmt.Errorf("invalid type definition")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Register all SQL type variants
	for _, sqlType := range def.SQLTypes {
		upperType := strings.ToUpper(sqlType)
		r.types[upperType] = def
	}

	return nil
}

// GetType retrieves a type definition by SQL type name
func (r *TypeRegistry) GetType(sqlType string) (*TypeDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	upperType := strings.ToUpper(sqlType)
	def, exists := r.types[upperType]
	return def, exists
}

// GetTypeCategory returns the category for a SQL type
func (r *TypeRegistry) GetTypeCategory(sqlType string) TypeCategory {
	if def, exists := r.GetType(sqlType); exists {
		return def.Category
	}
	return TypeCategoryOther
}

// OverrideType allows plugins to override a type definition
func (r *TypeRegistry) OverrideType(sqlType string, def *TypeDefinition) error {
	if def == nil || sqlType == "" {
		return fmt.Errorf("invalid override parameters")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	upperType := strings.ToUpper(sqlType)
	r.types[upperType] = def
	return nil
}

func (r *TypeRegistry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.types = make(map[string]*TypeDefinition)
}

// GetAllTypes returns all registered type definitions
func (r *TypeRegistry) GetAllTypes() map[string]*TypeDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*TypeDefinition, len(r.types))
	for k, v := range r.types {
		result[k] = v
	}
	return result
}
