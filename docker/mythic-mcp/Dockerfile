# Build stage
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY servers/mythic/go.mod servers/mythic/go.sum ./
RUN go mod download
COPY servers/mythic/ ./
COPY pkg/ ../pkg/
RUN CGO_ENABLED=0 go build -o /mythic-mcp ./cmd/server

# Run stage
FROM alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /mythic-mcp /usr/local/bin/mythic-mcp
EXPOSE 8082
ENTRYPOINT ["mythic-mcp"]
