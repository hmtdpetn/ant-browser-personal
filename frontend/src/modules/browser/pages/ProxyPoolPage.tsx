import { useCallback, useEffect, useState } from 'react'
import { FolderInput, X } from 'lucide-react'
import { Button, ConfirmModal, toast } from '../../../shared/components'
import type { SortOrder } from '../../../shared/components/Table'
import type { BrowserProxy, BrowserProxyGroupWithCount, ProxyIPHealthResult } from '../types'
import { fetchBrowserProxies, fetchProxyGroups, moveProxiesToGroup, saveBrowserProxies } from '../api'
import {
  buildChainImportCandidate,
  buildDirectImportCandidate,
  createInitialChainImportForm,
  INITIAL_DIRECT_IMPORT_FORM,
  ensureBuiltinProxies,
  toChainImportForm,
  toDirectImportForm,
  toDisplayList,
  type ChainImportForm,
  type DirectImportForm,
  type ProxyDisplayInfo,
} from './proxyPool/helpers'
import {
  ProxyPoolEditModal,
  ProxyPoolIPHealthDetailModal,
  ProxyPoolImportModal,
  ProxyPoolPreviewModal,
  type ProxyEditFormValue,
} from './proxyPool/ProxyPoolModals'
import { ProxyPoolHeader } from './proxyPool/ProxyPoolHeader'
import { ProxyPoolTableCard } from './proxyPool/ProxyPoolTableCard'
import { ProxyPoolCheckSettingsModal } from './proxyPool/ProxyPoolCheckSettingsModal'
import { ProxyPoolUsageGuideModal } from './proxyPool/ProxyPoolUsageGuideModal'
import { ProxyCoreDownloadModal } from './proxyPool/ProxyCoreDownloadModal'
import { useProxySourceRefresh } from './proxyPool/useProxySourceRefresh'
import { useProxyImportFlow } from './proxyPool/useProxyImportFlow'
import { useProxyChecks } from './proxyPool/useProxyChecks'
import { useProxySelection } from './proxyPool/useProxySelection'
import { useProxyCheckSettingsModal } from './proxyPool/useProxyCheckSettingsModal'
import { useProxyGlobalRefreshConfig } from './proxyPool/useProxyGlobalRefreshConfig'
import { useProxyDeleteFlow } from './proxyPool/useProxyDeleteFlow'
import { useProxyCoreDownload } from './proxyPool/useProxyCoreDownload'
import { useProxyPoolFilter } from './proxyPool/useProxyPoolFilter'
import { ProxyGroupSelect } from '../components/ProxyGroupSelect'
import { ProxyGroupTreeNav } from '../components/ProxyGroupTreeNav'

export function ProxyPoolPage() {
  const [proxies, setProxies] = useState<BrowserProxy[]>([])
  const [displayList, setDisplayList] = useState<ProxyDisplayInfo[]>([])
  const [loading, setLoading] = useState(true)
  const [groupSidebarCollapsed, setGroupSidebarCollapsed] = useState(() => {
    try {
      return localStorage.getItem('ant-browser.proxy-groups.collapsed') === 'true'
    } catch {
      return false
    }
  })
  useEffect(() => {
    try {
      localStorage.setItem('ant-browser.proxy-groups.collapsed', String(groupSidebarCollapsed))
    } catch {
      // Local preference persistence is optional.
    }
  }, [groupSidebarCollapsed])
  const [usageGuideOpen, setUsageGuideOpen] = useState(false)
  const {
    coreDownloadOpen,
    coreDownloadType,
    setCoreDownloadType,
    coreDownloadGOOS,
    setCoreDownloadGOOS,
    coreDownloadGOARCH,
    setCoreDownloadGOARCH,
    coreDownloadProxy,
    setCoreDownloadProxy,
    coreDownloadProgress,
    currentCoreStatus,
    downloadCoreStatus,
    downloadCoreStatusLoading,
    loadBrowserSettings,
    handleStartCoreDownload,
    openCoreDownload,
    closeCoreDownload,
  } = useProxyCoreDownload()
  const [groups, setGroups] = useState<BrowserProxyGroupWithCount[]>([])

  const [filterProtocol, setFilterProtocol] = useState<string>('all')
  const [filterKeyword, setFilterKeyword] = useState('')
  const [filterGroupId, setFilterGroupId] = useState('__all__')
  const [moveTargetGroupId, setMoveTargetGroupId] = useState('')
  const [movingSelected, setMovingSelected] = useState(false)
  const [filterAvailableOnly, setFilterAvailableOnly] = useState(false)
  const [sortColumn, setSortColumn] = useState<string>('') // 默认不排序
  const [sortOrder, setSortOrder] = useState<SortOrder>(undefined)

  const {
    checkSettingsOpen,
    setCheckSettingsOpen,
    checkSettings,
    setCheckSettings,
    checkTargetsText,
    setCheckTargetsText,
    savingCheckSettings,
    openCheckSettings,
    saveCheckSettings,
  } = useProxyCheckSettingsModal()

  const {
    globalAutoRefreshEnabled,
    setGlobalAutoRefreshEnabled,
    globalRefreshInterval,
    globalRefreshIntervalM,
    setGlobalRefreshIntervalM,
  } = useProxyGlobalRefreshConfig()

  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editingProxy, setEditingProxy] = useState<BrowserProxy | null>(null)
  const [chainEditMode, setChainEditMode] = useState(false)
  const [chainEditForm, setChainEditForm] = useState<ChainImportForm>(() => createInitialChainImportForm())
  const [directEditMode, setDirectEditMode] = useState(false)
  const [directEditForm, setDirectEditForm] = useState<DirectImportForm>({ ...INITIAL_DIRECT_IMPORT_FORM })
  const [editForm, setEditForm] = useState<ProxyEditFormValue>({
    proxyName: '',
    proxyConfig: '',
    preferredKernel: 'auto',
    dnsServers: '',
    groupId: '',
  })
  const [saving, setSaving] = useState(false)
  const saveProxies = useCallback(async (list: BrowserProxy[]) => {
    await saveBrowserProxies(list)
    setProxies(list)
    setDisplayList(toDisplayList(list))
    const grps = await fetchProxyGroups()
    setGroups(grps)
  }, [])

  const {
    importModalOpen, setImportModalOpen, importMode, importUrl, importFetchProxyId, importUserAgentPreset, importCustomUserAgent, importUserAgentFallback, importResolvedUrl, importText,
    importDnsServers, importNamePrefix, importGroupId, chainImportText, directImportText,
    chainImportForm, directImportForm, previewModalOpen, setPreviewModalOpen, previewList, removedPreviewProxyNames,
    importing, fetchingImportUrl, canParseImport, setImportText, setImportDnsServers,
    setImportNamePrefix, setImportGroupId, setImportFetchProxyId, setImportUserAgentPreset, setImportCustomUserAgent, setImportUserAgentFallback, setChainImportText, setDirectImportText,
    setChainImportForm, setDirectImportForm, handleRemovePreviewProxy, updateChainImportHop,
    handleImportModeChange, handleFillChainTemplate, handleFillDirectTemplate, handleCopyChainTemplate,
    handleCopyDirectTemplate, handleApplyChainJSON, handleApplyDirectText, handleImportUrlChange,
    handleFetchImportURL, handleParseImport, handleConfirmImport,
  } = useProxyImportFlow({
    proxies,
    groups,
    globalAutoRefreshEnabled,
    globalRefreshInterval,
    saveProxies,
  })

  const {
    hasURLImportSources,
    refreshingAllSources,
    refreshingSourceIds,
    refreshSingleSource,
    handleRefreshAllSources,
  } = useProxySourceRefresh({
    proxies,
    globalAutoRefreshEnabled,
    globalRefreshInterval,
    saveProxies,
  })

  const {
    latencyMap,
    latencyEngineMap,
    latencyErrorMap,
    testingAll,
    ipHealthMap,
    checkingIPHealthIds,
    checkingAllIPHealth,
    warmingBridgeIds,
    warmingAllBridges,
    ipHealthDetailOpen,
    setIPHealthDetailOpen,
    currentIPHealthDetail,
    setLatencyMap,
    setLatencyEngineMap,
    setIPHealthMap,
    handleTestOne,
    handleTestAll,
    handleWarmupOne,
    handleWarmupAll,
    handleCheckOneIPHealth,
    handleCheckAllIPHealth,
    openIPHealthDetail,
  } = useProxyChecks({ proxies })

  const loadProxies = useCallback(async () => {
    setLoading(true)
    try {
      const [list, groupList] = await Promise.all([
        fetchBrowserProxies(),
        fetchProxyGroups(),
      ])
      const finalList = await ensureBuiltinProxies(list)
      setProxies(finalList)
      setDisplayList(toDisplayList(finalList))
      setGroups(groupList)

      setLatencyMap(prev => {
        const validIds = new Set(finalList.map(p => p.proxyId))
        const next: Record<string, number> = {}
        Object.entries(prev).forEach(([proxyId, latency]) => {
          if (validIds.has(proxyId)) next[proxyId] = latency
        })
        return next
      })

      setLatencyEngineMap(prev => {
        const validIds = new Set(finalList.map(p => p.proxyId))
        const next: Record<string, string> = {}
        Object.entries(prev).forEach(([proxyId, engine]) => {
          if (validIds.has(proxyId)) next[proxyId] = engine
        })
        return next
      })

      setIPHealthMap(prev => {
        const validIds = new Set(finalList.map(p => p.proxyId))
        const next: Record<string, ProxyIPHealthResult> = {}
        Object.entries(prev).forEach(([proxyId, health]) => {
          if (validIds.has(proxyId)) next[proxyId] = health
        })
        return next
      })
    } catch (error: any) {
      toast.error(error?.message || '加载代理失败')
    } finally {
      setLoading(false)
    }
  }, [setIPHealthMap, setLatencyEngineMap, setLatencyMap])

  useEffect(() => {
    void loadProxies()
    void loadBrowserSettings()
  }, [loadProxies, loadBrowserSettings])

  const { protocolOptions, filteredList } = useProxyPoolFilter({
    displayList,
    filterProtocol,
    filterKeyword,
    groups,
    filterGroupId,
    filterAvailableOnly,
    sortColumn,
    sortOrder,
    latencyMap,
    ipHealthMap,
  })

  const {
    selectedIds,
    selectedCount,
    allFilteredSelected,
    someFilteredSelected,
    batchDeleteConfirmOpen,
    setBatchDeleteConfirmOpen,
    handleToggleAll,
    handleToggleOne,
    handleBatchDeleteConfirm,
    removeSelectedId,
    clearSelection,
  } = useProxySelection({ proxies, filteredList, saveProxies })

  const updateChainEditHop = (hop: 'first' | 'second', field: keyof ChainImportForm['first'], value: string) => {
    setChainEditForm(prev => ({
      ...prev,
      [hop]: {
        ...prev[hop],
        [field]: value,
      },
    }))
  }

  const handleEdit = (record: ProxyDisplayInfo) => {
    const proxy = proxies.find(p => p.proxyId === record.proxyId)
    if (proxy) {
      setEditingProxy(proxy)
      setEditForm({
        proxyName: proxy.proxyName,
        proxyConfig: proxy.proxyConfig,
        preferredKernel: proxy.preferredKernel || 'auto',
        dnsServers: proxy.dnsServers || '',
        groupId: proxy.groupId || '',
      })
      const nextChainForm = toChainImportForm(proxy.proxyName, proxy.proxyConfig)
      const nextDirectForm = nextChainForm ? null : toDirectImportForm(proxy.proxyName, proxy.proxyConfig)
      if (nextChainForm) {
        setChainEditMode(true)
        setChainEditForm(nextChainForm)
        setDirectEditMode(false)
        setDirectEditForm({ ...INITIAL_DIRECT_IMPORT_FORM })
      } else if (nextDirectForm) {
        setChainEditMode(false)
        setChainEditForm(createInitialChainImportForm())
        setDirectEditMode(true)
        setDirectEditForm(nextDirectForm)
      } else {
        setChainEditMode(false)
        setChainEditForm(createInitialChainImportForm())
        setDirectEditMode(false)
        setDirectEditForm({ ...INITIAL_DIRECT_IMPORT_FORM })
      }
      setEditModalOpen(true)
    }
  }

  const handleSaveProxy = async () => {
    if (!editingProxy) return

    let nextProxyName = editForm.proxyName.trim()
    let nextProxyConfig = editForm.proxyConfig
    if (chainEditMode) {
      try {
        const candidate = buildChainImportCandidate(chainEditForm)
        nextProxyName = candidate.proxyName
        nextProxyConfig = candidate.proxyConfig
      } catch (error: any) {
        toast.error(error?.message || '链式代理配置无效')
        return
      }
    } else if (directEditMode) {
      try {
        const candidate = buildDirectImportCandidate(directEditForm)
        nextProxyName = candidate.proxyName
        nextProxyConfig = candidate.proxyConfig
      } catch (error: any) {
        toast.error(error?.message || '代理配置无效')
        return
      }
    } else if (!nextProxyName) {
      toast.error('请输入代理名称')
      return
    }

    setSaving(true)
    try {
      const newProxies = proxies.map(p =>
        p.proxyId === editingProxy.proxyId
          ? {
            ...p,
            proxyName: nextProxyName,
            proxyConfig: nextProxyConfig,
            preferredKernel: editForm.preferredKernel === 'auto' ? undefined : editForm.preferredKernel,
            dnsServers: editForm.dnsServers.trim() || undefined,
            groupId: editForm.groupId || undefined,
            groupName: groups.find(group => group.groupId === editForm.groupId)?.groupName || undefined,
          }
          : p
      )
      await saveProxies(newProxies)
      setEditModalOpen(false)
      toast.success('代理已更新')
    } catch (error: any) {
      toast.error(error?.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }
  const {
    deleteConfirmOpen,
    setDeleteConfirmOpen,
    handleDeleteClick,
    handleDeleteConfirm,
  } = useProxyDeleteFlow({ proxies, saveProxies, removeSelectedId })

  const handleMoveSelected = async () => {
    if (selectedIds.size === 0) return
    setMovingSelected(true)
    try {
      await moveProxiesToGroup(Array.from(selectedIds), moveTargetGroupId)
      clearSelection()
      await loadProxies()
      const targetName = groups.find(group => group.groupId === moveTargetGroupId)?.groupName || '未分组'
      toast.success(`已将所选代理移动到“${targetName}”`)
    } catch (error: any) {
      toast.error(error?.message || '移动代理失败')
    } finally {
      setMovingSelected(false)
    }
  }
  return (
    <div className="space-y-5 animate-fade-in">
      <ProxyPoolHeader
        checkingAllIPHealth={checkingAllIPHealth}
        currentConnectorStatus={currentCoreStatus?.message || '未知'}
        hasURLImportSources={hasURLImportSources}
        onCheckAllIPHealth={() => void handleCheckAllIPHealth(filteredList)}
        onOpenSettings={() => void openCheckSettings()}
        onOpenUsageGuide={() => setUsageGuideOpen(true)}
        onOpenImport={() => setImportModalOpen(true)}
        onOpenCoreDownload={openCoreDownload}
        onRefreshAllSources={() => void handleRefreshAllSources(false)}
        onTestAll={() => void handleTestAll(filteredList)}
        refreshingAllSources={refreshingAllSources}
        testingAll={testingAll}
        totalCount={filteredList.length}
      />

      <div className={groupSidebarCollapsed ? 'grid items-start gap-3 lg:grid-cols-[56px_minmax(0,1fr)]' : 'grid items-start gap-4 lg:grid-cols-[280px_minmax(0,1fr)]'}>
        <ProxyGroupTreeNav
          groups={groups}
          selectedGroupId={filterGroupId}
          totalCount={proxies.length}
          ungroupedCount={proxies.filter(proxy => !proxy.groupId).length}
          onSelectGroup={setFilterGroupId}
          onRefresh={loadProxies}
          collapsed={groupSidebarCollapsed}
          onToggleCollapsed={() => setGroupSidebarCollapsed(previous => !previous)}
        />
        <div className="min-w-0 space-y-3">
          {selectedCount > 0 && (
            <div className="flex flex-wrap items-center gap-3 rounded-xl border border-[var(--color-accent)]/25 bg-[var(--color-accent)]/10 px-4 py-3 shadow-sm">
              <div className="flex items-center gap-2">
                <span className="inline-flex min-w-7 items-center justify-center rounded-full bg-[var(--color-accent)] px-2 py-1 text-xs font-semibold text-[var(--color-text-inverse)]">
                  {selectedCount}
                </span>
                <span className="text-sm font-medium text-[var(--color-text-primary)]">个代理已选择</span>
              </div>
              <div className="flex flex-wrap items-center gap-2 rounded-lg border border-[var(--color-border-default)] bg-[var(--color-bg-surface)] px-2.5 py-2">
                <FolderInput className="h-4 w-4 text-[var(--color-accent)]" />
                <span className="text-xs font-medium text-[var(--color-text-secondary)]">移动到</span>
                <ProxyGroupSelect
                  groups={groups}
                  value={moveTargetGroupId}
                  onChange={setMoveTargetGroupId}
                  className="min-w-[220px]"
                />
                <Button size="sm" onClick={() => void handleMoveSelected()} loading={movingSelected}>确认移动</Button>
              </div>
              <Button
                size="sm"
                variant="ghost"
                className="ml-auto"
                disabled={movingSelected}
                onClick={clearSelection}
              >
                <X className="h-3.5 w-3.5" />取消选择
              </Button>
            </div>
          )}
          <ProxyPoolTableCard
            allFilteredSelected={allFilteredSelected}
            checkingIPHealthIds={checkingIPHealthIds}
            data={filteredList}
            filterKeyword={filterKeyword}
            filterProtocol={filterProtocol}
            filterAvailableOnly={filterAvailableOnly}
            globalAutoRefreshEnabled={globalAutoRefreshEnabled}
            globalRefreshInterval={globalRefreshInterval}
            globalRefreshIntervalM={globalRefreshIntervalM}
            ipHealthMap={ipHealthMap}
            latencyMap={latencyMap}
            latencyEngineMap={latencyEngineMap}
            latencyErrorMap={latencyErrorMap}
            loading={loading}
            onCheckOneIPHealth={(record) => void handleCheckOneIPHealth(record)}
            onClearFilters={() => {
              setFilterProtocol('all')
              setFilterKeyword('')
              setFilterGroupId('__all__')
              setFilterAvailableOnly(false)
            }}
            onDelete={handleDeleteClick}
            onEdit={handleEdit}
            onFilterKeywordChange={setFilterKeyword}
            onFilterProtocolChange={setFilterProtocol}
            onFilterAvailableOnlyChange={setFilterAvailableOnly}
            onGlobalAutoRefreshEnabledChange={setGlobalAutoRefreshEnabled}
            onGlobalRefreshIntervalMChange={setGlobalRefreshIntervalM}
            onOpenBatchDelete={() => setBatchDeleteConfirmOpen(true)}
            onOpenIPHealthDetail={openIPHealthDetail}
            onRefreshSingleSource={(sourceId) => void refreshSingleSource(sourceId, false)}
            onSort={({ column, order }) => {
              setSortColumn(column)
              setSortOrder(order)
            }}
            onTestOne={(record) => void handleTestOne(record)}
            onToggleAll={handleToggleAll}
            onToggleOne={handleToggleOne}
            onWarmupOne={(record) => void handleWarmupOne(record)}
            onWarmupSelected={() => void handleWarmupAll(filteredList.filter(item => selectedIds.has(item.proxyId)))}
            protocolOptions={protocolOptions}
            refreshingSourceIds={refreshingSourceIds}
            selectedCount={selectedCount}
            selectedIds={selectedIds}
            someFilteredSelected={someFilteredSelected}
            sortColumn={sortColumn}
            sortOrder={sortOrder}
            warmingAllBridges={warmingAllBridges}
            warmingBridgeIds={warmingBridgeIds}
          />
        </div>
      </div>

      <ProxyPoolUsageGuideModal
        open={usageGuideOpen}
        onClose={() => setUsageGuideOpen(false)}
      />
      <ProxyPoolImportModal
        groups={groups}
        open={importModalOpen}
        importMode={importMode}
        importUrl={importUrl}
        importFetchProxyId={importFetchProxyId}
        importUserAgentPreset={importUserAgentPreset}
        importCustomUserAgent={importCustomUserAgent}
        importUserAgentFallback={importUserAgentFallback}
        importResolvedUrl={importResolvedUrl}
        importText={importText}
        importDnsServers={importDnsServers}
        importNamePrefix={importNamePrefix}
        importGroupId={importGroupId}
        chainImportText={chainImportText}
        directImportText={directImportText}
        chainImportForm={chainImportForm}
        directImportForm={directImportForm}
        fetchingImportUrl={fetchingImportUrl}
        fetchProxyOptions={proxies.filter(proxy => proxy.proxyConfig.trim() && !proxy.proxyConfig.trim().toLowerCase().startsWith('direct://'))}
        canParseImport={canParseImport}
        onClose={() => setImportModalOpen(false)}
        onParse={handleParseImport}
        onFetchImportUrl={handleFetchImportURL}
        onImportModeChange={handleImportModeChange}
        onImportUrlChange={handleImportUrlChange}
        onImportFetchProxyIdChange={setImportFetchProxyId}
        onImportUserAgentPresetChange={setImportUserAgentPreset}
        onImportCustomUserAgentChange={setImportCustomUserAgent}
        onImportUserAgentFallbackChange={setImportUserAgentFallback}
        onImportTextChange={setImportText}
        onImportDnsServersChange={setImportDnsServers}
        onImportNamePrefixChange={setImportNamePrefix}
        onImportGroupIdChange={setImportGroupId}
        onChainImportTextChange={setChainImportText}
        onDirectImportTextChange={setDirectImportText}
        onApplyChainJSON={handleApplyChainJSON}
        onApplyDirectText={handleApplyDirectText}
        onChainImportFormChange={(patch) => setChainImportForm((prev) => ({ ...prev, ...patch }))}
        onChainImportHopChange={updateChainImportHop}
        onFillChainTemplate={handleFillChainTemplate}
        onCopyChainTemplate={() => void handleCopyChainTemplate()}
        onFillDirectTemplate={handleFillDirectTemplate}
        onCopyDirectTemplate={() => void handleCopyDirectTemplate()}
        onDirectImportFormChange={(patch) => setDirectImportForm((prev) => ({ ...prev, ...patch }))}
      />

      <ProxyPoolPreviewModal
        open={previewModalOpen}
        importMode={importMode}
        importDnsServers={importDnsServers}
        previewList={previewList}
        removedPreviewProxyNames={removedPreviewProxyNames}
        importing={importing}
        onClose={() => setPreviewModalOpen(false)}
        onBack={() => {
          setPreviewModalOpen(false)
          setImportModalOpen(true)
        }}
        onConfirm={handleConfirmImport}
        onRemoveProxy={handleRemovePreviewProxy}
      />

      <ProxyPoolEditModal
        groups={groups}
        open={editModalOpen}
        saving={saving}
        editForm={editForm}
        chainEditMode={chainEditMode}
        chainEditForm={chainEditForm}
        directEditMode={directEditMode}
        directEditForm={directEditForm}
        onClose={() => setEditModalOpen(false)}
        onSave={handleSaveProxy}
        onChange={(patch) => setEditForm((prev) => ({ ...prev, ...patch }))}
        onChainEditFormChange={(patch) => setChainEditForm((prev) => ({ ...prev, ...patch }))}
        onDirectEditFormChange={(patch) => setDirectEditForm((prev) => ({ ...prev, ...patch }))}
        onChainEditHopChange={updateChainEditHop}
      />

      <ProxyPoolIPHealthDetailModal
        open={ipHealthDetailOpen}
        detail={currentIPHealthDetail}
        onClose={() => setIPHealthDetailOpen(false)}
      />

      <ProxyPoolCheckSettingsModal
        open={checkSettingsOpen}
        checkSettings={checkSettings}
        checkTargetsText={checkTargetsText}
        saving={savingCheckSettings}
        onClose={() => setCheckSettingsOpen(false)}
        onSave={saveCheckSettings}
        onCheckSettingsChange={setCheckSettings}
        onCheckTargetsTextChange={setCheckTargetsText}
      />

      <ProxyCoreDownloadModal
        open={coreDownloadOpen}
        core={coreDownloadType}
        goos={coreDownloadGOOS}
        goarch={coreDownloadGOARCH}
        downloadProxy={coreDownloadProxy}
        progress={coreDownloadProgress}
        status={downloadCoreStatus}
        statusLoading={downloadCoreStatusLoading}
        onCoreChange={setCoreDownloadType}
        onGOOSChange={setCoreDownloadGOOS}
        onGOARCHChange={setCoreDownloadGOARCH}
        onDownloadProxyChange={setCoreDownloadProxy}
        onClose={closeCoreDownload}
        onStart={handleStartCoreDownload}
      />

      <ConfirmModal open={deleteConfirmOpen} onClose={() => setDeleteConfirmOpen(false)} onConfirm={handleDeleteConfirm}
        title="确认删除" content="确定要删除这个代理吗？此操作不可恢复。" confirmText="删除" danger />

      <ConfirmModal open={batchDeleteConfirmOpen} onClose={() => setBatchDeleteConfirmOpen(false)} onConfirm={handleBatchDeleteConfirm}
        title="批量删除" content={`确定要删除选中的 ${selectedCount} 个代理吗？此操作不可恢复。`} confirmText="删除" danger />
    </div>
  )
}
