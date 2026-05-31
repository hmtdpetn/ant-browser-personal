import { Link } from 'react-router-dom'
import { XCircle } from 'lucide-react'
import { Button, FormItem, Input, Modal } from '../../../../shared/components'
import { KeywordsModal } from '../../components/KeywordsModal'
import type { BrowserProfile } from '../../types'

interface BrowserListDialogsProps {
  proxyErrorModal: boolean
  pendingStartId: string | null
  proxyErrorMsg: string
  onCloseProxyError: () => void
  onStartDirect: () => void
  startingDirect: boolean
  kwModal: { open: boolean; profile: BrowserProfile | null }
  onCloseKeywords: () => void
  onKeywordsSaved: (keywords: string[]) => void
  expandModalOpen: boolean
  onCloseExpand: () => void
  profilesCount: number
  maxProfileLimit: number
  cdKey: string
  onCdKeyChange: (value: string) => void
  onRedeem: () => void
  redeeming: boolean
  onOpenGithubStarGift: () => void
  copyModal: { open: boolean; profile: BrowserProfile | null }
  copyName: string
  onCopyNameChange: (value: string) => void
  onCloseCopy: () => void
  onConfirmCopy: () => void
  copying: boolean
  opError: string
  onCloseOpError: () => void
}

export function BrowserListDialogs({
  proxyErrorModal,
  pendingStartId,
  proxyErrorMsg,
  onCloseProxyError,
  onStartDirect,
  startingDirect,
  kwModal,
  onCloseKeywords,
  onKeywordsSaved,
  expandModalOpen,
  onCloseExpand,
  profilesCount,
  maxProfileLimit,
  cdKey,
  onCdKeyChange,
  onRedeem,
  redeeming,
  onOpenGithubStarGift,
  copyModal,
  copyName,
  onCopyNameChange,
  onCloseCopy,
  onConfirmCopy,
  copying,
  opError,
  onCloseOpError,
}: BrowserListDialogsProps) {
  void onOpenGithubStarGift

  return (
    <>
      <Modal
        open={proxyErrorModal}
        onClose={onCloseProxyError}
        title="Proxy unavailable"
        width="420px"
        footer={
          <>
            <Button variant="secondary" onClick={onCloseProxyError} disabled={startingDirect}>Cancel</Button>
            {pendingStartId && (
              <Button variant="secondary" onClick={onStartDirect} loading={startingDirect}>
                Start direct
              </Button>
            )}
            {pendingStartId && (
              <Link to={`/browser/edit/${pendingStartId}`}>
                <Button onClick={onCloseProxyError} disabled={startingDirect}>Edit proxy</Button>
              </Link>
            )}
          </>
        }
      >
        <div className="space-y-3">
          <div className="flex items-start gap-3 p-3 rounded-lg bg-[var(--color-bg-secondary)]">
            <XCircle className="w-5 h-5 text-red-500 mt-0.5 shrink-0" />
            <p className="text-sm text-[var(--color-text-primary)]">{proxyErrorMsg}</p>
          </div>
          <p className="text-sm text-[var(--color-text-muted)]">
            Update this profile proxy or refresh the subscription before starting again.
          </p>
        </div>
      </Modal>

      {kwModal.profile && (
        <KeywordsModal
          open={kwModal.open}
          profileId={kwModal.profile.profileId}
          profileName={kwModal.profile.profileName}
          initialKeywords={kwModal.profile.keywords || []}
          onClose={onCloseKeywords}
          onSaved={onKeywordsSaved}
        />
      )}

      <Modal
        open={expandModalOpen}
        onClose={onCloseExpand}
        title="Profile quota"
        width="480px"
        footer={<Button variant="secondary" onClick={onCloseExpand}>Close</Button>}
      >
        <div className="space-y-4">
          <div className="bg-[var(--color-bg-secondary)] p-4 rounded-lg flex items-center justify-between border border-[var(--color-border-default)]">
            <div>
              <p className="text-sm font-medium text-[var(--color-text-primary)]">Current usage</p>
              <p className="text-xs text-[var(--color-text-muted)] mt-1">Each profile uses one quota slot.</p>
            </div>
            <div className="text-right">
              <span className={`text-2xl font-semibold ${profilesCount >= maxProfileLimit ? 'text-red-500' : 'text-[var(--color-success)]'}`}>
                {profilesCount}
              </span>
              <span className="text-sm text-[var(--color-text-muted)] ml-1">/ {maxProfileLimit}</span>
            </div>
          </div>

          <div className="pt-2 border-t border-[var(--color-border-muted)]">
            <label className="block text-sm font-medium text-[var(--color-text-primary)] mb-2">Redeem code</label>
            <div className="flex gap-2">
              <Input
                value={cdKey}
                onChange={e => onCdKeyChange(e.target.value)}
                placeholder="Enter redeem code"
                onKeyDown={e => e.key === 'Enter' && onRedeem()}
                className="flex-1"
              />
              <Button onClick={onRedeem} loading={redeeming} disabled={!cdKey.trim()}>
                Redeem
              </Button>
            </div>
          </div>
        </div>
      </Modal>

      <Modal
        open={copyModal.open}
        onClose={onCloseCopy}
        title="Copy profile"
        width="420px"
        footer={
          <>
            <Button variant="secondary" onClick={onCloseCopy}>Cancel</Button>
            <Button onClick={onConfirmCopy} loading={copying}>Copy</Button>
          </>
        }
      >
        <div className="space-y-4">
          <p className="text-sm text-[var(--color-text-muted)]">
            Copying keeps proxy, core, launch arguments, and tags, then creates a fresh fingerprint seed.
          </p>
          <FormItem label="New profile name" required>
            <Input
              value={copyName}
              onChange={e => onCopyNameChange(e.target.value)}
              placeholder="Enter a profile name"
              autoFocus
            />
          </FormItem>
        </div>
      </Modal>

      <Modal
        open={!!opError}
        onClose={onCloseOpError}
        title="Action failed"
        width="420px"
        footer={<Button onClick={onCloseOpError}>OK</Button>}
      >
        <div className="text-[var(--color-text-secondary)] whitespace-pre-line">{opError}</div>
      </Modal>
    </>
  )
}
