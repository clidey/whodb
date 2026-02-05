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

INSERT INTO test_schema.data_types (
    col_smallint, col_integer, col_bigint, col_decimal, col_numeric,
    col_real, col_double, col_money, col_char, col_varchar, col_text,
    col_bytea, col_timestamp, col_timestamptz, col_date, col_time, col_timetz,
    col_boolean, col_point, col_line, col_lseg, col_box, col_path, col_polygon,
    col_circle, col_cidr, col_inet, col_macaddr, col_uuid, col_xml, col_json, col_jsonb
) VALUES (
    100, 1000, 100000, 123.45, 678.90,
    1.5, 2.5, 99.99, 'test', 'varchar_val', 'text_value',
    E'\\x48454c4c4f', '2025-02-01 12:00:00', '2025-02-01 12:00:00+00', '2025-02-01', '12:00:00', '12:00:00+00',
    true, '(1,2)', '{1,2,3}', '[(0,0),(1,1)]', '((0,0),(1,1))', '((0,0),(1,1),(1,0))', '((0,0),(1,0),(1,1),(0,1))',
    '<(0,0),5>', '192.168.10.0/24', '192.168.10.1', '08:00:2b:01:02:03',
    '550e8400-e29b-41d4-a716-446655440001', '<root>import1</root>',
    '{"key":"value1"}', '{"key":"value1"}'
),
(
    200, 2000, 200000, 223.45, 778.90,
    2.5, 3.5, 199.99, 'test2', 'varchar_val2', 'text_value2',
    E'\\x574f524c44', '2025-02-02 13:30:00', '2025-02-02 13:30:00+00', '2025-02-02', '13:30:00', '13:30:00+00',
    false, '(3,4)', '{3,4,5}', '[(1,1),(2,2)]', '((1,1),(2,2))', '((1,1),(2,2),(2,1))', '((1,1),(2,1),(2,2),(1,2))',
    '<(1,1),7>', '10.0.0.0/24', '10.0.0.1', '08:00:2b:01:02:04',
    '550e8400-e29b-41d4-a716-446655440002', '<root>import2</root>',
    '{"key":"value2"}', '{"key":"value2"}'
);
