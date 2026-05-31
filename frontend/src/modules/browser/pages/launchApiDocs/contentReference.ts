export const DOC_API_SUPPORT = `# 排障与日志

## 先查这 4 个接口

| 接口 | 用途 |
|------|------|
| \`GET /api/health\` | 先确认服务是否在线 |
| \`GET /api/launch/logs?limit=20\` | 查看最近接口调用和错误 |
| \`GET /api/runtime/active\` | 看统一 CDP 当前指向哪个实例 |
| \`GET /api/profiles/{profileId}/status\` | 精确确认某个实例是否在运行、是否 ready |

## 日志接口

| 方法 | 路径 | 用途 |
|------|------|------|
| \`GET\` | \`/api/launch/logs?limit=50\` | 查看最近接口调用和错误 |

## 日志里主要看

\`\`\`text
method
path
code
selector
profileId
status
error
durationMs
\`\`\`

## 错误码速查

| 状态码 | 一般代表什么 | 先做什么 |
|--------|--------------|----------|
| \`400\` | 请求体不对 / selector 缺失 | 先对照示例 JSON |
| \`401\` | API Key 错误 | 检查 \`X-Ant-Api-Key\` |
| \`403\` | 不是 localhost | 改成本机请求 |
| \`404\` | Code 或 selector 没命中 | 检查 \`code / keywords / tags / groupId\` |
| \`405\` | 方法不对 | 检查 GET / POST |
| \`409\` | 多命中 / launchCode 冲突 | 收窄 selector 或换 launchCode |
| \`500\` | 启动或脚本执行失败 | 看 \`/api/launch/logs\` 和应用日志 |
| \`503\` | 当前没有活动实例或能力没就绪 | 先查 \`/api/runtime/active\` |

## 看到这些错误时

| 错误文本 | 直接看哪里 |
|----------|------------|
| \`selector is required\` | 请求体里有没有 \`selector / code / profileId\` |
| \`launch code not found\` | Code 是否真实存在 |
| \`selector matched multiple profiles\` | 是否要补 \`tags / groupId\` 或改 \`matchMode=first\` |
| \`no active browser debug target\` | 先查 \`/api/runtime/active\` |
| \`automation script api is unavailable\` | 当前环境是否已启用脚本能力 |

## 排查顺序

\`\`\`text
1. health
2. 目标接口响应
3. launch/logs
4. runtime/active 或 profile status
5. 应用内日志 / 内核 / 代理
\`\`\`
`
