#!/bin/sh
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

# Wait for CockroachDB to be ready, then run init SQL.
# Supports both insecure and secure (certs) modes via COCKROACH_CERTS_DIR.
set -e

COCKROACH_HOST="${COCKROACH_HOST:-e2e_cockroachdb}"
COCKROACH_PORT="${COCKROACH_PORT:-26257}"
MAX_RETRIES=30
RETRY_INTERVAL=2

# Build connection flags: --insecure or --certs-dir
if [ -n "$COCKROACH_CERTS_DIR" ]; then
    CONN_FLAGS="--certs-dir=$COCKROACH_CERTS_DIR"
    echo "Using secure mode with certs from $COCKROACH_CERTS_DIR"
else
    CONN_FLAGS="--insecure"
    echo "Using insecure mode"
fi

echo "Waiting for CockroachDB at ${COCKROACH_HOST}:${COCKROACH_PORT}..."

retries=0
while [ $retries -lt $MAX_RETRIES ]; do
    if cockroach sql $CONN_FLAGS --host="${COCKROACH_HOST}:${COCKROACH_PORT}" -e "SELECT 1" > /dev/null 2>&1; then
        echo "CockroachDB is ready!"
        break
    fi
    retries=$((retries + 1))
    echo "Attempt $retries/$MAX_RETRIES - CockroachDB not ready yet, retrying in ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

if [ $retries -ge $MAX_RETRIES ]; then
    echo "ERROR: CockroachDB did not become ready within $((MAX_RETRIES * RETRY_INTERVAL))s"
    exit 1
fi

echo "Running init SQL..."
cockroach sql $CONN_FLAGS --host="${COCKROACH_HOST}:${COCKROACH_PORT}" < /data.sql

echo "CockroachDB initialization complete!"
