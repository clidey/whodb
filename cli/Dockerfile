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

FROM golang:1.24-alpine AS builder

RUN apk update && apk add --no-cache gcc musl-dev git bash

WORKDIR /app

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

# Build CLI binary
WORKDIR /app/cli
RUN CGO_ENABLED=1 GOOS=linux go build -o /whodb-cli .

# Final stage
FROM alpine:3.21

RUN apk update && apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /whodb-cli /usr/local/bin/whodb-cli

# Create config directory
RUN mkdir -p /root/.whodb-cli

ENTRYPOINT ["whodb-cli"]
CMD ["--help"]
