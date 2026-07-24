FROM ubuntu:24.04

# 基础依赖
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl git nodejs npm \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

# 安装 Codex CLI
# 注意：需要替换为实际的 Codex Linux 安装命令
ARG CODEX_VERSION=latest
RUN curl -fsSL https://get.codex.openai.com | sh -s -- --version ${CODEX_VERSION}

# 安装 Multica Daemon
# 注意：需要替换为实际的 Multica daemon 安装命令
RUN npm install -g multica

# 持久化目录
RUN mkdir -p /root/.multica /root/.codex /workspace

# Codex cc-switch 配置模板
COPY config/codex/config.toml /root/.codex/config.toml
COPY config/codex/cc-switch-model-catalog.json /root/.codex/cc-switch-model-catalog.json

WORKDIR /workspace
ENTRYPOINT ["multica", "daemon"]
