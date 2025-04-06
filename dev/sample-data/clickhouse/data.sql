-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id UInt32,
    username String,
    email String,
    password String,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;

-- Products Table
CREATE TABLE IF NOT EXISTS products (
    id UInt32,
    name String,
    description String,
    price Decimal(10,2),
    stock_quantity UInt32 DEFAULT 0,
    created_at DateTime DEFAULT now()
) ENGINE = MergeTree()
ORDER BY id;

-- Orders Table
CREATE TABLE IF NOT EXISTS orders (
    id UInt32,
    user_id UInt32,
    order_date DateTime DEFAULT now(),
    total_amount Decimal(10,2) DEFAULT 0,
    status Enum('pending', 'completed', 'canceled') DEFAULT 'pending'
) ENGINE = MergeTree()
ORDER BY id;

-- Order Items Table
CREATE TABLE IF NOT EXISTS order_items (
    id UInt32,
    order_id UInt32,
    product_id UInt32,
    quantity UInt32 DEFAULT 1,
    price_at_purchase Decimal(10,2)
) ENGINE = MergeTree()
ORDER BY id;

-- Payments Table
CREATE TABLE IF NOT EXISTS payments (
    id UInt32,
    order_id UInt32,
    payment_date DateTime DEFAULT now(),
    amount Decimal(10,2),
    payment_method Enum('credit_card', 'paypal', 'bank_transfer')
) ENGINE = MergeTree()
ORDER BY id;


