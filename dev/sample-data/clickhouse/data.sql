-- Create Database
CREATE DATABASE IF NOT EXISTS test_db;

-- Users Table
CREATE TABLE IF NOT EXISTS test_db.users (
    id UInt32 PRIMARY KEY,
    username String,
    email String,
    password String,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;

-- Products Table
CREATE TABLE IF NOT EXISTS test_db.products (
    id UInt32 PRIMARY KEY,
    name String,
    description String,
    price Decimal(10,2),
    stock_quantity UInt32 DEFAULT 0,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;

-- Orders Table
CREATE TABLE IF NOT EXISTS test_db.orders (
    id UInt32 PRIMARY KEY,
    user_id UInt32,
    order_date DateTime DEFAULT now(),
    total_amount Decimal(10,2) DEFAULT 0,
    status Enum('pending', 'completed', 'canceled') DEFAULT 'pending'
) ENGINE = MergeTree()
ORDER BY id;

-- Order Items Table
CREATE TABLE IF NOT EXISTS test_db.order_items (
    id UInt32 PRIMARY KEY,
    order_id UInt32,
    product_id UInt32,
    quantity UInt32 DEFAULT 1,
    price_at_purchase Decimal(10,2)
) ENGINE = MergeTree()
ORDER BY id;

-- Payments Table
CREATE TABLE IF NOT EXISTS test_db.payments (
    id UInt32 PRIMARY KEY,
    order_id UInt32,
    payment_date DateTime DEFAULT now(),
    amount Decimal(10,2),
    payment_method Enum('credit_card', 'paypal', 'bank_transfer')
) ENGINE = MergeTree()
ORDER BY id;

-- View for Order Summary
CREATE VIEW IF NOT EXISTS test_db.order_summary AS
SELECT 
    o.id AS order_id,
    u.username,
    o.order_date,
    o.status,
    o.total_amount
FROM test_db.orders o
JOIN test_db.users u ON o.user_id = u.id;

-- Sample Data for Users
INSERT INTO test_db.users (id, username, email, password) VALUES 
(1, 'john_doe', 'john@example.com', 'securepassword1'),
(2, 'jane_smith', 'jane@example.com', 'securepassword2'),
(3, 'admin_user', 'admin@example.com', 'adminpass');

-- Sample Data for Products
INSERT INTO test_db.products (id, name, description, price, stock_quantity) VALUES 
(1, 'Laptop', 'High-performance laptop', 1200.00, 10),
(2, 'Smartphone', 'Latest model smartphone', 800.00, 20),
(3, 'Headphones', 'Noise-canceling headphones', 150.00, 50),
(4, 'Monitor', '4K UHD Monitor', 400.00, 15);

-- Sample Orders
INSERT INTO test_db.orders (id, user_id, total_amount, status) VALUES 
(1, 1, 2000.00, 'completed'),
(2, 2, 150.00, 'pending');

-- Sample Order Items
INSERT INTO test_db.order_items (id, order_id, product_id, quantity, price_at_purchase) VALUES 
(1, 1, 1, 1, 1200.00),
(2, 1, 2, 1, 800.00),
(3, 2, 3, 1, 150.00);

-- Sample Payments
INSERT INTO test_db.payments (id, order_id, amount, payment_method) VALUES 
(1, 1, 2000.00, 'credit_card'),
(2, 2, 150.00, 'paypal');
