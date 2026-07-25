# Build stage
FROM docker.1ms.run/library/golang:1.26-alpine AS builder
WORKDIR /build
COPY servers/asset/go.mod servers/asset/go.sum ./
COPY pkg/ ../pkg/
RUN go mod download
COPY servers/asset/ ./
RUN CGO_ENABLED=0 go build -o /asset-mcp ./cmd/server

# Run stage
FROM docker.1ms.run/library/alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /asset-mcp /usr/local/bin/asset-mcp
RUN mkdir -p /data
EXPOSE 8081
ENTRYPOINT ["asset-mcp", "-db", "/data/asset.db"]
