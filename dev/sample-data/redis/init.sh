#!/bin/bash

# Wait for Redis to be ready
until redis-cli -h e2e_redis -a password -a password ping; do
  echo "Waiting for Redis to be ready..."
  sleep 1
done

echo "Redis is ready. Loading sample data..."

# Clear any existing data
redis-cli -h e2e_redis -a password -a password FLUSHALL

# Users hash - storing user data
redis-cli -h e2e_redis -a password HSET user:1 id 1 username "johndoe" email "john@example.com" password "hashed_password_1" created_at "2023-01-15T10:30:00Z"
redis-cli -h e2e_redis -a password HSET user:2 id 2 username "janesmith" email "jane@example.com" password "hashed_password_2" created_at "2023-02-20T14:45:00Z"
redis-cli -h e2e_redis -a password HSET user:3 id 3 username "bobwilson" email "bob@example.com" password "hashed_password_3" created_at "2023-03-10T09:15:00Z"
redis-cli -h e2e_redis -a password HSET user:4 id 4 username "alicebrown" email "alice@example.com" password "hashed_password_4" created_at "2023-04-05T16:20:00Z"
redis-cli -h e2e_redis -a password HSET user:5 id 5 username "charliemiller" email "charlie@example.com" password "hashed_password_5" created_at "2023-05-12T11:00:00Z"

# Products sorted set (by price) and hash
redis-cli -h e2e_redis -a password HSET product:1 id 1 name "Laptop" description "High-performance laptop with SSD" price 999.99 stock_quantity 50 created_at "2023-01-01T00:00:00Z"
redis-cli -h e2e_redis -a password ZADD products:by_price 999.99 1

redis-cli -h e2e_redis -a password HSET product:2 id 2 name "Mouse" description "Wireless optical mouse" price 29.99 stock_quantity 200 created_at "2023-01-02T00:00:00Z"
redis-cli -h e2e_redis -a password ZADD products:by_price 29.99 2

redis-cli -h e2e_redis -a password HSET product:3 id 3 name "Keyboard" description "Mechanical gaming keyboard" price 79.99 stock_quantity 150 created_at "2023-01-03T00:00:00Z"
redis-cli -h e2e_redis -a password ZADD products:by_price 79.99 3

redis-cli -h e2e_redis -a password HSET product:4 id 4 name "Monitor" description "27-inch 4K monitor" price 399.99 stock_quantity 75 created_at "2023-01-04T00:00:00Z"
redis-cli -h e2e_redis -a password ZADD products:by_price 399.99 4

redis-cli -h e2e_redis -a password HSET product:5 id 5 name "Headphones" description "Noise-cancelling headphones" price 199.99 stock_quantity 100 created_at "2023-01-05T00:00:00Z"
redis-cli -h e2e_redis -a password ZADD products:by_price 199.99 5

# Orders - using lists and hashes
redis-cli -h e2e_redis -a password HSET order:1 id 1 user_id 1 order_date "2023-06-01T10:00:00Z" total_amount 1079.98 status "completed"
redis-cli -h e2e_redis -a password LPUSH user:1:orders 1
redis-cli -h e2e_redis -a password LPUSH orders:recent 1

redis-cli -h e2e_redis -a password HSET order:2 id 2 user_id 2 order_date "2023-06-02T11:30:00Z" total_amount 109.98 status "completed"
redis-cli -h e2e_redis -a password LPUSH user:2:orders 2
redis-cli -h e2e_redis -a password LPUSH orders:recent 2

redis-cli -h e2e_redis -a password HSET order:3 id 3 user_id 3 order_date "2023-06-03T14:15:00Z" total_amount 479.98 status "pending"
redis-cli -h e2e_redis -a password LPUSH user:3:orders 3
redis-cli -h e2e_redis -a password LPUSH orders:recent 3

redis-cli -h e2e_redis -a password HSET order:4 id 4 user_id 1 order_date "2023-06-04T09:45:00Z" total_amount 229.97 status "shipped"
redis-cli -h e2e_redis -a password LPUSH user:1:orders 4
redis-cli -h e2e_redis -a password LPUSH orders:recent 4

redis-cli -h e2e_redis -a password HSET order:5 id 5 user_id 4 order_date "2023-06-05T16:00:00Z" total_amount 1599.96 status "completed"
redis-cli -h e2e_redis -a password LPUSH user:4:orders 5
redis-cli -h e2e_redis -a password LPUSH orders:recent 5

# Order items - using sets for order-to-items relationship
redis-cli -h e2e_redis -a password HSET order_item:1 id 1 order_id 1 product_id 1 quantity 1 price_at_purchase 999.99
redis-cli -h e2e_redis -a password SADD order:1:items 1

redis-cli -h e2e_redis -a password HSET order_item:2 id 2 order_id 1 product_id 3 quantity 1 price_at_purchase 79.99
redis-cli -h e2e_redis -a password SADD order:1:items 2

redis-cli -h e2e_redis -a password HSET order_item:3 id 3 order_id 2 product_id 2 quantity 1 price_at_purchase 29.99
redis-cli -h e2e_redis -a password SADD order:2:items 3

redis-cli -h e2e_redis -a password HSET order_item:4 id 4 order_id 2 product_id 3 quantity 1 price_at_purchase 79.99
redis-cli -h e2e_redis -a password SADD order:2:items 4

redis-cli -h e2e_redis -a password HSET order_item:5 id 5 order_id 3 product_id 4 quantity 1 price_at_purchase 399.99
redis-cli -h e2e_redis -a password SADD order:3:items 5

redis-cli -h e2e_redis -a password HSET order_item:6 id 6 order_id 3 product_id 3 quantity 1 price_at_purchase 79.99
redis-cli -h e2e_redis -a password SADD order:3:items 6

redis-cli -h e2e_redis -a password HSET order_item:7 id 7 order_id 4 product_id 2 quantity 3 price_at_purchase 29.99
redis-cli -h e2e_redis -a password SADD order:4:items 7

redis-cli -h e2e_redis -a password HSET order_item:8 id 8 order_id 4 product_id 5 quantity 1 price_at_purchase 199.99
redis-cli -h e2e_redis -a password SADD order:4:items 8

redis-cli -h e2e_redis -a password HSET order_item:9 id 9 order_id 5 product_id 1 quantity 1 price_at_purchase 999.99
redis-cli -h e2e_redis -a password SADD order:5:items 9

redis-cli -h e2e_redis -a password HSET order_item:10 id 10 order_id 5 product_id 4 quantity 1 price_at_purchase 399.99
redis-cli -h e2e_redis -a password SADD order:5:items 10

redis-cli -h e2e_redis -a password HSET order_item:11 id 11 order_id 5 product_id 5 quantity 1 price_at_purchase 199.99
redis-cli -h e2e_redis -a password SADD order:5:items 11

# Payments - using sorted set by date
redis-cli -h e2e_redis -a password HSET payment:1 id 1 order_id 1 payment_date "2023-06-01T10:05:00Z" amount 1079.98 payment_method "credit_card"
redis-cli -h e2e_redis -a password ZADD payments:by_date 1685616300 1

redis-cli -h e2e_redis -a password HSET payment:2 id 2 order_id 2 payment_date "2023-06-02T11:35:00Z" amount 109.98 payment_method "paypal"
redis-cli -h e2e_redis -a password ZADD payments:by_date 1685706900 2

redis-cli -h e2e_redis -a password HSET payment:3 id 3 order_id 5 payment_date "2023-06-05T16:05:00Z" amount 1599.96 payment_method "credit_card"
redis-cli -h e2e_redis -a password ZADD payments:by_date 1685980500 3

# Additional Redis-specific data structures

# Shopping cart using lists
redis-cli -h e2e_redis -a password LPUSH cart:user:1 "product:2:quantity:2"
redis-cli -h e2e_redis -a password LPUSH cart:user:1 "product:5:quantity:1"

# Product categories using sets
redis-cli -h e2e_redis -a password SADD category:electronics 1 2 3 4 5
redis-cli -h e2e_redis -a password SADD category:accessories 2 3 5
redis-cli -h e2e_redis -a password SADD category:computers 1 4

# User sessions with expiry
redis-cli -h e2e_redis -a password SETEX session:abc123 3600 "user:1"
redis-cli -h e2e_redis -a password SETEX session:def456 3600 "user:2"

# Product views counter
redis-cli -h e2e_redis -a password INCR product:1:views
redis-cli -h e2e_redis -a password INCRBY product:1:views 49
redis-cli -h e2e_redis -a password INCRBY product:2:views 150
redis-cli -h e2e_redis -a password INCRBY product:3:views 120
redis-cli -h e2e_redis -a password INCRBY product:4:views 80
redis-cli -h e2e_redis -a password INCRBY product:5:views 95

# Search index using sorted sets for autocomplete
redis-cli -h e2e_redis -a password ZADD search:products 0 "laptop"
redis-cli -h e2e_redis -a password ZADD search:products 0 "mouse"
redis-cli -h e2e_redis -a password ZADD search:products 0 "keyboard"
redis-cli -h e2e_redis -a password ZADD search:products 0 "monitor"
redis-cli -h e2e_redis -a password ZADD search:products 0 "headphones"

# Inventory tracking
redis-cli -h e2e_redis -a password SET inventory:product:1 50
redis-cli -h e2e_redis -a password SET inventory:product:2 200
redis-cli -h e2e_redis -a password SET inventory:product:3 150
redis-cli -h e2e_redis -a password SET inventory:product:4 75
redis-cli -h e2e_redis -a password SET inventory:product:5 100

# Top selling products (sorted set by sales count)
redis-cli -h e2e_redis -a password ZADD bestsellers 15 1
redis-cli -h e2e_redis -a password ZADD bestsellers 45 2
redis-cli -h e2e_redis -a password ZADD bestsellers 30 3
redis-cli -h e2e_redis -a password ZADD bestsellers 20 4
redis-cli -h e2e_redis -a password ZADD bestsellers 25 5

# Note: JSON commands would require RedisJSON module to be installed
# Skipping JSON.SET commands as they're not available in standard Redis

echo "Sample data loaded successfully!"

# Display summary
echo "Data summary:"
redis-cli -h e2e_redis -a password DBSIZE