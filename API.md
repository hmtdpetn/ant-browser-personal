# Ant Browser Personal Launch HTTP API

适用版本：2.2.0 及后续兼容版本。

本文档描述供本机脚本、自动化工具和内部服务调用的 Launch HTTP API。桌面界面使用的 Wails 绑定不是面向第三方的网络 API；第三方请只调用本文档中的 HTTP 路由，不要直接读写 SQLite 数据库。

## 连接与鉴权

服务只监听 `127.0.0.1`，默认端口为 `19876`。基础地址示例：

```text
http://127.0.0.1:19876
```

建议在 `config.yaml` 中启用 API Key：

```yaml
launch_server:
  port: 19876
  auth:
    enabled: true
    api_key: "replace-with-a-long-random-secret"
    header: "X-Ant-Api-Key"
```

启用后，每个 `/api/*` 请求都必须附带对应请求头：

```bash
curl -H "X-Ant-Api-Key: replace-with-a-long-random-secret" http://127.0.0.1:19876/api/health
```

默认请求头名为 `X-Ant-Api-Key`。若 `auth.enabled` 为 `false` 或 `api_key` 为空，现有兼容行为是不校验 Key；由于服务仅限本机访问，仍建议为自动化场景明确启用鉴权。

成功响应通常包含 `"ok": true`；失败响应至少包含 `"ok": false` 和 `error`。自动化接口还会返回 `code`、`field` 等便于程序处理的字段。

## 健康检查

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/health` | 返回服务健康状态。 |

## 实例 API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/profiles` | 列出实例。 |
| `POST` | `/api/profiles` | 创建实例，可选择创建后启动。 |
| `GET` | `/api/profiles/{profileId}` | 读取单个实例。 |
| `PUT` | `/api/profiles/{profileId}` | 更新实例。 |
| `DELETE` | `/api/profiles/{profileId}` | 删除未运行的实例。 |
| `GET` | `/api/profiles/{profileId}/status` | 查询运行状态。 |
| `POST` | `/api/profiles/{profileId}/stop` | 停止实例。 |

创建与更新的请求体使用 `profile` 对象；可选 `launchCode`、`autoLaunch` 和 `start`：

```json
{
  "profile": {
    "profileName": "Shop A",
    "userDataDir": "data/profiles/shop-a",
    "coreId": "",
    "fingerprintArgs": [],
    "proxyId": "",
    "proxyConfig": "",
    "launchArgs": [],
    "tags": ["shop"],
    "keywords": ["a"],
    "groupId": "instance-group-id"
  },
  "launchCode": "SHOP-A",
  "autoLaunch": false
}
```

`groupId` 可把实例放入已存在的实例分组；分组本身请通过下文的分组 API 创建和维护。

## 启动、运行状态与 CDP

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/launch/{code}` | 按启动码启动一个实例。 |
| `POST` | `/api/launch` | 按选择器启动一个或多个实例。 |
| `GET` | `/api/launch/logs?limit=50` | 读取最近启动/运行控制记录，`limit` 为 1–200。 |
| `GET` | `/api/runtime/active` | 读取当前活跃实例及统一 CDP 信息。 |
| `POST` | `/api/runtime/status` | 按选择器读取一个实例状态。 |
| `POST` | `/api/runtime/stop` | 按选择器停止一个实例。 |
| `POST` | `/api/runtime/session` | 启动目标并等待其调试端口成为可接管的 runtime session。 |

`POST /api/launch`、`POST /api/runtime/status` 和 `POST /api/runtime/stop` 都接受选择器。常用字段：

```json
{
  "selector": {
    "profileId": "profile-id",
    "code": "OPTIONAL-CODE",
    "profileName": "Shop A",
    "tags": ["shop"],
    "keywords": ["a"],
    "groupId": "instance-group-id",
    "matchMode": "first"
  },
  "launchArgs": ["--lang=en-US"],
  "startUrls": ["https://example.com"],
  "skipDefaultStartUrls": false,
  "proxyId": "",
  "proxyConfig": ""
}
```

启动时 `matchMode` 可使用 `first`、`unique` 或 `all`；运行状态与停止只允许唯一目标（`first` 或 `unique`）。`/` 根路径不是业务 REST 路由，而是当前活跃实例的本地 CDP 反向代理入口。

## 实例树分组 API（2.0.1 新增）

实例分组支持根分组、子分组和排序。`parentId` 为空字符串表示根分组。删除一个分组时，其子分组和直属实例会提升到该分组的父分组；删除根分组时会变为未分组。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/groups` | 返回树形实例分组的扁平列表及每组直属实例数。 |
| `POST` | `/api/groups` | 创建实例分组。 |
| `PUT` | `/api/groups/{groupId}` | 重命名、改父级或调整排序。 |
| `DELETE` | `/api/groups/{groupId}` | 删除分组并安全提升成员/子分组。 |
| `POST` | `/api/groups/move-profiles` | 批量移动实例；空 `groupId` 表示取消分组。 |

创建或更新请求体：

```json
{
  "groupName": "北美店铺",
  "parentId": "",
  "sortOrder": 10
}
```

批量移动请求体：

```json
{
  "profileIds": ["profile-id-1", "profile-id-2"],
  "groupId": "target-instance-group-id"
}
```

## 代理/订阅树分组 API（2.0.1 新增）

这里的“订阅分组”指订阅导入后保存到代理池的代理树分组；分组保存的是代理成员，不会把订阅 URL 或 UA 暴露成独立的 HTTP 接口。`parentId` 为空字符串表示根分组。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/proxy-groups` | 返回代理树分组及每组直属代理数。 |
| `POST` | `/api/proxy-groups` | 创建代理/订阅分组。 |
| `PUT` | `/api/proxy-groups/{groupId}` | 重命名、改父级或调整排序。 |
| `DELETE` | `/api/proxy-groups/{groupId}` | 删除分组并将子分组、直属代理提升到父分组。 |
| `POST` | `/api/proxy-groups/move-proxies` | 批量移动代理；空 `groupId` 表示取消分组。 |

创建和更新的请求体与实例分组相同。批量移动代理示例：

```json
{
  "proxyIds": ["proxy-id-1", "proxy-id-2"],
  "groupId": "target-proxy-group-id"
}
```

所有分组写操作都通过应用现有 DAO 与事务完成；无效父分组、循环引用、缺失成员或不存在的目标组会返回错误，且批量移动不会部分提交。

## 自动化 API

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/automation/scripts` | 列出自动化脚本摘要。 |
| `GET` | `/api/automation/scripts/{scriptId}` | 读取单个脚本详情。 |
| `POST` | `/api/automation/scripts/run` | 执行脚本，可覆盖脚本选择器、参数和超时。 |
| `GET` | `/api/automation/scripts/runs?limit=20` | 读取执行记录，`limit` 为 1–200。 |
| 脚本配置的方法 | `/api/automation/hooks/{path}` | 调用已启用的公开自动化 Hook。 |

执行脚本的最小请求：

```json
{
  "scriptId": "script-id",
  "useScriptSelector": true,
  "useScriptParams": true,
  "timeoutMs": 60000
}
```

可在请求中额外传入 `selector`、`targetInput` 或 `params` 对象。脚本包的本地文件/ZIP 导入导出仍属于桌面端操作，因为它们需要用户选择本地文件；当前没有上传脚本包的 HTTP 接口。

## 代理网关与热切换 API（2.2.0 新增）

2.2.0 为每个运行中的实例启动独立的本地 SOCKS5 网关。浏览器连接的是网关地址，网关再根据当前路由选择远端代理、直连或阻断。切换代理只影响新建连接；旧连接默认自然排空，也可以使用 `force: true` 立即断开。代理拨号失败时网关不会回退到直连。

### 按实例 ID 读取网关状态

```http
GET /api/proxy-gateway/status?profileId={profileId}
```

```bash
curl "http://127.0.0.1:19876/api/proxy-gateway/status?profileId=550e8400-e29b-41d4-a716-446655440000" \
  -H "X-Ant-Api-Key: <your-api-key>"
```

成功响应：

```json
{
  "ok": true,
  "profileId": "550e8400-e29b-41d4-a716-446655440000",
  "gateway": {
    "proxyUrl": "socks5://127.0.0.1:43127",
    "currentRouteId": "proxy-us-new",
    "mode": "proxy",
    "activeConnections": 2,
    "drainingConnections": 0,
    "browserPid": 12340,
    "browserDebugPort": 9333
  }
}
```

### 运行中热切换代理

```http
POST /api/proxy-gateway/switch
Content-Type: application/json
```

请求体：

```json
{
  "selector": { "profileId": "550e8400-e29b-41d4-a716-446655440000" },
  "proxyId": "proxy-us-new",
  "force": false
}
```

`selector` 支持 `profileId`、`code`、`profileName`、标签、关键字和 `groupId`；运行态操作必须唯一命中一个实例。`proxyId` 无效时可以改用 `proxyConfig`，例如 `socks5://127.0.0.1:1080`。`force: true` 会立即关闭切换前建立的旧连接。

成功响应会包含 `appliedLive`、`profileId` 和 `gateway`；实例未运行时 `appliedLive` 为 `false`，只保存下次启动使用的代理。

常见错误码：

| 状态码 | 含义 |
| --- | --- |
| `400` | selector、代理配置或请求体非法。 |
| `404` | selector 未命中实例，或代理节点不存在且没有有效 proxyConfig。 |
| `409` | selector 命中多个实例。 |
| `502` | 代理出口或网关应用失败；原路由保持不变。 |
| `503` | 代理网关控制能力未就绪。 |

### 读取和保存分流规则

读取当前配置：

```http
GET /api/proxy-gateway/routing?profileId={profileId}
```

保存并立即应用：

```http
PUT /api/proxy-gateway/routing
Content-Type: application/json
```

```json
{
  "selector": { "profileId": "550e8400-e29b-41d4-a716-446655440000" },
  "routing": {
    "mode": "rule",
    "rules": [
      {
        "id": "cn",
        "name": "中国大陆直连",
        "enabled": true,
        "matchType": "domain_suffix",
        "pattern": ".cn",
        "action": "direct"
      },
      {
        "id": "tracker",
        "name": "阻断追踪",
        "enabled": true,
        "matchType": "domain_keyword",
        "pattern": "tracker",
        "action": "block"
      }
    ]
  },
  "force": false
}
```

`mode` 可选 `proxy`（全部代理）、`rule`（按规则）或 `direct`（全部直连）。规则的 `action` 可选 `proxy`、`direct`、`block`；`matchType` 可选 `domain`、`domain_suffix`、`domain_keyword`、`ip_cidr`。规则按数组顺序从上到下匹配，第一条命中后停止；`mode=rule` 未命中时默认走代理。域名后缀的前导点可省略，`.cn` 与 `cn` 行为相同。

常见错误码：

| 状态码 | 含义 |
| --- | --- |
| `400` | 缺少 selector，或 mode、matchType、action、pattern、IP/CIDR 无效。 |
| `404` | selector 未命中实例。 |
| `502` | 运行中的网关应用新规则失败，持久化配置保持不变。 |
| `503` | 代理网关控制能力未就绪。 |

完整的结构化接口目录、字段表和可复制的 curl 示例，也可以在桌面端的“Launch API 文档”页面查看。

## 使用建议

- 外部调用请先使用 `GET /api/health` 确认服务，再查询或操作资源。
- 批量移动前先读取对应分组列表，以获取稳定的 `groupId`，不要按分组名称猜测 ID。
- 对生产自动化启用 API Key，不要将端口转发到公网或局域网。
- 代理订阅的 User-Agent、导入和刷新策略目前由桌面端控制；不要绕过应用直接改数据库。
- 需要新的自动化能力时，优先扩展本 API；不要依赖 Wails 前端绑定或 SQLite 内部表结构。
