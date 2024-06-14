package postgres

import (
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/clidey/whodb/core/src/engine"
	"github.com/clidey/whodb/core/src/plugins/common"
)

type PostgresPlugin struct{}

func (p *PostgresPlugin) GetStorageUnits(config *engine.PluginConfig) ([]engine.StorageUnit, error) {
	db, err := DB(config)
	if err != nil {
		return nil, err
	}
	storageUnits := []engine.StorageUnit{}
	rows, err := db.Raw(`
		SELECT
			t.table_name,
			t.table_type,
			t.table_schema,
			pg_size_pretty(pg_total_relation_size('"' || t.table_schema || '"."' || t.table_name || '"')) AS total_size,
			pg_size_pretty(pg_relation_size('"' || t.table_schema || '"."' || t.table_name || '"')) AS data_size,
			COALESCE((SELECT reltuples::bigint FROM pg_class WHERE oid = ('"' || t.table_schema || '"."' || t.table_name || '"')::regclass), 0) AS row_count,
			c.relowner::regrole AS table_owner,
			(SELECT description FROM pg_description WHERE objoid = c.oid AND objsubid = 0) AS description,
			COALESCE((SELECT spcname FROM pg_tablespace WHERE oid = c.reltablespace), 'pg_default') AS tablespace,
			(SELECT count(*) FROM pg_index WHERE indrelid = c.oid) AS num_indexes,
			(SELECT count(*) FROM information_schema.table_constraints WHERE table_name = t.table_name AND constraint_type = 'FOREIGN KEY') AS num_foreign_keys,
			(SELECT count(*) FROM information_schema.check_constraints WHERE table_name = t.table_name) AS num_check_constraints
		FROM
			information_schema.tables t
		JOIN
			pg_class c ON t.table_name = c.relname AND t.table_schema = c.relnamespace::regnamespace::text
		WHERE
			t.table_schema = 'public'
	`).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName, tableType, tableSchema, totalSize, dataSize, tableOwner, tablespace string
		var description sql.NullString
		var rowCount, numIndexes, numForeignKeys, numCheckConstraints int64
		if err := rows.Scan(&tableName, &tableType, &tableSchema, &totalSize, &dataSize, &rowCount, &tableOwner, &description, &tablespace, &numIndexes, &numForeignKeys, &numCheckConstraints); err != nil {
			log.Fatal(err)
		}

		desc := ""
		if description.Valid {
			desc = description.String
		}

		attributes := []engine.Record{
			{Key: "Table Type", Value: tableType},
			{Key: "Table Schema", Value: tableSchema},
			{Key: "Total Size", Value: totalSize},
			{Key: "Data Size", Value: dataSize},
			{Key: "Row Count", Value: fmt.Sprintf("%d", rowCount)},
			{Key: "Table Owner", Value: tableOwner},
			{Key: "Description", Value: desc},
			{Key: "Tablespace", Value: tablespace},
			{Key: "Number of Indexes", Value: fmt.Sprintf("%d", numIndexes)},
			{Key: "Number of Foreign Keys", Value: fmt.Sprintf("%d", numForeignKeys)},
			{Key: "Number of Check Constraints", Value: fmt.Sprintf("%d", numCheckConstraints)},
		}

		storageUnits = append(storageUnits, engine.StorageUnit{
			Name:       tableName,
			Attributes: attributes,
		})
	}
	return storageUnits, nil
}

func (p *PostgresPlugin) GetRows(config *engine.PluginConfig, storageUnit string) (*engine.GetRowsResult, error) {
	if !common.IsValidSQLTableName(storageUnit) {
		return nil, errors.New("invalid table name")
	}

	db, err := DB(config)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT * FROM %s LIMIT ? OFFSET ?", storageUnit)
	rows, err := db.Raw(query, 10, 0).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, err
	}

	result := &engine.GetRowsResult{}
	for _, col := range columns {
		for _, colType := range columnTypes {
			if col == colType.Name() {
				result.Columns = append(result.Columns, engine.Column{Name: col, Type: colType.DatabaseTypeName()})
				break
			}
		}
	}

	for rows.Next() {
		columnPointers := make([]interface{}, len(columns))
		row := make([]string, len(columns))
		for i := range columns {
			columnPointers[i] = new(sql.NullString)
		}

		if err := rows.Scan(columnPointers...); err != nil {
			return nil, err
		}

		for i, colPtr := range columnPointers {
			val := colPtr.(*sql.NullString)
			if val.Valid {
				row[i] = val.String
			} else {
				row[i] = ""
			}
		}

		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

func (p *PostgresPlugin) GetColumns(config *engine.PluginConfig, storageUnit string, row string) (map[string][]string, error) {
	return nil, nil
}

func (p *PostgresPlugin) GetConstraints(config *engine.PluginConfig) map[string]string {
	return nil
}

func (p *PostgresPlugin) RawExecute(config *engine.PluginConfig, sql string) error {
	return nil
}

func NewPostgresPlugin() *engine.Plugin {
	return &engine.Plugin{
		Type:            engine.DatabaseType_Postgres,
		PluginFunctions: &PostgresPlugin{},
	}
}
