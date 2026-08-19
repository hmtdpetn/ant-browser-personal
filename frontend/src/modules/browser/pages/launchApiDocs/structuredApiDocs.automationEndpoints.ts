import type { StructuredApiEndpointDoc } from './structuredApiDocs.types'

export const AUTOMATION_API_ENDPOINT_DOCS: StructuredApiEndpointDoc[] = [
  {
    id: 'api-automation-list-detail',
    parentId: 'api-automation',
    label: '脚本列表',
    method: 'GET',
    path: '/api/automation/scripts',
    purpose: '查询可执行脚本清单。',
    description: '返回脚本元数据，用于拿 scriptId、默认 selector / params 和脚本类型。',
    fields: [],
    requestExample: {
      language: 'bash',
      code: ({ launchBaseUrl, authHeader }) => `curl ${launchBaseUrl}/api/automation/scripts \\
  -H "${authHeader}: <your-api-key>"`,
    },
    responseExample: {
      language: 'json',
      code: () => `{
  "ok": true,
  "status": "success",
  "data": {
    "count": 1,
    "items": [
      {
        "id": "news-query-txt",
        "name": "查询新闻并写 TXT",
        "type": "playwright-cdp",
        "status": "ready",
        "entryFile": "index.cjs",
        "selector": { "code": "BUYER_001" },
        "params": { "keyword": "OpenAI", "limit": 10 }
      }
    ]
  }
}`,
    },
    responseCodes: [
      { code: '200', description: '返回脚本列表。' },
      { code: '503', description: '自动化脚本能力未启用。' },
    ],
    notes: [
      '不返回 scriptText。',
    ],
  },
  {
    id: 'api-automation-script-detail',
    parentId: 'api-automation',
    label: '脚本详情',
    method: 'GET',
    path: '/api/automation/scripts/{scriptId}',
    purpose: '按 scriptId 查询单个脚本详情。',
    description: '标准单资源读取接口，用于从脚本列表进入某个脚本时补充其来源和包格式等元数据。',
    fields: [
      { name: 'scriptId', type: 'string', required: true, location: 'Path', description: '脚本唯一 ID。' },
    ],
    requestExample: {
      language: 'bash',
      code: ({ launchBaseUrl, authHeader }) => `curl ${launchBaseUrl}/api/automation/scripts/news-query-txt \\
  -H "${authHeader}: <your-api-key>"`,
    },
    responseExample: {
      language: 'json',
      code: () => `{
  "ok": true,
  "status": "success",
  "data": {
    "item": {
      "id": "news-query-txt",
      "name": "查询新闻并写 TXT",
      "type": "playwright-cdp",
      "status": "ready",
      "entryFile": "index.cjs",
      "selector": { "code": "BUYER_001" },
      "params": { "keyword": "OpenAI", "limit": 10 },
      "packageFormat": "ant-automation-script",
      "manifestVersion": 1,
      "source": {
        "type": "git",
        "uri": "https://example.com/repo.git",
        "ref": "main"
      }
    }
  }
}`,
    },
    responseCodes: [
      { code: '200', description: '返回脚本详情。' },
      { code: '404', description: '脚本不存在。' },
      { code: '503', description: '自动化脚本能力未启用。' },
    ],
    notes: [
      '不返回 scriptText。',
    ],
  },
  {
    id: 'api-automation-run-detail',
    parentId: 'api-automation',
    label: '执行脚本',
    method: 'POST',
    path: '/api/automation/scripts/run',
    purpose: '按 scriptId 执行脚本。',
    description: '外部调用方只需传入脚本 ID 和对象形态的 selector / params。推荐优先传 selector.code；如果脚本已在 UI 中绑定目标实例，也可以只传 scriptId 直接执行。',
    fields: [
      { name: 'scriptId', type: 'string', required: true, location: 'Body', description: '要执行的脚本 ID。' },
      { name: 'selector', type: 'object', required: false, location: 'Body', description: '覆盖脚本默认 selector。' },
      { name: 'params', type: 'object', required: false, location: 'Body', description: '覆盖脚本默认 params。' },
      { name: 'useScriptSelector', type: 'boolean', required: false, location: 'Body', description: '显式指定是否沿用脚本默认 selector。' },
      { name: 'useScriptParams', type: 'boolean', required: false, location: 'Body', description: '显式指定是否沿用脚本默认 params。' },
      { name: 'timeoutMs', type: 'integer', required: false, location: 'Body', description: '本次脚本执行超时时间，范围 1000 到 1800000。' },
    ],
    requestExample: {
      language: 'bash',
      code: ({ launchBaseUrl, authHeader }) => `curl -X POST ${launchBaseUrl}/api/automation/scripts/run \\
  -H "Content-Type: application/json" \\
  -H "${authHeader}: <your-api-key>" \\
  -d '{
    "scriptId": "news-query-txt",
    "selector": { "code": "BUYER_001" },
    "params": { "keyword": "OpenAI", "limit": 10 }
  }'`,
    },
    responseExample: {
      language: 'json',
      code: () => `{
  "ok": true,
  "status": "success",
  "data": {
    "run": {
      "id": "run-1",
      "scriptId": "news-query-txt",
      "scriptName": "查询新闻并写 TXT",
      "scriptType": "playwright-cdp",
      "status": "success",
      "summary": "已抓取 10 条新闻并写入 TXT",
      "durationMs": 12034
    },
    "summary": "已抓取 10 条新闻并写入 TXT"
  }
}`,
    },
    responseCodes: [
      { code: '200', description: '执行成功。' },
      { code: '400', description: 'scriptId 缺失、selector / params 不是对象，或 timeoutMs 超出范围。' },
      { code: '500', description: '脚本执行失败。' },
    ],
    notes: [
      'selector / params 必须是 object。',
      '不传时沿用脚本默认配置。',
    ],
  },
  {
    id: 'api-automation-runs-detail',
    parentId: 'api-automation',
    label: '运行记录',
    method: 'GET',
    path: '/api/automation/scripts/runs?limit=20',
    purpose: '查询最近脚本运行记录。',
    description: '返回最近 N 次脚本执行记录，适合调试、审计和任务结果回看。',
    fields: [
      { name: 'limit', type: 'integer', required: false, location: 'Query', description: '返回记录条数，默认 20，最小 1，最大 200。' },
    ],
    requestExample: {
      language: 'bash',
      code: ({ launchBaseUrl, authHeader }) => `curl "${launchBaseUrl}/api/automation/scripts/runs?limit=20" \\
  -H "${authHeader}: <your-api-key>"`,
    },
    responseExample: {
      language: 'json',
      code: () => `{
  "ok": true,
  "status": "success",
  "data": {
    "count": 1,
    "limit": 20,
    "items": [
      {
        "id": "run-1",
        "scriptId": "news-query-txt",
        "status": "success",
        "summary": "已抓取 10 条新闻并写入 TXT",
        "durationMs": 12034
      }
    ]
  }
}`,
    },
    responseCodes: [
      { code: '200', description: '返回运行记录。' },
      { code: '503', description: '自动化脚本能力未启用。' },
    ],
    notes: [],
  },
  {
    id: 'api-automation-hook-detail',
    parentId: 'api-automation',
    label: '公开脚本 Hook',
    method: 'POST',
    path: '/api/automation/hooks/{hookPath}',
    purpose: '通过脚本自定义的公开路径执行自动化任务。',
    description: '脚本作者可以为单个脚本启用公开 Hook，并指定路径、HTTP 方法、默认参数、变量和超时。调用方无需知道 scriptId，只需按脚本配置的 Hook 契约请求。',
    fields: [
      { name: 'hookPath', type: 'string', required: true, location: 'Path', description: '脚本公开 API 配置的路径。' },
      { name: 'code', type: 'string', required: false, location: 'Body', description: '兼容写法：指定已有实例 launchCode；不能与 instance 同传。' },
      { name: 'instance', type: 'object', required: false, location: 'Body', description: '实例策略，type 支持 script-default、existing、rotate、create。' },
      { name: 'params', type: 'object', required: false, location: 'Body', description: '覆盖脚本默认参数；脚本可配置必填变量。' },
      { name: 'timeoutMs', type: 'integer', required: false, location: 'Body', description: '本次执行超时；也可作为 Query 参数传入。' },
    ],
    requestExample: {
      language: 'bash',
      code: ({ launchBaseUrl, authHeader }) => `curl -X POST ${launchBaseUrl}/api/automation/hooks/news/query \\
  -H "Content-Type: application/json" \\
  -H "${authHeader}: <your-api-key>" \\
  -d '{
    "instance": {
      "type": "existing",
      "selector": { "code": "BUYER_001" }
    },
    "params": { "keyword": "OpenAI", "limit": 10 },
    "timeoutMs": 120000
  }'`,
    },
    responseExample: {
      language: 'json',
      code: () => `{
  "ok": true,
  "status": "success",
  "summary": "已抓取 10 条新闻",
  "message": "已抓取 10 条新闻",
  "data": { "count": 10 },
  "result": { "count": 10 }
}`,
    },
    responseCodes: [
      { code: '200', description: '脚本执行完成；业务失败也可能以 ok=false 的 200 返回。' },
      { code: '400', description: '请求体、实例策略、变量或 timeoutMs 非法。' },
      { code: '404', description: 'Hook 不存在或脚本没有启用公开 API。' },
      { code: '405', description: '请求方法与脚本配置的方法不一致。' },
      { code: '500', description: '脚本执行异常。' },
      { code: '503', description: '自动化脚本能力未启用。' },
    ],
    notes: [
      '真实 HTTP 方法由脚本 Public API 配置决定，示例使用 POST。',
      'Hook 仍受 LaunchServer 的 localhost 限制和 API Key 认证保护。',
      'code 与 instance 不能同时传；不传二者时沿用脚本默认实例策略。',
    ],
  },
]
