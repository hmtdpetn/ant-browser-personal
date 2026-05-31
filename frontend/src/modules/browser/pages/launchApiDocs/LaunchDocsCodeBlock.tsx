import { useState, type ReactNode } from 'react'
import clsx from 'clsx'
import { CheckCircle, Copy } from 'lucide-react'
import { toast } from '../../../../shared/components'

type HighlightTokenKind =
  | 'plain'
  | 'comment'
  | 'keyword'
  | 'builtin'
  | 'property'
  | 'string'
  | 'number'
  | 'boolean'
  | 'null'
  | 'variable'
  | 'operator'
  | 'punctuation'

interface HighlightToken {
  content: string
  kind: HighlightTokenKind
}

interface LaunchDocsCodeBlockProps {
  language?: string
  code: string
  className?: string
  maxHeightClassName?: string
  showCopyButton?: boolean
}

const TOKEN_CLASS_NAMES: Record<HighlightTokenKind, string> = {
  plain: 'text-[var(--color-text-primary)]',
  comment: 'text-[var(--color-text-muted)] italic',
  keyword: 'font-medium text-[var(--color-info)]',
  builtin: 'font-medium text-[var(--color-accent)]',
  property: 'font-medium text-[var(--color-accent)]',
  string: 'text-[var(--color-success)]',
  number: 'text-[var(--color-warning)]',
  boolean: 'font-medium text-[var(--color-warning)]',
  null: 'font-medium text-[var(--color-warning)]',
  variable: 'text-[var(--color-warning)]',
  operator: 'text-[var(--color-text-secondary)]',
  punctuation: 'text-[var(--color-text-muted)]',
}

const JSON_PATTERN =
  /("(?:\\.|[^"\\])*")(?=\s*:)|("(?:\\.|[^"\\])*")|\b(?:true|false)\b|\bnull\b|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|[{}\[\],:]/g
const COMMAND_PATTERN =
  /(https?:\/\/[^\s"'`]+)|("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')|(\$env:[A-Za-z_][\w-]*|\$\{?[\w:.-]+\}?)|(--?[A-Za-z][\w-]*)|\b(?:curl|GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS|Invoke-RestMethod|fetch|python|node|powershell|pwsh|http)\b|\b\d+(?:\.\d+)?\b/g
const SCRIPT_PATTERN =
  /(\/\/.*$)|("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*'|`(?:\\.|[^`\\])*`)|\b(?:const|let|var|async|await|function|return|if|else|throw|new|import|from|export|default|try|catch|for|of|while|true|false|null|undefined)\b|\b(?:fetch|JSON|console|await)\b|-?\d+(?:\.\d+)?\b|[{}()[\].,:;=+\-*/<>!?]+/gm
const PYTHON_PATTERN =
  /("(?:\\.|[^"\\])*"|'(?:\\.|[^'\\])*')|\b(?:def|import|from|return|if|elif|else|for|in|while|True|False|None|async|await|with|as|raise|try|except|pass)\b|\b(?:requests|json|print)\b|-?\d+(?:\.\d+)?\b|[{}()[\].,:;=+\-*/<>!?]+/g

function normalizeLanguage(language?: string) {
  const value = (language || '').trim().toLowerCase()
  if (!value) {
    return 'text'
  }
  if (value === 'js') {
    return 'javascript'
  }
  if (value === 'ts') {
    return 'typescript'
  }
  if (value === 'py') {
    return 'python'
  }
  if (value === 'sh' || value === 'shell') {
    return 'bash'
  }
  if (value === 'ps1') {
    return 'powershell'
  }
  return value
}

function tokenizeWithPattern(
  line: string,
  pattern: RegExp,
  resolveKind: (match: RegExpExecArray) => HighlightTokenKind,
): HighlightToken[] {
  const tokens: HighlightToken[] = []
  let lastIndex = 0
  pattern.lastIndex = 0

  for (let match = pattern.exec(line); match; match = pattern.exec(line)) {
    if (match.index > lastIndex) {
      tokens.push({ content: line.slice(lastIndex, match.index), kind: 'plain' })
    }
    tokens.push({ content: match[0], kind: resolveKind(match) })
    lastIndex = match.index + match[0].length
  }

  if (lastIndex < line.length) {
    tokens.push({ content: line.slice(lastIndex), kind: 'plain' })
  }

  return tokens.length ? tokens : [{ content: line, kind: 'plain' }]
}

function highlightJsonLine(line: string) {
  return tokenizeWithPattern(line, JSON_PATTERN, (match) => {
    const [token] = match
    if (match[1]) {
      return 'property'
    }
    if (match[2]) {
      return 'string'
    }
    if (token === 'true' || token === 'false') {
      return 'boolean'
    }
    if (token === 'null') {
      return 'null'
    }
    if (/^-?\d/.test(token)) {
      return 'number'
    }
    return 'punctuation'
  })
}

function highlightCommandLine(line: string) {
  if (/^\s*#/.test(line)) {
    return [{ content: line, kind: 'comment' as const }]
  }

  return tokenizeWithPattern(line, COMMAND_PATTERN, (match) => {
    const [token] = match
    if (/^https?:\/\//.test(token)) {
      return 'string'
    }
    if (/^["']/.test(token)) {
      return 'string'
    }
    if (/^\$/.test(token)) {
      return 'variable'
    }
    if (/^--?/.test(token)) {
      return 'keyword'
    }
    if (/^-?\d/.test(token)) {
      return 'number'
    }
    return 'builtin'
  })
}

function highlightScriptLine(line: string) {
  return tokenizeWithPattern(line, SCRIPT_PATTERN, (match) => {
    const [token] = match
    if (token.startsWith('//')) {
      return 'comment'
    }
    if (/^["'`]/.test(token)) {
      return 'string'
    }
    if (/^(const|let|var|async|await|function|return|if|else|throw|new|import|from|export|default|try|catch|for|of|while|true|false|null|undefined)$/.test(token)) {
      return token === 'true' || token === 'false' ? 'boolean' : token === 'null' ? 'null' : 'keyword'
    }
    if (/^(fetch|JSON|console|await)$/.test(token)) {
      return 'builtin'
    }
    if (/^-?\d/.test(token)) {
      return 'number'
    }
    return 'operator'
  })
}

function highlightPythonLine(line: string) {
  if (/^\s*#/.test(line)) {
    return [{ content: line, kind: 'comment' as const }]
  }

  return tokenizeWithPattern(line, PYTHON_PATTERN, (match) => {
    const [token] = match
    if (/^["']/.test(token)) {
      return 'string'
    }
    if (/^(True|False)$/.test(token)) {
      return 'boolean'
    }
    if (token === 'None') {
      return 'null'
    }
    if (/^(def|import|from|return|if|elif|else|for|in|while|async|await|with|as|raise|try|except|pass)$/.test(token)) {
      return 'keyword'
    }
    if (/^(requests|json|print)$/.test(token)) {
      return 'builtin'
    }
    if (/^-?\d/.test(token)) {
      return 'number'
    }
    return 'operator'
  })
}

function highlightLine(language: string, line: string) {
  if (!line) {
    return [{ content: '\u00a0', kind: 'plain' as const }]
  }

  if (language === 'json') {
    return highlightJsonLine(line)
  }

  if (language === 'bash' || language === 'powershell') {
    return highlightCommandLine(line)
  }

  if (language === 'javascript' || language === 'typescript') {
    return highlightScriptLine(line)
  }

  if (language === 'python') {
    return highlightPythonLine(line)
  }

  return [{ content: line, kind: 'plain' as const }]
}

function renderHighlightedCode(language: string, code: string): ReactNode {
  return code.split('\n').map((line, lineIndex) => (
    <span key={`${language}-${lineIndex}`} className="block">
      {highlightLine(language, line).map((token, tokenIndex) => (
        <span
          key={`${language}-${lineIndex}-${tokenIndex}`}
          className={TOKEN_CLASS_NAMES[token.kind]}
        >
          {token.content}
        </span>
      ))}
    </span>
  ))
}

function CopyButton({ text }: { text: string }) {
  const [copied, setCopied] = useState(false)

  return (
    <button
      onClick={() => {
        navigator.clipboard.writeText(text).then(() => {
          setCopied(true)
          toast.success('已复制')
          setTimeout(() => setCopied(false), 2000)
        })
      }}
      className="flex items-center gap-1 rounded-md px-2 py-1 text-xs text-[var(--color-text-muted)] transition-colors hover:bg-[var(--color-bg-muted)] hover:text-[var(--color-accent)]"
    >
      {copied ? <CheckCircle className="h-3.5 w-3.5 text-green-500" /> : <Copy className="h-3.5 w-3.5" />}
      {copied ? '已复制' : '复制'}
    </button>
  )
}

export function LaunchDocsCodeBlock({
  language,
  code,
  className,
  maxHeightClassName,
  showCopyButton = true,
}: LaunchDocsCodeBlockProps) {
  const normalizedLanguage = normalizeLanguage(language)
  const displayCode = code.replace(/\n$/, '')

  return (
    <div
      className={clsx(
        'my-3 overflow-hidden rounded-2xl border border-[var(--color-border-default)] bg-[var(--color-bg-elevated)] shadow-[var(--shadow-sm)]',
        className,
      )}
    >
      <div className="flex items-center justify-between border-b border-[var(--color-border-muted)] bg-[var(--color-bg-surface)] px-4 py-2">
        <span className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--color-text-muted)]">
          {normalizedLanguage || 'code'}
        </span>
        {showCopyButton ? <CopyButton text={displayCode} /> : null}
      </div>
      <pre
        className={clsx(
          'overflow-x-auto bg-[var(--color-bg-elevated)]',
          maxHeightClassName,
        )}
      >
        <code className="block min-w-full px-4 py-3 text-sm leading-6 whitespace-pre">
          {renderHighlightedCode(normalizedLanguage, displayCode)}
        </code>
      </pre>
    </div>
  )
}
