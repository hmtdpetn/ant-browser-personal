# HTTP Patterns

Use these templates through the OpenClaw `exec` tool. On Windows, prefer `scripts/invoke_ant_chrome_api.ps1` for stable quoting and consistent JSON output. On Unix-like hosts, use the `curl` patterns below.

## Preferred Windows helper

```powershell
pwsh -File skills/ant-chrome-openclaw/scripts/invoke_ant_chrome_api.ps1 `
  -Method GET `
  -Path /api/health
```

Launch by exact code:

```powershell
pwsh -File skills/ant-chrome-openclaw/scripts/invoke_ant_chrome_api.ps1 `
  -Method GET `
  -Path /api/launch/YOUR_CODE
```

Launch by selector with inline JSON:

```powershell
$body = '{"selector":{"keywords":["buyer-001"],"matchMode":"unique"},"startUrls":["https://example.com"],"skipDefaultStartUrls":true}'
pwsh -File skills/ant-chrome-openclaw/scripts/invoke_ant_chrome_api.ps1 `
  -Method POST `
  -Path /api/launch `
  -JsonBody $body
```

## Shared shell variables

Unix shell:

```bash
BASE_URL="${ANT_CHROME_BASE_URL:-http://127.0.0.1:19876}"
API_HEADER="${ANT_CHROME_API_HEADER:-X-Ant-Api-Key}"

curl_ant() {
  if [ -n "${ANT_CHROME_API_KEY:-}" ]; then
    curl -sS -H "${API_HEADER}: ${ANT_CHROME_API_KEY}" "$@"
  else
    curl -sS "$@"
  fi
}
```

PowerShell:

```powershell
$baseUrl = if ($env:ANT_CHROME_BASE_URL) { $env:ANT_CHROME_BASE_URL } else { "http://127.0.0.1:19876" }
$apiHeader = if ($env:ANT_CHROME_API_HEADER) { $env:ANT_CHROME_API_HEADER } else { "X-Ant-Api-Key" }
```

## Health check

```bash
curl_ant "$BASE_URL/api/health"
```

PowerShell:

```powershell
$headers = @{}
if ($env:ANT_CHROME_API_KEY) { $headers[$apiHeader] = $env:ANT_CHROME_API_KEY }
Invoke-RestMethod -Method Get -Uri "$baseUrl/api/health" -Headers $headers
```

## Runtime active target

Use this before attaching OpenClaw Browser to confirm the current unified CDP target.

```bash
curl_ant "$BASE_URL/api/runtime/active"
```

Expected fields to capture:

- `ok`
- `active`
- `profileId`
- `profileName`
- `launchCode`
- `cdpUrl`

## Launch by exact code

```bash
curl_ant "$BASE_URL/api/launch/YOUR_CODE"
```

Expected fields to capture:

- `ok`
- `profileId`
- `profileName`
- `launchCode`
- `debugReady`
- `runtimeWarning`
- `cdpUrl`

## Launch by selector

Canonical request body:

```json
{
  "selector": {
    "profileId": "optional-stable-id",
    "profileName": "optional-name",
    "keywords": ["optional-keyword"],
    "tags": ["optional-tag"],
    "groupId": "optional-group",
    "matchMode": "unique"
  },
  "startUrls": ["https://example.com"],
  "skipDefaultStartUrls": true
}
```

`startUrls` must be a real target URL. Do not send an empty list or `about:blank` when you want to avoid blank startup tabs.

Unix shell:

```bash
curl_ant \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/launch" \
  --data-raw '{
    "selector": {
      "keywords": ["buyer-001"],
      "matchMode": "unique"
    },
    "startUrls": ["https://example.com"],
    "skipDefaultStartUrls": true
  }'
```

PowerShell:

```powershell
$headers = @{ "Content-Type" = "application/json" }
if ($env:ANT_CHROME_API_KEY) { $headers[$apiHeader] = $env:ANT_CHROME_API_KEY }
$body = @{
  selector = @{
    keywords = @("buyer-001")
    matchMode = "unique"
  }
  startUrls = @("https://example.com")
  skipDefaultStartUrls = $true
} | ConvertTo-Json -Depth 6
Invoke-RestMethod -Method Post -Uri "$baseUrl/api/launch" -Headers $headers -Body $body
```

## Profile list

```bash
curl_ant "$BASE_URL/api/profiles"
```

## Profile create

```bash
curl_ant \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/profiles" \
  --data-raw '{
    "profile": {
      "profileName": "buyer-001",
      "userDataDir": "buyers/buyer-001",
      "keywords": ["buyer-001"],
      "tags": ["sales"]
    },
    "launchCode": "BUYER_001"
  }'
```

## Profile status

```bash
curl_ant "$BASE_URL/api/profiles/YOUR_PROFILE_ID/status"
```

Expected fields to capture:

- `ok`
- `profileId`
- `running`
- `active`
- `debugReady`
- `cdpUrl`
- `directDebugUrl`

## Profile stop

```bash
curl_ant -X POST "$BASE_URL/api/profiles/YOUR_PROFILE_ID/stop"
```

Expected fields to capture:

- `ok`
- `stopped`
- `running`
- `active`
- `lastStopAt`

## Runtime status by launchCode or selector

Use this when you do not yet have a stable `profileId`.

```bash
curl_ant \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/runtime/status" \
  --data-raw '{
    "code": "BUYER_001"
  }'
```

Selector example with explicit tie-break:

```bash
curl_ant \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/runtime/status" \
  --data-raw '{
    "selector": {
      "keywords": ["buyer-001"],
      "matchMode": "first"
    }
  }'
```

Notes:

- Default runtime-control `matchMode` is `unique`.
- Runtime-control endpoints only support single-target resolution.
- `matchMode=all` is rejected for runtime control.

## Runtime stop by launchCode or selector

```bash
curl_ant \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/runtime/stop" \
  --data-raw '{
    "code": "BUYER_001"
  }'
```

## Launch logs

```bash
curl_ant "$BASE_URL/api/launch/logs?limit=20"
```

Use launch logs before retrying if the endpoint is reachable but the previous request failed.

## Automation script list

Use this when the user wants to see what Ant Browser scripts already exist before choosing one to run.

```bash
curl_ant "$BASE_URL/api/automation/scripts"
```

Expected fields to capture:

- `ok`
- `count`
- `items[].id`
- `items[].name`
- `items[].type`
- `items[].status`
- `items[].targetConfig`

## Automation script detail

Use this when the user already named a script and you need its saved selector or params behavior.

```bash
curl_ant "$BASE_URL/api/automation/scripts/YOUR_SCRIPT_ID"
```

Expected fields to capture:

- `ok`
- `item.id`
- `item.name`
- `item.type`
- `item.status`
- `item.selector`
- `item.params`
- `item.targetConfig`

## Automation script run

Minimal request when the script already stores its target and params:

```bash
curl_ant \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/automation/scripts/run" \
  --data-raw '{
    "scriptId": "news-query-txt"
  }'
```

Request with selector override:

```bash
curl_ant \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/automation/scripts/run" \
  --data-raw '{
    "scriptId": "news-query-txt",
    "selector": {
      "code": "BUYER_001"
    }
  }'
```

Request with params override while keeping the stored selector:

```bash
curl_ant \
  -H "Content-Type: application/json" \
  -X POST "$BASE_URL/api/automation/scripts/run" \
  --data-raw '{
    "scriptId": "news-query-txt",
    "params": {
      "keyword": "OpenAI",
      "limit": 10
    },
    "useScriptSelector": true
  }'
```

Expected fields to capture:

- `ok`
- `run.id`
- `run.scriptId`
- `run.scriptName`
- `run.status`
- `run.summary`
- `run.error`
- `run.resultText`
- `run.durationMs`

Notes:

- `selector` and `params` must be JSON objects when provided.
- If `selector` is omitted, the API uses the script default selector unless `useScriptSelector=false`.
- If `params` is omitted, the API uses the script default params unless `useScriptParams=false`.
- For scripts already bound to a target in Ant Browser, prefer the minimal body and only override what the user explicitly asked to change.

## Automation script runs

Use this when a previous script execution failed and you want to inspect recent outcomes before retrying.

```bash
curl_ant "$BASE_URL/api/automation/scripts/runs?limit=20"
```

Expected fields to capture:

- `ok`
- `count`
- `items[].scriptId`
- `items[].scriptName`
- `items[].status`
- `items[].summary`
- `items[].error`
- `items[].startedAt`
- `items[].durationMs`
