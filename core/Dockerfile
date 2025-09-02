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

FROM node:lts-alpine AS build-stage
RUN npm i -g pnpm
WORKDIR /app
COPY ./frontend/package.json ./frontend/pnpm-lock.yaml ./
RUN pnpm install
COPY ./frontend/ ./

# Create empty EE modules for Vite to resolve during CE build
RUN mkdir -p ee/frontend/src/components/theme ee/frontend/src/components/charts ee/frontend/src/pages/raw-execute ee/frontend/src && \
    echo "export default {}; export const LineChart = () => null;" > ee/frontend/src/components/charts/line-chart.tsx && \
    echo "export default {}; export const PieChart = () => null;" > ee/frontend/src/components/charts/pie-chart.tsx && \
    echo "export default {}; export const AnalyzeGraph = () => null;" > ee/frontend/src/pages/raw-execute/analyze-view.tsx && \
    echo "export default {};" > ee/frontend/src/index.ts && \
    echo "export default {}; export const eeDatabaseTypes = []; export const eeFeatures = {};" > ee/frontend/src/config.ts && \
    echo "export default {}; export const EEIcons = { Logos: {} };" > ee/frontend/src/icons.tsx

RUN pnpm run build

FROM golang:1.24.1-alpine AS backend-stage
RUN apk update && apk add --no-cache gcc musl-dev git
WORKDIR /app
COPY ./core/go.mod ./core/go.sum ./
RUN go mod download
COPY ./core/ ./

COPY ./scripts/ ../scripts/

COPY --from=build-stage /app/build/ ./build/

# Build backend
RUN CGO_ENABLED=1 GOOS=linux go build -o /core

FROM alpine:3.22.1
RUN apk update && apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=backend-stage /core /core

CMD ["/core"]