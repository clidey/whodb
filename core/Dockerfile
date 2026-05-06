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
COPY ./dataflow/package.json ./dataflow/pnpm-lock.yaml ./
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
RUN pnpm install
COPY ./dataflow/ ./

ARG VERSION=dev
ENV VITE_APP_VERSION=${VERSION}
RUN pnpm run build

FROM golang:1.26-alpine3.23 AS backend-stage
RUN apk add --no-cache \
	gcc \
	musl-dev \
	git \
	ca-certificates \
	curl
WORKDIR /app

# Copy core module files
COPY ./core/go.mod ./core/go.sum ./
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
RUN go mod download
COPY ./core/ ./


COPY --from=build-stage /app/build/ ./build/

# Build backend with version and environment
ARG VERSION
ARG TARGETARCH
ARG PLATFORM
RUN if [ -z "${TARGETARCH}" ] || [ -z "${PLATFORM}" ]; then echo "TARGETARCH and PLATFORM build args are required" >&2; exit 1; fi && \
	CGO_CFLAGS="-Os -ffunction-sections -fdata-sections -fno-asynchronous-unwind-tables -fno-unwind-tables -flto -fvisibility=hidden" \
	CGO_LDFLAGS="-Wl,--gc-sections -Wl,--as-needed -flto" \
	CGO_ENABLED=1 GOOS=linux GOARCH=${TARGETARCH} go build -tags prod -trimpath \
	-ldflags="-s -w -X github.com/clidey/whodb/core/src/env.ApplicationVersion=${VERSION} -X github.com/clidey/whodb/core/src/env.ApplicationEnvironment=${PLATFORM}-${TARGETARCH}" \
	-o /core

# Pre-download BAML native library for the target architecture (musl for Alpine)
# Extract BAML version from go.mod to stay in sync
ARG TARGETARCH
ARG BAML_RELEASE_BASE_URL=https://github.com/boundaryml/baml/releases/download
RUN if [ -z "$TARGETARCH" ]; then echo "TARGETARCH build arg is required" >&2; exit 1; fi && \
    BAML_VERSION=$(grep 'github.com/boundaryml/baml ' go.mod | awk '{print $2}' | sed 's/^v//') && \
    if [ "$TARGETARCH" = "amd64" ]; then \
        BAML_ARCH="x86_64"; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
        BAML_ARCH="aarch64"; \
    else \
        echo "Unsupported architecture: $TARGETARCH" && exit 1; \
    fi && \
    mkdir -p /baml-lib && \
    curl --fail --location --retry 5 --retry-delay 3 --retry-all-errors -o /baml-lib/libbaml_cffi.so \
        "${BAML_RELEASE_BASE_URL}/${BAML_VERSION}/libbaml_cffi-${BAML_ARCH}-unknown-linux-musl.so" && \
    echo "Downloaded BAML musl library for ${BAML_ARCH}"

FROM alpine:3.23
RUN apk add --no-cache ca-certificates libgcc
WORKDIR /app
COPY --from=backend-stage /core /core
# Copy BAML musl library and set path via env var
# (BAML Go client's isMusl() is not implemented, so we must use BAML_LIBRARY_PATH)
COPY --from=backend-stage /baml-lib/libbaml_cffi.so /usr/local/lib/libbaml_cffi.so
ENV BAML_LIBRARY_PATH=/usr/local/lib/libbaml_cffi.so

CMD ["/core"]
