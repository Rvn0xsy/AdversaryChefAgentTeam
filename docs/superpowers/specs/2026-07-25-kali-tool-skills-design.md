# Kali Tool Skills + Dictionary + Tool Escalation Boundary

> 2026-07-25 · Kali MCP 工具场景标准化 · Super Powers 风格 SKILL.md

## 背景

Kali 容器预装了 katana、httpx、naabu、gobuster、ffuf、nuclei 等工具，但 Agent 不知道何时用、怎么用、如何组合。需要按**场景**组织可复用的 skill，指导 Agent 通过 kali-mcp `exec` 正确使用这些工具。

同时解决两个安全风险：
1. Agent 自动从探测升级到扫描（触发 IDS/IPS 告警）
2. 字典/模板等数据文件的管理和同步

## 一、文件清单

```
skills/kali/                              # 新增：5 个场景 skill
├── port-scanning/SKILL.md                # naabu + nmap 端口扫描
├── web-probing/SKILL.md                  # httpx 纯探测（无 nuclei）
├── js-analysis/SKILL.md                  # katana JS 爬取 + 路由提取
├── web-fuzzing/SKILL.md                  # gobuster 目录爆破 + ffuf 参数 Fuzzing
└── web-vuln-scan/SKILL.md                # nuclei 漏洞扫描（仅显式触发）

data/dictionaries/                        # 新增：Fuzzing 字典
├── README.md                             # 来源说明 + 更新方式
├── dir/common.txt                        # 常见目录路径
├── api/common.txt                        # 常见 API 路径
├── param/common.txt                      # 常见参数名
└── pass/top100.txt                       # 常见弱密码

data/nuclei-templates/                    # gitignored，脚本管理
data/nuclei-templates-custom/             # 提交到 git，自定义模板
data/nuclei-templates-custom/README.md

prompts/echo-recon.md                     # 修改：加 Kali Tool Skills 段 + 工具升级边界
prompts/supervisor.md                     # 修改：加 Tool Escalation Boundary 段

scripts/update-nuclei-templates.sh        # 新增：拉取 + 合并 nuclei 模板

.gitignore                                # 修改：加 data/nuclei-templates/
docker/kali-mcp/Dockerfile                # 修改：加 COPY dictionaries + nuclei-templates
```

---

## 二、Skill 格式规范

统一 Super Powers 风格模板：

```markdown
---
name: <kebab-case>
description: <触发关键词 + 何时使用>
---

# <标题>

> **Purpose**: <一句话目的>
> **Requires**: kali-mcp (exec, get_job), asset-mcp (<用到的工具>)
> **Input**: <输入>
> **Output**: <产出 — 写到哪、什么格式>

## Boundaries
- **In scope**: ...
- **Out of scope**: <与谁 handoff> ...

## Workflow
1. `exec` with `<command>`
2. Poll `get_job` ...
...

## Error Recovery
| Failure | Action |
|---------|--------|
| ... | ... |

## Autonomy Rules
- **Proceed without asking**: ...
- **Escalate to supervisor**: ...

## Verified Patterns
- `<command>` — <说明>
```

### 关键约束

- **不写抽象概念** — 给 Agent 可直接复制的命令
- **Recording Results** 合并进 Workflow 最后一步 — 不是附录
- **Out of scope** 必须写明 handoff 到谁 — 防止越界
- **Autonomy Rules** 区分 🟢/🟡/🔴 三个工具等级

---

## 三、Skill 定义

### 3.1 `port-scanning` — 端口扫描

```markdown
---
name: port-scanning
description: Port scanning with naabu and nmap via kali-mcp. Use when asked to scan ports, discover services, find open ports, or map the attack surface of a target host.
---

# Port Scanning

> **Purpose**: Discover open TCP ports and running services on target hosts.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_asset, create_clue)
> **Input**: Target IP or hostname + project_id
> **Output**: Clues (open ports with service/version) + updated assets

## Boundaries
- **In scope**: TCP port discovery, service version detection, banner grabbing
- **Out of scope**: UDP scanning. Web application probing (hand off to web-probing skill). Exploitation (hand off to AC-Breach)

## Workflow
1. `exec` with `naabu -host <target> -top-ports 1000 -rate 1000`
2. Poll `get_job` until status "completed"
3. For each open port: `exec` with `nmap -sV -p <port> <target>` for service fingerprint
4. Record discoveries: `create_clue` with type="info_disclosure", content describing service, version, potential risks
5. Record new hosts: `create_asset` with IPs and discovered ports in description

## Error Recovery
| Failure | Action |
|---------|--------|
| naabu "no ports found" | Try `nmap -sT --top-ports 100 <target>` (full TCP connect) |
| nmap times out (>120s) | Reduce ports or add `--host-timeout 60s` |
| exec "failed" immediately | Check target reachability: `exec` with `ping -c 1 <target>` |

## Autonomy Rules
- **Proceed without asking**: 🟡 Scanning targets in scope. Standard port ranges. Recording findings.
- **Escalate to supervisor**: Target outside scope. Full 65535 scan needed. Unexpected service (potential backdoor).

## Verified Patterns
- `naabu -host 10.0.0.1 -top-ports 1000 -rate 1000` — broad scan, fast rate
- `nmap -sV -sC -p 22,80,443 target.com` — focused scan with scripts
- `nmap -sT -p- --min-rate 500 target` — full TCP connect for stealth
```

### 3.2 `web-probing` — Web 探测（纯 httpx，无 nuclei）

```markdown
---
name: web-probing
description: HTTP probing with httpx. Use when asked to probe web servers, fingerprint HTTP services, check which URLs are alive, or detect web technologies.
---

# Web Probing

> **Purpose**: Detect live web servers and fingerprint their technology stack.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (update_asset, create_clue)
> **Input**: List of URLs/IPs:ports + project_id
> **Output**: Updated assets (tech_stack) + clues (server info, interesting headers)

## Boundaries
- **In scope**: HTTP/HTTPS probing, status code checks, server header capture, technology detection
- **Out of scope**: Directory brute-force (hand off to web-fuzzing skill). JS crawling (hand off to js-analysis skill). Vulnerability scanning — NEVER run nuclei or sqlmap from this skill. Active exploitation (hand off to AC-Breach)

## Workflow
1. `exec` with `echo "<url1>\n<url2>" | httpx -status-code -server -title -tech-detect -silent`
2. Poll `get_job` until status "completed"
3. For each live URL: `update_asset` with discovered tech_stack
4. For interesting headers (Server, X-Powered-By, Set-Cookie): `create_clue` with type="info_disclosure"
5. For 401/403 responses: `create_clue` with type="info_disclosure" — these are target surfaces for AC-Breach

## Error Recovery
| Failure | Action |
|---------|--------|
| httpx returns 0 results | Verify URLs are reachable: `curl -I <url>` |
| All results are timeouts | Check if target blocks probing, try `-http-proxy http://proxy:port` |
| Technology detection fails | Fall back to `curl -I` and manually parse Server header |

## Autonomy Rules
- **Proceed without asking**: 🟡 Probing within scope. Recording technology stack. Identifying web surface.
- **Escalate to supervisor**: All URLs return 403/blocked (potential WAF). Discovery of exposed admin panels/PHPMyAdmin. Discovery of unauthenticated API docs (Swagger/GraphQL).

## Verified Patterns
- `cat urls.txt | httpx -status-code -server -title -tech-detect -silent -nc` — full probe, no color
- `httpx -u https://target.com -status-code -title -follow-redirects` — single URL with redirects
- `cat urls.txt | httpx -status-code -title -o live.txt` — probe and save results to file
```

### 3.3 `js-analysis` — JS 路由提取

```markdown
---
name: js-analysis
description: JavaScript crawling and route extraction with katana. Use when asked to crawl JS, extract API endpoints, find hidden routes, or map a single-page application.
---

# JS Analysis

> **Purpose**: Crawl JavaScript-heavy websites to extract API endpoints and hidden routes.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_clue)
> **Input**: Target URL + project_id
> **Output**: Clues (extracted API routes, hidden endpoints, JS file URLs)

## Boundaries
- **In scope**: JS crawling, URL extraction, API route discovery, hidden endpoint identification
- **Out of scope**: Directory brute-force (hand off to web-fuzzing skill). Parameter fuzzing (hand off to web-fuzzing skill). Active vulnerability scanning (NEVER run nuclei from this skill). Authentication testing.

## Workflow
1. `exec` with `katana -u <target> -jc -kf all -silent` (crawl JS files, extract all URLs)
2. Poll `get_job` until status "completed"
3. Parse output — focus on:
   - API paths matching `/api/*`, `/v1/*`, `/graphql`
   - Route patterns containing `:id`, `{uuid}`
   - File upload endpoints
   - Authentication endpoints (`/login`, `/oauth`, `/token`)
4. Record extracted routes: `create_clue` with type="info_disclosure", content listing routes and source JS files
5. Flag interesting findings: unprotected admin routes, debug endpoints, exposed `.map` files

## Error Recovery
| Failure | Action |
|---------|--------|
| katana returns 0 results | Try without `-jc`: `katana -u <target> -silent` (crawl without JS parsing) |
| katana times out | Add `-m 50` to limit max URLs crawled |
| JS files blocked | Try `curl <target>/app.js` then manually search for route strings |

## Autonomy Rules
- **Proceed without asking**: 🟡 JS crawling within scope. Recording discovered routes. Flagging suspicious endpoints.
- **Escalate to supervisor**: Discovery of exposed debug/diagnostic endpoints. Discovery of hardcoded API keys or secrets in JS. Request to crawl outside stated scope.

## Verified Patterns
- `katana -u https://target.com -jc -kf all -silent -o routes.txt` — full JS crawl
- `katana -u https://target.com -d 3 -silent` — depth 3 crawl, no JS parsing
- `katana -u https://target.com -jc -em json,woff,png,svg -silent` — skip static assets to reduce noise
```

### 3.4 `web-fuzzing` — 目录爆破 + 参数 Fuzzing

```markdown
---
name: web-fuzzing
description: Directory brute-force and parameter fuzzing with gobuster and ffuf. Use when asked to find hidden directories, brute-force API paths, fuzz parameters, or discover hidden content.
---

# Web Fuzzing

> **Purpose**: Discover hidden directories and test API parameters on web targets.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_clue)
> **Input**: Target URL + project_id
> **Output**: Clues (discovered paths, 200 responses with interesting content)

## Boundaries
- **In scope**: Directory brute-force, API path discovery, parameter fuzzing, file extension testing
- **Out of scope**: Password brute-force (🟡 In-scope only on explicit order — use pass/top100.txt dictionary). Vulnerability scanning (NEVER run nuclei). Exploitation (hand off to AC-Breach)

## Dictionaries
All dictionaries are mounted at `/data/dictionaries/`:
| Path | Content | When |
|------|---------|------|
| `/data/dictionaries/dir/common.txt` | Common web directories | Directory brute-force |
| `/data/dictionaries/api/common.txt` | Common API paths | API endpoint discovery |
| `/data/dictionaries/param/common.txt` | Common parameter names | Parameter fuzzing |
| `/data/dictionaries/pass/top100.txt` | Top 100 weak passwords | Password brute (🟡 only on order) |

## Workflow

### Directory Discovery
1. `exec` with `gobuster dir -u <url> -w /data/dictionaries/dir/common.txt -t 20 -q`
2. Poll `get_job`. Expected runtime: 60-180s depending on rate limiting.
3. Record 200/403 responses: `create_clue` with type="info_disclosure". 403 means the path exists but is protected.
4. Record 301/302 redirects: the redirect target may reveal internal architecture.

### Parameter Fuzzing
1. `exec` with `ffuf -u <url>?FUZZ=1 -w /data/dictionaries/param/common.txt -mc 200,403,500 -t 20`
2. Poll `get_job`
3. Any difference in response (200/403/500 vs baseline) indicates the parameter exists
4. Record discovered parameters: `create_clue`, include the parameter name and response difference

### API Path Discovery
1. `exec` with `gobuster dir -u <url> -w /data/dictionaries/api/common.txt -t 20 -q`
2. Same analysis as directory discovery

## Error Recovery
| Failure | Action |
|---------|--------|
| gobuster all 404 | Target may use routing (all paths return 200). Try `ffuf -u <url>/FUZZ -w dict -fc 404` |
| Rate limited (429) | Reduce `-t 5` and retry with delay `--delay 500ms` |
| ffuf baseline unknown | Run `ffuf -u <url>?FUZZ=1 -w dict -mc all` first to see default response pattern |

## Autonomy Rules
- **Proceed without asking**: 🟡 Directory/API fuzzing within scope. Parameter discovery. Recording findings.
- **Escalate to supervisor**: Discovery of `/admin` or unprotected management panels. Discovery of `.git/`, `.env`, or `backup.zip`. Password brute-force requests. Rate limiting from target (they're detecting us).

## Verified Patterns
- `gobuster dir -u https://target.com -w /data/dictionaries/dir/common.txt -t 20 -q -o results.txt` — standard brute
- `ffuf -u https://target.com/api/user?id=FUZZ -w /data/dictionaries/param/common.txt -mc 200 -t 20` — param fuzz with IDOR potential
- `ffuf -u https://target.com/FUZZ -w /data/dictionaries/dir/common.txt -mc 200,403 -t 20` — alternative to gobuster
```

### 3.5 `web-vuln-scan` — Nuclei 漏洞扫描（仅显式触发）

```markdown
---
name: web-vuln-scan
description: Vulnerability scanning with nuclei. Use ONLY when explicitly ordered by Supervisor — never run automatically. IDS/IPS WILL detect this.
---

# Web Vulnerability Scanning

> **Purpose**: Run nuclei vulnerability templates against confirmed web targets.
> **Requires**: kali-mcp (exec, get_job), asset-mcp (create_clue)
> **Input**: Target URL(s) + project_id + explicit Supervisor authorization
> **Output**: Clues (confirmed vulnerabilities with severity)

## Boundaries
- **In scope**: Running nuclei against explicitly authorized targets. Recording results as clues.
- **Out of scope**: Automatic scanning. Scanning without authorization. False positive triage (record all, let human triage).

## Workflow
1. Confirm explicit Supervisor authorization before running ANY nuclei command.
2. `exec` with `nuclei -u <url> -t /root/nuclei-templates/ -severity critical,high -silent`
3. Poll `get_job`. Expected runtime: 30-300s for a single URL.
4. Add `-severity medium` only if explicitly requested by Supervisor.
5. Record each finding: `create_clue` with type="vulnerability", content including template name, matched URL, severity.
6. Do NOT run `-severity low` or `-severity info` templates unless ordered — false positive mountain, not worth it.

## Error Recovery
| Failure | Action |
|---------|--------|
| nuclei "no templates found" | Verify `/root/nuclei-templates/` is mounted. Report to supervisor. |
| nuclei "target not reachable" | Double-check URL with `curl -I` |
| nuclei times out | Reduce template scope: `-t /root/nuclei-templates/cves/` |

## Autonomy Rules
- **Proceed without asking**: 🟡 NEVER. Nuclei is 🔴 Intrusive. Every run requires explicit authorization.
- **Escalate to supervisor**: After scanning completes — report findings immediately. If target blocks the scan mid-way — report and stop.

## Verified Patterns
- `nuclei -u https://target.com -t /root/nuclei-templates/ -severity critical,high -silent` — standard scan
- `nuclei -list targets.txt -t /root/nuclei-templates/cves/ -severity critical -silent` — CVE-only batch scan
```

---

## 四、Skill 边界规则

### 4.1 AC-Echo prompt 改动

在现有 `## MCP Tools` 和 `## Workflow` 之间插入两个新段：

```markdown
## Kali Tool Skills

The following skills are available via kali-mcp. When a task matches a skill's trigger, follow its exact workflow — do NOT improvise commands.

- **port-scanning**: naabu + nmap. Trigger: "scan ports", "find open ports", "discover services", "what's running".
- **web-probing**: httpx. Trigger: "probe web", "fingerprint", "check HTTP", "what web servers". 🟡 Active — no nuclei.
- **js-analysis**: katana. Trigger: "extract routes", "crawl JS", "find endpoints", "map API".
- **web-fuzzing**: gobuster + ffuf. Trigger: "fuzz", "brute force dirs", "find hidden", "discover params". Uses /data/dictionaries/.
- **web-vuln-scan**: nuclei. Trigger: ONLY on explicit Supervisor authorization. 🔴 Intrusive — IDS/IPS WILL detect.

## Tool Escalation Rules

| Level | Tools | When |
|-------|-------|------|
| 🟢 Passive | curl -I, dig, ping, whoami | Always allowed |
| 🟡 Active | nmap -sV, gobuster, katana, ffuf, httpx | Within scope, record findings |
| 🔴 Intrusive | nuclei, sqlmap --os-shell, nmap scripts, password brute | ONLY on explicit Supervisor order |

You operate at 🟡 Active. NEVER upgrade to 🔴 without explicit authorization. If you discover a target that needs 🔴 tools, report to Supervisor and wait.
```

### 4.2 Supervisor prompt 改动

在 `## Workflow` 后 `## Squad Members` 前插入：

```markdown
## Tool Escalation Boundary

Squad agents operate at assigned tool levels. You control escalation:

| Level | Tools | Agent Authority |
|-------|-------|:---:|
| 🟢 Passive | curl -I, dig, ping, ls, whoami, hostname | Auto-allowed |
| 🟡 Active | nmap -sV, gobuster, katana, ffuf, httpx | Agent decides within scope |
| 🔴 Intrusive | nuclei, sqlmap active, nmap scripts, password brute, mimikatz | Supervisor ONLY |

If an agent requests 🔴 escalation (e.g., "should I run nuclei?"), explicitly approve or reject. Never reply "go ahead" without understanding the IDS/IPS risk. Default to "no, record findings and proceed to next phase."
```

---

## 五、脚本

### `scripts/update-nuclei-templates.sh`

构建镜像前的必要步骤。拉取官方 nuclei-templates + 合并自定义模板到 `data/nuclei-templates/`，之后 Dockerfile COPY 进镜像。

```bash
#!/bin/bash
# Update nuclei-templates from official repo + merge custom templates.
# Usage: ./scripts/update-nuclei-templates.sh

set -euo pipefail

DATA_DIR="$(cd "$(dirname "$0")/../data" && pwd)"
OFFICIAL_DIR="$DATA_DIR/nuclei-templates"
CUSTOM_DIR="$DATA_DIR/nuclei-templates-custom"
OFFICIAL_REPO="https://github.com/projectdiscovery/nuclei-templates.git"

if [ -d "$OFFICIAL_DIR/.git" ]; then
    echo "[1/3] Pulling latest official templates..."
    git -C "$OFFICIAL_DIR" pull --ff-only origin main
else
    echo "[1/3] Cloning official templates (~500MB, one-time)..."
    git clone --depth 1 "$OFFICIAL_REPO" "$OFFICIAL_DIR"
fi

if [ -d "$CUSTOM_DIR" ] && [ "$(ls -A "$CUSTOM_DIR" 2>/dev/null)" ]; then
    echo "[2/3] Merging custom templates..."
    cp -r "$CUSTOM_DIR"/* "$OFFICIAL_DIR"/
    echo "       Custom templates applied."
else
    echo "[2/3] No custom templates to merge."
fi

count=$(find "$OFFICIAL_DIR" -name "*.yaml" -o -name "*.yml" 2>/dev/null | wc -l)
echo "[3/3] Done. $count templates available in $OFFICIAL_DIR"
```

---

## 六、Docker 镜像构建

### `docker/kali-mcp/Dockerfile` — 末尾加

```dockerfile
# Copy dictionaries (small, few KB)
COPY data/dictionaries/ /data/dictionaries/

# Clone and copy nuclei-templates (large, run update-nuclei-templates.sh first)
COPY data/nuclei-templates/ /root/nuclei-templates/
```

**镜像构建前置条件**：构建镜像前必须先运行 `./scripts/update-nuclei-templates.sh` 确保 `data/nuclei-templates/` 存在。

### `docker/docker-compose.yml` — 无需额外挂载

字典和 nuclei 模板已封入镜像，不需要 volume 挂载。

---

## 七、字典 `README.md`

`data/dictionaries/README.md`:

```markdown
# Fuzzing Dictionaries

## Structure

| Directory | Content | Source |
|-----------|---------|--------|
| `dir/` | Directory brute-force wordlists | SecLists/Discovery/Web-Content |
| `api/` | API path wordlists | Custom — common API naming patterns |
| `param/` | Parameter name wordlists | SecLists + custom |
| `pass/` | Password wordlists | SecLists/Passwords/Common-Credentials |

## Updating Dictionaries

Copy files from [SecLists](https://github.com/danielmiessler/SecLists) or add custom entries:

```bash
# Update directory dictionary from SecLists
curl -o dir/common.txt https://raw.githubusercontent.com/danielmiessler/SecLists/master/Discovery/Web-Content/common.txt

# Add custom API paths
echo "/api/internal/users" >> api/common.txt
```

## Custom Entries

Add one entry per line. Blank lines and # comments are ignored by fuzzing tools. Custom entries persist across SecLists updates (they're in separate files).
```

---

## 八、`.gitignore`

```diff
+data/nuclei-templates/
```
