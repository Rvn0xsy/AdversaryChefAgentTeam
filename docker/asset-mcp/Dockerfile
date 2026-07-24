# 构建阶段
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY servers/asset/go.mod servers/asset/go.sum ./
RUN go mod download
COPY servers/asset/ ./
COPY pkg/ ../pkg/
RUN CGO_ENABLED=0 go build -o /asset-mcp ./cmd/server

# 运行阶段
FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /asset-mcp /usr/local/bin/asset-mcp
RUN mkdir -p /data
EXPOSE 8081
ENTRYPOINT ["asset-mcp", "-db", "/data/asset.db"]
