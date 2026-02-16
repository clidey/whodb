-- Copyright 2025 Clidey, Inc.
--
-- Licensed under the Apache License, Version 2.0 (the "License");
-- you may not use this file except in compliance with the License.
-- You may obtain a copy of the License at
--
--     http://www.apache.org/licenses/LICENSE-2.0
--
-- Unless required by applicable law or agreed to in writing, software
-- distributed under the License is distributed on an "AS IS" BASIS,
-- WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
-- See the License for the specific language governing permissions and
-- limitations under the License.

-- Sample import data (single table: data_types)

INSERT INTO test_db.data_types (
    col_tinyint, col_smallint, col_mediumint, col_int, col_bigint,
    col_float, col_double, col_decimal, col_date, col_datetime,
    col_timestamp, col_time, col_year, col_char, col_varchar,
    col_tinytext, col_text, col_mediumtext, col_longtext,
    col_json, col_boolean
) VALUES
(25, 500, 50000, 500000, 5000000000,
 0.75, 1.25, 45.67, '2025-02-01', '2025-02-01 10:00:00',
 '2025-02-01 10:00:00', '10:00:00', 2025, 'char_val', 'varchar_val_2',
 'tiny text 2', 'text value 2', 'medium text 2', 'long text 2',
 '{"key":"value2"}', 1),
(75, 1500, 150000, 1500000, 15000000000,
 2.75, 3.25, 89.01, '2025-02-02', '2025-02-02 11:00:00',
 '2025-02-02 11:00:00', '11:00:00', 2026, 'char_val2', 'varchar_val_3',
 'tiny text 3', 'text value 3', 'medium text 3', 'long text 3',
 '{"key":"value3"}', 0);
