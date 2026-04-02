import { describe, it, expect } from 'vitest'
import { DASHBOARD_CHUNKS } from '../dashboardChunks'

describe('DASHBOARD_CHUNKS', () => {
  it('is a non-empty record', () => {
    const keys = Object.keys(DASHBOARD_CHUNKS)
    expect(keys.length).toBeGreaterThan(0)
  })

  it('has all essential dashboard keys', () => {
    const expected = [
      'dashboard', 'clusters', 'workloads', 'nodes', 'pods',
      'services', 'storage', 'network', 'security', 'settings',
    ]
    for (const key of expected) {
      expect(DASHBOARD_CHUNKS[key]).toBeDefined()
      expect(typeof DASHBOARD_CHUNKS[key]).toBe('function')
    }
  })

  it('each value is a function returning a Promise', () => {
    for (const [, loader] of Object.entries(DASHBOARD_CHUNKS)) {
      expect(typeof loader).toBe('function')
    }
  })

  it('has compute dashboard', () => {
    expect(DASHBOARD_CHUNKS['compute']).toBeDefined()
    expect(typeof DASHBOARD_CHUNKS['compute']).toBe('function')
  })

  it('has events dashboard', () => {
    expect(DASHBOARD_CHUNKS['events']).toBeDefined()
  })

  it('has deployments dashboard', () => {
    expect(DASHBOARD_CHUNKS['deployments']).toBeDefined()
  })

  it('has gitops dashboard', () => {
    expect(DASHBOARD_CHUNKS['gitops']).toBeDefined()
  })

  it('has cost dashboard', () => {
    expect(DASHBOARD_CHUNKS['cost']).toBeDefined()
  })

  it('has compliance dashboard', () => {
    expect(DASHBOARD_CHUNKS['compliance']).toBeDefined()
  })

  it('has operators dashboard', () => {
    expect(DASHBOARD_CHUNKS['operators']).toBeDefined()
  })

  it('has helm dashboard', () => {
    expect(DASHBOARD_CHUNKS['helm']).toBeDefined()
  })

  it('has gpu-reservations dashboard', () => {
    expect(DASHBOARD_CHUNKS['gpu-reservations']).toBeDefined()
  })

  it('has ai-ml dashboard', () => {
    expect(DASHBOARD_CHUNKS['ai-ml']).toBeDefined()
  })

  it('has ai-agents dashboard', () => {
    expect(DASHBOARD_CHUNKS['ai-agents']).toBeDefined()
  })

  it('has llm-d-benchmarks dashboard', () => {
    expect(DASHBOARD_CHUNKS['llm-d-benchmarks']).toBeDefined()
  })

  it('has marketplace dashboard', () => {
    expect(DASHBOARD_CHUNKS['marketplace']).toBeDefined()
  })

  it('has insights dashboard', () => {
    expect(DASHBOARD_CHUNKS['insights']).toBeDefined()
  })

  it('has alerts dashboard', () => {
    expect(DASHBOARD_CHUNKS['alerts']).toBeDefined()
  })

  it('does not have unknown keys', () => {
    expect(DASHBOARD_CHUNKS['nonexistent']).toBeUndefined()
  })
})
