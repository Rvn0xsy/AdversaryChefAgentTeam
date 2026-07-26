# Event-Driven Parallel Squad — Design Spec

> **Status**: Draft  
> **Date**: 2026-07-26  
> **Author**: brainstorming session  
> **Scope**: AdversaryChefAgentTeam — nexus-mcp, acasched, AC-Supervisor, AC-Echo  

## Problem

The current squad runs **sequentially**: Supervisor dispatches one agent at a time and waits for completion before dispatching the next. AC-Echo's prompt forces it to use `job_wait` — a blocking MCP call that holds the agent for up to 600 seconds per nmap scan. This creates two problems:

1. **Agent-level blocking**: AC-Echo is stuck waiting for nmap, can't do parallel work (DNS, HTTP probes) within its own task.
2. **Squad-level blocking**: Other agents (AC-Breach, AC-Forge) sit idle waiting for AC-Echo to finish.

The fix operates on both levels: prompt strategy (fire-and-forget) and squad architecture (event-driven parallel dispatch).

## Architecture

```
                        ┌─────────────┐
                        │  nexus-mcp  │
                        │             │
          ┌──write──────│  host/svc/  │──────webhook──────┐
          │             │  endpoint   │   (graph event)   │
          │             └─────────────┘                   │
          │                                               ▼
   ┌──────┴──────┐                              ┌─────────────────┐
   │  AC-Echo    │                              │    acasched     │
   │  fire &     │                              │                 │
   │  forget     │                              │  event loop     │
   │  scan       │                              │  trigger Supe   │
   └─────────────┘                              └────────┬────────┘
                                                         │
                                          ┌──────────────┴──────────────┐
                                          │         Supervisor           │
                                          │  (short-lived instances)     │
                                          │                              │
                                          │  "new service on :8080?"     │
                                          │  → dispatch AC-Breach        │
                                          │  "0 hosts?"                  │
                                          │  → dispatch AC-Echo          │
                                          │  "goal reached?"             │
                                          │  → complete                  │
                                          └──────────────┬──────────────┘
                                                         │
                                          ┌──────────────┼──────────────┐
                                          │              │              │
                                     AC-Echo      AC-Breach      AC-Forge
                                    (parallel)    (parallel)     (parallel)
```

### Data flow

```
nexus write → webhook → acasched → trigger Supervisor
Supervisor: query graph → evaluate → dispatch ALL eligible agents → complete
Agent A finishes → writes clue to nexus → webhook → acasched → trigger Supervisor
Agent B was already running (independent, continues in parallel)
```

## Component Designs

### 1. nexus-mcp — Graph Webhook

**Event schema** (HTTP POST to acasched):

```json
{
  "project_id": "proj_xxx",
  "action":     "create",
  "entity":     "service",
  "node_id":    "svc_xxx",
  "parent_id":  "host_xxx",
  "summary":    ":8080 nginx 1.24.0 open",
  "timestamp":  "2026-07-26T02:00:00Z"
}
```

**Entity types emitted**: `host`, `service`, `endpoint`, `evidence`, `vuln`, `session`.  
**Action**: `create` only (v1). Future: `update` for status transitions.  
**Hook point**: `EventedStore` — a decorator wrapping `store.Store`. Each `Create*` method calls `inner.Create*()` then fires an async `POST` to acasched.  
**Failure mode**: Fire-and-forget. Log warn on delivery failure. No retry — fallback polling covers gaps.

**Target URL**: `http://acasched:9090/api/events` (configurable via env `ACASCHED_WEBHOOK_URL`).

### 2. acasched — Event Loop & Parallel Dispatch

**New endpoint**: `POST /api/events` — receives nexus webhook events.

**Event processor** (new goroutine in main):

```
nexus event → POST /api/events → enqueue (per-project dedup, 5s window)
                                       │
                                       ▼
                             event processor goroutine
                                       │
                              check runningSupervisors[projectID]
                                       │
                           ┌───────────┴───────────┐
                           │ busy (already running) │ idle
                           ▼                         ▼
                          skip              create Supervisor task
                                            mark runningSupervisors=true

Supervisor completes → unmark runningSupervisors[projectID]=false
```

**Deduplication**: Events for the same project within a 5-second window are collapsed into one Supervisor trigger. Prevents burst spam from a single agent writing 20 endpoints at once.

**Fallback polling** (separate goroutine):
- Every 60s, scan all active projects
- If `now - lastEventTime > 120s` and no running Supervisor → trigger Supervisor
- Prevents missed events when nexus→acasched connection drops

**Existing dispatcher**: The dispatcher already runs each pending task in its own goroutine (`go d.dispatchOne(t)`). The only requirement is that Supervisor dispatches multiple children in a single evaluation cycle.

**Child completion trigger** (change to `trigger.go`): When a child task completes and its parent is a Supervisor (identified by agent name prefix `red-team/supervisor`), the existing `TriggerParent` logic is replaced: instead of setting the parent to `pending`, directly enqueue a Supervisor evaluation event. This ensures the Supervisor is re-triggered even when a child finishes but didn't write to nexus yet (e.g., AC-Echo completed a scan, parsed results, but is still writing hosts). Combined with the 5s debounce, this won't cause duplicate Supervisor instances.

**`scheduler_create_task` minimum `max_turns`**: Keep current floor of 150 for long-running agents (AC-Echo, AC-Breach, AC-Path). Short agents (AC-Quill) can use default.

### 3. Supervisor — Watch Loop Decision Model

**Mode**: Short-lived instances. Each trigger from acasched creates a new Supervisor task. The Supervisor evaluates, dispatches all eligible agents, then completes.

**Decision table** (evaluate ALL rows simultaneously, dispatch each row's actions in parallel):

| Graph State | Action |
|-------------|--------|
| hosts == 0 | dispatch AC-Echo + AC-Forge (parallel) |
| Has host with services AND clues/evidence | dispatch AC-Breach |
| Has confirmed vuln, no active session | dispatch AC-Ghost |
| Has active session | dispatch AC-Path |
| Goal reached → dispatch AC-Quill → complete | |
| All agents idle AND graph unchanged for 60s → dispatch AC-Quill → complete | |

**Dispatch guard**: Before dispatching any agent, query acasched for existing pending/running tasks of the same agent+project. Skip if already running.

**Goal definition**: Stored in project description. Supervisor parses natural language and matches against nexus graph nodes:

| Goal phrase | Matched by |
|-------------|-----------|
| "拿到 shell" / "get shell" | SessionNode(host=target) exists |
| "横向到内网" / "lateral movement" | SessionNode on internal IP exists |
| "提取数据" / "exfiltrate" | Clue with credential data exists |

**Completion condition**: All goal clauses satisfied OR all agents idle + graph unchanged for 60s.

### 4. AC-Echo — Fire-and-Forget Prompt

**Strategy**: Three non-blocking phases per cycle.

```
Phase 1: Fire (multiple exec calls, 1 turn each, no waiting)
  exec nmap -p- ...         → record job_id_1
  exec subfinder ...        → record job_id_2
  exec httpx ...            → record job_id_3

Phase 2: Non-blocking work (while scans run in background)
  → query nexus-mcp for existing assets
  → curl -I on known HTTP ports
  → record findings to nexus

Phase 3: Harvest results
  list_jobs                 → check which are done
  get_job(job_id_1)         → parse nmap output → write hosts/services to nexus
  get_job(job_id_2)         → parse DNS output → write hosts to nexus
  get_job(job_id_3)         → parse HTTP output → write endpoints to nexus
  
  If jobs still running → scheduler_complete_task("partial: N/3 scans complete")
                         → will resume next cycle
```

**Tooling rules**:

| Operation | Tool | Notes |
|-----------|------|-------|
| Start scan | `exec` | Fire-and-forget, record job_id |
| Check status | `list_jobs` | Lightweight, no blocking |
| Get results | `get_job` | Only after `list_jobs` shows completed/failed |
| **REMOVED** | ~~`job_wait`~~ | Blocked agent, prevented parallelism |

**Parallelism rules**:
- Launch 3-5 scans at once, don't wait between them
- Don't launch duplicate scans for the same target+tool
- If harvest finds unfinished scans, complete task and resume next cycle
- Rate-limit: don't fire 100 parallel curls against a production target

**Error recovery**:

| Scenario | Action |
|----------|--------|
| exec returns failed | Read stderr, adjust, retry once |
| Job timed out | Reduce scope (--top-ports 100), retry |
| Target rate-limited | Lower --min-rate, add --max-retries |
| Harvest: job still running | Leave for next cycle, complete with partial results |

## Prompt Changes Summary

### `echo-recon.md`

| Section | Change |
|---------|--------|
| **Key Tooling Rule** | Replace "Always use job_wait" with "Use list_jobs + get_job to harvest" |
| Error Recovery table | Replace "poll until completed" with "complete task, resume next cycle" |
| Workflow Phase 1-2 | Phase 1: fire all scans. Phase 2: non-blocking work. Phase 3: harvest. |

### `supervisor.md`

| Section | Change |
|---------|--------|
| Decision Rules | "Dispatch ONE" → "Dispatch ALL eligible agents in parallel" |
| Workflow | "dispatch → wait → next" → "dispatch all → complete" |
| Agent Catalog | Add `AC-Forge` parallel with `AC-Echo` on initial recon |
| New section | Goal parsing from project description |
| New section | Deduplication: check running tasks before dispatch |

## Files Changed

| File | Change |
|------|--------|
| `servers/nexus/internal/store/evented.go` | **NEW** — EventedStore decorator |
| `servers/nexus/internal/tools/register.go` | Wire EventedStore in RegisterAllV3 |
| `servers/nexus/cmd/main.go` | Pass ACASCHED_WEBHOOK_URL env |
| `cmd/acasched/internal/api/events.go` | **NEW** — POST /api/events handler |
| `cmd/acasched/internal/scheduler/trigger.go` | Child completion → enqueue Supervisor eval |
| `cmd/acasched/internal/scheduler/event_loop.go` | **NEW** — event processor + dedup |
| `cmd/acasched/internal/scheduler/fallback_poll.go` | **NEW** — 60s fallback polling |
| `cmd/acasched/internal/api/server.go` | Register /api/events route |
| `cmd/acasched/main.go` | Start event loop + fallback goroutines |
| `prompts/red-team/echo-recon.md` | Fire-and-forget strategy |
| `prompts/red-team/supervisor.md` | Parallel dispatch + goal-driven |

## Migration Plan

1. Deploy nexus-mcp with EventedStore (backward compatible — webhook failures are silent)
2. Deploy acasched with event loop + fallback poll
3. Update agent prompts (echo-recon.md, supervisor.md)
4. Verify: create a test project, observe parallel agent dispatch via acasched task list
