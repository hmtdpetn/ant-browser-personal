import {
  exportAutomationScript,
  importAutomationScript,
  loadAutomationScripts,
  normalizeAutomationScriptRecordPayload,
  normalizeAutomationScriptTargetConfig,
  saveAutomationScripts,
  type AutomationScriptRunInput,
  type AutomationScriptRunRecord,
  type AutomationScriptRecord,
} from "./automationScripts";
import { startBrowserInstanceByCode } from "./api/instances";

const getBindings = async () => {
  try {
    return await import("../../wailsjs/go/main/App");
  } catch {
    return null;
  }
};

function normalizeAutomationScriptRecord(payload: any): AutomationScriptRecord {
  const normalized = normalizeAutomationScriptRecordPayload(payload);
  if (normalized) {
    return normalized;
  }

  return {
    packageFormat: String(payload?.packageFormat || "ant-automation-script"),
    manifestVersion: Number(payload?.manifestVersion) || 1,
    id: String(payload?.id || ""),
    name: String(payload?.name || ""),
    description: String(payload?.description || ""),
    type: payload?.type === "launch-api" ? "launch-api" : "playwright-cdp",
    status:
      payload?.status === "ready" || payload?.status === "disabled"
        ? payload.status
        : "draft",
    entryFile: String(payload?.entryFile || "index.cjs"),
    tags: Array.isArray(payload?.tags)
      ? payload.tags
          .map((item: unknown) => String(item || "").trim())
          .filter(Boolean)
      : [],
    selectorText: String(payload?.selectorText || ""),
    paramsText: String(payload?.paramsText || ""),
    scriptText: String(payload?.scriptText || ""),
    notes: String(payload?.notes || ""),
    targetConfig: normalizeAutomationScriptTargetConfig(payload?.targetConfig),
    source: {
      type: String(payload?.source?.type || ""),
      uri: String(payload?.source?.uri || ""),
      ref: String(payload?.source?.ref || ""),
      path: String(payload?.source?.path || ""),
      importedAt: String(payload?.source?.importedAt || ""),
    },
    createdAt: String(payload?.createdAt || ""),
    updatedAt: String(payload?.updatedAt || ""),
  };
}

function sortScripts(
  items: AutomationScriptRecord[],
): AutomationScriptRecord[] {
  return [...items].sort(
    (left, right) =>
      new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime(),
  );
}

function normalizeAutomationScriptRunRecord(
  payload: any,
): AutomationScriptRunRecord {
  return {
    id: String(payload?.id || ""),
    scriptId: String(payload?.scriptId || ""),
    scriptName: String(payload?.scriptName || ""),
    scriptType: String(payload?.scriptType || ""),
    status:
      payload?.status === "success" || payload?.status === "running"
        ? payload.status
        : "failed",
    summary: String(payload?.summary || ""),
    error: String(payload?.error || ""),
    resultText: String(payload?.resultText || ""),
    startedAt: String(payload?.startedAt || ""),
    finishedAt: String(payload?.finishedAt || ""),
    durationMs: Number(payload?.durationMs) || 0,
  };
}

export interface AutomationScriptExportResult {
  cancelled: boolean;
  format: string;
  message: string;
  path: string;
  fileCount: number;
}

function normalizeAutomationScriptRunInput(
  input: string | AutomationScriptRunInput,
): AutomationScriptRunInput {
  if (typeof input === "string") {
    return {
      scriptId: input,
      selectorText: "",
      paramsText: "",
      useScriptSelector: true,
      useScriptParams: true,
      timeoutMs: 0,
      launchCode: "",
      startByCodeBeforeRun: false,
    };
  }

  return {
    scriptId: String(input?.scriptId || ""),
    selectorText: String(input?.selectorText || ""),
    paramsText: String(input?.paramsText || ""),
    useScriptSelector: input?.useScriptSelector !== false,
    useScriptParams: input?.useScriptParams !== false,
    timeoutMs: Number.isFinite(Number(input?.timeoutMs))
      ? Math.round(Number(input?.timeoutMs))
      : 0,
    launchCode: String(input?.launchCode || "")
      .trim()
      .toUpperCase(),
    startByCodeBeforeRun: input?.startByCodeBeforeRun === true,
  };
}

export async function fetchAutomationScripts(): Promise<
  AutomationScriptRecord[]
> {
  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptList) {
    const raw = (await bindings.AutomationScriptList()) || [];
    return sortScripts(
      Array.isArray(raw) ? raw.map(normalizeAutomationScriptRecord) : [],
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptList === "function") {
    const raw = (await goApp.AutomationScriptList()) || [];
    return sortScripts(
      Array.isArray(raw) ? raw.map(normalizeAutomationScriptRecord) : [],
    );
  }

  return loadAutomationScripts();
}

export async function saveAutomationScript(
  script: AutomationScriptRecord,
): Promise<AutomationScriptRecord> {
  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptSave) {
    return normalizeAutomationScriptRecord(
      await bindings.AutomationScriptSave(script),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptSave === "function") {
    return normalizeAutomationScriptRecord(
      await goApp.AutomationScriptSave(script),
    );
  }

  const current = loadAutomationScripts();
  const next = current.some((item) => item.id === script.id)
    ? current.map((item) => (item.id === script.id ? script : item))
    : [script, ...current];
  saveAutomationScripts(sortScripts(next));
  return script;
}

export async function deleteAutomationScript(scriptId: string): Promise<void> {
  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptDelete) {
    await bindings.AutomationScriptDelete(scriptId);
    return;
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptDelete === "function") {
    await goApp.AutomationScriptDelete(scriptId);
    return;
  }

  saveAutomationScripts(
    loadAutomationScripts().filter((item) => item.id !== scriptId),
  );
}

export async function importAutomationScriptFromLocalFile(): Promise<AutomationScriptRecord> {
  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptImportLocalFile) {
    return normalizeAutomationScriptRecord(
      await bindings.AutomationScriptImportLocalFile(),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptImportLocalFile === "function") {
    return normalizeAutomationScriptRecord(
      await goApp.AutomationScriptImportLocalFile(),
    );
  }

  throw new Error("当前环境不支持本地文件导入");
}

export async function importAutomationScriptFromText(
  text: string,
): Promise<AutomationScriptRecord> {
  const normalizedText = String(text || "").trim();
  if (!normalizedText) {
    throw new Error("导入内容不能为空");
  }

  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptImportText) {
    return normalizeAutomationScriptRecord(
      await bindings.AutomationScriptImportText(normalizedText),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptImportText === "function") {
    return normalizeAutomationScriptRecord(
      await goApp.AutomationScriptImportText(normalizedText),
    );
  }

  return importAutomationScript(normalizedText);
}

export async function importAutomationScriptFromLocalDirectory(): Promise<AutomationScriptRecord> {
  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptImportLocalDirectory) {
    return normalizeAutomationScriptRecord(
      await bindings.AutomationScriptImportLocalDirectory(),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptImportLocalDirectory === "function") {
    return normalizeAutomationScriptRecord(
      await goApp.AutomationScriptImportLocalDirectory(),
    );
  }

  throw new Error("当前环境不支持本地目录导入");
}

export async function importAutomationScriptFromRemote(url: string): Promise<AutomationScriptRecord> {
  const normalizedURL = String(url || "").trim();
  if (!normalizedURL) {
    throw new Error("远程脚本地址不能为空");
  }

  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptImportRemote) {
    return normalizeAutomationScriptRecord(
      await bindings.AutomationScriptImportRemote(normalizedURL),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptImportRemote === "function") {
    return normalizeAutomationScriptRecord(
      await goApp.AutomationScriptImportRemote(normalizedURL),
    );
  }

  throw new Error("当前环境不支持远程脚本导入");
}

export async function importAutomationScriptFromGit(
  repoURL: string,
  ref = "",
  scriptPath = "",
): Promise<AutomationScriptRecord> {
  const normalizedRepoURL = String(repoURL || "").trim();
  if (!normalizedRepoURL) {
    throw new Error("Git 仓库地址不能为空");
  }

  const normalizedRef = String(ref || "").trim();
  const normalizedScriptPath = String(scriptPath || "").trim();

  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptImportGit) {
    return normalizeAutomationScriptRecord(
      await bindings.AutomationScriptImportGit(
        normalizedRepoURL,
        normalizedRef,
        normalizedScriptPath,
      ),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptImportGit === "function") {
    return normalizeAutomationScriptRecord(
      await goApp.AutomationScriptImportGit(
        normalizedRepoURL,
        normalizedRef,
        normalizedScriptPath,
      ),
    );
  }

  throw new Error("当前环境不支持 Git 脚本导入");
}

export async function refreshAutomationScript(
  scriptId: string,
): Promise<AutomationScriptRecord> {
  const normalizedScriptId = String(scriptId || "").trim();
  if (!normalizedScriptId) {
    throw new Error("脚本 ID 不能为空");
  }

  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptRefresh) {
    return normalizeAutomationScriptRecord(
      await bindings.AutomationScriptRefresh(normalizedScriptId),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptRefresh === "function") {
    return normalizeAutomationScriptRecord(
      await goApp.AutomationScriptRefresh(normalizedScriptId),
    );
  }

  throw new Error("当前环境不支持按来源重新导入");
}

function normalizeAutomationScriptExportResult(
  payload: any,
): AutomationScriptExportResult {
  return {
    cancelled: payload?.cancelled === true,
    format: String(payload?.format || ""),
    message: String(payload?.message || ""),
    path: String(payload?.path || ""),
    fileCount: Number(payload?.fileCount) || 0,
  };
}

function buildAutomationTemplateFallbackFilename(script: AutomationScriptRecord): string {
  const normalizedName = String(script.name || "")
    .trim()
    .replace(/[\\/:*?"<>|]+/g, "-")
    .replace(/\s+/g, "-")
    .replace(/^-+|-+$/g, "");

  return `${normalizedName || "automation-script"}-template.json`;
}

function downloadAutomationTemplate(
  filename: string,
  content: string,
): AutomationScriptExportResult {
  const blob = new Blob([content], { type: "application/json;charset=utf-8" });
  const url = URL.createObjectURL(blob);

  try {
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = filename;
    anchor.click();
  } finally {
    URL.revokeObjectURL(url);
  }

  return {
    cancelled: false,
    format: "json",
    message: "模板已导出",
    path: filename,
    fileCount: 1,
  };
}

export async function exportAutomationScriptTemplate(
  scriptId: string,
  fallbackScript?: AutomationScriptRecord,
): Promise<AutomationScriptExportResult> {
  const normalizedScriptId = String(scriptId || "").trim();
  if (!normalizedScriptId) {
    throw new Error("脚本 ID 不能为空");
  }

  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptExport) {
    return normalizeAutomationScriptExportResult(
      await bindings.AutomationScriptExport(normalizedScriptId),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptExport === "function") {
    return normalizeAutomationScriptExportResult(
      await goApp.AutomationScriptExport(normalizedScriptId),
    );
  }

  if (fallbackScript && typeof document !== "undefined") {
    return downloadAutomationTemplate(
      buildAutomationTemplateFallbackFilename(fallbackScript),
      exportAutomationScript(fallbackScript),
    );
  }

  throw new Error("当前环境不支持脚本模板导出");
}

export async function exportAutomationScriptZip(
  scriptId: string,
): Promise<AutomationScriptExportResult> {
  const normalizedScriptId = String(scriptId || "").trim();
  if (!normalizedScriptId) {
    throw new Error("脚本 ID 不能为空");
  }

  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptExportZip) {
    return normalizeAutomationScriptExportResult(
      await bindings.AutomationScriptExportZip(normalizedScriptId),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptExportZip === "function") {
    return normalizeAutomationScriptExportResult(
      await goApp.AutomationScriptExportZip(normalizedScriptId),
    );
  }

  throw new Error("当前环境不支持 ZIP 脚本包导出");
}

export async function exportAutomationScriptDirectory(
  scriptId: string,
): Promise<AutomationScriptExportResult> {
  const normalizedScriptId = String(scriptId || "").trim();
  if (!normalizedScriptId) {
    throw new Error("脚本 ID 不能为空");
  }

  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptExportDirectory) {
    return normalizeAutomationScriptExportResult(
      await bindings.AutomationScriptExportDirectory(normalizedScriptId),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptExportDirectory === "function") {
    return normalizeAutomationScriptExportResult(
      await goApp.AutomationScriptExportDirectory(normalizedScriptId),
    );
  }

  throw new Error("当前环境不支持目录脚本包导出");
}

export async function runAutomationScript(
  input: string | AutomationScriptRunInput,
): Promise<AutomationScriptRunRecord> {
  const request = normalizeAutomationScriptRunInput(input);
  const { launchCode, startByCodeBeforeRun, ...bindingRequest } = request;

  if (startByCodeBeforeRun && launchCode) {
    const startedProfile = await startBrowserInstanceByCode(launchCode);
    if (!startedProfile) {
      throw new Error(`通过 Launch Code 启动实例失败: ${launchCode}`);
    }
  }

  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptRunWithOptions) {
    return normalizeAutomationScriptRunRecord(
      await bindings.AutomationScriptRunWithOptions(bindingRequest),
    );
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptRunWithOptions === "function") {
    return normalizeAutomationScriptRunRecord(
      await goApp.AutomationScriptRunWithOptions(bindingRequest),
    );
  }

  if (
    bindings?.AutomationScriptRun &&
    bindingRequest.useScriptSelector &&
    bindingRequest.useScriptParams
  ) {
    return normalizeAutomationScriptRunRecord(
      await bindings.AutomationScriptRun(bindingRequest.scriptId),
    );
  }

  if (
    typeof goApp?.AutomationScriptRun === "function" &&
    bindingRequest.useScriptSelector &&
    bindingRequest.useScriptParams
  ) {
    return normalizeAutomationScriptRunRecord(
      await goApp.AutomationScriptRun(bindingRequest.scriptId),
    );
  }

  const now = new Date().toISOString();
  return {
    id: `mock-run-${Date.now()}`,
    scriptId: bindingRequest.scriptId,
    scriptName: "",
    scriptType: "",
    status: "failed",
    summary: "当前环境未接入自动化脚本执行",
    error: "AutomationScriptRun binding is unavailable",
    resultText: "",
    startedAt: now,
    finishedAt: now,
    durationMs: 0,
  };
}

export async function fetchAutomationScriptRuns(
  limit = 20,
): Promise<AutomationScriptRunRecord[]> {
  const bindings: any = await getBindings();
  if (bindings?.AutomationScriptRunList) {
    const raw = (await bindings.AutomationScriptRunList(limit)) || [];
    return Array.isArray(raw)
      ? raw.map(normalizeAutomationScriptRunRecord)
      : [];
  }

  const goApp = (window as any).go?.main?.App;
  if (typeof goApp?.AutomationScriptRunList === "function") {
    const raw = (await goApp.AutomationScriptRunList(limit)) || [];
    return Array.isArray(raw)
      ? raw.map(normalizeAutomationScriptRunRecord)
      : [];
  }

  return [];
}
