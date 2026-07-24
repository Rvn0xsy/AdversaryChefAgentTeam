---
title: 红队多功能 Agent 技术方案
created: 2026-07-23
status: draft
---

# 红队多功能 Agent 技术方案

## 1. 背景与目标

### 1.1 业务场景

构建一个多功能的红队工作 Agent，覆盖以下工作场景：

- **渗透测试**：对目标系统进行安全评估，包括侦察、漏洞扫描、漏洞利用、后渗透
- **漏洞挖掘**：代码审计、FUZZ、自动化漏洞发现
- **工作汇报**：按周/月生成渗透工作汇总报告
- **技术复盘**：对已完成的渗透活动进行结构化复盘，沉淀经验

### 1.2 核心需求

| 维度 | 需求 |
|------|------|
| 执行环境 | 调用 Kali Linux 工具链（nmap、metasploit、sqlmap 等），通过 MCP 协议封装 |
| 协作模式 | 团队使用，需要任务分派、评论讨论、进度跟踪、复盘沉淀 |
| Agent 自主性 | 有一定自主决策能力，同时支持预定义 playbook/工作流 |
| 资产/凭据管理 | 敏感信息不能出现在 issue 描述中，需独立存储并通过 MCP 访问 |
| 模型路由 | 不使用 OpenAI 官方 API，需通过 cc-switch 代理转发至 DeepSeek |
| 数据主权 | 自部署 Multica 服务端，skills 和任务数据存储在自己控制的数据库中 |
| 环境隔离 | 渗透工具在容器中运行，不污染宿主环境 |

### 1.3 关键约束

- 不直接调用 OpenAI API（使用 cc-switch 路由至 DeepSeek V4 Pro）
- Skills（红队 playbook）需持久化但不需要外部第三方服务
- Agent 运行时产生的临时文件不能污染宿主文件系统

## 2. 方案决策

### 2.1 自建 Agent 框架 vs 使用 Multica

**结论：使用 Multica 作为协作与任务编排层，自建领域 MCP 作为能力层。**

理由：

- Multica 已提供：Web UI、Issue 管理、任务队列（queued → dispatched → running → completed/failed）、WebSocket 实时推送、成员权限、定时任务（Autopilots）、评论协作
- 不需要重复建设这些协作基础设施（预估节省 3-6 个月开发量）
- Multica 的 daemon 模型天然支持自定义 runtime + MCP 工具链的架构
- 自建不需要的部分仅限于红队领域逻辑：playbook 引擎、Kali MCP 封装、Asset MCP、报告生成

### 2.2 Runtime 选择：Codex

从 Multica 内置支持的 17 个 AI coding tools 中选择 Codex，原因：

- 支持 MCP（HTTP/SSE transport），可与 Kali 容器通信
- 支持 session resumption（thread/resume），任务中断可恢复
- 支持 cc-switch 模型路由（通过 config.toml 配置 custom model provider）
- JSON-RPC 2.0 协议，有细粒度审批机制（适用于渗透操作的安全边界控制）

## 3. 架构设计

### 3.1 整体架构

```
自部署 Multica 服务端
├── Postgres + Multica Server + Multica Frontend
├── Agent 定义（instructions, mcp_config, skill 绑定）
├── Workspace Skills（红队 playbook 和报告模板）
├── Issue / Task 队列
└── 成员与权限管理
        ▲ WebSocket / HTTP
        │
Docker Compose
├── Daemon + Codex 容器
│   ├── multica daemon
│   ├── Codex CLI
│   ├── Per-task workdir (daemon managed)
│   └── Persistent volumes: ~/.multica/, ~/.codex/
│
├── Kali 容器
│   ├── Kali Linux + Tools
│   ├── Kali MCP Server (HTTP/SSE)
│   │   ├── nmap
│   │   ├── metasploit
│   │   ├── sqlmap
│   │   ├── dirbuster
│   │   └── ...
│   └── 完全隔离（攻击面）
│        ▲ MCP (HTTP/SSE)
│        │
├── Asset MCP 容器
│   └── 资产信息/凭据查询服务
└── 内部 Docker network
```

### 3.2 组件职责

**Multica 服务端（自部署）**
- 存储所有持久数据：Agent 定义、Workspace Skills、Issue、Task、成员
- 运行 `docker-compose.selfhost.yml`，Postgres + Backend + Frontend
- 不执行任何 Agent 任务（任务由本地 daemon 驱动）

**Daemon + Codex 容器**
- 运行 `multica daemon` 和 Codex CLI
- 持久化 volume：`~/.multica/`（登录凭证）、`~/.codex/config.toml`（cc-switch 路由配置）
- Daemon 为每个任务创建 per-task `$CODEX_HOME`，注入 MCP 配置和 workspace skills
- 通过内部 Docker network 连接 Kali MCP 和 Asset MCP

**Kali 容器**
- 预装 Kali 工具链的容器镜像
- 运行 Kali MCP Server，将工具封装为 MCP tools
- 通过 HTTP/SSE transport 暴露 MCP endpoint
- 完全隔离——这是实际的攻击面

**Asset MCP**
- 结构化存储渗透目标的资产信息（IP、域名、技术栈、授权范围）
- 存储凭据（测试账号、API key 等敏感信息）
- 通过 MCP 协议供 Agent 按需查询
- 可基于 Postgres + 轻量 MCP wrapper 实现

## 4. Skills（Playbook）管理策略

### 4.1 三层知识注入机制

| 层 | 机制 | 存储位置 | 适用内容 |
|---|------|---------|---------|
| Agent Instructions | `instructions` 字段（system prompt） | Multica 服务端 | 渗透方法论概述、合规边界、角色定义 |
| Workspace Skills | Multica skill 系统 | Multica 服务端 | 结构化的 playbook 步骤、报告模板、检查清单 |
| Repo-scoped Skills | `.codex/skills/`（git 仓库内） | Git 仓库 | 项目私有的渗透 SOP、客户特定资产特征 |

### 4.2 Workspace Skills 生命周期

1. 团队专家在 Multica UI 中创建/编辑 Skill（`SKILL.md` + 可选脚本文件）
2. 将 Skill 绑定到 Agent
3. Agent 执行新任务时，daemon 从 Multica 服务端同步 Skill 到 per-task `$CODEX_HOME/skills/`
4. 已在执行的任务不受 Skill 更新影响（使用旧版本）
5. Skill 编辑仅对新创建的任务生效

**关于数据持久化：** Skills 存储在自部署的 Postgres 数据库中。即使 daemon 容器被销毁重建（只要 `~/.multica/` volume 保留登录凭证），daemon 重新注册后 Agent 和 Skill 绑定关系仍然存在，无需重建。

### 4.3 推荐的 Skills 结构

```
Workspace Skills:
├── penetration-methodology    — 标准渗透流程（侦察→扫描→利用→后渗透→报告）
├── owasp-checklist            — OWASP Top 10 检查清单
├── weekly-report-template     — 周报模板
├── retrospective-template     — 复盘报告模板
└── compliance-rules           — 授权范围和禁止操作列表
```

## 5. Codex cc-switch 配置

### 5.1 config.toml 内容

```toml
model_provider = "custom"
model = "deepseek-v4-pro"
model_reasoning_effort = "high"
disable_response_storage = true
model_catalog_json = "cc-switch-model-catalog.json"

[model_providers.custom]
name = "deepseek"
base_url = "http://127.0.0.1:15721/v1"
wire_api = "responses"
requires_openai_auth = true
experimental_bearer_token = "PROXY_MANAGED"
```

### 5.2 容器挂载

Daemon 容器中挂载为持久化 volume：

```yaml
volumes:
  - codex-config:/root/.codex   # 包含 config.toml
```

### 5.3 待验证问题（关键阻塞项）

**Daemon 创建 per-task `$CODEX_HOME` 时，是否会覆盖/忽略用户原有的 `config.toml` 中的 `[model_providers]` 配置？**

Multica 文档明确说明 daemon 会向 per-task `$CODEX_HOME/config.toml` 写入 `mcp_servers` 块，但未说明对用户已有配置的处理方式。

对比 Hermes 的行为（文档中有详细描述）：Hermes 使用 symlink mirror 原有 home，在 overlay 上叠加配置。Codex 的处理方式可能类似（继承原有配置 + 叠加 MCP），也可能不同（覆盖）。

**验证方案：**

在宿主上（不在容器中）：
1. 配置 cc-switch 的 `~/.codex/config.toml`
2. 启动 `multica daemon`
3. 创建一个测试 Agent（runtime 选 Codex），instructions 写简单测试内容
4. Assign 一个简单 issue
5. 查看 daemon 日志中 Codex 实际请求的 API endpoint
6. 如果是 cc-switch 地址 → 兼容，继续
7. 如果是 api.openai.com → 不兼容，需要备选方案

**备选方案（如果覆盖）：**
- 方案 A：在 Agent 的 `custom_env` 中设 `OPENAI_BASE_URL`（只能覆盖 base_url，无法覆盖 `wire_api`、`model_catalog_json`）
- 方案 B：放弃 daemon 的 MCP 注入，直接在 daemon 容器中用 `codex mcp add` 预配 Kali MCP 地址，让 Codex 使用原生 `~/.codex/config.toml`

## 6. 实施路线

### Phase 0：环境验证（0.5 天）

- [ ] 自部署 Multica 服务端（`git clone` → `make selfhost`）
- [ ] 在宿主上安装 Codex CLI 并配置 cc-switch
- [ ] 启动 `multica daemon`，验证 runtime 注册成功
- [ ] **关键验证**：测试 daemon 的 per-task CODEX_HOME 是否兼容 cc-switch 配置
- [ ] 测试 Codex Agent 在 Multica 中执行简单任务

### Phase 1：Kali MCP + 工具链容器化（1-2 天）

- [ ] 构建 Kali 容器镜像：Kali Linux + 渗透工具 + Kali MCP Server
- [ ] 编写 Kali MCP Server：
  - 封装 nmap（端口/服务扫描）
  - 封装 sqlmap（SQL 注入检测）
  - 封装 metasploit（漏洞利用，需确认步骤）
  - 封装 dirbuster/gobuster（目录爆破）
  - MCP transport：HTTP/SSE（非 stdio，因为跨容器）
- [ ] Docker Compose 编排：daemon + Codex 容器 + Kali 容器
- [ ] 验证 Codex 通过 MCP 调用 Kali 工具

### Phase 2：Asset MCP（0.5-1 天）

- [ ] 设计资产数据模型（目标、IP、域名、授权范围、凭据类型）
- [ ] 实现 Asset MCP Server（Postgres + MCP wrapper）
- [ ] 集成到 Docker Compose

### Phase 3：Playbook 编写（1-2 天）

- [ ] 编写 `penetration-methodology` workspace skill
- [ ] 编写 `owasp-checklist` workspace skill
- [ ] 编写 `report-template` 和 `retrospective-template`
- [ ] 编写 `compliance-rules`（授权边界、禁止操作列表）
- [ ] 创建 Agent，绑定 skills
- [ ] 端到端测试：Issue assign → Agent 执行 → 结果回写

### Phase 4：生产化（持续）

- [ ] 根据实际渗透结果调优 playbook
- [ ] 团队使用 + 反馈循环
- [ ] 复盘结论沉淀进 skills
- [ ] 考虑 Autopilots 定时任务（周报自动生成）

## 7. 已知风险与缓解措施

| 风险 | 概率 | 影响 | 缓解 |
|------|------|------|------|
| per-task CODEX_HOME 覆盖 cc-switch 配置 | 中 | 高（Agent 无法执行） | Phase 0 先验证；备选方案已准备 |
| MCP HTTP/SSE transport 在 Codex 中不稳定 | 低 | 中 | 文档确认 Codex 支持 HTTP/SSE；如不工作可回退到 stdio（同容器部署） |
| Kali 工具在容器内有权限问题 | 中 | 中 | 使用 `--privileged` 或精确 cap 配置；部分工具可能需要特定 kernel 模块 |
| 大规模并发渗透任务导致资源耗尽 | 低 | 中 | Docker Compose 设 resource limits；Multica daemon 层默认 20 并发，agent 层默认 6 并发 |
| Skills 更新后旧任务仍用旧版本 | 低 | 低 | 设计如此；如需刷新旧任务可手动 rerun |

## 8. 决策记录

| 决策点 | 选项 | 选择 | 理由 |
|--------|------|------|------|
| Agent 平台 | 自建 / Multica | Multica | 节省 3-6 月基础建设工作量 |
| AI Runtime | Claude Code / Codex / 其他 | Codex | MCP 支持 + cc-switch 兼容 + 自主推理 |
| Skills 存储 | 服务端 / 纯本地 / git 仓库 | Multica 服务端（自部署） | 一次配置持久化，删容器不丢 |
| 环境隔离 | Docker Compose 全部容器化 / 混合方案 | Docker Compose | Kali 必须容器化；daemon 容器化额外保障宿主清洁 |
| 模型路由 | 直接 API / cc-switch | cc-switch → DeepSeek V4 Pro | 成本控制 + 数据主权 |
| MCP transport | stdio / HTTP-SSE | HTTP-SSE | 跨容器通信必须用网络协议 |

## 9. 参考文档

- [Multica 官方文档 - How Multica works](https://docs.multica.ai/how-multica-works)
- [Multica 官方文档 - Skills](https://docs.multica.ai/skills)
- [Multica 官方文档 - AI coding tools matrix](https://docs.multica.ai/providers)
- [Multica 官方文档 - Self-host quickstart](https://docs.multica.ai/self-host-quickstart)
- [Multica 官方文档 - Daemon and runtimes](https://docs.multica.ai/daemon-runtimes)
- [Multica 官方文档 - Tasks](https://docs.multica.ai/tasks)
- [Multica 官方文档 - Creating and configuring agents](https://docs.multica.ai/agents-create)
