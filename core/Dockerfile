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
# Skip Cypress binary installation - not needed for production builds
ENV CYPRESS_INSTALL_BINARY=0
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

FROM golang:1.25.4-bookworm AS backend-stage
RUN apt-get update && apt-get install -y --no-install-recommends \
	gcc \
	libc6-dev \
	git \
	ca-certificates \
	curl \
	&& rm -rf /var/lib/apt/lists/*
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
	-ldflags="-X github.com/clidey/whodb/core/src/env.ApplicationVersion=${VERSION} -X github.com/clidey/whodb/core/src/env.ApplicationEnvironment=${PLATFORM}-${TARGETARCH}" \
	-o /core

# Pre-download BAML native library for the target architecture
# BAML version should match the version in go.mod (0.215.2)
ARG TARGETARCH
RUN BAML_VERSION=0.215.2 && \
    if [ "$TARGETARCH" = "amd64" ]; then \
        BAML_ARCH="x86_64"; \
    elif [ "$TARGETARCH" = "arm64" ]; then \
        BAML_ARCH="aarch64"; \
    else \
        echo "Unsupported architecture: $TARGETARCH" && exit 1; \
    fi && \
    mkdir -p /root/.cache/baml/libs/${BAML_VERSION} && \
    curl -L -o /root/.cache/baml/libs/${BAML_VERSION}/libbaml_cffi-${BAML_ARCH}-unknown-linux-gnu.so \
        https://github.com/boundaryml/baml/releases/download/${BAML_VERSION}/libbaml_cffi-${BAML_ARCH}-unknown-linux-gnu.so && \
    echo "Downloaded BAML library for ${BAML_ARCH}"

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
	ca-certificates \
	&& rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend-stage /core /core
# Copy pre-downloaded BAML library cache
COPY --from=backend-stage /root/.cache/baml /root/.cache/baml

CMD ["/core"]