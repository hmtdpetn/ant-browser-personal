# API 合约

这份文档描述的是当前已经落地的公共接口，不是未来假设接口。

## 通用约束

- 只支持本机调用：LaunchServer 默认只接受 `127.0.0.1`
- API 路径是 `/api/*`
- `/api/*` 可能启用 API Key 认证
- 默认认证 Header 是 `X-Ant-Api-Key`
- 请求体按 JSON 解析，未知字段会被拒绝
- OpenClaw 建议先通过 `GetLaunchServerInfo()` 读取：
  - `baseUrl`
  - `port`
  - `apiAuth.enabled`
  - `apiAuth.header`

## selector 规则

大多数运行时接口都支持这组 selector 字段：

```json
{
  "selector": {
    "code": "BUYER_001",
    "key": "buyer-001",
    "profileId": "profile-123",
    "profileName": "Amazon US",
    "keyword": "amazon-us",
    "keywords": ["amazon-us", "checkout"],
    "tag": "电商",
    "tags": ["电商", "北美"],
    "groupId": "group-sales",
    "matchMode": "unique"
  }
}
```

说明：

- `code` 优先表示 launch code
- 如果 `code` 不是已存在 launch code，部分路径会按关键字兜底
- `matchMode` 在运行时控制里只建议用：
  - `unique`
  - `first`
- `matchMode=all` 不适用于 `runtime/session`、`runtime/status`、`runtime/stop`

## 1. 准备可接管会话

### 请求

```text
POST /api/runtime/session
```

推荐请求体：

```json
{
  "selector": {
    "code": "BUYER_001"
  },
  "timeoutMs": 45000,
  "startUrls": ["https://example.com"],
  "skipDefaultStartUrls": true,
  "launchArgs": ["--window-size=1400,900"]
}
```

字段说明：

- `selector`
  目标实例选择条件
- `timeoutMs`
  等待 `debugReady=true` 的超时时间
- `startUrls`
  本次启动时额外打开的 URL
- `skipDefaultStartUrls`
  是否跳过实例默认启动 URL
- `launchArgs`
  本次启动时临时附加的启动参数

超时规则：

- 默认：`45000ms`
- 最小：`1000ms`
- 最大：`120000ms`

### 成功响应：已就绪

状态码：

```text
200 OK
```

示例：

```json
{
  "ok": true,
  "ready": true,
  "waitTimedOut": false,
  "retryable": false,
  "profileId": "profile-123",
  "profileName": "Amazon US",
  "launchCode": "BUYER_001",
  "running": true,
  "debugPort": 9333,
  "debugReady": true,
  "active": true,
  "cdpUrl": "http://127.0.0.1:19876",
  "directDebugUrl": "http://127.0.0.1:9333",
  "timeoutMs": 45000
}
```

语义：

- 可以立即接管
- 优先使用 `cdpUrl`
- `directDebugUrl` 更像诊断信息，不建议长期绑定

### 成功响应：启动了，但还没 ready

状态码：

```text
202 Accepted
```

示例：

```json
{
  "ok": true,
  "ready": false,
  "waitTimedOut": true,
  "retryable": true,
  "profileId": "profile-123",
  "launchCode": "BUYER_001",
  "running": true,
  "debugReady": false,
  "active": false,
  "runtimeWarning": "debug pending",
  "cdpUrl": "",
  "directDebugUrl": "",
  "timeoutMs": 45000
}
```

语义：

- 浏览器可能已经启动
- 但当前还不能保证可接管
- OpenClaw 应该稍后重试 `runtime/session`，或用 `runtime/status` 观察状态

### 常见失败

- `400`
  - `selector is required`
  - `invalid request body`
  - `matchMode must be unique or first for runtime control`
- `404`
  - `launch code not found`
  - `profile not found`
- `409`
  - selector 命中多个实例且未声明 `matchMode=first`
- `401`
  - API Key 错误或缺失

## 2. 查询运行时状态

### 查询当前活动实例

```text
GET /api/runtime/active
```

用途：

- 看当前统一 CDP 入口是否已有 active target
- 看哪个实例当前挂在 LaunchServer 的统一入口上

### 按 selector 查询某实例状态

```text
POST /api/runtime/status
```

请求体和 `runtime/session` 的 selector 部分一致。

用途：

- 不启动新实例
- 不等待 ready
- 只看当前状态

## 3. 停止实例

### 请求

```text
POST /api/runtime/stop
```

示例：

```json
{
  "selector": {
    "code": "BUYER_001"
  }
}
```

成功响应会带：

- `stopped=true`
- `running=false`
- `debugReady=false`

## 4. 列出自动化脚本

### 请求

```text
GET /api/automation/scripts
```

成功响应示例：

```json
{
  "ok": true,
  "count": 1,
  "items": [
    {
      "id": "news-query-txt",
      "name": "查询新闻并写 TXT",
      "type": "playwright-cdp",
      "status": "ready",
      "entryFile": "index.cjs",
      "tags": ["Playwright", "新闻"],
      "selector": {
        "code": "BUYER_001"
      },
      "params": {
        "keyword": "OpenAI",
        "limit": 10
      },
      "notes": "..."
    }
  ]
}
```

注意：

- 返回的是脚本元数据
- 不返回 `scriptText`

## 5. 查询单个脚本详情

### 请求

```text
GET /api/automation/scripts/{scriptId}
```

成功响应示例：

```json
{
  "ok": true,
  "item": {
    "id": "news-query-txt",
    "name": "查询新闻并写 TXT",
    "description": "测试脚本",
    "type": "playwright-cdp",
    "status": "ready",
    "entryFile": "index.cjs",
    "tags": ["Playwright", "新闻"],
    "selector": {
      "code": "BUYER_001"
    },
    "params": {
      "keyword": "OpenAI",
      "limit": 10
    },
    "notes": "...",
    "createdAt": "2026-04-08T10:00:00Z",
    "updatedAt": "2026-04-08T11:00:00Z",
    "packageFormat": "ant-automation-script",
    "manifestVersion": 1,
    "source": {
      "type": "git",
      "uri": "https://example.com/repo.git",
      "ref": "main",
      "path": "",
      "importedAt": "2026-04-08T10:00:00Z"
    }
  }
}
```

注意：

- 这是标准单资源查询接口
- 仍然不返回 `scriptText`
- 相比列表接口，会补充 `packageFormat`、`manifestVersion` 和 `source`

## 6. 执行自动化脚本

### 请求

```text
POST /api/automation/scripts/run
```

推荐请求体：

```json
{
  "scriptId": "news-query-txt",
  "selector": {
    "code": "BUYER_001"
  },
  "params": {
    "keyword": "OpenAI",
    "limit": 10
  }
}
```

字段规则：

- `scriptId` 必填
- `selector` 可选
- `params` 可选
- 不传 `selector`
  - 自动等价于 `useScriptSelector=true`
- 不传 `params`
  - 自动等价于 `useScriptParams=true`
- 如果显式写 `useScriptSelector=false`
  - 就必须传 `selector`
- 如果显式写 `useScriptParams=false`
  - 就必须传 `params`

### 成功响应

```json
{
  "ok": true,
  "run": {
    "id": "run-1",
    "scriptId": "news-query-txt",
    "scriptName": "查询新闻并写 TXT",
    "scriptType": "playwright-cdp",
    "status": "success",
    "summary": "已抓取 10 条新闻并写入 TXT",
    "error": "",
    "resultText": "{...}",
    "startedAt": "2026-04-08T10:00:00Z",
    "finishedAt": "2026-04-08T10:00:12Z",
    "durationMs": 12034
  }
}
```

### 失败场景

- `400`
  - `scriptId is required`
  - `selector must be a JSON object`
  - `params must be a JSON object`
- `500`
  - 脚本执行失败
  - automation runtime 不可用

## 7. 查询最近脚本运行记录

### 请求

```text
GET /api/automation/scripts/runs?limit=20
```

限制：

- 默认 `20`
- 最小 `1`
- 最大 `200`

## 推荐对接流程

### 方案 A：OpenClaw 接管浏览器

1. 调 `POST /api/runtime/session`
2. 如果返回 `200` 且 `ready=true`
   使用返回的 `cdpUrl` 接管
3. 如果返回 `202` 且 `ready=false`
   稍后重试 `runtime/session`
4. 结束后调 `POST /api/runtime/stop`

### 方案 B：OpenClaw 下发本地自动化任务

1. 调 `GET /api/automation/scripts`
2. 如需展示单个脚本详情，再调 `GET /api/automation/scripts/{scriptId}`
3. 让 OpenClaw 选择 `scriptId`
4. 调 `POST /api/automation/scripts/run`
5. 如需查看最近结果，再调 `GET /api/automation/scripts/runs`

## 兼容别名建议

如果未来一定要兼容 OpenClaw 私有路径，建议只做语义别名，不做新的执行链路：

- `POST /api/automation/openclaw/session`
  内部转发到 `POST /api/runtime/session`
- `POST /api/automation/openclaw/run`
  内部转发到 `POST /api/automation/scripts/run`
- `POST /api/automation/openclaw/stop`
  内部转发到 `POST /api/runtime/stop`
