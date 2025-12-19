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

-- Create a schema (optional, default is 'public')
CREATE SCHEMA IF NOT EXISTS test_schema;

-- Users Table
CREATE TABLE IF NOT EXISTS test_schema.users (
    id SERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(100) NOT NULL CHECK (LENGTH(password) >= 8), -- Enforce password length
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Products Table
CREATE TABLE IF NOT EXISTS test_schema.products (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    price DECIMAL(10,2) NOT NULL CHECK (price >= 0), -- Ensure price is not negative
    stock_quantity INT NOT NULL DEFAULT 0 CHECK (stock_quantity >= 0),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Orders Table
CREATE TABLE IF NOT EXISTS test_schema.orders (
    id SERIAL PRIMARY KEY,
    user_id INT NOT NULL,
    order_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0, -- Will be updated via trigger
    status VARCHAR(20) CHECK (status IN ('pending', 'completed', 'canceled')) DEFAULT 'pending',
    FOREIGN KEY (user_id) REFERENCES test_schema.users(id) ON DELETE CASCADE
);

-- Order Items Table
CREATE TABLE IF NOT EXISTS test_schema.order_items (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL CHECK (quantity > 0),
    price_at_purchase DECIMAL(10,2) NOT NULL CHECK (price_at_purchase >= 0),
    FOREIGN KEY (order_id) REFERENCES test_schema.orders(id) ON DELETE CASCADE,
    FOREIGN KEY (product_id) REFERENCES test_schema.products(id) ON DELETE CASCADE
);

-- Payments Table
CREATE TABLE IF NOT EXISTS test_schema.payments (
    id SERIAL PRIMARY KEY,
    order_id INT NOT NULL,
    payment_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    amount DECIMAL(10,2) NOT NULL CHECK (amount >= 0),
    payment_method VARCHAR(20) CHECK (payment_method IN ('credit_card', 'paypal', 'bank_transfer')),
    FOREIGN KEY (order_id) REFERENCES test_schema.orders(id) ON DELETE CASCADE
);

-- Indexes for faster queries
CREATE INDEX idx_users_email ON test_schema.users(email);
CREATE INDEX idx_orders_user_id ON test_schema.orders(user_id);
CREATE INDEX idx_order_items_order_id ON test_schema.order_items(order_id);
CREATE INDEX idx_payments_order_id ON test_schema.payments(order_id);

-- View for Order Summary
CREATE VIEW test_schema.order_summary AS
SELECT 
    o.id AS order_id,
    u.username,
    o.order_date,
    o.status,
    o.total_amount
FROM test_schema.orders o
JOIN test_schema.users u ON o.user_id = u.id;

-- Function to Update Order Total Automatically
CREATE OR REPLACE FUNCTION test_schema.update_order_total() RETURNS TRIGGER AS $$
BEGIN
    UPDATE test_schema.orders
    SET total_amount = (
        SELECT COALESCE(SUM(price_at_purchase * quantity), 0)
        FROM test_schema.order_items
        WHERE order_id = NEW.order_id
    )
    WHERE id = NEW.order_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to Recalculate Order Total when an Order Item is Added/Updated
CREATE TRIGGER trg_update_order_total
AFTER INSERT OR UPDATE OR DELETE ON test_schema.order_items
FOR EACH ROW EXECUTE FUNCTION test_schema.update_order_total();

-- Sample Data for Users
INSERT INTO test_schema.users (username, email, password) VALUES 
('john_doe', 'john@example.com', 'securepassword1'),
('jane_smith', 'jane@example.com', 'securepassword2'),
('admin_user', 'admin@example.com', 'adminpass');

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

-- Sample Data for Products
INSERT INTO test_schema.products (name, description, price, stock_quantity) VALUES
('Laptop', 'High-performance laptop', 1200.00, 10),
('Smartphone', 'Latest model smartphone', 800.00, 20),
('Headphones', 'Noise-canceling headphones', 150.00, 50),
('Monitor', '4K UHD Monitor', 400.00, 15),
-- ('Laptop1', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone1', 'Latest model smartphone', 800.00, 20),
-- ('Headphones1', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor1', '4K UHD Monitor', 400.00, 15),
-- ('Laptop2', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone2', 'Latest model smartphone', 800.00, 20),
-- ('Headphones2', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor2', '4K UHD Monitor', 400.00, 15),
-- ('Laptop3', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone3', 'Latest model smartphone', 800.00, 20),
-- ('Headphones3', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor3', '4K UHD Monitor', 400.00, 15),
-- ('Laptop4', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone4', 'Latest model smartphone', 800.00, 20),
-- ('Headphones4', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor4', '4K UHD Monitor', 400.00, 15),
-- ('Laptop5', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone5', 'Latest model smartphone', 800.00, 20),
-- ('Headphones5', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor5', '4K UHD Monitor', 400.00, 15),
-- ('Laptop6', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone6', 'Latest model smartphone', 800.00, 20),
-- ('Headphones6', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor6', '4K UHD Monitor', 400.00, 15),
-- ('Laptop7', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone7', 'Latest model smartphone', 800.00, 20),
-- ('Headphones7', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor7', '4K UHD Monitor', 400.00, 15),
-- ('Laptop8', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone8', 'Latest model smartphone', 800.00, 20),
-- ('Headphones8', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor8', '4K UHD Monitor', 400.00, 15),
-- ('Laptop9', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone9', 'Latest model smartphone', 800.00, 20),
-- ('Headphones9', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor9', '4K UHD Monitor', 400.00, 15),
-- ('Laptop10', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone10', 'Latest model smartphone', 800.00, 20),
-- ('Headphones10', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor10', '4K UHD Monitor', 400.00, 15),
-- ('Laptop11', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone11', 'Latest model smartphone', 800.00, 20),
-- ('Headphones11', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor11', '4K UHD Monitor', 400.00, 15),
-- ('Laptop12', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone12', 'Latest model smartphone', 800.00, 20),
-- ('Headphones12', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor12', '4K UHD Monitor', 400.00, 15),
-- ('Laptop13', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone13', 'Latest model smartphone', 800.00, 20),
-- ('Headphones13', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor13', '4K UHD Monitor', 400.00, 15),
-- ('Laptop14', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone14', 'Latest model smartphone', 800.00, 20),
-- ('Headphones14', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor14', '4K UHD Monitor', 400.00, 15),
-- ('Laptop15', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone15', 'Latest model smartphone', 800.00, 20),
-- ('Headphones15', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor15', '4K UHD Monitor', 400.00, 15),
-- ('Laptop16', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone16', 'Latest model smartphone', 800.00, 20),
-- ('Headphones16', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor16', '4K UHD Monitor', 400.00, 15),
-- ('Laptop17', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone17', 'Latest model smartphone', 800.00, 20),
-- ('Headphones17', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor17', '4K UHD Monitor', 400.00, 15),
-- ('Laptop18', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone18', 'Latest model smartphone', 800.00, 20),
-- ('Headphones18', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor18', '4K UHD Monitor', 400.00, 15),
-- ('Laptop19', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone19', 'Latest model smartphone', 800.00, 20),
-- ('Headphones19', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor19', '4K UHD Monitor', 400.00, 15),
-- ('Laptop20', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone20', 'Latest model smartphone', 800.00, 20),
-- ('Headphones20', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor20', '4K UHD Monitor', 400.00, 15),
-- ('Laptop21', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone21', 'Latest model smartphone', 800.00, 20),
-- ('Headphones21', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor21', '4K UHD Monitor', 400.00, 15),
-- ('Laptop22', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone22', 'Latest model smartphone', 800.00, 20),
-- ('Headphones22', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor22', '4K UHD Monitor', 400.00, 15),
-- ('Laptop23', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone23', 'Latest model smartphone', 800.00, 20),
-- ('Headphones23', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor23', '4K UHD Monitor', 400.00, 15),
-- ('Laptop24', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone24', 'Latest model smartphone', 800.00, 20),
-- ('Headphones24', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor24', '4K UHD Monitor', 400.00, 15),
-- ('Laptop25', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone25', 'Latest model smartphone', 800.00, 20),
-- ('Headphones25', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor25', '4K UHD Monitor', 400.00, 15),
-- ('Laptop26', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone26', 'Latest model smartphone', 800.00, 20),
-- ('Headphones26', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor26', '4K UHD Monitor', 400.00, 15),
-- ('Laptop27', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone27', 'Latest model smartphone', 800.00, 20),
-- ('Headphones27', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor27', '4K UHD Monitor', 400.00, 15),
-- ('Laptop28', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone28', 'Latest model smartphone', 800.00, 20),
-- ('Headphones28', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor28', '4K UHD Monitor', 400.00, 15),
-- ('Laptop29', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone29', 'Latest model smartphone', 800.00, 20),
-- ('Headphones29', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor29', '4K UHD Monitor', 400.00, 15),
-- ('Laptop30', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone30', 'Latest model smartphone', 800.00, 20),
-- ('Headphones30', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor30', '4K UHD Monitor', 400.00, 15),
-- ('Laptop31', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone31', 'Latest model smartphone', 800.00, 20),
-- ('Headphones31', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor31', '4K UHD Monitor', 400.00, 15),
-- ('Laptop32', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone32', 'Latest model smartphone', 800.00, 20),
-- ('Headphones32', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor32', '4K UHD Monitor', 400.00, 15),
-- ('Laptop33', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone33', 'Latest model smartphone', 800.00, 20),
-- ('Headphones33', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor33', '4K UHD Monitor', 400.00, 15),
-- ('Laptop34', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone34', 'Latest model smartphone', 800.00, 20),
-- ('Headphones34', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor34', '4K UHD Monitor', 400.00, 15),
-- ('Laptop35', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone35', 'Latest model smartphone', 800.00, 20),
-- ('Headphones35', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor35', '4K UHD Monitor', 400.00, 15),
-- ('Laptop36', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone36', 'Latest model smartphone', 800.00, 20),
-- ('Headphones36', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor36', '4K UHD Monitor', 400.00, 15),
-- ('Laptop37', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone37', 'Latest model smartphone', 800.00, 20),
-- ('Headphones37', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor37', '4K UHD Monitor', 400.00, 15),
-- ('Laptop38', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone38', 'Latest model smartphone', 800.00, 20),
-- ('Headphones38', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor38', '4K UHD Monitor', 400.00, 15),
-- ('Laptop39', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone39', 'Latest model smartphone', 800.00, 20),
-- ('Headphones39', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor39', '4K UHD Monitor', 400.00, 15),
-- ('Laptop40', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone40', 'Latest model smartphone', 800.00, 20),
-- ('Headphones40', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor40', '4K UHD Monitor', 400.00, 15),
-- ('Laptop41', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone41', 'Latest model smartphone', 800.00, 20),
-- ('Headphones41', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor41', '4K UHD Monitor', 400.00, 15),
-- ('Laptop42', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone42', 'Latest model smartphone', 800.00, 20),
-- ('Headphones42', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor42', '4K UHD Monitor', 400.00, 15),
-- ('Laptop43', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone43', 'Latest model smartphone', 800.00, 20),
-- ('Headphones43', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor43', '4K UHD Monitor', 400.00, 15),
-- ('Laptop44', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone44', 'Latest model smartphone', 800.00, 20),
-- ('Headphones44', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor44', '4K UHD Monitor', 400.00, 15),
-- ('Laptop45', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone45', 'Latest model smartphone', 800.00, 20),
-- ('Headphones45', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor45', '4K UHD Monitor', 400.00, 15),
-- ('Laptop46', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone46', 'Latest model smartphone', 800.00, 20),
-- ('Headphones46', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor46', '4K UHD Monitor', 400.00, 15),
-- ('Laptop47', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone47', 'Latest model smartphone', 800.00, 20),
-- ('Headphones47', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor47', '4K UHD Monitor', 400.00, 15),
-- ('Laptop48', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone48', 'Latest model smartphone', 800.00, 20),
-- ('Headphones48', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor48', '4K UHD Monitor', 400.00, 15),
-- ('Laptop49', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone49', 'Latest model smartphone', 800.00, 20),
-- ('Headphones49', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor49', '4K UHD Monitor', 400.00, 15),
-- ('Laptop50', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone50', 'Latest model smartphone', 800.00, 20),
-- ('Headphones50', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor50', '4K UHD Monitor', 400.00, 15),
-- ('Laptop51', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone51', 'Latest model smartphone', 800.00, 20),
-- ('Headphones51', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor51', '4K UHD Monitor', 400.00, 15),
-- ('Laptop52', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone52', 'Latest model smartphone', 800.00, 20),
-- ('Headphones52', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor52', '4K UHD Monitor', 400.00, 15),
-- ('Laptop53', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone53', 'Latest model smartphone', 800.00, 20),
-- ('Headphones53', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor53', '4K UHD Monitor', 400.00, 15),
-- ('Laptop54', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone54', 'Latest model smartphone', 800.00, 20),
-- ('Headphones54', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor54', '4K UHD Monitor', 400.00, 15),
-- ('Laptop55', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone55', 'Latest model smartphone', 800.00, 20),
-- ('Headphones55', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor55', '4K UHD Monitor', 400.00, 15),
-- ('Laptop56', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone56', 'Latest model smartphone', 800.00, 20),
-- ('Headphones56', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor56', '4K UHD Monitor', 400.00, 15),
-- ('Laptop57', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone57', 'Latest model smartphone', 800.00, 20),
-- ('Headphones57', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor57', '4K UHD Monitor', 400.00, 15),
-- ('Laptop58', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone58', 'Latest model smartphone', 800.00, 20),
-- ('Headphones58', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor58', '4K UHD Monitor', 400.00, 15),
-- ('Laptop59', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone59', 'Latest model smartphone', 800.00, 20),
-- ('Headphones59', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor59', '4K UHD Monitor', 400.00, 15),
-- ('Laptop60', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone60', 'Latest model smartphone', 800.00, 20),
-- ('Headphones60', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor60', '4K UHD Monitor', 400.00, 15),
-- ('Laptop61', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone61', 'Latest model smartphone', 800.00, 20),
-- ('Headphones61', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor61', '4K UHD Monitor', 400.00, 15),
-- ('Laptop62', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone62', 'Latest model smartphone', 800.00, 20),
-- ('Headphones62', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor62', '4K UHD Monitor', 400.00, 15),
-- ('Laptop63', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone63', 'Latest model smartphone', 800.00, 20),
-- ('Headphones63', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor63', '4K UHD Monitor', 400.00, 15),
-- ('Laptop64', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone64', 'Latest model smartphone', 800.00, 20),
-- ('Headphones64', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor64', '4K UHD Monitor', 400.00, 15),
-- ('Laptop65', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone65', 'Latest model smartphone', 800.00, 20),
-- ('Headphones65', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor65', '4K UHD Monitor', 400.00, 15),
-- ('Laptop66', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone66', 'Latest model smartphone', 800.00, 20),
-- ('Headphones66', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor66', '4K UHD Monitor', 400.00, 15),
-- ('Laptop67', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone67', 'Latest model smartphone', 800.00, 20),
-- ('Headphones67', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor67', '4K UHD Monitor', 400.00, 15),
-- ('Laptop68', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone68', 'Latest model smartphone', 800.00, 20),
-- ('Headphones68', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor68', '4K UHD Monitor', 400.00, 15),
-- ('Laptop69', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone69', 'Latest model smartphone', 800.00, 20),
-- ('Headphones69', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor69', '4K UHD Monitor', 400.00, 15),
-- ('Laptop70', 'High-performance laptop', 1200.00, 10),
-- ('Smartphone70', 'Latest model smartphone', 800.00, 20),
-- ('Headphones70', 'Noise-canceling headphones', 150.00, 50),
-- ('Monitor70', '4K UHD Monitor', 400.00, 15);


-- Sample Orders
INSERT INTO test_schema.orders (user_id, total_amount, status) VALUES 
(1, 0, 'completed'),
(2, 0, 'pending');

-- Sample Order Items (Trigger will update the total_amount)
INSERT INTO test_schema.order_items (order_id, product_id, quantity, price_at_purchase) VALUES 
(1, 1, 1, 1200.00), -- Laptop
(1, 2, 1, 800.00),  -- Smartphone
(2, 3, 1, 150.00);  -- Headphones

-- Sample Payments
INSERT INTO test_schema.payments (order_id, amount, payment_method)
VALUES
(1, 2000.00, 'credit_card'),
(2, 150.00, 'paypal');

-- Test Casting Table for type casting validation
CREATE TABLE IF NOT EXISTS test_schema.test_casting
(
    id
    SERIAL
    PRIMARY
    KEY,
    bigint_col
    BIGINT
    NOT
    NULL,
    integer_col
    INTEGER
    NOT
    NULL,
    smallint_col
    SMALLINT
    NOT
    NULL,
    numeric_col
    NUMERIC
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
INSERT INTO test_schema.test_casting (bigint_col, integer_col, smallint_col, numeric_col, description)
VALUES (9223372036854775807, 2147483647, 32767, 99999999.99, 'Maximum values'),
       (1000000, 1000, 100, 1234.56, 'Regular values'),
       (-9223372036854775808, -2147483648, -32768, -99999999.99, 'Minimum values');

-- Data Types Table for exhaustive type testing
CREATE TABLE IF NOT EXISTS test_schema.data_types (
    id SERIAL PRIMARY KEY,
    -- Numeric types
    col_smallint SMALLINT,
    col_integer INTEGER,
    col_bigint BIGINT,
    col_decimal DECIMAL(10,2),
    col_numeric NUMERIC(10,2),
    col_real REAL,
    col_double DOUBLE PRECISION,
    col_money MONEY,
    -- String types
    col_char CHAR(10),
    col_varchar VARCHAR(255),
    col_text TEXT,
    col_bytea BYTEA,
    -- Date/Time types
    col_timestamp TIMESTAMP,
    col_timestamptz TIMESTAMPTZ,
    col_date DATE,
    col_time TIME,
    col_timetz TIMETZ,
    -- Boolean
    col_boolean BOOLEAN,
    -- Geometric types
    col_point POINT,
    col_line LINE,
    col_lseg LSEG,
    col_box BOX,
    col_path PATH,
    col_polygon POLYGON,
    col_circle CIRCLE,
    -- Network types
    col_cidr CIDR,
    col_inet INET,
    col_macaddr MACADDR,
    -- Special types
    col_uuid UUID,
    col_xml XML,
    col_json JSON,
    col_jsonb JSONB
);

-- Insert seed data for data_types
INSERT INTO test_schema.data_types (
    col_smallint, col_integer, col_bigint, col_decimal, col_numeric,
    col_real, col_double, col_money, col_char, col_varchar, col_text,
    col_bytea, col_timestamp, col_timestamptz, col_date, col_time, col_timetz,
    col_boolean, col_point, col_line, col_lseg, col_box, col_path, col_polygon,
    col_circle, col_cidr, col_inet, col_macaddr, col_uuid, col_xml, col_json, col_jsonb
) VALUES (
    100, 1000, 100000, 123.45, 678.90,
    1.5, 2.5, 99.99, 'test', 'varchar_val', 'text_value',
    E'\\x48454c4c4f', '2025-01-01 12:00:00', '2025-01-01 12:00:00+00', '2025-01-01', '12:00:00', '12:00:00+00',
    true, '(1,2)', '{1,2,3}', '[(0,0),(1,1)]', '((0,0),(1,1))', '((0,0),(1,1),(1,0))', '((0,0),(1,0),(1,1),(0,1))',
    '<(0,0),5>', '192.168.0.0/24', '192.168.0.1', '08:00:2b:01:02:03',
    '550e8400-e29b-41d4-a716-446655440000', '<root>test</root>',
    '{"key":"value"}', '{"key":"value"}'
);
