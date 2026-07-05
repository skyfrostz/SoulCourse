# syntax=docker/dockerfile:1.7

FROM golang:1.26-alpine AS base

WORKDIR /app
ENV GOPROXY=https://goproxy.cn,direct \
    GOSUMDB=sum.golang.google.cn
RUN apk add --no-cache ca-certificates git tzdata

FROM base AS deps
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

FROM deps AS dev
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/air-verse/air@latest
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

FROM deps AS build
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.23 AS runtime
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/api /app/api
EXPOSE 8080
CMD ["/app/api"]
