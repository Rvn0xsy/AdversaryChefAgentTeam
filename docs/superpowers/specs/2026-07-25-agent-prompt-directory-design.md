# Agent Prompt Directory Design

> 2026-07-25 · 基于 AdversaryChef Agent 定义 · 面向 Multica 快速创建智能体

## 背景

AdversaryChef 项目有 10 个 Go 代码定义的 agent，每个 agent 的 prompt 用结构化的 `PromptSection` 渲染。现在需要将这些 prompt 适配为纯 Markdown 模板，放在 AdversaryChefAgentTeam 项目中，方便在 Multica 上快速复制粘贴创建智能体。

## 设计目标

1. **直接可用** — 展开占位符后复制粘贴到 Multica agent instruction
2. **边界清晰** — 每个 agent 有明确的 Purpose、Input、Output、In/Out scope
3. **工具同步** — MCP 工具变更时只需更新 `_tools/` 注册表，所有 prompt 自动跟随
4. **Super Powers 对齐** — 每个 agent 可独立理解、独立测试、错误恢复预定义

---

## 一、目录结构

```
prompts/
├── README.md                   # 使用说明 + 占位符替换规则
├── _tools/                     # 共享工具注册表
│   ├── asset.md                # asset-mcp 工具清单（28 tools）
│   ├── kali.md                 # kali-mcp 工具清单（4 tools）
│   └── mythic.md               # mythic-mcp 工具清单（14 tools）
├── supervisor.md               # 攻防总指挥 — 任务分类 + squad 分派
├── strategist.md               # 战略规划师 — 攻击路径设计
├── echo-recon.md               # 攻击面测绘师 — 外网侦察→路由提取→接口验证
├── breach-exploit.md           # 漏洞利用师 — RCE/SQLi/反序列化
├── ghost-mythic.md             # C2 操作员 — 纯 C2（tasking/免杀/文件）
├── path-lateral.md             # 内网路径师 — 提权/凭据/探测/横向
├── forge-resource.md           # 资源管理员 — 基础设施
└── quill-report.md             # 报告编写师 — 攻击路径还原/报告输出
```

命名规则：`{代号}-{角色}.md`，小写 + 连字符。`_tools/` 下划线前缀表示支撑文件。

---

## 二、Agent 体系

### 2.1 Agent 角色矩阵

| Agent | 代号 | 核心职责 | ATT&CK 覆盖 | Requires MCP |
|-------|------|---------|-------------|:---:|
| Supervisor | 总指挥 | 任务分类 → squad 分派 → 结果汇总 | — | — |
| Strategist | 战略家 | 攻击路径设计、剧本编排、风险评估 | — | asset |
| Echo | 攻击面测绘 | 外网侦察→JS路由提取→接口验证→漏洞线索 | TA0043 | asset, kali |
| Forge | 铁匠铺 | 基础设施：VPS/CDN/隧道/钓鱼站点 | TA0042 | asset |
| Breach | 突破口 | 漏洞利用：RCE/SQLi/命令注入/反序列化 | TA0001 | kali |
| Ghost | 幽灵 | **纯 C2**：任务下发、免杀维持、文件传输 | TA0011/TA0002/TA0005 | mythic, asset |
| Path | 内网路径 | 提权+凭据窃取+内网探测+横向移动 | TA0004/TA0006/TA0007/TA0008 | mythic, kali, asset |
| Quill | 羽毛笔 | 攻击路径还原、结构化报告 | — | asset |

### 2.2 Agent 手递手边界

```
Echo ──漏洞线索──→ Breach
   "http://target/api/user?id=1' 疑似SQL注入，参数: id, 类型: GET"
   
Breach ──初始shell──→ Ghost
   "target(10.0.0.5): 已获取 webshell，进程 PID 1234"
   
Ghost ──稳定回调──→ Path
   "callback ID 3 (DC-01): SYSTEM 权限，可开始内网"
   
Path ──凭据/新入口──→ Ghost（新 callback）
   "DC-02: 获取 domain admin 凭据，已下发新 agent"
```

### 2.3 Supervisor 任务分类

Supervisor 不假设任务总从侦察开始。先分类，再分派：

| 任务类型 | 触发模式 | 分派 |
|---------|---------|------|
| Full engagement | "penetration test X", "attack Y", "完整渗透" | Strategist → multi-agent chain |
| C2 operation | callback/agent mention, "task/shell/upload on" | Ghost directly |
| Lateral movement | "move to", "pivot", "横向", "from X to Y" | Path directly |
| Vulnerability exploit | specific vuln name, "exploit this", "利用" | Breach directly |
| Surface mapping | "scan", "recon", "侦察", "find subdomains", "资产测绘" | Echo directly |
| Infrastructure | "deploy", "server", "tunnel", "domain", "隧道" | Forge directly |
| Intelligence | "summarize", "报告", "what did we find", "history" | Self-query + Quill |
| Planning | "plan", "strategy", "方案", "what should we do" | Strategist |

---

## 三、Prompt 模板规范

### 3.1 统一模板

```markdown
# {{AGENT_DISPLAY_NAME}}

> **Purpose**: {{ONE_SENTENCE_PURPOSE}}
> **Requires**: {{REQUIRED_MCPS}}
> **Input**: {{WHAT_KIND_OF_TASKS}}
> **Output**: {{WHAT_IT_PRODUCES}}

## Boundaries

- **In scope**: ...
- **Out of scope**: ...

## MCP Tools

{{TOOLS_ASSET}}
{{TOOLS_KALI}}
{{TOOLS_MYTHIC}}

## Workflow

1. ...
2. ...

## Error Recovery

| Failure | Action |
|---------|--------|
| ... | ... |

## Autonomy Rules

- **Proceed without asking**: ...
- **Escalate to supervisor**: ...

## MCP Discovery

If other MCP servers are connected in the future, call the MCP tools/list endpoint 
first to discover new capabilities before assuming what is available.
```

### 3.2 占位符约定

| 占位符 | 含义 | 示例值 |
|--------|------|--------|
| `{{WORKSPACE}}` | Multica workspace 名称 | 画江南 |
| `{{MCP_ASSET_URL}}` | asset-mcp 地址 | http://127.0.0.1:8081 |
| `{{MCP_KALI_URL}}` | kali-mcp 地址 | http://127.0.0.1:8080 |
| `{{MCP_MYTHIC_URL}}` | mythic-mcp 地址 | http://127.0.0.1:8082 |
| `{{TOOLS_ASSET}}` | asset 工具清单 | 展开为 `_tools/asset.md` |
| `{{TOOLS_KALI}}` | kali 工具清单 | 展开为 `_tools/kali.md` |
| `{{TOOLS_MYTHIC}}` | mythic 工具清单 | 展开为 `_tools/mythic.md` |

### 3.3 每个 Agent 的 `{{TOOLS_*}}` 组合

| Agent | TOOLS_ASSET | TOOLS_KALI | TOOLS_MYTHIC |
|-------|:---:|:---:|:---:|
| Supervisor | — | — | — |
| Strategist | ✅ | — | — |
| Echo | ✅ | ✅ | — |
| Forge | ✅ | — | — |
| Breach | — | ✅ | — |
| Ghost | ✅ | — | ✅ |
| Path | ✅ | ✅ | ✅ |
| Quill | ✅ | — | — |

---

## 四、工具注册表

### 4.1 `_tools/asset.md`

对所有 28 个工具做分类表。CRUD 部分按实体分组（5 实体 × 5 操作），search/stats 单独列出。格式：

```markdown
## asset-mcp Tools (28 tools)

> **Server**: `http://{{MCP_ASSET_URL}}`

### Projects

| Tool | Description |
|------|-------------|
| `list_projects` | List all pentest projects |
| `get_project` | Get project by ID |
| `create_project` | Create a new project |
| `update_project` | Update project info |
| `delete_project` | Delete a project |

### Assets

| Tool | Description |
|------|-------------|
| `list_assets` | List assets in a project by project_id |
| `search_assets` | Full-text search across name, IPs, domains, tech stack, description |
| `get_asset` | Get asset by ID |
| `create_asset` | Create asset (change project by updating project_id) |
| `update_asset` | Update asset info |
| `delete_asset` | Delete asset |

### Clues / Findings

| Tool | Description |
|------|-------------|
| `list_clues` | List clues in a project by project_id |
| `search_clues` | Search clues by keyword + optional type/status filter |
| `get_clue` | Get clue by ID |
| `create_clue` | Create clue with type (vulnerability/info_disclosure/misconfig) and status |
| `update_clue` | Update clue info (title, content, type, status) |
| `delete_clue` | Delete clue |

### Credentials

| Tool | Description |
|------|-------------|
| `list_credentials` | List credentials in a project by project_id |
| `get_credential` | Get credential by ID |
| `create_credential` | Create credential with type, label, value, optional asset_id reference |
| `update_credential` | Update credential info |
| `delete_credential` | Delete credential |

### Work Logs

| Tool | Description |
|------|-------------|
| `list_work_logs` | List work logs in a project by project_id |
| `get_work_log` | Get work log by ID |
| `create_work_log` | Create work log entry |
| `update_work_log` | Update work log |
| `delete_work_log` | Delete work log |

### Project Stats

| Tool | Description |
|------|-------------|
| `project_summary` | Rollup view: asset count, clue count by type, credential count, worklog count |
```

### 4.2 `_tools/kali.md`

```markdown
## kali-mcp Tools (4 tools)

> **Server**: `http://{{MCP_KALI_URL}}`

| Tool | Description |
|------|-------------|
| `exec` | Run shell command asynchronously. Returns job_id. Supports nmap, sqlmap, gobuster, curl, netcat, dig, and any apt-installed tool. |
| `list_jobs` | List all jobs, optionally filter by status: running/completed/failed/killed/timed_out |
| `get_job` | Get job details including status and partial/full output |
| `kill_job` | Force-kill a running job and its entire process group |
```

### 4.3 `_tools/mythic.md`

```markdown
## mythic-mcp Tools (14 tools)

> **Server**: `http://{{MCP_MYTHIC_URL}}`

### Callbacks

| Tool | Description |
|------|-------------|
| `mythic_list_callbacks` | List all active agent callbacks |
| `mythic_get_callback` | Get callback details by display_id (host, user, IPs, PID, agent type) |
| `mythic_get_callback_commands` | List all loaded commands and their parameter groups for a callback |

### Tasking

| Tool | Description |
|------|-------------|
| `mythic_issue_task` | Issue a command to a callback. Use exact command name and parameters from get_callback_commands. Returns task_id. |
| `mythic_list_tasks` | List all tasks for a callback, most recent first |
| `mythic_get_task_status` | Check task status without blocking (status: pending/processing/completed/error) |
| `mythic_wait_for_task` | Block until task completes, then return output. Use for fast foreground commands. |
| `mythic_get_task_output` | Get full decoded output of a completed task |

### Files

| Tool | Description |
|------|-------------|
| `mythic_list_files` | List files in Mythic file store |
| `mythic_upload_file` | Upload a local file to Mythic |
| `mythic_download_file` | Download a file from Mythic by agent_file_id |
| `mythic_delete_file` | Delete a file from Mythic |

### Payloads

| Tool | Description |
|------|-------------|
| `mythic_create_payload` | Create a new payload/agent (exe, dll, shellcode, etc.) |
| `mythic_get_payload` | Get payload details by UUID |
```

---

## 五、脚本

### 5.1 目录

```
scripts/
├── expand-tools.sh           # 展开 {{TOOLS_*}} 占位符
└── validate-prompts.sh       # 检查占位符是否全部替换（未来）
```

### 5.2 `expand-tools.sh`

```bash
#!/bin/bash
# Usage: ./scripts/expand-tools.sh <prompt-file>
# Replaces {{TOOLS_ASSET}}, {{TOOLS_KALI}}, {{TOOLS_MYTHIC}} with content from prompts/_tools/
# Outputs to stdout. Works on macOS (bash 3.2+) and Linux.

set -euo pipefail

PROMPTS_DIR="$(cd "$(dirname "$0")/../prompts" && pwd)"

expand() {
    local content="$1"
    local placeholder="$2"
    local tool_file="$3"
    local tool_content
    tool_content=$(cat "$PROMPTS_DIR/_tools/$tool_file")
    echo "${content//$placeholder/$tool_content}"
}

input=$(cat "$1")
input=$(expand "$input" "{{TOOLS_ASSET}}" "asset.md")
input=$(expand "$input" "{{TOOLS_KALI}}" "kali.md")
input=$(expand "$input" "{{TOOLS_MYTHIC}}" "mythic.md")
echo "$input"
```

### 5.3 `validate-prompts.sh`

```bash
#!/bin/bash
# Checks that all prompt files have no unexpanded {{...}} placeholders except known ones.
# Usage: ./scripts/validate-prompts.sh
# Compatible with macOS (bash 3.2+) and Linux.

set -euo pipefail
PROMPTS_DIR="$(cd "$(dirname "$0")/../prompts" && pwd)"

# Known placeholders that are OK to leave unexpanded
KNOWN="{{TOOLS_ASSET}} {{TOOLS_KALI}} {{TOOLS_MYTHIC}} {{WORKSPACE}} {{MCP_ASSET_URL}} {{MCP_KALI_URL}} {{MCP_MYTHIC_URL}}"

for f in "$PROMPTS_DIR"/*.md; do
    base=$(basename "$f")
    unknown=$(grep -Eo '\{\{[A-Z_]+\}\}' "$f" 2>/dev/null | sort -u | while read -r p; do
        if ! echo "$KNOWN" | grep -qF "$p"; then
            echo "  $p"
        fi
    done)
    if [ -n "$unknown" ]; then
        echo "❌ $base: unknown placeholders:$unknown"
    else
        echo "✅ $base"
    fi
done
```

---

## 六、使用流程

### 在 Multica 创建 Agent

```bash
# 1. 展开占位符
./scripts/expand-tools.sh prompts/echo-recon.md > /tmp/echo-final.md

# 2. 手动替换 URL 占位符
sed -i 's/{{MCP_ASSET_URL}}/http:\/\/127.0.0.1:8081/g' /tmp/echo-final.md
sed -i 's/{{MCP_KALI_URL}}/http:\/\/127.0.0.1:8080/g' /tmp/echo-final.md

# 3. 复制到 Multica agent instruction
multica agent create --name "Echo" --instructions "$(cat /tmp/echo-final.md)" --runtime-id <codex-id>

# 4. 配置 MCP
multica agent update <agent-id> --mcp-config '{"mcpServers":{"asset":{"type":"http","url":"http://127.0.0.1:8081"},"kali":{"type":"http","url":"http://127.0.0.1:8080"}}}'
```

### MCP 工具变更时

```bash
# 只改工具注册表，prompt 自动跟随
vim prompts/_tools/asset.md  # 加新工具
./scripts/validate-prompts.sh  # 检查一致性
```

---

## 七、文件清单

| 文件 | 说明 |
|------|------|
| `prompts/README.md` | 使用说明 |
| `prompts/_tools/asset.md` | asset-mcp 工具注册表 |
| `prompts/_tools/kali.md` | kali-mcp 工具注册表 |
| `prompts/_tools/mythic.md` | mythic-mcp 工具注册表 |
| `prompts/supervisor.md` | 总指挥 prompt |
| `prompts/strategist.md` | 战略规划 prompt |
| `prompts/echo-recon.md` | 攻击面测绘 prompt |
| `prompts/breach-exploit.md` | 漏洞利用 prompt |
| `prompts/ghost-mythic.md` | C2 操作 prompt |
| `prompts/path-lateral.md` | 内网路径 prompt |
| `prompts/forge-resource.md` | 资源管理 prompt |
| `prompts/quill-report.md` | 报告编写 prompt |
| `scripts/expand-tools.sh` | 占位符展开脚本 |
| `scripts/validate-prompts.sh` | 占位符校验脚本 |
