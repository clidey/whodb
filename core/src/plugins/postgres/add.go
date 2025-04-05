package postgres

import (
	"fmt"
	"strings"
)

func (p *PostgresPlugin) GetCreateTableQuery(schema string, storageUnit string, columns []string) string {
	createTableQuery := "CREATE TABLE %s.%s (%s)"
	return fmt.Sprintf(createTableQuery, schema, storageUnit, strings.Join(columns, ", "))
}
