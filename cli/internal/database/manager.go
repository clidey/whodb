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

package database

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/clidey/whodb/cli/internal/config"
	"github.com/clidey/whodb/core/graph/model"
	"github.com/clidey/whodb/core/src"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"github.com/clidey/whodb/core/src/llm"
	"github.com/xuri/excelize/v2"
)

type Connection = config.Connection

// DefaultCacheTTL is the default time-to-live for cached metadata
const DefaultCacheTTL = 5 * time.Minute

// MetadataCache provides thread-safe caching for database metadata
// to reduce network calls during autocomplete operations.
type MetadataCache struct {
	mu sync.RWMutex

	// schemas cache
	schemas     []string
	schemasTime time.Time

	// tables cache keyed by schema name
	tables     map[string][]engine.StorageUnit
	tablesTime map[string]time.Time

	// columns cache keyed by "schema.table"
	columns     map[string][]engine.Column
	columnsTime map[string]time.Time

	ttl time.Duration
}

// NewMetadataCache creates a new metadata cache with the specified TTL
func NewMetadataCache(ttl time.Duration) *MetadataCache {
	return &MetadataCache{
		tables:      make(map[string][]engine.StorageUnit),
		tablesTime:  make(map[string]time.Time),
		columns:     make(map[string][]engine.Column),
		columnsTime: make(map[string]time.Time),
		ttl:         ttl,
	}
}

// Clear removes all cached data
func (c *MetadataCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.schemas = nil
	c.schemasTime = time.Time{}
	c.tables = make(map[string][]engine.StorageUnit)
	c.tablesTime = make(map[string]time.Time)
	c.columns = make(map[string][]engine.Column)
	c.columnsTime = make(map[string]time.Time)
}

// GetSchemas returns cached schemas if valid, or nil if expired/missing
func (c *MetadataCache) GetSchemas() ([]string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.schemas == nil || time.Since(c.schemasTime) > c.ttl {
		return nil, false
	}
	return c.schemas, true
}

// SetSchemas caches the schema list
func (c *MetadataCache) SetSchemas(schemas []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.schemas = schemas
	c.schemasTime = time.Now()
}

// GetTables returns cached tables for a schema if valid, or nil if expired/missing
func (c *MetadataCache) GetTables(schema string) ([]engine.StorageUnit, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	tables, ok := c.tables[schema]
	if !ok {
		return nil, false
	}
	cacheTime, ok := c.tablesTime[schema]
	if !ok || time.Since(cacheTime) > c.ttl {
		return nil, false
	}
	return tables, true
}

// SetTables caches the tables for a schema
func (c *MetadataCache) SetTables(schema string, tables []engine.StorageUnit) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.tables[schema] = tables
	c.tablesTime[schema] = time.Now()
}

// GetColumns returns cached columns for a table if valid, or nil if expired/missing
func (c *MetadataCache) GetColumns(schema, table string) ([]engine.Column, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	key := schema + "." + table
	columns, ok := c.columns[key]
	if !ok {
		return nil, false
	}
	cacheTime, ok := c.columnsTime[key]
	if !ok || time.Since(cacheTime) > c.ttl {
		return nil, false
	}
	return columns, true
}

// SetColumns caches the columns for a table
func (c *MetadataCache) SetColumns(schema, table string, columns []engine.Column) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := schema + "." + table
	c.columns[key] = columns
	c.columnsTime[key] = time.Now()
}

type Manager struct {
	engine            *engine.Engine
	currentConnection *Connection
	config            *config.Config
	cache             *MetadataCache
}

func (m *Manager) buildCredentials(conn *Connection) *engine.Credentials {
	credentials := &engine.Credentials{
		Type:     conn.Type,
		Hostname: conn.Host,
		Username: conn.Username,
		Password: conn.Password,
		Database: conn.Database,
	}

	if conn.Port > 0 {
		credentials.Advanced = append(credentials.Advanced, engine.Record{
			Key:   "Port",
			Value: fmt.Sprintf("%d", conn.Port),
		})
	}

	return credentials
}

func NewManager() (*Manager, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	eng := src.InitializeEngine()

	return &Manager{
		engine: eng,
		config: cfg,
		cache:  NewMetadataCache(DefaultCacheTTL),
	}, nil
}

func (m *Manager) ListConnections() []Connection {
	return m.config.Connections
}

func (m *Manager) GetConnection(name string) (*Connection, error) {
	return m.config.GetConnection(name)
}

func (m *Manager) Connect(conn *Connection) error {
	dbType := engine.DatabaseType(conn.Type)

	credentials := m.buildCredentials(conn)

	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		// Don't expose database type in error for security
		return fmt.Errorf("unsupported database type")
	}

	pluginConfig := engine.NewPluginConfig(credentials)
	if !plugin.IsAvailable(pluginConfig) {
		// Don't expose connection details in error message for security
		return fmt.Errorf("cannot connect to database. please check your credentials and ensure the database is accessible")
	}

	// Clear cache when connecting to a new database
	m.cache.Clear()
	m.currentConnection = conn
	return nil
}

func (m *Manager) Disconnect() error {
	m.cache.Clear()
	m.currentConnection = nil
	return nil
}

// InvalidateCache clears all cached metadata, forcing fresh fetches on next access.
// Useful when the database schema has changed externally.
func (m *Manager) InvalidateCache() {
	m.cache.Clear()
}

// GetCache returns the metadata cache (primarily for testing)
func (m *Manager) GetCache() *MetadataCache {
	return m.cache
}

func (m *Manager) GetCurrentConnection() *Connection {
	return m.currentConnection
}

func (m *Manager) GetSchemas() ([]string, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	// Check cache first
	if cached, ok := m.cache.GetSchemas(); ok {
		return cached, nil
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)

	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)
	pluginConfig := engine.NewPluginConfig(credentials)
	schemas, err := plugin.GetAllSchemas(pluginConfig)
	if err != nil {
		return nil, err
	}

	// Cache the result
	m.cache.SetSchemas(schemas)
	return schemas, nil
}

func (m *Manager) GetStorageUnits(schema string) ([]engine.StorageUnit, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	// Check cache first
	if cached, ok := m.cache.GetTables(schema); ok {
		return cached, nil
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)

	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)

	pluginConfig := engine.NewPluginConfig(credentials)
	tables, err := plugin.GetStorageUnits(pluginConfig, schema)
	if err != nil {
		return nil, err
	}

	// Cache the result
	m.cache.SetTables(schema, tables)
	return tables, nil
}

func (m *Manager) ExecuteQuery(query string) (*engine.GetRowsResult, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)

	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)

	pluginConfig := engine.NewPluginConfig(credentials)
	return plugin.RawExecute(pluginConfig, query)
}

func (m *Manager) GetRows(schema, storageUnit string, where *model.WhereCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)

	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)

	pluginConfig := engine.NewPluginConfig(credentials)
	return plugin.GetRows(pluginConfig, schema, storageUnit, where, nil, pageSize, pageOffset)
}

// ExecuteQueryWithContext executes a query with context support for cancellation and timeout.
// If the context is cancelled or times out, the function returns immediately with ctx.Err().
// Note: The underlying database operation may continue running; only the wait is cancelled.
func (m *Manager) ExecuteQueryWithContext(ctx context.Context, query string) (*engine.GetRowsResult, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)
	pluginConfig := engine.NewPluginConfig(credentials)

	// Use channels to receive result from goroutine
	type result struct {
		data *engine.GetRowsResult
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		data, err := plugin.RawExecute(pluginConfig, query)
		resultCh <- result{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		return res.data, res.err
	}
}

// GetRowsWithContext fetches rows with context support for cancellation and timeout.
// If the context is cancelled or times out, the function returns immediately with ctx.Err().
// Note: The underlying database operation may continue running; only the wait is cancelled.
func (m *Manager) GetRowsWithContext(ctx context.Context, schema, storageUnit string, where *model.WhereCondition, pageSize, pageOffset int) (*engine.GetRowsResult, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)
	pluginConfig := engine.NewPluginConfig(credentials)

	// Use channels to receive result from goroutine
	type result struct {
		data *engine.GetRowsResult
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		data, err := plugin.GetRows(pluginConfig, schema, storageUnit, where, nil, pageSize, pageOffset)
		resultCh <- result{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		return res.data, res.err
	}
}

// GetSchemasWithContext fetches schemas with context support for cancellation and timeout.
func (m *Manager) GetSchemasWithContext(ctx context.Context) ([]string, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)
	pluginConfig := engine.NewPluginConfig(credentials)

	type result struct {
		data []string
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		data, err := plugin.GetAllSchemas(pluginConfig)
		resultCh <- result{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		return res.data, res.err
	}
}

// GetStorageUnitsWithContext fetches storage units with context support for cancellation and timeout.
func (m *Manager) GetStorageUnitsWithContext(ctx context.Context, schema string) ([]engine.StorageUnit, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)
	pluginConfig := engine.NewPluginConfig(credentials)

	type result struct {
		data []engine.StorageUnit
		err  error
	}
	resultCh := make(chan result, 1)

	go func() {
		data, err := plugin.GetStorageUnits(pluginConfig, schema)
		resultCh <- result{data: data, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		return res.data, res.err
	}
}

// GetConfig returns the manager's configuration
func (m *Manager) GetConfig() *config.Config {
	return m.config
}

func (m *Manager) GetColumns(schema, storageUnit string) ([]engine.Column, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	// Check cache first
	if cached, ok := m.cache.GetColumns(schema, storageUnit); ok {
		return cached, nil
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)

	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)

	pluginConfig := engine.NewPluginConfig(credentials)
	columns, err := plugin.GetColumnsForTable(pluginConfig, schema, storageUnit)
	if err != nil {
		return nil, err
	}

	// Cache the result
	m.cache.SetColumns(schema, storageUnit, columns)
	return columns, nil
}

func (m *Manager) ExportToCSV(schema, storageUnit, filename, delimiter string) error {
	if m.currentConnection == nil {
		return fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)

	pluginConfig := engine.NewPluginConfig(credentials)

	// Get all rows
	result, err := plugin.GetRows(pluginConfig, schema, storageUnit, nil, nil, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %w", err)
	}

	// Write to a temp file first for atomic replace
	dir := filepath.Dir(filename)
	tmp, err := os.CreateTemp(dir, ".whodb-export-*.csv")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	writer := csv.NewWriter(tmp)
	delimRune := rune(delimiter[0])
	writer.Comma = delimRune

	// Write headers
	headers := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		headers[i] = col.Name
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Write rows
	for _, row := range result.Rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to flush CSV writer: %w", err)
	}
	_ = tmp.Sync()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	// Rename into place (replace if exists)
	if err := os.Rename(tmpPath, filename); err != nil {
		// On Windows, need to remove destination first
		_ = os.Remove(filename)
		if err2 := os.Rename(tmpPath, filename); err2 != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to save file: %w", err2)
		}
	}
	// Best-effort fsync of directory to persist rename
	syncDir(filepath.Dir(filename))
	_ = os.Chmod(filename, 0600)
	return nil
}

func (m *Manager) ExportToExcel(schema, storageUnit, filename string) error {
	if m.currentConnection == nil {
		return fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)

	pluginConfig := engine.NewPluginConfig(credentials)

	// Get all rows
	result, err := plugin.GetRows(pluginConfig, schema, storageUnit, nil, nil, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to fetch data: %w", err)
	}

	// Create Excel file
	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Sheet1"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet: %w", err)
	}

	// Write headers
	for i, col := range result.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, col.Name)
	}

	// Write rows
	for rowIdx, row := range result.Rows {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	f.SetActiveSheet(index)

	// Save to temp file then atomically replace
	dir := filepath.Dir(filename)
	tmp, err := os.CreateTemp(dir, ".whodb-export-*.xlsx")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	if err := f.SaveAs(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to save file: %w", err)
	}
	// Best-effort sync of the temp file's contents
	if tf, err := os.OpenFile(tmpPath, os.O_RDWR, 0); err == nil {
		_ = tf.Sync()
		_ = tf.Close()
	}
	if err := os.Rename(tmpPath, filename); err != nil {
		_ = os.Remove(filename)
		if err2 := os.Rename(tmpPath, filename); err2 != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to save file: %w", err2)
		}
	}
	// Best-effort fsync of directory to persist rename
	syncDir(filepath.Dir(filename))
	_ = os.Chmod(filename, 0600)
	return nil
}

func (m *Manager) ExportResultsToCSV(result *engine.GetRowsResult, filename, delimiter string) error {
	if result == nil {
		return fmt.Errorf("no results to export")
	}

	dir := filepath.Dir(filename)
	tmp, err := os.CreateTemp(dir, ".whodb-export-*.csv")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	writer := csv.NewWriter(tmp)
	delimRune := rune(delimiter[0])
	writer.Comma = delimRune
	// Flush explicitly before syncing/closing

	// Write headers
	headers := make([]string, len(result.Columns))
	for i, col := range result.Columns {
		headers[i] = col.Name
	}
	if err := writer.Write(headers); err != nil {
		return fmt.Errorf("failed to write headers: %w", err)
	}

	// Write rows
	for _, row := range result.Rows {
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("failed to write row: %w", err)
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to flush CSV writer: %w", err)
	}
	_ = tmp.Sync()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to close temp file: %w", err)
	}
	if err := os.Rename(tmpPath, filename); err != nil {
		_ = os.Remove(filename)
		if err2 := os.Rename(tmpPath, filename); err2 != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to save file: %w", err2)
		}
	}
	// Best-effort fsync of directory to persist rename
	syncDir(filepath.Dir(filename))
	_ = os.Chmod(filename, 0600)
	return nil
}

func (m *Manager) ExportResultsToExcel(result *engine.GetRowsResult, filename string) error {
	if result == nil {
		return fmt.Errorf("no results to export")
	}

	f := excelize.NewFile()
	defer f.Close()

	sheetName := "Sheet1"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return fmt.Errorf("failed to create sheet: %w", err)
	}

	// Write headers
	for i, col := range result.Columns {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, col.Name)
	}

	// Write rows
	for rowIdx, row := range result.Rows {
		for colIdx, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+2)
			f.SetCellValue(sheetName, cell, value)
		}
	}

	f.SetActiveSheet(index)

	dir := filepath.Dir(filename)
	tmp, err := os.CreateTemp(dir, ".whodb-export-*.xlsx")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	tmp.Close()
	if err := f.SaveAs(tmpPath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to save file: %w", err)
	}
	if tf, err := os.OpenFile(tmpPath, os.O_RDWR, 0); err == nil {
		_ = tf.Sync()
		_ = tf.Close()
	}
	if err := os.Rename(tmpPath, filename); err != nil {
		_ = os.Remove(filename)
		if err2 := os.Rename(tmpPath, filename); err2 != nil {
			_ = os.Remove(tmpPath)
			return fmt.Errorf("failed to save file: %w", err2)
		}
	}
	// Best-effort fsync of directory to persist rename
	syncDir(filepath.Dir(filename))
	_ = os.Chmod(filename, 0600)
	return nil
}

// syncDir attempts to fsync a directory so the rename of a file inside it is
// durably recorded on disk. Not all platforms support syncing directories; any
// resulting errors are ignored as this is a best-effort durability improvement.
func syncDir(dir string) {
	if dir == "" || dir == "." {
		return
	}
	if f, err := os.Open(dir); err == nil {
		_ = f.Sync()
		_ = f.Close()
	}
}

type AIProvider struct {
	Type       string
	ProviderId string
}

func (m *Manager) GetAIProviders() []AIProvider {
	providers := env.GetConfiguredChatProviders()
	aiProviders := []AIProvider{}
	for _, provider := range providers {
		aiProviders = append(aiProviders, AIProvider{
			Type:       provider.Type,
			ProviderId: provider.ProviderId,
		})
	}
	return aiProviders
}

func (m *Manager) GetAIModels(providerID, modelType, token string) ([]string, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	credentials := m.buildCredentials(m.currentConnection)
	config := engine.NewPluginConfig(credentials)

	config.ExternalModel = &engine.ExternalModel{
		Type: modelType,
	}

	if providerID != "" {
		providers := env.GetConfiguredChatProviders()
		for _, provider := range providers {
			if provider.ProviderId == providerID {
				config.ExternalModel.Token = provider.APIKey
				break
			}
		}
	} else if token != "" {
		config.ExternalModel.Token = token
	}

	return llm.Instance(config).GetSupportedModels()
}

// GetAIModelsWithContext fetches AI models with context support for timeout/cancellation
func (m *Manager) GetAIModelsWithContext(ctx context.Context, providerID, modelType, token string) ([]string, error) {
	type result struct {
		models []string
		err    error
	}
	resultCh := make(chan result, 1)

	go func() {
		models, err := m.GetAIModels(providerID, modelType, token)
		resultCh <- result{models: models, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		return res.models, res.err
	}
}

type ChatMessage struct {
	Type   string
	Result *engine.GetRowsResult
	Text   string
}

func (m *Manager) SendAIChat(providerID, modelType, token, schema, model, previousConversation, query string) ([]*ChatMessage, error) {
	if m.currentConnection == nil {
		return nil, fmt.Errorf("not connected to any database")
	}

	dbType := engine.DatabaseType(m.currentConnection.Type)
	plugin := m.engine.Choose(dbType)
	if plugin == nil {
		return nil, fmt.Errorf("plugin not found")
	}

	credentials := m.buildCredentials(m.currentConnection)
	config := engine.NewPluginConfig(credentials)

	if providerID != "" {
		providers := env.GetConfiguredChatProviders()
		for _, provider := range providers {
			if provider.ProviderId == providerID {
				config.ExternalModel = &engine.ExternalModel{
					Type:  modelType,
					Token: provider.APIKey,
				}
				break
			}
		}
	} else {
		config.ExternalModel = &engine.ExternalModel{
			Type: modelType,
		}
		if token != "" {
			config.ExternalModel.Token = token
		}
	}

	messages, err := plugin.Chat(config, schema, model, previousConversation, query)
	if err != nil {
		return nil, err
	}

	chatMessages := []*ChatMessage{}
	for _, msg := range messages {
		chatMessages = append(chatMessages, &ChatMessage{
			Type:   msg.Type,
			Result: msg.Result,
			Text:   msg.Text,
		})
	}

	return chatMessages, nil
}

// SendAIChatWithContext sends AI chat with context support for timeout/cancellation
func (m *Manager) SendAIChatWithContext(ctx context.Context, providerID, modelType, token, schema, model, previousConversation, query string) ([]*ChatMessage, error) {
	type result struct {
		messages []*ChatMessage
		err      error
	}
	resultCh := make(chan result, 1)

	go func() {
		messages, err := m.SendAIChat(providerID, modelType, token, schema, model, previousConversation, query)
		resultCh <- result{messages: messages, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-resultCh:
		return res.messages, res.err
	}
}
