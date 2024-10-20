package clickhouse

import (
	"context"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/clidey/whodb/core/src/engine"
)

func (p *ClickHousePlugin) GetRows(config *engine.PluginConfig, schema string, storageUnit string, where string, pageSize int, pageOffset int) (*engine.GetRowsResult, error) {
	query := fmt.Sprintf("SELECT * FROM %s.%s", schema, storageUnit)
	if where != "" {
		query += " WHERE " + where
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", pageSize, pageOffset)

	return p.executeQuery(config, query)
}

func (p *ClickHousePlugin) RawExecute(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	return p.executeQuery(config, query)
}

func (p *ClickHousePlugin) executeQuery(config *engine.PluginConfig, query string) (*engine.GetRowsResult, error) {
	conn, err := DB(config)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columnTypes := rows.ColumnTypes()
	result := &engine.GetRowsResult{
		Columns: make([]engine.Column, len(columnTypes)),
		Rows:    [][]string{},
	}

	for i, ct := range columnTypes {
		result.Columns[i] = engine.Column{Name: ct.Name(), Type: ct.DatabaseTypeName()}
	}

	for rows.Next() {
		row, err := scanRow(rows)
		if err != nil {
			return nil, err
		}
		result.Rows = append(result.Rows, row)
	}

	return result, nil
}

func scanRow(rows driver.Rows) ([]string, error) {
	columnTypes := rows.ColumnTypes()
	values := make([]interface{}, len(columnTypes))
	for i := range values {
		values[i] = new(interface{})
	}

	err := rows.Scan(values...)
	if err != nil {
		return nil, err
	}

	row := make([]string, len(columnTypes))
	for i, v := range values {
		row[i] = fmt.Sprintf("%v", *(v.(*interface{})))
	}

	return row, nil
}
