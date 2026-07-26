# Contributing to AdversaryChef Agent Team

## Development Workflow

Every change goes through: **Worktree → Branch → PR → Review → Merge**

### 1. Create an isolated worktree

```bash
# Always branch from latest main
git fetch github
git worktree add -b <type>/<description> .worktrees/<description> github/main
cd .worktrees/<description>
```

Branch naming: `<type>/<short-description>`

| Type | Use |
|------|-----|
| `feat/` | New feature or capability |
| `fix/` | Bug fix |
| `docs/` | Documentation only |
| `refactor/` | Code restructuring, no behavior change |
| `prompt/` | Agent prompt changes |

### 2. Develop + commit

```bash
git add <files>
git commit -m "<type>: <description>"
```

Commit message conventions:
- `feat:` — new feature
- `fix:` — bug fix
- `docs:` — documentation
- `refactor:` — code restructuring
- `prompt:` — agent prompt changes

### 3. Push + create PR

```bash
git push github <branch-name>
gh pr create --base main --title "<type>: <description>" --body "$(cat .github/PULL_REQUEST_TEMPLATE.md)"
```

**Before creating PR, verify:**
- [ ] Build passes: `go build ./cmd/acactl ./cmd/acasched ./servers/kali/... ./servers/nexus/... ./servers/mythic/... ./pkg/mcputil`
- [ ] Prompt changes: tested with squad flow
- [ ] Spec in `docs/superpowers/specs/` if applicable
- [ ] No `.env`, `.db`, binary files in commit

### 4. Code Review

Request review from at least one team member. Reviewers check:

| Change Type | What to Review |
|-------------|---------------|
| Go code | Builds, error handling, logs, no dead code |
| Agent prompts | Boundaries clear, pre-flight gate complete, matches squad manifest |
| Spec docs | Passes self-review checklist, no ambiguity |
| All | No secrets, binaries, or db files committed |

### 5. Merge

```bash
# After approval — merge with merge commit
gh pr merge <branch-name> --merge --delete-branch
```

**Always use `--merge` (not squash, not rebase).** Merge commits preserve branch history for traceability.

### 6. Sync + cleanup

```bash
git checkout main
git pull github main
git push origin main          # Sync to internal Gitea
git worktree remove .worktrees/<description>
```

## Code Standards

### Go

- Build passes: `go build ./cmd/acactl ./cmd/acasched ./servers/kali/... ./servers/nexus/... ./servers/mythic/... ./pkg/mcputil`
- New Go files: package comment at top
- Error handling: always check, always log or return
- Use `log.Printf` over `fmt.Println` in server code

### Agent Prompts

- Every agent MUST have a `🛑 Step 0: Pre-flight Gate` before any workflow
- Every agent MUST have clear `In scope` / `Out of scope` / `DO NOT` sections
- Boundary rules: if an agent lacks prerequisites, call `scheduler_complete_task` immediately — do not cross into another agent's territory
- Circuit breaker: 3 repeated failures → stop. Document the reason.

### Spec Documents

- Save to `docs/superpowers/specs/YYYY-MM-DD-<topic>-design.md`
- Self-review before committing: no TODOs, no contradictions, no ambiguity

## Project Architecture

```
cmd/acasched/     Scheduler daemon — event loop, dispatcher, goose runner
cmd/acactl/       CLI — builds + starts services, dispatches tasks
servers/nexus/    Graph DB MCP — CRUD, graph query, webhooks
servers/kali/     Async shell MCP — exec, jobs
servers/mythic/   Mythic C2 proxy MCP — callbacks, tasks, files
prompts/red-team/ Agent prompt files (.md)
docs/superpowers/ Design specs + implementation plans
```

## Questions?

Open an issue or ask in the team channel.
