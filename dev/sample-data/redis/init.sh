#!/bin/bash

#
# Copyright 2026 Clidey, Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

# Use environment variables with defaults for non-SSL
REDIS_HOST="${REDIS_HOST:-e2e_redis}"
REDIS_PORT="${REDIS_PORT:-6379}"
REDIS_PASSWORD="${REDIS_PASSWORD:-password}"

# Build redis-cli options
REDIS_CLI="redis-cli -h $REDIS_HOST -p $REDIS_PORT -a $REDIS_PASSWORD"

# Add TLS options if certificate is provided
if [ -n "$REDIS_CACERT" ]; then
  REDIS_CLI="$REDIS_CLI --tls --cacert $REDIS_CACERT"
fi

# Wait for Redis to be ready
until $REDIS_CLI ping; do
  echo "Waiting for Redis at $REDIS_HOST:$REDIS_PORT to be ready..."
  sleep 1
done

echo "Redis is ready. Loading sample data..."

# Clear any existing data
$REDIS_CLI FLUSHALL

# Users hash - storing user data
$REDIS_CLI HSET user:1 id 1 username "johndoe" email "john@example.com" password "hashed_password_1" created_at "2023-01-15T10:30:00Z"
$REDIS_CLI HSET user:2 id 2 username "janesmith" email "jane@example.com" password "hashed_password_2" created_at "2023-02-20T14:45:00Z"
$REDIS_CLI HSET user:3 id 3 username "bobwilson" email "bob@example.com" password "hashed_password_3" created_at "2023-03-10T09:15:00Z"
$REDIS_CLI HSET user:4 id 4 username "alicebrown" email "alice@example.com" password "hashed_password_4" created_at "2023-04-05T16:20:00Z"
$REDIS_CLI HSET user:5 id 5 username "charliemiller" email "charlie@example.com" password "hashed_password_5" created_at "2023-05-12T11:00:00Z"

# Products sorted set (by price) and hash
$REDIS_CLI HSET product:1 id 1 name "Laptop" description "High-performance laptop with SSD" price 999.99 stock_quantity 50 created_at "2023-01-01T00:00:00Z"
$REDIS_CLI ZADD products:by_price 999.99 1

$REDIS_CLI HSET product:2 id 2 name "Mouse" description "Wireless optical mouse" price 29.99 stock_quantity 200 created_at "2023-01-02T00:00:00Z"
$REDIS_CLI ZADD products:by_price 29.99 2

$REDIS_CLI HSET product:3 id 3 name "Keyboard" description "Mechanical gaming keyboard" price 79.99 stock_quantity 150 created_at "2023-01-03T00:00:00Z"
$REDIS_CLI ZADD products:by_price 79.99 3

$REDIS_CLI HSET product:4 id 4 name "Monitor" description "27-inch 4K monitor" price 399.99 stock_quantity 75 created_at "2023-01-04T00:00:00Z"
$REDIS_CLI ZADD products:by_price 399.99 4

$REDIS_CLI HSET product:5 id 5 name "Headphones" description "Noise-cancelling headphones" price 199.99 stock_quantity 100 created_at "2023-01-05T00:00:00Z"
$REDIS_CLI ZADD products:by_price 199.99 5

# Orders - using lists and hashes
$REDIS_CLI HSET order:1 id 1 user_id 1 order_date "2023-06-01T10:00:00Z" total_amount 1079.98 status "completed"
$REDIS_CLI LPUSH user:1:orders 1
$REDIS_CLI LPUSH orders:recent 1

$REDIS_CLI HSET order:2 id 2 user_id 2 order_date "2023-06-02T11:30:00Z" total_amount 109.98 status "completed"
$REDIS_CLI LPUSH user:2:orders 2
$REDIS_CLI LPUSH orders:recent 2

$REDIS_CLI HSET order:3 id 3 user_id 3 order_date "2023-06-03T14:15:00Z" total_amount 479.98 status "pending"
$REDIS_CLI LPUSH user:3:orders 3
$REDIS_CLI LPUSH orders:recent 3

$REDIS_CLI HSET order:4 id 4 user_id 1 order_date "2023-06-04T09:45:00Z" total_amount 229.97 status "shipped"
$REDIS_CLI LPUSH user:1:orders 4
$REDIS_CLI LPUSH orders:recent 4

$REDIS_CLI HSET order:5 id 5 user_id 4 order_date "2023-06-05T16:00:00Z" total_amount 1599.96 status "completed"
$REDIS_CLI LPUSH user:4:orders 5
$REDIS_CLI LPUSH orders:recent 5

# Order items - using sets for order-to-items relationship
$REDIS_CLI HSET order_item:1 id 1 order_id 1 product_id 1 quantity 1 price_at_purchase 999.99
$REDIS_CLI SADD order:1:items 1

$REDIS_CLI HSET order_item:2 id 2 order_id 1 product_id 3 quantity 1 price_at_purchase 79.99
$REDIS_CLI SADD order:1:items 2

$REDIS_CLI HSET order_item:3 id 3 order_id 2 product_id 2 quantity 1 price_at_purchase 29.99
$REDIS_CLI SADD order:2:items 3

$REDIS_CLI HSET order_item:4 id 4 order_id 2 product_id 3 quantity 1 price_at_purchase 79.99
$REDIS_CLI SADD order:2:items 4

$REDIS_CLI HSET order_item:5 id 5 order_id 3 product_id 4 quantity 1 price_at_purchase 399.99
$REDIS_CLI SADD order:3:items 5

$REDIS_CLI HSET order_item:6 id 6 order_id 3 product_id 3 quantity 1 price_at_purchase 79.99
$REDIS_CLI SADD order:3:items 6

$REDIS_CLI HSET order_item:7 id 7 order_id 4 product_id 2 quantity 3 price_at_purchase 29.99
$REDIS_CLI SADD order:4:items 7

$REDIS_CLI HSET order_item:8 id 8 order_id 4 product_id 5 quantity 1 price_at_purchase 199.99
$REDIS_CLI SADD order:4:items 8

$REDIS_CLI HSET order_item:9 id 9 order_id 5 product_id 1 quantity 1 price_at_purchase 999.99
$REDIS_CLI SADD order:5:items 9

$REDIS_CLI HSET order_item:10 id 10 order_id 5 product_id 4 quantity 1 price_at_purchase 399.99
$REDIS_CLI SADD order:5:items 10

$REDIS_CLI HSET order_item:11 id 11 order_id 5 product_id 5 quantity 1 price_at_purchase 199.99
$REDIS_CLI SADD order:5:items 11

# Payments - using sorted set by date
$REDIS_CLI HSET payment:1 id 1 order_id 1 payment_date "2023-06-01T10:05:00Z" amount 1079.98 payment_method "credit_card"
$REDIS_CLI ZADD payments:by_date 1685616300 1

$REDIS_CLI HSET payment:2 id 2 order_id 2 payment_date "2023-06-02T11:35:00Z" amount 109.98 payment_method "paypal"
$REDIS_CLI ZADD payments:by_date 1685706900 2

$REDIS_CLI HSET payment:3 id 3 order_id 5 payment_date "2023-06-05T16:05:00Z" amount 1599.96 payment_method "credit_card"
$REDIS_CLI ZADD payments:by_date 1685980500 3

# Additional Redis-specific data structures

# Shopping cart using lists
$REDIS_CLI LPUSH cart:user:1 "product:2:quantity:2"
$REDIS_CLI LPUSH cart:user:1 "product:5:quantity:1"

# Product categories using sets
$REDIS_CLI SADD category:electronics 1 2 3 4 5
$REDIS_CLI SADD category:accessories 2 3 5
$REDIS_CLI SADD category:computers 1 4

# User sessions with expiry
$REDIS_CLI SETEX session:abc123 3600 "user:1"
$REDIS_CLI SETEX session:def456 3600 "user:2"

# Product views counter
$REDIS_CLI INCR product:1:views
$REDIS_CLI INCRBY product:1:views 49
$REDIS_CLI INCRBY product:2:views 150
$REDIS_CLI INCRBY product:3:views 120
$REDIS_CLI INCRBY product:4:views 80
$REDIS_CLI INCRBY product:5:views 95

# Search index using sorted sets for autocomplete
$REDIS_CLI ZADD search:products 0 "laptop"
$REDIS_CLI ZADD search:products 0 "mouse"
$REDIS_CLI ZADD search:products 0 "keyboard"
$REDIS_CLI ZADD search:products 0 "monitor"
$REDIS_CLI ZADD search:products 0 "headphones"

# Inventory tracking
$REDIS_CLI SET inventory:product:1 50
$REDIS_CLI SET inventory:product:2 200
$REDIS_CLI SET inventory:product:3 150
$REDIS_CLI SET inventory:product:4 75
$REDIS_CLI SET inventory:product:5 100

# Top selling products (sorted set by sales count)
$REDIS_CLI ZADD bestsellers 15 1
$REDIS_CLI ZADD bestsellers 45 2
$REDIS_CLI ZADD bestsellers 30 3
$REDIS_CLI ZADD bestsellers 20 4
$REDIS_CLI ZADD bestsellers 25 5

# Note: JSON commands would require RedisJSON module to be installed
# Skipping JSON.SET commands as they're not available in standard Redis

echo "Sample data loaded successfully!"

# Display summary
echo "Data summary:"
$REDIS_CLI DBSIZE