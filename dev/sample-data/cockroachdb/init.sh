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

# Wait for CockroachDB to be ready, then run init SQL.
# Supports both insecure and secure (certs) modes via COCKROACH_CERTS_DIR.
set -e

COCKROACH_HOST="${COCKROACH_HOST:-e2e_cockroachdb}"
COCKROACH_PORT="${COCKROACH_PORT:-26257}"
COCKROACH_INIT_HOST="${COCKROACH_INIT_HOST:-$COCKROACH_HOST}"
COCKROACH_INIT_PORT="${COCKROACH_INIT_PORT:-26357}"
MAX_RETRIES="${COCKROACH_MAX_RETRIES:-90}"
RETRY_INTERVAL="${COCKROACH_RETRY_INTERVAL:-2}"
POST_INIT_STABLE_CHECKS="${COCKROACH_POST_INIT_STABLE_CHECKS:-5}"

# Build connection flags: --insecure or --certs-dir
if [ -n "$COCKROACH_CERTS_DIR" ]; then
    CONN_FLAGS="--certs-dir=$COCKROACH_CERTS_DIR"
    echo "Using secure mode with certs from $COCKROACH_CERTS_DIR"
else
    CONN_FLAGS="--insecure"
    echo "Using insecure mode"
fi

# For single-node mode, no cluster init is needed — skip straight to SQL readiness.
# For clustered mode, attempt cockroach init (idempotent if already initialized).
if [ "${COCKROACH_SKIP_INIT:-}" != "true" ]; then
    echo "Initializing CockroachDB at ${COCKROACH_INIT_HOST}:${COCKROACH_INIT_PORT}..."

    retries=0
    while [ $retries -lt $MAX_RETRIES ]; do
        init_output="$(cockroach init $CONN_FLAGS --host="${COCKROACH_INIT_HOST}:${COCKROACH_INIT_PORT}" 2>&1)" && {
            echo "CockroachDB cluster initialized!"
            break
        }

        if echo "$init_output" | grep -qi "already.*initialized"; then
            echo "CockroachDB cluster is already initialized!"
            break
        fi

        retries=$((retries + 1))
        echo "Attempt $retries/$MAX_RETRIES - CockroachDB init not ready yet, retrying in ${RETRY_INTERVAL}s..."
        sleep $RETRY_INTERVAL
    done

    if [ $retries -ge $MAX_RETRIES ]; then
        echo "$init_output"
        echo "ERROR: CockroachDB did not initialize within $((MAX_RETRIES * RETRY_INTERVAL))s"
        exit 1
    fi
fi

echo "Waiting for CockroachDB SQL at ${COCKROACH_HOST}:${COCKROACH_PORT}..."

retries=0
while [ $retries -lt $MAX_RETRIES ]; do
    if cockroach sql $CONN_FLAGS --host="${COCKROACH_HOST}:${COCKROACH_PORT}" -e "SELECT 1" > /dev/null 2>&1; then
        echo "CockroachDB SQL is ready!"
        break
    fi
    retries=$((retries + 1))
    echo "Attempt $retries/$MAX_RETRIES - CockroachDB SQL not ready yet, retrying in ${RETRY_INTERVAL}s..."
    sleep $RETRY_INTERVAL
done

if [ $retries -ge $MAX_RETRIES ]; then
    echo "ERROR: CockroachDB SQL did not become ready within $((MAX_RETRIES * RETRY_INTERVAL))s"
    exit 1
fi

echo "Applying single-node performance optimizations..."
cockroach sql $CONN_FLAGS --host="${COCKROACH_HOST}:${COCKROACH_PORT}" -e "
SET CLUSTER SETTING kv.range_merge.queue_interval = '50ms';
SET CLUSTER SETTING jobs.registry.interval.gc = '30s';
SET CLUSTER SETTING jobs.retention_time = '15s';
SET CLUSTER SETTING sql.stats.automatic_collection.enabled = false;
SET CLUSTER SETTING kv.range_split.by_load_merge_delay = '5s';
ALTER RANGE default CONFIGURE ZONE USING \"gc.ttlseconds\" = 600;
ALTER DATABASE system CONFIGURE ZONE USING \"gc.ttlseconds\" = 600;
"

echo "Running init SQL..."
cockroach sql $CONN_FLAGS --host="${COCKROACH_HOST}:${COCKROACH_PORT}" < /data.sql

echo "Checking CockroachDB seeded database stability..."
stable_checks=0
stability_attempts=0
STABILITY_USER_FLAGS=""
if [ -z "$COCKROACH_CERTS_DIR" ]; then
    STABILITY_USER_FLAGS="--user=user"
fi

while [ $stable_checks -lt $POST_INIT_STABLE_CHECKS ]; do
    if cockroach sql $CONN_FLAGS $STABILITY_USER_FLAGS --host="${COCKROACH_HOST}:${COCKROACH_PORT}" --database=test_db -e "SELECT datname FROM pg_database WHERE datistemplate = false AND datallowconn AND has_database_privilege(datname, 'CONNECT'); SELECT schema_name AS schemaname FROM information_schema.schemata WHERE has_schema_privilege(schema_name, 'USAGE') AND schema_name NOT IN ('information_schema', 'pg_catalog', 'crdb_internal', 'pg_extension'); SELECT table_name, table_type FROM information_schema.tables WHERE table_schema = 'test_schema'; SELECT column_name, data_type FROM information_schema.columns WHERE table_schema = 'test_schema' AND table_name = 'users' ORDER BY ordinal_position; SELECT COUNT(*) FROM test_schema.users; SELECT * FROM test_schema.users ORDER BY id LIMIT 100;" > /dev/null 2>&1; then
        stable_checks=$((stable_checks + 1))
        echo "CockroachDB seeded database check $stable_checks/$POST_INIT_STABLE_CHECKS succeeded"
    else
        stable_checks=0
        stability_attempts=$((stability_attempts + 1))
        if [ $stability_attempts -ge $MAX_RETRIES ]; then
            echo "ERROR: CockroachDB seeded database did not become stable within $((MAX_RETRIES * RETRY_INTERVAL))s"
            exit 1
        fi
        echo "CockroachDB seeded database check failed, retrying in ${RETRY_INTERVAL}s..."
    fi
    sleep $RETRY_INTERVAL
done

echo "CockroachDB initialization complete!"
