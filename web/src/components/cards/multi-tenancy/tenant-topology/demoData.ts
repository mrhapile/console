/**
 * Demo data for the Tenant Topology card.
 *
 * Mixed state: OVN and KubeFlex detected, K3s and KubeVirt not installed.
 * K3s/KubeVirt nodes appear grayed out with dashed borders. No throughput
 * on connections to missing components.
 */

import type { TenantTopologyData } from './useTenantTopology'

export const DEMO_TENANT_TOPOLOGY: TenantTopologyData = {
  ovnDetected: true,
  ovnHealthy: true,
  kubeflexDetected: true,
  kubeflexHealthy: true,
  k3sDetected: false,
  k3sHealthy: false,
  kubevirtDetected: false,
  kubevirtHealthy: false,
  kvEth0Rate: 0,
  kvEth1Rate: 0,
  k3sEth0Rate: 0,
  k3sEth1Rate: 0,
  isLoading: false,
  isDemoData: true,
}
