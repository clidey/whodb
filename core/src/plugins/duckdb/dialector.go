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

package duckdb

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/duckdb/duckdb-go/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/callbacks"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/migrator"
	"gorm.io/gorm/schema"
)

const DriverName = "duckdb"

// Dialector implements gorm.Dialector for DuckDB.
type Dialector struct {
	DSN  string
	Conn gorm.ConnPool
}

// Open creates a new DuckDB dialector with the given DSN.
// An empty DSN creates an in-memory database.
func Open(dsn string) gorm.Dialector {
	return &Dialector{DSN: dsn}
}

func (d Dialector) Name() string {
	return "duckdb"
}

func (d Dialector) Initialize(db *gorm.DB) error {
	if d.Conn != nil {
		db.ConnPool = d.Conn
	} else {
		conn, err := sql.Open(DriverName, d.DSN)
		if err != nil {
			return err
		}
		db.ConnPool = conn
	}

	callbacks.RegisterDefaultCallbacks(db, &callbacks.Config{
		CreateClauses: []string{"INSERT", "VALUES", "ON CONFLICT", "RETURNING"},
		UpdateClauses: []string{"UPDATE", "SET", "FROM", "WHERE", "RETURNING"},
		DeleteClauses: []string{"DELETE", "FROM", "WHERE", "RETURNING"},
	})

	for k, v := range d.ClauseBuilders() {
		if _, ok := db.ClauseBuilders[k]; !ok {
			db.ClauseBuilders[k] = v
		}
	}

	return nil
}

func (d Dialector) ClauseBuilders() map[string]clause.ClauseBuilder {
	return map[string]clause.ClauseBuilder{
		"INSERT": func(c clause.Clause, builder clause.Builder) {
			if insert, ok := c.Expression.(clause.Insert); ok {
				if stmt, ok := builder.(*gorm.Statement); ok {
					stmt.WriteString("INSERT ")
					if insert.Modifier != "" {
						stmt.WriteString(insert.Modifier)
						stmt.WriteByte(' ')
					}
					stmt.WriteString("INTO ")
					if insert.Table.Name == "" {
						stmt.WriteQuoted(stmt.Table)
					} else {
						stmt.WriteQuoted(insert.Table)
					}
					return
				}
			}
			c.Build(builder)
		},
		"LIMIT": func(c clause.Clause, builder clause.Builder) {
			if limit, ok := c.Expression.(clause.Limit); ok {
				var lmt = -1
				if limit.Limit != nil && *limit.Limit >= 0 {
					lmt = *limit.Limit
				}
				if lmt >= 0 || limit.Offset > 0 {
					builder.WriteString("LIMIT ")
					builder.WriteString(strconv.Itoa(lmt))
				}
				if limit.Offset > 0 {
					builder.WriteString(" OFFSET ")
					builder.WriteString(strconv.Itoa(limit.Offset))
				}
			}
		},
		"FOR": func(c clause.Clause, builder clause.Builder) {
			// DuckDB does not support row-level locking.
			if _, ok := c.Expression.(clause.Locking); ok {
				return
			}
			c.Build(builder)
		},
	}
}

func (d Dialector) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (d Dialector) Migrator(db *gorm.DB) gorm.Migrator {
	return DuckDBMigrator{migrator.Migrator{Config: migrator.Config{
		DB:                          db,
		Dialector:                   d,
		CreateIndexAfterCreateTable: true,
	}}}
}

func (d Dialector) BindVarTo(writer clause.Writer, stmt *gorm.Statement, v any) {
	writer.WriteByte('?')
}

// QuoteTo writes a double-quoted identifier (PostgreSQL-style).
func (d Dialector) QuoteTo(writer clause.Writer, str string) {
	var (
		underQuoted, selfQuoted bool
		continuousQuote         int8
		shiftDelimiter          int8
	)

	for _, v := range []byte(str) {
		switch v {
		case '"':
			continuousQuote++
			if continuousQuote == 2 {
				writer.WriteString(`""`)
				continuousQuote = 0
			}
		case '.':
			if continuousQuote > 0 || !selfQuoted {
				shiftDelimiter = 0
				underQuoted = false
				continuousQuote = 0
				writer.WriteByte('"')
			}
			writer.WriteByte(v)
			continue
		default:
			if shiftDelimiter-continuousQuote <= 0 && !underQuoted {
				writer.WriteByte('"')
				underQuoted = true
				if selfQuoted = continuousQuote > 0; selfQuoted {
					continuousQuote -= 1
				}
			}

			for ; continuousQuote > 0; continuousQuote -= 1 {
				writer.WriteString(`""`)
			}

			writer.WriteByte(v)
		}
		shiftDelimiter++
	}

	if continuousQuote > 0 && !selfQuoted {
		writer.WriteString(`""`)
	}
	writer.WriteByte('"')
}

func (d Dialector) Explain(sql string, vars ...any) string {
	return logger.ExplainSQL(sql, nil, `"`, vars...)
}

func (d Dialector) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "BOOLEAN"
	case schema.Int, schema.Uint:
		if field.AutoIncrement {
			return "INTEGER"
		}
		switch {
		case field.Size <= 8:
			return "TINYINT"
		case field.Size <= 16:
			return "SMALLINT"
		case field.Size <= 32:
			return "INTEGER"
		default:
			return "BIGINT"
		}
	case schema.Float:
		if field.Precision > 0 {
			return fmt.Sprintf("DECIMAL(%d,%d)", field.Precision, field.Scale)
		}
		if field.Size <= 32 {
			return "FLOAT"
		}
		return "DOUBLE"
	case schema.String:
		if field.Size > 0 {
			return fmt.Sprintf("VARCHAR(%d)", field.Size)
		}
		return "VARCHAR"
	case schema.Time:
		return "TIMESTAMP"
	case schema.Bytes:
		return "BLOB"
	}

	return string(field.DataType)
}

func (d Dialector) SavePoint(tx *gorm.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return nil
}

func (d Dialector) RollbackTo(tx *gorm.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return nil
}
