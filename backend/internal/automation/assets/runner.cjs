const fs = require('fs');
const http = require('http');
const https = require('https');
const path = require('path');
const util = require('util');
const { pathToFileURL } = require('url');

const ALLOWED_WAIT_UNTIL = new Set(['load', 'domcontentloaded', 'networkidle', 'commit']);

function normalizeTimeout(value, fallback) {
  const parsed = Number(value);
  if (Number.isFinite(parsed) && parsed > 0) {
    return Math.round(parsed);
  }
  return fallback;
}

function sleep(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function writeStream(stream, text) {
  return new Promise((resolve, reject) => {
    stream.write(text, (error) => {
      if (error) {
        reject(error);
        return;
      }
      resolve();
    });
  });
}

async function closeBrowserConnection(browser) {
  if (!browser || typeof browser.close !== 'function') {
    return;
  }
  await browser.close({ reason: 'automation task finished' }).catch(() => {});
}

function normalizeEndpointCandidate(value) {
  const normalized = String(value || '').trim();
  if (!normalized) {
    return '';
  }

  try {
    const parsed = new URL(normalized);
    if (!['http:', 'https:', 'ws:', 'wss:'].includes(parsed.protocol)) {
      return '';
    }
    if (parsed.port === '0') {
      return '';
    }
    if ((parsed.protocol === 'http:' || parsed.protocol === 'https:') && (!parsed.pathname || parsed.pathname === '/') && !parsed.search && !parsed.hash) {
      return parsed.origin;
    }
    return parsed.toString();
  } catch {
    return '';
  }
}

function buildConnectEndpoints(payload, session) {
  const candidates = [];
  const seen = new Set();

  const pushCandidate = (value) => {
    const endpoint = normalizeEndpointCandidate(value);
    if (!endpoint || seen.has(endpoint)) {
      return;
    }
    seen.add(endpoint);
    candidates.push(endpoint);
  };

  pushCandidate(session && session.cdpUrl);

  const debugPort = Number(session && session.debugPort);
  if (Number.isFinite(debugPort) && debugPort > 0) {
    pushCandidate(`http://127.0.0.1:${Math.round(debugPort)}`);
  }

  pushCandidate(payload && payload.launchBaseUrl);
  return candidates;
}

function normalizePathUnderRoot(rootDir, targetName) {
  const normalizedName = String(targetName || '').trim();
  const resolvedRoot = path.resolve(String(rootDir || ''));
  if (!resolvedRoot) {
    throw new Error('artifactDir is required');
  }

  const candidate = normalizedName ? path.resolve(resolvedRoot, normalizedName) : resolvedRoot;
  if (candidate !== resolvedRoot && !candidate.startsWith(`${resolvedRoot}${path.sep}`)) {
    throw new Error('artifact path escapes root directory');
  }
  return candidate;
}

async function requestJSON(method, requestURL, body, headers = {}) {
  const target = new URL(requestURL);
  const transport = target.protocol === 'https:' ? https : http;
  const payload = body == null ? '' : JSON.stringify(body);

  return await new Promise((resolve, reject) => {
    const req = transport.request(
      {
        protocol: target.protocol,
        hostname: target.hostname,
        port: target.port,
        path: `${target.pathname}${target.search}`,
        method,
        headers: {
          Accept: 'application/json',
          ...(payload
            ? {
                'Content-Type': 'application/json',
                'Content-Length': Buffer.byteLength(payload),
              }
            : {}),
          ...headers,
        },
      },
      (res) => {
        const chunks = [];
        res.on('data', (chunk) => chunks.push(chunk));
        res.on('end', () => {
          const rawText = Buffer.concat(chunks).toString('utf8').trim();
          let responseBody = {};
          if (rawText) {
            try {
              responseBody = JSON.parse(rawText);
            } catch {
              responseBody = { rawBody: rawText };
            }
          }
          resolve({
            status: res.statusCode || 0,
            body: responseBody,
          });
        });
      }
    );

    req.on('error', reject);
    if (payload) {
      req.write(payload);
    }
    req.end();
  });
}

function inspectValue(value) {
  return util.inspect(value, {
    depth: 4,
    breakLength: 120,
    maxArrayLength: 20,
    compact: false,
  });
}

function toSerializable(value, seen = new WeakSet()) {
  if (value == null) {
    return value;
  }
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return value;
  }
  if (typeof value === 'bigint') {
    return value.toString();
  }
  if (value instanceof Date) {
    return value.toISOString();
  }
  if (value instanceof Error) {
    return {
      name: value.name,
      message: value.message,
      stack: value.stack,
    };
  }
  if (Buffer.isBuffer(value)) {
    return value.toString('utf8');
  }
  if (Array.isArray(value)) {
    return value.map((item) => toSerializable(item, seen));
  }
  if (typeof value === 'function') {
    return `[Function ${value.name || 'anonymous'}]`;
  }
  if (typeof value !== 'object') {
    return inspectValue(value);
  }
  if (seen.has(value)) {
    return '[Circular]';
  }
  seen.add(value);

  const prototype = Object.getPrototypeOf(value);
  if (prototype === Object.prototype || prototype === null) {
    const result = {};
    for (const [key, entry] of Object.entries(value)) {
      result[key] = toSerializable(entry, seen);
    }
    return result;
  }

  return inspectValue(value);
}

function buildLaunchRequestBody(defaultSelector, options) {
  const launchOptions = options && typeof options === 'object' ? options : {};
  const body = {};

  for (const key of [
    'code',
    'key',
    'profileId',
    'profileName',
    'keyword',
    'keywords',
    'tag',
    'tags',
    'groupId',
    'matchMode',
    'launchArgs',
    'startUrls',
    'skipDefaultStartUrls',
  ]) {
    if (Object.prototype.hasOwnProperty.call(launchOptions, key)) {
      body[key] = launchOptions[key];
    }
  }

  const selector =
    launchOptions.selector &&
    typeof launchOptions.selector === 'object' &&
    !Array.isArray(launchOptions.selector)
      ? launchOptions.selector
      : defaultSelector;
  if (selector && typeof selector === 'object' && !Array.isArray(selector) && Object.keys(selector).length > 0) {
    body.selector = selector;
  }

  return body;
}

async function loadScriptModule(scriptPath) {
  const resolvedPath = path.resolve(String(scriptPath || ''));
  if (!resolvedPath) {
    throw new Error('scriptPath is required');
  }

  let requiredModule = null;
  let requireError = null;
  try {
    requiredModule = require(resolvedPath);
  } catch (error) {
    requireError = error;
  }

  const imported = async () => {
    const moduleURL = pathToFileURL(resolvedPath).href;
    return await import(`${moduleURL}?t=${Date.now()}`);
  };

  if (requiredModule && typeof requiredModule.run === 'function') {
    return requiredModule;
  }
  if (typeof requiredModule === 'function') {
    return { run: requiredModule };
  }
  if (requiredModule && requiredModule.default && typeof requiredModule.default.run === 'function') {
    return requiredModule.default;
  }

  try {
    const importedModule = await imported();
    if (importedModule && typeof importedModule.run === 'function') {
      return importedModule;
    }
    if (importedModule && typeof importedModule.default === 'function') {
      return { run: importedModule.default };
    }
    if (
      importedModule &&
      importedModule.default &&
      typeof importedModule.default.run === 'function'
    ) {
      return importedModule.default;
    }
  } catch (importError) {
    if (requireError) {
      throw requireError;
    }
    throw importError;
  }

  if (requireError) {
    throw requireError;
  }
  throw new Error('script must export run()');
}

async function runScriptTask(payload, chromium) {
  const scriptModule = await loadScriptModule(payload.scriptPath);
  if (!scriptModule || typeof scriptModule.run !== 'function') {
    throw new Error('script must export run()');
  }

  const logs = [];
  const artifacts = [];
  const connectedBrowsers = new Set();
  const selector = payload.selector && typeof payload.selector === 'object' ? payload.selector : {};
  const params = payload.params && typeof payload.params === 'object' ? payload.params : {};
  const timeout = normalizeTimeout(params.timeoutMs, 30000);
  const startedAt = new Date().toISOString();

  const log = (...entries) => {
    logs.push({
      time: new Date().toISOString(),
      values: entries.map((entry) => toSerializable(entry)),
    });
  };

  const artifact = (name) => {
    const fileName = String(name || '').trim() || `artifact-${Date.now()}`;
    const targetPath = normalizePathUnderRoot(payload.artifactDir, fileName);
    fs.mkdirSync(path.dirname(targetPath), { recursive: true });
    artifacts.push(targetPath);
    return targetPath;
  };

  const launchHeaders = {};
  if (payload.launchAuthHeader && payload.launchAuthValue) {
    launchHeaders[payload.launchAuthHeader] = payload.launchAuthValue;
  }

  const launch = async (options = {}) => {
    const body = buildLaunchRequestBody(selector, options);

    const response = await requestJSON(
      'POST',
      `${String(payload.launchBaseUrl || '').replace(/\/$/, '')}/api/launch`,
      body,
      launchHeaders
    );

    if (!(response.status >= 200 && response.status < 300) || response.body.ok === false) {
      const errorText =
        (response.body && response.body.error && String(response.body.error).trim()) ||
        `launch api returned http ${response.status}`;
      throw new Error(errorText);
    }

    return response.body;
  };

  const connect = async (session = {}) => {
    const endpoints = buildConnectEndpoints(payload, session);
    if (endpoints.length === 0) {
      throw new Error(
        `launch session does not contain a valid cdp endpoint (cdpUrl=${String(
          session && session.cdpUrl ? session.cdpUrl : ''
        )}, debugPort=${String(session && session.debugPort ? session.debugPort : '')})`
      );
    }

    const deadline = Date.now() + timeout;
    let lastError = null;

    while (Date.now() <= deadline) {
      for (const endpoint of endpoints) {
        const remaining = deadline - Date.now();
        if (remaining <= 0) {
          break;
        }

        try {
          const browser = await chromium.connectOverCDP(endpoint, {
            timeout: Math.max(1000, Math.min(remaining, timeout)),
          });
          connectedBrowsers.add(browser);
          const context = browser.contexts()[0] || null;
          const page = context && context.pages().length > 0 ? context.pages()[0] : null;
          return {
            browser,
            context,
            page,
            session: {
              ...session,
              cdpUrl: endpoint,
            },
          };
        } catch (error) {
          lastError = error;
        }
      }

      if (Date.now() >= deadline) {
        break;
      }

      await sleep(Math.min(500, Math.max(100, deadline - Date.now())));
    }

    const lastMessage =
      lastError && lastError.message ? lastError.message : String(lastError || 'unknown error');
    throw new Error(
      `cdp endpoint is not ready after ${timeout} ms (endpoints: ${endpoints.join(', ')}): ${lastMessage}`
    );
  };

  const api = {
    chromium,
    launch,
    connect,
    selector,
    params,
    log,
    artifact,
    artifactsDir: payload.artifactDir || '',
  };

  try {
    const rawResult = await scriptModule.run(api);
    const normalizedResult = toSerializable(rawResult);
    const ok = !(normalizedResult && typeof normalizedResult === 'object' && normalizedResult.ok === false);
    const summary =
      normalizedResult &&
      typeof normalizedResult === 'object' &&
      typeof normalizedResult.summary === 'string'
        ? normalizedResult.summary.trim()
        : ok
          ? '脚本执行完成'
          : '脚本执行失败';
    const error =
      normalizedResult &&
      typeof normalizedResult === 'object' &&
      typeof normalizedResult.error === 'string'
        ? normalizedResult.error.trim()
        : '';

    return {
      ok,
      summary,
      error,
      title:
        normalizedResult &&
        typeof normalizedResult === 'object' &&
        typeof normalizedResult.title === 'string'
          ? normalizedResult.title
          : '',
      url:
        normalizedResult &&
        typeof normalizedResult === 'object' &&
        typeof normalizedResult.url === 'string'
          ? normalizedResult.url
          : '',
      startedAt,
      finishedAt: new Date().toISOString(),
      isolatedPage: false,
      logs,
      artifacts: Array.from(new Set(artifacts)),
      result: normalizedResult,
    };
  } finally {
    await Promise.all(Array.from(connectedBrowsers, (browser) => closeBrowserConnection(browser)));
  }
}

async function main() {
  const payloadPath = process.argv[2];
  if (!payloadPath) {
    throw new Error('payload path is required');
  }

  const payload = JSON.parse(fs.readFileSync(payloadPath, 'utf8'));
  const runtimeDir = path.resolve(String(payload.runtimeDir || ''));
  if (!runtimeDir) {
    throw new Error('runtimeDir is required');
  }

  const { chromium } = require(path.join(runtimeDir, 'node_modules', 'playwright-core'));
  const taskType = String(payload.taskType || 'script').trim() || 'script';
  if (taskType !== 'script') {
    throw new Error(`unsupported automation task type: ${taskType}`);
  }

  const result = await runScriptTask(payload, chromium);
  await writeStream(process.stdout, JSON.stringify(result));
  process.exit(0);
}

main().catch(async (error) => {
  const message = error && error.message ? error.message : String(error);
  try {
    await writeStream(process.stderr, message);
  } finally {
    process.exit(1);
  }
});
