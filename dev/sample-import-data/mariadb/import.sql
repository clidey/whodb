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

-- Sample import data

INSERT INTO test_db.users (id, username, email, password, created_at) VALUES
(1001, 'sql_user_1', 'sql.user1@example.com', 'sqlpass001', '2025-01-20 09:00:00'),
(1002, 'sql_user_2', 'sql.user2@example.com', 'sqlpass002', '2025-01-20 09:10:00'),
(1003, 'sql_user_3', 'sql.user3@example.com', 'sqlpass003', '2025-01-20 09:20:00');

INSERT INTO test_db.products (id, name, description, price, stock_quantity, created_at) VALUES
(1001, 'SQL Laptop', 'Developer laptop', 1400.00, 8, '2025-01-20 10:00:00'),
(1002, 'SQL Monitor', 'Ultra-wide monitor', 499.99, 14, '2025-01-20 10:05:00'),
(1003, 'SQL Headset', 'Conference headset', 199.00, 30, '2025-01-20 10:10:00');

INSERT INTO test_db.orders (id, user_id, order_date, total_amount, status) VALUES
(2001, 1001, '2025-01-21 14:00:00', 2000.00, 'completed'),
(2002, 1002, '2025-01-21 15:30:00', 300.00, 'pending');

INSERT INTO test_db.order_items (id, order_id, product_id, quantity, price_at_purchase) VALUES
(3001, 2001, 1001, 1, 1400.00),
(3002, 2001, 1002, 1, 600.00),
(3003, 2002, 1003, 2, 150.00);

INSERT INTO test_db.payments (id, order_id, payment_date, amount, payment_method) VALUES
(4001, 2001, '2025-01-21 14:05:00', 2000.00, 'credit_card'),
(4002, 2002, '2025-01-21 15:40:00', 300.00, 'paypal');
