import { useState } from 'react'
import { createPortal } from 'react-dom'
import { X, Bug, Sparkles, Loader2, ExternalLink } from 'lucide-react'
import { useFeatureRequests, type RequestType } from '../../hooks/useFeatureRequests'

interface FeatureRequestModalProps {
  isOpen: boolean
  onClose: () => void
}

export function FeatureRequestModal({ isOpen, onClose }: FeatureRequestModalProps) {
  const { createRequest, isSubmitting } = useFeatureRequests()
  const [requestType, setRequestType] = useState<RequestType>('bug')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<{ issueUrl?: string } | null>(null)

  if (!isOpen) return null

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError(null)

    if (title.length < 5) {
      setError('Title must be at least 5 characters')
      return
    }
    if (description.length < 10) {
      setError('Description must be at least 10 characters')
      return
    }

    try {
      const result = await createRequest({
        title,
        description,
        request_type: requestType,
      })
      setSuccess({ issueUrl: result.github_issue_url })
      // Reset form after short delay
      setTimeout(() => {
        setTitle('')
        setDescription('')
        setRequestType('bug')
        setSuccess(null)
        onClose()
      }, 3000)
    } catch (err) {
      setError('Failed to submit request. Please try again.')
    }
  }

  const handleClose = () => {
    if (!isSubmitting) {
      setTitle('')
      setDescription('')
      setRequestType('bug')
      setError(null)
      setSuccess(null)
      onClose()
    }
  }

  const modalContent = (
    <>
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-black/50 z-[9999]"
        onClick={handleClose}
      />

      {/* Modal */}
      <div className="fixed inset-0 z-[9999] overflow-y-auto pointer-events-none">
        <div className="flex min-h-full items-start justify-center p-4 pt-20">
          <div
            className="bg-background border border-border rounded-lg shadow-xl w-full max-w-lg my-8 pointer-events-auto"
            onClick={e => e.stopPropagation()}
          >
          {/* Header */}
          <div className="p-4 border-b border-border flex items-center justify-between">
            <h2 className="text-lg font-semibold text-foreground">
              {requestType === 'bug' ? 'Report a Bug' : 'Request a Feature'}
            </h2>
            <button
              onClick={handleClose}
              disabled={isSubmitting}
              className="p-1 rounded hover:bg-secondary/50 text-muted-foreground disabled:opacity-50"
            >
              <X className="w-5 h-5" />
            </button>
          </div>

          {/* Content */}
          {success ? (
            <div className="p-6 text-center">
              <div className="w-12 h-12 mx-auto mb-4 rounded-full bg-green-500/20 flex items-center justify-center">
                <Sparkles className="w-6 h-6 text-green-400" />
              </div>
              <h3 className="text-lg font-medium text-foreground mb-2">
                Request Submitted!
              </h3>
              <p className="text-sm text-muted-foreground mb-4">
                Our AI will analyze your request and create a fix.
                You'll be notified when a preview is ready.
              </p>
              {success.issueUrl && (
                <a
                  href={success.issueUrl}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex items-center gap-1 text-sm text-purple-400 hover:text-purple-300"
                >
                  View on GitHub
                  <ExternalLink className="w-3 h-3" />
                </a>
              )}
            </div>
          ) : (
            <form onSubmit={handleSubmit}>
              <div className="p-4 space-y-4">
                {/* Type Selection */}
                <div className="flex gap-2">
                  <button
                    type="button"
                    onClick={() => setRequestType('bug')}
                    className={`flex-1 p-3 rounded-lg border transition-colors flex items-center justify-center gap-2 ${
                      requestType === 'bug'
                        ? 'bg-red-500/20 border-red-500/50 text-red-400'
                        : 'border-border text-muted-foreground hover:border-muted-foreground'
                    }`}
                  >
                    <Bug className="w-4 h-4" />
                    Bug Report
                  </button>
                  <button
                    type="button"
                    onClick={() => setRequestType('feature')}
                    className={`flex-1 p-3 rounded-lg border transition-colors flex items-center justify-center gap-2 ${
                      requestType === 'feature'
                        ? 'bg-purple-500/20 border-purple-500/50 text-purple-400'
                        : 'border-border text-muted-foreground hover:border-muted-foreground'
                    }`}
                  >
                    <Sparkles className="w-4 h-4" />
                    Feature Request
                  </button>
                </div>

                {/* Title */}
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    Title
                  </label>
                  <input
                    type="text"
                    value={title}
                    onChange={e => setTitle(e.target.value)}
                    placeholder={
                      requestType === 'bug'
                        ? 'e.g., Dashboard not loading cluster data'
                        : 'e.g., Add dark mode toggle to settings'
                    }
                    className="w-full px-3 py-2 bg-secondary/50 border border-border rounded-lg text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-purple-500/50"
                    disabled={isSubmitting}
                  />
                </div>

                {/* Description */}
                <div>
                  <label className="block text-sm font-medium text-foreground mb-1">
                    Description
                  </label>
                  <textarea
                    value={description}
                    onChange={e => setDescription(e.target.value)}
                    placeholder={
                      requestType === 'bug'
                        ? 'Describe what happened, what you expected, and steps to reproduce...'
                        : 'Describe the feature you would like to see and why it would be useful...'
                    }
                    rows={5}
                    className="w-full px-3 py-2 bg-secondary/50 border border-border rounded-lg text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-purple-500/50 resize-none"
                    disabled={isSubmitting}
                  />
                </div>

                {/* Error */}
                {error && (
                  <p className="text-sm text-red-400">{error}</p>
                )}

                {/* Info */}
                <p className="text-xs text-muted-foreground">
                  Your request will be analyzed by our AI, which will attempt to
                  create a fix automatically. You'll receive a notification when
                  a preview is ready for testing.
                </p>
              </div>

              {/* Footer */}
              <div className="p-4 border-t border-border flex justify-end gap-2">
                <button
                  type="button"
                  onClick={handleClose}
                  disabled={isSubmitting}
                  className="px-4 py-2 text-sm rounded-lg border border-border text-muted-foreground hover:text-foreground hover:bg-secondary/50 transition-colors disabled:opacity-50"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={isSubmitting}
                  className="px-4 py-2 text-sm rounded-lg bg-purple-500 hover:bg-purple-600 text-white transition-colors disabled:opacity-50 flex items-center gap-2"
                >
                  {isSubmitting ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" />
                      Submitting...
                    </>
                  ) : (
                    'Submit'
                  )}
                </button>
              </div>
            </form>
          )}
        </div>
        </div>
      </div>
    </>
  )

  return createPortal(modalContent, document.body)
}
