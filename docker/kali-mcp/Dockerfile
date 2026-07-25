# Build stage — uses direct pull (kalilinux not available on mirror)
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

# Copy pre-built binary (build on a machine with Go, then scp the binary)
COPY kali-mcp /usr/local/bin/kali-mcp
EXPOSE 8080
ENTRYPOINT ["kali-mcp"]
