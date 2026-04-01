-- Copyright 2026 Clidey, Inc.
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

-- Sample import data

INSERT INTO users (username, email, password, created_at) VALUES
('sql_user_1', 'sql.user1@example.com', 'sqlpass001', '2025-01-20 09:00:00'),
('sql_user_2', 'sql.user2@example.com', 'sqlpass002', '2025-01-20 09:10:00'),
('sql_user_3', 'sql.user3@example.com', 'sqlpass003', '2025-01-20 09:20:00');

INSERT INTO products (name, description, price, stock_quantity, created_at) VALUES
('SQL Laptop', 'Developer laptop', 1400.00, 8, '2025-01-20 10:00:00'),
('SQL Monitor', 'Ultra-wide monitor', 499.99, 14, '2025-01-20 10:05:00'),
('SQL Headset', 'Conference headset', 199.00, 30, '2025-01-20 10:10:00');
