# ACA Scheduler + nexus-mcp v2 — 设计规格

> **日期**: 2026-07-25  
> **状态**: draft  
> **范围**: 用 goose 替换 Multica daemon，自建轻量任务调度器，升级 nexus-mcp 为图数据模型

---

## 1. 架构总览

```
┌──────────────────────────────────────────────────────────┐
│                   acasched（调度器）                       │
│                                                          │
│  internal/scheduler/                                     │
│    dispatcher.go  ← 轮询 pending tasks → spawn goose     │
│    trigger.go     ← 子任务完成后唤醒父任务                  │
│    reaper.go      ← 超时检测 + 重试                        │
│                                                          │
│  internal/goose/                                         │
│    runner.go      ← goose CLI 子进程管理                  │
│                                                          │
│  internal/store/                                         │
│    sqlite.go      ← tasks + projects 持久化               │
│                                                          │
│  internal/api/                                           │
│    tasks.go       ← CRUD /tasks                          │
└──────────┬───────────────────────────────────────────────┘
           │ spawn goose run --instructions ... --with-streamable-http-extension ...
           ▼
┌──────────────────────┐  ┌──────────────┐  ┌───────────────┐
│   goose (AC-Echo)    │  │  nexus-mcp   │  │   kali-mcp     │
│   goose (AC-Breach)  │  │  :8081       │  │   :8080        │
│   goose (AC-Ghost)   │  │  Operation   │  │   nmap, sqlmap │
│   goose (AC-Path)    │  │  + Reasoning │  │   gobuster ... │
│   goose (AC-Quill)   │  │  Graph       │  └───────────────┘
│   goose (AC-Sup)     │  └──────────────┘
│   goose (AC-Strats)  │  ┌──────────────┐
│   goose (AC-Forge)   │  │  mythic-mcp   │
└──────────────────────┘  │  :8082        │
                          │  callbacks,   │
                          │  tasks, files │
                          └──────────────┘
```

**核心设计决策**：

| 决策 | 选择 | 原因 |
|------|------|------|
| Agent 运行时 | goose CLI（非交互式） | `--instructions` + `--with-streamable-http-extension` + `--max-turns` + `--no-session` 完整支持编程化调用 |
| 任务调度 | 自建 acasched（参考 Multica 机制） | pending→dispatched→running→done 状态机 + 子任务完成回触父任务 |
| Agent 协作 | Supervisor 通过 nexus-mcp 动态创建子任务 | 比固定 pipeline 灵活，比 Multica Squad @mention 简洁 |
| 数据模型 | nexus-mcp 新增 Operation Graph + Reasoning Graph | Agent 间共享记忆的唯一载体，需要结构化查询 |
| 项目隔离 | nexus-mcp session binding（首次调用绑死 project_id） | 不需要 MCP config 改动，每个 goose 进程天然隔离 |
| 无 Multica | 完全移除 Multica 依赖 | 消除 daemon/Codex/Squad @mention 摩擦 |

---

## 2. acasched — 任务调度器

### 2.1 数据模型

```sql
CREATE TABLE projects (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    description TEXT DEFAULT '',
    status      TEXT DEFAULT 'active',
    created_at  TEXT NOT NULL
);

CREATE TABLE tasks (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    parent_id     TEXT,              -- 父任务 ID（子任务完成后回触发父任务）
    agent         TEXT NOT NULL,     -- "supervisor" | "echo" | "breach" | "ghost" | "path" | "forge" | "quill" | "strategist"
    status        TEXT NOT NULL DEFAULT 'pending',  -- pending | dispatched | running | done | failed | timeout | skipped
    title         TEXT NOT NULL,
    description   TEXT NOT NULL,     -- 给 Agent 的任务指令
    result        TEXT DEFAULT '',   -- Agent 完成后的回复/摘要
    error         TEXT DEFAULT '',   -- 失败原因
    created_by    TEXT NOT NULL,     -- "human" | agent name
    max_turns     INTEGER DEFAULT 40,
    timeout_secs  INTEGER DEFAULT 1800,
    retry_count   INTEGER DEFAULT 1,
    attempt       INTEGER DEFAULT 0,
    created_at    TEXT NOT NULL,
    dispatched_at TEXT,
    completed_at  TEXT
);

CREATE INDEX idx_tasks_status ON tasks(project_id, status);
CREATE INDEX idx_tasks_parent ON tasks(parent_id);
```

### 2.2 状态机

```
              ┌──────────┐
              │ pending  │
              └────┬─────┘
                   │ dispatcher 选中
              ┌────▼─────┐
              │dispatched│
              └────┬─────┘
                   │ goose 进程启动成功
              ┌────▼─────┐
              │ running  │
              └────┬─────┘
         ┌─────────┼──────────┐
         │         │          │
    ┌────▼───┐ ┌───▼────┐ ┌──▼──────┐
    │  done  │ │ failed │ │ timeout │
    └────┬───┘ └───┬────┘ └──┬──────┘
         │         │         │
         │    ┌────▼────┐    │
         │    │retrying │◄───┘ (retry_count > attempt)
         │    └────┬────┘
         │    ┌────┼─────┐
         │    ▼         ▼
         │ ┌──────┐ ┌───────┐
         │ │ done │ │failed │  (重试耗尽)
         │ └──────┘ └───────┘
         │
    ┌────▼─────────────────────┐
    │ 检查父任务的兄弟任务状态    │
    │ 全部 done/failed →        │
    │ 父任务重新 dispatch        │
    └──────────────────────────┘
```

### 2.3 核心流程

**Dispatcher Loop**（调度器主循环）：

```
每 2 秒:
  1. 查询 status=pending 的 task，按 created_at ASC
  2. 跳过 parent_id 不为空但父任务未完成的
  3. 标记 status=dispatched
  4. 按 agent 选择对应的 prompt 文件（prompts/{agent}.md）
  5. fork+exec goose run → 标记 status=running
  6. 等待 goose 退出 → 解析结果 → 标记 done/failed/timeout
  7. 如果 done: 调用 trigger.go 检查父任务
```

**子任务回触发父任务**（trigger.go）：

```
task 完成 (status=done) →
  parent := findParent(task.parent_id)
  if parent == nil → 结束
  
  siblings := findAllChildTasks(parent.id)
  allDone := all(siblings, status in {done, failed, skipped})
  
  if allDone:
    parent.status = 'pending'
    parent.description += 注入子任务结果摘要
```

### 2.4 HTTP API

| Method | Path | 用途 |
|--------|------|------|
| `POST /api/projects` | 创建项目 |
| `GET /api/projects/:id` | 查询项目 |
| `POST /api/tasks` | 创建任务 |
| `GET /api/tasks?project_id=X&status=Y` | 查询任务列表 |
| `GET /api/tasks/:id` | 查询单个任务 |
| `PATCH /api/tasks/:id` | 更新任务状态/结果 |

### 2.5 Agent 如何创建子任务

AC-Supervisor 通过 nexus-mcp 的 `scheduler_create_task` tool：

```json
// agent → nexus-mcp tool call
// 创建 task（status=pending），调度器异步 pick up 并 dispatch
{
  "tool": "scheduler_create_task",
  "params": {
    "parent_id": "task_001",
    "agent": "echo",
    "title": "侦察 target.com",
    "description": "对 target.com 执行外部侦察：端口扫描、web 探测、JS 分析。将结果写入 nexus-mcp。",
    "max_turns": 40
  }
}
// nexus-mcp 内部 → HTTP POST http://acasched:8080/api/tasks → 返回 task_id
```

Agent 标记自己完成：

```json
// goose 进程在结束时调用（由 Agent prompt 要求）
{
  "tool": "scheduler_complete_task",
  "params": {
    "task_id": "task_002",
    "result": "侦察完成。发现 3 个服务（Apache :80, SSH :22, MySQL :3306），12 个端点，2 个漏洞线索。"
  }
}
```

---

## 3. nexus-mcp 数据模型升级

### 3.1 保留的旧表（不动）

- `projects` — 项目元信息
- `assets` — 向后兼容（Agent 可继续用 `create_asset`）
- `clues` — 向后兼容
- `credentials` — 向后兼容
- `worklogs` — 向后兼容

### 3.2 新增：Operation Graph 节点表

```sql
-- Host 节点（IP/域名/OS）
CREATE TABLE host_nodes (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    ips           TEXT NOT NULL DEFAULT '[]',       -- JSON array
    hostname      TEXT DEFAULT '',
    os            TEXT DEFAULT '',
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at    TEXT NOT NULL
);

-- Service 节点（端口上的服务）
CREATE TABLE service_nodes (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    host_id       TEXT NOT NULL,
    port          INTEGER NOT NULL,
    protocol      TEXT NOT NULL DEFAULT 'tcp',
    name          TEXT NOT NULL,                    -- http / ssh / mysql / ...
    version       TEXT DEFAULT '',
    banner        TEXT DEFAULT '',
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at    TEXT NOT NULL
);

-- Endpoint 节点（HTTP 端点）
CREATE TABLE endpoint_nodes (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    service_id    TEXT NOT NULL,
    url           TEXT NOT NULL,
    method        TEXT NOT NULL DEFAULT 'GET',
    parameters    TEXT NOT NULL DEFAULT '[]',       -- JSON array
    status        TEXT NOT NULL DEFAULT 'discovered', -- discovered | testing | tested | skipped
    discovered_by TEXT DEFAULT '',
    tested_by     TEXT DEFAULT '',
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at    TEXT NOT NULL
);
CREATE UNIQUE INDEX uq_endpoint_url_method ON endpoint_nodes(project_id, url, method);

-- Session 节点（Web session / C2 回调 / SSH 隧道）
CREATE TABLE session_nodes (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    asset_id      TEXT DEFAULT '',                  -- 关联的 Host/Service/Endpoint
    created_by    TEXT NOT NULL,                    -- Agent name
    session_type  TEXT NOT NULL,                    -- web | c2_shell | ssh | tunnel
    url           TEXT DEFAULT '',
    cookies       TEXT DEFAULT '',                  -- JSON
    token_value   TEXT DEFAULT '',
    metadata      TEXT DEFAULT '',                  -- JSON
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at    TEXT NOT NULL,
    expires_at    TEXT DEFAULT ''
);
```

### 3.3 新增：Reasoning Graph 节点表

```sql
-- Evidence 节点（具体证据：扫描结果、工具输出）
CREATE TABLE evidence_nodes (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    label         TEXT NOT NULL,                    -- e.g. "nmap scan result for :80"
    source        TEXT NOT NULL,                    -- "kali-mcp:nmap" | "kali-mcp:sqlmap"
    content_ref   TEXT DEFAULT '',                  -- 引用 Artifact content_hash 或 job_id
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at    TEXT NOT NULL
);

-- Hypothesis 节点（推测）
CREATE TABLE hypothesis_nodes (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    label         TEXT NOT NULL,                    -- e.g. "可能存在 SQL 注入"
    confidence    REAL NOT NULL DEFAULT 0.0,
    status        TEXT NOT NULL DEFAULT 'proposed', -- proposed | testing | confirmed | rejected
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at    TEXT NOT NULL
);

-- Vulnerability 节点（已确认漏洞）
CREATE TABLE vulnerability_nodes (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    title         TEXT NOT NULL,
    cve           TEXT DEFAULT '',
    severity      TEXT NOT NULL DEFAULT 'medium',  -- critical | high | medium | low
    cvss          REAL DEFAULT 0.0,
    description   TEXT DEFAULT '',
    remediation   TEXT DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'open',     -- open | fixed | accepted_risk
    evidence_refs TEXT NOT NULL,                    -- 必须非空！
    created_at    TEXT NOT NULL
);
```

### 3.4 新增：通用图边表

```sql
CREATE TABLE graph_edges (
    id            TEXT PRIMARY KEY,
    project_id    TEXT NOT NULL,
    from_id       TEXT NOT NULL,
    to_id         TEXT NOT NULL,
    edge_type     TEXT NOT NULL,                    -- has_port | exposes_endpoint | supports | contradicts | confirms | exploited_by | ...
    evidence_refs TEXT NOT NULL DEFAULT '[]',
    created_at    TEXT NOT NULL
);
CREATE INDEX idx_edges_from ON graph_edges(project_id, from_id);
CREATE INDEX idx_edges_to   ON graph_edges(project_id, to_id);
```

### 3.5 Edge Types 预定义

**Operation Graph**:

| edge_type | from → to | 含义 |
|-----------|-----------|------|
| `has_port` | Host → Service | IP 上有此端口/服务 |
| `exposes_endpoint` | Service → Endpoint | 服务暴露此 HTTP 端点 |
| `authenticated_by` | Session → Credential | 会话使用的凭据 |
| `session_on` | Session → Host/Service | 会话建立在此主机/服务上 |

**Reasoning Graph**:

| edge_type | from → to | 含义 |
|-----------|-----------|------|
| `supports` | Evidence → Hypothesis | 证据支持此推测 |
| `contradicts` | Evidence → Hypothesis | 证据反驳此推测 |
| `confirms` | Hypothesis → Vulnerability | 推测确认为漏洞 |
| `exploited_by` | Vulnerability → Session | 漏洞利用获得的会话 |

**跨图**:

| edge_type | from → to | 含义 |
|-----------|-----------|------|
| `observed_on` | Evidence → Host/Service/Endpoint | 证据从此实体采集 |

### 3.6 新增 MCP Tools

```go
// Operation Graph tools (10)
"host_create"     "host_list"     "host_get"
"service_create"  "service_list"  "service_get"
"endpoint_create" "endpoint_list" "endpoint_get" "endpoint_update_status" "find_untested_endpoints"
"session_create"  "session_list"  "session_get"  "find_sessions"

// Reasoning Graph tools (8)
"evidence_create"     "evidence_list"
"hypothesis_create"   "hypothesis_list"   "hypothesis_update"
"vulnerability_create" "vulnerability_list" "vulnerability_update"

// Graph tools (4)
"edge_create"  "edge_list"  "edge_delete"
"graph_query"  -- 从节点出发 BFS 2-hop 遍历，返回 Subgraph JSON
"graph_trace"  -- 反向追溯: Vulnerability → Hypothesis → Evidence → 源

// Scheduler bridge tools (2)
"scheduler_create_task"     -- Agent 创建子任务（nexus-mcp 转发到 acasched HTTP API）
"scheduler_complete_task"   -- Agent 标记自己完成

// 总计: 24 个新增 tools (旧 tools 保留不动，共 ~52 tools)
```

### 3.7 Store 接口

```go
type Store interface {
    // 旧接口（不动）
    ProjectStore    // projects CRUD
    AssetStore      // assets CRUD
    ClueStore       // clues CRUD
    CredentialStore // credentials CRUD
    WorkLogStore    // worklogs CRUD

    // 新增
    OperationStore  // Host, Service, Endpoint, Session CRUD + graph_query
    ReasoningStore  // Evidence, Hypothesis, Vulnerability CRUD
    GraphEdgeStore  // graph_edges CRUD
}

// SchedulerBridge 是独立组件（非 Store），直接 HTTP 转发到 acasched
type SchedulerBridge struct {
    schedulerURL string
    httpClient   *http.Client
}
```

### 3.8 Scheduler Bridge 实现

nexus-mcp 不自己管理任务生命周期，而是转发到 acasched：

```go
// tools/scheduler.go
func registerSchedulerBridge(server *mcp.Server, bridge *SchedulerBridge) {
    mcputil.AddLoggingTool(server, &mcp.Tool{
        Name: "scheduler_create_task",
        Description: "Create a sub-task for another agent to execute",
    }, func(ctx context.Context, req *mcp.CallToolRequest, params SchedulerCreateParams) (*mcp.CallToolResult, any, error) {
        task, err := bridge.CreateTask(ctx, params)
        if err != nil {
            return mcputil.TextResult("create task failed: " + err.Error()), nil, nil
        }
        b, _ := json.Marshal(task)
        return mcputil.TextResult(string(b)), nil, nil
    })

    mcputil.AddLoggingTool(server, &mcp.Tool{
        Name: "scheduler_complete_task",
        Description: "Mark your own task as complete with a result summary",
    }, func(ctx context.Context, req *mcp.CallToolRequest, params SchedulerCompleteParams) (*mcp.CallToolResult, any, error) {
        if err := bridge.CompleteTask(ctx, params.TaskID, params.Result); err != nil {
            return mcputil.TextResult("complete task failed: " + err.Error()), nil, nil
        }
        return mcputil.TextResult("task marked done"), nil, nil
    })
}

type SchedulerBridge struct {
    schedulerURL string  // http://acasched:8080
    httpClient   *http.Client
}
```

---

## 4. Goose Executor 集成

### 4.1 调度器如何调 goose

```go
// internal/goose/runner.go
type Runner struct {
    promptsDir string   // prompts/
    workDir    string
    nexusMCP   string   // http://127.0.0.1:8081
    kaliMCP    string   // http://127.0.0.1:8080
    mythicMCP  string   // http://127.0.0.1:8082
}

func (r *Runner) Execute(ctx context.Context, task *Task) (*TaskResult, error) {
    // 1. 准备 Agent prompt（注入 project_id + task context）
    prompt := r.buildPrompt(task)

    // 2. 写入临时 instructions 文件
    tmpFile, _ := writeTempFile(prompt)
    defer os.Remove(tmpFile)

    // 3. 构建 goose 命令
    cmd := exec.CommandContext(ctx, "goose", "run",
        "--instructions", tmpFile,
        "--text", task.Description,
        "--with-streamable-http-extension", r.nexusMCP,
        "--with-streamable-http-extension", r.kaliMCP,
        "--with-streamable-http-extension", r.mythicMCP,
        "--max-turns", strconv.Itoa(task.MaxTurns),
        "--no-session",
        "--output-format", "stream-json",
        "--no-profile",
    )
    cmd.Dir = r.workDir

    // 4. 执行并解析结果
    return r.runAndParse(cmd, task)
}

// buildPrompt 注入项目绑定 + task 上下文
func (r *Runner) buildPrompt(task *Task) string {
    content := r.loadPromptFile(task.Agent)
    return fmt.Sprintf(`## Session Binding
project_id: %s
task_id: %s
scheduler_url: http://acasched:8080

## Task Lifecycle
- Use scheduler_create_task to delegate work to other agents
- Use scheduler_complete_task to mark yourself done: {"task_id":"%s","result":"<your summary>"}
- Do NOT exit without calling scheduler_complete_task

---
%s`, task.ProjectID, task.ID, task.ID, content)
}
```

### 4.2 stream-json 结果解析

```go
type StreamLine struct {
    Type    string          `json:"type"`    // "assistant" | "tool_call" | "tool_result" | "result"
    Content json.RawMessage `json:"content"`
}

func (r *Runner) parseStreamOutput(output string) (*TaskResult, error) {
    var summary string
    scanner := bufio.NewScanner(strings.NewReader(output))
    for scanner.Scan() {
        var line StreamLine
        if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
            continue
        }
        if line.Type == "assistant" {
            // 累积 Agent 的文本回复作为摘要
            summary += extractText(line.Content)
        }
    }
    return &TaskResult{
        Status:  "done",
        Summary: truncate(summary, 2000),
        Output:  output,
    }, nil
}
```

### 4.3 超时和重试

```go
func (r *Runner) runAndParse(cmd *exec.Cmd, task *Task) (*TaskResult, error) {
    for attempt := 0; attempt <= task.RetryCount; attempt++ {
        result, err := r.runOnce(cmd, task)
        if err == nil {
            return result, nil
        }
        
        if attempt < task.RetryCount {
            log.Printf("task %s attempt %d/%d failed: %v, retrying...",
                task.ID, attempt+1, task.RetryCount+1, err)
            // 重试时追加提示
            task.Description += fmt.Sprintf("\n\n[RETRY %d/%d] Previous attempt failed: %v. Try a different approach.",
                attempt+1, task.RetryCount+1, err)
        }
    }
    return &TaskResult{Status: "failed", Error: fmt.Sprintf("exhausted %d retries", task.RetryCount)}, nil
}
```

---

## 5. 项目隔离

### 5.1 nexus-mcp Session Binding

```go
// mcputil 中新增 session 中间件
type SessionMap struct {
    mu       sync.RWMutex
    sessions map[string]*SessionBinding  // key: MCP session ID
}

type SessionBinding struct {
    projectID string
    bound     bool
    boundAt   time.Time
}

func (m *SessionMap) GetOrBind(sessionID string, callerProjectID string) (string, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    s, exists := m.sessions[sessionID]
    if !exists {
        s = &SessionBinding{projectID: callerProjectID, bound: true, boundAt: time.Now()}
        m.sessions[sessionID] = s
        return callerProjectID, nil
    }

    if s.projectID != callerProjectID {
        return "", fmt.Errorf("session %s bound to project %s, rejected %s",
            sessionID, s.projectID, callerProjectID)
    }
    return s.projectID, nil
}
```

### 5.2 Tool Handler 中的使用

```go
// 每个 tool handler 调用前
func requireProjectScope(ctx context.Context, reqProjectID string) (string, error) {
    sessionID := mcp.SessionIDFromContext(ctx)
    return sessionMap.GetOrBind(sessionID, reqProjectID)
}
```

每个 goose 进程 = 独立 MCP session = 独立的 project binding。进程结束 → session 自动过期（MCP session timeout 5 分钟）。

### 5.3 Agent Prompt 中的体现

Agent 不需要手动传 project_id——prompt 中已经注入：

```markdown
## Session Binding
project_id: proj_001
task_id: task_002
```

Agent 在 nexus-mcp tool calls 中带上 project_id 参数（首调用会 bind，后续自动校验）。

---

## 6. 目录结构（升级后）

```
AdversaryChefAgentTeam/
├── cmd/
│   └── acasched/
│       └── main.go                 ← 调度器入口
│
├── internal/
│   ├── scheduler/
│   │   ├── dispatcher.go           ← 主循环
│   │   ├── trigger.go              ← 子任务回触
│   │   ├── reaper.go               ← 超时清理
│   │   └── lifecycle.go            ← 状态机
│   │
│   ├── goose/
│   │   ├── runner.go               ← goose CLI 子进程管理
│   │   └── parser.go               ← stream-json 解析
│   │
│   ├── store/
│   │   └── sqlite.go               ← tasks + projects 持久化
│   │
│   └── api/
│       ├── server.go               ← HTTP handler
│       ├── tasks.go                ← /api/tasks
│       └── projects.go             ← /api/projects
│
├── servers/
│   ├── nexus/                      ← nexus-mcp (升级)
│   │   └── internal/
│   │       ├── models/
│   │       │   ├── models.go        ← 旧模型（不动）
│   │       │   └── graph.go         ← 新增：图节点模型
│   │       ├── store/
│   │       │   ├── store.go         ← 拆分 Store 接口
│   │       │   ├── operation.go     ← 新增：OperationStore 实现
│   │       │   ├── reasoning.go     ← 新增：ReasoningStore 实现
│   │       │   ├── edges.go         ← 新增：GraphEdgeStore 实现
│   │       │   ├── sqlite.go        ← 扩展：向现有 SQLiteStore 嵌入新 store
│   │       │   └── session.go       ← 新增：session binding
│   │       └── tools/
│   │           ├── ...              ← 旧 tools (不动)
│   │           ├── graph.go         ← 新增：图节点 tools
│   │           ├── edges.go         ← 新增：边 tools
│   │           ├── query.go         ← 新增：graph_query / graph_trace
│   │           └── scheduler.go     ← 新增：scheduler bridge
│   │
│   ├── kali/                       ← 不变
│   └── mythic/                     ← 不变
│
├── prompts/                        ← 不变（Agent prompt 文件）
├── skills/                         ← 不变
├── pkg/mcputil/                    ← 扩展：session binding, scheduler bridge
└── go.work                         ← 更新：添加 cmd/acasched 模块
```

---

## 7. 实施计划

### Phase 1: nexus-mcp 数据模型升级（2 天）

| 任务 | 估时 |
|------|------|
| 新增图节点模型 (`models/graph.go`) | 0.5h |
| 拆分 Store 接口 + 新增 OperationStore | 1h |
| 新增 ReasoningStore | 0.5h |
| 新增 GraphEdgeStore + graph_query/graph_trace | 2h |
| 新增 SchedulerBridge | 1h |
| 新增 24 个 MCP tools | 3h |
| 新增 Session Binding 中间件 | 1h |
| 单元测试 | 3h |

### Phase 2: acasched 调度器（2 天）

| 任务 | 估时 |
|------|------|
| SQLite tasks/projects schema | 0.5h |
| Task lifecycle + 状态机 | 2h |
| Dispatcher loop | 2h |
| Trigger (子任务回触父任务) | 1.5h |
| Reaper (超时检测) | 0.5h |
| Goose executor 集成 | 1.5h |
| HTTP API (tasks + projects) | 1.5h |
| 集成测试 | 2h |

### Phase 3: Agent Prompt 适配（1 天）

| 任务 | 估时 |
|------|------|
| AC-Supervisor prompt 重写 (调度器模式) | 1h |
| 各 Agent prompt 增加 project binding + task lifecycle | 1h |
| Prompt 模板化 (注入 project_id/task_id) | 0.5h |
| 增加 scheduler_create_task / scheduler_complete_task 使用指引 | 1h |
| 端到端测试 | 3h |

---

## 8. 风险与缓解

| 风险 | 概率 | 缓解 |
|------|------|------|
| goose stream-json 格式不稳定 | 中 | 先手动跑一次 `goose run --output-format stream-json` 确认格式；parser 做 defensive parsing |
| Scheduler + goose 子进程管理的内存泄漏 | 低 | 每个 goose 进程有 `context.WithTimeout`；reaper 定期清理僵尸进程 |
| nexus-mcp session binding 在 HTTP/SSE 模式下的 session 生命周期不确定 | 中 | Phase 1 最后加集成测试验证；如果 HTTP session 不稳定，回退到请求级 `X-Project-ID` header |
| Agent 不调 `scheduler_complete_task` | 高 | Prompt 中硬性要求 + reaper 超时自动标记 failed |
| 两个 Agent 同时修改同一 nexus-mcp 记录 | 低 | SQLite WAL 模式保证并发安全；单项目串行 Agent 模型下不太可能并发冲突 |
