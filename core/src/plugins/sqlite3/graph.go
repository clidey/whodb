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

package sqlite3

import (
	"gorm.io/gorm"
)

const graphQuery = `
SELECT DISTINCT
    f."table" AS table1,
    m.name AS table2,
    'OneToMany' AS relation,
    f."from" AS source_column,
    f."to" AS target_column
FROM
    sqlite_master m,
    pragma_foreign_key_list(m.name) f
WHERE
    m.type = 'table'
`

func (p *Sqlite3Plugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	return db.Raw(graphQuery)
}
