import { useEffect } from 'react'
import { X, PlusCircle, RefreshCw, AlertTriangle } from 'lucide-react'
import { ResetMode } from '../../hooks/useDashboardReset'

interface ResetDialogProps {
  isOpen: boolean
  onClose: () => void
  onReset: (mode: ResetMode) => void
}

/**
 * Dialog for resetting dashboard cards with two options:
 * - Add Missing: Keep current cards and add any missing defaults
 * - Replace All: Reset to only default cards (remove customizations)
 */
export function ResetDialog({ isOpen, onClose, onReset }: ResetDialogProps) {
  // ESC to close
  useEffect(() => {
    if (!isOpen) return

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        e.preventDefault()
        onClose()
      }
    }
    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [isOpen, onClose])

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/80">
      <div className="w-full max-w-md glass rounded-2xl overflow-hidden animate-fade-in-up">
        {/* Header */}
        <div className="flex items-center justify-between px-6 py-4 border-b border-border/50">
          <div>
            <h2 className="text-lg font-medium text-foreground">Reset Dashboard</h2>
            <p className="text-sm text-muted-foreground">
              Choose how to reset your dashboard cards
            </p>
          </div>
          <button
            onClick={onClose}
            className="p-2 rounded-lg hover:bg-secondary/50 text-muted-foreground hover:text-foreground"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Options */}
        <div className="p-6 space-y-3">
          {/* Add Missing Option */}
          <button
            onClick={() => onReset('add_missing')}
            className="w-full p-4 rounded-lg border border-border/50 hover:border-green-500/50 hover:bg-green-500/5 text-left transition-all group"
          >
            <div className="flex items-start gap-4">
              <div className="p-2 rounded-lg bg-green-500/10 text-green-400 group-hover:bg-green-500/20">
                <PlusCircle className="w-5 h-5" />
              </div>
              <div className="flex-1">
                <div className="font-medium text-foreground mb-1">Add Missing Cards</div>
                <p className="text-sm text-muted-foreground">
                  Keep your current cards and add any default cards that are missing.
                  Your customizations will be preserved.
                </p>
              </div>
            </div>
          </button>

          {/* Replace All Option */}
          <button
            onClick={() => onReset('replace')}
            className="w-full p-4 rounded-lg border border-border/50 hover:border-orange-500/50 hover:bg-orange-500/5 text-left transition-all group"
          >
            <div className="flex items-start gap-4">
              <div className="p-2 rounded-lg bg-orange-500/10 text-orange-400 group-hover:bg-orange-500/20">
                <RefreshCw className="w-5 h-5" />
              </div>
              <div className="flex-1">
                <div className="font-medium text-foreground mb-1">Replace All Cards</div>
                <p className="text-sm text-muted-foreground">
                  Remove all current cards and replace them with the default set.
                  This will remove any customizations.
                </p>
                <div className="flex items-center gap-1.5 mt-2 text-xs text-orange-400">
                  <AlertTriangle className="w-3.5 h-3.5" />
                  <span>This action cannot be undone</span>
                </div>
              </div>
            </div>
          </button>
        </div>

        {/* Footer */}
        <div className="flex justify-end px-6 py-4 border-t border-border/50">
          <button
            onClick={onClose}
            className="px-4 py-2 rounded-lg text-muted-foreground hover:text-foreground hover:bg-secondary/50"
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  )
}
