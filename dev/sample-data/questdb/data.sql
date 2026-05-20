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

-- QuestDB init script for E2E tests
-- QuestDB is a time-series database with Postgres wire protocol.
-- Key differences from standard Postgres:
--   - No schemas (no CREATE SCHEMA)
--   - No foreign keys, no primary keys, no constraints
--   - No UPDATE/DELETE on partitioned tables
--   - Tables can be partitioned by timestamp
--   - Limited data types (no SERIAL, no arrays, no JSON)
--   - No views, no triggers, no stored procedures

CREATE TABLE IF NOT EXISTS users (
    id INT,
    username STRING,
    email STRING,
    password STRING,
    created_at TIMESTAMP
) TIMESTAMP(created_at) PARTITION BY DAY;

INSERT INTO users VALUES
    (1, 'john_doe', 'john@example.com', 'securepassword1', '2024-01-01T12:00:00.000000Z'),
    (2, 'jane_smith', 'jane@example.com', 'securepassword2', '2024-01-02T12:00:00.000000Z'),
    (3, 'admin_user', 'admin@example.com', 'adminpass1', '2024-01-03T12:00:00.000000Z'),
    (4, 'bob_jones', 'bob@example.com', 'bobpass123', '2024-01-04T12:00:00.000000Z'),
    (5, 'alice_wong', 'alice@example.com', 'alicepass1', '2024-01-05T12:00:00.000000Z');

CREATE TABLE IF NOT EXISTS products (
    id INT,
    name STRING,
    description STRING,
    price DOUBLE,
    stock_quantity INT,
    created_at TIMESTAMP
) TIMESTAMP(created_at) PARTITION BY DAY;

INSERT INTO products VALUES
    (1, 'Laptop', 'High-performance laptop', 1200.00, 10, '2024-01-01T12:00:00.000000Z'),
    (2, 'Smartphone', 'Latest model smartphone', 800.00, 20, '2024-01-02T12:00:00.000000Z'),
    (3, 'Headphones', 'Noise-canceling headphones', 150.00, 50, '2024-01-03T12:00:00.000000Z'),
    (4, 'Keyboard', 'Mechanical keyboard', 95.00, 30, '2024-01-04T12:00:00.000000Z');

CREATE TABLE IF NOT EXISTS orders (
    id INT,
    user_id INT,
    total_amount DOUBLE,
    status STRING,
    order_date TIMESTAMP
) TIMESTAMP(order_date) PARTITION BY DAY;

INSERT INTO orders VALUES
    (1, 1, 2000.00, 'completed', '2024-01-10T12:00:00.000000Z'),
    (2, 2, 150.00, 'pending', '2024-01-11T12:00:00.000000Z'),
    (3, 3, 95.00, 'completed', '2024-01-12T12:00:00.000000Z');

CREATE TABLE IF NOT EXISTS order_items (
    id INT,
    order_id INT,
    product_id INT,
    quantity INT,
    price_at_purchase DOUBLE,
    created_at TIMESTAMP
) TIMESTAMP(created_at) PARTITION BY DAY;

INSERT INTO order_items VALUES
    (1, 1, 1, 1, 1200.00, '2024-01-10T12:00:01.000000Z'),
    (2, 1, 2, 1, 800.00, '2024-01-10T12:00:02.000000Z'),
    (3, 2, 3, 1, 150.00, '2024-01-11T12:00:01.000000Z'),
    (4, 3, 4, 1, 95.00, '2024-01-12T12:00:01.000000Z');

CREATE TABLE IF NOT EXISTS payments (
    id INT,
    order_id INT,
    amount DOUBLE,
    payment_method STRING,
    payment_date TIMESTAMP
) TIMESTAMP(payment_date) PARTITION BY DAY;

INSERT INTO payments VALUES
    (1, 1, 2000.00, 'credit_card', '2024-01-12T12:00:00.000000Z'),
    (2, 2, 150.00, 'paypal', '2024-01-13T12:00:00.000000Z'),
    (3, 3, 95.00, 'bank_transfer', '2024-01-14T12:00:00.000000Z');

CREATE TABLE IF NOT EXISTS addresses (
    id INT,
    user_id INT,
    street STRING,
    city STRING,
    state STRING,
    zip_code STRING,
    country STRING,
    created_at TIMESTAMP
) TIMESTAMP(created_at) PARTITION BY DAY;

INSERT INTO addresses VALUES
    (1, 1, '123 Main St', 'New York', 'NY', '10001', 'USA', '2024-01-01T12:00:00.000000Z'),
    (2, 2, '456 Oak Ave', 'Los Angeles', 'CA', '90001', 'USA', '2024-01-02T12:00:00.000000Z'),
    (3, 3, '789 Pine Rd', 'Chicago', 'IL', '60601', 'USA', '2024-01-03T12:00:00.000000Z');

CREATE TABLE IF NOT EXISTS data_types (
    id INT,
    col_boolean BOOLEAN,
    col_short SHORT,
    col_int INT,
    col_long LONG,
    col_float FLOAT,
    col_double DOUBLE,
    col_string STRING,
    col_symbol SYMBOL,
    col_uuid UUID,
    col_date DATE,
    col_timestamp TIMESTAMP
) TIMESTAMP(col_timestamp) PARTITION BY DAY;

INSERT INTO data_types VALUES
    (1, true, 100, 1000, 100000, 1.5, 2.5, 'hello', 'sym1', '550e8400-e29b-41d4-a716-446655440000', '2024-01-01', '2024-01-01T12:00:00.000000Z'),
    (2, false, 200, 2000, 200000, 3.14, 6.28, 'world', 'sym2', '550e8400-e29b-41d4-a716-446655440001', '2024-01-02', '2024-01-02T12:00:00.000000Z');
