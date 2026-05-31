import { useEffect, useState } from "react";
import { Copy, FileText, FolderOpen, Play } from "lucide-react";
import { useNavigate } from "react-router-dom";
import {
  Badge,
  Button,
  FormItem,
  Input,
  Modal,
  Select,
  Textarea,
  toast,
} from "../../../shared/components";
import {
  copyBrowserProfile,
  fetchBrowserProfiles,
  openCorePath,
} from "../api";
import { runAutomationScript } from "../automationScriptApi";
import {
  DUAL_INSTANCE_RUNTIME_SCRIPT_ID,
  describeAutomationScriptTargetConfig,
  getAutomationScriptTypeLabel,
  type AutomationScriptRecord,
  type AutomationScriptRunRecord,
} from "../automationScripts";
import {
  type AutomationDemoSession,
} from "../demoSession";
import { useAutomationDemoSession } from "../hooks/useAutomationDemoSession";
import type { BrowserProfile } from "../types";

type DemoPreparationMode = "select" | "create";

type SelectableProfile = BrowserProfile & {
  launchCode: string;
};

interface DemoCreateDraft {
  profileName: string;
  templateProfileId: string;
}

interface ResultOutputEntry {
  key: string;
  label: string;
  path: string;
}

interface AutomationScriptRunModalProps {
  open: boolean;
  script: AutomationScriptRecord | null;
  dirty?: boolean;
  onClose: () => void;
}

const DEFAULT_DEMO_CREATE_DRAFT: DemoCreateDraft = {
  profileName: "",
  templateProfileId: "",
};

function validateJsonObjectText(
  text: string,
  label: string,
  required: boolean,
): string {
  const normalized = text.trim();
  if (!normalized) {
    return required ? `${label}不能为空` : "";
  }

  try {
    const parsed = JSON.parse(normalized);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return `${label}必须是 JSON 对象`;
    }
    return "";
  } catch {
    return `${label}不是合法 JSON`;
  }
}

function formatDateTime(value?: string): string {
  if (!value) {
    return "-";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return date.toLocaleString("zh-CN", { hour12: false });
}

function formatDuration(durationMs?: number): string {
  if (!durationMs || durationMs <= 0) {
    return "-";
  }
  if (durationMs < 1000) {
    return `${durationMs} ms`;
  }
  return `${(durationMs / 1000).toFixed(2)} s`;
}

function parseRunResultOutputs(resultText?: string): ResultOutputEntry[] {
  const normalized = String(resultText || "").trim();
  if (!normalized) {
    return [];
  }

  try {
    const parsed = JSON.parse(normalized);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return [];
    }

    const seen = new Set<string>();
    const outputs: ResultOutputEntry[] = [];

    const addOutput = (key: string, value: string) => {
      const path = value.trim();
      if (!path || seen.has(path)) {
        return;
      }
      seen.add(path);
      outputs.push({
        key,
        label: formatRunResultOutputLabel(key),
        path,
      });
    };

    const collectOutputs = (value: unknown, keyHint = "") => {
      if (!value) {
        return;
      }
      if (typeof value === "string") {
        if (/path$/i.test(keyHint)) {
          addOutput(keyHint, value);
        }
        return;
      }
      if (Array.isArray(value)) {
        if (keyHint === "artifacts") {
          value.forEach((item) => {
            if (typeof item === "string") {
              addOutput(keyHint, item);
            }
          });
          return;
        }
        value.forEach((item) => collectOutputs(item, keyHint));
        return;
      }
      if (typeof value !== "object") {
        return;
      }

      for (const [nestedKey, nestedValue] of Object.entries(
        value as Record<string, unknown>,
      )) {
        collectOutputs(nestedValue, nestedKey);
      }
    };

    collectOutputs(parsed);
    return outputs;
  } catch {
    return [];
  }
}

function formatRunResultOutputLabel(key: string): string {
  switch (key) {
    case "outputPath":
      return "输出文件";
    case "screenshotPath":
      return "截图文件";
    case "artifacts":
      return "导出文件";
    default:
      return key;
  }
}

function formatRunResultOutputName(path: string): string {
  const segments = path.split(/[\\/]/).filter(Boolean);
  return segments[segments.length - 1] || path;
}

function formatRunResultText(resultText?: string): string {
  const normalized = String(resultText || "").trim();
  if (!normalized) {
    return "";
  }

  try {
    return JSON.stringify(JSON.parse(normalized), null, 2);
  } catch {
    return resultText || "";
  }
}

async function copyToClipboard(text: string, successMessage: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.success(successMessage);
  } catch {
    toast.error("复制失败");
  }
}

function buildDemoSelectorText(launchCode: string) {
  return JSON.stringify(
    {
      code: launchCode,
    },
    null,
    2,
  );
}

function normalizeLaunchCode(value?: string): string {
  return String(value || "")
    .trim()
    .toUpperCase();
}

function isPlaceholderSelectorText(text: string): boolean {
  const normalized = text.trim();
  if (!normalized) {
    return true;
  }

  try {
    const parsed = JSON.parse(normalized);
    const code =
      parsed && typeof parsed === "object" && !Array.isArray(parsed)
        ? String((parsed as Record<string, unknown>).code || "")
            .trim()
            .toUpperCase()
        : "";
    return !code || code === "BUYER_001" || code === "DEMO_ABC123";
  } catch {
    return false;
  }
}

function resolveInitialSelectorText(
  script: AutomationScriptRecord,
  demoSession: AutomationDemoSession,
): string {
  if (script.targetConfig.mode !== "manual") {
    return "";
  }
  const currentSelectorText = String(script.selectorText || "");
  if (
    script.type === "playwright-cdp" &&
    isPlaceholderSelectorText(currentSelectorText) &&
    demoSession.launchCode
  ) {
    return buildDemoSelectorText(demoSession.launchCode);
  }
  return currentSelectorText;
}

function resolveRunnableSelectorText(
  script: AutomationScriptRecord,
  currentSelectorText: string,
  demoSession: AutomationDemoSession,
): string {
  if (script.targetConfig.mode !== "manual") {
    return currentSelectorText;
  }
  if (
    script.type === "playwright-cdp" &&
    isPlaceholderSelectorText(currentSelectorText) &&
    demoSession.launchCode
  ) {
    return buildDemoSelectorText(demoSession.launchCode);
  }
  return currentSelectorText;
}

function resolveSelectorLaunchCode(text: string): string {
  const normalized = text.trim();
  if (!normalized) {
    return "";
  }

  try {
    const parsed = JSON.parse(normalized);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      return "";
    }

    return String((parsed as Record<string, unknown>).code || "")
      .trim()
      .toUpperCase();
  } catch {
    return "";
  }
}

function filterSelectableProfiles(profiles: BrowserProfile[]): SelectableProfile[] {
  return profiles
    .flatMap((profile) => {
      const launchCode = normalizeLaunchCode(profile.launchCode);
      if (!launchCode) {
        return [];
      }
      return [
        {
          ...profile,
          launchCode,
        },
      ];
    })
    .sort((left, right) => {
      if (left.running !== right.running) {
        return left.running ? -1 : 1;
      }
      return left.profileName.localeCompare(right.profileName, "zh-CN");
    });
}

function resolvePreferredProfileId(
  profiles: SelectableProfile[],
  preferredProfileId: string,
  preferredLaunchCode: string,
): string {
  const normalizedProfileId = String(preferredProfileId || "").trim();
  const normalizedCode = normalizeLaunchCode(preferredLaunchCode);
  if (!normalizedProfileId && !normalizedCode) {
    return "";
  }

  if (normalizedProfileId) {
    const matchedByID = profiles.find(
      (profile) => profile.profileId === normalizedProfileId,
    );
    if (matchedByID) {
      return matchedByID.profileId;
    }
  }

  const matchedByCode = profiles.find(
    (profile) => normalizeLaunchCode(profile.launchCode) === normalizedCode,
  );
  if (matchedByCode) {
    return matchedByCode.profileId;
  }

  return "";
}

function buildSelectableProfileOptions(profiles: SelectableProfile[]) {
  return profiles.map((profile) => ({
    value: profile.profileId,
    label: `${profile.launchCode} · ${profile.profileName} · ${formatSelectableProfileStatus(profile)}`,
  }));
}

function formatSelectableProfileStatus(profile: SelectableProfile): string {
  if (profile.running && profile.debugReady && profile.debugPort > 0) {
    return "可连接";
  }
  if (profile.running) {
    return "启动中";
  }
  return "未启动，执行时自动启动";
}

function sortTemplateProfiles(profiles: BrowserProfile[]) {
  return [...profiles].sort((left, right) =>
    left.profileName.localeCompare(right.profileName, "zh-CN"),
  );
}

function buildTemplateProfileOptions(profiles: BrowserProfile[]) {
  return profiles.map((profile) => ({
    value: profile.profileId,
    label: [profile.launchCode || "", profile.profileName || profile.profileId]
      .filter(Boolean)
      .join(" · "),
  }));
}

export function AutomationScriptRunModal({
  open,
  script,
  dirty = false,
  onClose,
}: AutomationScriptRunModalProps) {
  const navigate = useNavigate();
  const [selectorText, setSelectorText] = useState("");
  const [paramsText, setParamsText] = useState("");
  const [running, setRunning] = useState(false);
  const [demoBusy, setDemoBusy] = useState(false);
  const [lastRun, setLastRun] = useState<AutomationScriptRunRecord | null>(
    null,
  );
  const [demoMode, setDemoMode] = useState<DemoPreparationMode>("select");
  const [availableProfiles, setAvailableProfiles] = useState<SelectableProfile[]>(
    [],
  );
  const [templateProfiles, setTemplateProfiles] = useState<BrowserProfile[]>([]);
  const [profilesLoading, setProfilesLoading] = useState(false);
  const [selectedProfileId, setSelectedProfileId] = useState("");
  const [createDraft, setCreateDraft] = useState<DemoCreateDraft>(
    DEFAULT_DEMO_CREATE_DRAFT,
  );
  const {
    demoSession,
    setDemoSession,
    reloadDemoSession,
  } = useAutomationDemoSession({ enabled: open });

  const selectedProfile =
    availableProfiles.find((profile) => profile.profileId === selectedProfileId) ||
    null;
  const selectedTemplateProfile =
    templateProfiles.find(
      (profile) => profile.profileId === createDraft.templateProfileId,
    ) || null;
  const isDualInstanceRuntimeScript =
    script?.id === DUAL_INSTANCE_RUNTIME_SCRIPT_ID;
  const usesStoredTargetConfig =
    !!script && script.targetConfig.mode !== "manual";
  const showsSelectorInput =
    !!script && !usesStoredTargetConfig && !isDualInstanceRuntimeScript;
  const paramsLabel = isDualInstanceRuntimeScript ? "启动配置" : "运行参数";
  const paramsFieldLabel = isDualInstanceRuntimeScript
    ? "浏览器列表 / 启动配置 JSON"
    : "运行参数 JSON";
  const paramsPlaceholder = isDualInstanceRuntimeScript
    ? `{
  "browsers": [
    { "code": "BUYER_001", "skipDefaultStartUrls": true },
    { "code": "BUYER_002", "skipDefaultStartUrls": true }
  ],
  "timeoutMs": 45000
}`
    : '{"startUrls":["https://example.com"]}';

  const syncDemoSessionFromProfile = (
    profile: SelectableProfile,
    actionLabel: string,
  ) => {
    setDemoSession((current) => ({
      ...current,
      profileId: profile.profileId,
      profileName: profile.profileName,
      launchCode: profile.launchCode,
      cdpUrl:
        profile.running && profile.debugReady && profile.debugPort > 0
          ? `http://127.0.0.1:${profile.debugPort}`
          : "",
      debugPort:
        profile.running && profile.debugReady && profile.debugPort > 0
          ? profile.debugPort
          : 0,
      lastAction: actionLabel,
    }));
  };

  const refreshSelectableProfiles = async (
    preferredProfileId = "",
    preferredLaunchCode = "",
    showError = false,
  ) => {
    setProfilesLoading(true);
    try {
      const allProfiles = await fetchBrowserProfiles();
      const profiles = filterSelectableProfiles(allProfiles);
      const nextSelectedProfileId =
        resolvePreferredProfileId(
          profiles,
          preferredProfileId,
          preferredLaunchCode,
        ) ||
        (selectedProfileId &&
        profiles.some((profile) => profile.profileId === selectedProfileId)
          ? selectedProfileId
          : profiles[0]?.profileId || "");
      const nextSelectedProfile =
        profiles.find((profile) => profile.profileId === nextSelectedProfileId) ||
        null;

      setAvailableProfiles(profiles);
      setTemplateProfiles(sortTemplateProfiles(allProfiles));
      setSelectedProfileId(nextSelectedProfileId);
      if (demoMode === "select" && nextSelectedProfile) {
        const nextSelectorText = buildDemoSelectorText(
          nextSelectedProfile.launchCode,
        );
        if (
          resolveSelectorLaunchCode(selectorText) !==
          nextSelectedProfile.launchCode
        ) {
          setSelectorText(nextSelectorText);
        }
        syncDemoSessionFromProfile(nextSelectedProfile, "选择已有实例");
      }
      setCreateDraft((current) => {
        if (
          current.templateProfileId &&
          allProfiles.some((profile) => profile.profileId === current.templateProfileId)
        ) {
          return current;
        }
        return {
          ...current,
          templateProfileId: allProfiles[0]?.profileId || "",
        };
      });
      if (!profiles.length) {
        setDemoMode("create");
      }
    } catch (error: unknown) {
      if (showError) {
        const message =
          error instanceof Error ? error.message : "实例列表刷新失败";
        toast.error(message);
      }
    } finally {
      setProfilesLoading(false);
    }
  };

  useEffect(() => {
    if (!open || !script) {
      return;
    }

    const nextDemoSession = reloadDemoSession();
    const nextSelectorText = resolveInitialSelectorText(script, nextDemoSession);
    setSelectorText(nextSelectorText);
    setParamsText(script.paramsText || "");
    setLastRun(null);
    setCreateDraft(DEFAULT_DEMO_CREATE_DRAFT);
    setDemoMode(
      nextDemoSession.launchCode ||
        resolveSelectorLaunchCode(nextSelectorText)
        ? "select"
        : "create",
    );
  }, [open, script]);

  useEffect(() => {
    if (!open || !script || script.type !== "playwright-cdp") {
      setAvailableProfiles([]);
      setSelectedProfileId("");
      return;
    }
    if (usesStoredTargetConfig) {
      setAvailableProfiles([]);
      setSelectedProfileId("");
      return;
    }

    const nextDemoSession = reloadDemoSession();
    const nextSelectorText = resolveInitialSelectorText(script, nextDemoSession);
    void refreshSelectableProfiles(
      nextDemoSession.profileId,
      resolveSelectorLaunchCode(nextSelectorText) || nextDemoSession.launchCode,
      false,
    );
  }, [open, script, usesStoredTargetConfig]);

  useEffect(() => {
    if (!open || !script || script.type !== "playwright-cdp") {
      return;
    }
    if (usesStoredTargetConfig) {
      return;
    }
    if (demoMode !== "select") {
      return;
    }

    void refreshSelectableProfiles("", demoSession.launchCode, false);
  }, [demoMode, demoSession.launchCode, open, script, usesStoredTargetConfig]);

  const handleClose = () => {
    if (running || demoBusy) {
      return;
    }
    onClose();
  };

  const executeRun = async (nextSelectorText: string, nextParamsText: string) => {
    if (!script) {
      return;
    }

    const runnableSelectorText = usesStoredTargetConfig ? "" : nextSelectorText;
    const launchCode =
      script.type === "playwright-cdp" && !usesStoredTargetConfig
        ? resolveSelectorLaunchCode(runnableSelectorText)
        : "";

    setRunning(true);
    try {
      const run = await runAutomationScript({
        scriptId: script.id,
        selectorText: runnableSelectorText,
        paramsText: nextParamsText,
        useScriptSelector: usesStoredTargetConfig,
        useScriptParams: false,
        launchCode,
        startByCodeBeforeRun:
          script.type === "playwright-cdp" &&
          !usesStoredTargetConfig &&
          !!launchCode,
      });
      setLastRun(run);
      if (run.status === "success") {
        toast.success(run.summary || "脚本执行完成");
      } else {
        toast.error(run.error || run.summary || "脚本执行失败");
      }
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : "脚本执行失败";
      toast.error(message);
    } finally {
      setRunning(false);
    }
  };

  const handleSelectedProfileChange = (profileId: string) => {
    setSelectedProfileId(profileId);
    const profile =
      availableProfiles.find((item) => item.profileId === profileId) || null;
    if (!profile) {
      return;
    }

    setSelectorText(buildDemoSelectorText(profile.launchCode));
    syncDemoSessionFromProfile(profile, "选择已有实例");
  };

  const handleCreateProfileAndRun = async () => {
    const paramsError = validateJsonObjectText(paramsText, paramsLabel, false);
    if (paramsError) {
      toast.warning(paramsError);
      return;
    }

    const profileName = createDraft.profileName.trim();
    if (!profileName) {
      toast.warning("先输入实例名称");
      return;
    }
    if (!selectedTemplateProfile) {
      toast.warning("先选择一个模板");
      return;
    }

    setDemoBusy(true);
    try {
      const created = await copyBrowserProfile(
        selectedTemplateProfile.profileId,
        profileName,
      );
      if (!created) {
        throw new Error("实例创建失败");
      }

      const launchCode = normalizeLaunchCode(created.launchCode);
      if (!launchCode) {
        throw new Error("新实例未生成启动 code");
      }

      setDemoSession((current) => ({
        ...current,
        profileId: created.profileId,
        profileName: created.profileName,
        launchCode,
        cdpUrl: "",
        debugPort: 0,
        lastAction: "按模板创建实例",
      }));

      const nextSelectorText = buildDemoSelectorText(launchCode);
      setSelectorText(nextSelectorText);
      setDemoMode("select");
      setCreateDraft((current) => ({
        ...current,
        profileName: "",
      }));
      await refreshSelectableProfiles(created.profileId, launchCode, false);
      setDemoBusy(false);
      toast.success("实例已创建，开始执行脚本");
      await executeRun(nextSelectorText, paramsText);
      return;
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : "实例创建或启动失败";
      toast.error(message);
    } finally {
      setDemoBusy(false);
    }
  };

  const handleRun = async () => {
    if (!script) {
      return;
    }

    let nextSelectorText = usesStoredTargetConfig
      ? ""
      : resolveRunnableSelectorText(
          script,
          selectorText,
          demoSession,
        );
    if (
      script.type === "playwright-cdp" &&
      !usesStoredTargetConfig &&
      demoMode === "select" &&
      selectedProfile
    ) {
      nextSelectorText = buildDemoSelectorText(selectedProfile.launchCode);
    }
    const selectorError = usesStoredTargetConfig
      ? ""
      : validateJsonObjectText(
          nextSelectorText,
          "目标选择器",
          script.type === "launch-api" &&
            !usesStoredTargetConfig &&
            !isDualInstanceRuntimeScript,
        );
    if (selectorError) {
      toast.warning(selectorError);
      return;
    }

    const paramsError = validateJsonObjectText(paramsText, paramsLabel, false);
    if (paramsError) {
      toast.warning(paramsError);
      return;
    }

    if (
      script.type === "playwright-cdp" &&
      !usesStoredTargetConfig &&
      isPlaceholderSelectorText(nextSelectorText)
    ) {
      if (demoMode === "select" && selectedProfile) {
        nextSelectorText = buildDemoSelectorText(selectedProfile.launchCode);
        setSelectorText(nextSelectorText);
        syncDemoSessionFromProfile(selectedProfile, "选择已有实例");
        toast.success("已自动回填所选实例 selector");
      } else {
        toast.warning(
          demoMode === "create"
            ? "先创建一个实例，或填入可用 code"
            : "先选择一个已有实例，或填入可用 code",
        );
        return;
      }
    }

    if (nextSelectorText !== selectorText) {
      setSelectorText(nextSelectorText);
    }
    if (
      script.type === "playwright-cdp" &&
      !usesStoredTargetConfig &&
      demoMode === "select" &&
      selectedProfile
    ) {
      syncDemoSessionFromProfile(selectedProfile, "选择已有实例");
    }

    await executeRun(nextSelectorText, paramsText);
  };

  const handlePrimaryAction = async () => {
    if (!script) {
      return;
    }

    if (
      script.type === "playwright-cdp" &&
      !usesStoredTargetConfig &&
      demoMode === "create"
    ) {
      await handleCreateProfileAndRun();
      return;
    }

    await handleRun();
  };

  const handleOpenScriptDetail = () => {
    if (!script || running || demoBusy) {
      return;
    }
    onClose();
    navigate(`/browser/automation/${script.id}`);
  };

  const handleOpenOutputPath = async (path: string) => {
    try {
      await openCorePath(path);
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : "打开目录失败";
      toast.error(message);
    }
  };

  if (!script) {
    return null;
  }

  const launchApiExecutable = script.status !== "disabled";
  const showDemoProfilePicker =
    script.type === "playwright-cdp" && !usesStoredTargetConfig;
  const selectableProfileOptions = buildSelectableProfileOptions(availableProfiles);
  const templateProfileOptions = buildTemplateProfileOptions(templateProfiles);
  const resultOutputs = parseRunResultOutputs(lastRun?.resultText);
  const formattedResultText = formatRunResultText(lastRun?.resultText);

  return (
    <Modal
      open={open}
      onClose={handleClose}
      title="执行脚本"
      width="880px"
      footer={
        <>
          <Button
            variant="secondary"
            onClick={handleClose}
            disabled={running || demoBusy}
          >
            关闭
          </Button>
          <Button
            onClick={() => void handlePrimaryAction()}
            loading={running}
            disabled={!launchApiExecutable || demoBusy}
          >
            <Play className="h-4 w-4" />
            立即执行
          </Button>
        </>
      }
    >
      <div className="space-y-5">
        <div className="rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] px-4 py-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <div className="flex flex-wrap items-center gap-2">
                <Badge
                  variant={script.type === "launch-api" ? "info" : "default"}
                  size="sm"
                >
                  {getAutomationScriptTypeLabel(script.type)}
                </Badge>
                <Badge
                  variant={
                    script.status === "ready"
                      ? "success"
                      : script.status === "disabled"
                        ? "default"
                        : "warning"
                  }
                  size="sm"
                  dot
                >
                  {script.status === "ready"
                    ? "可用"
                    : script.status === "disabled"
                      ? "停用"
                      : "草稿"}
                </Badge>
              </div>
              <div className="mt-3 text-sm text-[var(--color-text-primary)]">
                {script.name}
              </div>
              <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                最近更新 {formatDateTime(script.updatedAt)}
              </div>
            </div>
            <Button
              variant="secondary"
              size="sm"
              onClick={handleOpenScriptDetail}
              disabled={running || demoBusy}
            >
              <FileText className="h-4 w-4" />
              查看脚本详情
            </Button>
          </div>
        </div>

        {dirty && (
          <div className="rounded-xl border border-[var(--color-warning)]/30 bg-[var(--color-warning)]/10 px-4 py-3 text-sm text-[var(--color-text-secondary)]">
            {isDualInstanceRuntimeScript
              ? "当前详情页还有未保存修改。本次执行只使用弹窗里的启动配置，不会自动保存页面内容。"
              : "当前详情页还有未保存修改。本次执行只使用弹窗里的 selector / params，不会自动保存页面内容。"}
          </div>
        )}

        {usesStoredTargetConfig && (
          <div className="rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-4 py-3 text-sm text-[var(--color-text-secondary)]">
            <div>
              {describeAutomationScriptTargetConfig(script.targetConfig)}
            </div>
            <div className="mt-2 text-xs text-[var(--color-text-muted)]">
              本次执行会直接沿用脚本里保存的目标策略，弹窗中不会覆盖 selector。
            </div>
          </div>
        )}

        {showDemoProfilePicker && (
          <div className="rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-4 py-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="text-sm font-medium text-[var(--color-text-primary)]">
                实例
              </div>
              <div className="inline-flex rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-secondary)] p-1">
                <button
                  type="button"
                  className={`rounded-md px-3 py-1.5 text-xs transition-colors ${
                    demoMode === "select"
                      ? "bg-[var(--color-bg-surface)] text-[var(--color-text-primary)] shadow-sm"
                      : "text-[var(--color-text-muted)]"
                  }`}
                  onClick={() => setDemoMode("select")}
                  disabled={running || demoBusy}
                >
                  选择已有
                </button>
                <button
                  type="button"
                  className={`rounded-md px-3 py-1.5 text-xs transition-colors ${
                    demoMode === "create"
                      ? "bg-[var(--color-bg-surface)] text-[var(--color-text-primary)] shadow-sm"
                      : "text-[var(--color-text-muted)]"
                  }`}
                  onClick={() => setDemoMode("create")}
                  disabled={running || demoBusy}
                >
                  创建新的
                </button>
              </div>
            </div>

            {demoMode === "select" ? (
              <div className="mt-4">
                <Select
                  value={selectedProfileId}
                  onChange={(event) =>
                    handleSelectedProfileChange(event.target.value)
                  }
                  options={
                    selectableProfileOptions.length > 0
                      ? selectableProfileOptions
                      : [
                          {
                            value: "",
                            label: profilesLoading ? "正在加载..." : "暂无可选实例",
                          },
                        ]
                  }
                  className="flex-1"
                  disabled={
                    running ||
                    demoBusy ||
                    selectableProfileOptions.length === 0
                  }
                />
              </div>
            ) : (
              <div className="mt-4 flex flex-col gap-2 xl:flex-row xl:items-center">
                <Input
                  value={createDraft.profileName}
                  onChange={(event) =>
                    setCreateDraft((current) => ({
                      ...current,
                      profileName: event.target.value,
                    }))
                  }
                  placeholder="实例名称"
                  className="xl:w-56"
                  disabled={running || demoBusy}
                />
                <Select
                  value={createDraft.templateProfileId}
                  onChange={(event) =>
                    setCreateDraft((current) => ({
                      ...current,
                      templateProfileId: event.target.value,
                    }))
                  }
                  options={
                    templateProfileOptions.length > 0
                      ? templateProfileOptions
                      : [
                          {
                            value: "",
                            label: profilesLoading ? "正在加载模板..." : "暂无模板",
                          },
                        ]
                  }
                  className="flex-1"
                  disabled={running || demoBusy || templateProfileOptions.length === 0}
                />
              </div>
            )}
          </div>
        )}

        {script.status === "disabled" ? (
          <div className="rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-4 py-4 text-sm text-[var(--color-text-secondary)]">
            该脚本当前处于停用状态，先把状态切回可用再执行。
          </div>
        ) : (
          <div
            className={
              !showsSelectorInput
                ? "grid grid-cols-1 gap-4"
                : "grid grid-cols-1 gap-4 xl:grid-cols-2"
            }
          >
            {showsSelectorInput && (
              <FormItem label="目标选择器 JSON">
                <Textarea
                  rows={12}
                  value={selectorText}
                  onChange={(event) => setSelectorText(event.target.value)}
                  className="font-mono"
                  placeholder='{"code":"DEMO_ABC123"}'
                  disabled={running || demoBusy}
                />
              </FormItem>
            )}

            <FormItem label={paramsFieldLabel}>
              <Textarea
                rows={12}
                value={paramsText}
                onChange={(event) => setParamsText(event.target.value)}
                className="font-mono"
                placeholder={paramsPlaceholder}
                disabled={running || demoBusy}
              />
            </FormItem>
          </div>
        )}

        {lastRun && (
          <div className="rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-4 py-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex flex-wrap items-center gap-2">
                <Badge
                  variant={lastRun.status === "success" ? "success" : "error"}
                  size="sm"
                  dot
                >
                  {lastRun.status === "success" ? "执行成功" : "执行失败"}
                </Badge>
                <span className="text-sm text-[var(--color-text-primary)]">
                  {lastRun.summary || "执行已完成"}
                </span>
              </div>
              <div className="text-xs text-[var(--color-text-muted)]">
                {formatDateTime(lastRun.startedAt)} ·{" "}
                {formatDuration(lastRun.durationMs)}
              </div>
            </div>

            {lastRun.error && (
              <div className="mt-3 break-all text-sm text-[var(--color-error)]">
                {lastRun.error}
              </div>
            )}

            {lastRun.resultText && (
              <div className="mt-4 space-y-2">
                <div className="flex items-center justify-between gap-2">
                  <div className="text-xs text-[var(--color-text-muted)]">
                    结果输出
                  </div>
                  <Button
                    size="sm"
                    variant="secondary"
                    onClick={() =>
                      void copyToClipboard(formattedResultText, "执行结果已复制")
                    }
                  >
                    <Copy className="h-3.5 w-3.5" />
                    复制结果
                  </Button>
                </div>
                <Textarea
                  rows={10}
                  value={formattedResultText}
                  readOnly
                  className="font-mono"
                />
                {resultOutputs.length > 0 && (
                  <div className="rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3">
                    <div className="space-y-2">
                      {resultOutputs.map((output) => (
                        <div
                          key={`${output.key}-${output.path}`}
                          className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-3 py-2"
                        >
                          <div className="min-w-0 flex-1">
                            <div className="text-sm text-[var(--color-text-primary)]">
                              {output.label} · {formatRunResultOutputName(output.path)}
                            </div>
                            <div className="mt-1 break-all text-xs text-[var(--color-text-muted)]">
                              {output.path}
                            </div>
                          </div>
                          <Button
                            size="sm"
                            variant="secondary"
                            onClick={() => void handleOpenOutputPath(output.path)}
                          >
                            <FolderOpen className="h-3.5 w-3.5" />
                            打开文件夹
                          </Button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            )}
          </div>
        )}
      </div>
    </Modal>
  );
}
