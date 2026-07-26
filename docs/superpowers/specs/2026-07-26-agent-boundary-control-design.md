# Agent Boundary Control — Design Spec

> **Status**: Draft  
> **Date**: 2026-07-26  
> **Scope**: Prompt-only — supervisor.md, echo-recon.md, breach-exploit.md, ghost-mythic.md, path-lateral.md  
> **Depends on**: 2026-07-26-event-driven-parallel-squad-design.md (already implemented)

## Problem

Agents perform work outside their scope and waste tokens on dead-end loops. Two symptoms:

1. **信息不足硬干** — Agent dispatched without enough data starts doing other agents' jobs (e.g., AC-Breach running nmap when it should wait for AC-Echo)
2. **死循环** — Agent hits a wall and keeps retrying the same failing approach, or the target is unreachable but no one declares it dead

## Design

Three layers of defense. No Go code changes — all prompt-level.

### Layer 1: Supervisor Dispatch Pre-conditions

Supervisor validates prerequisites before dispatching ANY agent.

```
Supervisor evaluation cycle:
  1. Full graph query (host, service, endpoint, evidence, vuln, session)
  2. For each candidate agent, check pre-condition matrix
  3. Only dispatch agents with satisfied pre-conditions
  4. Record skip reasons for unsatisfied ones
  5. If none satisfied → complete with "nothing to dispatch"
```

**Pre-condition Matrix:**

| Target Agent | Required Graph Data | Missing? → Instead Dispatch |
|-------------|-------------------|---------------------------|
| AC-Echo | Asset with target IP/domain | No asset → ask user |
| AC-Breach | Host + Service(with port/protocol) + Evidence OR Endpoint | No service → AC-Echo. No evidence → AC-Echo |
| AC-Ghost | Confirmed Vulnerability(status=confirmed) OR Active Session | No vuln → AC-Breach |
| AC-Path | Active Session | No session → AC-Ghost |
| AC-Forge | Always OK (deploy infra) | Dispatch with AC-Echo |
| AC-Quill | Any findings exist | No findings → wait |

**Hard rules:**
- ❌ "先 dispatch 让 Agent 自己想办法" (don't dispatch hoping the agent figures it out)
- ❌ "有一个 host 就够了" (one host is NOT enough for AC-Breach — needs service + evidence)
- ❌ "试试看" (no condition-satisfied = no dispatch)

### Layer 2: Agent Pre-flight Gate

Each agent checks its own prerequisites in Step 0 (before any real work). Takes 1-2 turns. Fails → immediately calls `scheduler_complete_task` with reason. Does NOT attempt to fill gaps.

**AC-Echo pre-flight:**

| Check | Action |
|-------|--------|
| Asset with target IP/domain exists | ✅ Proceed to Phase 1: Fire |
| No asset | ❌ `scheduler_complete_task("No target asset.")` — STOP |

**AC-Breach pre-flight:**

| Check | Action |
|-------|--------|
| Host + Service(port+protocol) + (Evidence or Endpoint) all exist | ✅ Proceed |
| Host missing | ❌ `scheduler_complete_task("No target host. Need AC-Echo.")` |
| Service missing | ❌ `scheduler_complete_task("No service data. Need AC-Echo port scan.")` |
| No evidence or endpoint | ❌ `scheduler_complete_task("Nothing to exploit. No findings from AC-Echo.")` |

**AC-Breach forbidden actions** (enforced by pre-flight + boundary rules):
- ❌ Run nmap, masscan, or any port scanner
- ❌ DNS lookups (host, dig, nslookup)
- ❌ Guess target IPs or scan adjacent hosts
- ❌ Brute-force without confirmed credentials

**AC-Ghost pre-flight:**

| Check | Action |
|-------|--------|
| Confirmed vuln OR active session exists | ✅ Proceed to callback check |
| Neither | ❌ `scheduler_complete_task("No vulnerability confirmed. Need AC-Breach.")` |

**AC-Path pre-flight:**

| Check | Action |
|-------|--------|
| Active session exists | ✅ Proceed |
| No session | ❌ `scheduler_complete_task("No active C2 session. Need AC-Ghost.")` |

### Layer 3: Circuit Breaker

Two-tier dead-end detection: Agent stops itself, Supervisor stops the engagement.

**Agent-side circuit breaker:**

| Signal | Action |
|--------|--------|
| Target unreachable (all ports closed/RST/timeout, 3+ scans) | Complete with "Target unreachable: [evidence]" |
| Rate-limited 3+ times, no new data | Complete with "Rate-limited. Need different source IP." |
| Same tool + same target + same result 3x in a row | STOP. Record reason. Do NOT retry. |
| Completed 3 harvest cycles with 0 new findings | Complete with "Recon exhausted." |
| < 5 turns remaining | Stop launching new scans. Harvest what remains. Complete. |

**Supervisor-side deadlock detection:**

| Signal | Action |
|--------|--------|
| 3+ consecutive evaluations, 0 graph changes | Engagement stuck. Dispatch AC-Quill. Complete. |
| All agents complete with "nothing to do" | Dispatch AC-Quill. Complete. |
| Target confirmed unreachable by AC-Echo | Dispatch AC-Quill with finding. Complete. |
| Goal clearly impossible | Document finding. Dispatch AC-Quill. Complete. |

**State tracking:** Supervisor writes current graph summary (host count, service count, vuln count, session count) in `scheduler_complete_task` result. Next evaluation compares against last result. No graph change = stale cycle.

## Affected Files

| File | Change |
|------|--------|
| `prompts/red-team/supervisor.md` | Add dispatch pre-condition matrix + deadlock detection section |
| `prompts/red-team/echo-recon.md` | Add pre-flight gate (Step 0) + circuit breaker |
| `prompts/red-team/breach-exploit.md` | Add pre-flight gate (Step 0) + circuit breaker + forbidden actions hardening |
| `prompts/red-team/ghost-mythic.md` | Add pre-flight gate (Step 0) |
| `prompts/red-team/path-lateral.md` | Add pre-flight gate (Step 0) |

No Go code changes. No new files. Pure prompt engineering. No nexus-mcp or acasched changes needed.
