import type { BrowserProfile } from "./types";

export type AutomationScriptType = "playwright-cdp" | "launch-api";

export type AutomationScriptStatus = "draft" | "ready" | "disabled";

export type AutomationScriptTargetMode =
  | "manual"
  | "existing"
  | "create"
  | "rotate";

export interface AutomationScriptSource {
  type: string;
  uri: string;
  ref: string;
  path: string;
  importedAt: string;
}

export interface AutomationScriptTargetSelector {
  code: string;
  profileId: string;
  profileName: string;
  groupId: string;
  keywords: string[];
  tags: string[];
}

export interface AutomationScriptTargetConfig {
  mode: AutomationScriptTargetMode;
  selector: AutomationScriptTargetSelector;
  templateSelector: AutomationScriptTargetSelector;
  createNameTemplate: string;
}

export interface AutomationScriptRecord {
  packageFormat: string;
  manifestVersion: number;
  id: string;
  name: string;
  description: string;
  type: AutomationScriptType;
  status: AutomationScriptStatus;
  entryFile: string;
  tags: string[];
  selectorText: string;
  paramsText: string;
  scriptText: string;
  notes: string;
  targetConfig: AutomationScriptTargetConfig;
  source: AutomationScriptSource;
  createdAt: string;
  updatedAt: string;
}

export interface AutomationScriptRunRecord {
  id: string;
  scriptId: string;
  scriptName: string;
  scriptType: string;
  status: "success" | "failed" | "running";
  summary: string;
  error: string;
  resultText: string;
  startedAt: string;
  finishedAt: string;
  durationMs: number;
}

export interface AutomationScriptRunInput {
  scriptId: string;
  selectorText?: string;
  paramsText?: string;
  useScriptSelector?: boolean;
  useScriptParams?: boolean;
  timeoutMs?: number;
  launchCode?: string;
  startByCodeBeforeRun?: boolean;
}

const AUTOMATION_SCRIPTS_STORAGE_KEY = "automation_scripts_v1";
export const AUTOMATION_SCRIPT_PACKAGE_FORMAT = "ant-automation-script";
export const AUTOMATION_SCRIPT_MANIFEST_VERSION = 1;

export const AUTOMATION_SCRIPT_TYPE_OPTIONS: Array<{
  value: AutomationScriptType;
  label: string;
}> = [
  { value: "playwright-cdp", label: "Playwright CDP" },
  { value: "launch-api", label: "Launch API" },
];

export const AUTOMATION_SCRIPT_STATUS_OPTIONS: Array<{
  value: AutomationScriptStatus;
  label: string;
}> = [
  { value: "draft", label: "草稿" },
  { value: "ready", label: "可用" },
  { value: "disabled", label: "停用" },
];

export const AUTOMATION_SCRIPT_TARGET_MODE_OPTIONS: Array<{
  value: AutomationScriptTargetMode;
  label: string;
}> = [
  { value: "manual", label: "手动 selector" },
  { value: "existing", label: "使用已有实例" },
  { value: "create", label: "按模板新建实例" },
  { value: "rotate", label: "按条件轮询实例" },
];

export const DUAL_INSTANCE_RUNTIME_SCRIPT_ID = "dual-instance-runtime-switch";
const DUAL_INSTANCE_DEFAULT_CODES = ["BUYER_001", "BUYER_002"] as const;
const DUAL_INSTANCE_DEFAULT_START_URLS = [
  "https://finance.sina.com.cn/",
  "https://map.baidu.com/",
] as const;

function nowIso(): string {
  return new Date().toISOString();
}

function createScriptId(): string {
  if (
    typeof crypto !== "undefined" &&
    typeof crypto.randomUUID === "function"
  ) {
    return crypto.randomUUID();
  }
  return `script-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

function normalizeSource(source: unknown): AutomationScriptSource {
  if (!source || typeof source !== "object") {
    return {
      type: "",
      uri: "",
      ref: "",
      path: "",
      importedAt: "",
    };
  }

  const raw = source as Partial<AutomationScriptSource>;
  return {
    type: typeof raw.type === "string" ? raw.type.trim() : "",
    uri: typeof raw.uri === "string" ? raw.uri.trim() : "",
    ref: typeof raw.ref === "string" ? raw.ref.trim() : "",
    path: typeof raw.path === "string" ? raw.path.trim() : "",
    importedAt:
      typeof raw.importedAt === "string" ? raw.importedAt.trim() : "",
  };
}

function normalizeTargetTerms(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  const deduped = new Set<string>();
  for (const item of value) {
    const normalized = String(item || "").trim();
    if (normalized) {
      deduped.add(normalized);
    }
  }
  return Array.from(deduped);
}

function normalizeTargetSelector(
  selector: unknown,
): AutomationScriptTargetSelector {
  if (!selector || typeof selector !== "object") {
    return {
      code: "",
      profileId: "",
      profileName: "",
      groupId: "",
      keywords: [],
      tags: [],
    };
  }

  const raw = selector as Partial<AutomationScriptTargetSelector>;
  return {
    code:
      typeof raw.code === "string"
        ? raw.code.trim().toUpperCase()
        : typeof (selector as { launchCode?: unknown }).launchCode === "string"
          ? String((selector as { launchCode?: unknown }).launchCode)
              .trim()
              .toUpperCase()
          : "",
    profileId:
      typeof raw.profileId === "string" ? raw.profileId.trim() : "",
    profileName:
      typeof raw.profileName === "string" ? raw.profileName.trim() : "",
    groupId: typeof raw.groupId === "string" ? raw.groupId.trim() : "",
    keywords: normalizeTargetTerms(raw.keywords),
    tags: normalizeTargetTerms(raw.tags),
  };
}

export function createAutomationScriptTargetSelector(): AutomationScriptTargetSelector {
  return {
    code: "",
    profileId: "",
    profileName: "",
    groupId: "",
    keywords: [],
    tags: [],
  };
}

export function normalizeAutomationScriptTargetConfig(
  config: unknown,
): AutomationScriptTargetConfig {
  if (!config || typeof config !== "object") {
    return {
      mode: "manual",
      selector: createAutomationScriptTargetSelector(),
      templateSelector: createAutomationScriptTargetSelector(),
      createNameTemplate: "",
    };
  }

  const raw = config as Partial<AutomationScriptTargetConfig>;
  const mode: AutomationScriptTargetMode =
    raw.mode === "existing" ||
    raw.mode === "create" ||
    raw.mode === "rotate"
      ? raw.mode
      : "manual";

  return {
    mode,
    selector: normalizeTargetSelector(raw.selector),
    templateSelector: normalizeTargetSelector(raw.templateSelector),
    createNameTemplate:
      typeof raw.createNameTemplate === "string"
        ? raw.createNameTemplate.trim()
        : "",
  };
}

function selectorSummaryParts(selector: AutomationScriptTargetSelector): string[] {
  const parts: string[] = [];
  if (selector.code) {
    parts.push(`Code=${selector.code}`);
  }
  if (selector.profileName) {
    parts.push(`实例=${selector.profileName}`);
  }
  if (selector.profileId && !selector.code) {
    parts.push(`实例ID=${selector.profileId}`);
  }
  if (selector.groupId) {
    parts.push(`分组=${selector.groupId}`);
  }
  if (selector.tags.length > 0) {
    parts.push(`标签=${selector.tags.join(" / ")}`);
  }
  if (selector.keywords.length > 0) {
    parts.push(`关键字=${selector.keywords.join(" / ")}`);
  }
  return parts;
}

export function getAutomationScriptTargetModeLabel(
  mode: AutomationScriptTargetMode,
): string {
  return (
    AUTOMATION_SCRIPT_TARGET_MODE_OPTIONS.find((item) => item.value === mode)
      ?.label || mode
  );
}

function normalizeSelectorCode(value?: string): string {
  return String(value || "")
    .trim()
    .toUpperCase();
}

function normalizeSelectorText(value?: string): string {
  return String(value || "").trim();
}

export function findAutomationTargetProfile(
  selector: AutomationScriptTargetSelector,
  profiles: BrowserProfile[],
): BrowserProfile | null {
  const normalizedProfileId = normalizeSelectorText(selector.profileId);
  if (normalizedProfileId) {
    const matchedById = profiles.find(
      (profile) => normalizeSelectorText(profile.profileId) === normalizedProfileId,
    );
    if (matchedById) {
      return matchedById;
    }
  }

  const normalizedCode = normalizeSelectorCode(selector.code);
  if (normalizedCode) {
    const matchedByCode = profiles.find(
      (profile) => normalizeSelectorCode(profile.launchCode) === normalizedCode,
    );
    if (matchedByCode) {
      return matchedByCode;
    }
  }

  const normalizedProfileName = normalizeSelectorText(selector.profileName);
  if (normalizedProfileName) {
    const matchedByName = profiles.find(
      (profile) =>
        normalizeSelectorText(profile.profileName).toLowerCase() ===
        normalizedProfileName.toLowerCase(),
    );
    if (matchedByName) {
      return matchedByName;
    }
  }

  return null;
}

export function formatAutomationTargetIdentity(
  selector: AutomationScriptTargetSelector,
  profiles: BrowserProfile[],
  options?: {
    includeProfileId?: boolean;
    fallback?: string;
  },
): string {
  const profile = findAutomationTargetProfile(selector, profiles);
  const code = normalizeSelectorCode(profile?.launchCode || selector.code);
  const profileName = normalizeSelectorText(
    profile?.profileName || selector.profileName,
  );
  const profileId = normalizeSelectorText(profile?.profileId || selector.profileId);

  const parts = [code, profileName].filter(Boolean);
  if (options?.includeProfileId && profileId) {
    parts.push(profileId);
  }

  if (parts.length > 0) {
    return parts.join(" · ");
  }
  if (profileId) {
    return options?.includeProfileId ? profileId : `实例 ID ${profileId}`;
  }

  return options?.fallback || "-";
}

export function describeAutomationScriptTargetConfig(
  config: AutomationScriptTargetConfig,
): string {
  switch (config.mode) {
    case "existing": {
      const parts = selectorSummaryParts(config.selector);
      return parts.length > 0
        ? `使用已有实例：${parts.join(" · ")}`
        : "使用已有实例";
    }
    case "create": {
      const parts = selectorSummaryParts(config.templateSelector);
      const namePart = config.createNameTemplate
        ? `命名=${config.createNameTemplate}`
        : "";
      return [
        "按模板新建实例",
        ...parts,
        namePart,
      ]
        .filter(Boolean)
        .join(" · ");
    }
    case "rotate": {
      const parts = selectorSummaryParts(config.selector);
      return parts.length > 0
        ? `按条件轮询实例：${parts.join(" · ")}`
        : "按条件轮询实例";
    }
    default:
      return "手动填写 selector JSON";
  }
}

export function getAutomationScriptSourceLabel(source: AutomationScriptSource): string {
  switch (source.type) {
    case "builtin":
      return "内置基线";
    case "local-file":
      return "本地文件";
    case "local-dir":
      return "本地目录";
    case "remote-url":
      return "远程 URL";
    case "git":
      return "Git";
    case "text":
      return "文本导入";
    case "manual":
      return "手动维护";
    default:
      return source.type || "未标记";
  }
}

export function canRefreshAutomationScriptSource(
  source: AutomationScriptSource,
): boolean {
  return (
    source.type === "local-file" ||
    source.type === "local-dir" ||
    source.type === "remote-url" ||
    source.type === "git"
  );
}

export function getAutomationScriptRefreshLabel(
  source: AutomationScriptSource,
): string {
  return source.type === "git" ? "重新拉取" : "重新导入";
}

export function getAutomationScriptTypeLabel(
  type: AutomationScriptType,
): string {
  return (
    AUTOMATION_SCRIPT_TYPE_OPTIONS.find((item) => item.value === type)?.label ||
    type
  );
}

export function getAutomationScriptStatusLabel(
  status: AutomationScriptStatus,
): string {
  return (
    AUTOMATION_SCRIPT_STATUS_OPTIONS.find((item) => item.value === status)
      ?.label || status
  );
}

function buildSelectorTemplate(type: AutomationScriptType): string {
  if (type === "launch-api") {
    return `{
  "code": "BUYER_001"
}`;
  }

  return "";
}

function buildParamsTemplate(type: AutomationScriptType): string {
  if (type === "launch-api") {
    return `{
  "startUrls": ["https://example.com"],
  "skipDefaultStartUrls": true
}`;
  }

  return `{
  "url": "https://www.baidu.com",
  "keyword": "OpenAI",
  "timeoutMs": 30000,
  "waitAfterSearchMs": 1500,
  "captureScreenshot": true
}`;
}

function buildScriptTemplate(type: AutomationScriptType): string {
  if (type === "launch-api") {
    return `export async function run({ baseUrl, apiKey, selector, params }) {
  const response = await fetch(\`\${baseUrl}/api/launch\`, {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
      ...(apiKey ? { 'X-Ant-Api-Key': apiKey } : {}),
    },
    body: JSON.stringify({
      selector,
      ...(params || {}),
    }),
  })

  if (!response.ok) {
    throw new Error(\`launch failed: \${response.status}\`)
  }

  return await response.json()
}`;
  }

  return `module.exports.run = async ({ launch, connect, selector, params, log, artifact }) => {
  const targetUrl =
    typeof params.url === 'string' && params.url.trim()
      ? params.url.trim()
      : 'https://www.baidu.com'
  const keyword =
    typeof params.keyword === 'string' && params.keyword.trim()
      ? params.keyword.trim()
      : 'OpenAI'
  const timeout =
    Number.isFinite(Number(params.timeoutMs)) && Number(params.timeoutMs) > 0
      ? Math.round(Number(params.timeoutMs))
      : 30000
  const waitAfterSearchMs =
    Number.isFinite(Number(params.waitAfterSearchMs)) && Number(params.waitAfterSearchMs) >= 0
      ? Math.round(Number(params.waitAfterSearchMs))
      : 1500

  const session = await launch({
    selector,
    startUrls: params.startUrls || [targetUrl],
    skipDefaultStartUrls: true,
  })

  const connection = await connect(session)
  const browser = connection.browser
  const context = connection.context || browser.contexts()[0]
  const page = connection.page || context.pages()[0] || await context.newPage()

  await page.goto(targetUrl, {
    waitUntil: 'domcontentloaded',
    timeout,
  })

  const searchInput = page.locator('textarea[name="wd"], input[name="wd"]').first()
  await searchInput.waitFor({
    state: 'visible',
    timeout,
  })
  await searchInput.fill(keyword)
  await searchInput.press('Enter').catch(async () => {
    const submitButton = page.locator('#su, input[type="submit"]').first()
    await submitButton.click({ timeout })
  })
  await page.waitForURL(/wd=/, { timeout }).catch(() => {})

  if (waitAfterSearchMs > 0) {
    await page.waitForTimeout(waitAfterSearchMs)
  }

  if (params.captureScreenshot !== false) {
    await page.screenshot({
      path: artifact('baidu-search.png'),
      fullPage: true,
    })
  }

  const title = await page.title()
  log('keyword', keyword)
  log('title', title)

  return {
    ok: true,
    summary: \`已在百度搜索 \${keyword}\`,
    keyword,
    url: page.url(),
    title,
  }
}`;
}

function buildNotesTemplate(type: AutomationScriptType): string {
  if (type === "launch-api") {
    return "适合外部调度器或 HTTP 中台。脚本负责组装 selector 和 launch 参数，不直接接管页面。";
  }

  return "默认示例会启动浏览器并搜索 keyword。首次执行可先选择已有实例，或创建一个新实例后再执行。";
}

function buildDualInstanceRuntimeParamsText(codes = [...DUAL_INSTANCE_DEFAULT_CODES]): string {
  return `{
  "browsers": [
    {
      "code": "${codes[0] || DUAL_INSTANCE_DEFAULT_CODES[0]}",
      "skipDefaultStartUrls": true,
      "startUrls": ["${DUAL_INSTANCE_DEFAULT_START_URLS[0]}"]
    },
    {
      "code": "${codes[1] || DUAL_INSTANCE_DEFAULT_CODES[1]}",
      "skipDefaultStartUrls": true,
      "startUrls": ["${DUAL_INSTANCE_DEFAULT_START_URLS[1]}"]
    }
  ],
  "timeoutMs": 45000
}`;
}

function buildDualInstanceRuntimeScriptText(): string {
  return `export async function run({ baseUrl, apiKey, params, log }) {
  const normalizeCode = (value, fallback) =>
    String(value || fallback || "").trim().toUpperCase()
  const normalizeStringArray = (value) =>
    Array.isArray(value)
      ? value
          .map((item) => String(item || "").trim())
          .filter(Boolean)
      : []
  const normalizeBrowserInput = (value, fallbackCode, fallbackStartUrls, defaultSkip) => {
    const raw = value && typeof value === "object" ? value : {}
    const code = normalizeCode(raw.code || raw.launchCode, fallbackCode)
    if (!code) {
      return null
    }
    const startUrls = normalizeStringArray(raw.startUrls)
    const fallbackUrls = normalizeStringArray(fallbackStartUrls)
    const launchArgs = normalizeStringArray(raw.launchArgs)

    return {
      code,
      skipDefaultStartUrls:
        raw.skipDefaultStartUrls !== undefined
          ? raw.skipDefaultStartUrls !== false
          : defaultSkip,
      startUrls: startUrls.length > 0 ? startUrls : fallbackUrls,
      launchArgs,
    }
  }

  const timeoutMs = Number.isFinite(Number(params.timeoutMs))
    ? Math.max(1000, Math.round(Number(params.timeoutMs)))
    : 45000
  const defaultSkipDefaultStartUrls = params.skipDefaultStartUrls !== false

  let browsers = Array.isArray(params.browsers)
    ? params.browsers
        .map((item, index) =>
          normalizeBrowserInput(
            item,
            ${JSON.stringify([...DUAL_INSTANCE_DEFAULT_CODES])}[index] || "",
            ${JSON.stringify([...DUAL_INSTANCE_DEFAULT_START_URLS])}[index] || [],
            defaultSkipDefaultStartUrls,
          ),
        )
        .filter(Boolean)
    : []

  if (browsers.length === 0) {
    browsers = [
      normalizeBrowserInput(
        { code: params.primaryCode, skipDefaultStartUrls: params.skipDefaultStartUrls },
        ${JSON.stringify(DUAL_INSTANCE_DEFAULT_CODES[0])},
        ${JSON.stringify([DUAL_INSTANCE_DEFAULT_START_URLS[0]])},
        defaultSkipDefaultStartUrls,
      ),
      normalizeBrowserInput(
        { code: params.secondaryCode, skipDefaultStartUrls: params.skipDefaultStartUrls },
        ${JSON.stringify(DUAL_INSTANCE_DEFAULT_CODES[1])},
        ${JSON.stringify([DUAL_INSTANCE_DEFAULT_START_URLS[1]])},
        defaultSkipDefaultStartUrls,
      ),
    ].filter(Boolean)
  }

  if (browsers.length === 0) {
    throw new Error("params.browsers 不能为空")
  }

  const headers = {
    "Content-Type": "application/json",
    ...(apiKey ? { "X-Ant-Api-Key": apiKey } : {}),
  }

  const post = async (path, payload) => {
    const response = await fetch(\`\${baseUrl}\${path}\`, {
      method: "POST",
      headers,
      body: JSON.stringify(payload),
    })
    const text = await response.text()
    let body = text
    try {
      body = text ? JSON.parse(text) : null
    } catch {
      body = text
    }
    if (!response.ok) {
      throw new Error(\`\${path} failed: \${response.status} \${text}\`)
    }
    return body
  }

  const sessions = []

  for (const browser of browsers) {
    const sessionResult = await post("/api/runtime/session", {
      selector: { code: browser.code, matchMode: "unique" },
      skipDefaultStartUrls: browser.skipDefaultStartUrls,
      ...(browser.startUrls.length > 0 ? { startUrls: browser.startUrls } : {}),
      ...(browser.launchArgs.length > 0 ? { launchArgs: browser.launchArgs } : {}),
      timeoutMs,
    })

    sessions.push(sessionResult)
  }

  const browserCodes = browsers.map((item) => item.code)
  log("browserCodes", browserCodes)

  return {
    ok: true,
    summary: \`\${browserCodes.length} 个浏览器已就绪：\${browserCodes.join(" / ")}\`,
    browserCodes,
    sessions,
  }
}`;
}

function normalizeDualInstanceRuntimeParamsText(text: string): string {
  const fallback = buildDualInstanceRuntimeParamsText();

  try {
    const parsed = JSON.parse(text);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return fallback;
    }

    const raw = parsed as Record<string, unknown>;
    const topLevelSkipDefaultStartUrls = raw.skipDefaultStartUrls !== false;
    const rawBrowsers = Array.isArray(raw.browsers) ? raw.browsers : [];
    const browsers = rawBrowsers
      .map((item, index) => {
        if (!item || typeof item !== "object") {
          return null;
        }
        const entry = item as Record<string, unknown>;
        const code = normalizeTargetSelector({
          code:
            typeof entry.code === "string"
              ? entry.code
              : typeof entry.launchCode === "string"
                ? entry.launchCode
                : "",
        }).code;
        if (!code) {
          return null;
        }

        const startUrls = Array.isArray(entry.startUrls)
          ? entry.startUrls
              .map((value) => String(value || "").trim())
              .filter(Boolean)
          : [];
        const launchArgs = Array.isArray(entry.launchArgs)
          ? entry.launchArgs
              .map((value) => String(value || "").trim())
              .filter(Boolean)
          : [];

        const fallbackStartUrls = DUAL_INSTANCE_DEFAULT_START_URLS[index]
          ? [DUAL_INSTANCE_DEFAULT_START_URLS[index]]
          : [];

        return {
          code: code || DUAL_INSTANCE_DEFAULT_CODES[index] || "",
          skipDefaultStartUrls:
            entry.skipDefaultStartUrls !== undefined
              ? entry.skipDefaultStartUrls !== false
              : topLevelSkipDefaultStartUrls,
          startUrls: startUrls.length > 0 ? startUrls : fallbackStartUrls,
          ...(launchArgs.length > 0 ? { launchArgs } : {}),
        };
      })
      .filter(
        (
          item,
        ): item is {
          code: string;
          skipDefaultStartUrls: boolean;
          startUrls: string[];
          launchArgs?: string[];
        } => item !== null,
      );

    const legacyCodes = [
      normalizeTargetSelector({
        code: typeof raw.primaryCode === "string" ? raw.primaryCode : "",
      }).code,
      normalizeTargetSelector({
        code: typeof raw.secondaryCode === "string" ? raw.secondaryCode : "",
      }).code,
    ].filter(Boolean);

    const normalizedBrowsers =
      browsers.length > 0
        ? browsers
        : legacyCodes.length > 0
          ? legacyCodes.map((code, index) => ({
              code,
              skipDefaultStartUrls: topLevelSkipDefaultStartUrls,
              startUrls: DUAL_INSTANCE_DEFAULT_START_URLS[index]
                ? [DUAL_INSTANCE_DEFAULT_START_URLS[index]]
                : [],
            }))
          : DUAL_INSTANCE_DEFAULT_CODES.map((code, index) => ({
              code,
              skipDefaultStartUrls: true,
              startUrls: DUAL_INSTANCE_DEFAULT_START_URLS[index]
                ? [DUAL_INSTANCE_DEFAULT_START_URLS[index]]
                : [],
            }));

    const timeoutMs =
      Number.isFinite(Number(raw.timeoutMs)) && Number(raw.timeoutMs) > 0
        ? Math.round(Number(raw.timeoutMs))
        : 45000;

    return JSON.stringify(
      {
        browsers: normalizedBrowsers,
        timeoutMs,
      },
      null,
      2,
    );
  } catch {
    return fallback;
  }
}

function createNewsTxtScriptDraft(): AutomationScriptRecord {
  const createdAt = nowIso();

  return {
    packageFormat: AUTOMATION_SCRIPT_PACKAGE_FORMAT,
    manifestVersion: AUTOMATION_SCRIPT_MANIFEST_VERSION,
    id: "news-query-txt",
    name: "查询新闻并写 TXT",
    description: "通过 Bing 搜索新闻关键词，提取结果并写入本地 txt 文件。",
    type: "playwright-cdp",
    status: "ready",
    entryFile: "index.cjs",
    tags: ["Playwright", "新闻", "TXT"],
    selectorText: "",
    paramsText: `{
  "keyword": "OpenAI",
  "limit": 10,
  "timeRange": "week",
  "outputFileName": "openai-news.txt",
  "timeoutMs": 30000,
  "waitAfterLoadMs": 1500,
  "captureScreenshot": false
}`,
    scriptText: String.raw`const fs = require('fs')

const DEFAULT_EXCLUDED_DOMAINS = [
  'zhihu.com',
  'baidu.com',
  'qq.com',
  '36kr.com',
  'apifox.com',
  'chatgpt-chinese.com',
  'openwebui.cn',
  'open-openai.com',
  'xiniushu.com',
  'reddit.com',
  'quora.com',
  'tieba.baidu.com',
  'weibo.com',
  'x.com',
  'twitter.com',
  'youtube.com',
  'bilibili.com',
  'douyin.com',
  'xiaohongshu.com',
]

function normalizeInt(value, fallback, min, max) {
  const parsed = Number(value)
  if (!Number.isFinite(parsed)) {
    return fallback
  }

  const rounded = Math.round(parsed)
  if (rounded < min) {
    return min
  }
  if (rounded > max) {
    return max
  }
  return rounded
}

function normalizeText(value) {
  return String(value || '').trim()
}

function normalizeDomainList(value) {
  if (!Array.isArray(value)) {
    return []
  }

  const deduped = new Set()
  for (const item of value) {
    const normalized = normalizeText(item).replace(/^https?:\/\//, '').replace(/^www\./, '').toLowerCase()
    if (normalized) {
      deduped.add(normalized)
    }
  }
  return Array.from(deduped)
}

function buildDefaultQuery(keyword) {
  const normalizedKeyword = normalizeText(keyword) || 'OpenAI'
  if (/[\u3400-\u9fff]/.test(normalizedKeyword)) {
    return normalizedKeyword + ' 新闻'
  }
  return normalizedKeyword + ' news'
}

function buildFallbackQueries(keyword, baseQuery) {
  const normalizedKeyword = normalizeText(keyword) || 'OpenAI'
  const normalizedBaseQuery = normalizeText(baseQuery)
  const candidates = [
    normalizedBaseQuery,
  ]

  if (/[\u3400-\u9fff]/.test(normalizedKeyword)) {
    candidates.push(normalizedKeyword + ' 最新新闻')
  } else {
    candidates.push(normalizedKeyword + ' latest news')
  }

  const deduped = new Set()
  for (const item of candidates) {
    const normalized = normalizeText(item)
    if (normalized) {
      deduped.add(normalized)
    }
  }
  return Array.from(deduped)
}

function buildSearchQuery(baseQuery, excludedDomains) {
  const normalizedBaseQuery = normalizeText(baseQuery)
  const normalizedDomains = normalizeDomainList(excludedDomains)
  const parts = [normalizedBaseQuery]

  for (const domain of normalizedDomains) {
    parts.push('-site:' + domain)
  }

  return parts.filter(Boolean).join(' ')
}

function mapTimeRangeToBingFilter(value) {
  switch (normalizeText(value).toLowerCase()) {
    case 'day':
    case '24h':
    case 'today':
      return 'ex1:"ez1"'
    case 'week':
      return 'ex1:"ez2"'
    case 'month':
      return 'ex1:"ez3"'
    default:
      return ''
  }
}

function buildSearchURL(query, timeRange, firstResultIndex) {
  const searchParams = new URLSearchParams({ q: query })
  const filter = mapTimeRangeToBingFilter(timeRange)
  if (filter) {
    searchParams.set('filters', filter)
  }
  if (Number.isFinite(firstResultIndex) && firstResultIndex > 1) {
    searchParams.set('first', String(firstResultIndex))
  }
  return 'https://www.bing.com/search?' + searchParams.toString()
}

function splitSnippet(snippet) {
  const normalized = normalizeText(snippet)
  if (!normalized) {
    return { publishedAt: '', summary: '' }
  }

  const match = normalized.match(/^([^·]{0,40})\s*·\s*(.+)$/)
  if (
    match &&
    /(前|分钟|小时|天前|周前|月前|昨天|\d{4}|\d{1,2}[/-]\d{1,2})/.test(match[1])
  ) {
    return {
      publishedAt: normalizeText(match[1]),
      summary: normalizeText(match[2]),
    }
  }

  return {
    publishedAt: '',
    summary: normalized,
  }
}

function parseHostname(rawUrl) {
  const normalized = normalizeText(rawUrl)
  if (!normalized) {
    return ''
  }

  try {
    return new URL(normalized).hostname.replace(/^www\./, '').toLowerCase()
  } catch {
    return ''
  }
}

function parsePathname(rawUrl) {
  const normalized = normalizeText(rawUrl)
  if (!normalized) {
    return ''
  }

  try {
    const pathname = new URL(normalized).pathname.replace(/\/+/g, '/').toLowerCase()
    if (!pathname) {
      return ''
    }
    return pathname === '/' ? pathname : pathname.replace(/\/$/, '')
  } catch {
    return ''
  }
}

function looksLikeQuestionTitle(title) {
  const normalized = normalizeText(title)
  if (!normalized) {
    return false
  }

  if (/[？?]/.test(normalized)) {
    return true
  }

  return /^(如何|为什么|怎么看|怎样|怎么|是否|有没有|谁能|请问|评价|如何评价|如何看待|为什么说)/.test(normalized)
}

function looksLikeAggregateText(text) {
  const normalized = normalizeText(text).toLowerCase()
  if (!normalized) {
    return false
  }

  return /(roundup|digest|flash report|llm news today|ai news today|daily ai news|news today|model releases)/.test(normalized)
}

function looksLikeListingPath(pathname) {
  const normalized = normalizeText(pathname).toLowerCase()
  if (!normalized || normalized === '/') {
    return false
  }

  if (/(^|\/)(tag|tags|topic|topics|category|categories|label|labels|brand|brands)(\/|$)/.test(normalized)) {
    return true
  }

  if (/(^|\/)(news|latest|headlines|insights)$/.test(normalized)) {
    return true
  }

  return /\/news\/(brand|brands|topic|topics|tag|tags)(\/|$)/.test(normalized)
}

function looksLikeListingText(text) {
  const normalized = normalizeText(text).toLowerCase()
  if (!normalized) {
    return false
  }

  return /(latest news|breaking headlines|news and insights|news and analysis|everything you need to know|get the latest|最新资讯|最新动态|实时追踪|热点快讯|快讯)/.test(normalized)
}

function isBlockedHostname(hostname) {
  const normalized = normalizeText(hostname).toLowerCase()
  if (!normalized) {
    return false
  }

  const blockedSuffixes = DEFAULT_EXCLUDED_DOMAINS
  const blockedKeywords = [
    'aitrack',
    'aitoolly',
    'aiflashreport',
    'llm-stats',
    'opentools',
  ]

  if (blockedSuffixes.some(function (suffix) {
    return normalized === suffix || normalized.endsWith('.' + suffix)
  })) {
    return true
  }

  return blockedKeywords.some(function (keyword) {
    return normalized.includes(keyword)
  })
}

function evaluateNewsItem(item) {
  const hostname = parseHostname(item.url)
  const pathname = parsePathname(item.url)
  const summary = normalizeText(item.summary)
  const source = normalizeText(item.source)
  const reasons = []

  if (!normalizeText(item.url)) {
    reasons.push('missing-url')
  }
  if (!hostname) {
    reasons.push('invalid-url')
  }
  if (hostname && isBlockedHostname(hostname)) {
    reasons.push('blocked-host')
  }
  if (!source) {
    reasons.push('missing-source')
  }
  if (summary.length < 20) {
    reasons.push('summary-too-short')
  }
  if (looksLikeQuestionTitle(item.title)) {
    reasons.push('question-title')
  }
  if (looksLikeAggregateText(item.title) || looksLikeAggregateText(summary)) {
    reasons.push('aggregate-page')
  }
  if (looksLikeListingPath(pathname) || looksLikeListingText(item.title) || looksLikeListingText(summary)) {
    reasons.push('listing-page')
  }

  return Object.assign({}, item, {
    hostname: hostname,
    pathname: pathname,
    qualityAccepted: reasons.length === 0,
    qualityReasons: reasons,
  })
}

function formatRejectedReason(reason) {
  switch (reason) {
    case 'missing-url':
      return '缺少链接'
    case 'invalid-url':
      return '链接无效'
    case 'blocked-host':
      return '来源站点已过滤'
    case 'missing-source':
      return '缺少来源'
    case 'summary-too-short':
      return '摘要过短'
    case 'question-title':
      return '标题更像问答'
    case 'aggregate-page':
      return '更像聚合页'
    case 'listing-page':
      return '更像列表页/专题页'
    default:
      return reason
  }
}

function formatReport(items, metadata) {
  const lines = [
    '新闻抓取结果',
    '查询词: ' + metadata.query,
    '抓取时间: ' + metadata.generatedAt,
    '搜索地址: ' + metadata.searchUrl,
    '原始结果: ' + metadata.rawCount,
    '通过校验: ' + items.length,
    '过滤数量: ' + metadata.rejectedItems.length,
    '',
  ]

  for (const item of items) {
    lines.push(item.rank + '. ' + item.title)
    if (item.source) {
      lines.push('来源: ' + item.source)
    }
    if (item.publishedAt) {
      lines.push('时间: ' + item.publishedAt)
    }
    lines.push('链接: ' + item.url)
    if (item.summary) {
      lines.push('摘要: ' + item.summary)
    }
    lines.push('')
  }

  if (metadata.rejectedItems.length > 0) {
    lines.push('被过滤结果（最多展示 5 条）')
    lines.push('')
    for (const item of metadata.rejectedItems.slice(0, 5)) {
      lines.push(item.rank + '. ' + item.title)
      if (item.hostname) {
        lines.push('站点: ' + item.hostname)
      }
      lines.push('原因: ' + item.qualityReasons.map(formatRejectedReason).join(' / '))
      lines.push('')
    }
  }

  return lines.join('\n')
}

function pickBestAttempt(current, candidate) {
  if (!current) {
    return candidate
  }

  if (candidate.acceptedItems.length !== current.acceptedItems.length) {
    return candidate.acceptedItems.length > current.acceptedItems.length ? candidate : current
  }

  if (candidate.distinctHostCount !== current.distinctHostCount) {
    return candidate.distinctHostCount > current.distinctHostCount ? candidate : current
  }

  if (candidate.rawItems.length !== current.rawItems.length) {
    return candidate.rawItems.length > current.rawItems.length ? candidate : current
  }

  return candidate
}

module.exports.run = async ({ launch, connect, selector, params, log, artifact }) => {
  const timeout = normalizeInt(params.timeoutMs, 30000, 1000, 120000)
  const waitAfterLoadMs = normalizeInt(params.waitAfterLoadMs, 1500, 0, 10000)
  const limit = normalizeInt(params.limit, 10, 1, 50)
  const maxPages = normalizeInt(params.maxPages, 3, 1, 5)
  const baseQuery = normalizeText(params.query) || buildDefaultQuery(params.keyword)
  const excludedDomains = normalizeDomainList(params.excludeDomains).length > 0
    ? normalizeDomainList(params.excludeDomains)
    : DEFAULT_EXCLUDED_DOMAINS
  const outputFileName = normalizeText(params.outputFileName) || 'news-results.txt'
  const scanLimit = Math.max(10, Math.min(20, limit * 2))
  const startUrls = Array.isArray(params.startUrls) && params.startUrls.length > 0
    ? params.startUrls
    : undefined

  const session = await launch({
    selector,
    startUrls,
    skipDefaultStartUrls: true,
  })

  const connection = await connect(session)
  const browser = connection.browser
  const context = connection.context || browser.contexts()[0]
  const page = await context.newPage()
  const closeRunnerPage = async function () {
    if (!page.isClosed()) {
      await page.close().catch(function () {})
    }
  }

  const searchCandidates = buildFallbackQueries(params.keyword, baseQuery)
  const minAcceptedCount = Math.min(limit, Math.max(2, Math.ceil(limit * 0.2)))
  const minDistinctHostCount = Math.min(3, minAcceptedCount)
  let bestAttempt = null

  try {
    for (const candidateQuery of searchCandidates) {
      const searchQuery = buildSearchQuery(candidateQuery, excludedDomains)
      const normalizedItems = []
      const seenUrls = new Set()
      let scannedPageCount = 0
      let firstSearchUrl = ''

      for (let pageIndex = 0; pageIndex < maxPages; pageIndex += 1) {
        const firstResultIndex = pageIndex * 10 + 1
        const searchUrl = buildSearchURL(searchQuery, params.timeRange, firstResultIndex)

        try {
          await page.goto(searchUrl, {
            waitUntil: 'domcontentloaded',
            timeout,
          })
          await page.waitForSelector('li.b_algo', { timeout })
        } catch (error) {
          if (pageIndex > 0 && normalizedItems.length > 0) {
            break
          }
          throw error
        }

        if (waitAfterLoadMs > 0) {
          await page.waitForTimeout(waitAfterLoadMs)
        }

        if (!firstSearchUrl) {
          firstSearchUrl = page.url()
        }

        const pageItems = await page.$$eval('li.b_algo', function (nodes, maxItems) {
          const clean = function (value) {
            return String(value || '').replace(/\s+/g, ' ').trim()
          }

          return nodes
            .slice(0, maxItems)
            .map(function (node) {
              const titleLink = node.querySelector('h2 a')
              const title = clean(titleLink && titleLink.textContent)
              const url = titleLink ? titleLink.href : ''
              const sourceNode = node.querySelector('.tptt')
              const source = clean(sourceNode && sourceNode.textContent)
              const citeNode = node.querySelector('.b_attribution cite')
              const cite = clean(citeNode && citeNode.textContent)
              const snippetNode = node.querySelector('.b_caption p')
              const snippet = clean(snippetNode && snippetNode.textContent)

              if (!title) {
                return null
              }

              return {
                title,
                url,
                source: source || cite,
                snippet,
              }
            })
            .filter(Boolean)
        }, scanLimit)

        let appendedCount = 0
        for (const item of pageItems) {
          const dedupeKey = normalizeText(item.url)
          if (!dedupeKey || seenUrls.has(dedupeKey)) {
            continue
          }

          seenUrls.add(dedupeKey)
          normalizedItems.push(
            evaluateNewsItem(
              Object.assign(
                {
                  rank: normalizedItems.length + 1,
                },
                item,
                splitSnippet(item.snippet)
              )
            )
          )
          appendedCount += 1
        }

        scannedPageCount += 1
        if (appendedCount === 0 || pageItems.length < 8) {
          break
        }
      }

      const acceptedItems = normalizedItems.filter(function (item) {
        return item.qualityAccepted
      }).slice(0, limit)
      const rejectedItems = normalizedItems.filter(function (item) {
        return !item.qualityAccepted
      })
      const distinctHostCount = new Set(
        acceptedItems
          .map(function (item) {
            return item.hostname
          })
          .filter(Boolean)
      ).size

      log('searchQuery', searchQuery)
      log('rawItemCount', normalizedItems.length)
      log('acceptedItemCount', acceptedItems.length)
      log('rejectedItemCount', rejectedItems.length)
      log('distinctHostCount', distinctHostCount)
      log('scannedPageCount', scannedPageCount)

      bestAttempt = pickBestAttempt(bestAttempt, {
        baseQuery: candidateQuery,
        searchQuery: searchQuery,
        searchUrl: firstSearchUrl || page.url(),
        rawItems: normalizedItems,
        acceptedItems: acceptedItems,
        rejectedItems: rejectedItems,
        distinctHostCount: distinctHostCount,
        scannedPageCount: scannedPageCount,
      })

      if (acceptedItems.length >= minAcceptedCount && distinctHostCount >= minDistinctHostCount) {
        break
      }
    }
  } catch (error) {
    await closeRunnerPage()
    throw error
  }

  if (!bestAttempt || bestAttempt.rawItems.length === 0) {
    await closeRunnerPage()
    throw new Error('未抓到新闻搜索结果，当前页面: ' + page.url())
  }

  const normalizedItems = bestAttempt.rawItems
  const acceptedItems = bestAttempt.acceptedItems
  const rejectedItems = bestAttempt.rejectedItems
  const distinctHostCount = bestAttempt.distinctHostCount
  const searchUrl = bestAttempt.searchUrl
  const scannedPageCount = bestAttempt.scannedPageCount || 1

  const outputName = outputFileName.toLowerCase().endsWith('.txt')
    ? outputFileName
    : outputFileName + '.txt'
  const outputPath = artifact(outputName)
  const reportText = formatReport(acceptedItems, {
    query: bestAttempt.baseQuery,
    generatedAt: new Date().toISOString(),
    searchUrl: searchUrl,
    rawCount: normalizedItems.length,
    rejectedItems: rejectedItems,
  })
  fs.writeFileSync(outputPath, reportText, 'utf8')

  let screenshotPath = ''
  if (params.captureScreenshot === true) {
    screenshotPath = artifact('news-search.png')
    await page.screenshot({
      path: screenshotPath,
      fullPage: true,
    })
  }

  log('outputPath', outputPath)
  await closeRunnerPage()

  if (acceptedItems.length < minAcceptedCount || distinctHostCount < minDistinctHostCount) {
    return {
      ok: false,
      summary: '新闻结果质量不足，仅 ' + acceptedItems.length + '/' + normalizedItems.length + ' 条通过校验',
      error: '搜索结果更像普通搜索、问答页或聚合页，未达到新闻抓取标准',
      query: bestAttempt.baseQuery,
      searchQuery: bestAttempt.searchQuery,
      searchUrl: searchUrl,
      outputPath,
      screenshotPath,
      rawItemCount: normalizedItems.length,
      itemCount: acceptedItems.length,
      rejectedCount: rejectedItems.length,
      distinctHostCount: distinctHostCount,
      scannedPageCount: scannedPageCount,
      firstTitle: acceptedItems[0] ? acceptedItems[0].title : '',
    }
  }

  return {
    ok: true,
    summary: '已筛出 ' + acceptedItems.length + ' 条有效新闻并写入 TXT',
    query: bestAttempt.baseQuery,
    searchQuery: bestAttempt.searchQuery,
    searchUrl: searchUrl,
    outputPath,
    screenshotPath,
    rawItemCount: normalizedItems.length,
    itemCount: acceptedItems.length,
    rejectedCount: rejectedItems.length,
    distinctHostCount: distinctHostCount,
    scannedPageCount: scannedPageCount,
    firstTitle: acceptedItems[0] ? acceptedItems[0].title : '',
  }
}`,
    notes:
      "脚本会优先使用 Bing 搜索真实新闻结果，并自动追加时间过滤、排除问答/聚合站点、回退查询词和质量校验；只有达到新闻质量门槛时才会判定成功，并把结果写入本地 txt。执行成功后可在结果里的 outputPath 找到文件。",
    targetConfig: normalizeAutomationScriptTargetConfig(null),
    source: {
      type: "builtin",
      uri: "repo://backend/internal/automation/default_scripts.go",
      ref: "HEAD",
      path: "news-query-txt",
      importedAt: "",
    },
    createdAt,
    updatedAt: createdAt,
  };
}

function createDualInstanceRuntimeScriptDraft(): AutomationScriptRecord {
  const createdAt = nowIso();

  return {
    packageFormat: AUTOMATION_SCRIPT_PACKAGE_FORMAT,
    manifestVersion: AUTOMATION_SCRIPT_MANIFEST_VERSION,
    id: DUAL_INSTANCE_RUNTIME_SCRIPT_ID,
    name: "双实例启动与 Runtime 切换",
    description:
      "通过 Launch API 分别启动两个实例，切换 Runtime 会话后交给 OpenClaw 执行。",
    type: "launch-api",
    status: "ready",
    entryFile: "index.cjs",
    tags: ["Launch API", "OpenClaw", "双实例"],
    selectorText: "",
    paramsText: buildDualInstanceRuntimeParamsText(),
    scriptText: buildDualInstanceRuntimeScriptText(),
    notes:
      "先通过接口启动两个实例并切换 Runtime 会话；随后把实例信息交给 OpenClaw 执行自动化动作。",
    targetConfig: normalizeAutomationScriptTargetConfig(null),
    source: {
      type: "builtin",
      uri: "repo://backend/internal/automation/default_scripts.go",
      ref: "HEAD",
      path: "dual-instance-runtime-switch",
      importedAt: "",
    },
    createdAt,
    updatedAt: createdAt,
  };
}

function normalizeTags(tags: unknown): string[] {
  if (!Array.isArray(tags)) {
    return [];
  }

  const deduped = new Set<string>();
  for (const item of tags) {
    const normalized = String(item || "").trim();
    if (normalized) {
      deduped.add(normalized);
    }
  }
  return Array.from(deduped);
}

function normalizeScriptRecord(raw: unknown): AutomationScriptRecord | null {
  if (!raw || typeof raw !== "object") {
    return null;
  }

  const source = raw as Partial<AutomationScriptRecord>;
  const type = source.type === "launch-api" ? "launch-api" : "playwright-cdp";
  const status =
    source.status === "ready" || source.status === "disabled"
      ? source.status
      : "draft";
  const createdAt =
    typeof source.createdAt === "string" && source.createdAt.trim()
      ? source.createdAt
      : nowIso();
  const updatedAt =
    typeof source.updatedAt === "string" && source.updatedAt.trim()
      ? source.updatedAt
      : createdAt;
  const normalizedSource = normalizeSource(source.source);
  const normalizedTargetConfig = normalizeAutomationScriptTargetConfig(
    source.targetConfig,
  );

  const record: AutomationScriptRecord = {
    packageFormat:
      typeof source.packageFormat === "string" && source.packageFormat.trim()
        ? source.packageFormat.trim()
        : AUTOMATION_SCRIPT_PACKAGE_FORMAT,
    manifestVersion:
      typeof source.manifestVersion === "number" && source.manifestVersion > 0
        ? source.manifestVersion
        : AUTOMATION_SCRIPT_MANIFEST_VERSION,
    id:
      typeof source.id === "string" && source.id.trim()
        ? source.id
        : createScriptId(),
    name:
      typeof source.name === "string" && source.name.trim()
        ? source.name.trim()
        : "未命名脚本",
    description:
      typeof source.description === "string" ? source.description.trim() : "",
    type,
    status,
    entryFile:
      typeof source.entryFile === "string" && source.entryFile.trim()
        ? source.entryFile.trim()
        : "index.cjs",
    tags: normalizeTags(source.tags),
    selectorText:
      typeof source.selectorText === "string" && source.selectorText.trim()
        ? source.selectorText
        : buildSelectorTemplate(type),
    paramsText:
      typeof source.paramsText === "string" && source.paramsText.trim()
        ? source.paramsText
        : buildParamsTemplate(type),
    scriptText:
      typeof source.scriptText === "string" && source.scriptText.trim()
        ? source.scriptText
        : buildScriptTemplate(type),
    notes:
      typeof source.notes === "string" && source.notes.trim()
        ? source.notes
        : buildNotesTemplate(type),
    targetConfig: normalizedTargetConfig,
    source: normalizedSource,
    createdAt,
    updatedAt,
  };

  if (record.id === DUAL_INSTANCE_RUNTIME_SCRIPT_ID) {
    const dualInstanceDraft = createDualInstanceRuntimeScriptDraft();
    const usesLegacyDualInstanceScript =
      record.scriptText.includes("params.primaryCode") ||
      record.scriptText.includes("params.secondaryCode");

    return {
      ...record,
      selectorText: "",
      paramsText: normalizeDualInstanceRuntimeParamsText(record.paramsText),
      targetConfig: normalizeAutomationScriptTargetConfig(null),
      scriptText: usesLegacyDualInstanceScript
        ? dualInstanceDraft.scriptText
        : record.scriptText,
    };
  }

  return record;
}

export function normalizeAutomationScriptRecordPayload(
  raw: unknown,
): AutomationScriptRecord | null {
  return normalizeScriptRecord(raw);
}

export function createAutomationScriptDraft(
  type: AutomationScriptType = "playwright-cdp",
): AutomationScriptRecord {
  const createdAt = nowIso();
  const name =
    type === "launch-api" ? "新建 Launch API 脚本" : "百度搜索示例";
  const description =
    type === "launch-api"
      ? ""
      : "启动示例实例，打开百度并搜索关键词，用来验证 Launch API + Playwright CDP 链路。";

  return {
    packageFormat: AUTOMATION_SCRIPT_PACKAGE_FORMAT,
    manifestVersion: AUTOMATION_SCRIPT_MANIFEST_VERSION,
    id: createScriptId(),
    name,
    description,
    type,
    status: type === "playwright-cdp" ? "ready" : "draft",
    entryFile: "index.cjs",
    tags: type === "launch-api" ? ["HTTP"] : ["Playwright", "示例"],
    selectorText: buildSelectorTemplate(type),
    paramsText: buildParamsTemplate(type),
    scriptText: buildScriptTemplate(type),
    notes: buildNotesTemplate(type),
    targetConfig: normalizeAutomationScriptTargetConfig(null),
    source: {
      type: "manual",
      uri: "",
      ref: "",
      path: "",
      importedAt: "",
    },
    createdAt,
    updatedAt: createdAt,
  };
}

export function duplicateAutomationScript(
  script: AutomationScriptRecord,
): AutomationScriptRecord {
  const createdAt = nowIso();
  return {
    ...script,
    id: createScriptId(),
    name: `${script.name} - 副本`,
    status: "draft",
    createdAt,
    updatedAt: createdAt,
  };
}

function stringifyJsonBlock(value: unknown, fallback: string): string {
  if (typeof value === "string" && value.trim()) {
    return value;
  }

  if (value && typeof value === "object") {
    try {
      return JSON.stringify(value, null, 2);
    } catch {
      return fallback;
    }
  }

  return fallback;
}

export function importAutomationScript(text: string): AutomationScriptRecord {
  const normalized = text.trim();
  if (!normalized) {
    throw new Error("导入内容不能为空");
  }

  let parsed: any;
  try {
    parsed = JSON.parse(normalized);
  } catch {
    throw new Error("导入内容不是合法 JSON");
  }

  const manifest =
    parsed?.manifest && typeof parsed.manifest === "object"
      ? parsed.manifest
      : parsed;
  const type: AutomationScriptType =
    manifest?.type === "launch-api" ? "launch-api" : "playwright-cdp";
  const timestamp = nowIso();
  const imported = normalizeScriptRecord({
    packageFormat:
      typeof parsed?.packageFormat === "string"
        ? parsed.packageFormat
        : typeof parsed?.format === "string"
          ? parsed.format
          : AUTOMATION_SCRIPT_PACKAGE_FORMAT,
    manifestVersion:
      typeof parsed?.manifestVersion === "number"
        ? parsed.manifestVersion
        : AUTOMATION_SCRIPT_MANIFEST_VERSION,
    id: createScriptId(),
    name: typeof manifest?.name === "string" ? manifest.name : undefined,
    description:
      typeof manifest?.description === "string" ? manifest.description : "",
    type,
    status: "draft",
    entryFile:
      typeof manifest?.entryFile === "string"
        ? manifest.entryFile
        : "index.cjs",
    tags: Array.isArray(manifest?.tags) ? manifest.tags : [],
    selectorText: stringifyJsonBlock(
      parsed?.selector ?? parsed?.selectorText,
      buildSelectorTemplate(type),
    ),
    paramsText: stringifyJsonBlock(
      parsed?.params ?? parsed?.paramsText,
      buildParamsTemplate(type),
    ),
    scriptText:
      typeof parsed?.script === "string"
        ? parsed.script
        : typeof parsed?.scriptText === "string"
          ? parsed.scriptText
          : buildScriptTemplate(type),
    notes:
      typeof parsed?.notes === "string"
        ? parsed.notes
        : typeof parsed?.manifest?.notes === "string"
          ? parsed.manifest.notes
          : buildNotesTemplate(type),
    targetConfig:
      parsed?.targetConfig && typeof parsed.targetConfig === "object"
        ? parsed.targetConfig
        : parsed?.manifest?.targetConfig &&
            typeof parsed.manifest.targetConfig === "object"
          ? parsed.manifest.targetConfig
          : null,
    source:
      parsed?.source && typeof parsed.source === "object"
        ? parsed.source
        : {
            type: "text",
            uri: "",
            ref: "",
            path: "",
            importedAt: timestamp,
          },
    createdAt: timestamp,
    updatedAt: timestamp,
  });

  if (!imported) {
    throw new Error("导入内容无法识别为脚本");
  }

  return imported;
}

function buildDefaultScripts(): AutomationScriptRecord[] {
  const dualInstanceScript = createDualInstanceRuntimeScriptDraft();
  const newsScript = createNewsTxtScriptDraft();

  return [newsScript, dualInstanceScript];
}

function sortScripts(
  items: AutomationScriptRecord[],
): AutomationScriptRecord[] {
  return [...items].sort((left, right) => {
    return (
      new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime()
    );
  });
}

export function loadAutomationScripts(): AutomationScriptRecord[] {
  if (typeof window === "undefined" || !window.localStorage) {
    return buildDefaultScripts();
  }

  try {
    const raw = window.localStorage.getItem(AUTOMATION_SCRIPTS_STORAGE_KEY);
    if (!raw) {
      return buildDefaultScripts();
    }

    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      return buildDefaultScripts();
    }

    const scripts = parsed
      .map((item) => normalizeScriptRecord(item))
      .filter((item): item is AutomationScriptRecord => item !== null);

    if (scripts.length === 0) {
      return buildDefaultScripts();
    }

    return sortScripts(scripts);
  } catch {
    return buildDefaultScripts();
  }
}

export function saveAutomationScripts(scripts: AutomationScriptRecord[]) {
  if (typeof window === "undefined" || !window.localStorage) {
    return;
  }

  try {
    window.localStorage.setItem(
      AUTOMATION_SCRIPTS_STORAGE_KEY,
      JSON.stringify(sortScripts(scripts)),
    );
  } catch {
    // Ignore storage failures and keep the editor usable.
  }
}

export function exportAutomationScript(script: AutomationScriptRecord): string {
  return JSON.stringify(
    {
      manifest: {
        packageFormat: script.packageFormat,
        manifestVersion: script.manifestVersion,
        id: script.id,
        name: script.name,
        description: script.description,
        type: script.type,
        status: script.status,
        entryFile: script.entryFile,
        tags: script.tags,
        notes: script.notes,
        targetConfig: script.targetConfig,
        source: script.source,
        createdAt: script.createdAt,
        updatedAt: script.updatedAt,
      },
      format: script.packageFormat,
      manifestVersion: script.manifestVersion,
      selector: safeParseJson(script.selectorText),
      params: safeParseJson(script.paramsText),
      script: script.scriptText,
      notes: script.notes,
      targetConfig: script.targetConfig,
      source: script.source,
    },
    null,
    2,
  );
}

function safeParseJson(text: string): unknown {
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}
