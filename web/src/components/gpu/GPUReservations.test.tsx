import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

vi.mock('../../lib/demoMode', () => ({
  isDemoMode: () => true, getDemoMode: () => true, isNetlifyDeployment: false,
  isDemoModeForced: false, canToggleDemoMode: () => true, setDemoMode: vi.fn(),
  toggleDemoMode: vi.fn(), subscribeDemoMode: () => () => { },
  isDemoToken: () => true, hasRealToken: () => false, setDemoToken: vi.fn(),
  isFeatureEnabled: () => true,
}))
vi.mock('../../hooks/useDemoMode', () => ({
  getDemoMode: () => true, default: () => true, useDemoMode: () => true, isDemoModeForced: false,
}))
vi.mock('../../lib/analytics', () => ({
  emitNavigate: vi.fn(), emitLogin: vi.fn(), emitEvent: vi.fn(), analyticsReady: Promise.resolve(),
}))
vi.mock('../../hooks/useTokenUsage', () => ({
  useTokenUsage: () => ({ usage: { total: 0, remaining: 0, used: 0 }, isLoading: false }),
  tokenUsageTracker: { getUsage: () => ({ total: 0, remaining: 0, used: 0 }), trackRequest: vi.fn(), getSettings: () => ({ enabled: false }) },
}))

vi.mock('../../lib/dashboards/DashboardPage', () => ({
  DashboardPage: ({ title, subtitle, children, beforeCards }: { title: string; subtitle?: string; children?: React.ReactNode; beforeCards?: React.ReactNode }) => (
    <div data-testid="dashboard-page" data-title={title} data-subtitle={subtitle}>
      <h1>{title}</h1>
      {subtitle && <p>{subtitle}</p>}
      {beforeCards}
      {children}
    </div>
  ),
}))

vi.mock('../../hooks/useMCP', () => ({
  useClusters: () => ({
    clusters: [], isLoading: false, isRefreshing: false, refetch: vi.fn(), error: null,
  }),
  useGPUNodes: () => ({ nodes: [], isLoading: false, refetch: vi.fn() }),
  useResourceQuotas: () => ({ resourceQuotas: [] }),
  useNamespaces: () => ({ namespaces: [], isLoading: false }),
}))

vi.mock('../../hooks/useGPUReservations', () => ({
  useGPUReservations: () => ({
    reservations: [],
    createReservation: vi.fn(),
    updateReservation: vi.fn(),
    deleteReservation: vi.fn(),
    activateReservation: vi.fn(),
  }),
}))

vi.mock('../../hooks/useGlobalFilters', () => ({
  useGlobalFilters: () => ({
    selectedClusters: [], isAllClustersSelected: true,
    customFilter: '', filterByCluster: (items: unknown[]) => items,
  }),
}))

vi.mock('../../lib/unified/demo', () => ({
  useIsModeSwitching: () => false,
}))

vi.mock('../../hooks/useDrillDown', () => ({
  useDrillDownActions: () => ({
    drillToAllGPU: vi.fn(),
  }),
}))

vi.mock('../../hooks/useUniversalStats', () => ({
  useUniversalStats: () => ({ getStatValue: () => ({ value: 0 }) }),
  createMergedStatValueGetter: () => () => ({ value: 0 }),
}))

vi.mock('../../hooks/useAIMode', () => ({
  useAIMode: () => ({ isFeatureEnabled: () => true }),
  getAIMode: () => 'basic',
}))

vi.mock('../../lib/auth', () => ({
  useAuth: () => ({ user: { login: 'test-user', name: 'Test User' } }),
}))

vi.mock('../ui/Toast', () => ({
  useToast: () => ({ showToast: vi.fn() }),
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({ t: (key: string) => key, i18n: { language: 'en' } }),
}))

import { GPUReservations } from './GPUReservations'

describe('GPUReservations Component', () => {
  const renderGPU = () =>
    render(
      <MemoryRouter>
        <GPUReservations />
      </MemoryRouter>
    )

  it('renders without crashing', () => {
    expect(() => renderGPU()).not.toThrow()
  })

  it('renders the GPU reservations title', () => {
    renderGPU()
    expect(screen.getAllByText(/gpu/i).length).toBeGreaterThan(0)
  })
})
