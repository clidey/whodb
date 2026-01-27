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

# WhoDB CLI Dockerfile

FROM golang:1.25-alpine3.22 AS builder

ARG VERSION=dev

RUN apk update && apk add --no-cache gcc musl-dev git bash curl

WORKDIR /app

# Copy ee-stub for CE builds (provides stub packages for EE imports)
COPY ./ee-stub/ ./ee-stub/

# Copy core module
COPY core/go.mod core/go.sum ./core/
WORKDIR /app/core
RUN go mod download

# Copy cli module
WORKDIR /app
COPY cli/go.mod cli/go.sum ./cli/
WORKDIR /app/cli
RUN go mod download

# Copy all source code
WORKDIR /app
COPY core/ ./core/
COPY cli/ ./cli/

# Build CLI binary with version injection
WORKDIR /app/cli
ARG TARGETARCH
RUN CGO_ENABLED=1 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags "-s -w -X github.com/clidey/whodb/cli/pkg/version.Version=${VERSION}" \
    -o /whodb-cli .

# Pre-download BAML native library for the target architecture (musl for Alpine)
# Extract BAML version from go.mod to stay in sync
RUN BAML_VERSION=$(grep 'github.com/boundaryml/baml ' /app/core/go.mod | awk '{print $2}' | sed 's/^v//') && \
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

# Final stage
FROM alpine:3.22

RUN apk update && apk add --no-cache ca-certificates libgcc

WORKDIR /app

COPY --from=builder /whodb-cli /usr/local/bin/whodb-cli

# Copy BAML musl library and set path via env var
COPY --from=builder /baml-lib/libbaml_cffi.so /usr/local/lib/libbaml_cffi.so
ENV BAML_LIBRARY_PATH=/usr/local/lib/libbaml_cffi.so

# Create config directory
RUN mkdir -p /root/.whodb-cli

ENTRYPOINT ["whodb-cli"]
CMD ["--help"]
