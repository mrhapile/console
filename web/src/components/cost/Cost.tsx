import { useState, useEffect, useCallback, memo, useMemo } from 'react'
import { useLocation } from 'react-router-dom'
import { DollarSign, RefreshCw, Hourglass, GripVertical } from 'lucide-react'
import {
  DndContext,
  closestCenter,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
  DragEndEvent,
  DragStartEvent,
  DragOverlay,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  rectSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { useClusters, useGPUNodes } from '../../hooks/useMCP'
import { useGlobalFilters } from '../../hooks/useGlobalFilters'
import { useShowCards } from '../../hooks/useShowCards'
import { useDashboardReset } from '../../hooks/useDashboardReset'
import { CardWrapper } from '../cards/CardWrapper'
import { CARD_COMPONENTS, DEMO_DATA_CARDS } from '../cards/cardRegistry'
import { AddCardModal } from '../dashboard/AddCardModal'
import { TemplatesModal } from '../dashboard/TemplatesModal'
import { ConfigureCardModal } from '../dashboard/ConfigureCardModal'
import { FloatingDashboardActions } from '../dashboard/FloatingDashboardActions'
import { DashboardTemplate } from '../dashboard/templates'
import { formatCardTitle } from '../../lib/formatCardTitle'
import { StatsOverview, StatBlockValue } from '../ui/StatsOverview'

interface CostCard {
  id: string
  card_type: string
  config: Record<string, unknown>
  title?: string
  position?: { w: number; h: number }
}

// Width class lookup for Tailwind (dynamic classes don't work)
const WIDTH_CLASSES: Record<number, string> = {
  3: 'col-span-3',
  4: 'col-span-4',
  5: 'col-span-5',
  6: 'col-span-6',
  7: 'col-span-7',
  8: 'col-span-8',
  9: 'col-span-9',
  10: 'col-span-10',
  11: 'col-span-11',
  12: 'col-span-12',
}

// Sortable Card Component
interface SortableCardProps {
  card: CostCard
  onRemove: () => void
  onConfigure: () => void
  onWidthChange: (width: number) => void
  isDragging: boolean
}

const SortableCard = memo(function SortableCard({
  card,
  onRemove,
  onConfigure,
  onWidthChange,
  isDragging,
}: SortableCardProps) {
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
  } = useSortable({ id: card.id })

  const style = {
    transform: CSS.Transform.toString(transform),
    transition,
    opacity: isDragging ? 0.5 : 1,
  }

  const CardComponent = CARD_COMPONENTS[card.card_type]
  if (!CardComponent) {
    console.warn(`Card component not found: ${card.card_type}`)
    return null
  }

  const width = Math.min(12, Math.max(3, card.position?.w || 6))
  const colSpan = WIDTH_CLASSES[width] || 'col-span-6'

  return (
    <div ref={setNodeRef} style={style} className={colSpan}>
      <CardWrapper
        title={formatCardTitle(card.card_type)}
        onRemove={onRemove}
        onConfigure={onConfigure}
        cardType={card.card_type}
        cardWidth={width}
        onWidthChange={onWidthChange}
        isDemoData={DEMO_DATA_CARDS.has(card.card_type)}
        dragHandle={
          <button {...attributes} {...listeners} className="cursor-grab active:cursor-grabbing">
            <GripVertical className="w-4 h-4 text-muted-foreground" />
          </button>
        }
      >
        <CardComponent config={card.config} />
      </CardWrapper>
    </div>
  )
})

// Drag preview component
function DragPreviewCard({ card }: { card: CostCard }) {
  const CardComponent = CARD_COMPONENTS[card.card_type]
  if (!CardComponent) return null

  const width = Math.min(12, Math.max(3, card.position?.w || 6))
  const colSpan = WIDTH_CLASSES[width] || 'col-span-6'

  return (
    <div className={colSpan}>
      <CardWrapper
        title={formatCardTitle(card.card_type)}
        cardType={card.card_type}
      >
        <CardComponent config={card.config} />
      </CardWrapper>
    </div>
  )
}

const COST_CARDS_KEY = 'kubestellar-cost-cards'

// Default cards for the Cost dashboard
const DEFAULT_COST_CARDS: CostCard[] = [
  { id: 'cost-1', card_type: 'cluster_costs', config: {}, position: { w: 6, h: 4 } },
  { id: 'cost-2', card_type: 'opencost_overview', config: {}, position: { w: 6, h: 4 } },
  { id: 'cost-3', card_type: 'kubecost_overview', config: {}, position: { w: 6, h: 4 } },
  { id: 'cost-4', card_type: 'resource_usage', config: {}, position: { w: 3, h: 2 } },
  { id: 'cost-5', card_type: 'resource_capacity', config: {}, position: { w: 3, h: 2 } },
]

function loadCostCards(): CostCard[] {
  try {
    const stored = localStorage.getItem(COST_CARDS_KEY)
    if (stored) {
      return JSON.parse(stored)
    }
  } catch {
    // Fall through to return defaults
  }
  return DEFAULT_COST_CARDS
}

export function Cost() {
  const location = useLocation()
  const { clusters, isLoading, refetch, lastUpdated, isRefreshing } = useClusters()
  const { nodes: gpuNodes } = useGPUNodes()
  const { selectedClusters: globalSelectedClusters, isAllClustersSelected } = useGlobalFilters()
  const { showCards, expandCards } = useShowCards('kubestellar-cost')

  const [cards, setCards] = useState<CostCard[]>(() => loadCostCards())

  const [showAddCard, setShowAddCard] = useState(false)
  const [showTemplates, setShowTemplates] = useState(false)
  const [configuringCard, setConfiguringCard] = useState<CostCard | null>(null)
  const [autoRefresh, setAutoRefresh] = useState(true)
  const [activeId, setActiveId] = useState<string | null>(null)

  // Reset functionality using shared hook
  const { isCustomized, setCustomized, reset } = useDashboardReset({
    storageKey: COST_CARDS_KEY,
    defaultCards: DEFAULT_COST_CARDS,
    setCards,
    cards,
  })

  // Save cards to localStorage (mark as customized)
  useEffect(() => {
    localStorage.setItem(COST_CARDS_KEY, JSON.stringify(cards))
    setCustomized(true)
  }, [cards, setCustomized])

  // DnD sensors
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 8 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates })
  )

  const handleDragStart = useCallback((event: DragStartEvent) => {
    setActiveId(event.active.id as string)
  }, [])

  const handleDragEnd = useCallback((event: DragEndEvent) => {
    const { active, over } = event
    setActiveId(null)

    if (over && active.id !== over.id) {
      setCards(prev => {
        const oldIndex = prev.findIndex(c => c.id === active.id)
        const newIndex = prev.findIndex(c => c.id === over.id)
        return arrayMove(prev, oldIndex, newIndex)
      })
    }
  }, [])

  // Trigger refresh when navigating to this page
  useEffect(() => {
    refetch()
  }, [location.key]) // eslint-disable-line react-hooks/exhaustive-deps

  // Auto-refresh every 30 seconds
  useEffect(() => {
    if (!autoRefresh) return
    const interval = setInterval(() => refetch(), 30000)
    return () => clearInterval(interval)
  }, [autoRefresh, refetch])

  const handleRefresh = useCallback(() => {
    refetch()
  }, [refetch])

  const handleAddCards = useCallback((newCards: Array<{ type: string; title: string; config: Record<string, unknown> }>) => {
    const cardsToAdd: CostCard[] = newCards.map(card => ({
      id: `card-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      card_type: card.type,
      config: card.config,
      title: card.title,
    }))
    setCards(prev => [...prev, ...cardsToAdd])
    expandCards()
    setShowAddCard(false)
  }, [expandCards])

  const handleRemoveCard = useCallback((cardId: string) => {
    setCards(prev => prev.filter(c => c.id !== cardId))
  }, [])

  const handleConfigureCard = useCallback((cardId: string) => {
    const card = cards.find(c => c.id === cardId)
    if (card) setConfiguringCard(card)
  }, [cards])

  const handleSaveCardConfig = useCallback((cardId: string, config: Record<string, unknown>) => {
    setCards(prev => prev.map(c =>
      c.id === cardId ? { ...c, config } : c
    ))
    setConfiguringCard(null)
  }, [])

  const handleWidthChange = useCallback((cardId: string, newWidth: number) => {
    setCards(prev => prev.map(c =>
      c.id === cardId ? { ...c, position: { ...(c.position || { w: 6, h: 2 }), w: newWidth } } : c
    ))
  }, [])

  const applyTemplate = useCallback((template: DashboardTemplate) => {
    const newCards: CostCard[] = template.cards.map(card => ({
      id: `card-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`,
      card_type: card.card_type,
      config: card.config || {},
      title: card.title,
    }))
    setCards(newCards)
    expandCards()
    setShowTemplates(false)
  }, [expandCards])

  // Filter clusters based on global selection
  const filteredClusters = clusters.filter(c =>
    isAllClustersSelected || globalSelectedClusters.includes(c.name)
  )
  const reachableClusters = filteredClusters.filter(c => c.reachable !== false)

  // Cloud provider pricing (same as ClusterCosts card for consistency)
  type CloudProvider = 'estimate' | 'aws' | 'gcp' | 'azure' | 'oci' | 'openshift'
  const CLOUD_PRICING: Record<CloudProvider, { cpu: number; memory: number; gpu: number }> = {
    estimate: { cpu: 0.05, memory: 0.01, gpu: 2.50 },
    aws: { cpu: 0.048, memory: 0.012, gpu: 3.06 },
    gcp: { cpu: 0.0475, memory: 0.0064, gpu: 2.48 },
    azure: { cpu: 0.05, memory: 0.011, gpu: 2.07 },
    oci: { cpu: 0.025, memory: 0.0015, gpu: 2.95 },
    openshift: { cpu: 0.048, memory: 0.012, gpu: 3.00 },
  }

  // Read provider overrides from localStorage (same key as ClusterCosts card)
  const [providerOverrides, setProviderOverrides] = useState<Record<string, CloudProvider>>(() => {
    try {
      const saved = localStorage.getItem('kubestellar-cluster-provider-overrides')
      return saved ? JSON.parse(saved) : {}
    } catch { return {} }
  })

  // Listen for localStorage changes (when user changes provider in ClusterCosts card)
  useEffect(() => {
    const handleStorageChange = () => {
      try {
        const saved = localStorage.getItem('kubestellar-cluster-provider-overrides')
        setProviderOverrides(saved ? JSON.parse(saved) : {})
      } catch { /* ignore */ }
    }
    window.addEventListener('storage', handleStorageChange)
    // Also poll for changes since storage event doesn't fire for same-tab changes
    const interval = setInterval(handleStorageChange, 1000)
    return () => {
      window.removeEventListener('storage', handleStorageChange)
      clearInterval(interval)
    }
  }, [])

  // Detect cloud provider from cluster name (matches ClusterCosts logic)
  const detectClusterProvider = (name: string, context?: string): CloudProvider => {
    // Check for manual override first
    if (providerOverrides[name]) {
      return providerOverrides[name]
    }
    const searchStr = `${name} ${context || ''}`.toLowerCase()
    if (searchStr.includes('openshift') || searchStr.includes('ocp') || searchStr.includes('rosa') || searchStr.includes('aro')) return 'openshift'
    if (searchStr.includes('eks') || searchStr.includes('aws') || searchStr.includes('amazon')) return 'aws'
    if (searchStr.includes('gke') || searchStr.includes('gcp') || searchStr.includes('google')) return 'gcp'
    if (searchStr.includes('aks') || searchStr.includes('azure') || searchStr.includes('microsoft')) return 'azure'
    if (searchStr.includes('oke') || searchStr.includes('oci') || searchStr.includes('oracle') || name.toLowerCase() === 'prow') return 'oci'
    return 'estimate'
  }

  // Count GPUs from GPU nodes
  const gpuByCluster = useMemo(() => {
    const map: Record<string, number> = {}
    gpuNodes.forEach(node => {
      const clusterKey = node.cluster.split('/')[0]
      map[clusterKey] = (map[clusterKey] || 0) + node.gpuCount
    })
    return map
  }, [gpuNodes])

  // Calculate per-cluster costs (matches ClusterCosts card exactly)
  const costStats = useMemo(() => {
    let totalCPU = 0
    let totalMemoryGB = 0
    let totalGPUs = 0
    let totalMonthly = 0
    let cpuMonthly = 0
    let memoryMonthly = 0
    let gpuMonthly = 0

    reachableClusters.forEach(cluster => {
      const cpus = cluster.cpuCores || 0
      const memory = 32 * (cluster.nodeCount || 0) // Estimate 32GB per node (matches ClusterCosts)
      const gpus = gpuByCluster[cluster.name] || 0

      // Get per-cluster pricing based on detected provider
      const provider = detectClusterProvider(cluster.name, cluster.context)
      const pricing = CLOUD_PRICING[provider]

      const clusterHourly = (cpus * pricing.cpu) + (memory * pricing.memory) + (gpus * pricing.gpu)
      const clusterMonthly = clusterHourly * 24 * 30

      totalCPU += cpus
      totalMemoryGB += memory
      totalGPUs += gpus
      totalMonthly += clusterMonthly
      cpuMonthly += cpus * pricing.cpu * 24 * 30
      memoryMonthly += memory * pricing.memory * 24 * 30
      gpuMonthly += gpus * pricing.gpu * 24 * 30
    })

    const totalStorageGB = reachableClusters.reduce((sum, c) => sum + (c.storageGB || 0), 0)
    const storageCostPerGBMonth = 0.10
    const storageMonthly = totalStorageGB * storageCostPerGBMonth

    return {
      totalCPU,
      totalMemoryGB,
      totalGPUs,
      totalStorageGB,
      totalMonthly: totalMonthly + storageMonthly,
      cpuMonthly,
      memoryMonthly,
      gpuMonthly,
      storageMonthly,
    }
  }, [reachableClusters, gpuByCluster, providerOverrides])

  // Stats value getter for the configurable StatsOverview component
  const getStatValue = useCallback((blockId: string): StatBlockValue => {
    switch (blockId) {
      case 'total_cost':
        return { value: `$${Math.round(costStats.totalMonthly).toLocaleString()}`, sublabel: 'est. monthly' }
      case 'cpu_cost':
        return { value: `$${Math.round(costStats.cpuMonthly).toLocaleString()}`, sublabel: `${costStats.totalCPU} cores` }
      case 'memory_cost':
        return { value: `$${Math.round(costStats.memoryMonthly).toLocaleString()}`, sublabel: `${costStats.totalMemoryGB} GB` }
      case 'storage_cost':
        return { value: `$${Math.round(costStats.storageMonthly).toLocaleString()}`, sublabel: costStats.totalStorageGB >= 1024 ? `${(costStats.totalStorageGB / 1024).toFixed(1)} TB` : `${Math.round(costStats.totalStorageGB)} GB` }
      case 'network_cost':
        return { value: '$0', sublabel: 'not tracked' }
      case 'gpu_cost':
        return { value: `$${Math.round(costStats.gpuMonthly).toLocaleString()}`, sublabel: `${costStats.totalGPUs} GPUs` }
      default:
        return { value: 0 }
    }
  }, [costStats])

  // Transform card for ConfigureCardModal
  const configureCard = configuringCard ? {
    id: configuringCard.id,
    card_type: configuringCard.card_type,
    config: configuringCard.config,
    title: configuringCard.title,
  } : null

  return (
    <div className="pt-16">
      {/* Header */}
      <div className="mb-6">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div>
              <h1 className="text-2xl font-bold text-foreground flex items-center gap-2">
                <DollarSign className="w-6 h-6 text-green-400" />
                Cost Management
              </h1>
              <p className="text-muted-foreground">Monitor and optimize resource costs across clusters</p>
            </div>
            {isRefreshing && (
              <span className="flex items-center gap-1 text-xs text-amber-400 animate-pulse" title="Updating...">
                <Hourglass className="w-3 h-3" />
                <span>Updating</span>
              </span>
            )}
          </div>
          <div className="flex items-center gap-3">
            <label htmlFor="cost-auto-refresh" className="flex items-center gap-1.5 cursor-pointer text-xs text-muted-foreground" title="Auto-refresh every 30s">
              <input
                type="checkbox"
                id="cost-auto-refresh"
                checked={autoRefresh}
                onChange={(e) => setAutoRefresh(e.target.checked)}
                className="rounded border-border w-3.5 h-3.5"
              />
              Auto
            </label>
            <button
              onClick={handleRefresh}
              disabled={isRefreshing}
              className="p-2 rounded-lg hover:bg-secondary transition-colors disabled:opacity-50"
              title="Refresh data"
            >
              <RefreshCw className={`w-4 h-4 ${isRefreshing ? 'animate-spin' : ''}`} />
            </button>
          </div>
        </div>
      </div>

      {/* Configurable Stats Overview */}
      <StatsOverview
        dashboardType="cost"
        getStatValue={getStatValue}
        hasData={reachableClusters.length > 0}
        isLoading={isLoading}
        lastUpdated={lastUpdated}
        collapsedStorageKey="kubestellar-cost-stats-collapsed"
      />

      {/* Cards Grid */}
      {showCards && (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragStart={handleDragStart}
          onDragEnd={handleDragEnd}
        >
          <SortableContext items={cards.map(c => c.id)} strategy={rectSortingStrategy}>
            <div className="grid grid-cols-12 gap-4 pb-32">
              {cards.map(card => (
                <SortableCard
                  key={card.id}
                  card={card}
                  onRemove={() => handleRemoveCard(card.id)}
                  onConfigure={() => handleConfigureCard(card.id)}
                  onWidthChange={(width) => handleWidthChange(card.id, width)}
                  isDragging={activeId === card.id}
                />
              ))}
            </div>
          </SortableContext>

          <DragOverlay>
            {activeId ? (
              <div className="opacity-80 rotate-3 scale-105">
                <DragPreviewCard card={cards.find(c => c.id === activeId)!} />
              </div>
            ) : null}
          </DragOverlay>
        </DndContext>
      )}

      {/* Floating Actions */}
      <FloatingDashboardActions
        onAddCard={() => setShowAddCard(true)}
        onOpenTemplates={() => setShowTemplates(true)}
        onReset={reset}
        isCustomized={isCustomized}
      />

      {/* Add Card Modal */}
      <AddCardModal
        isOpen={showAddCard}
        onClose={() => setShowAddCard(false)}
        onAddCards={handleAddCards}
        existingCardTypes={cards.map(c => c.card_type)}
      />

      {/* Templates Modal */}
      <TemplatesModal
        isOpen={showTemplates}
        onClose={() => setShowTemplates(false)}
        onApplyTemplate={applyTemplate}
      />

      {/* Configure Card Modal */}
      <ConfigureCardModal
        isOpen={!!configuringCard}
        card={configureCard}
        onClose={() => setConfiguringCard(null)}
        onSave={(cardId, config) => {
          handleSaveCardConfig(cardId, config)
        }}
      />
    </div>
  )
}
