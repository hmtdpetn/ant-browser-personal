import { useNavigate } from 'react-router-dom'
import { BookOpen, Settings2 } from 'lucide-react'
import { Button } from '../../../shared/components'

interface AutomationEntryActionsProps {
  onBeforeNavigate?: () => void
  size?: 'sm' | 'md' | 'lg'
}

export function AutomationEntryActions({
  onBeforeNavigate,
  size = 'sm',
}: AutomationEntryActionsProps) {
  const navigate = useNavigate()

  const openRoute = (path: string) => {
    onBeforeNavigate?.()
    navigate(path)
  }

  return (
    <div className="flex flex-wrap gap-2">
      <Button
        size={size}
        variant="secondary"
        onClick={() => openRoute('/system/docs')}
      >
        <BookOpen className="h-4 w-4" />
        文档中心
      </Button>
      <Button
        size={size}
        variant="secondary"
        onClick={() => openRoute('/settings')}
      >
        <Settings2 className="h-4 w-4" />
        运行时设置
      </Button>
    </div>
  )
}
