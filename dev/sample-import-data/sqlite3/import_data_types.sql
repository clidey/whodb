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

INSERT INTO data_types (
    col_integer, col_real, col_text, col_numeric,
    col_boolean, col_date, col_datetime
) VALUES
(2000000, 2.71828, 'text_value_2', 456.78, 0, '2025-02-01', '2025-02-01 10:00:00'),
(3000000, 1.41421, 'text_value_3', 789.01, 1, '2025-02-02', '2025-02-02 11:00:00');
