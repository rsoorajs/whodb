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

-- StarRocks init script for E2E tests
-- StarRocks uses MySQL protocol but has different DDL:
--   - No AUTO_INCREMENT, no SERIAL
--   - No FOREIGN KEYS
--   - Tables use DISTRIBUTED BY HASH
--   - No triggers or stored procedures

CREATE DATABASE IF NOT EXISTS test_db;
USE test_db;

CREATE TABLE IF NOT EXISTS users (
    id INT NOT NULL,
    username VARCHAR(50) NOT NULL,
    email VARCHAR(100) NOT NULL,
    password VARCHAR(100) NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DISTRIBUTED BY HASH(id);

CREATE TABLE IF NOT EXISTS products (
    id INT NOT NULL,
    name VARCHAR(100) NOT NULL,
    description VARCHAR(500),
    price DECIMAL(10,2) NOT NULL,
    stock_quantity INT NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
)
DISTRIBUTED BY HASH(id);

CREATE TABLE IF NOT EXISTS orders (
    id INT NOT NULL,
    user_id INT NOT NULL,
    order_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending'
)
DISTRIBUTED BY HASH(id);

CREATE TABLE IF NOT EXISTS order_items (
    id INT NOT NULL,
    order_id INT NOT NULL,
    product_id INT NOT NULL,
    quantity INT NOT NULL,
    price_at_purchase DECIMAL(10,2) NOT NULL
)
DISTRIBUTED BY HASH(id);

CREATE TABLE IF NOT EXISTS payments (
    id INT NOT NULL,
    order_id INT NOT NULL,
    payment_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    amount DECIMAL(10,2) NOT NULL,
    payment_method VARCHAR(20)
)
DISTRIBUTED BY HASH(id);

-- Insert sample data
INSERT INTO users VALUES
    (1, 'john_doe', 'john@example.com', 'securepassword1', '2024-01-01 12:00:00'),
    (2, 'jane_smith', 'jane@example.com', 'securepassword2', '2024-01-02 12:00:00'),
    (3, 'admin_user', 'admin@example.com', 'adminpass', '2024-01-03 12:00:00');

INSERT INTO products VALUES
    (1, 'Laptop', 'High-performance laptop', 1200.00, 10, '2024-01-01 12:00:00'),
    (2, 'Smartphone', 'Latest model smartphone', 800.00, 20, '2024-01-02 12:00:00'),
    (3, 'Headphones', 'Noise-canceling headphones', 150.00, 50, '2024-01-03 12:00:00');

INSERT INTO orders VALUES
    (1, 1, '2024-01-10 12:00:00', 2000.00, 'completed'),
    (2, 2, '2024-01-11 12:00:00', 150.00, 'pending');

INSERT INTO order_items VALUES
    (1, 1, 1, 1, 1200.00),
    (2, 1, 2, 1, 800.00),
    (3, 2, 3, 1, 150.00);

INSERT INTO payments VALUES
    (1, 1, '2024-01-12 12:00:00', 2000.00, 'credit_card'),
    (2, 2, '2024-01-13 12:00:00', 150.00, 'paypal');
