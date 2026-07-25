# Kali Tool Skills + Dictionary Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create 5 Kali tool scenario skills, dictionary directory, nuclei-templates management script, and tool escalation boundary rules in AC-Echo + Supervisor prompts.

**Architecture:** Markdown SKILL.md files in `skills/kali/` with Super Powers format. Dictionaries are text files under `data/dictionaries/`. Prompts modified to reference skills and enforce 🟢🟡🔴 tool escalation levels.

**Tech Stack:** Markdown, Bash, Dockerfile COPY directives

## Global Constraints

- All skill files use Super Powers format: YAML frontmatter + Purpose/Requires/Input/Output header + Boundaries + Workflow + Error Recovery + Autonomy Rules + Verified Patterns
- Tool levels: 🟢 Passive (always), 🟡 Active (within scope), 🔴 Intrusive (Supervisor only)
- web-vuln-scan skill must NOT run unless explicitly ordered
- web-probing skill must NOT reference nuclei
- Dictionaries and nuclei-templates baked into Docker image via COPY (not VOLUME)
- update-nuclei-templates.sh is pre-build step; dictionaries manually populated

---

## File Map

```
skills/kali/                              (new — 5 dirs + SKILL.md each)
├── port-scanning/SKILL.md
├── web-probing/SKILL.md
├── js-analysis/SKILL.md
├── web-fuzzing/SKILL.md
└── web-vuln-scan/SKILL.md

data/dictionaries/                        (new — 5 files)
├── README.md
├── dir/common.txt
├── api/common.txt
├── param/common.txt
└── pass/top100.txt

data/nuclei-templates-custom/             (new)
└── README.md

scripts/update-nuclei-templates.sh        (new)

prompts/echo-recon.md                     (modify)
prompts/supervisor.md                     (modify)
.gitignore                                (modify)
docker/kali-mcp/Dockerfile                (modify)
```

---

### Task 1: Scaffold directories + dictionary files

**Files:**
- Create: `data/dictionaries/README.md`
- Create: `data/dictionaries/dir/common.txt`
- Create: `data/dictionaries/api/common.txt`
- Create: `data/dictionaries/param/common.txt`
- Create: `data/dictionaries/pass/top100.txt`
- Create: `data/nuclei-templates-custom/README.md`

**Interfaces:**
- Produces: dictionary files available for `web-fuzzing` skill, nuclei custom template directory for `update-nuclei-templates.sh`

- [ ] **Step 1: Create directories**

```bash
mkdir -p skills/kali/port-scanning skills/kali/web-probing skills/kali/js-analysis \
         skills/kali/web-fuzzing skills/kali/web-vuln-scan \
         data/dictionaries/dir data/dictionaries/api data/dictionaries/param data/dictionaries/pass \
         data/nuclei-templates-custom
```

- [ ] **Step 2: Write `data/dictionaries/README.md`**

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

- [ ] **Step 3: Write `data/dictionaries/dir/common.txt`**

```
admin
backup
config
db
dev
.git
.gitignore
login
logout
api
test
tmp
upload
logs
assets
static
includes
src
vendor
node_modules
wp-admin
wp-content
wp-includes
robots.txt
sitemap.xml
.env
.env.backup
.htaccess
.bash_history
.bashrc
composer.json
package.json
Dockerfile
README.md
web.config
crossdomain.xml
phpinfo.php
info.php
test.php
status
health
metrics
actuator
swagger
swagger-ui
graphql
api-docs
docs
documentation
wiki
confluence
jira
jenkins
gitlab
phpmyadmin
mysql
adminer
```

- [ ] **Step 4: Write `data/dictionaries/api/common.txt`**

```
users
user
login
logout
register
signup
auth
token
refresh
password
reset
profile
account
settings
config
admin
dashboard
search
upload
download
export
import
files
images
documents
notifications
messages
chat
webhook
callback
health
status
version
info
ping
metrics
stats
api/v1
api/v2
graphql
rest
oauth
sso
saml
```

- [ ] **Step 5: Write `data/dictionaries/param/common.txt`**

```
id
user
user_id
username
name
email
password
token
key
api_key
secret
file
path
url
redirect
return
next
page
limit
offset
sort
order
filter
search
q
query
type
action
cmd
command
exec
debug
admin
role
group
access
lang
locale
format
callback
jsonp
```

- [ ] **Step 6: Write `data/dictionaries/pass/top100.txt`**

```
password
123456
12345678
qwerty
abc123
monkey
1234567
letmein
trustno1
dragon
baseball
iloveyou
master
sunshine
ashley
bailey
shadow
123123
654321
superman
qazwsx
michael
football
admin
admin123
root
toor
secret
changeme
Passw0rd
P@ssw0rd
p@ssw0rd
Pa$$w0rd
password123
admin@123
Welcome1
Welcome123
test
test123
guest
user
user123
temp
temp123
sa
oracle
postgres
mysql
ftp
cisco
juniper
netgear
linksys
default
```

- [ ] **Step 7: Write `data/nuclei-templates-custom/README.md`**

```markdown
# Custom Nuclei Templates

Place custom YAML nuclei templates here. These will be merged over the official templates
when running `scripts/update-nuclei-templates.sh`.

## Directory Structure

Match the official nuclei-templates structure:

```
cves/           — CVE-specific templates
vulnerabilities/ — General vulnerability templates
exposures/      — Exposure/misconfiguration templates
misconfig/      — Misconfiguration templates
workflows/      — Multi-step workflow templates
```

## Template Format

Standard nuclei YAML format:

```yaml
id: my-custom-check
info:
  name: My Custom Vulnerability Check
  author: rvn0xsy
  severity: medium
  description: Detects custom misconfiguration
requests:
  - method: GET
    path:
      - "{{BaseURL}}/internal/status"
    matchers:
      - type: word
        words:
          - "internal service"
```

## Updating Custom Templates

1. Add/modify `.yaml` files in this directory
2. Run `./scripts/update-nuclei-templates.sh` to merge into official templates
3. Rebuild kali-mcp Docker image
```

- [ ] **Step 8: Run validation**

```bash
ls data/dictionaries/dir/common.txt data/dictionaries/api/common.txt \
   data/dictionaries/param/common.txt data/dictionaries/pass/top100.txt
# Expected: all 4 files exist, non-empty
wc -l data/dictionaries/pass/top100.txt
# Expected: 50
```

- [ ] **Step 9: Commit**

```bash
git add data/dictionaries/ data/nuclei-templates-custom/
git commit -m "feat: dictionary directory + nuclei custom template scaffold"
```

---

### Task 2: Write 5 Kali tool SKILL.md files

**Files:**
- Create: `skills/kali/port-scanning/SKILL.md`
- Create: `skills/kali/web-probing/SKILL.md`
- Create: `skills/kali/js-analysis/SKILL.md`
- Create: `skills/kali/web-fuzzing/SKILL.md`
- Create: `skills/kali/web-vuln-scan/SKILL.md`

**Interfaces:**
- Consumes: `data/dictionaries/` directory (referenced by web-fuzzing skill)
- Produces: 5 SKILL.md files for AC-Echo prompt to reference

- [ ] **Step 1: Write `skills/kali/port-scanning/SKILL.md`**

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

- [ ] **Step 2: Write `skills/kali/web-probing/SKILL.md`**

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
1. `exec` with `echo -e "<url1>\n<url2>" | httpx -status-code -server -title -tech-detect -silent`
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
- **Escalate to supervisor**: All URLs return 403/blocked (potential WAF). Discovery of exposed admin panels. Discovery of unauthenticated API docs (Swagger/GraphQL).

## Verified Patterns
- `cat urls.txt | httpx -status-code -server -title -tech-detect -silent -nc` — full probe, no color
- `httpx -u https://target.com -status-code -title -follow-redirects` — single URL with redirects
```

- [ ] **Step 3: Write `skills/kali/js-analysis/SKILL.md`**

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
- **Out of scope**: Directory brute-force (hand off to web-fuzzing skill). Parameter fuzzing (hand off to web-fuzzing skill). Active vulnerability scanning — NEVER run nuclei from this skill. Authentication testing.

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
- `katana -u https://target.com -jc -em json,woff,png,svg -silent` — skip static assets
```

- [ ] **Step 4: Write `skills/kali/web-fuzzing/SKILL.md`**

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
- **Out of scope**: Password brute-force (🟡 only on explicit order — use pass/top100.txt). Vulnerability scanning — NEVER run nuclei. Exploitation (hand off to AC-Breach)

## Dictionaries
All dictionaries are mounted at `/data/dictionaries/`:
| Path | Content | When |
|------|---------|------|
| `/data/dictionaries/dir/common.txt` | Common web directories | Directory brute-force |
| `/data/dictionaries/api/common.txt` | Common API paths | API endpoint discovery |
| `/data/dictionaries/param/common.txt` | Common parameter names | Parameter fuzzing |
| `/data/dictionaries/pass/top100.txt` | Top 50 weak passwords | Password brute (🟡 only on order) |

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
- `gobuster dir -u https://target.com -w /data/dictionaries/dir/common.txt -t 20 -q` — standard brute
- `ffuf -u https://target.com/api/user?id=FUZZ -w /data/dictionaries/param/common.txt -mc 200 -t 20` — param fuzz with IDOR potential
- `ffuf -u https://target.com/FUZZ -w /data/dictionaries/dir/common.txt -mc 200,403 -t 20` — alternative to gobuster
```

- [ ] **Step 5: Write `skills/kali/web-vuln-scan/SKILL.md`**

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
| nuclei "no templates found" | Verify `/root/nuclei-templates/` is available. Report to supervisor. |
| nuclei "target not reachable" | Double-check URL with `curl -I` |
| nuclei times out | Reduce template scope: `-t /root/nuclei-templates/cves/` |

## Autonomy Rules
- **Proceed without asking**: NEVER. Nuclei is 🔴 Intrusive. Every run requires explicit authorization.
- **Escalate to supervisor**: After scanning completes — report findings immediately. If target blocks the scan mid-way — report and stop.

## Verified Patterns
- `nuclei -u https://target.com -t /root/nuclei-templates/ -severity critical,high -silent` — standard scan
- `nuclei -list targets.txt -t /root/nuclei-templates/cves/ -severity critical -silent` — CVE-only batch scan
```

- [ ] **Step 6: Verify all 5 skill files exist**

```bash
ls skills/kali/*/SKILL.md
# Expected: 5 files
for f in skills/kali/*/SKILL.md; do echo "=== $f ===" && head -5 "$f"; done
# Expected: each has YAML frontmatter with name + description
```

- [ ] **Step 7: Commit**

```bash
git add skills/kali/
git commit -m "feat: 5 Kali tool scenario skills (port-scanning, web-probing, js-analysis, web-fuzzing, web-vuln-scan)"
```

---

### Task 3: update-nuclei-templates.sh + .gitignore

**Files:**
- Create: `scripts/update-nuclei-templates.sh`
- Modify: `.gitignore`

**Interfaces:**
- Consumes: `data/nuclei-templates-custom/` directory
- Produces: script that populates `data/nuclei-templates/` for Docker COPY

- [ ] **Step 1: Write `scripts/update-nuclei-templates.sh`**

```bash
#!/bin/bash
# Update nuclei-templates from official repo + merge custom templates.
# Run BEFORE building kali-mcp Docker image.
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

- [ ] **Step 2: Make script executable**

```bash
chmod +x scripts/update-nuclei-templates.sh
```

- [ ] **Step 3: Update `.gitignore` — add nuclei-templates**

```bash
echo "" >> .gitignore
echo "# Nuclei official templates (managed by update-nuclei-templates.sh)" >> .gitignore
echo "data/nuclei-templates/" >> .gitignore
```

- [ ] **Step 4: Verify script syntax**

```bash
bash -n scripts/update-nuclei-templates.sh
# Expected: no output (syntax OK)
```

- [ ] **Step 5: Commit**

```bash
git add scripts/update-nuclei-templates.sh .gitignore
git commit -m "feat: nuclei-templates update script + .gitignore"
```

---

### Task 4: Update prompts/echo-recon.md

**Files:**
- Modify: `prompts/echo-recon.md` — insert two new sections between `## MCP Tools` and `## Workflow`

**Interfaces:**
- Consumes: the 5 skill names defined in Task 2
- Produces: AC-Echo prompt with Kali Tool Skills reference + Tool Escalation Rules

- [ ] **Step 1: Insert after `{{TOOLS_KALI}}` line and before `## Workflow`**

The insertion point is after the `{{TOOLS_KALI}}` placeholder line and before the `## Workflow` section header. Content to insert:

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
| 🟢 Passive | curl -I, dig, ping | Always allowed |
| 🟡 Active | nmap -sV, gobuster, katana, ffuf, httpx | Within scope, record findings |
| 🔴 Intrusive | nuclei, sqlmap --os-shell, nmap scripts, password brute | ONLY on explicit Supervisor order |

You operate at 🟡 Active. NEVER upgrade to 🔴 without explicit authorization. If you discover a target that needs 🔴 tools, report to Supervisor and wait.

## Workflow
```

- [ ] **Step 2: Verify the prompt structure**

Run the existing `## Workflow` section check — the insertion must be between `{{TOOLS_KALI}}` and the existing `## Workflow` header:

```bash
grep -n "^## Workflow\|^## MCP Tools\|{{TOOLS_KALI}}" prompts/echo-recon.md
# Expected: order is MCP Tools → {{TOOLS_KALI}} → Kali Tool Skills → Tool Escalation Rules → Workflow
```

- [ ] **Step 3: Run validation**

```bash
./scripts/validate-prompts.sh
# Expected: ok echo-recon.md
```

- [ ] **Step 4: Commit**

```bash
git add prompts/echo-recon.md
git commit -m "feat: AC-Echo Kali tool skills reference + tool escalation rules"
```

---

### Task 5: Update prompts/supervisor.md

**Files:**
- Modify: `prompts/supervisor.md` — insert `## Tool Escalation Boundary` section between `## Workflow` and `## Squad Members`

- [ ] **Step 1: Insert between Workflow and Squad Members**

The insertion point is after the `## Workflow` section's last step and before the `## Squad Members` section. Content to insert:

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

- [ ] **Step 2: Verify insertion point**

```bash
grep -n "^## " prompts/supervisor.md
# Expected: Workflow → Tool Escalation Boundary → Squad Members → ... (in order)
```

- [ ] **Step 3: Run validation**

```bash
./scripts/validate-prompts.sh
# Expected: ok supervisor.md
```

- [ ] **Step 4: Commit**

```bash
git add prompts/supervisor.md
git commit -m "feat: Supervisor tool escalation boundary (🟢🟡🔴)"
```

---

### Task 6: Update Dockerfile + final validation

**Files:**
- Modify: `docker/kali-mcp/Dockerfile` — add COPY directives

- [ ] **Step 1: Add COPY directives to Dockerfile**

Insert at the end of the Dockerfile (after the `CMD` line or last `RUN`):

```dockerfile
# Copy dictionaries (bundled in image)
COPY data/dictionaries/ /data/dictionaries/

# Copy nuclei-templates (must exist — run update-nuclei-templates.sh before building)
COPY data/nuclei-templates/ /root/nuclei-templates/
```

- [ ] **Step 2: Verify Dockerfile syntax**

```bash
cat docker/kali-mcp/Dockerfile
# Expected: COPY directives appear near the end
```

- [ ] **Step 3: Full validation**

```bash
./scripts/validate-prompts.sh
# Expected: all ok
ls skills/kali/*/SKILL.md
# Expected: 5 files
ls data/dictionaries/*/common.txt data/dictionaries/*/top100.txt
# Expected: 4 wordlist files
ls data/nuclei-templates-custom/README.md
# Expected: exists
bash -n scripts/update-nuclei-templates.sh
# Expected: syntax OK (no output)
git diff --stat .gitignore
# Expected: data/nuclei-templates/ added
```

- [ ] **Step 4: Commit**

```bash
git add docker/kali-mcp/Dockerfile
git commit -m "feat: COPY dictionaries + nuclei-templates into kali-mcp image"
```
