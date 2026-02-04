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

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Generating SSL certificates for E2E testing..."
echo "WARNING: These certificates are for testing only. Do not use in production."

# Function to generate standard CA, server, and client certs
generate_standard_certs() {
    local DB=$1
    local EXTRA_SANS=${2:-""}

    echo ""
    echo "=== Generating certificates for $DB ==="

    CA_DIR="ca/$DB"
    SERVER_DIR="server/$DB"
    CLIENT_DIR="client/$DB"

    mkdir -p "$CA_DIR" "$SERVER_DIR" "$CLIENT_DIR"

    # Generate CA key and certificate
    echo "Creating CA certificate..."
    openssl genrsa -out "$CA_DIR/ca-key.pem" 2048 2>/dev/null
    openssl req -new -x509 -nodes -days 3650 \
        -key "$CA_DIR/ca-key.pem" \
        -out "$CA_DIR/ca.pem" \
        -subj "/C=US/ST=Test/L=Test/O=WhoDB-Test/CN=${DB}_CA"

    # Generate server key and certificate
    echo "Creating server certificate..."
    openssl genrsa -out "$SERVER_DIR/server-key.pem" 2048 2>/dev/null

    # Create server cert config with SANs
    cat > "$SERVER_DIR/server.cnf" << EOF
[req]
distinguished_name = req_distinguished_name
x509_extensions = v3_req
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = US
ST = Test
L = Test
O = WhoDB-Test
CN = $DB

[v3_req]
basicConstraints = CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
DNS.2 = $DB
DNS.3 = e2e_${DB}_ssl
DNS.4 = *.localhost
IP.1 = 127.0.0.1
$EXTRA_SANS
EOF

    openssl req -new -key "$SERVER_DIR/server-key.pem" \
        -out "$SERVER_DIR/server-req.pem" \
        -config "$SERVER_DIR/server.cnf"

    openssl x509 -req -in "$SERVER_DIR/server-req.pem" \
        -days 3650 \
        -CA "$CA_DIR/ca.pem" \
        -CAkey "$CA_DIR/ca-key.pem" \
        -CAcreateserial \
        -out "$SERVER_DIR/server-cert.pem" \
        -extfile "$SERVER_DIR/server.cnf" \
        -extensions v3_req

    # Generate client key and certificate
    echo "Creating client certificate..."
    openssl genrsa -out "$CLIENT_DIR/client-key.pem" 2048 2>/dev/null
    openssl req -new -key "$CLIENT_DIR/client-key.pem" \
        -out "$CLIENT_DIR/client-req.pem" \
        -subj "/C=US/ST=Test/L=Test/O=WhoDB-Test/CN=${DB}_client"

    openssl x509 -req -in "$CLIENT_DIR/client-req.pem" \
        -days 3650 \
        -CA "$CA_DIR/ca.pem" \
        -CAkey "$CA_DIR/ca-key.pem" \
        -CAcreateserial \
        -out "$CLIENT_DIR/client-cert.pem"

    # Set permissions (readable by containers)
    chmod 644 "$CA_DIR/ca.pem"
    chmod 644 "$SERVER_DIR/server-cert.pem"
    chmod 644 "$SERVER_DIR/server-key.pem"
    chmod 644 "$CLIENT_DIR/client-cert.pem"
    chmod 644 "$CLIENT_DIR/client-key.pem"

    echo "Done with $DB certificates"
}

# ============================================
# CE Database Certificates
# ============================================

# PostgreSQL
generate_standard_certs "postgres"

# MySQL
generate_standard_certs "mysql"

# MariaDB (same format as MySQL)
generate_standard_certs "mariadb"

# MongoDB - needs combined cert+key PEM file
generate_standard_certs "mongodb" "DNS.5 = e2e_mongo_ssl"
echo "Creating combined MongoDB PEM file..."
cat "server/mongodb/server-cert.pem" "server/mongodb/server-key.pem" > "server/mongodb/mongodb.pem"
chmod 644 "server/mongodb/mongodb.pem"

# Redis - standard certs work
generate_standard_certs "redis" "DNS.5 = e2e_redis_ssl"
# Redis also needs a combined file sometimes
cat "server/redis/server-cert.pem" "server/redis/server-key.pem" > "server/redis/redis.pem"
chmod 644 "server/redis/redis.pem"

# Elasticsearch - standard certs with xpack format
generate_standard_certs "elasticsearch" "DNS.5 = e2e_elasticsearch_ssl"

# ClickHouse - standard certs
generate_standard_certs "clickhouse" "DNS.5 = e2e_clickhouse_ssl"

# Create marker file
echo "These certificates are for E2E testing only. DO NOT USE IN PRODUCTION." > DO_NOT_USE_IN_PRODUCTION

echo ""
echo "=== Certificate generation complete ==="
echo "Certificates created in: $SCRIPT_DIR"
echo ""
echo "Generated certificates for:"
echo "  - postgres"
echo "  - mysql"
echo "  - mariadb"
echo "  - mongodb"
echo "  - redis"
echo "  - elasticsearch"
echo "  - clickhouse"
