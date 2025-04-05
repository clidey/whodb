package mysql

import (
	"fmt"
	"strings"
)

func (p *MySQLPlugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return value, nil
}

func (p *MySQLPlugin) EscapeSpecificIdentifier(identifier string) string {
	identifier = strings.Replace(identifier, "`", "``", -1)
	return fmt.Sprintf("`%s`", identifier)
}

func (p *MySQLPlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT k.column_name
		FROM information_schema.table_constraints t
		JOIN information_schema.key_column_usage k
		USING (constraint_name, table_schema, table_name)
		WHERE t.constraint_type = 'PRIMARY KEY'
		AND t.table_schema = ?
		AND t.table_name = ?;
	`
}

func (p *MySQLPlugin) GetColTypeQuery() string {
	return `
		SELECT column_name, data_type
		FROM information_schema.columns
		WHERE table_schema = ? AND table_name = ?;
	`
}
