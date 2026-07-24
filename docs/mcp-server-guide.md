---
title: MCP Server 构建指南
created: 2026-07-24
---

# MCP Server 构建指南

本项目所有 MCP Server 使用 Go 语言，基于 [modelcontextprotocol/go-sdk](https://github.com/modelcontextprotocol/go-sdk) v1.6.1，通过 HTTP/SSE transport 对外暴露。

## 约定

- 所有 MCP Server 放在 `mcp_servers/` 目录下，每个 server 一个独立子目录
- Go module 路径格式：`adversarychef/<server-name>`
- 每个 server 是独立的 Go module，互不依赖
- 入口在 `cmd/server/main.go`，业务代码在 `internal/`
- 传输层统一使用 `mcp.NewSSEHandler`（HTTP/SSE），不使用 stdio

## 标准目录结构

```
mcp_servers/<server-name>/
├── cmd/
│   └── server/
│       └── main.go           # 入口：创建 server、注册 tools、启动 HTTP
├── internal/
│   ├── models/
│   │   └── models.go         # 数据模型（仅限需要持久化的 server）
│   ├── store/
│   │   ├── store.go          # Store 接口定义
│   │   └── memory.go         # 内存实现（开发/单机阶段）
│   ├── tools/
│   │   ├── register.go       # RegisterAll(server, deps...) — 注册所有 tools
│   │   ├── xxxx.go           # 按领域拆分的 tool handler 文件
│   │   └── ...
│   └── executor/
│       └── executor.go       # 核心执行逻辑（仅限需要的 server）
├── migrations/
│   └── .gitkeep              # 后续数据库迁移脚本
├── go.mod
└── go.sum
```

### 什么时候需要 `models/` + `store/`

如果 MCP Server 需要管理有状态数据（CRUD），引入 `Store` 接口 + `MemoryStore`：

- `store.go` 定义 `Store` interface，所有方法通过接口调用
- `memory.go` 提供基于 `sync.RWMutex` + `map` 的线程安全内存实现
- 后续可替换为 Postgres 实现，tool handler 代码无需修改

参考：`asset-mcp`。

### 什么时候只需要 `executor/`

如果 MCP Server 只做无状态的命令转发（如 `kali-mcp`），不需要 `models/` 和 `store/`。核心逻辑放在 `executor/` 中。

参考：`kali-mcp`。

## 从零构建一个 MCP Server

### Step 1：创建目录和 go module

```bash
mkdir -p mcp_servers/<name>/cmd/server
mkdir -p mcp_servers/<name>/internal/tools
cd mcp_servers/<name>
go mod init adversarychef/<name>
go get github.com/modelcontextprotocol/go-sdk@v1.6.1
```

### Step 2：编写 main.go

```go
package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"adversarychef/<name>/internal/tools"
)

var (
	host = flag.String("host", "0.0.0.0", "host to listen on")
	port = flag.Int("port", 8080, "port to listen on")
)

func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "<Description of this MCP server>.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()

	addr := fmt.Sprintf("%s:%d", *host, *port)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "<name>",
		Version: "0.1.0",
	}, nil)

	tools.Register(server) // 或 tools.RegisterAll(server, deps...)

	handler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		return server
	}, nil)

	log.Printf("<Name> MCP Server listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
```

### Step 3：定义 Tool

每个 tool 由三个要素组成：

1. **参数 struct**：定义输入，`jsonschema` tag 会被 go-sdk 自动转换为 JSON Schema 暴露给调用方
2. **`mcp.Tool` 定义**：Name、Description
3. **Handler 函数**：签名 `func(ctx context.Context, req *mcp.CallToolRequest, params XxxParams) (*mcp.CallToolResult, any, error)`

```go
package tools

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DoSomethingParams 定义参数。
type DoSomethingParams struct {
	Name    string `json:"name" jsonschema:"操作目标的名称"`
	Timeout int    `json:"timeout,omitempty" jsonschema:"超时秒数，默认 300"`
}

func Register(server *mcp.Server) {
	mcp.AddTool(server, &mcp.Tool{
		Name:        "do_something",
		Description: "执行某个操作。描述要清晰，Agent 会根据描述决定何时调用此 tool。",
	}, handleDoSomething)
}

func handleDoSomething(ctx context.Context, req *mcp.CallToolRequest, params DoSomethingParams) (*mcp.CallToolResult, any, error) {
	// 业务逻辑
	result := fmt.Sprintf("处理完成: %s", params.Name)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: result},
		},
	}, nil, nil
}
```

### Step 4：编译和运行

```bash
cd mcp_servers/<name>
go mod tidy
go build ./cmd/server/

# 运行
./server -host 0.0.0.0 -port 8080
```

## Store 接口模式（有状态 MCP）

当 MCP Server 需要管理持久化数据时，使用接口隔离存储实现：

**`internal/store/store.go`** — 接口定义：

```go
package store

import "adversarychef/<name>/internal/models"

type Store interface {
	ListProjects() ([]models.Project, error)
	GetProject(id string) (*models.Project, error)
	CreateProject(p *models.Project) error
	UpdateProject(p *models.Project) error
	DeleteProject(id string) error
	// ... 其他实体
}
```

**`internal/store/memory.go`** — 内存实现：

```go
type MemoryStore struct {
	mu       sync.RWMutex
	projects map[string]*models.Project
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		projects: make(map[string]*models.Project),
	}
}
```

Tool handler 通过依赖注入获取 Store：

```go
func RegisterAll(server *mcp.Server, s store.Store) {
	mcp.AddTool(server, &mcp.Tool{Name: "list_projects", ...},
		func(ctx context.Context, req *mcp.CallToolRequest, params ListParams) (*mcp.CallToolResult, any, error) {
			projects, err := s.ListProjects()
			// ...
		},
	)
}
```

后续切换到 Postgres 只需实现新的 `Store`，main.go 中替换 `NewMemoryStore()` → `NewPostgresStore(dsn)`，tool handler 代码不变。

## 关键要点

**Transport 统一用 SSE**

所有 MCP Server 都使用 `mcp.NewSSEHandler`，不通过 stdio 通信。这是因为 MCP Server 运行在独立容器中，与 Codex Agent 跨容器调用。HTTP/SSE 是唯一可行的 transport。

**参数用 jsonschema tag 描述**

go-sdk 会自动将 struct 的 `jsonschema` tag 生成为 MCP tool 的 `inputSchema`。这是 Agent 理解 tool 参数结构的唯一方式，务必写清楚每个字段的含义。

**Tool Description 决定 Agent 行为**

MCP tool 的 `Description` 字段是 Agent 决定调用时机的关键依据。描述要准确、具体，说明 tool 的功能和使用场景。错误的描述会导致 Agent 在错误的时机调用或完全不调用。

**错误处理**

Tool handler 返回 error 时，MCP 框架会将其作为 tool 执行失败返回给 Agent。对于可恢复的错误（如资源不存在），建议在 `TextContent` 中返回错误信息而不是直接 return error，这样 Agent 可以阅读错误内容自行决策。

**端口规划**

当前端口分配：

| Server     | 默认端口 |
|------------|---------|
| kali-mcp   | 8080    |
| asset-mcp  | 8081    |

新增 MCP Server 时按序递增即可。

## 示例代码参考

| Server     | 特点                        | 参考重点                              |
|------------|----------------------------|---------------------------------------|
| kali-mcp   | 无状态、单一 tool            | 最简模式：main + 1 个 tool + executor  |
| asset-mcp  | 有状态、多 tool、Store 接口   | CRUD 模式：models + store + 多 tool   |
