# ant-chrome + OpenClaw 自动化整合方案

这不是现成 skill。

这是当前推荐的整合方案和对接文档。

## 目标

把职责拆开：

- OpenClaw 负责对话、任务编排、页面理解
- Ant Browser 负责实例选择、启动、停止、自动化执行

核心目标不是“让 OpenClaw 深度耦合 ant-chrome 内部实现”，而是让 ant-chrome 暴露稳定、通用、可复用的公共入口。

## 设计原则

- 低耦合：不默认做 `/api/automation/openclaw/*` 私有后端协议
- 公共入口优先：优先用通用 `runtime` / `automation/scripts` API
- 稳定性优先：把 `LaunchServer + CDP + Playwright` 复杂度尽量收进 ant-chrome
- 同机优先：当前只建议 `127.0.0.1` 本机调用
- 单实例串行优先：先把一条链路做稳，再谈并发

## 当前推荐接法

### 1. 只想接管浏览器

优先调用：

```text
POST /api/runtime/session
```

作用：

- 按 selector 选中实例
- 必要时启动实例
- 等待 `debugReady=true`
- 返回可接管的 `cdpUrl`

### 2. 想把自动化复杂度交给 ant-chrome

优先调用：

```text
POST /api/automation/scripts/run
```

作用：

- 选择现成脚本
- 透传 `selector` / `params`
- 由 ant-chrome 本地执行 `playwright-cdp`
- 返回结果、日志、产物路径

### 3. 任务结束后回收实例

优先调用：

```text
POST /api/runtime/stop
```

## 当前状态

截至 `2026-04-09`，第一阶段公共入口已经落地：

- 已落地：`POST /api/runtime/session`
- 已落地：`POST /api/runtime/status`
- 已落地：`GET /api/runtime/active`
- 已落地：`POST /api/runtime/stop`
- 已落地：`GET /api/automation/scripts`
- 已落地：`GET /api/automation/scripts/{scriptId}`
- 已落地：`POST /api/automation/scripts/run`
- 已落地：`GET /api/automation/scripts/runs`

当前没有默认落地的内容：

- 未默认提供：`/api/automation/openclaw/*` 私有别名
- 未解决：多实例并发接管编排
- 未解决：跨机器 / 远程分布式接入
- 未解决：长期会话租约或 session token 模型

## 为什么优先走公共 API

因为这条路径最稳：

- OpenClaw 不需要自己拼 LaunchServer 细节
- OpenClaw 不需要自己处理 CDP 就绪等待
- OpenClaw 不需要自己运行 Playwright runtime
- 后面如果接别的 orchestrator，也能直接复用

如果未来外部已经写死 OpenClaw 私有协议，再补一层很薄的兼容别名即可。

## 文档索引

- `script-compat.md`
  说明 OpenClaw 怎么使用自动化脚本能力，以及为什么要让 `playwright-cdp` 吞掉复杂度
- `api-contract.md`
  说明当前已落地公共 API 的请求、响应、状态码和接入建议
- `delivery-status.md`
  说明当前哪些已经完成，哪些是刻意延期，下一步该做什么

## 兼容映射

如果外部之前是按 OpenClaw 私有路径思考，可以先按下面映射理解：

- `openclaw/session` -> `POST /api/runtime/session`
- `openclaw/run` -> `POST /api/automation/scripts/run`
- `openclaw/stop` -> `POST /api/runtime/stop`

这里只是语义映射，不代表当前后端已经实现这些私有路径。

## 一句话结论

先把 ant-chrome 做成稳定的通用浏览器执行端，再让 OpenClaw 调这些公共能力。
