#!/bin/sh
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

# OpenSearch init script for E2E tests
# OpenSearch uses the same REST API as Elasticsearch

set -e

OPENSEARCH_URL="${OS_HOST:-http://e2e_opensearch:9200}"

CURL_OPTS=""
if [ -n "$OS_USER" ] && [ -n "$OS_PASS" ]; then
  CURL_OPTS="$CURL_OPTS -u $OS_USER:$OS_PASS"
fi
if [ -n "$OS_INSECURE" ]; then
  CURL_OPTS="$CURL_OPTS -k"
fi

echo "Waiting for OpenSearch at $OPENSEARCH_URL..."
until curl -s $CURL_OPTS "$OPENSEARCH_URL" > /dev/null; do
  sleep 5
done

echo "OpenSearch is up!"

# Delete existing indices
echo "Deleting existing indices (if any)..."
curl -X DELETE $CURL_OPTS "$OPENSEARCH_URL/users,products,orders,order_items,payments" -s || true
sleep 2

# Creating Users Index
echo "Creating users index..."
curl -X PUT $CURL_OPTS "$OPENSEARCH_URL/users" -H "Content-Type: application/json" -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "username": { "type": "keyword" },
      "email": { "type": "keyword" },
      "password": { "type": "text" },
      "created_at": { "type": "date" }
    }
  }
}'

# Creating Products Index
echo "Creating products index..."
curl -X PUT $CURL_OPTS "$OPENSEARCH_URL/products" -H "Content-Type: application/json" -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "name": { "type": "keyword" },
      "description": { "type": "text" },
      "price": { "type": "float" },
      "stock_quantity": { "type": "integer" },
      "created_at": { "type": "date" }
    }
  }
}'

# Creating Orders Index
echo "Creating orders index..."
curl -X PUT $CURL_OPTS "$OPENSEARCH_URL/orders" -H "Content-Type: application/json" -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "user_id": { "type": "integer" },
      "order_date": { "type": "date" },
      "total_amount": { "type": "float" },
      "status": { "type": "keyword" }
    }
  }
}'

# Creating Order Items Index
echo "Creating order_items index..."
curl -X PUT $CURL_OPTS "$OPENSEARCH_URL/order_items" -H "Content-Type: application/json" -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "order_id": { "type": "integer" },
      "product_id": { "type": "integer" },
      "quantity": { "type": "integer" },
      "price_at_purchase": { "type": "float" }
    }
  }
}'

# Creating Payments Index
echo "Creating payments index..."
curl -X PUT $CURL_OPTS "$OPENSEARCH_URL/payments" -H "Content-Type: application/json" -d '{
  "settings": {
    "number_of_shards": 1,
    "number_of_replicas": 0
  },
  "mappings": {
    "properties": {
      "order_id": { "type": "integer" },
      "payment_date": { "type": "date" },
      "amount": { "type": "float" },
      "payment_method": { "type": "keyword" }
    }
  }
}'

# Inserting Sample Data
echo "Inserting users..."
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/users/_doc/1" -H "Content-Type: application/json" -d '{"username": "john_doe", "email": "john@example.com", "password": "securepassword1", "created_at": "2024-01-01T12:00:00"}'
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/users/_doc/2" -H "Content-Type: application/json" -d '{"username": "jane_smith", "email": "jane@example.com", "password": "securepassword2", "created_at": "2024-01-02T12:00:00"}'
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/users/_doc/3" -H "Content-Type: application/json" -d '{"username": "admin_user", "email": "admin@example.com", "password": "adminpass", "created_at": "2024-01-03T12:00:00"}'

echo "Inserting products..."
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/products/_doc/1" -H "Content-Type: application/json" -d '{"name": "Laptop", "description": "High-performance laptop", "price": 1200.00, "stock_quantity": 10, "created_at": "2024-01-01T12:00:00"}'
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/products/_doc/2" -H "Content-Type: application/json" -d '{"name": "Smartphone", "description": "Latest model smartphone", "price": 800.00, "stock_quantity": 20, "created_at": "2024-01-02T12:00:00"}'
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/products/_doc/3" -H "Content-Type: application/json" -d '{"name": "Headphones", "description": "Noise-canceling headphones", "price": 150.00, "stock_quantity": 50, "created_at": "2024-01-03T12:00:00"}'

echo "Inserting orders..."
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/orders/_doc/1" -H "Content-Type: application/json" -d '{"user_id": 1, "order_date": "2024-01-10T12:00:00", "total_amount": 2000.00, "status": "completed"}'
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/orders/_doc/2" -H "Content-Type: application/json" -d '{"user_id": 2, "order_date": "2024-01-11T12:00:00", "total_amount": 150.00, "status": "pending"}'

echo "Inserting order items..."
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/order_items/_doc/1" -H "Content-Type: application/json" -d '{"order_id": 1, "product_id": 1, "quantity": 1, "price_at_purchase": 1200.00}'
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/order_items/_doc/2" -H "Content-Type: application/json" -d '{"order_id": 1, "product_id": 2, "quantity": 1, "price_at_purchase": 800.00}'
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/order_items/_doc/3" -H "Content-Type: application/json" -d '{"order_id": 2, "product_id": 3, "quantity": 1, "price_at_purchase": 150.00}'

echo "Inserting payments..."
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/payments/_doc/1" -H "Content-Type: application/json" -d '{"order_id": 1, "payment_date": "2024-01-12T12:00:00", "amount": 2000.00, "payment_method": "credit_card"}'
curl -X POST $CURL_OPTS "$OPENSEARCH_URL/payments/_doc/2" -H "Content-Type: application/json" -d '{"order_id": 2, "payment_date": "2024-01-13T12:00:00", "amount": 150.00, "payment_method": "paypal"}'

echo "OpenSearch data initialization complete!"
