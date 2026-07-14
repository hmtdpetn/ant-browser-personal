import { PointerEvent as ReactPointerEvent, ReactNode, useEffect, useMemo, useRef, useState } from 'react'
import clsx from 'clsx'
import { ArrowDown, ArrowUp } from 'lucide-react'

export type SortOrder = 'asc' | 'desc' | undefined

export interface SorterResult {
  column: string
  order: SortOrder
}

export interface TableColumn<T> {
  key: string
  title: ReactNode
  width?: string | number
  minWidth?: number
  maxWidth?: number
  resizable?: boolean
  align?: 'left' | 'center' | 'right'
  render?: (value: any, record: T, index: number) => ReactNode
  sortable?: boolean
}

interface TableProps<T> {
  columns: TableColumn<T>[]
  data: T[]
  rowKey: string | ((record: T) => string)
  loading?: boolean
  emptyText?: string
  onRowClick?: (record: T) => void
  className?: string
  maxHeight?: string
  stickyHeader?: boolean
  onSort?: (sorterResult: SorterResult) => void
  sortColumn?: string
  sortOrder?: SortOrder
  resizableColumns?: boolean
  columnWidthsStorageKey?: string
}

interface ActiveResize {
  columnKey: string
  startX: number
  startWidth: number
  minWidth: number
  maxWidth: number
  previousCursor: string
  previousUserSelect: string
}

export function Table<T extends Record<string, any>>({
  columns,
  data,
  rowKey,
  loading = false,
  emptyText = '暂无数据',
  onRowClick,
  className,
  maxHeight = 'calc(100vh - 320px)',
  stickyHeader = true,
  onSort,
  sortColumn,
  sortOrder,
  resizableColumns = false,
  columnWidthsStorageKey,
}: TableProps<T>) {
  const initialWidths = useMemo(() => {
    const defaults: Record<string, number> = {}
    columns.forEach((column) => {
      if (typeof column.width === 'number') defaults[column.key] = column.width
    })
    if (!columnWidthsStorageKey) return defaults
    try {
      const saved = JSON.parse(localStorage.getItem(columnWidthsStorageKey) || '{}') as Record<string, unknown>
      Object.entries(saved).forEach(([key, value]) => {
        if (typeof value === 'number' && Number.isFinite(value)) defaults[key] = value
      })
    } catch {
      // Ignore invalid local settings and keep the source-code defaults.
    }
    return defaults
    // The persisted key defines the initial user preference; column updates are synchronized below.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [columnWidthsStorageKey])
  const [columnWidths, setColumnWidths] = useState<Record<string, number>>(initialWidths)
  const widthsRef = useRef(columnWidths)
  const activeResizeRef = useRef<ActiveResize | null>(null)

  useEffect(() => {
    widthsRef.current = columnWidths
  }, [columnWidths])

  useEffect(() => {
    setColumnWidths((previous) => {
      let changed = false
      const next = { ...previous }
      columns.forEach((column) => {
        if (typeof column.width === 'number' && typeof next[column.key] !== 'number') {
          next[column.key] = column.width
          changed = true
        }
      })
      return changed ? next : previous
    })
  }, [columns])

  useEffect(() => {
    const handlePointerMove = (event: PointerEvent) => {
      const active = activeResizeRef.current
      if (!active) return
      const nextWidth = Math.min(active.maxWidth, Math.max(active.minWidth, active.startWidth + event.clientX - active.startX))
      setColumnWidths((previous) => {
        const next = { ...previous, [active.columnKey]: Math.round(nextWidth) }
        widthsRef.current = next
        return next
      })
    }

    const finishResize = () => {
      const active = activeResizeRef.current
      if (!active) return
      activeResizeRef.current = null
      document.body.style.cursor = active.previousCursor
      document.body.style.userSelect = active.previousUserSelect
      if (columnWidthsStorageKey) {
        try {
          localStorage.setItem(columnWidthsStorageKey, JSON.stringify(widthsRef.current))
        } catch {
          // A disabled localStorage must not break table resizing.
        }
      }
    }

    window.addEventListener('pointermove', handlePointerMove)
    window.addEventListener('pointerup', finishResize)
    window.addEventListener('pointercancel', finishResize)
    return () => {
      window.removeEventListener('pointermove', handlePointerMove)
      window.removeEventListener('pointerup', finishResize)
      window.removeEventListener('pointercancel', finishResize)
      finishResize()
    }
  }, [columnWidthsStorageKey])

  const getColumnWidth = (column: TableColumn<T>) => {
    if (typeof column.width === 'number') return columnWidths[column.key] ?? column.width
    return column.width
  }

  const beginResize = (event: ReactPointerEvent<HTMLButtonElement>, column: TableColumn<T>) => {
    if (!resizableColumns || column.resizable === false) return
    const width = getColumnWidth(column)
    if (typeof width !== 'number') return
    event.preventDefault()
    event.stopPropagation()
    activeResizeRef.current = {
      columnKey: column.key,
      startX: event.clientX,
      startWidth: width,
      minWidth: column.minWidth ?? 64,
      maxWidth: column.maxWidth ?? 640,
      previousCursor: document.body.style.cursor,
      previousUserSelect: document.body.style.userSelect,
    }
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }

  const getRowKey = (record: T, index: number): string => {
    if (typeof rowKey === 'function') return rowKey(record)
    return record[rowKey] ?? index.toString()
  }

  const handleSortClick = (column: TableColumn<T>) => {
    if (!column.sortable || !onSort) return
    let newOrder: SortOrder
    if (sortColumn !== column.key) newOrder = 'asc'
    else newOrder = sortOrder === 'asc' ? 'desc' : sortOrder === 'desc' ? undefined : 'asc'
    onSort({ column: column.key, order: newOrder })
  }

  const renderSortIcon = (column: TableColumn<T>) => {
    if (!column.sortable) return null
    if (sortColumn === column.key && sortOrder === 'asc') {
      return <ArrowUp className="ml-1 h-3.5 w-3.5 shrink-0 text-[var(--color-accent)]" />
    }
    if (sortColumn === column.key && sortOrder === 'desc') {
      return <ArrowDown className="ml-1 h-3.5 w-3.5 shrink-0 text-[var(--color-accent)]" />
    }
    return <ArrowUp className="ml-1 h-3 w-3 shrink-0 text-[var(--color-text-muted)] opacity-40 group-hover:opacity-70" />
  }

  if (loading) {
    return (
      <div className="flex items-center justify-center py-16" style={{ maxHeight }}>
        <div className="flex flex-col items-center gap-3">
          <div className="h-6 w-6 animate-spin rounded-full border-2 border-[var(--color-border-default)] border-t-[var(--color-accent)]" />
          <span className="text-sm text-[var(--color-text-muted)]">加载中...</span>
        </div>
      </div>
    )
  }

  const numericTableWidth = resizableColumns
    ? columns.reduce((total, column) => {
        const width = getColumnWidth(column)
        return total + (typeof width === 'number' ? width : 0)
      }, 0)
    : 0

  return (
    <div className={clsx('overflow-auto', className)} style={{ maxHeight }}>
      <table
        className={clsx(resizableColumns ? 'table-fixed' : 'min-w-full')}
        style={resizableColumns ? { width: numericTableWidth, minWidth: '100%' } : undefined}
      >
        {resizableColumns && (
          <colgroup>
            {columns.map((column) => <col key={column.key} style={{ width: getColumnWidth(column) }} />)}
          </colgroup>
        )}
        <thead className={clsx(stickyHeader && 'sticky top-0 z-10')}>
          <tr>
            {columns.map((column) => {
              const canResize = resizableColumns && column.resizable !== false && typeof getColumnWidth(column) === 'number'
              return (
                <th
                  key={column.key}
                  className={clsx(
                    'relative bg-[var(--color-bg-muted)] px-4 py-3 align-middle text-xs font-semibold uppercase tracking-wider text-[var(--color-text-muted)]',
                    column.align === 'center' && 'text-center',
                    column.align === 'right' && 'text-right',
                    !column.align && 'text-left',
                    column.sortable && 'group cursor-pointer hover:text-[var(--color-text-primary)]',
                  )}
                  style={{ width: getColumnWidth(column), minWidth: column.minWidth, maxWidth: column.maxWidth }}
                  onClick={() => column.sortable && handleSortClick(column)}
                >
                  <span className={clsx(
                    'flex min-h-5 items-center whitespace-normal break-words leading-tight',
                    column.align === 'center' && 'justify-center',
                    column.align === 'right' && 'justify-end',
                  )}>
                    {column.title}
                    {renderSortIcon(column)}
                  </span>
                  {canResize && (
                    <button
                      type="button"
                      className="absolute -right-1 top-0 z-20 h-full w-2 cursor-col-resize touch-none select-none outline-none"
                      onPointerDown={(event) => beginResize(event, column)}
                      onClick={(event) => event.stopPropagation()}
                      title="左右拖动调整列宽"
                      aria-label={typeof column.title === 'string' ? '调整“' + column.title + '”列宽' : '调整 ' + column.key + ' 列宽'}
                    >
                      <span className="absolute bottom-2 left-1/2 top-2 w-px -translate-x-1/2 bg-[var(--color-border-default)] opacity-0 transition-opacity hover:opacity-100" />
                    </button>
                  )}
                </th>
              )
            })}
          </tr>
        </thead>
        <tbody className="divide-y divide-[var(--color-border-muted)] bg-[var(--color-bg-surface)]">
          {data.length === 0 ? (
            <tr>
              <td colSpan={columns.length} className="px-4 py-16 text-center">
                <div className="flex flex-col items-center gap-2">
                  <div className="flex h-12 w-12 items-center justify-center rounded-full bg-[var(--color-bg-muted)]">
                    <svg className="h-6 w-6 text-[var(--color-text-muted)]" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M20 13V6a2 2 0 00-2-2H6a2 2 0 00-2 2v7m16 0v5a2 2 0 01-2 2H6a2 2 0 01-2-2v-5m16 0h-2.586a1 1 0 00-.707.293l-2.414 2.414a1 1 0 01-.707.293h-3.172a1 1 0 01-.707-.293l-2.414-2.414A1 1 0 006.586 13H4" />
                    </svg>
                  </div>
                  <span className="text-sm text-[var(--color-text-muted)]">{emptyText}</span>
                </div>
              </td>
            </tr>
          ) : (
            data.map((record, index) => (
              <tr
                key={getRowKey(record, index)}
                className={clsx('transition-colors duration-150 hover:bg-[var(--color-bg-muted)]/50', onRowClick && 'cursor-pointer')}
                onClick={() => onRowClick?.(record)}
              >
                {columns.map((column) => (
                  <td
                    key={column.key}
                    className={clsx(
                      'px-4 py-3.5 text-sm text-[var(--color-text-secondary)]',
                      column.align === 'center' && 'text-center',
                      column.align === 'right' && 'text-right',
                    )}
                  >
                    {column.render ? column.render(record[column.key], record, index) : record[column.key]}
                  </td>
                ))}
              </tr>
            ))
          )}
        </tbody>
      </table>
    </div>
  )
}
