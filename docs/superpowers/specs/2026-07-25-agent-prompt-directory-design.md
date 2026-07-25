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
├── _tests/                     # 测试用例（粘贴到 Multica issue 验证）
│   ├── echo-subdomains.md      # AC-Echo 子域名发现
│   ├── echo-export-assets.md   # AC-Echo 资产入库
│   ├── ghost-list-callbacks.md # AC-Ghost 回调列表
│   ├── supervisor-route-recon.md   # Supervisor 应分派给 AC-Echo
│   ├── supervisor-route-c2.md      # Supervisor 应直接给 AC-Ghost
│   └── chain-echo-to-breach.md     # AC-Echo → AC-Breach 手递手
├── supervisor.md               # AC-Supervisor — 任务分类 + squad 分派
├── strategist.md               # AC-Strategist — 攻击路径设计
├── echo-recon.md               # AC-Echo — 外网侦察→路由提取→接口验证
├── breach-exploit.md           # AC-Breach — RCE/SQLi/反序列化
├── ghost-mythic.md             # AC-Ghost — 纯 C2（tasking/免杀/文件）
├── path-lateral.md             # AC-Path — 提权/凭据/探测/横向
├── forge-resource.md           # AC-Forge — 基础设施
└── quill-report.md             # AC-Quill — 攻击路径还原/报告输出
```

命名规则：`{代号}-{角色}.md`，小写 + 连字符。`_tools/` 下划线前缀表示支撑文件。

---

## 二、Agent 体系

### 2.1 Agent 角色矩阵

所有 agent 使用 `AC-` 前缀（AdversaryChef 缩写），与 Multica 现有 agent 区分。

| Agent | 代号 | 核心职责 | ATT&CK 覆盖 | Requires MCP |
|-------|------|---------|-------------|:---:|
| AC-Supervisor | 总指挥 | 任务分类 → squad 分派 → 结果汇总 | — | — |
| AC-Strategist | 战略家 | 攻击路径设计、剧本编排、风险评估 | — | asset |
| AC-Echo | 攻击面测绘 | 外网侦察→JS路由提取→接口验证→漏洞线索 | TA0043 | asset, kali |
| AC-Forge | 铁匠铺 | 基础设施：VPS/CDN/隧道/钓鱼站点 | TA0042 | asset |
| AC-Breach | 突破口 | 漏洞利用：RCE/SQLi/命令注入/反序列化 | TA0001 | kali |
| AC-Ghost | 幽灵 | **纯 C2**：任务下发、免杀维持、文件传输 | TA0011/TA0002/TA0005 | mythic, asset |
| AC-Path | 内网路径 | 提权+凭据窃取+内网探测+横向移动 | TA0004/TA0006/TA0007/TA0008 | mythic, kali, asset |
| AC-Quill | 羽毛笔 | 攻击路径还原、结构化报告 | — | asset |

### 2.2 Agent 手递手边界

```
AC-Echo ──漏洞线索──→ AC-Breach
   "http://target/api/user?id=1' 疑似SQL注入，参数: id, 类型: GET"
   
AC-Breach ──初始shell──→ AC-Ghost
   "target(10.0.0.5): 已获取 webshell，进程 PID 1234"
   
AC-Ghost ──稳定回调──→ AC-Path
   "callback ID 3 (DC-01): SYSTEM 权限，可开始内网"
   
AC-Path ──凭据/新入口──→ AC-Ghost（新 callback）
   "DC-02: 获取 domain admin 凭据，已下发新 agent"
```

### 2.3 Supervisor 任务分类

Supervisor 不假设任务总从侦察开始。先分类，再分派：

| 任务类型 | 触发模式 | 分派 |
|---------|---------|------|
| Full engagement | "penetration test X", "attack Y", "完整渗透" | AC-Strategist → multi-agent chain |
| C2 operation | callback/agent mention, "task/shell/upload on" | AC-Ghost directly |
| Lateral movement | "move to", "pivot", "横向", "from X to Y" | AC-Path directly |
| Vulnerability exploit | specific vuln name, "exploit this", "利用" | AC-Breach directly |
| Surface mapping | "scan", "recon", "侦察", "find subdomains", "资产测绘" | AC-Echo directly |
| Infrastructure | "deploy", "server", "tunnel", "domain", "隧道" | AC-Forge directly |
| Intelligence | "summarize", "报告", "what did we find", "history" | Self-query + AC-Quill |
| Planning | "plan", "strategy", "方案", "what should we do" | AC-Strategist |

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

### 5.2 `expand-tools.sh`（通用版）

不再硬编码 MCP 名称。自动扫描 prompt 中的 `{{TOOLS_*}}` 占位符，匹配 `_tools/*.md` 展开：

```bash
#!/bin/bash
# Usage: ./scripts/expand-tools.sh <prompt-file>
# Auto-discovers all {{TOOLS_*}} placeholders and expands them
# from prompts/_tools/<name>.md. Works on macOS (bash 3.2+) and Linux.

set -euo pipefail

PROMPTS_DIR="$(cd "$(dirname "$0")/../prompts" && pwd)"

input=$(cat "$1")

# Find all {{TOOLS_XXX}} placeholders
placeholders=$(echo "$input" | grep -Eo '\{\{TOOLS_[A-Z_]+\}\}' | sort -u)

for ph in $placeholders; do
    # Extract name: {{TOOLS_ASSET}} → asset
    name=$(echo "$ph" | sed 's/{{TOOLS_//;s/}}//' | tr '[:upper:]' '[:lower:]')
    tool_file="$PROMPTS_DIR/_tools/${name}.md"
    if [ -f "$tool_file" ]; then
        tool_content=$(cat "$tool_file")
        input="${input//$ph/$tool_content}"
    else
        echo "⚠ expand-tools: $_tools/${name}.md not found, leaving $ph as-is" >&2
    fi
done

echo "$input"
```

**优势**：以后加 Cloudflare R2 MCP 时，只需创建 `prompts/_tools/r2.md`，在 prompt 里加 `{{TOOLS_R2}}`，脚本不用改。

### 5.3 `validate-prompts.sh`

```bash
#!/bin/bash
# Validates that all prompt files have no unknown {{...}} placeholders.
# {{TOOLS_*}} placeholders are auto-validated — expand-tools.sh handles them generically.
# Usage: ./scripts/validate-prompts.sh

set -euo pipefail
PROMPTS_DIR="$(cd "$(dirname "$0")/../prompts" && pwd)"

# Known non-tool placeholders that are OK to leave unexpanded
KNOWN="{{WORKSPACE}} {{MCP_ASSET_URL}} {{MCP_KALI_URL}} {{MCP_MYTHIC_URL}}"

for f in "$PROMPTS_DIR"/*.md; do
    base=$(basename "$f")
    unknown=$(grep -Eo '\{\{[A-Z_]+\}\}' "$f" 2>/dev/null | sort -u | while read -r p; do
        # {{TOOLS_*}} placeholders are always valid (generic expansion)
        if [[ "$p" =~ ^\{\{TOOLS_[A-Z_]+\}\}$ ]]; then
            continue
        fi
        # Check against known non-tool placeholders
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

## 六、测试策略

### 6.1 三层验证

| 层级 | 做什么 | 频率 |
|------|--------|------|
| **结构校验** | `validate-prompts.sh` 检查占位符、格式完整性 | 每次改 prompt |
| **单 Agent 测试** | 在 Multica 发一个单步任务，观察 Agent 行为 | 新建/改 agent 时 |
| **链式测试** | Squad 模式下发跨 agent 任务，观察手递手 | 关键场景回归 |

### 6.2 `_tests/` 目录

每个测试文件就是一个短任务描述，直接粘贴到 Multica issue。测试用例覆盖：

| 测试文件 | 目标 | 发给 |
|----------|------|------|
| `echo-subdomains.md` | "对 example.com 做子域名发现和端口扫描" | AC-Echo |
| `echo-export-assets.md` | "把发现的资产写入项目 X 的 asset-mcp" | AC-Echo |
| `ghost-list-callbacks.md` | "列出所有活跃 callback 和各自支持的命令" | AC-Ghost |
| `breach-verify-sqli.md` | "验证 http://target/api?id=1' 的 SQL 注入" | AC-Breach |
| `path-recon-from-callback.md` | "从 callback 3 执行内网存活主机探测" | AC-Path |
| `quill-generate-report.md` | "生成项目 X 的完整攻防报告" | AC-Quill |
| `supervisor-route-recon.md` | "侦察 target.com 的外网攻击面" | AC-Supervisor（应分派给 AC-Echo） |
| `supervisor-route-c2.md` | "看下 callback 3 的当前状态" | AC-Supervisor（应直接给 AC-Ghost） |
| `chain-echo-to-breach.md` | "发现 http://target 的漏洞线索后交给利用 agent" | AC-Supervisor（应 Echo→Breach） |

### 6.3 文件传输归属

| 场景 | Agent | 原因 |
|------|-------|------|
| "把 mimikatz.exe 传到 callback 3" | AC-Ghost | C2 文件传输 |
| "把回调的扫描结果下载回来" | AC-Ghost | C2 文件传输 |
| "把工具存到 R2 以备后用" | AC-Forge | 资源管理（未来 `r2_upload`） |
| "把报告导出上传到飞书" | AC-Quill | 报告交付 |
| "从 R2 拉工具传到 callback" | AC-Forge → AC-Ghost | 跨 agent 链，Supervisor 编排 |

### 6.4 Cloudflare R2 扩展

未来加 R2 MCP 时只需三步，无需修改脚本：

```
1. 新建 prompts/_tools/r2.md    # 工具注册表
2. prompt 中加 {{TOOLS_R2}}     # 引用
3. 新建 _tests/forge-r2-upload.md  # 测试用例
```

`expand-tools.sh` 自动识别 `{{TOOLS_R2}}` → 匹配 `_tools/r2.md`。

---

## 七、使用流程

### 在 Multica 创建 Agent

```bash
# 1. 展开占位符
./scripts/expand-tools.sh prompts/echo-recon.md > /tmp/echo-final.md

# 2. 手动替换 URL 占位符
sed -i 's/{{MCP_ASSET_URL}}/http:\/\/127.0.0.1:8081/g' /tmp/echo-final.md
sed -i 's/{{MCP_KALI_URL}}/http:\/\/127.0.0.1:8080/g' /tmp/echo-final.md

# 3. 复制到 Multica agent instruction
multica agent create --name "AC-Echo" --instructions "$(cat /tmp/echo-final.md)" --runtime-id <codex-id>

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

## 八、文件清单

| 文件 | 说明 |
|------|------|
| `prompts/README.md` | 使用说明 |
| `prompts/_tools/asset.md` | asset-mcp 工具注册表 |
| `prompts/_tools/kali.md` | kali-mcp 工具注册表 |
| `prompts/_tools/mythic.md` | mythic-mcp 工具注册表 |
| `prompts/_tests/echo-subdomains.md` | AC-Echo 子域名测试 |
| `prompts/_tests/echo-export-assets.md` | AC-Echo 资产入库测试 |
| `prompts/_tests/ghost-list-callbacks.md` | AC-Ghost 回调测试 |
| `prompts/_tests/breach-verify-sqli.md` | AC-Breach 漏洞验证测试 |
| `prompts/_tests/path-recon-from-callback.md` | AC-Path 内网探测测试 |
| `prompts/_tests/quill-generate-report.md` | AC-Quill 报告测试 |
| `prompts/_tests/supervisor-route-recon.md` | Supervisor 路由测试（侦察） |
| `prompts/_tests/supervisor-route-c2.md` | Supervisor 路由测试（C2） |
| `prompts/_tests/chain-echo-to-breach.md` | Echo→Breach 链式测试 |
| `prompts/supervisor.md` | AC-Supervisor 总指挥 prompt |
| `prompts/strategist.md` | AC-Strategist 战略规划 prompt |
| `prompts/echo-recon.md` | AC-Echo 攻击面测绘 prompt |
| `prompts/breach-exploit.md` | AC-Breach 漏洞利用 prompt |
| `prompts/ghost-mythic.md` | AC-Ghost C2 操作 prompt |
| `prompts/path-lateral.md` | AC-Path 内网路径 prompt |
| `prompts/forge-resource.md` | AC-Forge 资源管理 prompt |
| `prompts/quill-report.md` | AC-Quill 报告编写 prompt |
| `scripts/expand-tools.sh` | 占位符展开脚本（通用版） |
| `scripts/validate-prompts.sh` | 占位符校验脚本 |
