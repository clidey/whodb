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

FROM node:25.6.0-alpine3.23 AS build-stage
RUN npm i -g pnpm@10.29.3
WORKDIR /app
COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./
# Skip test framework browser downloads - not needed for production builds
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
RUN pnpm install
COPY ./frontend/ ./

ARG VERSION=dev
ENV VITE_APP_VERSION=${VERSION}
RUN pnpm run build
ARG TARGETARCH
ARG PLATFORM=docker

FROM golang:1.26-alpine3.23 AS backend-stage
RUN apk add --no-cache \
	gcc \
	g++ \
	musl-dev \
	git \
	ca-certificates \
	curl

# DuckDB's pre-built static library references glibc-specific symbols unavailable in musl.
# Build a static compat library so the linker deduplicates across CGO packages.
RUN printf '#include <stddef.h>\nint __res_init(void) { return 0; }\nint backtrace(void **b, int s) { return 0; }\nchar **backtrace_symbols(void *const *b, int s) { return NULL; }\nvoid backtrace_symbols_fd(void *const *b, int s, int fd) {}\nint malloc_trim(size_t pad) { return 0; }\n' \
    | gcc -c -x c - -o /tmp/duckdb_musl_compat.o \
    && ar rcs /usr/lib/libduckdb_musl_compat.a /tmp/duckdb_musl_compat.o

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
	CGO_LDFLAGS="-Wl,--gc-sections -Wl,--as-needed -flto -lduckdb_musl_compat" \
	CGO_ENABLED=1 GOOS=linux GOARCH=${TARGETARCH} go build -tags prod -trimpath \
	-ldflags="-s -w -X github.com/clidey/whodb/core/src/env.ApplicationVersion=${VERSION} -X github.com/clidey/whodb/core/src/env.ApplicationEnvironment=${PLATFORM}-${TARGETARCH}" \
	-o /core ./cmd/whodb

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
    echo "Downloaded BAML musl library for ${BAML_ARCH}"

FROM alpine:3.23
RUN apk add --no-cache ca-certificates libgcc libstdc++
WORKDIR /app
COPY --from=backend-stage /core /core
# Copy BAML musl library and set path via env var
# (BAML Go client's isMusl() is not implemented, so we must use BAML_LIBRARY_PATH)
COPY --from=backend-stage /baml-lib/libbaml_cffi.so /usr/local/lib/libbaml_cffi.so
ENV BAML_LIBRARY_PATH=/usr/local/lib/libbaml_cffi.so

CMD ["/core"]
