# AdversaryChef Agent Team — Squad Manifest

> **Squad Name**: 攻防厨师团 (AdversaryChef)
> **Alignment**: Red Team / Offensive Security Automation
> **Coordinator**: AC-Supervisor
> **Members**: 7 specialist agents + 1 supervisor

## Overview

AdversaryChef is an 8-agent red team squad running on Multica. It covers the full attack lifecycle — from reconnaissance to report delivery — using MCP-connected tools: asset-mcp (op data), kali-mcp (offensive tools), and mythic-mcp (C2 operations).

The squad uses **Super Powers-style prompts**: each agent has clear boundaries, playbook-driven workflows, error recovery procedures, and a strict tool escalation model (🟢 Passive → 🟡 Active → 🔴 Intrusive). AC-Supervisor classifies and routes all tasks — specialist agents never talk directly to the user.

---

## Agent Roster

| # | Agent | Role | MCP Tools | Key Capability |
|---|-------|------|-----------|----------------|
| 0 | **AC-Supervisor** | Attack Director | None (coordinates via Multica issues) | Task classification, decomposition, routing, result aggregation |
| 1 | **AC-Strategist** | Attack Strategist | asset-mcp | Attack path design, kill-chain phasing, risk assessment |
| 2 | **AC-Echo** | Attack Surface Mapper | asset-mcp, kali-mcp | Recon, JS crawling, API extraction, fuzzing, vuln clues |
| 3 | **AC-Breach** | Vulnerability Exploiter | kali-mcp, asset-mcp | Exploit verification, PoC, shell delivery, initial access |
| 4 | **AC-Ghost** | C2 Operator | mythic-mcp, asset-mcp | Callback triage, tasking, file ops, payload generation |
| 5 | **AC-Path** | Internal Pathfinder | mythic-mcp, kali-mcp, asset-mcp | PrivEsc, credential theft, lateral movement, network mapping |
| 6 | **AC-Forge** | Infrastructure Operator | asset-mcp | VPS, CDN, tunnel, domain, phishing site deployment |
| 7 | **AC-Quill** | Report Writer | asset-mcp | Findings aggregation, attack path reconstruction, risk scoring |

---

## Task Routing (AC-Supervisor Classification)

When a task arrives at AC-Supervisor, it is classified and dispatched per this table:

| Task Type | Pattern | Route To | Mode |
|-----------|---------|----------|------|
| Full engagement | "penetration test X", "attack Y", "完整渗透" | AC-Strategist → multi-agent chain | Sequential |
| C2 operation | callback/agent mention, "task/shell/upload on" | AC-Ghost | Direct |
| Lateral movement | "move to", "pivot", "横向", "from X to Y" | AC-Path | Direct |
| Vulnerability exploit | specific vuln name, "exploit this", "利用" | AC-Breach | Direct |
| Surface mapping | "scan", "recon", "侦察", "find subdomains" | AC-Echo | Direct |
| Infrastructure | "deploy", "server", "tunnel", "domain", "隧道" | AC-Forge | Direct |
| Intelligence | "summarize", "报告", "what did we find" | Self-query → AC-Quill | Sequential |
| Planning | "plan", "strategy", "方案", "what should we do" | AC-Strategist | Direct |

---

## Attack Chains (Multi-Agent Workflows)

### Chain A: Full Engagement
```
AC-Strategist (plan)
  → AC-Echo (recon, surface mapping)
    → AC-Breach (exploit vulnerabilities)
      → AC-Ghost (deploy C2, stabilize access)
        → AC-Path (lateral movement, priv esc)
          → AC-Quill (report)
```
*AC-Forge runs in parallel to provision infrastructure as needed.*

### Chain B: C2-Driven Operation
```
AC-Ghost (callback triage, initial enumeration)
  → AC-Path (internal recon from callback)
    → AC-Breach (exploit discovered internal targets)
```

### Chain C: Quick Recon
```
AC-Echo (port scan → web probe → JS analysis → fuzzing)
  → Report findings to supervisor
```

---

## MCP Server Dependencies

| MCP Server | URL Pattern | Provides | Used By |
|------------|-------------|----------|---------|
| **kali-mcp** | `http://127.0.0.1:8080` | exec, get_job, kill_job, list_jobs | AC-Echo, AC-Breach, AC-Path |
| **asset-mcp** | `http://127.0.0.1:8081` | CRUD for assets, clues, credentials | All agents |
| **mythic-mcp** | `http://127.0.0.1:8082` | callback/task/file management | AC-Ghost, AC-Path |

All agents connect via Multica agent `mcp_config` using `type: "http"`.

---

## Tool Escalation Boundary

All agents enforce a three-tier escalation model:

| Tier | Label | Scope | Examples |
|------|:-----:|-------|----------|
| 🟢 | Passive | Always allowed, no target interaction | `curl -I`, `dig`, `whois`, `ping` |
| 🟡 | Active | Within scope, record all findings | `nmap -sV`, `gobuster`, `katana`, `ffuf`, `httpx` |
| 🔴 | Intrusive | Supervisor authorization REQUIRED | `nuclei`, `sqlmap --os-shell`, password brute, NSE scripts |

- Agents operate at **🟡 Active** by default
- Upgrade to 🔴 only on explicit AC-Supervisor order
- AC-Echo's `web-vuln-scan` playbook is 🔴 and requires authorization
- AC-Echo's `web-fuzzing` playbook marks password brute as requiring explicit order

---

## Project ID Discipline

All specialist agents enforce a strict `project_id` discipline:

> Use the project_id from the Supervisor's dispatched task. Do NOT guess, do NOT use a random project. If a task arrives without a project_id, query asset-mcp for the workspace's active project before proceeding.

This ensures all findings (assets, clues, credentials) are scoped to the correct Multica project and never cross-contaminated.

---

## Skills

| Skill | Type | Loaded By | Description |
|-------|------|-----------|-------------|
| `kali` | Orchestration + Playbooks | AC-Echo, AC-Breach, AC-Path | Routes recon tasks to playbooks (port-scanning, web-probing, js-analysis, web-fuzzing, web-vuln-scan) |
| `mythic-cli` | CLI Reference | AC-Ghost, AC-Path | mythic-cli command reference and safety rules |

---

## Setup

### Prerequisites
- Multica workspace configured
- Codex runtime (`codex-runtime`) online
- MCP servers running: `kali-mcp:8080`, `asset-mcp:8081`, `mythic-mcp:8082`
- `prompts/_tools/` expanded via `scripts/expand-tools.sh`

### Create Agents

```bash
# 1. Expand tool placeholders for each agent
for agent in supervisor strategist echo-recon breach-exploit ghost-mythic path-lateral forge-resource quill-report; do
  ./scripts/expand-tools.sh prompts/${agent}.md > /tmp/${agent}-final.md
done

# 2. Create agents in Multica (example for AC-Echo)
multica agent create \
  --name "AC-Echo" \
  --description "Attack Surface Mapper — recon, JS crawling, API fuzzing" \
  --instructions "$(cat /tmp/echo-recon-final.md)" \
  --runtime-id <codex-runtime-id> \
  --mcp-config '{"mcpServers":{"asset":{"type":"http","url":"http://127.0.0.1:8081"},"kali":{"type":"http","url":"http://127.0.0.1:8080"}}}'

# 3. Assign skills to agents
multica agent skills add <agent-id> --skill-ids <kali-skill-id>
```

### Create Squad in Multica

```bash
multica squad create \
  --name "攻防厨师团" \
  --description "8-agent red team squad: full attack lifecycle automation via MCP" \
  --leader AC-Supervisor \
  --members AC-Strategist,AC-Echo,AC-Breach,AC-Ghost,AC-Path,AC-Forge,AC-Quill
```

---

## File Map

```
prompts/
├── README.md              ← Quick start & placeholder reference
├── squad.md               ← THIS FILE — squad manifest
├── supervisor.md          ← AC-Supervisor prompt
├── strategist.md          ← AC-Strategist prompt
├── echo-recon.md          ← AC-Echo prompt
├── breach-exploit.md      ← AC-Breach prompt
├── ghost-mythic.md         ← AC-Ghost prompt
├── path-lateral.md        ← AC-Path prompt
├── forge-resource.md      ← AC-Forge prompt
├── quill-report.md        ← AC-Quill prompt
├── _tools/                ← MCP tool references (expanded by expand-tools.sh)
│   ├── asset.md
│   ├── kali.md
│   └── mythic.md
├── _tests/                ← Agent behavior verification scenarios
└── scripts/
    ├── expand-tools.sh    ← {{TOOLS_*}} placeholder expansion
    └── validate-prompts.sh ← Prompt consistency checker
```

---

## Changelog

| Date | Change |
|------|--------|
| 2026-07-25 | Initial squad manifest — 8-agent structure, task routing table, attack chains, setup guide |
