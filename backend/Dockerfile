FROM golang:1.26-alpine AS base

WORKDIR /app
RUN apk add --no-cache ca-certificates git tzdata

FROM base AS dev
RUN go install github.com/air-verse/air@latest
COPY go.mod ./
RUN go mod download
COPY . .
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

FROM base AS build
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api

FROM alpine:3.23 AS runtime
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /out/api /app/api
EXPOSE 8080
CMD ["/app/api"]
