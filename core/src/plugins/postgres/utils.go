package postgres

import (
	"fmt"
	"strings"
)

func (p *PostgresPlugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return value, nil
}

func (p *PostgresPlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT a.attname
		FROM pg_index i
		JOIN pg_attribute a ON a.attrelid = i.indrelid AND a.attnum = ANY(i.indkey)
		JOIN pg_class c ON c.oid = i.indrelid
		JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = ? AND c.relname = ? AND i.indisprimary;
	`
}

func (p *PostgresPlugin) GetColTypeQuery() string {
	return `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?;
	`
}

func (p *PostgresPlugin) EscapeSpecificIdentifier(identifier string) string {
	identifier = strings.Replace(identifier, "\"", "\"\"", -1)
	return fmt.Sprintf("\"%s\"", identifier)
}
