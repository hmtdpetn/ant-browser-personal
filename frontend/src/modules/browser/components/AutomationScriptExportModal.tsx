import { useEffect, useState } from "react";
import { Button, Modal } from "../../../shared/components";

export type AutomationScriptExportFormat = "json" | "zip" | "directory";

interface AutomationScriptExportModalProps {
  open: boolean;
  busy: boolean;
  onClose: () => void;
  onSubmit: (format: AutomationScriptExportFormat) => void;
}

const EXPORT_OPTIONS: Array<{
  value: AutomationScriptExportFormat;
  title: string;
  description: string;
}> = [
  {
    value: "json",
    title: "JSON 模板",
    description: "适合复制、粘贴、文本分发和远程单文件导入。",
  },
  {
    value: "zip",
    title: "ZIP 脚本包",
    description: "适合对外分发和备份，多文件脚本会按真实目录结构导出。",
  },
  {
    value: "directory",
    title: "目录脚本包",
    description: "适合本地维护、人工查看和继续提交到 Git。",
  },
];

export function AutomationScriptExportModal({
  open,
  busy,
  onClose,
  onSubmit,
}: AutomationScriptExportModalProps) {
  const [selectedFormat, setSelectedFormat] =
    useState<AutomationScriptExportFormat>("zip");

  useEffect(() => {
    if (open) {
      setSelectedFormat("zip");
    }
  }, [open]);

  return (
    <Modal
      open={open}
      onClose={busy ? () => undefined : onClose}
      title="导出脚本"
      width="560px"
      footer={
        <>
          <Button variant="secondary" onClick={onClose} disabled={busy}>
            取消
          </Button>
          <Button onClick={() => onSubmit(selectedFormat)} loading={busy}>
            导出
          </Button>
        </>
      }
    >
      <div className="space-y-3">
        {EXPORT_OPTIONS.map((option) => {
          const active = option.value === selectedFormat;
          return (
            <button
              key={option.value}
              type="button"
              className={`w-full rounded-2xl border px-4 py-4 text-left transition-colors ${
                active
                  ? "border-[var(--color-border-strong)] bg-[var(--color-bg-secondary)]"
                  : "border-[var(--color-border-default)] bg-[var(--color-bg-surface)] hover:border-[var(--color-border-strong)]"
              }`}
              onClick={() => setSelectedFormat(option.value)}
              disabled={busy}
            >
              <div className="text-sm font-semibold text-[var(--color-text-primary)]">
                {option.title}
              </div>
              <div className="mt-1 text-sm leading-6 text-[var(--color-text-secondary)]">
                {option.description}
              </div>
            </button>
          );
        })}
      </div>
    </Modal>
  );
}
