package clickhouse

import (
	"fmt"
	"strings"
)

func (p *ClickHousePlugin) ConvertStringValueDuringMap(value, columnType string) (interface{}, error) {
	return p.ConvertStringValue(value, columnType)
}

func (p *ClickHousePlugin) EscapeSpecificIdentifier(identifier string) string {
	identifier = strings.Replace(identifier, "`", "``", -1)
	return fmt.Sprintf("`%s`", identifier)
}

func (p *ClickHousePlugin) GetPrimaryKeyColQuery() string {
	return `
		SELECT name
		FROM system.columns
		WHERE database = ? AND table = ? AND is_in_primary_key = 1
	`
}

func (p *ClickHousePlugin) GetColTypeQuery() string {
	return `
		SELECT 
			name,
			type
		FROM system.columns
		WHERE database = ? AND table = ?`
}
