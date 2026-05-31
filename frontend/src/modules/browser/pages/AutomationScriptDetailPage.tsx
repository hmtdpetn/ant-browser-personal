import { type ReactNode, useEffect, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import {
  ArrowLeft,
  Copy,
  Download,
  Play,
  RefreshCw,
  Save,
  Trash2,
} from "lucide-react";
import {
  Badge,
  Button,
  FormItem,
  Input,
  Select,
  Textarea,
  toast,
} from "../../../shared/components";
import { fetchBrowserProfiles, fetchGroups } from "../api";
import {
  deleteAutomationScript,
  exportAutomationScriptDirectory,
  exportAutomationScriptTemplate,
  exportAutomationScriptZip,
  fetchAutomationScripts,
  refreshAutomationScript,
  saveAutomationScript,
} from "../automationScriptApi";
import {
  AutomationScriptExportModal,
  type AutomationScriptExportFormat,
} from "../components/AutomationScriptExportModal";
import { AutomationScriptRunModal } from "../components/AutomationScriptRunModal";
import {
  AUTOMATION_SCRIPT_STATUS_OPTIONS,
  AUTOMATION_SCRIPT_TARGET_MODE_OPTIONS,
  DUAL_INSTANCE_RUNTIME_SCRIPT_ID,
  canRefreshAutomationScriptSource,
  createAutomationScriptTargetSelector,
  findAutomationTargetProfile,
  formatAutomationTargetIdentity,
  getAutomationScriptRefreshLabel,
  getAutomationScriptSourceLabel,
  getAutomationScriptTypeLabel,
  type AutomationScriptRecord,
  type AutomationScriptStatus,
  type AutomationScriptTargetConfig,
  type AutomationScriptTargetSelector,
} from "../automationScripts";
import { useLaunchContext } from "../hooks/useLaunchContext";
import type { BrowserGroupWithCount, BrowserProfile } from "../types";

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

function targetModeBadgeVariant(
  mode: AutomationScriptTargetConfig["mode"],
): "default" | "info" | "warning" | "success" {
  switch (mode) {
    case "existing":
      return "info";
    case "create":
      return "success";
    case "rotate":
      return "warning";
    default:
      return "default";
  }
}

function formatTargetModeLabel(mode: AutomationScriptTargetConfig["mode"]): string {
  return (
    AUTOMATION_SCRIPT_TARGET_MODE_OPTIONS.find((item) => item.value === mode)
      ?.label || mode
  );
}

function formatScriptSource(script: AutomationScriptRecord): string {
  const { source } = script;
  if (!source.type && !source.uri && !source.path) {
    return "手动维护";
  }

  const mainValue = source.uri || source.path || "已导入";
  const extras = [source.ref, source.path && source.uri ? source.path : ""]
    .filter(Boolean)
    .join(" · ");

  return extras ? `${getAutomationScriptSourceLabel(source)} · ${mainValue} · ${extras}` : `${getAutomationScriptSourceLabel(source)} · ${mainValue}`;
}

function parseSelectorTerms(text: string): string[] {
  const deduped = new Set<string>();
  for (const item of text.split(/[\n,]/g)) {
    const normalized = item.trim();
    if (normalized) {
      deduped.add(normalized);
    }
  }
  return Array.from(deduped);
}

function formatSelectorTerms(items: string[]): string {
  return items.join("\n");
}

function normalizeStringList(value: unknown): string[] {
  if (!Array.isArray(value)) {
    return [];
  }

  return value
    .map((item) => String(item || "").trim())
    .filter(Boolean);
}

function normalizeDualRuntimeCode(value: unknown, fallback = ""): string {
  return String(value || fallback || "").trim().toUpperCase();
}

interface DualRuntimeRequestPreview {
  code: string;
  payload: Record<string, unknown>;
}

function buildDualRuntimeRequestPreviews(
  paramsText: string,
): { requests: DualRuntimeRequestPreview[]; error: string } {
  const sourceText = paramsText.trim();
  if (!sourceText) {
    return { requests: [], error: "" };
  }

  try {
    const parsed = JSON.parse(sourceText) as Record<string, unknown>;
    const timeoutMs = Number.isFinite(Number(parsed.timeoutMs))
      ? Math.max(1000, Math.round(Number(parsed.timeoutMs)))
      : 45000;
    const defaultSkipDefaultStartUrls = parsed.skipDefaultStartUrls !== false;

    const normalizeBrowser = (
      value: unknown,
      fallbackCode: string,
    ): DualRuntimeRequestPreview | null => {
      const raw =
        value && typeof value === "object"
          ? (value as Record<string, unknown>)
          : {};
      const directCode =
        typeof value === "string" || typeof value === "number" ? value : "";
      const code = normalizeDualRuntimeCode(
        raw.code ?? raw.launchCode ?? directCode,
        fallbackCode,
      );
      if (!code) {
        return null;
      }

      const skipDefaultStartUrls =
        raw.skipDefaultStartUrls !== undefined
          ? raw.skipDefaultStartUrls !== false
          : defaultSkipDefaultStartUrls;
      const startUrls = normalizeStringList(raw.startUrls);
      const launchArgs = normalizeStringList(raw.launchArgs);

      return {
        code,
        payload: {
          selector: { code, matchMode: "unique" },
          skipDefaultStartUrls,
          ...(startUrls.length > 0 ? { startUrls } : {}),
          ...(launchArgs.length > 0 ? { launchArgs } : {}),
          timeoutMs,
        },
      };
    };

    let requests = Array.isArray(parsed.browsers)
      ? parsed.browsers
          .map((item, index) =>
            normalizeBrowser(item, index === 0 ? "BUYER_001" : "BUYER_002"),
          )
          .filter((item): item is DualRuntimeRequestPreview => Boolean(item))
      : [];

    if (requests.length === 0) {
      requests = [
        normalizeBrowser(parsed.primaryCode, "BUYER_001"),
        normalizeBrowser(parsed.secondaryCode, "BUYER_002"),
      ].filter((item): item is DualRuntimeRequestPreview => Boolean(item));
    }

    return { requests, error: "" };
  } catch (error: unknown) {
    return {
      requests: [],
      error: error instanceof Error ? error.message : "JSON 解析失败",
    };
  }
}

function buildRuntimeSessionHttpPreview(
  baseUrl: string,
  authHeader: string,
  payload: Record<string, unknown>,
): string {
  const lines = [
    `POST ${baseUrl}/api/runtime/session`,
    "Content-Type: application/json",
  ];

  if (authHeader) {
    lines.push(`${authHeader}: <your-api-key>`);
  }

  lines.push("", JSON.stringify(payload, null, 2));
  return lines.join("\n");
}

function buildOpenClawDualSiteCommand(scriptID: string, codes: string[]): string {
  const dedupedCodes = Array.from(
    new Set(
      codes
        .map((item) => item.trim().toUpperCase())
        .filter(Boolean),
    ),
  );
  const primaryCode = dedupedCodes[0] || "BUYER_001";
  const secondaryCode = dedupedCodes[1] || "BUYER_002";
  const targetScriptID = scriptID.trim() || DUAL_INSTANCE_RUNTIME_SCRIPT_ID;

  return [
    "使用 ant-chrome-openclaw skill。",
    `请由 OpenClaw 触发执行预置脚本 ${targetScriptID}。`,
    `参数里 browsers 使用 ${primaryCode} 和 ${secondaryCode}，并分别设置 startUrls。`,
    `${primaryCode} 的 startUrls 固定为 ["https://finance.sina.com.cn/"]。`,
    `${secondaryCode} 的 startUrls 固定为 ["https://map.baidu.com/"]。`,
    "必须直接在 runtime/session 请求中带 startUrls，一次启动到目标站点。",
    "不要先空启动 runtime/session 再调用 launch（否则会先出现 about:blank）。",
    "两个站点必须分开实例执行，不要混用会话，不要停止实例。",
    "返回两个实例各自的页面标题、当前 URL 和执行结果。",
  ].join("\n");
}

interface SelectorSuggestion {
  key: string;
  value: string;
  label?: string;
}

function buildProfileSuggestions(
  profiles: BrowserProfile[],
  resolveValue: (profile: BrowserProfile) => string | undefined,
  resolveLabel: (profile: BrowserProfile) => string,
): SelectorSuggestion[] {
  const seen = new Set<string>();
  const suggestions: SelectorSuggestion[] = [];

  profiles.forEach((profile) => {
    const value = resolveValue(profile)?.trim();
    if (!value || seen.has(value)) {
      return;
    }
    seen.add(value);
    suggestions.push({
      key: profile.profileId,
      value,
      label: resolveLabel(profile),
    });
  });

  return suggestions.sort((left, right) =>
    left.value.localeCompare(right.value, "zh-CN"),
  );
}

function buildGroupOptions(
  groups: BrowserGroupWithCount[],
): Array<{ value: string; label: string }> {
  const result: Array<{ value: string; label: string }> = [];

  const appendChildren = (parentId: string, level: number) => {
    groups
      .filter((group) => group.parentId === parentId)
      .sort((left, right) => left.sortOrder - right.sortOrder)
      .forEach((group) => {
        result.push({
          value: group.groupId,
          label: `${"\u3000".repeat(level)}${group.groupName}`,
        });
        appendChildren(group.groupId, level + 1);
      });
  };

  appendChildren("", 0);
  return result;
}

function buildExactProfileOptions(
  profiles: BrowserProfile[],
  selector: AutomationScriptTargetSelector,
  placeholder: string,
): Array<{ value: string; label: string }> {
  const options = [
    { value: "", label: placeholder },
    ...profiles
      .slice()
      .sort((left, right) =>
        (left.profileName || left.profileId).localeCompare(
          right.profileName || right.profileId,
          "zh-CN",
        ),
      )
      .map((profile) => ({
        value: profile.profileId,
        label: [
          profile.launchCode || "",
          profile.profileName || profile.profileId,
        ]
          .filter(Boolean)
          .join(" · "),
      })),
  ];

  const currentProfileId = selector.profileId.trim();
  const currentCode = selector.code.trim().toUpperCase();
  if (
    (currentProfileId || currentCode) &&
    !options.some((item) => item.value === currentProfileId)
  ) {
    options.push({
      value: currentProfileId,
      label: [currentCode, currentProfileId || "未绑定实例", "已不存在"]
        .filter(Boolean)
        .join(" · "),
    });
  }

  return options;
}

function hasTargetSelectorCondition(
  selector: AutomationScriptTargetSelector,
): boolean {
  return Boolean(
    selector.code ||
      selector.profileId ||
      selector.profileName ||
      selector.groupId ||
      selector.keywords.length > 0 ||
      selector.tags.length > 0,
  );
}

function validateTargetConfig(config: AutomationScriptTargetConfig): string {
  switch (config.mode) {
    case "existing":
      return hasTargetSelectorCondition(config.selector)
        ? ""
        : "“使用已有实例”至少要填一个匹配条件";
    case "create":
      return hasTargetSelectorCondition(config.templateSelector)
        ? ""
        : "“按模板新建实例”至少要填一个模板条件";
    case "rotate":
      return hasTargetSelectorCondition(config.selector)
        ? ""
        : "“按条件轮询实例”至少要填一个轮询条件";
    default:
      return "";
  }
}

interface TargetSelectorEditorProps {
  selector: AutomationScriptTargetSelector;
  onChange: (patch: Partial<AutomationScriptTargetSelector>) => void;
  codeSuggestions: SelectorSuggestion[];
  profileIdSuggestions: SelectorSuggestion[];
  profileNameSuggestions: SelectorSuggestion[];
  groupOptions: Array<{ value: string; label: string }>;
  disabled?: boolean;
}

function TargetSelectorEditor({
  selector,
  onChange,
  codeSuggestions,
  profileIdSuggestions,
  profileNameSuggestions,
  groupOptions,
  disabled = false,
}: TargetSelectorEditorProps) {
  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <FormItem label="Code">
        <Input
          value={selector.code}
          onChange={(event) =>
            onChange({ code: event.target.value.trim().toUpperCase() })
          }
          placeholder="优先推荐，例如 BUYER_001"
          list="automation-script-target-code-options"
          disabled={disabled}
        />
        {codeSuggestions.length > 0 ? (
          <datalist id="automation-script-target-code-options">
            {codeSuggestions.map((item) => (
              <option key={item.key} value={item.value}>
                {item.label}
              </option>
            ))}
          </datalist>
        ) : null}
      </FormItem>

      <FormItem label="实例 ID（高级）">
        <Input
          value={selector.profileId}
          onChange={(event) =>
            onChange({ profileId: event.target.value.trim() })
          }
          placeholder="内部主键，通常不需要手填"
          list="automation-script-target-profile-id-options"
          disabled={disabled}
        />
        {profileIdSuggestions.length > 0 ? (
          <datalist id="automation-script-target-profile-id-options">
            {profileIdSuggestions.map((item) => (
              <option key={item.key} value={item.value}>
                {item.label}
              </option>
            ))}
          </datalist>
        ) : null}
      </FormItem>

      <FormItem label="实例名称">
        <Input
          value={selector.profileName}
          onChange={(event) =>
            onChange({ profileName: event.target.value.trim() })
          }
          placeholder="精确匹配实例名称"
          list="automation-script-target-profile-name-options"
          disabled={disabled}
        />
        {profileNameSuggestions.length > 0 ? (
          <datalist id="automation-script-target-profile-name-options">
            {profileNameSuggestions.map((item) => (
              <option key={item.key} value={item.value}>
                {item.label}
              </option>
            ))}
          </datalist>
        ) : null}
      </FormItem>

      <FormItem label="分组 ID">
        <Select
          value={selector.groupId}
          onChange={(event) =>
            onChange({ groupId: event.target.value.trim() })
          }
          options={groupOptions}
          disabled={disabled}
        />
      </FormItem>

      <FormItem label="标签">
        <Textarea
          rows={3}
          value={formatSelectorTerms(selector.tags)}
          onChange={(event) =>
            onChange({ tags: parseSelectorTerms(event.target.value) })
          }
          placeholder={"sales-us\nbuyer"}
          disabled={disabled}
        />
      </FormItem>

      <FormItem label="关键字">
        <Textarea
          rows={3}
          value={formatSelectorTerms(selector.keywords)}
          onChange={(event) =>
            onChange({ keywords: parseSelectorTerms(event.target.value) })
          }
          placeholder={"buyer-001\nwarm-account"}
          disabled={disabled}
        />
      </FormItem>
    </div>
  );
}

interface DetailPanelProps {
  title: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
}

function DetailPanel({
  title,
  actions,
  children,
  className = "",
}: DetailPanelProps) {
  return (
    <section
      className={`rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] p-4 ${className}`}
    >
      <div className="flex flex-wrap items-start justify-between gap-2">
        <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">
          {title}
        </h2>
        {actions ? <div className="flex items-center gap-2">{actions}</div> : null}
      </div>
      <div className="mt-4 space-y-3">{children}</div>
    </section>
  );
}

function CompactMetaField({
  label,
  value,
}: {
  label: string;
  value: ReactNode;
}) {
  return (
    <div className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3">
      <div className="text-[10px] font-semibold uppercase tracking-[0.14em] text-[var(--color-text-muted)]">
        {label}
      </div>
      <div className="mt-1 text-sm font-medium text-[var(--color-text-primary)]">
        {value}
      </div>
    </div>
  );
}

function StructuredInfoCell({
  label,
  children,
}: {
  label: string;
  children: ReactNode;
}) {
  return (
    <div className="px-4 py-3">
      <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--color-text-muted)]">
        {label}
      </div>
      <div className="mt-2 text-sm font-medium text-[var(--color-text-primary)]">
        {children}
      </div>
    </div>
  );
}

function ExactTargetSummary({
  title,
  selector,
  profiles,
}: {
  title: string;
  selector: AutomationScriptTargetSelector;
  profiles: BrowserProfile[];
}) {
  const profile = findAutomationTargetProfile(selector, profiles);
  const profileId = String(profile?.profileId || selector.profileId || "").trim();
  const targetLabel = formatAutomationTargetIdentity(selector, profiles, {
    fallback: "未匹配到实例",
  });

  return (
    <div className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3">
      <div className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--color-text-muted)]">
        {title}
      </div>
      <div className="mt-2 text-sm font-medium text-[var(--color-text-primary)]">
        {targetLabel}
      </div>
      {profileId ? (
        <div className="mt-1 break-all text-xs text-[var(--color-text-muted)]">
          实例 ID {profileId}
        </div>
      ) : null}
    </div>
  );
}

export function AutomationScriptDetailPage() {
  const navigate = useNavigate();
  const { scriptId = "" } = useParams();
  const [draft, setDraft] = useState<AutomationScriptRecord | null>(null);
  const [profiles, setProfiles] = useState<BrowserProfile[]>([]);
  const [groups, setGroups] = useState<BrowserGroupWithCount[]>([]);
  const [loading, setLoading] = useState(true);
  const [notFound, setNotFound] = useState(false);
  const [dirty, setDirty] = useState(false);
  const [runModalOpen, setRunModalOpen] = useState(false);
  const [exportModalOpen, setExportModalOpen] = useState(false);
  const [showDualRuntimeRequests, setShowDualRuntimeRequests] = useState(false);
  const [busyAction, setBusyAction] = useState<
    "none" | "save" | "delete" | "refresh" | "export"
  >("none");
  const isDualInstanceRuntimeScript =
    draft?.id === DUAL_INSTANCE_RUNTIME_SCRIPT_ID;
  const { launchBaseUrl, apiAuth } = useLaunchContext({
    enabled: isDualInstanceRuntimeScript,
  });

  useEffect(() => {
    let disposed = false;

    setLoading(true);
    setNotFound(false);

    void fetchAutomationScripts()
      .then((items) => {
        if (disposed) {
          return;
        }

        const current = items.find((item) => item.id === scriptId) || null;
        setDraft(current);
        setDirty(false);
        setNotFound(!current);
      })
      .catch(() => {
        if (!disposed) {
          toast.error("脚本加载失败");
        }
      })
      .finally(() => {
        if (!disposed) {
          setLoading(false);
        }
      });

    return () => {
      disposed = true;
    };
  }, [scriptId]);

  useEffect(() => {
    setShowDualRuntimeRequests(false);
  }, [scriptId]);

  useEffect(() => {
    let disposed = false;

    void Promise.allSettled([fetchBrowserProfiles(), fetchGroups()]).then(
      ([profilesResult, groupsResult]) => {
        if (disposed) {
          return;
        }
        if (profilesResult.status === "fulfilled") {
          setProfiles(profilesResult.value || []);
        }
        if (groupsResult.status === "fulfilled") {
          setGroups(groupsResult.value || []);
        }
      },
    );

    return () => {
      disposed = true;
    };
  }, []);

  useEffect(() => {
    if (!dirty) {
      return undefined;
    }

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };

    window.addEventListener("beforeunload", handleBeforeUnload);
    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [dirty]);

  const updateDraft = (patch: Partial<AutomationScriptRecord>) => {
    setDraft((current) => {
      if (!current) {
        return current;
      }
      return {
        ...current,
        ...patch,
      };
    });
    setDirty(true);
  };

  const updateTargetConfig = (patch: Partial<AutomationScriptTargetConfig>) => {
    setDraft((current) => {
      if (!current) {
        return current;
      }
      return {
        ...current,
        targetConfig: {
          ...current.targetConfig,
          ...patch,
        },
      };
    });
    setDirty(true);
  };

  const updateTargetSelector = (
    key: "selector" | "templateSelector",
    patch: Partial<AutomationScriptTargetSelector>,
  ) => {
    setDraft((current) => {
      if (!current) {
        return current;
      }
      return {
        ...current,
        targetConfig: {
          ...current.targetConfig,
          [key]: {
            ...current.targetConfig[key],
            ...patch,
          },
        },
      };
    });
    setDirty(true);
  };

  const leavePage = () => {
    if (dirty && !window.confirm("当前脚本有未保存修改，确认离开吗？")) {
      return;
    }
    navigate("/browser/automation");
  };

  const handleSave = async () => {
    if (!draft) {
      return;
    }

    if (!draft.name.trim()) {
      toast.warning("脚本名称不能为空");
      return;
    }
    if (!draft.scriptText.trim()) {
      toast.warning(
        draft.type === "launch-api"
          ? "固定接口模板不能为空"
          : "脚本内容不能为空",
      );
      return;
    }
    const targetConfigError = validateTargetConfig(draft.targetConfig);
    if (targetConfigError) {
      toast.warning(targetConfigError);
      return;
    }

    setBusyAction("save");
    try {
      const saved = await saveAutomationScript({
        ...draft,
        name: draft.name.trim(),
        description: draft.description.trim(),
        scriptText: draft.scriptText,
        updatedAt: new Date().toISOString(),
      });
      setDraft(saved);
      setDirty(false);
      toast.success("脚本已保存");
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : "脚本保存失败";
      toast.error(message);
    } finally {
      setBusyAction("none");
    }
  };

  const handleOpenRunModal = () => {
    if (!draft) {
      return;
    }
    setRunModalOpen(true);
  };

  const handleDelete = async () => {
    if (!draft) {
      return;
    }
    if (!window.confirm(`确认删除脚本「${draft.name || "未命名脚本"}」吗？`)) {
      return;
    }

    setBusyAction("delete");
    try {
      await deleteAutomationScript(draft.id);
      toast.success("脚本已删除");
      navigate("/browser/automation", { replace: true });
    } catch (error: unknown) {
      const message = error instanceof Error ? error.message : "脚本删除失败";
      toast.error(message);
    } finally {
      setBusyAction("none");
    }
  };

  const handleRefresh = async () => {
    if (!draft) {
      return;
    }
    if (!canRefreshAutomationScriptSource(draft.source)) {
      toast.warning("当前脚本来源不支持重新导入");
      return;
    }
    if (dirty) {
      toast.warning("请先保存当前修改，再重新导入");
      return;
    }
    if (!window.confirm("确认按来源重新导入当前脚本吗？这会覆盖当前脚本内容。")) {
      return;
    }

    setBusyAction("refresh");
    try {
      const refreshed = await refreshAutomationScript(draft.id);
      setDraft(refreshed);
      setDirty(false);
      toast.success(
        draft.source.type === "git" ? "脚本已重新拉取" : "脚本已重新导入",
      );
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : "脚本重新导入失败";
      toast.error(message);
    } finally {
      setBusyAction("none");
    }
  };

  const handleOpenExportModal = () => {
    if (!draft) {
      return;
    }
    setExportModalOpen(true);
  };

  const handleExport = async (format: AutomationScriptExportFormat) => {
    if (!draft) {
      return;
    }
    if (dirty) {
      toast.warning("请先保存当前修改，再导出");
      return;
    }

    setBusyAction("export");
    try {
      const result =
        format === "json"
          ? await exportAutomationScriptTemplate(draft.id, draft)
          : format === "directory"
            ? await exportAutomationScriptDirectory(draft.id)
            : await exportAutomationScriptZip(draft.id);
      if (result.cancelled) {
        setExportModalOpen(false);
        return;
      }

      setExportModalOpen(false);
      switch (format) {
        case "directory":
          toast.success(
            result.fileCount > 1
              ? `目录已导出，包含 ${result.fileCount} 个文件`
              : result.message || "目录已导出",
          );
          break;
        case "zip":
          toast.success(
            result.fileCount > 1
              ? `ZIP 已导出，包含 ${result.fileCount} 个文件`
              : result.message || "ZIP 已导出",
          );
          break;
        default:
          toast.success(
            result.fileCount > 1
              ? `模板已导出，包含 ${result.fileCount} 个文件`
              : result.message || "模板已导出",
          );
          break;
      }
    } catch (error: unknown) {
      const message =
        error instanceof Error ? error.message : "脚本导出失败";
      toast.error(message);
    } finally {
      setBusyAction("none");
    }
  };

  if (loading) {
    return (
      <div className="animate-fade-in rounded-2xl border border-dashed border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-6 py-12 text-center text-sm text-[var(--color-text-muted)]">
        正在加载脚本...
      </div>
    );
  }

  if (notFound || !draft) {
    return (
      <div className="space-y-4 animate-fade-in">
        <Button
          variant="secondary"
          size="sm"
          onClick={() => navigate("/browser/automation")}
        >
          <ArrowLeft className="h-4 w-4" />
          返回列表
        </Button>
        <div className="rounded-2xl border border-dashed border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-6 py-12 text-center text-sm text-[var(--color-text-muted)]">
          脚本不存在或已被删除。
        </div>
      </div>
    );
  }

  const busy = busyAction !== "none";
  const canRefresh = canRefreshAutomationScriptSource(draft.source);
  const isLaunchApiScript = draft.type === "launch-api";
  const usesManualSelector = draft.targetConfig.mode === "manual";
  const targetModeLabel = formatTargetModeLabel(draft.targetConfig.mode);
  const dualRuntimePreview = isDualInstanceRuntimeScript
    ? buildDualRuntimeRequestPreviews(draft.paramsText)
    : { requests: [], error: "" };
  const dualRuntimeCodes = dualRuntimePreview.requests.map(
    (request) => request.code,
  );
  const openClawDualSiteCommand = buildOpenClawDualSiteCommand(
    draft.id,
    dualRuntimeCodes,
  );
  const codeSuggestions = buildProfileSuggestions(
    profiles,
    (profile) => profile.launchCode,
    (profile) =>
      profile.profileName
        ? `${profile.launchCode || "未设 Code"} · ${profile.profileName}`
        : profile.profileId,
  );
  const profileIdSuggestions = buildProfileSuggestions(
    profiles,
    (profile) => profile.profileId,
    (profile) =>
      profile.launchCode
        ? `${profile.launchCode} · ${profile.profileName || profile.profileId}`
        : profile.profileName || profile.profileId,
  );
  const profileNameSuggestions = buildProfileSuggestions(
    profiles,
    (profile) => profile.profileName,
    (profile) =>
      profile.launchCode
        ? `${profile.launchCode} · ${profile.profileId}`
        : profile.profileId,
  );
  const groupOptions = [
    { value: "", label: "不限制" },
    ...buildGroupOptions(groups),
  ];
  const existingProfileOptions = buildExactProfileOptions(
    profiles,
    draft.targetConfig.selector,
    "选择已有实例",
  );
  const templateProfileOptions = buildExactProfileOptions(
    profiles,
    draft.targetConfig.templateSelector,
    "选择模板实例",
  );
  const handleSelectExactProfile = (
    key: "selector" | "templateSelector",
    profileId: string,
  ) => {
    const nextSelector = createAutomationScriptTargetSelector();
    const normalizedProfileId = profileId.trim();
    nextSelector.profileId = normalizedProfileId;
    const matchedProfile = profiles.find(
      (profile) => profile.profileId === normalizedProfileId,
    );
    if (matchedProfile?.launchCode) {
      nextSelector.code = matchedProfile.launchCode.trim().toUpperCase();
    }
    updateTargetSelector(key, nextSelector);
  };
  const handleCopyOpenClawCommand = async () => {
    try {
      await navigator.clipboard.writeText(openClawDualSiteCommand);
      toast.success("OpenClaw 指令已复制");
    } catch {
      toast.error("复制失败");
    }
  };

  return (
    <div className="space-y-5 animate-fade-in">
      <section className="rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-4 py-3 shadow-[var(--shadow-sm)]">
        <div className="flex flex-wrap items-center gap-3">
          <Button variant="secondary" size="sm" onClick={leavePage}>
            <ArrowLeft className="h-4 w-4" />
            返回目录
          </Button>
          <div className="text-sm font-semibold text-[var(--color-text-secondary)]">
            当前脚本
          </div>
          <div className="min-w-[320px] flex-1">
            <Input
              value={draft.name}
              onChange={(event) => updateDraft({ name: event.target.value })}
              placeholder="脚本名称"
              className="h-10 text-base font-semibold"
            />
          </div>
          {canRefresh ? (
            <Button
              size="sm"
              variant="secondary"
              onClick={() => void handleRefresh()}
              loading={busyAction === "refresh"}
              disabled={busyAction !== "none" && busyAction !== "refresh"}
            >
              <RefreshCw className="h-4 w-4" />
              {getAutomationScriptRefreshLabel(draft.source)}
            </Button>
          ) : null}
        </div>

        <div className="mt-3 flex flex-wrap items-center justify-end gap-2 border-t border-[var(--color-border-muted)] pt-3">
          <Button
            size="sm"
            variant="secondary"
            onClick={handleOpenRunModal}
            disabled={busyAction !== "none"}
          >
            <Play className="h-4 w-4" />
            执行
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={handleOpenExportModal}
            loading={busyAction === "export"}
            disabled={busyAction !== "none" && busyAction !== "export"}
          >
            <Download className="h-4 w-4" />
            导出
          </Button>
          <Button
            size="sm"
            onClick={() => void handleSave()}
            loading={busyAction === "save"}
            disabled={busyAction !== "none" && busyAction !== "save"}
          >
            <Save className="h-4 w-4" />
            保存
          </Button>
          <Button
            size="sm"
            variant="danger"
            onClick={() => void handleDelete()}
            loading={busyAction === "delete"}
            disabled={busyAction !== "none" && busyAction !== "delete"}
          >
            <Trash2 className="h-4 w-4" />
            删除
          </Button>
        </div>
      </section>

      <div className="grid grid-cols-1 items-stretch gap-4 2xl:grid-cols-[minmax(0,1.1fr)_minmax(320px,0.9fr)]">
        <DetailPanel title="基础信息" className="h-full">
          <FormItem label="描述">
            <Textarea
              rows={2}
              value={draft.description}
              onChange={(event) =>
                updateDraft({ description: event.target.value })
              }
              placeholder="说明这套脚本要做什么"
            />
          </FormItem>

          <div className="overflow-hidden rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] shadow-[var(--shadow-sm)]">
            <div className="grid grid-cols-1 divide-y divide-[var(--color-border-muted)] md:grid-cols-2 md:divide-x md:divide-y xl:grid-cols-4 xl:divide-y-0">
              <StructuredInfoCell label="类型">
                {getAutomationScriptTypeLabel(draft.type)}
              </StructuredInfoCell>
              <StructuredInfoCell label="状态">
                <Select
                  value={draft.status}
                  options={AUTOMATION_SCRIPT_STATUS_OPTIONS}
                  onChange={(event) =>
                    updateDraft({
                      status: event.target.value as AutomationScriptStatus,
                    })
                  }
                  className="h-9"
                  disabled={busy}
                />
              </StructuredInfoCell>
              <StructuredInfoCell label="最近更新">
                {formatDateTime(draft.updatedAt)}
              </StructuredInfoCell>
              <StructuredInfoCell label="编辑状态">
                <span
                  className={dirty ? "text-[var(--color-warning)]" : undefined}
                >
                  {dirty ? "未保存" : "已保存"}
                </span>
              </StructuredInfoCell>
            </div>
          </div>

          <div className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-2.5">
            <div className="text-sm text-[var(--color-text-secondary)]">
              {formatScriptSource(draft)}
            </div>
            {draft.source.importedAt ? (
              <div className="mt-1 text-xs text-[var(--color-text-muted)]">
                最近导入 {formatDateTime(draft.source.importedAt)}
              </div>
            ) : null}
          </div>
        </DetailPanel>

        {isDualInstanceRuntimeScript ? (
          <DetailPanel title="执行模型" className="h-full">
            <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
              <CompactMetaField label="类型" value="接口模拟" />
              <CompactMetaField label="执行方式" value="逐个启动" />
              <CompactMetaField label="Selector" value="不使用" />
            </div>
          </DetailPanel>
        ) : (
          <DetailPanel
            title="实例策略"
            className="h-full"
            actions={
              <Badge
                variant={targetModeBadgeVariant(draft.targetConfig.mode)}
                size="sm"
              >
                {targetModeLabel}
              </Badge>
            }
          >
            <FormItem label="策略模式">
              <Select
                value={draft.targetConfig.mode}
                options={AUTOMATION_SCRIPT_TARGET_MODE_OPTIONS}
                onChange={(event) =>
                  updateTargetConfig({
                    mode:
                      event.target.value as AutomationScriptTargetConfig["mode"],
                  })
                }
                disabled={busy}
              />
            </FormItem>

            {draft.targetConfig.mode === "existing" ? (
              <div className="space-y-3">
                <FormItem label="已有实例">
                  <Select
                    value={draft.targetConfig.selector.profileId}
                    options={existingProfileOptions}
                    onChange={(event) =>
                      handleSelectExactProfile("selector", event.target.value)
                    }
                    disabled={busy}
                  />
                </FormItem>
                <ExactTargetSummary
                  title="当前绑定"
                  selector={draft.targetConfig.selector}
                  profiles={profiles}
                />
              </div>
            ) : null}

            {draft.targetConfig.mode === "rotate" ? (
              <TargetSelectorEditor
                selector={draft.targetConfig.selector}
                onChange={(patch) => updateTargetSelector("selector", patch)}
                codeSuggestions={codeSuggestions}
                profileIdSuggestions={profileIdSuggestions}
                profileNameSuggestions={profileNameSuggestions}
                groupOptions={groupOptions}
                disabled={busy}
              />
            ) : null}

            {draft.targetConfig.mode === "create" ? (
              <div className="space-y-3">
                <FormItem label="模板实例">
                  <Select
                    value={draft.targetConfig.templateSelector.profileId}
                    options={templateProfileOptions}
                    onChange={(event) =>
                      handleSelectExactProfile(
                        "templateSelector",
                        event.target.value,
                      )
                    }
                    disabled={busy}
                  />
                </FormItem>
                <ExactTargetSummary
                  title="当前模板"
                  selector={draft.targetConfig.templateSelector}
                  profiles={profiles}
                />
                <FormItem label="新实例命名模板">
                  <Input
                    value={draft.targetConfig.createNameTemplate}
                    onChange={(event) =>
                      updateTargetConfig({
                        createNameTemplate: event.target.value,
                      })
                    }
                    placeholder="${templateName}-${timestamp}"
                    disabled={busy}
                  />
                </FormItem>
              </div>
            ) : null}
          </DetailPanel>
        )}
      </div>

      <DetailPanel title={isDualInstanceRuntimeScript ? "启动配置" : "运行配置"}>
        {isDualInstanceRuntimeScript ? (
          <div className="space-y-4">
            <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-4">
              <CompactMetaField label="Method" value="POST" />
              <CompactMetaField
                label="Path"
                value={<code>/api/runtime/session</code>}
              />
              <CompactMetaField
                label="Base URL"
                value={<span className="break-all">{launchBaseUrl}</span>}
              />
              <CompactMetaField
                label="调用次数"
                value={`${dualRuntimePreview.requests.length || 0} 次`}
              />
            </div>

            {dualRuntimePreview.error ? (
              <div className="rounded-xl border border-[var(--color-warning)]/30 bg-[var(--color-warning)]/10 px-3 py-3 text-sm text-[var(--color-text-secondary)]">
                当前启动配置 JSON 无法解析，暂时不能展开接口示例：{" "}
                <code>{dualRuntimePreview.error}</code>
              </div>
            ) : null}

            {dualRuntimePreview.requests.length > 0 ? (
              <div className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <div className="text-sm font-medium text-[var(--color-text-primary)]">
                    已生成 {dualRuntimePreview.requests.length} 次接口调用示例
                  </div>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() =>
                      setShowDualRuntimeRequests((current) => !current)
                    }
                  >
                    {showDualRuntimeRequests ? "收起请求示例" : "展开请求示例"}
                  </Button>
                </div>
                {showDualRuntimeRequests ? (
                  <div className="mt-3 grid grid-cols-1 gap-4 xl:grid-cols-2">
                    {dualRuntimePreview.requests.map((request, index) => (
                      <div
                        key={`${request.code}-${index}`}
                        className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-3 py-3"
                      >
                        <div className="mb-2 text-sm font-medium text-[var(--color-text-primary)]">
                          第 {index + 1} 次接口调用 · <code>{request.code}</code>
                        </div>
                        <pre className="overflow-x-auto rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] p-3 text-xs leading-6 text-[var(--color-text-secondary)]">
                          <code>
                            {buildRuntimeSessionHttpPreview(
                              launchBaseUrl,
                              apiAuth.enabled ? apiAuth.header : "",
                              request.payload,
                            )}
                          </code>
                        </pre>
                      </div>
                    ))}
                  </div>
                ) : null}
              </div>
            ) : null}

            <div className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="text-sm font-medium text-[var(--color-text-primary)]">
                  OpenClaw 指令模板
                </div>
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  onClick={() => void handleCopyOpenClawCommand()}
                >
                  <Copy className="h-4 w-4" />
                  复制指令
                </Button>
              </div>
              <pre className="mt-3 overflow-x-auto rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] p-3 text-xs leading-6 text-[var(--color-text-secondary)]">
                <code>{openClawDualSiteCommand}</code>
              </pre>
            </div>

            <FormItem label="接口请求源配置 JSON">
              <Textarea
                rows={12}
                value={draft.paramsText}
                onChange={(event) =>
                  updateDraft({ paramsText: event.target.value })
                }
                className="font-mono"
                placeholder={`{
  "browsers": [
    { "code": "BUYER_001", "skipDefaultStartUrls": true },
    { "code": "BUYER_002", "skipDefaultStartUrls": true }
  ],
  "timeoutMs": 45000
}`}
                disabled={busy}
              />
            </FormItem>
          </div>
        ) : (
          <div
            className={`grid grid-cols-1 gap-4 ${
              usesManualSelector ? "xl:grid-cols-2" : ""
            }`}
          >
            {usesManualSelector ? (
              <FormItem label="目标选择器 JSON">
                <Textarea
                  rows={12}
                  value={draft.selectorText}
                  onChange={(event) =>
                    updateDraft({ selectorText: event.target.value })
                  }
                  className="font-mono"
                  placeholder='{"code":"BUYER_001"}'
                  disabled={busy}
                />
              </FormItem>
            ) : null}

            <FormItem label="运行参数 JSON">
              <Textarea
                rows={12}
                value={draft.paramsText}
                onChange={(event) =>
                  updateDraft({ paramsText: event.target.value })
                }
                className="font-mono"
                placeholder='{"startUrls":["https://example.com"]}'
                disabled={busy}
              />
            </FormItem>
          </div>
        )}
      </DetailPanel>

      {isLaunchApiScript ? (
        <DetailPanel title="固定模板">
          <div className="grid grid-cols-1 gap-3 md:grid-cols-3">
            <CompactMetaField label="类型" value="接口模式" />
            <CompactMetaField label="执行方式" value="系统固定" />
            <CompactMetaField
              label="可编辑项"
              value={
                isDualInstanceRuntimeScript
                  ? "启动配置"
                  : "目标策略 / 运行参数"
              }
            />
          </div>
          <div className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3 text-sm text-[var(--color-text-secondary)]">
            固定接口模板由系统维护。
          </div>
        </DetailPanel>
      ) : (
        <DetailPanel title="脚本">
          <FormItem label="脚本内容">
            <Textarea
              rows={24}
              value={draft.scriptText}
              onChange={(event) =>
                updateDraft({ scriptText: event.target.value })
              }
              className="min-h-[520px] font-mono leading-6"
              placeholder="module.exports.run = async () => {}"
            />
          </FormItem>
        </DetailPanel>
      )}

      <AutomationScriptRunModal
        open={runModalOpen}
        script={draft}
        dirty={dirty}
        onClose={() => setRunModalOpen(false)}
      />
      <AutomationScriptExportModal
        open={exportModalOpen}
        busy={busyAction === "export"}
        onClose={() => setExportModalOpen(false)}
        onSubmit={(format) => void handleExport(format)}
      />
    </div>
  );
}
