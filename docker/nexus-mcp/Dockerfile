FROM docker.1ms.run/library/golang:1.26-alpine AS builder
WORKDIR /build
COPY pkg/mcputil/ pkg/mcputil/
COPY servers/nexus/ servers/nexus/
WORKDIR /build/servers/nexus
RUN GOWORK=off CGO_ENABLED=0 go build -o /nexus-mcp ./cmd/server

FROM docker.1ms.run/library/alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /nexus-mcp /usr/local/bin/nexus-mcp
RUN mkdir -p /data
EXPOSE 8081
ENTRYPOINT ["nexus-mcp", "-db", "/data/nexus.db"]
