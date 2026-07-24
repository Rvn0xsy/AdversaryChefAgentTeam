# MCP Server 生产化重构 — 实现计划

> **Goal:** 将 AdversaryChefAgentTeam 的 MCP Server 从原型推到生产级：共享库、目录重命名、代码消除重复、graceful shutdown + health check

> **Architecture:** go.work monorepo 管理 3 个 module：pkg/mcputil（共享启动库）、servers/kali、servers/asset。mcputil.Run() 封装 MCP server 全生命周期（启动/健康检查/优雅关闭）

> **Tech Stack:** Go 1.26, modelcontextprotocol/go-sdk v1.6.1, Docker Compose

## Global Constraints

- Go 1.26.4 module 路径格式：adversarychef/<name>
- HTTP/SSE transport，非 stdio
- 不改功能行为，只做结构性重构
- Docker 内部网络 HTTP 明文

---

### Task 1: 创建新目录结构

**Files:**
- Create: `servers/kali/cmd/server/.gitkeep`
- Create: `servers/kali/internal/job/.gitkeep`
- Create: `servers/kali/internal/tools/.gitkeep`
- Create: `servers/asset/cmd/server/.gitkeep`
- Create: `servers/asset/internal/models/.gitkeep`
- Create: `servers/asset/internal/store/.gitkeep`
- Create: `servers/asset/internal/tools/.gitkeep`
- Create: `pkg/mcputil/.gitkeep`
- Create: `go.work`

**Interfaces:**
- Produces: 目录骨架就绪，等待文件迁移

- [ ] **Step 1: 创建目录**

```bash
mkdir -p servers/kali/cmd/server
mkdir -p servers/kali/internal/job
mkdir -p servers/kali/internal/tools
mkdir -p servers/asset/cmd/server
mkdir -p servers/asset/internal/models
mkdir -p servers/asset/internal/store
mkdir -p servers/asset/internal/tools
mkdir -p pkg/mcputil
```

- [ ] **Step 2: 创建 go.work**

```
go 1.26.4

use (
    ./pkg/mcputil
    ./servers/asset
    ./servers/kali
)
```

写文件: `go.work`

- [ ] **Step 3: 验证**

```bash
cat go.work
ls -R servers/ pkg/
```

---

### Task 2: 迁移 kali-mcp → servers/kali（go module）

**Files:**
- Copy + Modify: `mcp_servers/kali-mcp/go.mod` → `servers/kali/go.mod`
- Copy + Modify: `mcp_servers/kali-mcp/go.sum` → `servers/kali/go.sum`
- Copy: `mcp_servers/kali-mcp/cmd/server/main.go` → `servers/kali/cmd/server/main.go`
- Copy: `mcp_servers/kali-mcp/internal/job/*.go` → `servers/kali/internal/job/`
- Copy: `mcp_servers/kali-mcp/internal/tools/bash.go` → `servers/kali/internal/tools/bash.go`

**Interfaces:**
- Produces: `servers/kali/` 完整可编译

- [ ] **Step 1: 复制文件**

```bash
cp mcp_servers/kali-mcp/go.mod servers/kali/go.mod
cp mcp_servers/kali-mcp/go.sum servers/kali/go.sum
cp mcp_servers/kali-mcp/cmd/server/main.go servers/kali/cmd/server/main.go
cp mcp_servers/kali-mcp/internal/job/*.go servers/kali/internal/job/
cp mcp_servers/kali-mcp/internal/tools/bash.go servers/kali/internal/tools/bash.go
cp mcp_servers/kali-mcp/internal/executor/executor.go servers/kali/internal/job/executor.go
cp mcp_servers/kali-mcp/internal/executor/executor_test.go servers/kali/internal/job/executor_test.go
```

- [ ] **Step 2: 更新 go.mod module 路径**

修改 `servers/kali/go.mod`:
```
module adversarychef/kali-mcp
```
改为:
```
module adversarychef/kali
```

- [ ] **Step 3: 更新 import 路径**

在 `servers/kali/` 下所有 `.go` 文件中：
```
"adversarychef/kali-mcp/internal/job"
"adversarychef/kali-mcp/internal/tools"
```
改为:
```
"adversarychef/kali/internal/job"
"adversarychef/kali/internal/tools"
```

- [ ] **Step 4: 验证编译**

```bash
cd servers/kali && go build ./...
```

---

### Task 3: 迁移 asset-mcp → servers/asset（go module）

**Files:**
- Copy + Modify: `mcp_servers/asset-mcp/go.mod` → `servers/asset/go.mod`
- Copy + Modify: `mcp_servers/asset-mcp/go.sum` → `servers/asset/go.sum`
- Copy: `mcp_servers/asset-mcp/cmd/server/main.go` → `servers/asset/cmd/server/main.go`
- Copy: `mcp_servers/asset-mcp/internal/**/*.go` → `servers/asset/internal/`

**Interfaces:**
- Produces: `servers/asset/` 完整可编译

- [ ] **Step 1: 复制文件**

```bash
cp mcp_servers/asset-mcp/go.mod servers/asset/go.mod
cp mcp_servers/asset-mcp/go.sum servers/asset/go.sum
cp mcp_servers/asset-mcp/cmd/server/main.go servers/asset/cmd/server/main.go
cp mcp_servers/asset-mcp/internal/models/models.go servers/asset/internal/models/models.go
cp mcp_servers/asset-mcp/internal/store/*.go servers/asset/internal/store/
cp mcp_servers/asset-mcp/internal/tools/*.go servers/asset/internal/tools/
```

- [ ] **Step 2: 更新 go.mod module 路径**

修改 `servers/asset/go.mod`:
```
module adversarychef/asset-mcp
```
改为:
```
module adversarychef/asset
```

- [ ] **Step 3: 更新 import 路径**

在 `servers/asset/` 下所有 `.go` 文件中：
```
"adversarychef/asset-mcp/internal/models"
"adversarychef/asset-mcp/internal/store"
"adversarychef/asset-mcp/internal/tools"
```
改为:
```
"adversarychef/asset/internal/models"
"adversarychef/asset/internal/store"
"adversarychef/asset/internal/tools"
```

- [ ] **Step 4: 验证编译**

```bash
cd servers/asset && go build ./...
```

---

### Task 4: 创建 pkg/mcputil 共享库

**Files:**
- Create: `pkg/mcputil/go.mod`
- Create: `pkg/mcputil/mcputil.go`

**Interfaces:**
- Consumes: 无
- Produces: `mcputil.ParseConfig(name, version string, defaultPort int) ServerConfig`
- Produces: `mcputil.TextResult(text string) *mcp.CallToolResult`
- Produces: `mcputil.Run(cfg ServerConfig, register func(*mcp.Server)) error`
- Produces: `ServerConfig{Host, Port, DBPath, Name, Version}`

- [ ] **Step 1: 初始化 go module**

```bash
cd pkg/mcputil
go mod init adversarychef/mcputil
go get github.com/modelcontextprotocol/go-sdk@v1.6.1
```

- [ ] **Step 2: 写 mcputil.go**

```go
// Package mcputil provides shared MCP server infrastructure:
// config parsing, graceful shutdown, health check, text results.
package mcputil

import (
    "context"
    "flag"
    "fmt"
    "log"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerConfig holds common server configuration.
type ServerConfig struct {
    Host    string
    Port    int
    DBPath  string
    Name    string
    Version string
}

// Addr returns the listen address.
func (c ServerConfig) Addr() string {
    return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// ParseConfig parses CLI flags and returns a ServerConfig.
// Registers --host, --port, and --db flags.
func ParseConfig(name, version string, defaultPort int) ServerConfig {
    host := flag.String("host", "0.0.0.0", "host to listen on")
    port := flag.Int("port", defaultPort, "port to listen on")
    dbPath := flag.String("db", "asset.db", "sqlite database file path")

    flag.Usage = func() {
        fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
        fmt.Fprintf(os.Stderr, "  %s MCP Server v%s\n\n", name, version)
        fmt.Fprintf(os.Stderr, "Options:\n")
        flag.PrintDefaults()
    }
    flag.Parse()

    return ServerConfig{
        Host:    *host,
        Port:    *port,
        DBPath:  *dbPath,
        Name:    name,
        Version: version,
    }
}

// TextResult creates a text content MCP result.
func TextResult(text string) *mcp.CallToolResult {
    return &mcp.CallToolResult{
        Content: []mcp.Content{
            &mcp.TextContent{Text: text},
        },
    }
}

// Run starts the MCP server with graceful shutdown.
// register is called with the server for tool registration.
func Run(cfg ServerConfig, register func(*mcp.Server)) error {
    server := mcp.NewServer(&mcp.Implementation{
        Name:    cfg.Name,
        Version: cfg.Version,
    }, nil)

    register(server)

    sseHandler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
        return server
    }, nil)

    mux := http.NewServeMux()
    mux.HandleFunc("/health", healthHandler(cfg))
    mux.Handle("/", sseHandler)

    wrapped := withMiddleware(mux)

    srv := &http.Server{
        Addr:    cfg.Addr(),
        Handler: wrapped,
    }

    go func() {
        log.Printf("[%s] listening on %s", cfg.Name, cfg.Addr())
        if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            log.Fatalf("[%s] server failed: %v", cfg.Name, err)
        }
    }()

    ctx, stop := signal.NotifyContext(context.Background(),
        syscall.SIGINT, syscall.SIGTERM)
    defer stop()
    <-ctx.Done()

    log.Printf("[%s] shutting down...", cfg.Name)
    shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    return srv.Shutdown(shutdownCtx)
}

func healthHandler(cfg ServerConfig) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, `{"status":"ok","server":"%s","version":"%s"}`, cfg.Name, cfg.Version)
    }
}

// withMiddleware applies request logging and panic recovery.
func withMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        // recovery
        defer func() {
            if rec := recover(); rec != nil {
                log.Printf("[%s] PANIC %s %s: %v",
                    r.URL.Path, r.Method, r.URL.Path, rec)
                http.Error(w, "internal server error", http.StatusInternalServerError)
            }
        }()

        // delegate to SSE handler
        next.ServeHTTP(w, r)

        log.Printf("[%s] %s %s %v",
            r.URL.Path, r.Method, r.URL.Path, time.Since(start))
    })
}
```

- [ ] **Step 3: 验证编译**

```bash
cd pkg/mcputil && go build ./...
```

---

### Task 5: 重构 servers/kali — 删除 executor、精简 main.go

**Files:**
- Delete: `servers/kali/internal/job/executor.go`
- Delete: `servers/kali/internal/job/executor_test.go`
- Modify: `servers/kali/cmd/server/main.go`
- Create: `servers/kali/internal/tools/register.go`
- Create: `servers/kali/internal/tools/exec.go`
- Create: `servers/kali/internal/tools/query.go`
- Create: `servers/kali/internal/tools/kill.go`
- Delete: `servers/kali/internal/tools/bash.go`

**Interfaces:**
- Consumes: `mcputil.ParseConfig`, `mcputil.TextResult`, `mcputil.Run`
- Produces: `tools.Register(server *mcp.Server, mgr *job.Manager)`

- [ ] **Step 1: 删除死代码**

```bash
rm servers/kali/internal/job/executor.go
rm servers/kali/internal/job/executor_test.go
```

- [ ] **Step 2: 拆分 bash.go → register.go + exec.go + query.go + kill.go**

创建 `servers/kali/internal/tools/register.go`:
```go
package tools

import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "adversarychef/kali/internal/job"
)

func Register(server *mcp.Server, mgr *job.Manager) {
    registerExec(server, mgr)
    registerListJobs(server, mgr)
    registerGetJob(server, mgr)
    registerKillJob(server, mgr)
}
```

创建 `servers/kali/internal/tools/exec.go`:
```go
package tools

import (
    "context"
    "fmt"
    "time"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "adversarychef/kali/internal/job"
    "adversarychef/mcputil"
)

const (
    defaultTimeout = 30 * time.Minute
)

type BashParams struct {
    Command string `json:"command" jsonschema:"要执行的 shell 命令"`
    Timeout int    `json:"timeout,omitempty" jsonschema:"超时秒数，默认 1800（30分钟）"`
}

func registerExec(server *mcp.Server, mgr *job.Manager) {
    mcp.AddTool(server, &mcp.Tool{
        Name: "exec",
        Description: "在 Kali Linux 容器中异步执行 shell 命令。返回 job_id，可通过 get_job 查询进度和输出，通过 kill_job 终止。",
    }, func(ctx context.Context, req *mcp.CallToolRequest, params BashParams) (*mcp.CallToolResult, any, error) {
        timeout := defaultTimeout
        if params.Timeout > 0 {
            timeout = time.Duration(params.Timeout) * time.Second
        }
        jobID := mgr.Start(params.Command, timeout)
        return mcputil.TextResult(
            fmt.Sprintf("job started: %s\ncommand: %s\ntimeout: %v\nuse get_job to check progress, kill_job to stop.",
                jobID, params.Command, timeout)), nil, nil
    })
}
```

创建 `servers/kali/internal/tools/query.go`:
```go
package tools

import (
    "context"
    "encoding/json"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "adversarychef/kali/internal/job"
    "adversarychef/mcputil"
)

func registerListJobs(server *mcp.Server, mgr *job.Manager) {
    mcp.AddTool(server, &mcp.Tool{
        Name: "list_jobs",
        Description: "列出所有任务。可选按状态过滤（running/completed/failed/killed/timed_out）。",
    }, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
        Status string `json:"status,omitempty" jsonschema:"按状态过滤"`
    }) (*mcp.CallToolResult, any, error) {
        jobs := mgr.List(job.Status(params.Status))
        b, _ := json.Marshal(jobs)
        return mcputil.TextResult(string(b)), nil, nil
    })
}

func registerGetJob(server *mcp.Server, mgr *job.Manager) {
    mcp.AddTool(server, &mcp.Tool{
        Name: "get_job",
        Description: "获取指定任务的详细信息，包括当前状态和已产生的输出。",
    }, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
        JobID string `json:"job_id" jsonschema:"任务 ID"`
    }) (*mcp.CallToolResult, any, error) {
        j, err := mgr.Get(params.JobID)
        if err != nil {
            return mcputil.TextResult("未找到: " + err.Error()), nil, nil
        }
        b, _ := json.Marshal(j)
        return mcputil.TextResult(string(b)), nil, nil
    })
}
```

创建 `servers/kali/internal/tools/kill.go`:
```go
package tools

import (
    "context"

    "github.com/modelcontextprotocol/go-sdk/mcp"
    "adversarychef/kali/internal/job"
    "adversarychef/mcputil"
)

func registerKillJob(server *mcp.Server, mgr *job.Manager) {
    mcp.AddTool(server, &mcp.Tool{
        Name: "kill_job",
        Description: "终止指定任务。会强制 kill 整个进程组。",
    }, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
        JobID string `json:"job_id" jsonschema:"任务 ID"`
    }) (*mcp.CallToolResult, any, error) {
        if err := mgr.Kill(params.JobID); err != nil {
            return mcputil.TextResult("终止失败: " + err.Error()), nil, nil
        }
        return mcputil.TextResult("job killed: " + params.JobID), nil, nil
    })
}
```

删除旧文件:
```bash
rm servers/kali/internal/tools/bash.go
```

- [ ] **Step 3: 精简 main.go**

写入 `servers/kali/cmd/server/main.go`:
```go
package main

import (
    "adversarychef/kali/internal/job"
    "adversarychef/kali/internal/tools"
    "adversarychef/mcputil"
)

func main() {
    cfg := mcputil.ParseConfig("kali", "0.3.0", 8080)
    mgr := job.NewManager(job.DefaultMaxOutput, job.DefaultTimeout)
    mcputil.Run(cfg, func(s *mcp.Server) { tools.Register(s, mgr) })
}
```

需要添加 `mcp` import:
```go
import (
    "github.com/modelcontextprotocol/go-sdk/mcp"
    "adversarychef/kali/internal/job"
    "adversarychef/kali/internal/tools"
    "adversarychef/mcputil"
)
```

- [ ] **Step 4: 给 job/manager.go 加常量**

在 `servers/kali/internal/job/manager.go` 的 `NewManager` 前加:
```go
const (
    DefaultMaxOutput = 500_000
    DefaultTimeout   = 30 * time.Minute
)
```

- [ ] **Step 5: go mod tidy + 编译**

```bash
cd servers/kali
go get adversarychef/mcputil@latest
go mod tidy
go build ./...
```

- [ ] **Step 6: 运行测试**

```bash
cd servers/kali && go test ./...
```

---

### Task 6: 重构 servers/asset — crud.go + 精简 main.go

**Files:**
- Create: `servers/asset/internal/tools/crud.go`
- Delete: `servers/asset/internal/tools/helper.go`
- Modify: `servers/asset/internal/tools/project.go`（精简 list/get/delete）
- Modify: `servers/asset/internal/tools/asset.go`（精简 list/get/delete）
- Modify: `servers/asset/internal/tools/clue.go`（精简 list/get/delete）
- Modify: `servers/asset/internal/tools/credential.go`（精简 list/get/delete）
- Modify: `servers/asset/internal/tools/worklog.go`（精简 list/get/delete）
- Modify: `servers/asset/cmd/server/main.go`

**Interfaces:**
- Consumes: `mcputil.ParseConfig`, `mcputil.TextResult`, `mcputil.Run`
- Produces: `crud.Lister`, `crud.ListerAll`, `crud.Getter`, `crud.Deleter`

- [ ] **Step 1: 创建 crud.go**

写入 `servers/asset/internal/tools/crud.go`:
```go
package tools

import (
    "context"
    "encoding/json"

    "github.com/modelcontextprotocol/go-sdk/mcp"

    "adversarychef/mcputil"
)

// Lister registers a list tool that filters by project ID.
func Lister[T any](server *mcp.Server, name, desc, entityName string,
    fn func(string) ([]T, error)) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        name,
        Description: desc,
    }, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
        ProjectID string `json:"project_id" jsonschema:"项目 ID"`
    }) (*mcp.CallToolResult, any, error) {
        items, err := fn(params.ProjectID)
        if err != nil {
            return mcputil.TextResult("查询失败: " + err.Error()), nil, nil
        }
        b, _ := json.Marshal(items)
        return mcputil.TextResult(string(b)), nil, nil
    })
}

// ListerAll registers a list tool with no filter (for projects).
func ListerAll[T any](server *mcp.Server, name, desc string,
    fn func() ([]T, error)) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        name,
        Description: desc,
    }, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
        items, err := fn()
        if err != nil {
            return mcputil.TextResult("查询失败: " + err.Error()), nil, nil
        }
        b, _ := json.Marshal(items)
        return mcputil.TextResult(string(b)), nil, nil
    })
}

// Getter registers a get-by-ID tool.
func Getter[T any](server *mcp.Server, name, desc, entityName string,
    fn func(string) (*T, error)) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        name,
        Description: desc,
    }, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
        ID string `json:"id" jsonschema:"记录 ID"`
    }) (*mcp.CallToolResult, any, error) {
        item, err := fn(params.ID)
        if err != nil {
            return mcputil.TextResult(entityName + "不存在: " + err.Error()), nil, nil
        }
        b, _ := json.Marshal(item)
        return mcputil.TextResult(string(b)), nil, nil
    })
}

// Deleter registers a delete-by-ID tool.
func Deleter(server *mcp.Server, name, desc, entityName string,
    fn func(string) error) {
    mcp.AddTool(server, &mcp.Tool{
        Name:        name,
        Description: desc,
    }, func(ctx context.Context, req *mcp.CallToolRequest, params struct {
        ID string `json:"id" jsonschema:"记录 ID"`
    }) (*mcp.CallToolResult, any, error) {
        if err := fn(params.ID); err != nil {
            return mcputil.TextResult("删除失败: " + err.Error()), nil, nil
        }
        return mcputil.TextResult("已删除"), nil, nil
    })
}
```

- [ ] **Step 2: 删除 helper.go**

```bash
rm servers/asset/internal/tools/helper.go
```

- [ ] **Step 3: 精简 project.go**

重写 `servers/asset/internal/tools/project.go`，前 5 行 list/get/delete 换成 crud 调用:
```go
package tools

import (
    "context"
    "encoding/json"

    "github.com/modelcontextprotocol/go-sdk/mcp"

    "adversarychef/asset/internal/models"
    "adversarychef/asset/internal/store"
    "adversarychef/mcputil"
)

type createProjectParams struct {
    Name        string `json:"name" jsonschema:"项目名称"`
    Description string `json:"description,omitempty" jsonschema:"项目描述"`
    Status      string `json:"status" jsonschema:"项目状态，如 active/completed/archived"`
}

type updateProjectParams struct {
    ID          string `json:"id" jsonschema:"项目 ID"`
    Name        string `json:"name,omitempty" jsonschema:"项目名称"`
    Description string `json:"description,omitempty" jsonschema:"项目描述"`
    Status      string `json:"status,omitempty" jsonschema:"项目状态"`
}

func registerProjects(server *mcp.Server, s store.Store) {
    ListerAll(server, "list_projects", "列出所有渗透项目", s.ListProjects)
    Getter(server, "get_project", "获取单个项目详情", "项目", s.GetProject)
    Deleter(server, "delete_project", "删除项目", "项目", s.DeleteProject)

    mcp.AddTool(server, &mcp.Tool{
        Name:        "create_project",
        Description: "创建新的渗透项目",
    }, func(ctx context.Context, req *mcp.CallToolRequest, params createProjectParams) (*mcp.CallToolResult, any, error) {
        p := &models.Project{
            Name:        params.Name,
            Description: params.Description,
            Status:      params.Status,
        }
        if err := s.CreateProject(p); err != nil {
            return mcputil.TextResult("创建失败: " + err.Error()), nil, nil
        }
        b, _ := json.Marshal(p)
        return mcputil.TextResult(string(b)), nil, nil
    })

    mcp.AddTool(server, &mcp.Tool{
        Name:        "update_project",
        Description: "更新项目信息",
    }, func(ctx context.Context, req *mcp.CallToolRequest, params updateProjectParams) (*mcp.CallToolResult, any, error) {
        existing, err := s.GetProject(params.ID)
        if err != nil {
            return mcputil.TextResult("项目不存在: " + err.Error()), nil, nil
        }
        if params.Name != "" {
            existing.Name = params.Name
        }
        if params.Description != "" {
            existing.Description = params.Description
        }
        if params.Status != "" {
            existing.Status = params.Status
        }
        if err := s.UpdateProject(existing); err != nil {
            return mcputil.TextResult("更新失败: " + err.Error()), nil, nil
        }
        b, _ := json.Marshal(existing)
        return mcputil.TextResult(string(b)), nil, nil
    })
}
```

- [ ] **Step 4: 精简 asset.go**

同样模式 — 前 3 行 list/get/delete 换 crud:
```go
func registerAssets(server *mcp.Server, s store.Store) {
    Lister(server, "list_assets", "按项目 ID 列出所有目标资产", "资产", s.ListAssets)
    Getter(server, "get_asset", "获取单个资产详情", "资产", s.GetAsset)
    Deleter(server, "delete_asset", "删除资产", "资产", s.DeleteAsset)
    // create_asset + update_asset 保留原样
    ...
}
```

- [ ] **Step 5: 精简 clue.go**

```go
func registerClues(server *mcp.Server, s store.Store) {
    Lister(server, "list_clues", "按项目 ID 列出所有渗透线索/发现", "线索", s.ListClues)
    Getter(server, "get_clue", "获取单个线索详情", "线索", s.GetClue)
    Deleter(server, "delete_clue", "删除线索", "线索", s.DeleteClue)
    // create_clue + update_clue 保留原样
    ...
}
```

- [ ] **Step 6: 精简 credential.go**

```go
func registerCredentials(server *mcp.Server, s store.Store) {
    Lister(server, "list_credentials", "按项目 ID 列出所有账户凭据", "凭据", s.ListCredentials)
    Getter(server, "get_credential", "获取单个凭据详情", "凭据", s.GetCredential)
    Deleter(server, "delete_credential", "删除凭据", "凭据", s.DeleteCredential)
    // create_credential + update_credential 保留原样
    ...
}
```

- [ ] **Step 7: 精简 worklog.go**

```go
func registerWorkLogs(server *mcp.Server, s store.Store) {
    Lister(server, "list_work_logs", "按项目 ID 列出所有工作日志", "日志", s.ListWorkLogs)
    Getter(server, "get_work_log", "获取单条工作日志详情", "日志", s.GetWorkLog)
    Deleter(server, "delete_work_log", "删除工作日志", "日志", s.DeleteWorkLog)
    // create_work_log + update_work_log 保留原样
    ...
}
```

- [ ] **Step 8: 精简 main.go**

写入 `servers/asset/cmd/server/main.go`:
```go
package main

import (
    "log"

    "github.com/modelcontextprotocol/go-sdk/mcp"

    "adversarychef/asset/internal/store"
    "adversarychef/asset/internal/tools"
    "adversarychef/mcputil"
)

func main() {
    cfg := mcputil.ParseConfig("asset", "0.2.0", 8081)
    s, err := store.NewSQLiteStore(cfg.DBPath)
    if err != nil {
        log.Fatalf("failed to open database: %v", err)
    }
    defer s.Close()
    mcputil.Run(cfg, func(server *mcp.Server) { tools.RegisterAll(server, s) })
}
```

- [ ] **Step 9: go mod tidy + 编译**

```bash
cd servers/asset
go get adversarychef/mcputil@latest
go mod tidy
go build ./...
```

---

### Task 7: 更新 Docker 文件

**Files:**
- Modify: `docker/kali-mcp/Dockerfile`
- Modify: `docker/asset-mcp/Dockerfile`
- Modify: `docker/docker-compose.yml`

- [ ] **Step 1: 更新 kali Dockerfile**

修改 `docker/kali-mcp/Dockerfile` 中的 COPY 路径:
```dockerfile
# 构建阶段
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY servers/kali/go.mod servers/kali/go.sum ./
RUN go mod download
COPY servers/kali/ ./
COPY pkg/ ../pkg/
RUN CGO_ENABLED=0 go build -o /kali-mcp ./cmd/server
```

`COPY mcp_servers/kali-mcp/` → `COPY servers/kali/`，加 `COPY pkg/ ../pkg/`

- [ ] **Step 2: 更新 asset Dockerfile**

修改 `docker/asset-mcp/Dockerfile`:
```dockerfile
FROM golang:1.26-alpine AS builder
WORKDIR /build
COPY servers/asset/go.mod servers/asset/go.sum ./
RUN go mod download
COPY servers/asset/ ./
COPY pkg/ ../pkg/
RUN CGO_ENABLED=0 go build -o /asset-mcp ./cmd/server
```

- [ ] **Step 3: 更新 docker-compose.yml**

加 healthcheck 到 `docker/docker-compose.yml`:
```yaml
services:
  daemon:
    ...
  kali-mcp:
    ...
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
  asset-mcp:
    ...
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8081/health"]
      interval: 30s
      timeout: 3s
      retries: 3
```

---

### Task 8: Skills & Config 骨架

**Files:**
- Create: `skills/penetration-methodology/SKILL.md`
- Create: `skills/owasp-checklist/SKILL.md`
- Create: `skills/compliance-rules/SKILL.md`
- Create: `skills/weekly-report-template/SKILL.md`
- Create: `skills/retrospective-template/SKILL.md`
- Create: `config/codex/config.toml.example`
- Create: `config/multica/docker-compose.selfhost.yml.example`
- Delete: `skills/*/.gitkeep` (5 files)
- Delete: `config/codex/.gitkeep`
- Delete: `config/multica/.gitkeep`

- [ ] **Step 1: 创建 SKILL.md 骨架**

5 个 skill 文件使用统一模板:

`skills/penetration-methodology/SKILL.md`:
```markdown
---
title: "Penetration Testing Methodology"
status: draft
---

# Penetration Testing Methodology

Standard methodology to follow during penetration testing engagements,
based on PTES (Penetration Testing Execution Standard).

<!-- TODO: Fill in detailed steps per phase -->

## Phases

### 1. Reconnaissance
<!-- TODO -->

### 2. Scanning & Enumeration
<!-- TODO -->

### 3. Exploitation
<!-- TODO -->

### 4. Post-Exploitation
<!-- TODO -->

### 5. Reporting
<!-- TODO -->
```

其他 4 个同理，替换 title 和 description。

- [ ] **Step 2: 创建 config example 文件**

`config/codex/config.toml.example`:
```toml
# Codex cc-switch configuration
# Rename to config.toml and fill in actual values

[provider.custom]
url = "https://cc-switch.example.com/v1/chat/completions"
api_key = "YOUR_CC_SWITCH_API_KEY"
model = "deepseek-v4-pro"
```

`config/multica/docker-compose.selfhost.yml.example`:
```yaml
# Multica self-hosted deployment template
# Rename and fill in required values

services:
  multica-server:
    image: multica/server:latest
    ports:
      - "3000:3000"
    environment:
      DATABASE_URL: postgres://user:password@db:5432/multica
      SECRET_KEY: YOUR_SECRET_KEY
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: multica
      POSTGRES_PASSWORD: YOUR_PASSWORD
      POSTGRES_DB: multica
    volumes:
      - pgdata:/var/lib/postgresql/data
volumes:
  pgdata:
```

- [ ] **Step 3: 清理旧的 .gitkeep**

```bash
rm skills/*/.gitkeep
rm config/codex/.gitkeep
rm config/multica/.gitkeep
```

---

### Task 9: 删除旧目录，最终验证

**Files:**
- Delete: `mcp_servers/`（整个目录）

- [ ] **Step 1: 删除旧目录**

```bash
rm -rf mcp_servers/
```

- [ ] **Step 2: 全量编译验证**

```bash
cd /Users/rvn0xsy/Documents/Git/AdversaryChefAgentTeam
cd servers/kali && go build ./... && go test ./...
cd ../asset && go build ./... && go test ./...
cd ../.. && go work sync
```

- [ ] **Step 3: Docker 构建验证**

```bash
docker compose -f docker/docker-compose.yml build --no-cache
```

- [ ] **Step 4: 最终目录结构检查**

```bash
tree -L 3 -I '.git|__pycache__|*.sum|.DS_Store' /Users/rvn0xsy/Documents/Git/AdversaryChefAgentTeam
```

期望输出:
```
├── go.work
├── pkg/mcputil/
│   ├── go.mod
│   └── mcputil.go
├── servers/
│   ├── kali/
│   │   ├── cmd/server/main.go
│   │   └── internal/
│   │       ├── job/{job,manager}_*.go
│   │       └── tools/{register,exec,query,kill}.go
│   └── asset/
│       ├── cmd/server/main.go
│       └── internal/
│           ├── models/models.go
│           ├── store/{store,memory,sqlite}.go
│           └── tools/{crud,register,project,asset,clue,credential,worklog}.go
├── skills/*/
│   └── SKILL.md
├── config/*/
│   └── *.example
├── docs/
├── docker/
└── scripts/
```
