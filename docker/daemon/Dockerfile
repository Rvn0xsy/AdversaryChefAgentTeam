FROM docker.1ms.run/ubuntu:24.04

# Install Multica CLI
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl nodejs npm \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/* \
    && npm install -g multica

# Copy Codex binary and cc-switch
COPY codex /usr/local/bin/codex
COPY cc-switch /usr/local/bin/cc-switch

# Copy config templates
COPY config.toml /root/.codex/config.toml

# Copy workspace skills
COPY skills/ /root/.multica/skills/

RUN mkdir -p /workspace
WORKDIR /workspace

ENTRYPOINT ["multica", "daemon", "start", "--foreground"]
