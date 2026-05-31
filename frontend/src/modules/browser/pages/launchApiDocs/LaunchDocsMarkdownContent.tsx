import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { FolderOpen } from 'lucide-react'
import { BrowserOpenURL } from '../../../../wailsjs/runtime/runtime'
import { Button, toast } from '../../../../shared/components'
import { openProjectRoot } from '../../api/filesystem'
import { LaunchDocsCodeBlock } from './LaunchDocsCodeBlock'

export function LaunchDocsMarkdownContent({ content, docId }: { content: string; docId?: string }) {
  const showProjectRootAction = docId === 'tutorial-skill'

  return (
    <div className="space-y-4">
      {showProjectRootAction ? (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] px-4 py-3 shadow-[var(--shadow-sm)]">
          <span className="text-sm text-[var(--color-text-secondary)]">
            安装命令默认相对项目根目录执行。
          </span>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => {
              void openProjectRoot().catch((error: any) => {
                toast.error(error?.message || '打开根目录失败')
              })
            }}
          >
            <FolderOpen className="h-4 w-4" />
            打开根目录
          </Button>
        </div>
      ) : null}
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          h1: ({ children }) => (
            <h1 className="text-2xl font-bold text-[var(--color-text-primary)] mb-6 pb-3 border-b border-[var(--color-border-default)]">
              {children}
            </h1>
          ),
          h2: ({ children }) => (
            <h2 className="text-lg font-semibold text-[var(--color-text-primary)] mt-8 mb-3 flex items-center gap-2">
              <span className="w-1 h-5 bg-[var(--color-accent)] rounded-full inline-block shrink-0" />
              {children}
            </h2>
          ),
          h3: ({ children }) => (
            <h3 className="text-base font-semibold text-[var(--color-text-primary)] mt-6 mb-2">
              {children}
            </h3>
          ),
          p: ({ children }) => (
            <p className="text-sm text-[var(--color-text-secondary)] leading-relaxed mb-3">
              {children}
            </p>
          ),
          ul: ({ children }) => (
            <ul className="space-y-1 mb-4 pl-5 list-disc marker:text-[var(--color-accent)]">{children}</ul>
          ),
          ol: ({ children }) => (
            <ol className="space-y-1 mb-4 pl-5 list-decimal marker:text-[var(--color-accent)]">{children}</ol>
          ),
          li: ({ children }) => (
            <li className="text-sm text-[var(--color-text-secondary)] leading-relaxed">
              {children}
            </li>
          ),
          code: ({ children, className }) => {
            const isBlock = className?.includes('language-')
            if (isBlock) {
              return <code className={className}>{children}</code>
            }
            return (
              <code className="rounded border border-[var(--color-border-muted)] bg-[var(--color-bg-muted)] px-1.5 py-0.5 text-xs font-mono text-[var(--color-accent)]">
                {children}
              </code>
            )
          },
          pre: ({ children }) => {
            const codeEl = (children as any)?.props
            const lang = codeEl?.className?.replace('language-', '') || ''
            const codeText = Array.isArray(codeEl?.children)
              ? codeEl.children.join('')
              : String(codeEl?.children || '')
            return <LaunchDocsCodeBlock language={lang} code={codeText} />
          },
          table: ({ children }) => (
            <div className="my-4 overflow-x-auto rounded-xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] shadow-[var(--shadow-sm)]">
              <table className="w-full text-sm">{children}</table>
            </div>
          ),
          thead: ({ children }) => (
            <thead className="border-b border-[var(--color-border-default)] bg-[var(--color-bg-surface)]">
              {children}
            </thead>
          ),
          th: ({ children }) => (
            <th className="px-4 py-2.5 text-left text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wide">
              {children}
            </th>
          ),
          td: ({ children }) => (
            <td className="px-4 py-2.5 text-[var(--color-text-secondary)] border-t border-[var(--color-border-muted)]">
              {children}
            </td>
          ),
          blockquote: ({ children }) => (
            <blockquote className="my-3 pl-4 border-l-2 border-[var(--color-accent)] text-[var(--color-text-muted)] italic">
              {children}
            </blockquote>
          ),
          strong: ({ children }) => (
            <strong className="font-semibold text-[var(--color-text-primary)]">{children}</strong>
          ),
          hr: () => <hr className="my-6 border-[var(--color-border-default)]" />,
          a: ({ href, children }) => (
            <a
              href={href}
              onClick={(event) => {
                event.preventDefault()
                if (href) {
                  BrowserOpenURL(href)
                }
              }}
              className="text-[var(--color-accent)] hover:underline cursor-pointer"
              title={href}
            >
              {children}
            </a>
          ),
        }}
      >
        {content}
      </ReactMarkdown>
    </div>
  )
}
