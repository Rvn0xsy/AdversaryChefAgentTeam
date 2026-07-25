# Build stage
FROM docker.1ms.run/library/golang:1.26-alpine AS builder
WORKDIR /build
COPY servers/nexus/go.mod servers/nexus/go.sum ./
COPY pkg/ ../pkg/
COPY servers/nexus/ ./
RUN CGO_ENABLED=0 go build -o /nexus-mcp ./cmd/server

# Run stage
FROM docker.1ms.run/library/alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /nexus-mcp /usr/local/bin/nexus-mcp
RUN mkdir -p /data
EXPOSE 8081
ENTRYPOINT ["nexus-mcp", "-db", "/data/nexus.db"]
