# 构建阶段
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY servers/kali/go.mod servers/kali/go.sum ./
RUN go mod download
COPY servers/kali/ ./
COPY pkg/ ../pkg/
RUN CGO_ENABLED=0 go build -o /kali-mcp ./cmd/server

# 运行阶段
FROM kalilinux/kali-rolling
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    nmap sqlmap metasploit-framework gobuster hydra \
    nikto dirb enum4linux smbclient \
    impacket-scripts responder wireshark-common \
    john hashcat crunch \
    curl wget netcat-openbsd dnsutils \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /kali-mcp /usr/local/bin/kali-mcp
EXPOSE 8080
ENTRYPOINT ["kali-mcp"]
