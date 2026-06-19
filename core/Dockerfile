# Copyright 2025 Clidey, Inc.
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

FROM node:26-alpine3.24 AS build-stage
RUN npm i -g pnpm@10.29.3
WORKDIR /app
COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./
ARG USE_PREBUILT_FRONTEND=false
# Skip test framework browser downloads - not needed for production builds
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
RUN if [ "${USE_PREBUILT_FRONTEND}" != "true" ]; then pnpm install; fi
COPY ./frontend/ ./

ARG VERSION=dev
ENV VITE_APP_VERSION=${VERSION}
ENV WHODB_BUILD_SOURCEMAP=false
RUN if [ "${USE_PREBUILT_FRONTEND}" = "true" ]; then \
        test -d build && test -f build/index.html && test -f build/.whodb-edition && grep -qx 'ce' build/.whodb-edition; \
    else \
        pnpm run build:ce; \
    fi && \
    test -f build/.whodb-edition && grep -qx 'ce' build/.whodb-edition && \
    rm -f build/.whodb-edition
ARG TARGETARCH
ARG PLATFORM=docker

FROM golang:1.26.4-alpine3.24 AS backend-stage
SHELL ["/bin/ash", "-eo", "pipefail", "-c"]
RUN apk add --no-cache \
	gcc \
	g++ \
	musl-dev \
	git \
	ca-certificates \
	curl \
	binutils \
	upx

# Download official musl-compiled DuckDB shared library.
# Native musl build — no glibc stubs or compatibility hacks needed.
# Extract DuckDB version from go.mod: duckdb-go/v2 v2.XYYYZ.0 → vX.YY.Z
COPY ./core/go.mod /tmp/go.mod
ARG TARGETARCH
RUN ENCODED=$(grep 'github.com/duckdb/duckdb-go/v2 ' /tmp/go.mod | awk '{print $2}' | sed 's/v2\.//;s/\.[0-9]*$//') && \
    MAJOR=$((ENCODED / 10000)) && MINOR=$(( (ENCODED % 10000) / 100 )) && PATCH=$((ENCODED % 100)) && \
    DUCKDB_VERSION="v${MAJOR}.${MINOR}.${PATCH}" && \
    echo "DuckDB version: $DUCKDB_VERSION" && \
    if [ "$TARGETARCH" = "amd64" ]; then DUCKDB_ARCH="linux-amd64"; \
    elif [ "$TARGETARCH" = "arm64" ]; then DUCKDB_ARCH="linux-arm64"; \
    else echo "Unsupported architecture: $TARGETARCH" && exit 1; fi && \
    mkdir -p /usr/local/include && \
    curl -L -o /tmp/libduckdb-musl.zip \
        "https://github.com/duckdb/duckdb/releases/download/${DUCKDB_VERSION}/libduckdb-${DUCKDB_ARCH}-musl.zip" && \
    unzip -j /tmp/libduckdb-musl.zip libduckdb.so -d /usr/local/lib/ && \
    unzip -j /tmp/libduckdb-musl.zip duckdb.h -d /usr/local/include/ && \
    strip --strip-unneeded /usr/local/lib/libduckdb.so && \
    rm /tmp/libduckdb-musl.zip

WORKDIR /app

# Copy core module files
COPY ./core/go.mod ./core/go.sum ./
RUN go mod download
COPY ./core/ ./

COPY --from=build-stage /app/build/ ./cmd/whodb/build/

# Build CE backend from the new entry point
ARG VERSION
ARG TARGETARCH
ARG PLATFORM
RUN CGO_CFLAGS="-Os -ffunction-sections -fdata-sections -fno-asynchronous-unwind-tables -fno-unwind-tables -flto -fvisibility=hidden" \
	CGO_LDFLAGS="-Wl,--gc-sections -Wl,--as-needed -flto -L/usr/local/lib -lduckdb" \
	CGO_ENABLED=1 GOOS=linux GOARCH=${TARGETARCH} go build -tags "prod duckdb_use_lib" -trimpath \
	-ldflags="-s -w -X github.com/clidey/whodb/core/src/env.ApplicationVersion=${VERSION} -X github.com/clidey/whodb/core/src/env.ApplicationEnvironment=${PLATFORM}-${TARGETARCH}" \
	-o /core ./cmd/whodb

RUN chmod +x /core && \
    upx --best /core

RUN mkdir -p /runtime-libs && \
    chmod +x /usr/local/lib/libduckdb.so && \
    cp /usr/local/lib/libduckdb.so /runtime-libs/

# Pre-download BAML native library for the target architecture (musl for Alpine)
# Extract BAML version from go.mod to stay in sync
ARG TARGETARCH
RUN BAML_VERSION=$(grep 'github.com/boundaryml/baml ' go.mod | awk '{print $2}' | sed 's/^v//') && \
    if [ "$TARGETARCH" = "amd64" ]; then \
        BAML_ARCH="x86_64"; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
        BAML_ARCH="aarch64"; \
    else \
        echo "Unsupported architecture: $TARGETARCH" && exit 1; \
    fi && \
    mkdir -p /baml-lib && \
    curl -L -o /baml-lib/libbaml_cffi.so \
        https://github.com/boundaryml/baml/releases/download/${BAML_VERSION}/libbaml_cffi-${BAML_ARCH}-unknown-linux-musl.so && \
    chmod +x /baml-lib/libbaml_cffi.so && \
    strip --strip-debug /baml-lib/libbaml_cffi.so && \
    cp /baml-lib/libbaml_cffi.so /runtime-libs/ && \
    echo "Downloaded BAML musl library for ${BAML_ARCH}"

FROM alpine:3.24
RUN apk add --no-cache ca-certificates libgcc libstdc++
WORKDIR /app
COPY --from=backend-stage /core /core
COPY --from=backend-stage /runtime-libs/ /usr/local/lib/
# Copy runtime native libraries and set BAML path via env var.
# (BAML Go client's isMusl() is not implemented, so we must use BAML_LIBRARY_PATH)
ENV BAML_LIBRARY_PATH=/usr/local/lib/libbaml_cffi.so

CMD ["/core"]
