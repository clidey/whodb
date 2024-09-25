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