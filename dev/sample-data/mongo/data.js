/**
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

// Switch to the database (creates if it doesn't exist)
db = db.getSiblingDB('test_db');

// Create Collections
db.createCollection("users");
db.createCollection("products");
db.createCollection("orders");
db.createCollection("order_items");
db.createCollection("payments");

// Ensure Unique Indexes
db.users.createIndex({ email: 1 }, { unique: true });
db.users.createIndex({ username: 1 }, { unique: true });
db.products.createIndex({ name: 1 }, { unique: true });
db.orders.createIndex({ user_id: 1 });
db.order_items.createIndex({ order_id: 1 });
db.order_items.createIndex({ product_id: 1 });
db.payments.createIndex({ order_id: 1 });

// Insert Sample Users
db.users.insertMany([
    { username: "john_doe", email: "john@example.com", password: "securepassword1", created_at: new Date() },
    { username: "jane_smith", email: "jane@example.com", password: "securepassword2", created_at: new Date() },
    { username: "admin_user", email: "admin@example.com", password: "adminpass", created_at: new Date() }
]);

// Insert Sample Products
db.products.insertMany([
    { name: "Laptop", description: "High-performance laptop", price: 1200.00, stock_quantity: 10, created_at: new Date() },
    { name: "Smartphone", description: "Latest model smartphone", price: 800.00, stock_quantity: 20, created_at: new Date() },
    { name: "Headphones", description: "Noise-canceling headphones", price: 150.00, stock_quantity: 50, created_at: new Date() },
    { name: "Monitor", description: "4K UHD Monitor", price: 400.00, stock_quantity: 15, created_at: new Date() }
]);

// Insert Sample Orders
db.orders.insertMany([
    { user_id: db.users.findOne({ username: "john_doe" })._id, order_date: new Date(), total_amount: 0, status: "completed" },
    { user_id: db.users.findOne({ username: "jane_smith" })._id, order_date: new Date(), total_amount: 0, status: "pending" }
]);

// Insert Sample Order Items
db.order_items.insertMany([
    { order_id: db.orders.findOne({ status: "completed" })._id, product_id: db.products.findOne({ name: "Laptop" })._id, quantity: 1, price_at_purchase: 1200.00 },
    { order_id: db.orders.findOne({ status: "completed" })._id, product_id: db.products.findOne({ name: "Smartphone" })._id, quantity: 1, price_at_purchase: 800.00 },
    { order_id: db.orders.findOne({ status: "pending" })._id, product_id: db.products.findOne({ name: "Headphones" })._id, quantity: 1, price_at_purchase: 150.00 }
]);

// Insert Sample Payments
db.payments.insertMany([
    { order_id: db.orders.findOne({ status: "completed" })._id, payment_date: new Date(), amount: 2000.00, payment_method: "credit_card" },
    { order_id: db.orders.findOne({ status: "pending" })._id, payment_date: new Date(), amount: 150.00, payment_method: "paypal" }
]);

// Aggregation View Equivalent: Order Summary
db.createCollection("order_summary", {
    viewOn: "orders",
    pipeline: [
        {
            $lookup: {
                from: "users",
                localField: "user_id",
                foreignField: "_id",
                as: "user"
            }
        },
        { $unwind: "$user" },
        {
            $project: {
                order_id: "$_id",
                username: "$user.username",
                order_date: 1,
                status: 1,
                total_amount: 1
            }
        }
    ]
});