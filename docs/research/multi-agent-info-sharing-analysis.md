# 多 Agent 渗透测试小队信息共享机制 — 研究报告

> **研究日期**: 2026-07-25
> **研究对象**: AdversaryChefAgentTeam（以下简称 ACA）+ Clinkz 对比分析
> **核心问题**: 8 个专精 Agent 如何在渗透测试全生命周期中共享资产、发现、凭据等关键信息？

---

## 目录

1. [当前 ACA 架构的共享机制](#1-当前-aca-架构的共享机制)
2. [当前机制的差距分析](#2-当前机制的差距分析)
3. [业界参考：Clinkz 的信息共享设计](#3-业界参考clinkz-的信息共享设计)
4. [改进方案：三层信息共享架构](#4-改进方案三层信息共享架构)
5. [推荐实施路线](#5-推荐实施路线)

---

## 1. 当前 ACA 架构的共享机制

### 1.1 架构总览

```
AC-Supervisor (Multica Issue 编排)
    │
    │ project_id 透传
    ▼
┌──────────┬──────────┬──────────┬──────────┬──────────┬──────────┬──────────┐
│Strategist│  Echo    │ Breach   │  Ghost   │  Path    │  Forge   │  Quill   │
│  规划者   │ 侦察兵   │  漏洞手   │ C2操作员 │ 内网先锋 │ 基建手   │ 报告员   │
└────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬─────┴────┬─────┘
     │          │          │          │          │          │          │
     └──────────┴──────────┴──────────┴──────────┴──────────┴──────────┘
                                    │
                            asset-mcp (HTTP/SSE :8081)
                           ┌────────┼────────┐
                           │ Projects │ Assets  │
                           │ Clues    │ Creds   │
                           │ WorkLogs          │
                           └───────────────────┘
```

### 1.2 当前数据模型

```go
// Project — 渗透项目
type Project struct {
    ID, Name, Description, Status string
}

// Asset — 目标资产（IP、域名、技术栈、授权范围）
type Asset struct {
    ID, ProjectID, Name string
    IPs, Domains, TechStack []string
    Scope, Description string
}

// Clue — 发现/线索（漏洞、信息泄露、配置缺陷）
type Clue struct {
    ID, ProjectID, Title, Content string
    Type   string  // vulnerability / info_disclosure / misconfig
    Status string  // open / confirmed / false_positive / resolved
}

// Credential — 凭据（SSH密钥、密码、API Key、Token）
type Credential struct {
    ID, ProjectID, AssetID string
    CredentialType string  // ssh_key / password / api_key / token
    Label, Value, ExpiresAt, Notes string
}

// WorkLog — 操作日志
type WorkLog struct {
    ID, ProjectID, Title, Content string
}
```

### 1.3 当前共享流程

| 共享内容 | 生产者 → 消费者 | 机制 |
|---------|----------------|------|
| **资产** (IP/域名/端口) | AC-Echo → AC-Breach/Ghost/Path | asset-mcp CRUD，consumer 主动查询 |
| **漏洞线索** (Clue) | AC-Echo → AC-Breach | `create_clue` → AC-Breach `list_clues` |
| **凭据** (密码/密钥) | AC-Path/Ghost → AC-Ghost/Path | `create_credential` → `list_credentials` |
| **任务指令** | Supervisor → 各 Agent | Multica Issue 系统 |
| **操作日志** (WorkLog) | 各 Agent → AC-Quill | `create_work_log` → `list_work_logs` |
| **项目摘要** | asset-mcp → AC-Strategist/Quill | `project_summary` |

### 1.4 当前协调模式

- **project_id 透传**：Supervisor 将 project_id 嵌入每个分派的任务
- **轮询式查询**：Agent 在开始任务前先查 asset-mcp 了解上下文
- **写入后通知**：Agent 完成任务后将发现写入 asset-mcp，通过 Multica issue 向 Supervisor 汇报
- **无直接 Agent-to-Agent 通信**：所有协调通过 Supervisor 中转

---

## 2. 当前机制的差距分析

### 2.1 🔴 严重差距

| 差距 | 影响 | 案例 |
|------|------|------|
| **无实时推送/通知** | Agent 不知道其他 Agent 的新发现，必须等待 Supervisor 重新分派任务才感知 | AC-Path 在 10.0.0.5 发现域控凭据 → AC-Ghost 无法立即利用，需等 Supervisor 手动协调 |
| **无跨交互上下文保持** | 每次渗透都是"冷启动"，Agent 不记得之前渗透中学到的教训 | 上次目标的 Log4j 版本和利用链 → 本次类似目标需重新从零发现 |
| **C2 回调状态游离** | C2 回调的进程/网络/会话状态只存于 AC-Ghost 的对话上下文中，AC-Path 接手时需重复枚举 | Ghost 已 enum 过的系统信息，Path 还要重做一遍 |

### 2.2 🟡 中等差距

| 差距 | 影响 |
|------|------|
| **线索与资产弱关联** | Clue 只记录 `project_id`，不与 Asset 建立外键，无法快速查询"这个 IP 上有哪些漏洞" |
| **凭据缺少有效性标记** | 凭据有 `expires_at` 字段但无验证状态（已测试/未测试/已失效），Agent 可能使用失效凭据 |
| **无会话共享** | 登录后的 cookie/session token 无法在 Echo → Breach 之间传递，Breach 需要重新认证 |
| **无依赖跟踪** | AC-Breach 不知道 AC-Echo 的扫描是否已完成，可能出现"漏洞线索还没准备好就被要求利用"的情况 |

### 2.3 🟢 轻微差距

| 差距 | 影响 |
|------|------|
| **无操作审计** | 只有手动的 WorkLog，无法追溯"谁在什么时候创建/修改了什么" |
| **数据模型扁平** | 所有实体平铺，无层级关系（如：端口属于 IP，IP 属于域名，域名属于项目） |
| **work_log 需要手动创建** | Agent prompt 中写了"Record everything in asset-mcp"，但需要 Agent 记得调用 |

---

## 3. 业界参考：Clinkz 的信息共享设计

> Clinkz 是 GitHub 上实现最完备的 AI 渗透测试多 Agent 框架（406 commits, ~750 tests），其信息共享机制具有重要参考价值。

### 3.1 双层状态存储

```
┌─────────────────────────────────────┐
│        clinkz.db (Per-Engagement)    │
│  ┌─────────────────────────────────┐│
│  │ engagements / targets / findings ││
│  │ actions / attempts / endpoints  ││
│  │ agent_messages / sessions       ││
│  │ runbook / research_leads        ││
│  └─────────────────────────────────┘│
│         ↑ 当前任务状态               │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│   clinkz_knowledge.db (Cross-Engage)│
│  ┌─────────────────────────────────┐│
│  │ capability_facts                ││  ← 技术栈能力事实
│  │ capability_observations         ││  ← 观察记录
│  │ technique_results               ││  ← 技术利用结果
│  │ topology_reaches                ││  ← 跨服务拓扑
│  └─────────────────────────────────┘│
│         ↑ 跨渗透持久化知识            │
└─────────────────────────────────────┘
```

**设计原则**：
- **Per-Engagement DB**：存本次渗透的实时状态，所有 Agent 并发读写
- **Cross-Engagement KB**：存"去目标化"的技术知识。存储的是 `capability_fact`（如"Log4j 2.14.1 存在 JNDI 注入"），而不是"10.0.0.5 上存在 Log4j 漏洞"。**Schema 级别保证不泄露目标信息**。

### 3.2 MessageBus：Orchestrator 中介通信

```python
# 核心规则：Phase Agent 之间不能直接通信，必须通过 Orchestrator

# ✅ 合法：Agent → Orchestrator
await bus.send(AgentMessage.result("recon", "orchestrator", eid, {...}))

# ✅ 合法：Orchestrator → Agent
await bus.send(AgentMessage.task("orchestrator", "exploit", eid, {...}))

# ❌ 非法：Agent → Agent（直接抛 ValueError）
await bus.send(AgentMessage.task("recon", "exploit", eid, {...}))
```

**消息类型**：`TASK | RESULT | QUERY | RESPONSE | STATUS | ERROR`

**关键特性**：
- 每条消息有 UUID、parent_message_id（形成对话链）
- 可选择持久化到 StateStore 作为审计追踪
- 支持 `broadcast()`（系统级状态同步）
- 支持 `get_pending()`（非阻塞批量读取）
- Trace 集成：每次跨 Agent 消息自动记录到 trace.jsonl

### 3.3 会话共享（Session Handoff）

```sql
CREATE TABLE IF NOT EXISTS sessions (
    id              TEXT PRIMARY KEY,
    engagement_id   TEXT NOT NULL,
    agent           TEXT NOT NULL,      -- 创建会话的 Agent
    cookies_json    TEXT NOT NULL,      -- 认证 Cookie
    cookie_jar_path TEXT NOT NULL,      -- Netscape cookie jar 路径
    metadata_json   TEXT NOT NULL       -- 登录 URL、技术栈等元信息
);
```

**场景**：Scan Agent 登录 Web 应用 → 将 cookie 存入 sessions 表 → Exploit Agent 读取 cookie，以认证态进行漏洞利用

这在 ACA 场景中对应：
- AC-Echo 发现登录接口 → 尝试默认凭据 → 将 session 存入
- AC-Breach 读取 session → 以认证态利用注入漏洞

### 3.4 端点跟踪（Endpoint Tracking）

```sql
CREATE TABLE IF NOT EXISTS endpoints (
    id              TEXT PRIMARY KEY,
    engagement_id   TEXT NOT NULL,
    url             TEXT NOT NULL,
    method          TEXT NOT NULL,
    parameters      TEXT NOT NULL,      -- JSON
    status          TEXT NOT NULL,      -- discovered / testing / tested / skipped
    discovered_by   TEXT NOT NULL,      -- 发现者 Agent
    tested_by       TEXT,               -- 测试者 Agent
    service_type    TEXT NOT NULL,      -- http / ftp / ssh / smb / database
    port            INTEGER,
);
CREATE UNIQUE INDEX uq_endpoints_url_method ON endpoints(engagement_id, url, method);
```

**设计精妙之处**：
1. UNIQUE(engagement_id, url, method) → 自动去重，多个 Agent 扫描同一端点不会重复
2. status 状态机 → 明确标记"已发现但未测试 / 正在测试 / 已测试"，后续 Agent 能知道哪些端点还没处理
3. discovered_by / tested_by → 跨 Agent 可见的工作分工

### 3.5 跨交互学习（Capability Recall）

```
Engagement A (Vulhub Solr 8.11.0):
  ├── Discovery Engine 从源码发现 log4j-core 2.14.1
  ├── P6 OOB 确认 Log4Shell
  └── 写入 capability_fact: {tech: "log4j-core", version: "=2.14.1", 
                              capability: "log_interpolation", 
                              sink: "log4j.log_sink"}

Engagement B (新目标, 不同项目):
  ├── 扫描发现 log4j-core（但版本未知/无源码）
  ├── Capability Recall 查询 KB: "log4j-core" → 匹配到之前的 fact
  ├── 种子假设: "此目标可能存在 Log4Shell"
  └── P6 验证: 确认 → 发现漏洞（冷启动无法发现）
```

**关键防护**：
- KB 中 **绝不存储目标 IP/域名/URL**
- 跨交互共享的是"技术栈-能力映射"，不是"上次在哪发现了什么"
- 写入时通过 Schema 级别的 fence（`abstract_reaches_identity`）防止目标信息泄露

### 3.6 并发安全

- **SQLite WAL 模式**：允许并发读+写，各 Agent 可同时操作 DB
- **最大并发连接 1**（`SetMaxOpenConns(1)`）：虽用 WAL，但限制写并发避免锁冲突
- **deduplicate 机制**：upsert_target 检查 ip 是否已存在，避免重复插入

---

## 4. 改进方案：三层信息共享架构

基于 ACA 现有架构（Multica + asset-mcp + squad Agent），引入三层信息共享：

### 4.1 架构总览

```
┌──────────────────────────────────────────────────────────────┐
│                     Layer 0: Multica Issue 编排               │
│          任务分派 · 状态追踪 · 人机协作 · 审批流              │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│               Layer 1: asset-mcp 增强（实时共享层）            │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ 现有: Projects · Assets · Clues · Credentials · WorkLogs│ │
│  │ 新增: Sessions · Endpoints · Events · Dependencies      │ │
│  └─────────────────────────────────────────────────────────┘ │
│         HTTP/SSE (所有 Agent 共享同一数据源)                  │
└──────────────────────────┬───────────────────────────────────┘
                           │
┌──────────────────────────▼───────────────────────────────────┐
│          Layer 2: knowledge-mcp 新增（跨项目知识层）           │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │ capability_facts · technique_results · topology_edges    │ │
│  └─────────────────────────────────────────────────────────┘ │
│     去目标化的技术知识（不存目标 IP/域名/项目信息）            │
└──────────────────────────────────────────────────────────────┘
```

### 4.2 具体改进项

#### 改进 1：Session 共享表

**新增数据模型**：

```go
type Session struct {
    ID           string    `json:"id"`
    ProjectID    string    `json:"project_id"`
    AssetID      string    `json:"asset_id"`      // 关联到哪个资产
    CreatedBy    string    `json:"created_by"`    // 哪个 Agent 创建的（echo/breach/ghost）
    URL          string    `json:"url"`           // 登录 URL
    CookiesJSON  string    `json:"cookies_json"`  // Cookie 键值对
    TokenValue   string    `json:"token_value"`   // Bearer token / API key
    Metadata     string    `json:"metadata"`      // 认证方式、技术栈
    CreatedAt    time.Time `json:"created_at"`
    ExpiresAt    string    `json:"expires_at"`    // 预估过期时间
}
```

**新增 MCP Tools**：
- `list_sessions(project_id)` — 查询可用的认证会话
- `create_session(...)` — 保存认证会话
- `get_session(id)` — 获取特定会话详情

**受益 Agent**：AC-Echo → AC-Breach（Web 应用认证态传递）、AC-Ghost → AC-Path（C2 回调会话状态传递）

**Clinkz 的做法**（参考实现）：

```sql
CREATE TABLE sessions (
    id              TEXT PRIMARY KEY,
    engagement_id   TEXT NOT NULL,
    agent           TEXT NOT NULL,       -- 创建会话的 Agent
    cookies_json    TEXT NOT NULL,       -- {"PHPSESSID": "abc123", ...}
    cookie_jar_path TEXT NOT NULL,       -- Netscape cookie jar 文件路径
    metadata_json   TEXT NOT NULL        -- {"login_url": "...", "technology": "PHP"}
);
```

Scan Agent 成功登录后存储 cookie → Exploit Agent 读取 cookie 以认证态利用漏洞

#### 改进 2：Endpoint 表（线索的依赖前置）

**新增数据模型**：

```go
type Endpoint struct {
    ID           string    `json:"id"`
    ProjectID    string    `json:"project_id"`
    AssetID      string    `json:"asset_id"`
    URL          string    `json:"url"`
    Method       string    `json:"method"`       // GET/POST/PUT/...
    Parameters   []string  `json:"parameters"`   // 发现的参数列表
    Status       string    `json:"status"`       // discovered/testing/tested/skipped
    DiscoveredBy string    `json:"discovered_by"` // echo/breach/...
    TestedBy     string    `json:"tested_by"`
    Port         int       `json:"port"`
    ServiceType  string    `json:"service_type"` // http/ftp/ssh/smb/db/other
    Notes        string    `json:"notes"`
}
```

**状态机**：
```
discovered ──→ testing ──→ tested
                 │
                 └──→ skipped (手动跳过)
```

**新增 MCP Tools**：
- `list_endpoints(project_id, status?)` — 查询端点（可按状态过滤）
- `create_endpoint(...)` — 记录新发现的端点（URL+Method 唯一约束，自动去重）
- `update_endpoint_status(id, status, tested_by)` — 更新测试状态
- `get_untested_endpoints(project_id)` — 获取未测试端点

**受益 Agent**：AC-Echo 发现端点 → AC-Breach 查询 `status=discovered` 的端点进行利用

#### 改进 3：依赖与就绪追踪

**新增数据模型**：

```go
type Dependency struct {
    ID          string    `json:"id"`
    ProjectID   string    `json:"project_id"`
    DependsOn   string    `json:"depends_on"`    // 依赖的 clue_id / asset_id
    RequiredBy  string    `json:"required_by"`    // 被哪个 Agent 需要
    Ready       bool      `json:"ready"`          // 依赖是否已满足
}
```

**场景**：
- AC-Breach 需要 AC-Echo 完成端口扫描才能开始利用
- AC-Path 需要 AC-Ghost 提供稳定的回调才能开始内网移动

AC-Supervisor 可通过检查依赖就绪状态来决定是否分派下游任务。

#### 改进 4：Event 实时通知

**在 asset-mcp 中增加事件推送能力**：

由于 MCP HTTP/SSE 天然支持 Server-Sent Events，可在 asset-mcp 中增加 `subscribe_project_events` 端点：

```json
// SSE 事件流
event: asset_created
data: {"id": "asset_xxx", "name": "10.0.0.5-DC01", "created_by": "AC-Path"}

event: clue_updated
data: {"id": "clue_yyy", "status": "confirmed", "title": "SQLi in /api/login"}

event: credential_added
data: {"id": "cred_zzz", "label": "domain_admin", "credential_type": "password"}
```

Agent 在执行任务时订阅事件流，实时感知其他 Agent 的进展。

#### 改进 5：增强 Credential 模型

**新增字段**：

```go
type Credential struct {
    // ... 现有字段
    Verified    bool      `json:"verified"`     // 是否已验证有效
    VerifiedBy  string    `json:"verified_by"`  // 哪个 Agent 验证的
    VerifiedAt  time.Time `json:"verified_at"`  // 验证时间
    SourceAsset string    `json:"source_asset"`  // 从哪个资产获取的
    TargetAssets []string `json:"target_assets"` // 可能有效的目标资产
    RotationNeeded bool   `json:"rotation_needed"` // 是否需要轮换
}
```

**新增 MCP Tools**：
- `verify_credential(id)` — 标记凭据已验证
- `invalidate_credential(id, reason)` — 标记凭据已失效
- `search_credentials(project_id, credential_type?, verified?)` — 多条件搜索

#### 改进 6：跨项目知识库 (knowledge-mcp)

**新增 MCP Server**：独立于 asset-mcp 的知识 MCP Server

**数据模型**：

```go
type CapabilityFact struct {
    ID             string `json:"id"`
    Technology     string `json:"technology"`    // e.g. "log4j-core"
    Version        string `json:"version"`       // e.g. "=2.14.1"（精确版本约束）
    Capability     string `json:"capability"`    // e.g. "log_interpolation"
    Sink           string `json:"sink"`          // e.g. "log4j.log_sink"
    Confidence     float64 `json:"confidence"`   // 置信度（基于验证次数的衰减）
    Observations   int    `json:"observations"`  // 验证观察次数
    // 注意：NO target IP, NO project_id, NO asset_id
}

type TechniqueResult struct {
    ID             string `json:"id"`
    Technology     string `json:"technology"`
    Technique      string `json:"technique"`     // e.g. "JNDI injection via ${jndi:dns://...}"
    Outcome        string `json:"outcome"`       // success / failure / partial
    Prerequisites  []string `json:"prerequisites"` // 前置条件
    Notes          string `json:"notes"`         // 经验教训
}
```

**MCP Tools**：
- `search_capabilities(technology)` — 查询已知技术栈能力
- `record_capability(...)` — 记录确认的能力
- `search_techniques(technology, technique?)` — 查询利用技巧
- `record_technique_result(...)` — 记录利用结果

**设计原则**：
- **绝不存储目标信息**：同 Clinkz 的 Schema 级别 fence
- **跨项目共享**：所有项目共享同一个 knowledge-mcp，Agent 自动积累经验
- **置信度衰减**：长时间未验证的 fact 置信度降低

**Clinkz 的 Capability Fact 设计**（参考实现）：

```sql
-- 能力事实表：去目标化的技术栈能力记录
CREATE TABLE capability_facts (
    id                     INTEGER PRIMARY KEY,
    technology_key         TEXT NOT NULL,   -- 标准化技术标识（如 "log4j-core"）
    version_predicate      TEXT NOT NULL,   -- 版本约束（如 "=2.14.1" 或 "<2.15.0"）
    primitive_class        TEXT NOT NULL,   -- 能力分类（egress_fetch/file_read/log_interp）
    sink_shape_id          TEXT NOT NULL,   -- sink 形状标识（java.file_sink/log4j.log_sink）
    evidence_grade         TEXT NOT NULL,   -- confirmed | derived_unconfirmed | transferred | gated
    confidence             REAL NOT NULL,   -- 0..1 置信度（corroboration × recency）
    -- ✗ NO target IP, NO host, NO URL, NO project_id, NO secret
    UNIQUE(technology_key, version_predicate, primitive_class, sink_shape_id)
);

-- 观察记录表：每次验证都留下 provenance 记录
CREATE TABLE capability_observations (
    capability_fact_id     INTEGER,          -- 关联的事实 ID
    engagement_id          TEXT NOT NULL,    -- 仅用于 provenance 审计
    observed_technology    TEXT,             -- 技术栈指纹（非主机名！）
    observed_version       TEXT,             -- 精确版本号
    outcome                TEXT NOT NULL,    -- confirmed | failed_unreachable | failed_gated
    evidence_ref           TEXT,             -- 指向 ConfirmationEvidence 的 LINK（非字节拷贝）
    ...
);
```

**置信度公式**（Clinkz 设计）：
```
confidence = corroboration × recency
  corroboration = 1 - 0.5^k     （k = 不同渗透任务中的确认次数）
  recency = 0.3 + 0.7 × 0.5^(age_days / 90)   （半衰期 90 天，最低 0.3）
```
> 注意：confidence 仅作为先验种子参考，**永远不阻塞漏洞发现**。

**跨渗透学习流程示例**：
```
Engagement A：
  AC-Echo 发现 Apache + log4j-core 2.14.1
  AC-Breach P6 确认 Log4Shell
  → 写入 capability_fact: {tech:"log4j-core", ver:"=2.14.1", 
                            class:"log_interp", sink:"log4j.log_sink"}

Engagement B（完全不同的目标，无源码）：
  AC-Echo 扫描发现 log4j-core（版本未知）
  AC-Strategist 查询 knowledge-mcp → 匹配到 fact
  → 种子假设："可能存在 Log4Shell"
  AC-Breach P6 再次确认 → confidence 从 0.5 提升到 0.75
  冷启动本无法发现的漏洞 → 通过 knowledge-mcp 成功发现
```

---

## 5. 推荐实施路线

### Phase 1：立即可做（0改动，仅优化 Agent Prompt）

| 改进 | 方式 |
|------|------|
| 强化"先查询再行动"原则 | 在每个 Agent prompt 的 Workflow 第一步明确：`list_assets → list_clues → list_credentials` 三连查 |
| 增加 Asset- Clue 关联 | Agent 创建 Clue 时在 content 中通过命名约定引用 Asset ID |
| 增加 WorkLog 自动化 | Agent prompt 中的 "Record EVERYTHING" 改为 check-list 式的明确步骤 |

### Phase 2：asset-mcp 增强（数据模型改进）

| 优先级 | 改进项 | 工作量 | 影响范围 |
|--------|--------|--------|---------|
| 🔴 P0 | Session 表 + MCP Tools | 1天 | asset-mcp 新增 1 表 + 4 tools |
| 🔴 P0 | Endpoint 表 + 状态机 | 1.5天 | asset-mcp 新增 1 表 + 4 tools |
| 🟡 P1 | Credential 增强（verified 字段 + 验证工具） | 0.5天 | asset-mcp 扩展现有模型 |
| 🟡 P1 | Dependency 跟踪 | 0.5天 | asset-mcp 新增 1 表 |

### Phase 3：实时通知

| 优先级 | 改进项 | 工作量 | 说明 |
|--------|--------|--------|------|
| 🟡 P1 | asset-mcp SSE 事件流 | 1.5天 | 新增 `/subscribe?project_id=X` SSE 端点 |
| 🟡 P1 | Agent prompt 增加事件监听指令 | 0.5天 | 告诉 Agent 在执行任务的间隙 `get_events()` |

### Phase 4：跨项目学习

| 优先级 | 改进项 | 工作量 | 说明 |
|--------|--------|--------|------|
| 🟢 P2 | knowledge-mcp 新建 | 2天 | 独立 MCP Server，SQLite 存储 |
| 🟢 P2 | Strategist 接入 knowledge-mcp | 0.5天 | 规划时查询历史经验 |
| 🟢 P2 | Breach/Path 写入 knowledge-mcp | 0.5天 | 利用成功后记录技术事实 |

### Phase 5：高级特性

| 优先级 | 改进项 | 说明 |
|--------|--------|------|
| 🟢 P3 | MessageBus 替代 Multica issue 直连 | 引入 Clinkz 式 Orchestrator 中介模式 |
| 🟢 P3 | Trace/Audit Logging | 所有 MCP 操作自动记录审计日志 |

---

## 6. 关键设计决策

### 6.1 为什么用 asset-mcp 增强而不是引入 MessageBus？

**当前 ACA 的架构优势**：
- Multica 已提供任务队列、Issue 系统、WebSocket 实时推送
- asset-mcp 已经是所有 Agent 共享的集中式数据源
- 引入 MessageBus 会引入新的基础设施组件，增加运维复杂度

**策略**：先在现有 asset-mcp 上做数据模型增强，满足 80% 的共享需求。MessageBus 仅在需要"Agent 间对话式协商"时引入。

### 6.2 Session 共享的敏感信息处理

- Cookie/Token 存储在 asset-mcp 的 SQLite 中
- 可选：`value` 字段支持加密存储（通过 mcputil 中间件透明加密）
- 通过 project_id 做访问隔离（Agent 只能访问所属项目的 session）

### 6.3 knowledge-mcp 为什么要独立？

- **数据主权分离**：asset-mcp 包含目标信息（IP、域名），knowledge-mcp 绝不能包含任何目标信息
- **生命周期不同**：asset-mcp 的数据可以按项目清理，knowledge-mcp 的积累是永久资产
- **安全边界**：物理隔离两个 MCP Server，防止跨项目数据泄露

---

## 7. 参考

| 来源 | 链接 | 关键借鉴 |
|------|------|---------|
| Clinkz | https://github.com/ptkvaibhav/clinkz | MessageBus、双层 DB、Session 共享、Capability Recall |
| YAAP (LangGraph) | https://github.com/Breakintelligence/yaap | 基于状态图的 Agent 编排 |
| RedPilot | https://github.com/Thanwisut/RedPilot | 浏览器自动化 + 多 Agent 编排 |
| Pentest Command Center | https://github.com/batko15/pentest-command-center | Multi-Agent 渗透测试编排平台 |
| Hunter | https://github.com/Pillow-mycode/Hunter | Kali 工具编排 + 多 Agent 协作 |

---

## 8. 总结

AdversaryChefAgentTeam 当前的多 Agent 信息共享机制以 **asset-mcp 为集中式数据源** + **project_id 透传**为核心，已经具备了良好的基础架构。但存在以下关键差距：

1. **Session/Cookie 无法跨 Agent 传递**（P0）
2. **端点扫描结果缺乏结构化跟踪**（P0）
3. **凭据缺乏验证状态跟踪**（P1）
4. **Agent 之间缺乏实时通知**（P1）
5. **跨项目经验无法积累**（P2）

建议按照 Phase 1→2→3→4→5 路线逐步增强，优先完成 Phase 2 的 Session 表和 Endpoint 表，这将立即解决最核心的跨 Agent 协作痛点。
