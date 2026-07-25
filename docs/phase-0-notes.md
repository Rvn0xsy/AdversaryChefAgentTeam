# Phase 0 实验记录 & 踩坑手册

> 2026-07-24 · home-dev (192.168.100.37) · Multica + Codex + asset-mcp

## 环境

| 组件 | 位置 | 角色 |
|------|------|------|
| Multica 服务端 | home-ubuntu (192.168.100.33) | 自部署，前端 :3001，后端 :8080 |
| Multica daemon | home-dev, `dev` 用户 | 原生运行，非容器化 |
| Codex 0.145.0 | home-dev `/usr/local/bin/codex` | AI runtime |
| cc-switch 5.9.3 | home-dev `/usr/local/bin/cc-switch` | 模型路由 → DeepSeek V4 Pro |
| asset-mcp | home-dev Docker (:8081) | 唯一容器，Alpine 32MB |
| Gitea | home-ubuntu :3000/:2222 | Git 仓库 |

---

## 1. MCP 传输协议：SSE vs Streamable HTTP（最大坑）

### 问题

Go MCP SDK v1.6.1 提供两种 HTTP 传输：

| 传输 | 协议版本 | Session 管理 | Codex 支持 |
|------|---------|-------------|:---:|
| `NewSSEHandler` | 2024-11-05 | `?sessionid=` query param | ❌ |
| `NewStreamableHTTPHandler` | 2025-03-26 | `Mcp-Session-Id` header | ✅ |

**Codex 使用 Streamable HTTP（2025 新协议）**。如果服务端用旧的 SSE handler，Codex POST 不带 `?sessionid=` query param，服务端返回 `HTTP 400: sessionid must be provided`。

### 修复

`pkg/mcputil/mcputil.go` 中 `Run()` 必须用：

```go
handler := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
    return server
}, &mcp.StreamableHTTPOptions{
    SessionTimeout: 5 * time.Minute,
})
```

验证初始化握手：
```bash
curl -s -X POST http://localhost:8081/ \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}},"id":1}'
# 应返回 protocolVersion: "2025-03-26"
```

### Multica Agent MCP 配置格式

```json
{
  "mcpServers": {
    "asset": {
      "type": "http",
      "url": "http://127.0.0.1:8081"
    }
  }
}
```

注意 `type` 是 `"http"`（不是 `"sse"`），`url` 是根路径（不带 `/sse`）。

---

## 2. Per-task CODEX_HOME 兼容性（设计文档里最担心的点）

### 结论：兼容 ✅

Multica daemon 为每个任务创建 per-task `$CODEX_HOME`，但**会保留用户原有的 `[model_providers]` 配置**，只在 config.toml 中追加 `[mcp_servers]` 块。

验证方式：
- 在 `~/.codex/config.toml` 中配置 cc-switch 模型路由
- 在 Multica Agent 的 `--mcp-config` 中配置 MCP server
- 发起任务 → Agent 能同时调用模型（DeepSeek）和 MCP 工具（asset-mcp） ✅

---

## 3. Docker 镜像加速

### `docker.1ms.run` 路径格式

官方 Docker Hub 镜像必须加 `/library/` 前缀：

```dockerfile
# ✅ 正确
FROM docker.1ms.run/library/golang:1.26-alpine
FROM docker.1ms.run/library/alpine:3.21
FROM docker.1ms.run/library/ubuntu:24.04

# ❌ 错误
FROM docker.1ms.run/golang:1.26-alpine
```

非官方镜像直接用 `docker.1ms.run/<namespace>/<image>`：

```dockerfile
FROM docker.1ms.run/kalilinux/kali-rolling
# 但 kali-rolling 镜像太大，镜像站未同步 — 需直接拉
```

### build 缓存导致旧 layer 被复用

修改 go.mod 后必须 `--no-cache` 重建，否则 Docker 复用旧的 COPY + RUN layer，replace 指令不生效。

```bash
docker build --no-cache -t asset-mcp -f docker/asset-mcp/Dockerfile .
```

---

## 4. go.work + Docker 构建冲突

go.work 中的本地模块引用（如 `adversarychef/mcputil`）在 Docker 构建上下文中失效，因为 Docker `COPY` 重排了目录结构。

### 解决方案：在 go.mod 中加 replace 指令

```go
// servers/asset/go.mod
require (
    adversarychef/mcputil v0.0.0
    // ...
)

replace adversarychef/mcputil => ../../pkg/mcputil
```

且 replace 依赖的 `pkg/` 目录必须在 `go mod download` 之前 COPY：

```dockerfile
COPY servers/asset/go.mod servers/asset/go.sum ./
COPY pkg/ ../pkg/          # ← 必须在 go mod download 之前
RUN go mod download
```

---

## 5. cc-switch 配置必须存在

### 问题

Codex 启动时读取 `~/.codex/config.toml`。文件不存在 → Codex 尝试直连 `api.openai.com`，反复断开重连直到 5 次耗尽，任务被 Multica 取消（`task cancelled by upstream context`）。

### 修复

在运行 daemon 的用户下创建 `~/.codex/config.toml`：

```toml
model_provider = "custom"
model = "deepseek-v4-pro"
model_reasoning_effort = "high"
disable_response_storage = true

[model_providers.custom]
name = "deepseek"
base_url = "http://127.0.0.1:15722/v1"    # ← Codex proxy port
wire_api = "responses"
requires_openai_auth = true
experimental_bearer_token = "proxy-placeholder"
```

---

## 6. daemon 需要重启才能检测新安装的 runtime

Codex 安装到 `/usr/local/bin` 后，Multica daemon 不会自动检测 — daemon 只在启动时扫描 `PATH`。

```bash
multica daemon restart  # 重启后 Agent 列表出现 codex
```

---

## 7. home-dev 内存约束

| 资源 | 值 | 影响 |
|------|-----|------|
| RAM | 3.3 GB | 只能跑轻量容器，Kali 镜像（~5GB）放不下 |
| 磁盘 | 48GB，剩 15GB | 足够轻量容器，但 Kali Docker image + apt 装工具会爆 |

### 已优化

```bash
# 停止不必要的服务（释放 ~330MB）
sudo systemctl stop fwupd fwupd-refresh snapd snapd.socket
sudo systemctl mask fwupd-refresh.timer
```

---

## 8. asset-mcp 端口映射

Multica daemon 是宿主机原生进程，asset-mcp 是 Docker 容器。MCP URL 必须用**宿主机回环地址**：

```
http://127.0.0.1:8081
```

不能用 `http://asset-mcp:8081`（Docker 内部 DNS，宿主机进程无法解析）。

启动时必须 `-p 8081:8081`。

---

## 9. daemon 以哪个用户运行很重要

home-dev 上 daemon 以 `dev` 用户运行，但 codex/cc-switch 最初以 `root` 安装。

- `which codex` 以 root 执行 → 能找到
- 但 daemon 是 `dev` 用户 → **也能找到**（`/usr/local/bin` 在 PATH 中）
- `~/.codex/config.toml` 是 per-user 的 → 必须为 `dev` 用户创建

---

## 10. Docker build 残留层占用磁盘

多次 `docker build --no-cache` 产生大量 `<none>` 镜像：

```bash
docker image prune -f  # 清理（Phase 0 清理了 483MB）
```

---

## 快速排障清单

| 症状 | 检查 |
|------|------|
| Codex 任务无响应 | `~/.codex/config.toml` 是否存在？base_url 端口是否正确（15722）？ |
| MCP 返回 400 sessionid | 是否用了 `NewSSEHandler`？应改为 `NewStreamableHTTPHandler` |
| MCP 配置改了不生效 | `multica daemon restart` |
| Docker build 用旧 go.mod | 加 `--no-cache` |
| go.mod replace 不生效 | `pkg/` COPY 在 `go mod download` 之前？ |
| Agent 找不到 codex | `multica daemon restart` |
| 内存不足 | `free -h`，关 fwupd/snapd/Xvfb |
