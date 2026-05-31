---
name: ant-chrome-openclaw-automation-plan
description: Use when implementing, reviewing, auditing, or updating the ant-chrome and OpenClaw integration design, rollout status, or public HTTP API contract. Prefer this skill for low-coupling integration work around runtime/session, runtime/stop, automation/scripts, playwright-cdp execution, and OpenClaw compatibility mapping. Do not use it for day-to-day LaunchServer API invocation; use the sibling ant-chrome-openclaw skill instead.
---

# Ant Browser OpenClaw Integration Plan

Use this skill for engineering work on the integration itself: planning, code changes, API alignment, audits, and rollout decisions.

If the task is operational usage of LaunchServer from OpenClaw, read the sibling skill at `../ant-chrome-openclaw/SKILL.md` instead.

## Read Only What You Need

- Read `references/overview.md` for the architecture split, scope, and recommended integration path.
- Read `references/api-contract.md` for the exact public API surface that exists today.
- Read `references/delivery-status.md` for what is actually shipped versus intentionally deferred.
- Read `references/script-compat.md` for how `playwright-cdp` and script execution should absorb LaunchServer, CDP, and Playwright complexity.

## Working Rules

1. Keep OpenClaw as the orchestration layer and Ant Browser as the execution layer.
2. Prefer public/common APIs first:
   - `POST /api/runtime/session`
   - `POST /api/runtime/status`
   - `POST /api/runtime/stop`
   - `GET /api/automation/scripts`
   - `POST /api/automation/scripts/run`
   - `GET /api/automation/scripts/runs`
3. Do not add `/api/automation/openclaw/*` private routes unless the user explicitly needs a compatibility alias for an already-fixed external protocol.
4. Do not re-invent a second automation runtime for OpenClaw; reuse the existing script execution chain.
5. Keep the external script protocol object-shaped (`selector` / `params`) and only convert to internal text fields at the boundary.
6. Treat `playwright-cdp` as the main complexity-reduction layer for OpenClaw-facing automation.

## Validation

After changing code or the plan:

1. Re-check the current LaunchServer routes under `backend/internal/launchcode`.
2. Run `go test ./backend/test/launchcode/...`.
3. Run `go test ./backend/...`.
4. Run `python C:\Users\Administrator\.codex\skills\.system\skill-creator\scripts\quick_validate.py <this-skill-dir>`.
5. If `SKILL.md` changes meaningfully, regenerate `agents/openai.yaml`.
