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

-- WhoDB Sample Database
-- A simple e-commerce schema to demonstrate WhoDB features

-- Enable foreign key support
PRAGMA
foreign_keys = ON;

-- Users Table
CREATE TABLE IF NOT EXISTS users
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    username
    TEXT
    UNIQUE
    NOT
    NULL,
    email
    TEXT
    UNIQUE
    NOT
    NULL,
    password
    TEXT
    NOT
    NULL,
    created_at
    DATETIME
    DEFAULT
    CURRENT_TIMESTAMP,
    CHECK (
    LENGTH
(
    password
) >= 8)
    );

-- Products Table
CREATE TABLE IF NOT EXISTS products
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    name
    TEXT
    NOT
    NULL
    UNIQUE,
    description
    TEXT,
    price
    REAL
    NOT
    NULL
    CHECK
(
    price
    >=
    0
),
    stock_quantity INTEGER NOT NULL DEFAULT 0 CHECK
(
    stock_quantity
    >=
    0
),
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
    );

-- Orders Table
CREATE TABLE IF NOT EXISTS orders
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    user_id
    INTEGER
    NOT
    NULL,
    order_date
    DATETIME
    DEFAULT
    CURRENT_TIMESTAMP,
    total_amount
    REAL
    NOT
    NULL
    DEFAULT
    0,
    status
    TEXT
    DEFAULT
    'pending'
    CHECK (
    status
    IN
(
    'pending',
    'completed',
    'canceled'
)),
    FOREIGN KEY
(
    user_id
) REFERENCES users
(
    id
) ON DELETE CASCADE
    );

-- Order Items Table
CREATE TABLE IF NOT EXISTS order_items
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    order_id
    INTEGER
    NOT
    NULL,
    product_id
    INTEGER
    NOT
    NULL,
    quantity
    INTEGER
    NOT
    NULL
    CHECK
(
    quantity >
    0
),
    price_at_purchase REAL NOT NULL CHECK
(
    price_at_purchase
    >=
    0
),
    FOREIGN KEY
(
    order_id
) REFERENCES orders
(
    id
) ON DELETE CASCADE,
    FOREIGN KEY
(
    product_id
) REFERENCES products
(
    id
)
  ON DELETE CASCADE
    );

-- Payments Table
CREATE TABLE IF NOT EXISTS payments
(
    id
    INTEGER
    PRIMARY
    KEY
    AUTOINCREMENT,
    order_id
    INTEGER
    NOT
    NULL,
    payment_date
    DATETIME
    DEFAULT
    CURRENT_TIMESTAMP,
    amount
    REAL
    NOT
    NULL
    CHECK
(
    amount
    >=
    0
),
    payment_method TEXT CHECK
(
    payment_method
    IN
(
    'credit_card',
    'paypal',
    'bank_transfer'
)),
    FOREIGN KEY
(
    order_id
) REFERENCES orders
(
    id
) ON DELETE CASCADE
    );

-- Indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_orders_user_id ON orders(user_id);
CREATE INDEX IF NOT EXISTS idx_order_items_order_id ON order_items(order_id);
CREATE INDEX IF NOT EXISTS idx_payments_order_id ON payments(order_id);

-- View for Order Summary
CREATE VIEW IF NOT EXISTS order_summary AS
SELECT o.id AS order_id,
       u.username,
       o.order_date,
       o.status,
       o.total_amount
FROM orders o
         JOIN users u ON o.user_id = u.id;

-- Triggers to update total_amount in orders
CREATE TRIGGER IF NOT EXISTS trg_insert_order_total
AFTER INSERT ON order_items
BEGIN
UPDATE orders
SET total_amount = (SELECT COALESCE(SUM(price_at_purchase * quantity), 0)
                    FROM order_items
                    WHERE order_id = NEW.order_id)
WHERE id = NEW.order_id;
END;

CREATE TRIGGER IF NOT EXISTS trg_update_order_total
AFTER
UPDATE ON order_items
BEGIN
UPDATE orders
SET total_amount = (SELECT COALESCE(SUM(price_at_purchase * quantity), 0)
                    FROM order_items
                    WHERE order_id = NEW.order_id)
WHERE id = NEW.order_id;
END;

CREATE TRIGGER IF NOT EXISTS trg_delete_order_total
AFTER
DELETE
ON order_items
BEGIN
UPDATE orders
SET total_amount = (SELECT COALESCE(SUM(price_at_purchase * quantity), 0)
                    FROM order_items
                    WHERE order_id = OLD.order_id)
WHERE id = OLD.order_id;
END;

-- Sample Data for Users
INSERT INTO users (username, email, password)
VALUES ('john_doe', 'john@example.com', 'securepassword1'),
       ('jane_smith', 'jane@example.com', 'securepassword2'),
       ('admin_user', 'admin@example.com', 'adminpass1');

-- Sample Data for Products
INSERT INTO products (name, description, price, stock_quantity)
VALUES ('Laptop', 'High-performance laptop', 1200.00, 10),
       ('Smartphone', 'Latest model smartphone', 800.00, 20),
       ('Headphones', 'Noise-canceling headphones', 150.00, 50),
       ('Monitor', '4K UHD Monitor', 400.00, 15);

-- Sample Orders
INSERT INTO orders (user_id, total_amount, status)
VALUES (1, 0, 'completed'),
       (2, 0, 'pending');

-- Sample Order Items (triggers will update total_amount)
INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase)
VALUES (1, 1, 1, 1200.00),
       (1, 2, 1, 800.00),
       (2, 3, 1, 150.00);

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

-- Sample Payments
INSERT INTO payments (order_id, amount, payment_method)
VALUES (1, 2000.00, 'credit_card'),
       (2, 150.00, 'paypal');
