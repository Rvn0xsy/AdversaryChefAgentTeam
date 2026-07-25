# Build stage
FROM docker.1ms.run/library/golang:1.26-alpine AS builder
WORKDIR /build
COPY servers/kali/go.mod servers/kali/go.sum ./
COPY pkg/ ../pkg/
RUN go mod download
COPY servers/kali/ ./
RUN CGO_ENABLED=0 go build -o /kali-mcp ./cmd/server

# Run stage — minimal Kali with essential tools
FROM docker.1ms.run/kalilinux/kali-rolling
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    nmap \
    sqlmap \
    gobuster \
    curl wget netcat-openbsd dnsutils \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*
COPY --from=builder /kali-mcp /usr/local/bin/kali-mcp
EXPOSE 8080
ENTRYPOINT ["kali-mcp"]

# Copy dictionaries (bundled in image)
COPY data/dictionaries/ /data/dictionaries/

# Copy nuclei-templates (must exist — run update-nuclei-templates.sh before building)
COPY data/nuclei-templates/ /root/nuclei-templates/
