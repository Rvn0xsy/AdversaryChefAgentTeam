FROM docker.1ms.run/library/golang:1.26-alpine AS builder
WORKDIR /build
COPY pkg/mcputil/ pkg/mcputil/
COPY cmd/acasched/ cmd/acasched/
WORKDIR /build/cmd/acasched
RUN GOWORK=off CGO_ENABLED=0 go build -o /acasched .

FROM docker.1ms.run/library/alpine:3.21
RUN apk add --no-cache ca-certificates curl
COPY --from=builder /acasched /usr/local/bin/acasched
RUN mkdir -p /data
EXPOSE 9090
ENTRYPOINT ["acasched"]
