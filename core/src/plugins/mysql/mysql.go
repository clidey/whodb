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

package mysql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/log"
	"github.com/clidey/whodb/core/src/plugins"
	gorm_plugin "github.com/clidey/whodb/core/src/plugins/gorm"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	supportedOperators = map[string]string{
		"=": "=", ">=": ">=", ">": ">", "<=": "<=", "<": "<", "<>": "<>",
		"!=": "!=", "BETWEEN": "BETWEEN", "NOT BETWEEN": "NOT BETWEEN",
		"LIKE": "LIKE", "NOT LIKE": "NOT LIKE", "IN": "IN", "NOT IN": "NOT IN",
		"IS NULL": "IS NULL", "IS NOT NULL": "IS NOT NULL", "AND": "AND", "OR": "OR", "NOT": "NOT",
	}
)

type MySQLPlugin struct {
	gorm_plugin.GormPlugin
}

func (p *MySQLPlugin) GetDatabases(config *engine.PluginConfig) ([]string, error) {
	return plugins.WithConnection(config, p.DB, func(db *gorm.DB) ([]string, error) {
		var allDBs []struct {
			Database string `gorm:"column:Database"`
		}
		if err := db.Raw("SHOW DATABASES").Scan(&allDBs).Error; err != nil {
			return nil, err
		}

		// Verify each database with USE — the only reliable access check in MySQL.
		// INFORMATION_SCHEMA privilege tables are unreliable due to GRANTEE format
		// mismatches, role-based grants, and server-specific defaults.
		// Cost is O(N) but runs once per profile (Apollo caches the result).
		sqlDB, err := db.DB()
		if err != nil {
			return nil, err
		}
		ctx := context.Background()
		conn, err := sqlDB.Conn(ctx)
		if err != nil {
			return nil, err
		}
		defer conn.Close()

		var currentDB sql.NullString
		conn.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&currentDB)

		accessible := make([]string, 0, len(allDBs))
		for _, d := range allDBs {
			escaped := strings.ReplaceAll(d.Database, "`", "``")
			if _, err := conn.ExecContext(ctx, "USE `"+escaped+"`"); err == nil {
				accessible = append(accessible, d.Database)
			}
		}

		if currentDB.Valid && currentDB.String != "" {
			escaped := strings.ReplaceAll(currentDB.String, "`", "``")
			conn.ExecContext(ctx, "USE `"+escaped+"`")
		}

		return accessible, nil
	})
}

func (p *MySQLPlugin) GetAllSchemasQuery() string {
	return "SELECT SCHEMA_NAME AS schemaname FROM INFORMATION_SCHEMA.SCHEMATA"
}

// GetLastInsertID returns the most recently auto-generated ID using MySQL's LAST_INSERT_ID().
func (p *MySQLPlugin) GetLastInsertID(db *gorm.DB) (int64, error) {
	var id int64
	if err := db.Raw("SELECT LAST_INSERT_ID()").Scan(&id).Error; err != nil {
		return 0, err
	}
	return id, nil
}

func (p *MySQLPlugin) GetSupportedOperators() map[string]string {
	return supportedOperators
}

func (p *MySQLPlugin) GetTableInfoQuery() string {
	return `
		SELECT
			TABLE_NAME,
			TABLE_TYPE,
			IFNULL(ROUND(((DATA_LENGTH + INDEX_LENGTH) / 1024 / 1024), 2), 0) AS total_size,
			IFNULL(ROUND((DATA_LENGTH / 1024 / 1024), 2), 0) AS data_size
		FROM
			INFORMATION_SCHEMA.TABLES
		WHERE
			TABLE_SCHEMA = ?`
}

func (p *MySQLPlugin) GetStorageUnitExistsQuery() string {
	return `SELECT EXISTS(SELECT 1 FROM INFORMATION_SCHEMA.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?)`
}

func (p *MySQLPlugin) GetPlaceholder(index int) string {
	return "?"
}

func (p *MySQLPlugin) GetTableNameAndAttributes(rows *sql.Rows) (string, []engine.Record) {
	var tableName, tableType string
	var totalSize, dataSize float64
	if err := rows.Scan(&tableName, &tableType, &totalSize, &dataSize); err != nil {
		log.WithError(err).Error("Failed to scan MySQL table information")
		return "", []engine.Record{}
	}

	attributes := []engine.Record{
		{Key: "Type", Value: tableType},
		{Key: "Total Size", Value: fmt.Sprintf("%.2f MB", totalSize)},
		{Key: "Data Size", Value: fmt.Sprintf("%.2f MB", dataSize)},
	}
	return tableName, attributes
}

func (p *MySQLPlugin) RawExecute(config *engine.PluginConfig, query string, params ...any) (*engine.GetRowsResult, error) {
	return p.ExecuteRawSQL(config, func(cfg *engine.PluginConfig) (*gorm.DB, error) {
		return p.openDB(cfg, true)
	}, query, params...)
}

// CreateSQLBuilder creates a MySQL-specific SQL builder
func (p *MySQLPlugin) CreateSQLBuilder(db *gorm.DB) gorm_plugin.SQLBuilderInterface {
	return NewMySQLSQLBuilder(db, p)
}

func (p *MySQLPlugin) GetForeignKeyRelationships(config *engine.PluginConfig, schema string, storageUnit string) (map[string]*engine.ForeignKeyRelationship, error) {
	query := `
		SELECT
			COLUMN_NAME,
			REFERENCED_TABLE_NAME,
			REFERENCED_COLUMN_NAME
		FROM INFORMATION_SCHEMA.KEY_COLUMN_USAGE
		WHERE TABLE_SCHEMA = ?
			AND TABLE_NAME = ?
			AND REFERENCED_TABLE_NAME IS NOT NULL
	`
	return p.QueryForeignKeyRelationships(config, query, schema, storageUnit)
}

// NormalizeType converts MySQL type aliases to their canonical form.
func (p *MySQLPlugin) NormalizeType(typeName string) string {
	return NormalizeType(typeName)
}

// MarkGeneratedColumns detects MySQL generated columns (VIRTUAL or STORED)
// and marks them as IsComputed.
func (p *MySQLPlugin) MarkGeneratedColumns(config *engine.PluginConfig, schema string, storageUnit string, columns []engine.Column) error {
	computed, err := p.QueryComputedColumns(config, `
		SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? AND GENERATION_EXPRESSION IS NOT NULL AND GENERATION_EXPRESSION != ''
	`, schema, storageUnit)
	if err != nil {
		return err
	}

	for i := range columns {
		if computed[columns[i].Name] {
			columns[i].IsComputed = true
		}
	}
	return nil
}

// BuildSkipConflictClause returns ON DUPLICATE KEY UPDATE pk = pk for MySQL.
// MySQL's GORM driver can't generate the id=id fallback without schema info when
// using .Table() with map records, so we provide explicit identity assignments.
func (p *MySQLPlugin) BuildSkipConflictClause(pkColumns []string) clause.OnConflict {
	conflictCols := make([]clause.Column, len(pkColumns))
	assignments := make([]clause.Assignment, len(pkColumns))
	for i, col := range pkColumns {
		c := clause.Column{Name: col}
		conflictCols[i] = c
		assignments[i] = clause.Assignment{Column: c, Value: c}
	}
	return clause.OnConflict{
		Columns:   conflictCols,
		DoUpdates: assignments,
	}
}

func init() {
	engine.RegisterPlugin(NewMySQLPlugin())
	engine.RegisterPlugin(NewMyMariaDBPlugin())
}

func NewMySQLPlugin() *engine.Plugin {
	mysqlPlugin := &MySQLPlugin{}
	mysqlPlugin.Type = engine.DatabaseType_MySQL
	mysqlPlugin.PluginFunctions = mysqlPlugin
	mysqlPlugin.GormPluginFunctions = mysqlPlugin
	return &mysqlPlugin.Plugin
}

func NewMyMariaDBPlugin() *engine.Plugin {
	mysqlPlugin := &MySQLPlugin{}
	mysqlPlugin.Type = engine.DatabaseType_MariaDB
	mysqlPlugin.PluginFunctions = mysqlPlugin
	mysqlPlugin.GormPluginFunctions = mysqlPlugin
	return &mysqlPlugin.Plugin
}
