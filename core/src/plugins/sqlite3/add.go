package sqlite3

import (
	"fmt"
	"strings"
)

func (p *Sqlite3Plugin) GetCreateTableQuery(schema string, storageUnit string, columns []string) string {
	createTableQuery := "CREATE TABLE %s (%s)"
	return fmt.Sprintf(createTableQuery, storageUnit, strings.Join(columns, ", "))
}
