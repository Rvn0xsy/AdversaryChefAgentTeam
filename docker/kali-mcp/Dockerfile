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
    curl wget unzip netcat-openbsd dnsutils \
    whois \
    naabu \
    nuclei \
    ffuf \
    httpx-toolkit \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# httpx-toolkit binary is named differently; create symlink for playbook compatibility
RUN ln -s /usr/bin/httpx-toolkit /usr/local/bin/httpx

# katana is not packaged in Kali; install from GitHub releases
RUN KATANA_VER=$(curl -s https://api.github.com/repos/projectdiscovery/katana/releases/latest | grep tag_name | cut -d\" -f4) \
    && curl -sLo /tmp/katana.zip https://github.com/projectdiscovery/katana/releases/download/${KATANA_VER}/katana_${KATANA_VER#v}_linux_amd64.zip \
    && cd /usr/local/bin && unzip -qo /tmp/katana.zip katana && rm /tmp/katana.zip \
    && chmod +x /usr/local/bin/katana
COPY --from=builder /kali-mcp /usr/local/bin/kali-mcp
EXPOSE 8080
ENTRYPOINT ["kali-mcp"]

# Copy dictionaries (bundled in image)
COPY data/dictionaries/ /data/dictionaries/

# Copy nuclei-templates (must exist — run update-nuclei-templates.sh before building)
COPY data/nuclei-templates/ /root/nuclei-templates/
