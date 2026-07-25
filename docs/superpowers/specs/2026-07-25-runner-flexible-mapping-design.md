# Runner MCP/Skills/Squad 灵活映射设计

**日期**: 2026-07-25  
**状态**: 已确认  

## 背景

当前 `acasched/internal/goose/runner.go` 将所有 MCP extension 无条件传给每个 Agent：

```go
if r.NexusMCP != "" { args = append(args, "--with-streamable-http-extension", r.NexusMCP) }
if r.KaliMCP  != "" { args = append(args, "--with-streamable-http-extension", r.KaliMCP) }
if r.MythicMCP != "" { args = append(args, "--with-streamable-http-extension", r.MythicMCP) }
```

问题：
1. 每个 Agent 实际需要不同 MCP 集合（echo-recon 不需要 mythic-mcp）
2. Skills 未映射到容器（goose 加载 `~/.agents/skills/`）
3. 新增 squad（如代码审计）需要改代码

## 设计目标

1. **声明式配置** — Agent 在 prompt 文件中声明自己需要什么
2. **Squad 隔离** — 每个 squad 独立目录，Agent/Skills/MCP 互不干扰
3. **互联网 MCP 友好** — MCP URL 可以指向互联网服务
4. **零代码扩展** — 新增 squad 只需加目录和配置，不改 Runner 代码

## 目录结构

```
skills/
├── _shared/                ← 所有 Agent 默认 mount
│   └── scheduler/          ← scheduler_create_task / complete_task
│       └── SKILL.md
├── red-team/
│   └── kali/
│       └── SKILL.md
└── code-audit/             ← 未来 squad
    ├── java-sql-audit/
    └── java-route-mapper/

prompts/
├── _mcp-registry.yaml       ← MCP 名字 → URL
├── _squads.yaml              ← squad 列表 + 默认配置
├── red-team/
│   ├── supervisor.md
│   ├── echo-recon.md
│   ├── breach-exploit.md
│   ├── ghost-mythic.md
│   ├── path-lateral.md
│   ├── forge-resource.md
│   ├── strategist.md
│   └── quill-report.md
└── code-audit/              ← 未来 squad
    └── sql-injection-review.md
```

## 配置文件

### `prompts/_mcp-registry.yaml`

MCP 名字到 URL 的单一真相源。Runner 不关心 MCP 是本地还是互联网。

```yaml
# 本地（容器通过 --network host 访问）
nexus-mcp:  "http://127.0.0.1:8081"
kali-mcp:   "http://127.0.0.1:8080"
mythic-mcp: "http://127.0.0.1:8082"

# 互联网 MCP（示例）
# shodan-mcp: "https://api.shodan.io/mcp"
# virustotal-mcp: "https://www.virustotal.com/mcp"
# semgrep-mcp: "https://semgrep.internal/mcp"
```

### `prompts/_squads.yaml`

声明存在的 squad 及其目录映射。

```yaml
squads:
  red-team:
    description: "攻防厨师团 — Red Team Operations"
    prompt_dir: "red-team"
    skill_dir: "red-team"
  code-audit:
    description: "代码审计小队 — Source Code Audit"
    prompt_dir: "code-audit"
    skill_dir: "code-audit"
```

### Prompt 头部格式

每个 Agent 的 `.md` 文件头部用标准字段声明依赖：

```markdown
# AC-Echo — Attack Surface Mapper

> **Purpose**: Map external attack surface
> **Requires**: nexus-mcp, kali-mcp
> **Skills**: red-team/kali
> **Input**: Target domain, IP range, or URL
```

解析规则：
- `Requires` — 逗号分隔的 MCP 名字，查 registry 得 URL
- `Skills` — 逗号分隔的 skill 路径（相对 skills/ 根），每个路径 mount 到容器的 `~/.agents/skills/{basename}/`
- `_shared/` 下的 skills **始终** mount，所有 Agent 共享

### Agent 命名

Task 创建时 `agent` 字段格式：`"<squad>/<agent>"`。

```
red-team/echo-recon
red-team/supervisor
code-audit/sql-injection-review
```

Runner 据此定位：
- prompt: `prompts/{squad}/{agent}.md`
- skills: `skills/{squad}/` + `skills/_shared/`
- squad 配置: 查 `_squads.yaml`

## Runner 执行流程

```
Task { agent: "red-team/echo-recon", description: "scan target.com" }
  │
  ├─ 1. 解析 agent → squad=red-team, agent=echo-recon
  │
  ├─ 2. 读 prompts/red-team/echo-recon.md
  │      解析 Requires: nexus-mcp, kali-mcp
  │      解析 Skills: red-team/kali
  │
  ├─ 3. 读 _mcp-registry.yaml
  │      nexus-mcp → http://127.0.0.1:8081
  │      kali-mcp  → http://127.0.0.1:8080
  │
  ├─ 4. 拼 Docker run 参数:
  │      args = ["run", "--rm", "--network", "host", "goose"]
  │
  │      # MCP extensions
  │      for each mcp in ["nexus-mcp", "kali-mcp"]:
  │        args += "--with-streamable-http-extension", registry[mcp]
  │
  │      # Mount agent prompt as system template
  │      args += "-v", prompt_path + ":" + "/root/.config/goose/prompts/system.md:ro"
  │
  │      # Mount skills (shared always, squad-specific per Skills field)
  │      for each skill in ["scheduler", "kali"]:
  │        args += "-v", skill_path + ":" + "/root/.agents/skills/" + skill_name + ":ro"
  │
  │      # Task instructions
  │      args += "run", "--instructions", instructions_path, ...
  │
  └─ 5. exec.CommandContext(ctx, "docker", args...)
```

## 关键决策

| 决策 | 结论 | 原因 |
|------|------|------|
| MCP 映射方式 | Prompt 文件声明 + registry 查 URL | Agent 自描述、改 URL 不改代码 |
| Skills 映射方式 | Prompt 文件声明 + 按路径 mount | 跟 MCP 一致，Runner 逻辑统一 |
| Squad 组织方式 | 子目录隔离 | 物理隔离，新增 squad = 创建目录 |
| Agent 命名 | `squad/agent` 格式 | 一步定位 prompt + skills |
| `_shared/` skills | 始终 mount | scheduler 工具是基础设施 |

## 错误处理

| 场景 | 行为 |
|------|------|
| MCP 名在 registry 中不存在 | 跳过 + 日志 warn，不阻塞任务 |
| Skill 目录不存在 | 跳过 + 日志 warn |
| Squad 目录不存在 | 报错，任务失败 |
| Prompt 文件读不到 | 报错，任务失败 |
| Requires / Skills 字段缺失 | 视为空列表，只 mount shared skills |

## 新增 Squad 步骤（以 code-audit 为例）

1. 创建 `prompts/code-audit/` 目录，写 agent `.md` 文件
2. 创建 `skills/code-audit/` 目录，放 squad 专用 skills
3. 在 `_squads.yaml` 中添加 squad 声明
4. 在 `_mcp-registry.yaml` 中添加 squad 需要的 MCP URL
5. 完成 — **零代码改动**
