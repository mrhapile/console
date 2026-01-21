import { useState } from 'react'
import { Bug } from 'lucide-react'
import { FeatureRequestModal } from './FeatureRequestModal'

export function FeatureRequestButton() {
  const [isModalOpen, setIsModalOpen] = useState(false)

  const handleClick = () => {
    setIsModalOpen(true)
  }

  return (
    <>
      <button
        onClick={handleClick}
        className="relative p-2 rounded-lg hover:bg-secondary/50 text-muted-foreground hover:text-foreground transition-colors"
        title="Report a bug or request a feature"
      >
        <Bug className="w-5 h-5" />
      </button>

      <FeatureRequestModal
        isOpen={isModalOpen}
        onClose={() => setIsModalOpen(false)}
      />
    </>
  )
}
