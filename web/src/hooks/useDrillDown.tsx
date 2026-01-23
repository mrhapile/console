import { createContext, useContext, useState, useCallback, ReactNode } from 'react'

// Types for drill-down navigation
export type DrillDownViewType =
  | 'cluster'
  | 'namespace'
  | 'deployment'
  | 'replicaset'
  | 'pod'
  | 'service'
  | 'configmap'
  | 'secret'
  | 'serviceaccount'
  | 'pvc'
  | 'job'
  | 'hpa'
  | 'node'
  | 'events'
  | 'logs'
  | 'gpu-node'
  | 'yaml'
  | 'resources'
  | 'custom'
  // Phase 2: GitOps and operational views
  | 'helm'
  | 'argoapp'
  | 'kustomization'
  | 'drift'
  // Phase 2: Policy and compliance views
  | 'policy'
  | 'crd'
  // Phase 2: Alerting and monitoring views
  | 'alert'
  | 'alertrule'
  // Phase 2: Cost and RBAC views
  | 'cost'
  | 'rbac'
  // Phase 2: Operator views
  | 'operator'

export interface DrillDownView {
  type: DrillDownViewType
  title: string
  subtitle?: string
  data: Record<string, unknown>
  // Optional custom component to render
  customComponent?: ReactNode
}

export interface DrillDownState {
  isOpen: boolean
  stack: DrillDownView[]
  currentView: DrillDownView | null
}

interface DrillDownContextType {
  state: DrillDownState
  // Open drill-down with initial view
  open: (view: DrillDownView) => void
  // Push a new view onto the stack (drill deeper)
  push: (view: DrillDownView) => void
  // Pop the current view (go back)
  pop: () => void
  // Go back to a specific index in the stack
  goTo: (index: number) => void
  // Close the drill-down modal
  close: () => void
  // Replace current view
  replace: (view: DrillDownView) => void
}

const DrillDownContext = createContext<DrillDownContextType | null>(null)

export function DrillDownProvider({ children }: { children: ReactNode }) {
  const [state, setState] = useState<DrillDownState>({
    isOpen: false,
    stack: [],
    currentView: null,
  })

  const open = useCallback((view: DrillDownView) => {
    setState({
      isOpen: true,
      stack: [view],
      currentView: view,
    })
  }, [])

  const push = useCallback((view: DrillDownView) => {
    setState(prev => ({
      ...prev,
      stack: [...prev.stack, view],
      currentView: view,
    }))
  }, [])

  const pop = useCallback(() => {
    setState(prev => {
      if (prev.stack.length <= 1) {
        return { isOpen: false, stack: [], currentView: null }
      }
      const newStack = prev.stack.slice(0, -1)
      return {
        ...prev,
        stack: newStack,
        currentView: newStack[newStack.length - 1],
      }
    })
  }, [])

  const goTo = useCallback((index: number) => {
    setState(prev => {
      if (index < 0 || index >= prev.stack.length) return prev
      const newStack = prev.stack.slice(0, index + 1)
      return {
        ...prev,
        stack: newStack,
        currentView: newStack[newStack.length - 1],
      }
    })
  }, [])

  const close = useCallback(() => {
    setState({ isOpen: false, stack: [], currentView: null })
  }, [])

  const replace = useCallback((view: DrillDownView) => {
    setState(prev => {
      const newStack = [...prev.stack.slice(0, -1), view]
      return {
        ...prev,
        stack: newStack,
        currentView: view,
      }
    })
  }, [])

  return (
    <DrillDownContext.Provider value={{ state, open, push, pop, goTo, close, replace }}>
      {children}
    </DrillDownContext.Provider>
  )
}

export function useDrillDown() {
  const context = useContext(DrillDownContext)
  if (!context) {
    throw new Error('useDrillDown must be used within a DrillDownProvider')
  }
  return context
}

// Helper to generate a unique key for a view to detect duplicates
function getViewKey(view: DrillDownView): string {
  const { type, data } = view
  switch (type) {
    case 'cluster':
      return `cluster:${data.cluster}`
    case 'namespace':
      return `namespace:${data.cluster}:${data.namespace}`
    case 'deployment':
      return `deployment:${data.cluster}:${data.namespace}:${data.deployment}`
    case 'replicaset':
      return `replicaset:${data.cluster}:${data.namespace}:${data.replicaset}`
    case 'pod':
      return `pod:${data.cluster}:${data.namespace}:${data.pod}`
    case 'configmap':
      return `configmap:${data.cluster}:${data.namespace}:${data.configmap}`
    case 'secret':
      return `secret:${data.cluster}:${data.namespace}:${data.secret}`
    case 'serviceaccount':
      return `serviceaccount:${data.cluster}:${data.namespace}:${data.serviceaccount}`
    case 'pvc':
      return `pvc:${data.cluster}:${data.namespace}:${data.pvc}`
    case 'job':
      return `job:${data.cluster}:${data.namespace}:${data.job}`
    case 'hpa':
      return `hpa:${data.cluster}:${data.namespace}:${data.hpa}`
    case 'service':
      return `service:${data.cluster}:${data.namespace}:${data.service}`
    case 'node':
    case 'gpu-node':
      return `node:${data.cluster}:${data.node}`
    case 'logs':
      return `logs:${data.cluster}:${data.namespace}:${data.pod}:${data.container || ''}`
    case 'events':
      return `events:${data.cluster}:${data.namespace || ''}:${data.objectName || ''}`
    default:
      return `${type}:${JSON.stringify(data)}`
  }
}

// Helper hook to create drill-down actions
export function useDrillDownActions() {
  const { state, open, push, goTo } = useDrillDown()

  // Helper to navigate - checks if view already exists in stack
  const openOrPush = useCallback((view: DrillDownView) => {
    if (!state.isOpen) {
      open(view)
      return
    }

    // Check if this view already exists in the stack
    const viewKey = getViewKey(view)
    const existingIndex = state.stack.findIndex(v => getViewKey(v) === viewKey)

    if (existingIndex >= 0) {
      // Navigate to existing view instead of pushing duplicate
      goTo(existingIndex)
    } else {
      push(view)
    }
  }, [state.isOpen, state.stack, open, push, goTo])

  const drillToCluster = useCallback((cluster: string, clusterData?: Record<string, unknown>) => {
    openOrPush({
      type: 'cluster',
      title: cluster.split('/').pop() || cluster,
      subtitle: 'Cluster Overview',
      data: { cluster, ...clusterData },
    })
  }, [openOrPush])

  const drillToNamespace = useCallback((cluster: string, namespace: string) => {
    openOrPush({
      type: 'namespace',
      title: namespace,
      subtitle: `Namespace in ${cluster.split('/').pop()}`,
      data: { cluster, namespace },
    })
  }, [openOrPush])

  const drillToDeployment = useCallback((cluster: string, namespace: string, deployment: string, deploymentData?: Record<string, unknown>) => {
    openOrPush({
      type: 'deployment',
      title: deployment,
      subtitle: `Deployment in ${namespace}`,
      data: { cluster, namespace, deployment, ...deploymentData },
    })
  }, [openOrPush])

  const drillToPod = useCallback((cluster: string, namespace: string, pod: string, podData?: Record<string, unknown>) => {
    openOrPush({
      type: 'pod',
      title: pod,
      data: { cluster, namespace, pod, ...podData },
    })
  }, [openOrPush])

  const drillToLogs = useCallback((cluster: string, namespace: string, pod: string, container?: string) => {
    openOrPush({
      type: 'logs',
      title: `Logs: ${pod}`,
      subtitle: container ? `Container: ${container}` : 'All containers',
      data: { cluster, namespace, pod, container },
    })
  }, [openOrPush])

  const drillToEvents = useCallback((cluster: string, namespace?: string, objectName?: string) => {
    openOrPush({
      type: 'events',
      title: objectName ? `Events: ${objectName}` : 'Events',
      subtitle: namespace || cluster.split('/').pop(),
      data: { cluster, namespace, objectName },
    })
  }, [openOrPush])

  const drillToNode = useCallback((cluster: string, node: string, nodeData?: Record<string, unknown>) => {
    openOrPush({
      type: 'node',
      title: node,
      subtitle: `Node in ${cluster.split('/').pop()}`,
      data: { cluster, node, ...nodeData },
    })
  }, [openOrPush])

  const drillToGPUNode = useCallback((cluster: string, node: string, gpuData?: Record<string, unknown>) => {
    openOrPush({
      type: 'gpu-node',
      title: node,
      subtitle: 'GPU Node',
      data: { cluster, node, ...gpuData },
    })
  }, [openOrPush])

  const drillToYAML = useCallback((
    cluster: string,
    namespace: string,
    resourceType: string,
    resourceName: string,
    resourceData?: Record<string, unknown>
  ) => {
    openOrPush({
      type: 'yaml',
      title: `${resourceType}: ${resourceName}`,
      subtitle: `YAML definition`,
      data: { cluster, namespace, resourceType, resourceName, ...resourceData },
    })
  }, [openOrPush])

  const drillToResources = useCallback(() => {
    openOrPush({
      type: 'resources',
      title: 'Resource Usage',
      subtitle: 'All clusters',
      data: {},
    })
  }, [openOrPush])

  const drillToReplicaSet = useCallback((cluster: string, namespace: string, replicaset: string, replicasetData?: Record<string, unknown>) => {
    openOrPush({
      type: 'replicaset',
      title: replicaset,
      data: { cluster, namespace, replicaset, ...replicasetData },
    })
  }, [openOrPush])

  const drillToConfigMap = useCallback((cluster: string, namespace: string, configmap: string, configmapData?: Record<string, unknown>) => {
    openOrPush({
      type: 'configmap',
      title: configmap,
      data: { cluster, namespace, configmap, ...configmapData },
    })
  }, [openOrPush])

  const drillToSecret = useCallback((cluster: string, namespace: string, secret: string, secretData?: Record<string, unknown>) => {
    openOrPush({
      type: 'secret',
      title: secret,
      data: { cluster, namespace, secret, ...secretData },
    })
  }, [openOrPush])

  const drillToServiceAccount = useCallback((cluster: string, namespace: string, serviceaccount: string, serviceaccountData?: Record<string, unknown>) => {
    openOrPush({
      type: 'serviceaccount',
      title: serviceaccount,
      data: { cluster, namespace, serviceaccount, ...serviceaccountData },
    })
  }, [openOrPush])

  const drillToPVC = useCallback((cluster: string, namespace: string, pvc: string, pvcData?: Record<string, unknown>) => {
    openOrPush({
      type: 'pvc',
      title: pvc,
      subtitle: `PVC in ${namespace}`,
      data: { cluster, namespace, pvc, ...pvcData },
    })
  }, [openOrPush])

  const drillToJob = useCallback((cluster: string, namespace: string, job: string, jobData?: Record<string, unknown>) => {
    openOrPush({
      type: 'job',
      title: job,
      subtitle: `Job in ${namespace}`,
      data: { cluster, namespace, job, ...jobData },
    })
  }, [openOrPush])

  const drillToHPA = useCallback((cluster: string, namespace: string, hpa: string, hpaData?: Record<string, unknown>) => {
    openOrPush({
      type: 'hpa',
      title: hpa,
      subtitle: `HPA in ${namespace}`,
      data: { cluster, namespace, hpa, ...hpaData },
    })
  }, [openOrPush])

  const drillToService = useCallback((cluster: string, namespace: string, service: string, serviceData?: Record<string, unknown>) => {
    openOrPush({
      type: 'service',
      title: service,
      subtitle: `Service in ${namespace}`,
      data: { cluster, namespace, service, ...serviceData },
    })
  }, [openOrPush])

  return {
    drillToCluster,
    drillToNamespace,
    drillToDeployment,
    drillToReplicaSet,
    drillToPod,
    drillToLogs,
    drillToEvents,
    drillToNode,
    drillToGPUNode,
    drillToYAML,
    drillToResources,
    drillToConfigMap,
    drillToSecret,
    drillToServiceAccount,
    drillToPVC,
    drillToJob,
    drillToHPA,
    drillToService,
  }
}
