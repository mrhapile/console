import { useState } from 'react'
import { Plus, Layout, RotateCcw } from 'lucide-react'
import { useMissions } from '../../hooks/useMissions'
import { ResetMode } from '../../hooks/useDashboardReset'
import { ResetDialog } from './ResetDialog'

interface FloatingDashboardActionsProps {
  onAddCard: () => void
  onOpenTemplates: () => void
  /** Callback for reset with mode selection */
  onReset?: (mode: ResetMode) => number
  /** Legacy: callback to reset dashboard to default cards (replace mode only) */
  onResetToDefaults?: () => void
  /** Whether the dashboard has been customized from defaults */
  isCustomized?: boolean
}

/**
 * Floating action buttons for Add Card, Templates, and Reset to Defaults.
 * These buttons stay fixed at the bottom right of the viewport
 * so users can always access them when scrolling through dashboards.
 * Shifts left when mission sidebar is open to avoid overlap.
 */
export function FloatingDashboardActions({
  onAddCard,
  onOpenTemplates,
  onReset,
  onResetToDefaults,
  isCustomized,
}: FloatingDashboardActionsProps) {
  const { isSidebarOpen, isSidebarMinimized } = useMissions()
  const [showResetDialog, setShowResetDialog] = useState(false)

  // Shift buttons left based on mission sidebar state
  // w-96 = 384px (full sidebar), w-12 = 48px (minimized)
  const getRightOffset = () => {
    if (!isSidebarOpen) return 'right-6'
    if (isSidebarMinimized) return 'right-[72px]' // 48px + 24px margin
    return 'right-[420px]' // 384px + 36px margin
  }
  const rightOffset = getRightOffset()

  const handleReset = (mode: ResetMode) => {
    setShowResetDialog(false)
    if (onReset) {
      onReset(mode)
    } else if (onResetToDefaults && mode === 'replace') {
      // Legacy support for old onResetToDefaults prop
      onResetToDefaults()
    }
  }

  const showResetButton = isCustomized && (onReset || onResetToDefaults)

  return (
    <>
      <div className={`fixed bottom-20 ${rightOffset} z-40 flex flex-col gap-1.5 transition-all duration-300`}>
        {/* Reset button */}
        {showResetButton && (
          <button
            onClick={() => setShowResetDialog(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground bg-card/95 hover:bg-card border border-border rounded-md shadow-md backdrop-blur-sm transition-all hover:shadow-lg"
            title="Reset dashboard cards"
          >
            <RotateCcw className="w-3.5 h-3.5" />
            Reset
          </button>
        )}
        {/* Templates button */}
        <button
          onClick={onOpenTemplates}
          data-tour="templates"
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground bg-card/95 hover:bg-card border border-border rounded-md shadow-md backdrop-blur-sm transition-all hover:shadow-lg"
          title="Browse dashboard templates"
        >
          <Layout className="w-3.5 h-3.5" />
          Templates
        </button>
        {/* Add Card button */}
        <button
          onClick={onAddCard}
          data-tour="add-card"
          className="flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium bg-gradient-ks text-foreground rounded-md shadow-md transition-all hover:shadow-lg hover:scale-105"
          title="Add a new card"
        >
          <Plus className="w-3.5 h-3.5" />
          Add Card
        </button>
      </div>

      {/* Reset Dialog */}
      <ResetDialog
        isOpen={showResetDialog}
        onClose={() => setShowResetDialog(false)}
        onReset={handleReset}
      />
    </>
  )
}
