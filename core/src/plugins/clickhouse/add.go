package clickhouse

import (
	"fmt"
	"strings"
)

func (p *ClickHousePlugin) GetCreateTableQuery(schema string, storageUnit string, columns []string) string {
	//todo: we need to figure out how to handle this engine + orderby more dynamically
	createTableQuery := `
		CREATE TABLE %s.%s 
		(%s) 
    	ENGINE = MergeTree
    	ORDER BY (%s)` // todo: shitty way of setting the order by for now
	params := strings.Join(columns, ", ")

	// use the first column as the sorting key FOR NOW
	orderBy := strings.Split(columns[0], " ")[0]
	orderBy = strings.Trim(orderBy, "`")

	return fmt.Sprintf(createTableQuery, schema, storageUnit, params, orderBy)
}
