package sqlite3

import (
	"strings"
)

func (p *Sqlite3Plugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return value, nil
}

func (p *Sqlite3Plugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT p.name AS pk_column
		FROM sqlite_master m,
			 pragma_table_info(m.name) p
		WHERE m.type = 'table'
		  AND m.name NOT LIKE ?
		  AND p.pk > 0
		ORDER BY m.name, p.pk;`
}

func (p *Sqlite3Plugin) GetColTypeQuery() string {
	return `
		SELECT p.name AS column_name,
			   p.type AS data_type
		FROM sqlite_master m,
			 pragma_table_info(m.name) p
		WHERE m.type = 'table'
		  AND m.name NOT LIKE 'sqlite_%';
	`
}

func (p *Sqlite3Plugin) EscapeSpecificIdentifier(identifier string) string {
	identifier = strings.Replace(identifier, "\"", "\"\"", -1)
	return identifier
}
