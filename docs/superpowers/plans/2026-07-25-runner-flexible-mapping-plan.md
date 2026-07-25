# Runner MCP/Skills/Squad 灵活映射 — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 让 Runner 根据 Agent prompt 头部声明按需挂载 MCP 和 Skills，支持按 Squad 分子目录扩展，不再硬编码。

**Architecture:** Runner 解析 prompt 文件的 `> **Requires**` / `> **Skills**` 字段，查 `_mcp-registry.yaml` 得 MCP URL，查 skills 目录得挂载路径。`_shared/` skills 始终挂载。Squad 通过 `_squads.yaml` 声明，Agent 用 `squad/agent` 命名。

**Tech Stack:** Go 1.26, YAML (`gopkg.in/yaml.v3`), shell scripts

## Global Constraints

- 新增 squad 需零代码改动（只需目录 + yaml 配置）
- `_shared/` 下 skills 始终 mount 到所有 Agent
- Prompt 头部字段解析失败降级为空列表 + warn 日志，不阻塞任务
- MCP 名在 registry 中不存在时 warn + 跳过，不阻塞任务
- Agent 命名统一为 `squad/agent` 格式

---

### Task 1: 重组目录结构 — prompts 分子目录

**Files:**
- Create: `prompts/red-team/` (目录)
- Move: `prompts/*.md` → `prompts/red-team/` (8 个 agent + squad.md)
- Keep: `prompts/_tools/`, `prompts/_tests/`, `prompts/README.md` 原位

- [ ] **Step 1: 创建 red-team 子目录并移动 agent prompt 文件**

```bash
mkdir -p prompts/red-team
mv prompts/supervisor.md    prompts/red-team/
mv prompts/strategist.md    prompts/red-team/
mv prompts/echo-recon.md    prompts/red-team/
mv prompts/breach-exploit.md  prompts/red-team/
mv prompts/ghost-mythic.md  prompts/red-team/
mv prompts/path-lateral.md  prompts/red-team/
mv prompts/forge-resource.md  prompts/red-team/
mv prompts/quill-report.md  prompts/red-team/
mv prompts/squad.md         prompts/red-team/
```

- [ ] **Step 2: 验证移动结果**

```bash
ls prompts/red-team/
# 预期: breach-exploit.md echo-recon.md forge-resource.md ghost-mythic.md
#        path-lateral.md quill-report.md squad.md strategist.md supervisor.md
```

- [ ] **Step 3: 更新 prompts/README.md 中的目录结构说明**

```bash
# 把 README 中 "Agent Roster" 表格的 File 列路径从 supervisor.md 改为 red-team/supervisor.md
```

- [ ] **Step 4: 更新 prompts/squad.md 中 squad 名描述**

确保 squad.md 内容引用正确的相对路径（可选，因为该文件之后会随脚本迁移）

- [ ] **Step 5: 为所有 agent prompt 添加 `> **Skills**` 头部字段**

编辑 8 个 agent 文件，在 `> **Requires**` 行后面加上 `> **Skills**` 行：

| Agent 文件 | 添加行 |
|---|---|
| `supervisor.md` | `> **Skills**: red-team/_none` (或省略该行) |
| `strategist.md` | `> **Skills**: red-team/_none` |
| `echo-recon.md` | `> **Skills**: red-team/kali` |
| `breach-exploit.md` | `> **Skills**: red-team/kali` |
| `ghost-mythic.md` | `> **Skills**: ` (空，不需要 kali) |
| `path-lateral.md` | `> **Skills**: red-team/kali` |
| `forge-resource.md` | `> **Skills**: ` (空) |
| `quill-report.md` | `> **Skills**: ` (空) |

实际编辑示例 (`prompts/red-team/echo-recon.md`)：
```markdown
> **Purpose**: Map the external attack surface
> **Requires**: nexus-mcp, kali-mcp
> **Skills**: red-team/kali
> **Input**: Target domain, IP range, or URL
```

- [ ] **Step 6: Commit**

```bash
git add prompts/
git commit -m "refactor: move agent prompts to red-team/ subdirectory, add Skills headers"
```

---

### Task 2: 重组 skills 目录结构

**Files:**
- Create: `skills/red-team/` (目录)
- Move: `skills/kali/` → `skills/red-team/kali/`
- Create: `skills/_shared/` (目录)
- Create: `skills/_shared/scheduler/SKILL.md`

- [ ] **Step 1: 移动 kali skill 到 red-team 子目录**

```bash
mkdir -p skills/red-team
mv skills/kali skills/red-team/
```

- [ ] **Step 2: 创建 _shared/scheduler skill**

```bash
mkdir -p skills/_shared/scheduler
```

创建 `skills/_shared/scheduler/SKILL.md`:
```markdown
---
name: scheduler
description: Acasched task lifecycle tools for agent self-management. All agents must use scheduler_complete_task before exiting.
---

# Scheduler Task Lifecycle

> **Requires**: scheduler_create_task, scheduler_complete_task (provided via MCP)

## Task Lifecycle Rules

- When you complete your assigned task, you **MUST** call `scheduler_complete_task` with a summary of results.
- To delegate sub-work, use `scheduler_create_task` to spawn a child task to another agent.
- Do **NOT** exit or hang without calling `scheduler_complete_task` — the scheduler will retry failed tasks.
- If your task is blocked (missing info, tool failure), still call `scheduler_complete_task` with a descriptive error result.

## Delegation

When you need another agent's help:
1. Call `scheduler_create_task(agent="red-team/<name>", description="...")` 
2. Wait — the scheduler handles lifecycle, you don't poll
3. On re-trigger, check child task results and continue

## Error Handling

| Situation | Action |
|-----------|--------|
| Tool unavailable | Call `scheduler_complete_task` with error details |
| Ambiguous instructions | Call `scheduler_complete_task` asking for clarification |
| Task complete with findings | Call `scheduler_complete_task` with structured summary |
```

- [ ] **Step 3: 验证结构**

```bash
find skills -type f | sort
# skills/_shared/scheduler/SKILL.md
# skills/red-team/kali/SKILL.md
# skills/red-team/kali/playbooks/*.md
# skills/red-team/kali/reference/*.md
# skills/red-team/kali/tricks/.gitkeep
```

- [ ] **Step 4: Commit**

```bash
git add skills/
git commit -m "refactor: move kali to skills/red-team/, add shared scheduler skill"
```

---

### Task 3: 创建配置文件 `_mcp-registry.yaml` 和 `_squads.yaml`

**Files:**
- Create: `prompts/_mcp-registry.yaml`
- Create: `prompts/_squads.yaml`

- [ ] **Step 1: 创建 `prompts/_mcp-registry.yaml`**

```bash
cat > prompts/_mcp-registry.yaml << 'EOF'
# MCP Registry — maps MCP names to URLs
# Runner reads this to resolve agent "Requires" declarations.
# Supports both local (http://127.0.0.1) and internet (https://api.example.com) MCPs.

nexus-mcp:  "http://127.0.0.1:8081"
kali-mcp:   "http://127.0.0.1:8080"
mythic-mcp: "http://127.0.0.1:8082"

# Internet MCP examples (uncomment when needed):
# shodan-mcp:      "https://api.shodan.io/mcp"
# virustotal-mcp:  "https://www.virustotal.com/mcp"
# semgrep-mcp:     "https://semgrep.internal/mcp"
EOF
```

- [ ] **Step 2: 创建 `prompts/_squads.yaml`**

```bash
cat > prompts/_squads.yaml << 'EOF'
# Squad Registry — declares available squads and their directory mappings
# To add a new squad: add an entry here + create prompts/<dir>/ and skills/<dir>/

squads:
  red-team:
    description: "攻防厨师团 — Red Team Operations"
    prompt_dir: "red-team"
    skill_dir: "red-team"
EOF
```

- [ ] **Step 3: Commit**

```bash
git add prompts/_mcp-registry.yaml prompts/_squads.yaml
git commit -m "feat: add MCP registry and squad registry config files"
```

---

### Task 4: 新增 Prompt 解析器 — `cmd/acasched/internal/goose/prompt.go`

**Files:**
- Create: `cmd/acasched/internal/goose/prompt.go`

**Interfaces:**
- Produces: `type PromptMeta struct { Requires []string; Skills []string }`
- Produces: `func ParsePromptMeta(content []byte) PromptMeta`

- [ ] **Step 1: 创建 prompt.go — 解析 `> **Requires**` 和 `> **Skills**` 字段**

```go
package goose

import (
	"regexp"
	"strings"
)

// PromptMeta holds parsed agent prompt header fields.
type PromptMeta struct {
	Requires []string // MCP names from "> **Requires**: ..."
	Skills   []string // Skill paths from "> **Skills**: ..."
}

var (
	reRequires = regexp.MustCompile(`\*{1,2}Requires\*{1,2}:\s*(.*)`)
	reSkills   = regexp.MustCompile(`\*{1,2}Skills\*{1,2}:\s*(.*)`)
)

// ParsePromptMeta extracts Requires and Skills from a prompt file's header block.
// Handles the format: > **Requires**: nexus-mcp, kali-mcp
func ParsePromptMeta(content []byte) PromptMeta {
	meta := PromptMeta{}
	text := string(content)

	if m := reRequires.FindStringSubmatch(text); len(m) >= 2 {
		meta.Requires = splitTrim(m[1])
	}
	if m := reSkills.FindStringSubmatch(text); len(m) >= 2 {
		meta.Skills = splitTrim(m[1])
	}
	return meta
}

func splitTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
```

- [ ] **Step 2: 写测试 `cmd/acasched/internal/goose/prompt_test.go`**

```go
package goose

import (
	"reflect"
	"testing"
)

func TestParsePromptMeta(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    PromptMeta
	}{
		{
			name: "echo-recon",
			content: `# AC-Echo
> **Purpose**: Map external attack surface
> **Requires**: nexus-mcp, kali-mcp
> **Skills**: red-team/kali
> **Input**: Target domain`,
			want: PromptMeta{
				Requires: []string{"nexus-mcp", "kali-mcp"},
				Skills:   []string{"red-team/kali"},
			},
		},
		{
			name: "quill-report - no skills",
			content: `# AC-Quill
> **Purpose**: Generate reports
> **Requires**: nexus-mcp
> **Skills**:
> **Input**: Project ID`,
			want: PromptMeta{
				Requires: []string{"nexus-mcp"},
				Skills:   nil,
			},
		},
		{
			name: "no fields",
			content: `# No metadata agent
Just does stuff.`,
			want: PromptMeta{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParsePromptMeta([]byte(tt.content))
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParsePromptMeta() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
```

- [ ] **Step 3: 运行测试**

```bash
cd cmd/acasched && go test ./internal/goose/ -run TestParsePromptMeta -v
# 预期: PASS
```

- [ ] **Step 4: Commit**

```bash
git add cmd/acasched/internal/goose/prompt.go cmd/acasched/internal/goose/prompt_test.go
git commit -m "feat: add prompt header parser for Requires/Skills fields"
```

---

### Task 5: 新增配置加载器 — `cmd/acasched/internal/goose/config.go`

**Files:**
- Create: `cmd/acasched/internal/goose/config.go`

**Interfaces:**
- Produces: `func LoadRegistry(path string) (map[string]string, error)` — loads _mcp-registry.yaml
- Produces: `func LoadSquads(path string) (map[string]SquadConfig, error)` — loads _squads.yaml
- Produces: `type SquadConfig struct { Description string; PromptDir string; SkillDir string }`

- [ ] **Step 1: 安装 yaml 依赖**

```bash
cd cmd/acasched && go get gopkg.in/yaml.v3
```

- [ ] **Step 2: 创建 config.go**

```go
package goose

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SquadConfig holds a squad's directory mapping from _squads.yaml.
type SquadConfig struct {
	Description string `yaml:"description"`
	PromptDir   string `yaml:"prompt_dir"`
	SkillDir    string `yaml:"skill_dir"`
}

type squadFile struct {
	Squads map[string]SquadConfig `yaml:"squads"`
}

// LoadRegistry reads _mcp-registry.yaml and returns MCP name → URL map.
func LoadRegistry(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry: %w", err)
	}
	registry := map[string]string{}
	if err := yaml.Unmarshal(data, &registry); err != nil {
		return nil, fmt.Errorf("parse registry: %w", err)
	}
	return registry, nil
}

// LoadSquads reads _squads.yaml and returns squad name → SquadConfig map.
func LoadSquads(path string) (map[string]SquadConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read squads: %w", err)
	}
	var sf squadFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("parse squads: %w", err)
	}
	return sf.Squads, nil
}
```

- [ ] **Step 3: 写测试 `cmd/acasched/internal/goose/config_test.go`**

```go
package goose

import (
	"os"
	"testing"
)

func TestLoadRegistry(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/registry.yaml"
	os.WriteFile(path, []byte("nexus-mcp: http://127.0.0.1:8081\nkali-mcp: http://127.0.0.1:8080\n"), 0644)

	reg, err := LoadRegistry(path)
	if err != nil {
		t.Fatalf("LoadRegistry: %v", err)
	}
	if reg["nexus-mcp"] != "http://127.0.0.1:8081" {
		t.Errorf("got nexus-mcp=%q", reg["nexus-mcp"])
	}
	if reg["kali-mcp"] != "http://127.0.0.1:8080" {
		t.Errorf("got kali-mcp=%q", reg["kali-mcp"])
	}
}

func TestLoadRegistryMissing(t *testing.T) {
	_, err := LoadRegistry("/nonexistent.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadSquads(t *testing.T) {
	tmp := t.TempDir()
	path := tmp + "/squads.yaml"
	os.WriteFile(path, []byte(`squads:
  red-team:
    description: "Red Team"
    prompt_dir: "red-team"
    skill_dir: "red-team"
`), 0644)

	squads, err := LoadSquads(path)
	if err != nil {
		t.Fatalf("LoadSquads: %v", err)
	}
	if _, ok := squads["red-team"]; !ok {
		t.Error("missing red-team squad")
	}
	if squads["red-team"].PromptDir != "red-team" {
		t.Errorf("prompt_dir=%q", squads["red-team"].PromptDir)
	}
}
```

- [ ] **Step 4: 运行测试**

```bash
cd cmd/acasched && go test ./internal/goose/ -run TestLoad -v
# 预期: PASS
```

- [ ] **Step 5: Commit**

```bash
git add cmd/acasched/internal/goose/config.go cmd/acasched/internal/goose/config_test.go cmd/acasched/go.mod cmd/acasched/go.sum
git commit -m "feat: add config loader for MCP registry and squad registry"
```

---

### Task 6: 改造 Runner — 按需挂载 MCP 和 Skills

**Files:**
- Modify: `cmd/acasched/internal/goose/runner.go` (完全重写 Execute 方法中的 MCP/Skills 部分)

**Interfaces:**
- Consumes: `PromptMeta` from prompt.go, `LoadRegistry`, `LoadSquads` from config.go
- Produces: Same `Runner.Execute(ctx, task) (*Result, error)` signature, 但 Runner 结构体改变

- [ ] **Step 1: 更新 Runner 结构体**

将 `cmd/acasched/internal/goose/runner.go` 中的 Runner 结构体从：

```go
type Runner struct {
	PromptsDir string
	WorkDir    string
	LogDir     string
	NexusMCP   string
	KaliMCP    string
	MythicMCP  string
}
```

改为：

```go
type Runner struct {
	PromptsDir string   // e.g., "prompts"
	SkillsDir  string   // e.g., "skills"
	LogDir     string
	Registry   map[string]string // loaded from _mcp-registry.yaml
}
```

删除 `WorkDir` (未使用), `NexusMCP`, `KaliMCP`, `MythicMCP` 字段。

- [ ] **Step 2: 重写 Execute 中的 MCP 扩展挂载逻辑**

将 `Execute` 方法中硬编码的 MCP 部分：

```go
if r.NexusMCP != "" {
    args = append(args, "--with-streamable-http-extension", r.NexusMCP)
}
if r.KaliMCP != "" {
    args = append(args, "--with-streamable-http-extension", r.KaliMCP)
}
if r.MythicMCP != "" {
    args = append(args, "--with-streamable-http-extension", r.MythicMCP)
}
```

替换为：

```go
// Parse prompt for MCP requirements
meta := ParsePromptMeta(agentPrompt)

// Mount only required MCP extensions
for _, mcpName := range meta.Requires {
    url, ok := r.Registry[mcpName]
    if !ok {
        log.Printf("runner: MCP %q not found in registry, skipping", mcpName)
        continue
    }
    args = append(args, "--with-streamable-http-extension", url)
}
```

- [ ] **Step 3: 添加 Skills 挂载逻辑**

在 MCP 挂载之后、指令文件挂载之前插入：

```go
// Mount _shared skills (always)
sharedSkillsDir := filepath.Join(r.SkillsDir, "_shared")
if entries, err := os.ReadDir(sharedSkillsDir); err == nil {
    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }
        hostPath := filepath.Join(sharedSkillsDir, entry.Name())
        containerPath := "/root/.agents/skills/" + entry.Name()
        args = append(args, "-v", hostPath+":"+containerPath+":ro")
    }
}

// Mount agent-specific skills
for _, skill := range meta.Skills {
    hostPath := filepath.Join(r.SkillsDir, skill)
    if _, err := os.Stat(hostPath); os.IsNotExist(err) {
        log.Printf("runner: skill dir %q not found, skipping", hostPath)
        continue
    }
    skillName := filepath.Base(skill)
    containerPath := "/root/.agents/skills/" + skillName
    args = append(args, "-v", hostPath+":"+containerPath+":ro")
}
```

- [ ] **Step 4: 更新 prompt 文件读取路径**

将：

```go
agentPrompt, err := os.ReadFile(r.PromptsDir + "/" + task.Agent + ".md")
```

改为按 squad/agent 格式定位：

```go
// task.Agent format: "red-team/echo-recon"
parts := strings.SplitN(task.Agent, "/", 2)
agentPath := task.Agent + ".md"
if len(parts) == 2 {
    agentPath = filepath.Join(r.PromptsDir, parts[0], parts[1]+".md")
}
agentPrompt, err := os.ReadFile(agentPath)
```

- [ ] **Step 5: 添加需要的 import**

新增 import: `"log"`, `"strings"`, `"path/filepath"`（filepath 已在 runner.go 中）

- [ ] **Step 6: 编译验证**

```bash
cd cmd/acasched && go build ./...
# 预期: 编译成功
```

- [ ] **Step 7: Commit**

```bash
git add cmd/acasched/internal/goose/runner.go
git commit -m "refactor: runner mounts MCP and skills selectively from prompt metadata"
```

---

### Task 7: 更新 main.go — 简化启动参数

**Files:**
- Modify: `cmd/acasched/main.go`

- [ ] **Step 1: 替换 CLI flags**

删除：
```go
nexusURL := flag.String("nexus-mcp", "http://127.0.0.1:8081", "nexus-mcp URL")
kaliURL := flag.String("kali-mcp", "http://127.0.0.1:8080", "kali-mcp URL")
mythicURL := flag.String("mythic-mcp", "http://127.0.0.1:8082", "mythic-mcp URL")
```

改为：
```go
skillsDir := flag.String("skills", "skills", "skills directory")
mcpRegistry := flag.String("mcp-registry", "prompts/_mcp-registry.yaml", "MCP registry file")
```

- [ ] **Step 2: 替换 Runner 初始化**

将：

```go
runner := &goose.Runner{
    PromptsDir: *promptsDir,
    LogDir:     *logDir,
    NexusMCP:   *nexusURL,
    KaliMCP:    *kaliURL,
    MythicMCP:  *mythicURL,
}
```

改为：

```go
registry, err := goose.LoadRegistry(*mcpRegistry)
if err != nil {
    log.Fatalf("load mcp registry: %v", err)
}

runner := &goose.Runner{
    PromptsDir: *promptsDir,
    SkillsDir:  *skillsDir,
    LogDir:     *logDir,
    Registry:   registry,
}
```

- [ ] **Step 3: 编译验证**

```bash
cd cmd/acasched && go build ./...
# 预期: 编译成功
```

- [ ] **Step 4: Commit**

```bash
git add cmd/acasched/main.go
git commit -m "refactor: replace hardcoded MCP flags with registry + skills flags in acasched"
```

---

### Task 8: 更新 acactl up.go — 适配新参数

**Files:**
- Modify: `cmd/acactl/commands/up.go`

- [ ] **Step 1: 替换 acasched 启动参数**

将：

```go
Args: []string{
    "-db", acaschedDB,
    "-nexus-mcp", fmt.Sprintf("http://127.0.0.1:%d", nexusPort),
    "-kali-mcp", fmt.Sprintf("http://127.0.0.1:%d", kaliPort),
    "-prompts", filepath.Join(projectRoot, "prompts"),
    "-log-dir", logDir,
},
```

改为：

```go
Args: []string{
    "-db", acaschedDB,
    "-prompts", filepath.Join(projectRoot, "prompts"),
    "-skills", filepath.Join(projectRoot, "skills"),
    "-mcp-registry", filepath.Join(projectRoot, "prompts", "_mcp-registry.yaml"),
    "-log-dir", logDir,
},
```

- [ ] **Step 2: 编译验证**

```bash
cd cmd/acactl && go build ./...
# 预期: 编译成功
```

- [ ] **Step 3: 整体编译验证**

```bash
go build ./cmd/acasched/... ./cmd/acactl/...
# 预期: 编译成功
```

- [ ] **Step 4: 运行现有单元测试**

```bash
cd cmd/acasched && go test ./... -v
# 预期: 全部 PASS
```

- [ ] **Step 5: Commit**

```bash
git add cmd/acactl/commands/up.go
git commit -m "refactor: acactl up passes skills and registry flags instead of individual MCP URLs"
```

---

### Task 9: 更新 docker-compose.yml 和 prompts/README.md

**Files:**
- Modify: `docker/docker-compose.yml` (已无 acasched 服务，确认即可)
- Modify: `prompts/README.md` (更新 agent file 路径引用为 red-team/)

- [ ] **Step 1: 更新 prompts/README.md 中的路径引用**

将 Agent Roster 表格中的 File 列路径加上 `red-team/` 前缀：

```markdown
| Agent | File | MCP Required |
|-------|------|:---:|
| AC-Supervisor | `red-team/supervisor.md` | nexus |
| AC-Strategist | `red-team/strategist.md` | nexus |
| AC-Echo | `red-team/echo-recon.md` | nexus, kali |
...
```

- [ ] **Step 2: 更新 Quick Start 部分**

提到 `prompts/` → `prompts/red-team/`，以及新的 `Skills` 头部字段。

- [ ] **Step 3: 添加 MCP Registry 和 Squad Registry 说明**

```markdown
## Configuration Files

### `_mcp-registry.yaml`
Maps MCP names to URLs. Agents declare which MCPs they need via `Requires` field.
Supports both local (`http://127.0.0.1`) and internet (`https://api.example.com`) URLs.

### `_squads.yaml`
Declares available squads. To add a new squad, add an entry here + create `prompts/<dir>/` and `skills/<dir>/`.

### Adding a New Squad
1. Create `prompts/<squad>/` with agent `.md` files
2. Create `skills/<squad>/` with squad-specific skills
3. Add entry to `_squads.yaml`
4. Add MCP entries to `_mcp-registry.yaml`
5. Done — zero code changes
```

- [ ] **Step 4: Commit**

```bash
git add prompts/README.md
git commit -m "docs: update README for squad directory structure and config files"
```

---

### Task 10: 端到端验证

- [ ] **Step 1: 确认目录结构正确**

```bash
echo "=== prompts ==="
find prompts -maxdepth 2 -type f | sort
echo "=== skills ==="
find skills -maxdepth 3 -type f | sort
# 预期: 
# prompts/_mcp-registry.yaml, _squads.yaml, _tools/*.md, _tests/*.md, README.md
# prompts/red-team/supervisor.md, echo-recon.md, ...  
# skills/_shared/scheduler/SKILL.md
# skills/red-team/kali/SKILL.md, ...
```

- [ ] **Step 2: 完整编译 + 测试**

```bash
go build ./...
cd cmd/acasched && go test ./... -v
# 预期: 全部 PASS
```

- [ ] **Step 3: 确认 goose 镜像可用**

```bash
docker run --rm goose --version
# 预期: 1.44.0
```

- [ ] **Step 4: 验证 acasched 启动（不执行任务，只检查参数加载）**

```bash
./acasched -prompts ./prompts -skills ./skills -mcp-registry ./prompts/_mcp-registry.yaml -port 9091 &
sleep 2
curl http://127.0.0.1:9091/health
# 预期: {"status":"ok"}
kill %1
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "chore: final verification — all tests pass, directory structure correct"
```
