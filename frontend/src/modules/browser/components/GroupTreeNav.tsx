import { useEffect, useMemo, useState } from 'react'
import {
  ChevronDown,
  ChevronRight,
  Folder,
  FolderOpen,
  FolderPlus,
  Folders,
  Pencil,
  Plus,
  PanelLeftClose,
  PanelLeftOpen,
  Trash2,
  Users,
} from 'lucide-react'
import { Button, ConfirmModal, Input, Modal, toast } from '../../../shared/components'
import type { BrowserGroupInput, BrowserGroupWithCount } from '../types'
import { createGroup, deleteGroup, updateGroup } from '../api'

interface GroupTreeNavProps {
  groups: BrowserGroupWithCount[]
  selectedGroupId: string | null
  totalCount: number
  ungroupedCount: number
  onSelectGroup: (groupId: string | null) => void
  onRefresh: () => void | Promise<void>
  onOpenManager?: () => void
  collapsed: boolean
  onToggleCollapsed: () => void
}

interface TreeNode extends BrowserGroupWithCount {
  children: TreeNode[]
  subtreeCount: number
  level: number
}

function buildTree(groups: BrowserGroupWithCount[]): TreeNode[] {
  const map = new Map<string, TreeNode>()
  groups.forEach(group => map.set(group.groupId, { ...group, children: [], subtreeCount: group.instanceCount, level: 0 }))

  const roots: TreeNode[] = []
  groups.forEach(group => {
    const node = map.get(group.groupId)!
    const parent = group.parentId ? map.get(group.parentId) : undefined
    if (parent && parent.groupId !== node.groupId) {
      node.level = parent.level + 1
      parent.children.push(node)
    } else {
      roots.push(node)
    }
  })

  const finalize = (nodes: TreeNode[], level = 0): number => {
    nodes.sort((a, b) => a.sortOrder - b.sortOrder || a.groupName.localeCompare(b.groupName, 'zh-CN'))
    let sum = 0
    nodes.forEach(node => {
      node.level = level
      node.subtreeCount = node.instanceCount + finalize(node.children, level + 1)
      sum += node.subtreeCount
    })
    return sum
  }
  finalize(roots)
  return roots
}

function collectAncestors(groups: BrowserGroupWithCount[], groupId: string): string[] {
  const byId = new Map(groups.map(group => [group.groupId, group]))
  const result: string[] = []
  const visited = new Set<string>()
  let current = byId.get(groupId)
  while (current?.parentId && !visited.has(current.parentId)) {
    visited.add(current.parentId)
    result.push(current.parentId)
    current = byId.get(current.parentId)
  }
  return result
}

export function GroupTreeNav({
  groups,
  selectedGroupId,
  totalCount,
  ungroupedCount,
  onSelectGroup,
  onRefresh,
  onOpenManager,
  collapsed,
  onToggleCollapsed,
}: GroupTreeNavProps) {
  const tree = useMemo(() => buildTree(groups), [groups])
  const [expanded, setExpanded] = useState<Set<string>>(new Set())
  const [editor, setEditor] = useState<{ mode: 'create' | 'rename'; group: BrowserGroupWithCount | null; parentId: string } | null>(null)
  const [name, setName] = useState('')
  const [pendingDelete, setPendingDelete] = useState<BrowserGroupWithCount | null>(null)
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    setExpanded(previous => {
      const next = new Set(previous)
      tree.forEach(root => next.add(root.groupId))
      if (selectedGroupId && selectedGroupId !== '__ungrouped__') {
        collectAncestors(groups, selectedGroupId).forEach(id => next.add(id))
      }
      return next
    })
  }, [groups, selectedGroupId, tree])

  const toggle = (groupId: string) => {
    setExpanded(previous => {
      const next = new Set(previous)
      next.has(groupId) ? next.delete(groupId) : next.add(groupId)
      return next
    })
  }

  const openCreate = (parentId = '') => {
    setName('')
    setEditor({ mode: 'create', group: null, parentId })
  }

  const openRename = (group: BrowserGroupWithCount) => {
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
        const input: BrowserGroupInput = { groupName, parentId: editor.parentId, sortOrder }
        const created = await createGroup(input)
        if (editor.parentId) setExpanded(previous => new Set(previous).add(editor.parentId))
        if (created?.groupId) onSelectGroup(created.groupId)
        toast.success(editor.parentId ? '子分组已创建' : '主分组已创建')
      } else if (editor.group) {
        await updateGroup(editor.group.groupId, {
          groupName,
          parentId: editor.group.parentId,
          sortOrder: editor.group.sortOrder,
        })
        toast.success('分组已重命名')
      }
      setEditor(null)
      await onRefresh()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '保存分组失败')
    } finally {
      setBusy(false)
    }
  }

  const performDelete = async () => {
    if (!pendingDelete) return
    setBusy(true)
    try {
      await deleteGroup(pendingDelete.groupId)
      if (selectedGroupId === pendingDelete.groupId) onSelectGroup(pendingDelete.parentId || null)
      toast.success('分组已删除，实例和子分组已移至上一级')
      setPendingDelete(null)
      await onRefresh()
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '删除分组失败')
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
        <div
          className={`group flex min-h-10 items-center gap-1 rounded-xl border px-1.5 transition-all ${
            isSelected
              ? 'border-[var(--color-accent)]/30 bg-[var(--color-accent)]/10 text-[var(--color-accent)] shadow-sm'
              : 'border-transparent text-[var(--color-text-secondary)] hover:border-[var(--color-border-muted)] hover:bg-[var(--color-bg-secondary)] hover:text-[var(--color-text-primary)]'
          }`}
        >
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
            <span className={`flex h-7 w-7 shrink-0 items-center justify-center rounded-lg ${isSelected ? 'bg-[var(--color-accent)]/15' : 'bg-amber-500/10'}`}>
              {isExpanded && hasChildren ? <FolderOpen className="h-4 w-4 text-amber-500" /> : <Folder className="h-4 w-4 text-amber-500" />}
            </span>
            <span className="min-w-0 flex-1">
              <span className="block truncate text-sm font-medium">{node.groupName}</span>
              {hasChildren && <span className="block text-[10px] text-[var(--color-text-muted)]">{node.children.length} 个子分组</span>}
            </span>
            <span className="rounded-full bg-[var(--color-bg-muted)] px-2 py-0.5 text-[11px] tabular-nums text-[var(--color-text-muted)]" title={`本组 ${node.instanceCount} 个，包含子组 ${node.subtreeCount} 个`}>
              {node.subtreeCount}
            </span>
          </button>
          <div className="hidden shrink-0 items-center group-hover:flex">
            <button type="button" className="rounded-md p-1 text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-accent)]" onClick={() => openCreate(node.groupId)} title="新建子分组"><FolderPlus className="h-3.5 w-3.5" /></button>
            <button type="button" className="rounded-md p-1 text-[var(--color-text-muted)] hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-accent)]" onClick={() => openRename(node)} title="重命名"><Pencil className="h-3.5 w-3.5" /></button>
            <button type="button" className="rounded-md p-1 text-[var(--color-text-muted)] hover:bg-red-500/10 hover:text-red-500" onClick={() => setPendingDelete(node)} title="删除"><Trash2 className="h-3.5 w-3.5" /></button>
          </div>
        </div>
        {hasChildren && isExpanded && <div className="mt-1 space-y-1">{node.children.map(renderNode)}</div>}
      </div>
    )
  }

  const selectedName = editor?.parentId ? groups.find(group => group.groupId === editor.parentId)?.groupName : ''

  if (collapsed) {
    return (
      <aside className="flex w-14 justify-self-start flex-col items-center gap-2 rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] p-2 shadow-sm lg:sticky lg:top-0">
        <button type="button" onClick={onToggleCollapsed} className="flex h-9 w-9 items-center justify-center rounded-xl text-[var(--color-accent)] transition-colors hover:bg-[var(--color-accent)]/10" title="展开实例分组" aria-label="展开实例分组">
          <PanelLeftOpen className="h-4 w-4" />
        </button>
        <div className="h-px w-7 bg-[var(--color-border-muted)]" />
        <button type="button" onClick={() => onSelectGroup(null)} className={selectedGroupId === null ? 'flex h-9 w-9 items-center justify-center rounded-xl bg-[var(--color-accent)]/10 text-[var(--color-accent)]' : 'flex h-9 w-9 items-center justify-center rounded-xl text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg-secondary)]'} title={'全部实例（' + totalCount + '）'}>
          <Users className="h-4 w-4" />
        </button>
        <button type="button" onClick={() => onSelectGroup('__ungrouped__')} className={selectedGroupId === '__ungrouped__' ? 'flex h-9 w-9 items-center justify-center rounded-xl bg-[var(--color-accent)]/10 text-[var(--color-accent)]' : 'flex h-9 w-9 items-center justify-center rounded-xl text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg-secondary)]'} title={'未分组（' + ungroupedCount + '）'}>
          <Folder className="h-4 w-4" />
        </button>
      </aside>
    )
  }

  return (
    <aside className="flex min-h-[420px] flex-col overflow-hidden rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] shadow-sm lg:sticky lg:top-0 lg:max-h-[calc(100vh-190px)]">
      <div className="border-b border-[var(--color-border-muted)] bg-gradient-to-br from-[var(--color-accent)]/10 via-transparent to-transparent p-4">
        <div className="flex items-center justify-between gap-2">
          <div className="flex items-center gap-2">
            <span className="flex h-9 w-9 items-center justify-center rounded-xl bg-[var(--color-accent)]/12 text-[var(--color-accent)]"><Folders className="h-5 w-5" /></span>
            <div>
              <h2 className="text-sm font-semibold text-[var(--color-text-primary)]">实例分组</h2>
              <p className="text-[11px] text-[var(--color-text-muted)]">主分组 / 子分组</p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <button type="button" onClick={onToggleCollapsed} className="flex h-8 w-8 items-center justify-center rounded-lg text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg-secondary)] hover:text-[var(--color-accent)]" title="收起分组侧栏" aria-label="收起分组侧栏"><PanelLeftClose className="h-4 w-4" /></button>
            <Button size="sm" variant="secondary" onClick={() => openCreate()} title="新建主分组"><Plus className="h-4 w-4" /></Button>
          </div>
        </div>
      </div>

      <div className="space-y-1 border-b border-[var(--color-border-muted)] p-2">
        <button type="button" onClick={() => onSelectGroup(null)} className={`flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-sm transition-colors ${selectedGroupId === null ? 'bg-[var(--color-accent)]/10 font-medium text-[var(--color-accent)]' : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-secondary)]'}`}>
          <Users className="h-4 w-4" /><span className="flex-1 text-left">全部实例</span><span className="text-xs tabular-nums text-[var(--color-text-muted)]">{totalCount}</span>
        </button>
        <button type="button" onClick={() => onSelectGroup('__ungrouped__')} className={`flex w-full items-center gap-2 rounded-xl px-3 py-2.5 text-sm transition-colors ${selectedGroupId === '__ungrouped__' ? 'bg-[var(--color-accent)]/10 font-medium text-[var(--color-accent)]' : 'text-[var(--color-text-secondary)] hover:bg-[var(--color-bg-secondary)]'}`}>
          <Folder className="h-4 w-4 text-slate-400" /><span className="flex-1 text-left">未分组</span><span className="text-xs tabular-nums text-[var(--color-text-muted)]">{ungroupedCount}</span>
        </button>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto p-2">
        {tree.length > 0 ? <div className="space-y-1">{tree.map(renderNode)}</div> : (
          <button type="button" onClick={() => openCreate()} className="flex w-full flex-col items-center rounded-xl border border-dashed border-[var(--color-border-default)] px-4 py-8 text-center text-[var(--color-text-muted)] hover:border-[var(--color-accent)]/50 hover:bg-[var(--color-accent)]/5">
            <FolderPlus className="mb-2 h-7 w-7" /><span className="text-sm font-medium">还没有实例分组</span><span className="mt-1 text-xs">点击这里新建主分组</span>
          </button>
        )}
      </div>

      {onOpenManager && (
        <div className="border-t border-[var(--color-border-muted)] p-2">
          <Button variant="secondary" size="sm" className="w-full" onClick={onOpenManager}>打开完整分组管理</Button>
        </div>
      )}

      <Modal
        open={editor !== null}
        onClose={() => !busy && setEditor(null)}
        title={editor?.mode === 'rename' ? '重命名分组' : editor?.parentId ? '新建子分组' : '新建主分组'}
        width="420px"
        footer={<><Button variant="secondary" onClick={() => setEditor(null)} disabled={busy}>取消</Button><Button onClick={saveEditor} loading={busy} disabled={!name.trim()}>保存</Button></>}
      >
        <div className="space-y-3">
          {selectedName && <div className="rounded-lg bg-[var(--color-bg-secondary)] px-3 py-2 text-xs text-[var(--color-text-muted)]">上级分组：<span className="font-medium text-[var(--color-text-primary)]">{selectedName}</span></div>}
          <Input value={name} onChange={event => setName(event.target.value)} onKeyDown={event => { if (event.key === 'Enter') void saveEditor() }} autoFocus placeholder="请输入分组名称" />
        </div>
      </Modal>

      <ConfirmModal
        open={pendingDelete !== null}
        onClose={() => !busy && setPendingDelete(null)}
        onConfirm={() => { void performDelete() }}
        title="删除分组"
        content={pendingDelete ? `确定删除“${pendingDelete.groupName}”吗？其中的实例和子分组会自动移动到上一级。` : ''}
        confirmText="删除"
        danger
      />
    </aside>
  )
}
