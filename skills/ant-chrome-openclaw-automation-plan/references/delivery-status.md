# 交付状态

这份文档只记录当前实际交付情况，不记录理想状态。

更新时间：`2026-04-09`

## 已完成

### 公共运行时入口

- 已完成：`POST /api/runtime/session`
  - 负责按 selector 启动实例并等待 `debugReady=true`
  - ready 时返回 `200`
  - 超时未 ready 时返回 `202`
- 已完成：`POST /api/runtime/status`
  - 负责按 selector 查询实例状态
- 已完成：`GET /api/runtime/active`
  - 负责查看当前统一 CDP 入口对应的 active target
- 已完成：`POST /api/runtime/stop`
  - 负责停止实例并清理 active target

### 公共自动化脚本入口

- 已完成：`GET /api/automation/scripts`
  - 返回脚本元数据
- 已完成：`GET /api/automation/scripts/{scriptId}`
  - 返回单个脚本详情和来源元数据
- 已完成：`POST /api/automation/scripts/run`
  - 支持外部 `selector` / `params` 对象协议
  - 内部复用 `AutomationScriptRunWithOptions`
- 已完成：`GET /api/automation/scripts/runs`
  - 返回最近运行记录

### 关键实现决策

- 已完成：不默认引入 OpenClaw 私有后端语义
- 已完成：OpenClaw 通过公共 HTTP 入口接入
- 已完成：`playwright-cdp` 作为复杂度收口层
- 已完成：脚本外部协议用 JSON object，内部再转 `selectorText` / `paramsText`
- 已完成：等待 `debugReady` 的能力保留在 App 侧，由 LaunchServer 做编排

### 测试覆盖

- 已完成：LaunchServer 脚本 API focused tests
- 已完成：LaunchServer `runtime/session` focused tests
- 已完成：`go test ./backend/test/launchcode/...`
- 已完成：`go test ./backend/...`

## 刻意未做

这些不是漏做，而是当前阶段故意不做：

- 未做：`/api/automation/openclaw/*` 私有路径
- 未做：为 OpenClaw 单独复制一套运行时执行链路
- 未做：跨机器远程调用模型
- 未做：多实例并发接管调度
- 未做：长期 session token / lease 保活协议

原因：

- 会增加耦合
- 会复制已有执行链路
- 会在稳定性没完全收敛前把问题面扩大

## 仍然存在的边界

- 只建议本机 `127.0.0.1` 调用
- 只建议单实例串行接管
- `runtime/session` 返回 `202` 时，表示当前还不能保证可接管
- 不建议让外部长期依赖某次返回的统一 `cdpUrl` 做永久绑定

## 下一优先级

如果后续还要继续做，推荐顺序是：

1. 补一个很薄的 OpenClaw 私有别名层
2. 增加 OpenClaw 侧接入示例或伪代码
3. 视实际需要再考虑更严格的 session 管理模型

## 不建议的方向

当前不建议优先做：

- OpenClaw 自己拼 `LaunchServer + CDP + Playwright`
- OpenClaw 自己加载和执行 ant-chrome 的脚本文件
- 为 OpenClaw 重新写一套 automation runtime

## 一句话总结

这一阶段已经把“通用公共执行端”做出来了。

后面如果要补 OpenClaw 兼容层，应该是很薄的一层映射，不应该重新发明执行链路。
