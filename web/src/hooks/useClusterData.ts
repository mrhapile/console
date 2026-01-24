/**
 * Aggregated cluster data hook for drill-down views
 * Combines data from multiple MCP hooks for convenience
 */

import { useClusters, usePods, useDeployments, useNamespaces, useEvents, useHelmReleases, useOperatorSubscriptions, useSecurityIssues } from './useMCP'

export function useClusterData() {
  const { clusters } = useClusters()
  const { pods } = usePods()
  const { deployments } = useDeployments()
  const { namespaces } = useNamespaces()
  const { events } = useEvents()
  const { releases: helmReleases } = useHelmReleases()
  const { subscriptions: operatorSubscriptions } = useOperatorSubscriptions()
  const { issues: securityIssues } = useSecurityIssues()

  return {
    clusters,
    pods,
    deployments,
    namespaces,
    events,
    helmReleases,
    operatorSubscriptions,
    securityIssues,
  }
}
