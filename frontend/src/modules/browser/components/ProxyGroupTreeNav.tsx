import { useEffect, useMemo, useState } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  FolderPlus,
  Layers3,
  Pencil,
  Plus,
  PanelLeftClose,
  PanelLeftOpen,
  Trash2,
  Waypoints,
} from 'lucide-react'
import { Button, ConfirmModal, Input, Modal, toast } from '../../../shared/components'
import type { BrowserProxyGroupInput, BrowserProxyGroupWithCount } from '../types'
import { createProxyGroup, deleteProxyGroup, updateProxyGroup } from '../api'

interface ProxyGroupTreeNavProps {
  groups: BrowserProxyGroupWithCount[]
  selectedGroupId: string
  totalCount: number
  ungroupedCount: number
  onSelectGroup: (groupId: string) => void
  onRefresh: () => void | Promise<void>
  collapsed: boolean
  onToggleCollapsed: () => void
}

interface TreeNode extends BrowserProxyGroupWithCount {
  children: TreeNode[]
  subtreeCount: number
  level: number
}

function buildTree(groups: BrowserProxyGroupWithCount[]): TreeNode[] {
  const byId = new Map<string, TreeNode>()
  groups.forEach(group => byId.set(group.groupId, {
    ...group,
    children: [],
    subtreeCount: group.proxyCount,
    level: 0,
  }))
  const roots: TreeNode[] = []
  groups.forEach(group => {
    const node = byId.get(group.groupId)!
    const parent = group.parentId ? byId.get(group.parentId) : undefined
    if (parent && parent.groupId !== node.groupId) parent.children.push(node)
    else roots.push(node)
  })
  const finalize = (nodes: TreeNode[], level = 0): number => {
    nodes.sort((a, b) => a.sortOrder - b.sortOrder || a.groupName.localeCompare(b.groupName, 'zh-CN'))
    let sum = 0
    nodes.forEach(node => {
      node.level = level
      node.subtreeCount = node.proxyCount + finalize(node.children, level + 1)
      sum += node.subtreeCount
    })
    return sum
  }
  finalize(roots)
  return roots
}

function collectAncestors(groups: BrowserProxyGroupWithCount[], groupId: string) {
  const byId = new Map(groups.map(group => [group.groupId, group]))
  const ancestors: string[] = []
  const seen = new Set<string>()
  let current = byId.get(groupId)
  while (current?.parentId && !seen.has(current.parentId)) {
    seen.add(current.parentId)
    ancestors.push(current.parentId)
    current = byId.get(current.parentId)
  }
  return ancestors
}

export function ProxyGroupTreeNav({
  groups,
  selectedGroupId,
  totalCount,
  ungroupedCount,
  onSelectGroup,
  onRefresh,
  collapsed,
  onToggleCollapsed,
}: ProxyGroupTreeNavProps) {
  const tree = useMemo(() => buildTree(groups), [groups])
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [editor, setEditor] = useState<{
    mode: 'create' | 'rename'
    group: BrowserProxyGroupWithCount | null
    parentId: string
  } | null>(null)
  const [name, setName] = useState('')
  const [pendingDelete, setPendingDelete] = useState<BrowserProxyGroupWithCount | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    setExpanded(previous => {
      const next = new Set(previous)
      tree.forEach(root => next.add(root.groupId))
      if (selectedGroupId && selectedGroupId !== '__all__') {
        collectAncestors(groups, selectedGroupId).forEach(id => next.add(id))
      }
      return next
    })
  }, [groups, selectedGroupId, tree])

  const toggle = (groupId: string) => {
    setExpanded(previous => {
      const next = new Set(previous)
      if (next.has(groupId)) next.delete(groupId)
      else next.add(groupId)
      return next
    })
  }

  const openCreate = (parentId = '') => {
    setName('')
    setEditor({ mode: 'create', group: null, parentId })
  }

  const openRename = (group: BrowserProxyGroupWithCount) => {
    setName(group.groupName)
    setEditor({ mode: 'rename', group, parentId: group.parentId })
  }

  const saveEditor = async () => {
    const groupName = name.trim()
    if (!editor || !groupName) return
    setBusy(true)
    try {
      if (editor.mode === 'create') {
        const siblings = groups.filter(group => group.parentId === editor.parentId)
        const sortOrder = siblings.reduce((max, group) => Math.max(max, group.sortOrder), -1) + 1
        const input: BrowserProxyGroupInput = { groupName, parentId: editor.parentId, sortOrder }
        const created = await createProxyGroup(input)
        if (editor.parentId) setExpanded(previous => new Set(previous).add(editor.parentId))
        if (created?.groupId) onSelectGroup(created.groupId)
        toast.success(editor.parentId ? '子分组已创建' : '代理分组已创建')
      } else if (editor.group) {
        await updateProxyGroup(editor.group.groupId, {
          groupName,
          parentId: editor.group.parentId,
          sortOrder: editor.group.sortOrder,
        })
        toast.success('代理分组已重命名')
      }
      setEditor(null)
      await onRefresh()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存代理分组失败')
    } finally {
      setBusy(false)
    }
  }

  const performDelete = async () => {
    if (!pendingDelete) return
    setBusy(true)
    try {
      await deleteProxyGroup(pendingDelete.groupId)
      if (selectedGroupId === pendingDelete.groupId) onSelectGroup(pendingDelete.parentId || '__all__')
      toast.success('分组已删除，代理和子分组已移至上一级')
      setPendingDelete(null)
      await onRefresh()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除代理分组失败')
    } finally {
      setBusy(false)
    }
  }

  const renderNode = (node: TreeNode) => {
    const hasChildren = node.children.length > 0
    const isExpanded = expanded.has(node.groupId)
    const isSelected = selectedGroupId === node.groupId
    return (
      <div key={node.groupId} className={node.level > 0 ? 'ml-4 border-l border-[var(--color-border-muted)] pl-2' : ''}>
        <div className={`group flex min-h-10 items-center gap-1 rounded-xl border px-1.5 transition-all ${
          isSelected
            ? 'border-[var(--color-accent)]/30 bg-[var(--color-accent)]/10 text-[var(--color-accent)] shadow-sm'
            : 'border-transparent text-[var(--color-text-secondary)] hover:border-[var(--color-border-muted)] hover:bg-[var(--color-bg-secondary)] hover:text-[var(--color-text-primary)]'
        }`}>
          <button
            type="button"
            className="flex h-7 w-6 shrink-0 items-center justify-center rounded-md hover:bg-black/5 disabled:opacity-30"
            disabled={!hasChildren}
            onClick={() => hasChildren && toggle(node.groupId)}
            title={hasChildren ? (isExpanded ? '收起子分组' : '展开子分组') : '没有子分组'}
          >
            {hasChildren ? (isExpanded ? <ChevronDown className="h-3.5 w-3.5" /> : <ChevronRight className="h-3.5 w-3.5" />) : null}
          </button>
          <button type="button" className="flex min-w-0 flex-1 items-center gap-2 py-2 text-left" onClick={() => onSelectGroup(node.groupId)}>
            <span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-lg ${isSelected ? 'bg-[var(--color-accent)]/15' : 'bg-sky-500/10'}`}>
              {isExpanded && hasChildren ? <FolderOpen className="h-4 w-4 text-sky-500" /> : <Folder className="h-4 w-4 text-sky-500" />}
            </span>
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-medium">{node.groupName}</span>
              {hasChildren && <span className="block text-[10px] text-[var(--color-text-muted)]">{node.children.length} 个子分组</span>}
            </span>
            <span className="rounded-full bg-[var(--color-bg-muted)] px-2 py-0.5 text-[11px] tabular-nums text-[var(--color-text-muted)]" title={`本组 ${node.proxyCount} 个，含子组共 ${node.subtreeCount} 个`}>
              {node.subtreeCount}
            </span>
          </button>
          <div className="hidden shrink-0 items-center gap-0.5 pr-1 group-hover:flex">
            <button type="button" className="rounded-md p-1 text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-accent)]" onClick={() => openCreate(node.groupId)} title="新建子分组"><FolderPlus className="h-3.5 w-3.5" /></button>
            <button type="button" className="rounded-md p-1 text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-accent)]" onClick={() => openRename(node)} title="重命名"><Pencil className="h-3.5 w-3.5" /></button>
            <button type="button" className="rounded-md p-1 text-[var(--color-text-muted)] hover:bg-red-500/10 hover:text-red-500" onClick={() => setPendingDelete(node)} title="删除"><Trash2 className="h-3.5 w-3.5" /></button>
          </div>
        </div>
        {hasChildren && isExpanded && <div className="mt-1 space-y-1">{node.children.map(renderNode)}</div>}
      </div>
    )
  }

  const parentName = editor?.parentId ? groups.find(group => group.groupId === editor.parentId)?.groupName : ''

  if (collapsed) {
    return (
      <aside className="flex w-14 justify-self-start flex-col items-center gap-2 rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] p-2 shadow-sm lg:sticky lg:top-0">
        <button type="button" onClick={onToggleCollapsed} className="flex h-9 w-9 items-center justify-center rounded-xl text-sky-500 transition-colors hover:bg-sky-500/10" title="展开代理分组" aria-label="展开代理分组">
          <PanelLeftOpen className="h-4 w-4" />
        </button>
        <div className="h-px w-7 bg-[var(--color-border-muted)]" />
        <button type="button" onClick={() => onSelectGroup('__all__')} className={selectedGroupId === '__all__' ? 'flex h-9 w-9 items-center justify-center rounded-xl bg-[var(--color-accent)]/10 text-[var(--color-accent)]' : 'flex h-9 w-9 items-center justify-center rounded-xl text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg-secondary)]'} title={'全部代理（' + totalCount + '）'}>
          <Layers3 className="h-4 w-4" />
        </button>
        <button type="button" onClick={() => onSelectGroup('')} className={selectedGroupId === '' ? 'flex h-9 w-9 items-center justify-center rounded-xl bg-[var(--color-accent)]/10 text-[var(--color-accent)]' : 'flex h-9 w-9 items-center justify-center rounded-xl text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg-secondary)]'} title={'未分组（' + ungroupedCount + '）'}>
          <Folder className="h-4 w-4" />
        </button>
      </aside>
    )
  }

  return (
    <aside className="flex min-h-[420px] flex-col overflow-hidden rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] shadow-sm lg:sticky lg:top-0 lg:max-h-[calc(100vh-190px)]">
      <div className="border-b border-[var(--color-border-muted)] bg-gradient-to-br from-sky-500/10 via-transparent to-transparent p-4">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-sky-500/10 text-sky-500"><Waypoints className="h-5 w-5" /></span>
            <div>
              <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">代理分组</h2>
              <p className="text-[11px] text-[var(--color-text-muted)]">文件夹层级管理</p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <button type="button" onClick={onToggleCollapsed} className="flex h-8 w-8 items-center justify-center rounded-lg text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg-secondary)] hover:text-sky-500" title="收起分组侧栏" aria-label="收起分组侧栏"><PanelLeftClose className="h-4 w-4" /></button>
            <Button size="sm" variant="secondary" onClick={() => openCreate()} title="新建主分组"><Plus className="h-4 w-4" /></Button>
          </div>
        </div>
      </div>

      <div className="space-y-1 border-b border-[var(--color-border-muted)] p-2">
        <button type="button" onClick={() => onSelectGroup('__all__')} className={`flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-sm transition-colors ${selectedGroupId === '__all__' ? 'bg-[var(--color-accent)]/10 font-medium text-[var(--color-accent)]' : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-secondary)]'}`}>
          <Layers3 className="h-4 w-4" /><span className="flex-1 text-left">全部代理</span><span className="text-xs tabular-nums text-[var(--color-text-muted)]">{totalCount}</span>
        </button>
        <button type="button" onClick={() => onSelectGroup('')} className={`flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-sm transition-colors ${selectedGroupId === '' ? 'bg-[var(--color-accent)]/10 font-medium text-[var(--color-accent)]' : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-secondary)]'}`}>
          <Folder className="h-4 w-4 text-slate-400" /><span className="flex-1 text-left">未分组</span><span className="text-xs tabular-nums text-[var(--color-text-muted)]">{ungroupedCount}</span>
        </button>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-2">
        {tree.length > 0 ? <div className="space-y-1">{tree.map(renderNode)}</div> : (
          <button type="button" onClick={() => openCreate()} className="flex w-full flex-col items-center rounded-xl border border-dashed border-[var(--color-border-default)] px-4 py-8 text-center text-[var(--color-text-muted)] hover:border-sky-500/50 hover:bg-sky-500/5">
            <FolderPlus className="mb-2 h-7 w-7" /><span className="text-sm font-medium">还没有代理分组</span><span className="mt-1 text-xs">创建主分组后可继续添加子分组</span>
          </button>
        )}
      </div>

      <Modal
        open={editor !== null}
        onClose={() => !busy && setEditor(null)}
        title={editor?.mode === 'rename' ? '重命名代理分组' : editor?.parentId ? '新建代理子分组' : '新建代理主分组'}
        width="420px"
        footer={<><Button variant="secondary" onClick={() => setEditor(null)} disabled={busy}>取消</Button><Button onClick={saveEditor} loading={busy} disabled={!name.trim()}>保存</Button></>}
      >
        <div className="space-y-3">
          {parentName && <div className="rounded-lg bg-[var(--color-bg-secondary)] px-3 py-2 text-xs text-[var(--color-text-muted)]">上级分组：<span className="font-medium text-[var(--color-text-primary)]">{parentName}</span></div>}
          <Input value={name} onChange={event => setName(event.target.value)} onKeyDown={event => { if (event.key === 'Enter') void saveEditor() }} autoFocus placeholder="请输入分组名称" />
        </div>
      </Modal>

      <ConfirmModal
        open={pendingDelete !== null}
        onClose={() => !busy && setPendingDelete(null)}
        onConfirm={() => { void performDelete() }}
        title="删除代理分组"
        content={pendingDelete ? `确定删除“${pendingDelete.groupName}”吗？其中的代理和子分组会自动移动到上一级。` : ''}
        confirmText="删除"
        danger
      />
    </aside>
  )
}
