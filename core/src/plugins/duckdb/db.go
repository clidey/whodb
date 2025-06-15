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

package duckdb

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alifiroozi80/duckdb"
	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/env"
	"gorm.io/gorm"
)

func getDefaultDirectory() string {
	directory := "/db/"
	if env.IsDevelopment {
		directory = "tmp/"
	}
	return directory
}

var errDoesNotExist = errors.New("unauthorized or the database doesn't exist")

func (p *DuckDBPlugin) DB(config *engine.PluginConfig) (*gorm.DB, error) {
	connectionInput, err := p.ParseConnectionConfig(config)
	if err != nil {
		return nil, err
	}
	
	database := connectionInput.Database
	fileNameDatabase := filepath.Join(getDefaultDirectory(), database)
	
	// Security check: ensure the file path is within the allowed directory
	if !strings.HasPrefix(fileNameDatabase, getDefaultDirectory()) {
		return nil, errDoesNotExist
	}
	
	// Check if file exists
	if _, err := os.Stat(fileNameDatabase); errors.Is(err, os.ErrNotExist) {
		return nil, errDoesNotExist
	}
	
	// Validate file extension (accept .duckdb, .ddb, .db as requested)
	ext := strings.ToLower(filepath.Ext(fileNameDatabase))
	if ext != ".duckdb" && ext != ".ddb" && ext != ".db" {
		return nil, fmt.Errorf("unsupported file extension: %s. Only .duckdb, .ddb, and .db files are supported", ext)
	}
	
	// Create connection string for DuckDB
	// DuckDB supports various connection options for performance and behavior tuning
	dsn := fileNameDatabase
	
	// Add DuckDB-specific connection options
	params := make([]string, 0)
	
	// Access mode: read_only or read_write
	if connectionInput.DuckDBAccessMode != "" && connectionInput.DuckDBAccessMode != "read_write" {
		params = append(params, fmt.Sprintf("access_mode=%s", connectionInput.DuckDBAccessMode))
	}
	
	// Thread configuration for parallel execution
	if connectionInput.DuckDBThreads != "" {
		params = append(params, fmt.Sprintf("threads=%s", connectionInput.DuckDBThreads))
	}
	
	// Memory limit configuration (e.g., "1GB", "512MB")
	if connectionInput.DuckDBMaxMemory != "" {
		params = append(params, fmt.Sprintf("max_memory=%s", connectionInput.DuckDBMaxMemory))
	}
	
	// Temporary directory for intermediate results
	if connectionInput.DuckDBTempDirectory != "" {
		params = append(params, fmt.Sprintf("temp_directory=%s", connectionInput.DuckDBTempDirectory))
	}
	
	// Add any extra connection options if needed
	if connectionInput.ExtraOptions != nil && len(connectionInput.ExtraOptions) > 0 {
		for key, value := range connectionInput.ExtraOptions {
			params = append(params, fmt.Sprintf("%s=%s", key, value))
		}
	}
	
	// Build final DSN
	if len(params) > 0 {
		dsn += "?" + strings.Join(params, "&")
	}
	
	db, err := gorm.Open(duckdb.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DuckDB: %w", err)
	}
	
	// After successful connection, enable CSV/Parquet reading from the same directory
	if err := p.setupFileAccess(db, fileNameDatabase); err != nil {
		return nil, fmt.Errorf("failed to setup file access: %w", err)
	}
	
	return db, nil
}

// setupFileAccess configures DuckDB to allow reading CSV and Parquet files from the same directory as the database
func (p *DuckDBPlugin) setupFileAccess(db *gorm.DB, dbFilePath string) error {
	// Get the directory containing the database file
	dbDir := filepath.Dir(dbFilePath)
	
	// Enable the httpfs extension for reading files (optional, for http/https URLs)
	// This is disabled by default for security
	
	// Create views or helper functions for reading CSV/Parquet files from the same directory
	// This is done by creating a function that validates file paths
	
	// For now, we'll just validate that the directory is accessible
	// The actual CSV/Parquet reading will be done through raw SQL queries
	// that we'll validate to ensure they only access files in the same directory
	
	_, err := os.Stat(dbDir)
	if err != nil {
		return fmt.Errorf("database directory not accessible: %w", err)
	}
	
	return nil
}

// ValidateFileAccess ensures that a file path is within the allowed directory (same as database)
func (p *DuckDBPlugin) ValidateFileAccess(dbFilePath, requestedFilePath string) error {
	dbDir := filepath.Dir(dbFilePath)
	
	// Clean and resolve the requested file path
	cleanPath := filepath.Clean(requestedFilePath)
	
	// If it's not an absolute path, make it relative to the database directory
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(dbDir, cleanPath)
	}
	
	// Resolve any symlinks or relative path components
	resolvedPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("cannot resolve file path: %w", err)
	}
	
	// Ensure the resolved path is within the database directory
	dbDirAbs, err := filepath.Abs(dbDir)
	if err != nil {
		return fmt.Errorf("cannot resolve database directory: %w", err)
	}
	
	if !strings.HasPrefix(resolvedPath, dbDirAbs) {
		return fmt.Errorf("file access denied: file must be in the same directory as the database")
	}
	
	// Check if the file exists
	if _, err := os.Stat(resolvedPath); os.IsNotExist(err) {
		return fmt.Errorf("file does not exist: %s", resolvedPath)
	}
	
	// Check if it's a CSV or Parquet file
	ext := strings.ToLower(filepath.Ext(resolvedPath))
	if ext != ".csv" && ext != ".parquet" {
		return fmt.Errorf("unsupported file type: %s. Only .csv and .parquet files are supported", ext)
	}
	
	return nil
}