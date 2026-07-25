# Build stage
FROM docker.1ms.run/library/golang:1.26-alpine AS builder
WORKDIR /build
COPY cmd/acasched/go.mod cmd/acasched/go.sum ./
COPY pkg/ ../pkg/
RUN go mod download
COPY cmd/acasched/ ./
RUN CGO_ENABLED=0 go build -o /acasched .

# Run stage
FROM docker.1ms.run/library/alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /acasched /usr/local/bin/acasched
RUN mkdir -p /data
EXPOSE 9090
ENTRYPOINT ["acasched"]
