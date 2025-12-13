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

-- Create Database
USE test_db_842;

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CHECK (CHAR_LENGTH(password) >= 8) -- Enforce password length
);

-- Products Table
CREATE TABLE IF NOT EXISTS products (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    price DECIMAL(10,2) NOT NULL CHECK (price >= 0), -- Ensure price is not negative
    stock_quantity INT NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Orders Table
CREATE TABLE IF NOT EXISTS orders (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0, -- Will be updated via trigger
    status ENUM('pending', 'completed', 'canceled') DEFAULT 'pending',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

-- Order Items Table
CREATE TABLE IF NOT EXISTS order_items (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    price_at_purchase DECIMAL(10,2) NOT NULL CHECK (price_at_purchase >= 0),
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES products(id) ON DELETE CASCADE
 );

-- Payments Table
CREATE TABLE IF NOT EXISTS payments (
    id INT AUTO_INCREMENT PRIMARY KEY,
    order_id INT NOT NULL,
    payment_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    amount DECIMAL(10,2) NOT NULL CHECK (amount >= 0),
    payment_method ENUM('credit_card', 'paypal', 'bank_transfer'),
    FOREIGN KEY (order_id) REFERENCES orders(id) ON DELETE CASCADE
);

-- Indexes for faster queries
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_orders_user_id ON orders(user_id);
CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_payments_order_id ON payments(order_id);

-- View for Order Summary
CREATE VIEW order_summary AS
SELECT
    o.id AS order_id,
    u.username,
    o.order_date,
    o.status,
    o.total_amount
FROM orders o
JOIN users u ON o.user_id = u.id;

-- Function to Update Order Total Automatically
DELIMITER //
CREATE TRIGGER trg_insert_order_total
    AFTER INSERT ON order_items
    FOR EACH ROW
BEGIN
    UPDATE orders
    SET total_amount = (
        SELECT COALESCE(SUM(price_at_purchase * quantity), 0)
        FROM order_items
        WHERE order_id = NEW.order_id
    )
    WHERE id = NEW.order_id;
END;
//
DELIMITER ;

DELIMITER //
CREATE TRIGGER trg_update_order_total
    AFTER UPDATE ON order_items
    FOR EACH ROW
BEGIN
    UPDATE orders
    SET total_amount = (
        SELECT COALESCE(SUM(price_at_purchase * quantity), 0)
        FROM order_items
        WHERE order_id = NEW.order_id
    )
    WHERE id = NEW.order_id;
END;
//
DELIMITER ;

DELIMITER //
CREATE TRIGGER trg_delete_order_total
    AFTER DELETE ON order_items
    FOR EACH ROW
BEGIN
    UPDATE orders
    SET total_amount = (
        SELECT COALESCE(SUM(price_at_purchase * quantity), 0)
        FROM order_items
        WHERE order_id = OLD.order_id
    )
    WHERE id = OLD.order_id;
END;
//
DELIMITER ;

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

-- Sample Data for Users
INSERT INTO users (username, email, password) VALUES
('john_doe', 'john@example.com', 'securepassword1'),
('jane_smith', 'jane@example.com', 'securepassword2'),
('admin_user', 'admin@example.com', 'adminpass');

-- Sample Data for Products
INSERT INTO products (name, description, price, stock_quantity) VALUES
('Laptop', 'High-performance laptop', 1200.00, 10),
('Smartphone', 'Latest model smartphone', 800.00, 20),
('Headphones', 'Noise-canceling headphones', 150.00, 50),
('Monitor', '4K UHD Monitor', 400.00, 15);

-- Sample Orders
INSERT INTO orders (user_id, total_amount, status) VALUES
(1, 0, 'completed'),
(2, 0, 'pending');

-- Sample Order Items (Trigger will update the total_amount)
INSERT INTO order_items (order_id, product_id, quantity, price_at_purchase) VALUES
(1, 1, 1, 1200.00), -- Laptop
(1, 2, 1, 800.00),  -- Smartphone
(2, 3, 1, 150.00);  -- Headphones

-- Sample Payments
INSERT INTO payments (order_id, amount, payment_method)
VALUES
    (1, 2000.00, 'credit_card'),
    (2, 150.00, 'paypal');

-- Test Casting Table for type casting validation
CREATE TABLE IF NOT EXISTS test_casting
(
    id
    INT
    AUTO_INCREMENT
    PRIMARY
    KEY,
    bigint_col
    BIGINT
    NOT
    NULL,
    integer_col
    INT
    NOT
    NULL,
    smallint_col
    SMALLINT
    NOT
    NULL,
    numeric_col
    DECIMAL
(
    10,
    2
),
    description VARCHAR
(
    100
)
    );

-- Insert sample data for test_casting
INSERT INTO test_casting (bigint_col, integer_col, smallint_col, numeric_col, description)
VALUES (9223372036854775807, 2147483647, 32767, 99999999.99, 'Maximum values'),
       (1000000, 1000, 100, 1234.56, 'Regular values'),
       (-9223372036854775808, -2147483648, -32768, -99999999.99, 'Minimum values');

-- Data Types Table for exhaustive type testing
CREATE TABLE IF NOT EXISTS data_types (
    id INT AUTO_INCREMENT PRIMARY KEY,
    -- Numeric types
    col_tinyint TINYINT,
    col_smallint SMALLINT,
    col_mediumint MEDIUMINT,
    col_int INT,
    col_bigint BIGINT,
    col_float FLOAT,
    col_double DOUBLE,
    col_decimal DECIMAL(10,2),
    -- Date/Time types
    col_date DATE,
    col_datetime DATETIME,
    col_timestamp TIMESTAMP NULL,
    col_time TIME,
    col_year YEAR,
    -- String types
    col_char CHAR(10),
    col_varchar VARCHAR(255),
    col_tinytext TINYTEXT,
    col_text TEXT,
    col_mediumtext MEDIUMTEXT,
    col_longtext LONGTEXT,
    -- Special types
    col_json JSON,
    col_boolean BOOLEAN
);

-- Insert seed data for data_types
INSERT INTO data_types (
    col_tinyint, col_smallint, col_mediumint, col_int, col_bigint,
    col_float, col_double, col_decimal, col_date, col_datetime,
    col_timestamp, col_time, col_year, col_char, col_varchar,
    col_tinytext, col_text, col_mediumtext, col_longtext,
    col_json, col_boolean
) VALUES (
    50, 1000, 100000, 1000000, 10000000000,
    1.5, 2.5, 123.45, '2025-01-01', '2025-01-01 12:00:00',
    '2025-01-01 12:00:00', '12:00:00', 2025, 'test', 'varchar_val',
    'tiny text', 'text value', 'medium text value', 'long text value',
    '{"key":"value"}', 1
);
