import { Link } from 'react-router-dom'
import { Activity, CheckCircle, ChevronRight, ChevronUp, Edit2, FileText, Gift, LayoutGrid, List, Play, Plus, RefreshCw, Sliders, Square, Star, Trash2, XCircle } from 'lucide-react'

import { Button, Card, FormItem, Input, Modal, StatCard, Switch, Table, Textarea } from '../../../shared/components'
import type { TableColumn } from '../../../shared/components/Table'

import type { BrowserCore, BrowserCoreInput, BrowserGroupWithCount, BrowserProxy, BrowserSettings } from '../types'
import { InstanceFilterBar } from './InstanceFilterBar'
import type { InstanceFilters } from './InstanceFilterBar'

export type BrowserViewMode = 'card' | 'table'

interface BrowserListHeaderProps {
  profileCount: number
  filteredProfileCount: number
  runningCount: number
  headerCollapsed: boolean
  viewMode: BrowserViewMode
  proxies: BrowserProxy[]
  cores: BrowserCore[]
  groups: BrowserGroupWithCount[]
  allTags: string[]
  filters: InstanceFilters
  onFiltersChange: (next: InstanceFilters) => void
  onToggleHeaderCollapsed: () => void
  onRefresh: () => void
  onOpenSettings: () => void
  onOpenExpandModal: () => void
  onViewModeChange: (next: BrowserViewMode) => void
}

export function BrowserListHeader({
  profileCount,
  filteredProfileCount,
  runningCount,
  headerCollapsed,
  viewMode,
  proxies,
  cores,
  groups,
  allTags,
  filters,
  onFiltersChange,
  onToggleHeaderCollapsed,
  onRefresh,
  onOpenSettings,
  onOpenExpandModal,
  onViewModeChange,
}: BrowserListHeaderProps) {
  return (
    <>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold text-[var(--color-text-primary)]">实例列表</h1>
          <p className="text-sm text-[var(--color-text-muted)] mt-1">
            当前配置总数 {profileCount}
            {filteredProfileCount !== profileCount && (
              <span className="ml-1 text-[var(--color-accent)]">（已筛选 {filteredProfileCount}）</span>
            )}
          </p>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" size="sm" onClick={onToggleHeaderCollapsed}>
            {headerCollapsed ? <ChevronRight className="w-4 h-4" /> : <ChevronUp className="w-4 h-4" />}
            {headerCollapsed ? '展开面板' : '收起面板'}
          </Button>
          <Button variant="secondary" size="sm" onClick={onRefresh}>
            <RefreshCw className="w-4 h-4" />刷新
          </Button>
          <Button variant="secondary" size="sm" onClick={onOpenSettings}>
            <Sliders className="w-4 h-4" />基础配置
          </Button>
          <Button
            variant="secondary"
            size="sm"
            onClick={onOpenExpandModal}
            className="text-[var(--color-primary)] border-[var(--color-primary)] hover:bg-[var(--color-primary)]/10"
          >
            <Gift className="w-4 h-4" />扩容实例
          </Button>
          <div className="flex items-center bg-[var(--color-bg-secondary)] rounded-md border border-[var(--color-border-default)] p-0.5 ml-2">
            <button
              className={`p-1.5 rounded text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors ${viewMode === 'card' ? 'bg-[var(--color-bg-surface)] shadow-sm text-[var(--color-accent)]' : ''}`}
              onClick={() => onViewModeChange('card')}
              title="卡片视图"
            >
              <LayoutGrid className="w-4 h-4" />
            </button>
            <button
              className={`p-1.5 rounded text-[var(--color-text-muted)] hover:text-[var(--color-text-primary)] transition-colors ${viewMode === 'table' ? 'bg-[var(--color-bg-surface)] shadow-sm text-[var(--color-accent)]' : ''}`}
              onClick={() => onViewModeChange('table')}
              title="表格视图"
            >
              <List className="w-4 h-4" />
            </button>
          </div>
          <span className="w-px h-4 bg-[var(--color-border-muted)] mx-1 self-center"></span>
          <Link to="/browser/edit/new">
            <Button size="sm">
              <Play className="w-4 h-4" />新建配置
            </Button>
          </Link>
        </div>
      </div>

      {!headerCollapsed && (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
            <StatCard title="配置总数" value={`${profileCount}`} icon={<FileText className="w-5 h-5" />} />
            <StatCard title="运行中实例" value={`${runningCount}`} icon={<Activity className="w-5 h-5" />} />
            <StatCard title="停止实例" value={`${profileCount - runningCount}`} icon={<Square className="w-5 h-5 text-gray-400" />} />
          </div>

          <InstanceFilterBar
            filters={filters}
            onChange={onFiltersChange}
            proxies={proxies}
            cores={cores}
            allTags={allTags}
            groups={groups}
          />
        </>
      )}
    </>
  )
}

interface BrowserListSettingsModalProps {
  open: boolean
  settings: BrowserSettings
  fingerprintText: string
  launchText: string
  startUrlsText: string
  savingSettings: boolean
  cores: BrowserCore[]
  onClose: () => void
  onSave: () => void
  onSettingsChange: (patch: Partial<BrowserSettings>) => void
  onFingerprintTextChange: (next: string) => void
  onLaunchTextChange: (next: string) => void
  onStartUrlsTextChange: (next: string) => void
  onAddCore: () => void
  onEditCore: (core: BrowserCore) => void
  onDeleteCore: (coreId: string) => void
  onSetDefaultCore: (coreId: string) => void
}

export function BrowserListSettingsModal({
  open,
  settings,
  fingerprintText,
  launchText,
  startUrlsText,
  savingSettings,
  cores,
  onClose,
  onSave,
  onSettingsChange,
  onFingerprintTextChange,
  onLaunchTextChange,
  onStartUrlsTextChange,
  onAddCore,
  onEditCore,
  onDeleteCore,
  onSetDefaultCore,
}: BrowserListSettingsModalProps) {
  const coreColumns: TableColumn<BrowserCore>[] = [
    { key: 'coreName', title: '名称' },
    { key: 'corePath', title: '路径' },
    {
      key: 'isDefault',
      title: '默认',
      render: (value) => (value ? <Star className="w-4 h-4 text-yellow-500 fill-yellow-500" /> : null),
    },
    {
      key: 'actions',
      title: '操作',
      align: 'right',
      render: (_, record) => (
        <div className="flex justify-end gap-1">
          {!record.isDefault && (
            <Button size="sm" variant="ghost" onClick={() => onSetDefaultCore(record.coreId)} title="设为默认">
              <Star className="w-4 h-4" />
            </Button>
          )}
          <Button size="sm" variant="ghost" onClick={() => onEditCore(record)} title="编辑">
            <Edit2 className="w-4 h-4" />
          </Button>
          <Button size="sm" variant="ghost" onClick={() => onDeleteCore(record.coreId)} title="删除">
            <Trash2 className="w-4 h-4" />
          </Button>
        </div>
      ),
    },
  ]

  return (
    <Modal
      open={open}
      onClose={onClose}
      title="基础配置"
      width="700px"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>取消</Button>
          <Button onClick={onSave} loading={savingSettings}>保存</Button>
        </>
      }
    >
      <div className="space-y-6">
        <div>
          <div className="flex items-center justify-between mb-2">
            <span className="text-sm font-medium text-[var(--color-text-primary)]">内核管理</span>
            <div className="flex gap-2">
              <Button size="sm" onClick={onAddCore}>
                <Plus className="w-4 h-4" />新增内核
              </Button>
            </div>
          </div>
          <Card padding="none">
            <Table columns={coreColumns} data={cores} rowKey="coreId" />
          </Card>
        </div>

        <FormItem label="用户数据根目录">
          <Input
            value={settings.userDataRoot}
            onChange={(event) => onSettingsChange({ userDataRoot: event.target.value })}
            placeholder="data"
          />
        </FormItem>
        <FormItem label="默认指纹参数（每行一个）">
          <Textarea
            value={fingerprintText}
            onChange={(event) => onFingerprintTextChange(event.target.value)}
            rows={3}
            placeholder="--fingerprint-brand=Chrome"
          />
        </FormItem>
        <FormItem label="默认启动参数（每行一个）">
          <Textarea
            value={launchText}
            onChange={(event) => onLaunchTextChange(event.target.value)}
            rows={3}
            placeholder="--disable-sync"
          />
        </FormItem>
        <FormItem label="默认启动页面（每行一个 URL）" hint="留空则启动时不再自动打开页面">
          <Textarea
            value={startUrlsText}
            onChange={(event) => onStartUrlsTextChange(event.target.value)}
            rows={4}
            placeholder="启动 URL"
          />
        </FormItem>
        <FormItem label="恢复上次关闭的标签页" hint="关闭后只打开默认启动页或空白页">
          <div className="flex items-center justify-between rounded-lg border border-[var(--color-border-default)] px-3 py-2">
            <div>
              <p className="text-sm text-[var(--color-text-primary)]">允许恢复旧 tab</p>
              <p className="text-xs text-[var(--color-text-muted)] mt-1">关闭后，下次启动会继续恢复之前的标签页和窗口。</p>
            </div>
            <Switch
              checked={settings.restoreLastSession}
              onChange={(checked) => onSettingsChange({ restoreLastSession: checked })}
            />
          </div>
        </FormItem>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <FormItem label="启动就绪超时（毫秒）" hint="默认 3000，慢机器可调到 5000-10000">
            <Input
              type="number"
              min={1000}
              step={500}
              value={settings.startReadyTimeoutMs}
              onChange={(event) =>
                onSettingsChange({
                  startReadyTimeoutMs: Math.max(1000, Number(event.target.value) || 3000),
                })
              }
              placeholder="3000"
            />
          </FormItem>
          <FormItem label="启动稳定窗口（毫秒）" hint="建议 1200-3000">
            <Input
              type="number"
              min={0}
              step={100}
              value={settings.startStableWindowMs}
              onChange={(event) =>
                onSettingsChange({
                  startStableWindowMs: Math.max(0, Number(event.target.value) || 1200),
                })
              }
              placeholder="1200"
            />
          </FormItem>
        </div>
      </div>
    </Modal>
  )
}

interface BrowserCoreEditorModalProps {
  open: boolean
  coreForm: BrowserCoreInput
  coreValidation: { valid: boolean; message: string } | null
  savingCore: boolean
  onClose: () => void
  onSave: () => void
  onValidate: () => void
  onCoreFormChange: (patch: Partial<BrowserCoreInput>) => void
}

export function BrowserCoreEditorModal({
  open,
  coreForm,
  coreValidation,
  savingCore,
  onClose,
  onSave,
  onValidate,
  onCoreFormChange,
}: BrowserCoreEditorModalProps) {
  return (
    <Modal
      open={open}
      onClose={onClose}
      title={coreForm.coreId ? '编辑内核' : '新增内核'}
      width="500px"
      footer={
        <>
          <Button variant="secondary" onClick={onClose}>取消</Button>
          <Button onClick={onSave} loading={savingCore}>保存</Button>
        </>
      }
    >
      <div className="space-y-4">
        <FormItem label="内核名称" required>
          <Input
            value={coreForm.coreName}
            onChange={(event) => onCoreFormChange({ coreName: event.target.value })}
            placeholder="Chrome 142"
          />
        </FormItem>
        <FormItem label="内核路径" required>
          <div className="flex gap-2">
            <Input
              value={coreForm.corePath}
              onChange={(event) => onCoreFormChange({ corePath: event.target.value })}
              placeholder="chrome 或 D:/browsers/chrome-120"
              className="flex-1"
            />
            <Button variant="secondary" onClick={onValidate}>验证</Button>
          </div>
          {coreValidation && (
            <div className={`flex items-center gap-1 mt-1 text-sm ${coreValidation.valid ? 'text-green-600' : 'text-red-600'}`}>
              {coreValidation.valid ? <CheckCircle className="w-4 h-4" /> : <XCircle className="w-4 h-4" />}
              {coreValidation.message}
            </div>
          )}
        </FormItem>
      </div>
    </Modal>
  )
}
