// Copyright 2025 Clidey, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
