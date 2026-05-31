---
name: ant-chrome-openclaw
description: Manually invoke when the user wants OpenClaw to work inside a local Ant Browser / ant-chrome instance through LaunchServer and remote CDP, including launching by launchCode, checking or switching the active runtime target, stopping an instance, or running a saved Ant Browser automation script.
when_to_use: Use this skill whenever the task mentions Ant Browser, ant-chrome, LaunchServer, launchCode, profileId/profileName selectors, runtime active/status/stop, attaching OpenClaw to an already running browser via remote CDP, or executing a saved Ant Browser automation script such as news-query-txt. This workflow can launch, stop, or mutate browser state, so prefer explicit invocation instead of silent automatic use.
compatibility: Designed for OpenClaw with same-host Ant Browser LaunchServer access. Requires loopback access to the LaunchServer base URL plus either curl or PowerShell for host-side HTTP calls.
disable-model-invocation: true
argument-hint: "[goal-or-script]"
metadata: {"openclaw":{"skillKey":"ant-chrome-openclaw","primaryEnv":"ANT_CHROME_API_KEY","requires":{"anyBins":["curl","pwsh","powershell"]}}}
---

# Ant Browser via OpenClaw

Invoke this skill explicitly when the user wants OpenClaw to control browser work that should run inside Ant Browser (`ant-chrome`) instead of a browser launched directly by OpenClaw.

Do not use this skill for generic browsing tasks that should stay inside OpenClaw's own browser runtime.

This skill assumes a same-host setup:

- OpenClaw Gateway and Ant Browser run on the same machine.
- Ant Browser LaunchServer is reachable on loopback.
- OpenClaw Browser has a preconfigured remote CDP profile pointing at the Ant Browser LaunchServer URL.

Read `references/setup.md` in this skill directory when the user is doing first-time setup, the browser profile is missing, or the launch endpoint cannot be reached.

Read `references/http.md` in this skill directory when you need exact request patterns for health checks, launch requests, runtime control, profile CRUD, automation script APIs, or log inspection.

## Defaults

- Base URL: `ANT_CHROME_BASE_URL`, default `http://127.0.0.1:19876`
- API header: `ANT_CHROME_API_HEADER`, default `X-Ant-Api-Key`
- API key: `ANT_CHROME_API_KEY`, optional
- Browser profile name: assume `ant-chrome` unless the user tells you a different OpenClaw browser profile name

## Workflow

1. On Windows, prefer `scripts/invoke_ant_chrome_api.ps1` through `pwsh` or `powershell` for LaunchServer calls. On Unix-like hosts, use the `curl` patterns in `references/http.md`.
2. Before any launch, stop, profile mutation, or automation-script run, verify LaunchServer with `GET /api/health` from the host using `exec`.
3. Choose the narrowest safe action:
   - Check current unified CDP target: `GET /api/runtime/active`
   - Exact launch code: `GET /api/launch/{code}`
   - Selector or launch-code based runtime check: `POST /api/runtime/status`
   - Selector or launch-code based stop: `POST /api/runtime/stop`
   - Selector-based launch: `POST /api/launch`
   - Profile list/create/update/delete: `/api/profiles`
   - Automation script list/detail/run/runs: `/api/automation/scripts`
4. Prefer exact identifiers in this order: `launchCode`, `profileId`, `profileName`, then keyword/tag/group selectors.
5. For selector-based launches, use `matchMode: "unique"` unless the user explicitly wants first-match or all-match behavior.
6. For saved automation scripts, prefer `POST /api/automation/scripts/run` over reconstructing the workflow by hand. If the script is already bound to a target and default params, `scriptId` alone is enough. Only send selector or params overrides when the user explicitly wants a different target or different parameters.
7. After a launch call, or after a script run that should lead to browser attachment, require `ok=true`, a non-empty `cdpUrl`, and `debugReady=true` before using the `browser` tool.
8. If the response contains `runtimeWarning` or `debugReady=false`, stop and report the issue instead of continuing.
9. If a request fails, inspect `GET /api/launch/logs?limit=20` or the recent automation run records before retrying blind.

## Prompt Contract

- Keep user-facing prompts short. For prebuilt script execution, the user should only need to say which script to run, which instance to use, and whether to use default params or a few overrides.
- If a saved script already stores its target and params, treat omitted selector and params as “use the saved defaults”.
- Do not require the user to restate the internal workflow in every prompt. LaunchServer health checks, active-target checks, unique-match handling, `debugReady` validation, and `runtimeWarning` fail-fast behavior come from this skill by default unless the user explicitly overrides them.

## Operating Rules

- Treat Ant Browser as the source of truth for profile state, proxies, tags, launch codes, and Chrome cores.
- Do not switch to another Ant Browser target while the current remote CDP session is still being used unless the user explicitly asks to switch. LaunchServer exposes one active CDP target at a time. Query `GET /api/runtime/active` before switching if the current target matters.
- Do not claim that OpenClaw `browser stop` stops the Ant Browser process. For remote CDP profiles it only detaches the OpenClaw control session.
- Do not invent HTTP routes that do not exist. The current stable surface is `health`, `profiles`, `profile status`, `profile stop`, `runtime active`, `runtime status`, `runtime stop`, `launch`, `launch logs`, and the automation script APIs under `/api/automation/scripts`.
- If selector resolution is ambiguous, stop and ask the user to narrow with `launchCode`, `profileId`, or a stronger selector instead of picking a target silently.
- If setup is incomplete, explain the missing prerequisite and point to `references/setup.md`.
