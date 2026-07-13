import type { RefObject } from "react";
import { Copy, FolderOpen, Play, Plus, Sparkles, Trash2 } from "lucide-react";
import { Button, FormItem, Input, Modal, Select, Switch } from "../../../shared/components";
import { AutomationInstanceSelector } from "./AutomationInstanceSelector";
import {
  buildCurlPreview,
  copyText,
  formatInvokeResult,
  formatPublicApiOutputName,
  type PublicApiOutputEntry,
} from "./AutomationScriptPublicApiModal.helpers";
import {
  AUTOMATION_SCRIPT_PUBLIC_API_METHOD_OPTIONS,
  type AutomationScriptPublicAPIConfig,
  type AutomationScriptPublicAPIVariable,
  type AutomationScriptRecord,
} from "../automationScripts";
import type { BrowserProfile } from "../types";
import type { AutomationScriptPublicApiInvokeResult } from "../automationScriptApi";
import { AutomationScriptPublicApiBodyExamples } from "./AutomationScriptPublicApiBodyExamples";

interface VisiblePublicApiVariable {
  variable: AutomationScriptPublicAPIVariable;
  index: number;
}

interface AutomationScriptPublicApiModalViewProps {
  open: boolean;
  onClose: () => void;
  busy: boolean;
  script: AutomationScriptRecord;
  launchBaseUrl: string;
  apiAuthEnabled: boolean;
  apiAuthHeader: string;
  profiles: BrowserProfile[];
  fullURL: string;
  fullPath: string;
  resolvedConfig: AutomationScriptPublicAPIConfig;
  requestExampleFallback: string;
  responseExampleFallback: string;
  visibleVariables: VisiblePublicApiVariable[];
  variableError: string;
  requestBodyError: string;
  responseBodyError: string;
  isDualInstanceRuntimeScript: boolean;
  selectedTargetCode: string;
  selectedPrimaryTargetCode: string;
  selectedSecondaryTargetCode: string;
  invokeDisabled: boolean;
  apiKey: string;
  setApiKey: (value: string) => void;
  invoking: boolean;
  invokeResult: AutomationScriptPublicApiInvokeResult | null;
  invokeError: string;
  outputEntries: PublicApiOutputEntry[];
  testSectionRef: RefObject<HTMLDivElement>;
  updateConfig: (patch: Partial<AutomationScriptPublicAPIConfig>) => void;
  updateVariable: (index: number, patch: Partial<AutomationScriptPublicAPIVariable>) => void;
  handleApplySuggestedPath: () => void;
  handleAddVariable: () => void;
  handleRemoveVariable: (index: number) => void;
  handleTargetCodeChange: (code: string) => void;
  handleDualTargetCodeChange: (index: number, code: string) => void;
  handleInvoke: () => Promise<void>;
  handleOpenOutputPath: (path: string) => Promise<void>;
}

export function AutomationScriptPublicApiModalView({
  open,
  onClose,
  busy,
  script,
  launchBaseUrl,
  apiAuthEnabled,
  apiAuthHeader,
  profiles,
  fullURL,
  fullPath,
  resolvedConfig,
  requestExampleFallback,
  responseExampleFallback,
  visibleVariables,
  variableError,
  requestBodyError,
  responseBodyError,
  isDualInstanceRuntimeScript,
  selectedTargetCode,
  selectedPrimaryTargetCode,
  selectedSecondaryTargetCode,
  invokeDisabled,
  apiKey,
  setApiKey,
  invoking,
  invokeResult,
  invokeError,
  outputEntries,
  testSectionRef,
  updateConfig,
  updateVariable,
  handleApplySuggestedPath,
  handleAddVariable,
  handleRemoveVariable,
  handleTargetCodeChange,
  handleDualTargetCodeChange,
  handleInvoke,
  handleOpenOutputPath,
}: AutomationScriptPublicApiModalViewProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title="对外接口管理"
      width="1100px"
      footer={
        <div className="flex w-full items-center justify-end gap-2">
          <Button variant="secondary" onClick={onClose} disabled={invoking}>
            取消
          </Button>
          <Button
            type="button"
            onClick={() => void handleInvoke()}
            loading={invoking}
            disabled={invokeDisabled}
          >
            <Play className="h-4 w-4" />
            发送测试请求
          </Button>
        </div>
      }
    >
      <div className="space-y-4">
        <div
          ref={testSectionRef}
          className="rounded-xl border border-slate-200/70 bg-slate-100 px-3 py-3 shadow-inner"
        >
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="text-sm font-semibold text-slate-800">
              测试接口
            </div>
          </div>

          <div className="mt-3 grid grid-cols-1 gap-4 xl:grid-cols-[320px_minmax(0,1fr)]">
            <div className="space-y-3">
              <div className="rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-3 py-3">
                <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-[var(--color-text-muted)]">
                  目标地址
                </div>
                <div className="mt-2 break-all text-sm text-[var(--color-text-primary)]">
                  {fullURL}
                </div>
              </div>

              {apiAuthEnabled ? (
                <FormItem label={`API Key (${apiAuthHeader})`}>
                  <Input
                    value={apiKey}
                    onChange={(event) => setApiKey(event.target.value)}
                    placeholder="留空则使用当前应用里的 Launch API Key"
                  />
                </FormItem>
              ) : (
                <div className="rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-3 py-3 text-sm text-[var(--color-text-secondary)]">
                  当前 Launch API 未启用认证，可以直接测试。
                </div>
              )}
            </div>

            <div className="rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-3 py-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="text-sm font-medium text-[var(--color-text-primary)]">
                  返回结果
                </div>
                {invokeResult ? (
                  <div className="text-xs text-[var(--color-text-muted)]">
                    HTTP {invokeResult.status} {invokeResult.statusText}
                  </div>
                ) : null}
              </div>

              {invokeError ? (
                <div className="mt-3 rounded-lg border border-[var(--color-error)]/30 bg-[var(--color-error)]/10 px-3 py-3 text-sm text-[var(--color-text-secondary)]">
                  {invokeError}
                </div>
              ) : null}

              {!invokeError && !invokeResult ? (
                <div className="mt-3 rounded-lg border border-dashed border-[var(--color-border-muted)] px-3 py-8 text-center text-sm text-[var(--color-text-muted)]">
                  发送一次测试请求后，这里显示真实响应。
                </div>
              ) : null}

              {invokeResult ? (
                <div className="mt-3 space-y-3">
                  <pre className="overflow-x-auto rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] p-3 text-xs leading-6 text-[var(--color-text-secondary)]">
                    <code>{formatInvokeResult(invokeResult)}</code>
                  </pre>
                  {outputEntries.length > 0 ? (
                    <div className="rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3">
                      <div className="space-y-2">
                        {outputEntries.map((output) => (
                          <div
                            key={`${output.key}-${output.path}`}
                            className="flex flex-wrap items-center justify-between gap-3 rounded-md border border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-3 py-2"
                          >
                            <div className="min-w-0 flex-1">
                              <div className="text-sm text-[var(--color-text-primary)]">
                                {output.label} · {formatPublicApiOutputName(output.path)}
                              </div>
                              <div className="mt-1 break-all text-xs text-[var(--color-text-muted)]">
                                {output.path}
                              </div>
                            </div>
                            <Button
                              type="button"
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
                  ) : null}
                </div>
              ) : null}
            </div>
          </div>
        </div>
        <div className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3">
          <div className="flex flex-wrap items-center gap-2">
            <Select
              value={resolvedConfig.method}
              options={AUTOMATION_SCRIPT_PUBLIC_API_METHOD_OPTIONS.map((item) => ({
                value: item.value,
                label: item.label,
              }))}
              onChange={(event) =>
                updateConfig({ method: event.target.value as "POST" })
              }
              className="w-[96px] shrink-0 font-semibold"
              disabled
            />
            <Input
              value={fullURL}
              readOnly
              className="min-w-0 flex-1 font-mono sm:min-w-[280px]"
            />
            <Button
              type="button"
              size="sm"
              variant="secondary"
              onClick={() => void copyText(fullURL, "接口地址已复制")}
              disabled={busy}
            >
              <Copy className="h-4 w-4" />
              URL
            </Button>
            <Button
              type="button"
              size="sm"
              variant="secondary"
              onClick={() =>
                void copyText(
                  buildCurlPreview(
                    script,
                    resolvedConfig,
                    launchBaseUrl,
                    apiAuthEnabled,
                    apiAuthHeader,
                  ),
                  "curl 已复制",
                )
              }
              disabled={busy}
            >
              <Copy className="h-4 w-4" />
              curl
            </Button>
            <div className="ml-auto flex h-9 items-center gap-2 rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-3 text-sm text-[var(--color-text-secondary)]">
              <span>{resolvedConfig.enabled ? "已启用" : "未启用"}</span>
              <Switch
                checked={resolvedConfig.enabled}
                onChange={(checked) => updateConfig({ enabled: checked })}
                disabled={busy}
              />
            </div>
          </div>

          <div className="mt-3 grid grid-cols-1 gap-3 lg:grid-cols-[minmax(0,1fr)_180px]">
            <FormItem label="Path">
              <div className="flex gap-2">
                <Input
                  value={resolvedConfig.path}
                  onChange={(event) => updateConfig({ path: event.target.value })}
                  placeholder="mail/proton-first-message"
                  className="font-mono"
                  disabled={busy}
                />
                <Button
                  type="button"
                  variant="secondary"
                  size="sm"
                  className="!h-9 !min-w-[88px] shrink-0 whitespace-nowrap"
                  onClick={handleApplySuggestedPath}
                  disabled={busy}
                >
                  <Sparkles className="h-4 w-4" />
                  推荐
                </Button>
              </div>
              <div className="mt-1 break-all text-xs text-[var(--color-text-muted)]">
                {fullPath}
              </div>
            </FormItem>

            <FormItem label="Timeout">
              <Input
                type="number"
                min={1000}
                max={1800000}
                value={String(resolvedConfig.timeoutMs)}
                onChange={(event) =>
                  updateConfig({ timeoutMs: Number(event.target.value) || 0 })
                }
                disabled={busy}
              />
            </FormItem>
          </div>
        </div>

        <div className="rounded-xl border border-[var(--color-border-muted)] bg-[var(--color-bg-secondary)] px-3 py-3">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="text-sm font-medium text-[var(--color-text-primary)]">
              接口变量
            </div>
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={handleAddVariable}
              disabled={busy}
            >
              <Plus className="h-4 w-4" />
              新增
            </Button>
          </div>

          {visibleVariables.length > 0 ? (
            <div className="mt-3 space-y-2">
              {visibleVariables.map(({ variable, index }) => (
                <div
                  key={`${index}-${variable.name}`}
                  className="grid grid-cols-1 gap-2 rounded-lg border border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-2 py-2 lg:grid-cols-[180px_minmax(0,1fr)_minmax(0,1fr)_86px_36px]"
                >
                  <Input
                    value={variable.name}
                    onChange={(event) =>
                      updateVariable(index, { name: event.target.value })
                    }
                    placeholder="searchQuery"
                    className="font-mono"
                    disabled={busy}
                  />
                  <Input
                    value={variable.defaultValue}
                    onChange={(event) =>
                      updateVariable(index, { defaultValue: event.target.value })
                    }
                    placeholder="默认值"
                    disabled={busy}
                  />
                  <Input
                    value={variable.description}
                    onChange={(event) =>
                      updateVariable(index, { description: event.target.value })
                    }
                    placeholder="说明"
                    disabled={busy}
                  />
                  <label className="flex h-9 items-center justify-center gap-2 rounded-lg border border-[var(--color-border-muted)] text-sm text-[var(--color-text-secondary)]">
                    <input
                      type="checkbox"
                      checked={variable.required}
                      onChange={(event) =>
                        updateVariable(index, { required: event.target.checked })
                      }
                      disabled={busy}
                    />
                    必填
                  </label>
                  <Button
                    type="button"
                    variant="secondary"
                    size="sm"
                    onClick={() => handleRemoveVariable(index)}
                    disabled={busy}
                    aria-label="删除变量"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>
          ) : (
            <div className="mt-3 rounded-lg border border-dashed border-[var(--color-border-muted)] px-3 py-3 text-sm text-[var(--color-text-muted)]">
              未配置变量
            </div>
          )}

          {variableError ? (
            <p className="mt-2 text-xs text-[var(--color-error)]">
              {variableError}
            </p>
          ) : (
            <p className="mt-2 text-xs text-[var(--color-text-muted)]">
              用 <code>{"{{name}}"}</code> 占位；实例 Code 由下方实例选择维护。
            </p>
          )}
        </div>

        {isDualInstanceRuntimeScript ? (
          <div className="grid grid-cols-1 gap-3 xl:grid-cols-2">
            <AutomationInstanceSelector
              title="传入实例 1"
              mode="manual"
              modes={["manual"]}
              profiles={profiles}
              selectedCode={selectedPrimaryTargetCode}
              disabled={busy}
              codePlaceholder="例如 BUYER_001"
              onCodeChange={(code) => handleDualTargetCodeChange(0, code)}
            />
            <AutomationInstanceSelector
              title="传入实例 2"
              mode="manual"
              modes={["manual"]}
              profiles={profiles}
              selectedCode={selectedSecondaryTargetCode}
              disabled={busy}
              codePlaceholder="例如 BUYER_002"
              onCodeChange={(code) => handleDualTargetCodeChange(1, code)}
            />
          </div>
        ) : (
          <AutomationInstanceSelector
            title="传入实例"
            mode="manual"
            modes={["manual"]}
            profiles={profiles}
            selectedCode={selectedTargetCode}
            disabled={busy}
            codePlaceholder="例如 BUYER_001"
            onCodeChange={handleTargetCodeChange}
          />
        )}

        <AutomationScriptPublicApiBodyExamples
          busy={busy}
          resolvedConfig={resolvedConfig}
          requestExampleFallback={requestExampleFallback}
          responseExampleFallback={responseExampleFallback}
          requestBodyError={requestBodyError}
          responseBodyError={responseBodyError}
          updateConfig={updateConfig}
        />
      </div>
    </Modal>
  );
}
