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

FROM node:18.12.1-alpine AS build-stage
RUN npm i -g pnpm
WORKDIR /app
COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./
RUN pnpm install
COPY ./frontend/ ./
RUN pnpm run build

FROM golang:1.22.5-alpine3.19 AS backend-stage
RUN apk update && apk add --no-cache gcc musl-dev
WORKDIR /app
COPY ./core/go.mod ./core/go.sum ./
RUN go mod download
COPY ./core/ ./
COPY --from=build-stage /app/build/ ./build/
RUN CGO_ENABLED=1 GOOS=linux go build -o /core

FROM alpine:3.19
RUN apk update && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-stage /core /core

CMD ["/core"]