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

-- Sample Data for Products
INSERT INTO test_schema.products (name, description, price, stock_quantity) VALUES 
('Laptop', 'High-performance laptop', 1200.00, 10),
('Smartphone', 'Latest model smartphone', 800.00, 20),
('Headphones', 'Noise-canceling headphones', 150.00, 50),
('Monitor', '4K UHD Monitor', 400.00, 15);

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
INSERT INTO test_schema.payments (order_id, amount, payment_method) VALUES 
(1, 2000.00, 'credit_card'),
(2, 150.00, 'paypal');
