# OpenClaw 和自动化脚本怎么兼容

## 先说结论

现在已经有“自动化脚本执行能力”。

但这个能力目前是应用内入口，不是给 OpenClaw 直接调的 HTTP 接口。

所以正确做法不是：

- OpenClaw 自己去读取脚本文件
- OpenClaw 自己去拼 LaunchServer + CDP + Playwright 细节

正确做法是：

- OpenClaw 调 Ant Browser 的脚本执行接口
- Ant Browser 在本地执行脚本
- OpenClaw 只负责下达任务和接收结果

## 现在的真实执行链路

当前链路已经存在：

1. OpenClaw 或前端发起“执行脚本”
2. Ant Browser 读取 `scriptId`
3. Ant Browser 调 `AutomationScriptRunWithOptions`
4. 再进入 `RunScriptTask`
5. 本地 runner 执行脚本
6. runner 内部自己调用 LaunchServer
7. runner 自己 connect CDP
8. 返回结果、日志、产物路径

也就是说：

脚本系统本身已经知道怎么：

- 启动实例
- 连接浏览器
- 跑 Playwright
- 输出 artifact

这套能力不需要 OpenClaw 再重写一遍。

## 为什么不能让 OpenClaw 直接跑脚本

因为你现在的脚本运行模型，不只是“打开一个页面然后执行几行 JS”。

它还依赖这些运行时能力：

- 本地 automation runtime
- 本地 Playwright runner
- LaunchServer 地址和认证
- selector / params 注入
- artifact 输出目录
- 运行结果记录

这些都已经在 Ant Browser 后端里了。

如果让 OpenClaw 直接跑：

- 会重复实现一套 runtime
- 会重复处理 LaunchServer 细节
- 会把脚本和 Ant Browser 当前能力拆开
- 后面维护会很乱

## 最合理的兼容方式

第一阶段先做一层公共 HTTP 适配，不写 OpenClaw 私有语义。

这样做的好处是：

- Ant Browser 继续只提供通用执行入口
- OpenClaw 只是其中一个调用方
- 后续如果还有别的 agent / orchestrator，要接也能直接复用

如果未来一定要兼容某个 OpenClaw 既有协议，再单独加一层薄别名即可。

### 1. 列出脚本

```text
GET /api/automation/scripts
```

作用：

- 返回可执行脚本列表
- 返回 `scriptId`、名称、类型、默认 selector、默认 params

### 2. 执行脚本

```text
POST /api/automation/scripts/run
```

请求体建议：

```json
{
  "scriptId": "news-query-txt",
  "selector": {
    "code": "BUYER_001"
  },
  "params": {
    "keyword": "OpenAI",
    "limit": 10
  },
  "useScriptSelector": false,
  "useScriptParams": false
}
```

后端收到后，直接映射到：

- `AutomationScriptRunWithOptions`

如果要兼容现有数据结构，后端只要做一层转换：

- `selector` -> `selectorText`
- `params` -> `paramsText`

外部默认规则建议是：

- 不传 `selector` => `UseScriptSelector=true`
- 传了 `selector` => `UseScriptSelector=false`
- 不传 `params` => `UseScriptParams=true`
- 传了 `params` => `UseScriptParams=false`

## 更稳一点的做法

对于 `playwright-cdp` 类型脚本，建议后端把复杂度都收进去：

- 不要求 OpenClaw 先手工“启动实例再执行”
- 直接让脚本 runner 内部自己处理 `launch() + connect()`

也就是把 LaunchServer + CDP + Playwright 的细节都收进 Ant Browser，不要让 OpenClaw 自己补：

1. OpenClaw 只给 `scriptId + selector + params`
2. Ant Browser 在本地运行 `playwright-cdp` 脚本
3. runner 内部自己调用 LaunchServer 并重试接管 CDP

这样 OpenClaw 就只管一句话：

```text
执行脚本 news-query-txt，目标实例 BUYER_001，参数 keyword=OpenAI
```

## 推荐的接口形态

建议最终做成 3 个接口：

### 脚本列表

```text
GET /api/automation/scripts
```

### 执行脚本

```text
POST /api/automation/scripts/run
```

### 最近结果

```text
GET /api/automation/scripts/runs?limit=20
```

这样 OpenClaw 就能：

- 先看有哪些脚本
- 再按 `scriptId` 调用
- 再拿最近执行结果

## OpenClaw 这一侧怎么用

OpenClaw 不需要理解脚本细节。

只要做两件事：

1. 把用户的话整理成：
   - `scriptId`
   - `selector`
   - `params`
2. 调本地 HTTP 接口

比如：

```text
用户：用 BUYER_001 执行新闻抓取脚本，关键词 OpenAI，保存结果
```

OpenClaw 转成：

```json
{
  "scriptId": "news-query-txt",
  "selector": {
    "code": "BUYER_001"
  },
  "params": {
    "keyword": "OpenAI"
  }
}
```

然后 POST 给 Ant Browser。

## 最重要的一点

OpenClaw 和脚本系统的关系，建议这样定：

- OpenClaw = 编排层
- Ant Browser 脚本系统 = 执行层

不要反过来。

## 什么时候用 session，什么时候用 script

如果 OpenClaw 只是想“接管一个已经可调试的浏览器实例”，优先走：

```text
POST /api/runtime/session
```

它负责：

- 选中实例
- 必要时启动实例
- 等待 `debugReady=true`
- 返回可接管的 `cdpUrl`

如果 OpenClaw 想把 LaunchServer + Playwright + artifact 输出这些复杂度都交给 Ant Browser，本地直接执行现成脚本，优先走：

```text
POST /api/automation/scripts/run
```

它负责：

- 选脚本
- 透传 `selector` / `params`
- 在本地跑 `playwright-cdp` 脚本
- 返回运行结果、日志和产物路径

## 当前落地建议

如果按“低耦合、稳定优先”推进，建议先把 OpenClaw 接到这 3 个公共接口上：

- `GET /api/automation/scripts`
- `POST /api/automation/scripts/run`
- `GET /api/automation/scripts/runs`

OpenClaw 专用路径不是第一优先级。

只有在外部已经写死某套 OpenClaw 私有协议时，才考虑再补一层 `/api/automation/openclaw/*` 的兼容别名。

## 一句话方案

不是让 OpenClaw 兼容脚本文件。

而是给 OpenClaw 暴露“脚本执行 API”，让它调用现成脚本执行链路。
