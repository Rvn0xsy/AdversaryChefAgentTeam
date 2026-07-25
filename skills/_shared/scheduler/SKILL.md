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
