# MCP Server 生产化重构 — 设计文档

- **created**: 2026-07-25
- **status**: approved
- **scope**: servers/ 目录重构 + pkg/ 共享库 + 代码质量提升

---

## 1. 目标

把 AdversaryChefAgentTeam 的 MCP Server 从原型阶段推到生产级：
- 消除两个 server 间的重复代码和启动 boilerplate
- 修复目录命名不规范
- 加 graceful shutdown / health check / 中间件
- 用泛型消除 asset-mcp 的机械 CRUD 重复
- 清理死代码
- Skills/config 从空占位升级为有内容的骨架

## 2. 目录结构变更

### 前

```
mcp_servers/
├── kali-mcp/
│   ├── cmd/server/main.go
│   └── internal/
│       ├── executor/       # 死代码
│       ├── job/
│       └── tools/
└── asset-mcp/
    ├── cmd/server/main.go
    └── internal/
        ├── models/
        ├── store/
        └── tools/
```

### 后

```
servers/
├── kali/
│   ├── cmd/server/main.go
│   └── internal/
│       ├── job/
│       │   ├── job.go
│       │   ├── manager.go
│       │   └── manager_test.go
│       └── tools/
│           ├── register.go
│           ├── exec.go
│           ├── query.go
│           └── kill.go
└── asset/
    ├── cmd/server/main.go
    └── internal/
        ├── models/
        │   └── models.go
        ├── store/
        │   ├── store.go
        │   ├── memory.go
        │   └── sqlite.go
        └── tools/
            ├── crud.go
            ├── register.go
            ├── project.go
            ├── asset.go
            ├── clue.go
            ├── credential.go
            └── worklog.go

pkg/
└── mcputil/
    └── mcputil.go

go.work                          # 新
```

## 3. 共享库 `pkg/mcputil`

### API

```go
type ServerConfig struct {
    Host    string
    Port    int
    DBPath  string          // asset 专用，kali 忽略
    Name    string
    Version string
}

func ParseConfig(name, version string, defaultPort int) ServerConfig
func TextResult(text string) *mcp.CallToolResult
func Run(cfg ServerConfig, register func(*mcp.Server)) error
```

### Run() 内置行为

| 功能 | 实现 |
|------|------|
| MCP server 初始化 + SSE handler | `mcp.NewServer` + `mcp.NewSSEHandler` |
| `/health` | `GET /health` → `{"status":"ok","server":"...","version":"..."}` |
| 请求日志 | method path status duration |
| Panic recovery | `defer/recover` → 500 |
| 优雅关闭 | SIGINT/SIGTERM → drain → 5s timeout |

### 调用方示例

```go
// servers/kali/cmd/server/main.go
func main() {
    cfg := mcputil.ParseConfig("kali", "0.3.0", 8080)
    mgr := job.NewManager(job.DefaultMaxOutput, job.DefaultTimeout)
    mcputil.Run(cfg, func(s *mcp.Server) { tools.Register(s, mgr) })
}

// servers/asset/cmd/server/main.go
func main() {
    cfg := mcputil.ParseConfig("asset", "0.2.0", 8081)
    store, _ := store.NewSQLiteStore(cfg.DBPath)
    defer store.Close()
    mcputil.Run(cfg, func(s *mcp.Server) { tools.RegisterAll(s, store) })
}
```

## 4. Kali 重构

### 删除 executor 包

`executor/executor.go` + `executor/executor_test.go` — 两个文件从未被 import 过。
`job/manager.go` 自己内联实现了完整的命令执行逻辑（streaming pipe + 进程组 + 超时）。

### tools/bash.go 拆分

| 文件 | 内容 |
|------|------|
| `tools/register.go` | `Register(server, mgr)` 入口 |
| `tools/exec.go` | `exec` tool handler |
| `tools/query.go` | `list_jobs` + `get_job` handlers |
| `tools/kill.go` | `kill_job` handler |

### manager.go 优化

- 常量提取：`DefaultMaxOutput = 500_000`、`DefaultTimeout = 30 * time.Minute`
- `genJobID()` 加随机后缀防并发碰撞

## 5. Asset 泛型 CRUD

### 新增 `tools/crud.go`

```go
func Lister[T any](server, name, desc string, fn func(string) ([]T, error))
func ListerAll[T any](server, name, desc string, fn func() ([]T, error))
func Getter[T any](server, name, desc string, fn func(string) (*T, error))
func Deleter(server, name, desc string, fn func(string) error)
```

### Per-entity 文件精简

| 文件 | 前（行） | 后（行） | 保留手写 |
|------|---------|---------|---------|
| project.go | 104 | ~45 | create/update |
| asset.go | 126 | ~45 | create/update |
| clue.go | 114 | ~45 | create/update |
| credential.go | 122 | ~45 | create/update |
| worklog.go | 102 | ~40 | create/update |

`tools/helper.go` 删除（textResult 移到 mcputil）。

### create/update 保留手写原因

每个实体的 params → model 映射完全不同（字段数 3~7，类型含 `[]string`、`string` 混合），泛型化得不偿失。

## 6. Docker 配套更新

### docker-compose healthcheck

```yaml
services:
  kali:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
  asset:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
      interval: 30s
      timeout: 3s
      retries: 3
```

## 7. Skills & Config 占位

### Skills：空 `.gitkeep` → 最小 `SKILL.md`

5 个 skill 目录各创建 `SKILL.md`，统一格式：

```markdown
---
title: "Skill Name"
status: draft
---

# Skill Name

<!-- TODO: fill in content -->
```

### Config：空 `.gitkeep` → example 模板

- `config/codex/config.toml.example`
- `config/multica/docker-compose.selfhost.yml.example`

## 8. 变更影响范围

| 模块 | 操作 | 风险 |
|------|------|------|
| `mcp_servers/` → `servers/` | 重命名 | 需更新 docker-compose build context 和 Dockerfile |
| `mcp_servers/kali-mcp` → `servers/kali` | 名 + 路径 | go module 路径改为 `adversarychef/kali` |
| `mcp_servers/asset-mcp` → `servers/asset` | 名 + 路径 | go module 路径改为 `adversarychef/asset` |
| 新增 `pkg/mcputil` | 新 module | `go.work` 引用 |
| executor 包 | 删除 | 无影响（死代码） |
| tools/bash.go | 拆分为 4 文件 | 功能不变 |
| tools/helper.go | 删除 | `TextResult` 移到 mcputil |
| asset tools | 加泛型 crud.go | List/Get/Delete handler 变调用；create/update 不变 |
| main.go | 50+ 行 → 10 行 | 启动行为由 mcputil.Run 统一 |

## 9. go.work

```go
go 1.26.4

use (
    ./pkg/mcputil
    ./servers/asset
    ./servers/kali
)
```

## 10. 不做

- CORS — Docker 内部 SSE 直连不需要
- Rate limiting — 内部网络
- Metrics / Prometheus — 无监控对接
- TLS — 内部明文 HTTP
- Skills 内容填充 — 用户后续自行决定
- 泛型化 create/update — 得不偿失
