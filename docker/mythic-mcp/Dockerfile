FROM docker.1ms.run/library/golang:1.26-alpine AS builder
WORKDIR /build
COPY pkg/mcputil/ pkg/mcputil/
COPY servers/mythic/ servers/mythic/
WORKDIR /build/servers/mythic
RUN GOWORK=off CGO_ENABLED=0 go build -o /mythic-mcp ./cmd/server

FROM docker.1ms.run/library/alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /mythic-mcp /usr/local/bin/mythic-mcp
EXPOSE 8082
ENTRYPOINT ["mythic-mcp"]
