FROM node:18.12.1-alpine as build-stage

RUN npm i -g pnpm

WORKDIR /app
COPY ./frontend/package.json .
COPY ./frontend/pnpm-lock.yaml .
RUN pnpm i
COPY ./frontend/src ./src/
COPY ./frontend/public ./public/
COPY ./frontend/tailwind.config.js ./tailwind.config.js
COPY ./frontend/tsconfig.json ./tsconfig.json
COPY ./frontend/config-overrides.js ./config-overrides.js
RUN pnpm run build

FROM golang:1.22.1-bullseye

WORKDIR /app
COPY ./core/go.mod /app
COPY ./core/go.sum /app
RUN go mod download

COPY ./core/server.go /app
COPY ./core/gqlgen.yml /app/
COPY ./core/src/ /app/src/
COPY ./core/graph/ /app/graph/
COPY ./core/tools/ /app/tools/
COPY --from=build-stage /app/build/ /app/build/

RUN CGO_ENABLED=0 GOOS=linux go build -o /core
CMD ["/core"]