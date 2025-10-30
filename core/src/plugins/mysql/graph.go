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

package mysql

import "gorm.io/gorm"

const graphQuery = `
SELECT DISTINCT
    rc.REFERENCED_TABLE_NAME AS Table1,
    rc.TABLE_NAME AS Table2,
    'OneToMany' AS Relation,
    kcu.COLUMN_NAME AS SourceColumn,
    kcu.REFERENCED_COLUMN_NAME AS TargetColumn
FROM
    INFORMATION_SCHEMA.REFERENTIAL_CONSTRAINTS rc
    JOIN INFORMATION_SCHEMA.KEY_COLUMN_USAGE kcu
        ON rc.CONSTRAINT_NAME = kcu.CONSTRAINT_NAME
        AND rc.CONSTRAINT_SCHEMA = kcu.CONSTRAINT_SCHEMA
WHERE
    rc.CONSTRAINT_SCHEMA = ?
`

func (p *MySQLPlugin) GetGraphQueryDB(db *gorm.DB, schema string) *gorm.DB {
	return db.Raw(graphQuery, schema)
}
