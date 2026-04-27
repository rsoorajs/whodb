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

set -e

YUGABYTEDB_HOST="${YUGABYTEDB_HOST:-e2e_yugabytedb}"
YUGABYTEDB_PORT="${YUGABYTEDB_PORT:-5433}"
MAX_RETRIES="${YUGABYTEDB_MAX_RETRIES:-90}"
RETRY_INTERVAL="${YUGABYTEDB_RETRY_INTERVAL:-2}"
SEED_RETRIES="${YUGABYTEDB_SEED_RETRIES:-3}"
SEED_RETRY_INTERVAL="${YUGABYTEDB_SEED_RETRY_INTERVAL:-10}"

run_admin_sql() {
    PGPASSWORD=yugabyte psql -v ON_ERROR_STOP=1 -h "$YUGABYTEDB_HOST" -p "$YUGABYTEDB_PORT" -U yugabyte "$@"
}

run_test_sql() {
    PGPASSWORD=password psql -v ON_ERROR_STOP=1 -h "$YUGABYTEDB_HOST" -p "$YUGABYTEDB_PORT" -U user -d test_db "$@"
}

wait_for_admin_sql() {
    retries=0

    echo "Waiting for YugabyteDB at ${YUGABYTEDB_HOST}:${YUGABYTEDB_PORT}..."
    while [ $retries -lt "$MAX_RETRIES" ]; do
        if run_admin_sql -c "SELECT 1" >/dev/null 2>&1; then
            echo "YugabyteDB admin connection is ready!"
            return 0
        fi

        retries=$((retries + 1))
        echo "Attempt $retries/$MAX_RETRIES - YugabyteDB not ready yet, retrying in ${RETRY_INTERVAL}s..."
        sleep "$RETRY_INTERVAL"
    done

    echo "ERROR: YugabyteDB did not become ready within $((MAX_RETRIES * RETRY_INTERVAL))s"
    exit 1
}

wait_for_test_sql() {
    retries=0

    echo "Waiting for YugabyteDB test connection..."
    while [ $retries -lt "$MAX_RETRIES" ]; do
        if run_test_sql -c "SELECT 1" >/dev/null 2>&1; then
            echo "YugabyteDB test connection is ready!"
            return 0
        fi

        retries=$((retries + 1))
        echo "Attempt $retries/$MAX_RETRIES - YugabyteDB test connection not ready yet, retrying in ${RETRY_INTERVAL}s..."
        sleep "$RETRY_INTERVAL"
    done

    echo "ERROR: YugabyteDB test connection did not become ready within $((MAX_RETRIES * RETRY_INTERVAL))s"
    exit 1
}

wait_for_admin_sql

if ! run_admin_sql -tAc "SELECT 1 FROM pg_database WHERE datname = 'test_db'" | grep -q 1; then
    run_admin_sql -c "CREATE DATABASE test_db;"
fi

run_admin_sql -c "DO \$\$ BEGIN IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'user') THEN CREATE USER \"user\" WITH PASSWORD 'password'; END IF; END \$\$;"
run_admin_sql -d test_db -c "GRANT ALL PRIVILEGES ON DATABASE test_db TO \"user\";"

wait_for_test_sql

seed_attempt=1
while [ $seed_attempt -le "$SEED_RETRIES" ]; do
    echo "Running YugabyteDB seed SQL (attempt $seed_attempt/$SEED_RETRIES)..."
    if run_test_sql -f /data.sql; then
        echo "YugabyteDB data loaded"
        exit 0
    fi

    if [ $seed_attempt -ge "$SEED_RETRIES" ]; then
        echo "ERROR: YugabyteDB seed SQL failed after $SEED_RETRIES attempts"
        exit 1
    fi

    seed_attempt=$((seed_attempt + 1))
    echo "YugabyteDB seed SQL failed, retrying in ${SEED_RETRY_INTERVAL}s..."
    sleep "$SEED_RETRY_INTERVAL"
    wait_for_test_sql
done
