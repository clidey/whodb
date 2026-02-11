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

FROM node:24.11.1-alpine3.22 AS build-stage
RUN npm i -g pnpm
WORKDIR /app
COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./
# Skip test framework browser downloads - not needed for production builds
ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
RUN pnpm install
COPY ./frontend/ ./

# Create empty EE modules for Vite to resolve during CE build
RUN mkdir -p ee/frontend/src/components/theme ee/frontend/src/components/charts ee/frontend/src/pages/raw-execute ee/frontend/src && \
    echo "export default {}; export const LineChart = () => null;" > ee/frontend/src/components/charts/line-chart.tsx && \
    echo "export default {}; export const PieChart = () => null;" > ee/frontend/src/components/charts/pie-chart.tsx && \
    echo "export default {}; export const AnalyzeGraph = () => null;" > ee/frontend/src/pages/raw-execute/analyze-view.tsx && \
    echo "export default {};" > ee/frontend/src/index.ts && \
    echo "export default {}; export const eeDatabaseTypes = []; export const eeFeatures = {};" > ee/frontend/src/config.ts && \
    echo "export default {};" > ee/frontend/src/components/export.ts && \
    echo "export default {}; export const EEIcons = { Logos: {} };" > ee/frontend/src/icons.tsx

RUN pnpm run build

ARG VERSION=dev
ARG TARGETARCH
ARG PLATFORM=docker

FROM golang:1.25-alpine3.22 AS backend-stage
RUN apk add --no-cache \
	gcc \
	musl-dev \
	git \
	ca-certificates \
	curl
WORKDIR /app

# Copy ee-stub for CE builds (provides stub packages for EE imports)
COPY ./ee-stub/ ./ee-stub/

# Copy core module files and fix replace path for Docker context
COPY ./core/go.mod ./core/go.sum ./
RUN sed -i 's|../ee-stub|./ee-stub|g' go.mod
RUN go mod download
COPY ./core/ ./


COPY --from=build-stage /app/build/ ./build/

# Build backend with version and environment
ARG VERSION
ARG TARGETARCH
ARG PLATFORM
RUN CGO_ENABLED=1 GOOS=linux GOARCH=${TARGETARCH} go build -tags prod \
	-ldflags="-s -w -X github.com/clidey/whodb/core/src/env.ApplicationVersion=${VERSION} -X github.com/clidey/whodb/core/src/env.ApplicationEnvironment=${PLATFORM}-${TARGETARCH}" \
	-o /core

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

FROM alpine:3.22
RUN apk add --no-cache ca-certificates libgcc
WORKDIR /app
COPY --from=backend-stage /core /core
# Copy BAML musl library and set path via env var
# (BAML Go client's isMusl() is not implemented, so we must use BAML_LIBRARY_PATH)
COPY --from=backend-stage /baml-lib/libbaml_cffi.so /usr/local/lib/libbaml_cffi.so
ENV BAML_LIBRARY_PATH=/usr/local/lib/libbaml_cffi.so

CMD ["/core"]