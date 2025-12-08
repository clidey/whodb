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

-- Use the test_db database
CREATE DATABASE IF NOT EXISTS test_db;

-- Users Table
CREATE TABLE IF NOT EXISTS test_db.users (
    id UInt32,
    username String,
    email String,
    password String,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;

-- Products Table
CREATE TABLE IF NOT EXISTS test_db.products (
    id UInt32,
    name String,
    description String,
    price Decimal(10,2),
    stock_quantity UInt32 DEFAULT 0,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;

-- Orders Table
CREATE TABLE IF NOT EXISTS test_db.orders (
    id UInt32,
    user_id UInt32,
    order_date DateTime DEFAULT now(),
    total_amount Decimal(10,2) DEFAULT 0,
    status Enum('pending', 'completed', 'canceled') DEFAULT 'pending'
) ENGINE = MergeTree()
ORDER BY id;

-- Order Items Table
CREATE TABLE IF NOT EXISTS test_db.order_items (
    id UInt32,
    order_id UInt32,
    product_id UInt32,
    quantity UInt32 DEFAULT 1,
    price_at_purchase Decimal(10,2)
) ENGINE = MergeTree()
ORDER BY id;

-- Payments Table
CREATE TABLE IF NOT EXISTS test_db.payments (
    id UInt32,
    order_id UInt32,
    payment_date DateTime DEFAULT now(),
    amount Decimal(10,2),
    payment_method Enum('credit_card', 'paypal', 'bank_transfer')
) ENGINE = MergeTree()
ORDER BY id;

-- Test Casting Table (for type casting tests)
CREATE TABLE IF NOT EXISTS test_db.test_casting (
    id INT,
    bigint_col BIGINT,
    integer_col INT,
    smallint_col SMALLINT,
    numeric_col DOUBLE,
    description TEXT
) ENGINE = MergeTree()
ORDER BY id;

-- Materialized View for Order Summary
CREATE MATERIALIZED VIEW IF NOT EXISTS test_db.order_summary
ENGINE = MergeTree()
ORDER BY order_id
POPULATE AS
SELECT 
    o.id AS order_id,
    u.username AS username,
    o.order_date,
    o.status,
    o.total_amount
FROM test_db.orders o
INNER JOIN test_db.users u ON o.user_id = u.id;

-- Sample Data

-- Users
INSERT INTO test_db.users (id, username, email, password, created_at) VALUES 
(1, 'john_doe', 'john@example.com', 'securepassword1', now());

INSERT INTO test_db.users (id, username, email, password, created_at) VALUES 
(2, 'jane_smith', 'jane@example.com', 'securepassword2', now());

INSERT INTO test_db.users (id, username, email, password, created_at) VALUES 
(3, 'admin_user', 'admin@example.com', 'adminpass', now());

-- Products
INSERT INTO test_db.products (id, name, description, price, stock_quantity, created_at) VALUES 
(1, 'Laptop', 'High-performance laptop', 1200.00, 10, now());

INSERT INTO test_db.products (id, name, description, price, stock_quantity, created_at) VALUES 
(2, 'Smartphone', 'Latest model smartphone', 800.00, 20, now());

INSERT INTO test_db.products (id, name, description, price, stock_quantity, created_at) VALUES 
(3, 'Headphones', 'Noise-canceling headphones', 150.00, 50, now());

INSERT INTO test_db.products (id, name, description, price, stock_quantity, created_at) VALUES 
(4, 'Monitor', '4K UHD Monitor', 400.00, 15, now());

-- Orders
INSERT INTO test_db.orders (id, user_id, order_date, total_amount, status) VALUES 
(1, 1, now(), 2000.00, 'completed');

INSERT INTO test_db.orders (id, user_id, order_date, total_amount, status) VALUES 
(2, 2, now(), 150.00, 'pending');

-- Order Items
INSERT INTO test_db.order_items (id, order_id, product_id, quantity, price_at_purchase) VALUES 
(1, 1, 1, 1, 1200.00);

INSERT INTO test_db.order_items (id, order_id, product_id, quantity, price_at_purchase) VALUES 
(2, 1, 2, 1, 800.00);

INSERT INTO test_db.order_items (id, order_id, product_id, quantity, price_at_purchase) VALUES 
(3, 2, 3, 1, 150.00);

/*
 * Copyright 2025 Clidey, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

-- Payments
INSERT INTO test_db.payments (id, order_id, payment_date, amount, payment_method) VALUES 
(1, 1, now(), 2000.00, 'credit_card');

INSERT INTO test_db.payments (id, order_id, payment_date, amount, payment_method) VALUES
(2, 2, now(), 150.00, 'paypal');

-- Test Casting Data
INSERT INTO test_db.test_casting (id, bigint_col, integer_col, smallint_col, numeric_col, description) VALUES
(1, 9223372036854775807, 2147483647, 32767, 12345.67, 'Max values test');

INSERT INTO test_db.test_casting (id, bigint_col, integer_col, smallint_col, numeric_col, description) VALUES
(2, 1000000, 1000, 100, 99.99, 'Standard values');

INSERT INTO test_db.test_casting (id, bigint_col, integer_col, smallint_col, numeric_col, description) VALUES
(3, -1000000, -1000, -100, -99.99, 'Negative values');
