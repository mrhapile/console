import { useState, useCallback } from 'react'
import { Shield, Check, X, Loader2, AlertCircle } from 'lucide-react'
import { useCanI } from '../../hooks/usePermissions'
import { useClusters } from '../../hooks/useMCP'

const COMMON_VERBS = ['get', 'list', 'create', 'update', 'delete', 'watch', 'patch']

// Resource to API group mapping - required for correct permission checks
const RESOURCE_API_GROUPS: Record<string, string> = {
  // Core API (empty string)
  pods: '',
  services: '',
  secrets: '',
  configmaps: '',
  namespaces: '',
  nodes: '',
  persistentvolumeclaims: '',
  serviceaccounts: '',
  events: '',
  endpoints: '',
  // apps API group
  deployments: 'apps',
  replicasets: 'apps',
  statefulsets: 'apps',
  daemonsets: 'apps',
  // rbac.authorization.k8s.io
  roles: 'rbac.authorization.k8s.io',
  rolebindings: 'rbac.authorization.k8s.io',
  clusterroles: 'rbac.authorization.k8s.io',
  clusterrolebindings: 'rbac.authorization.k8s.io',
  // batch
  jobs: 'batch',
  cronjobs: 'batch',
  // networking.k8s.io
  ingresses: 'networking.k8s.io',
  networkpolicies: 'networking.k8s.io',
  // autoscaling
  horizontalpodautoscalers: 'autoscaling',
  // storage.k8s.io
  storageclasses: 'storage.k8s.io',
}

const COMMON_RESOURCES = [
  'pods',
  'deployments',
  'services',
  'secrets',
  'configmaps',
  'namespaces',
  'nodes',
  'persistentvolumeclaims',
  'serviceaccounts',
  'roles',
  'rolebindings',
  'clusterroles',
  'clusterrolebindings',
  'jobs',
  'cronjobs',
  'ingresses',
  'statefulsets',
  'daemonsets',
]

export function CanIChecker() {
  const { clusters: rawClusters } = useClusters()
  const clusters = rawClusters.map(c => c.name)
  const { checkPermission, checking, result, error, reset } = useCanI()

  const [cluster, setCluster] = useState('')
  const [verb, setVerb] = useState('get')
  const [resource, setResource] = useState('pods')
  const [namespace, setNamespace] = useState('')
  const [customVerb, setCustomVerb] = useState('')
  const [customResource, setCustomResource] = useState('')
  const [group, setGroup] = useState('')
  const [showAdvanced, setShowAdvanced] = useState(false)

  const handleCheck = useCallback(async () => {
    const selectedCluster = cluster || clusters[0]
    if (!selectedCluster) return

    const selectedVerb = verb === 'custom' ? customVerb : verb
    const selectedResource = resource === 'custom' ? customResource : resource

    if (!selectedVerb || !selectedResource) return

    // Auto-detect API group from resource, or use manual override if provided
    const autoGroup = RESOURCE_API_GROUPS[selectedResource]
    const effectiveGroup = group || autoGroup

    await checkPermission({
      cluster: selectedCluster,
      verb: selectedVerb,
      resource: selectedResource,
      namespace: namespace || undefined,
      group: effectiveGroup !== undefined ? effectiveGroup : undefined,
    })
  }, [cluster, clusters, verb, customVerb, resource, customResource, namespace, group, checkPermission])

  const handleReset = useCallback(() => {
    reset()
    setVerb('get')
    setResource('pods')
    setNamespace('')
    setCustomVerb('')
    setCustomResource('')
    setGroup('')
  }, [reset])

  return (
    <div className="glass rounded-xl p-6">
      <div className="flex items-center gap-3 mb-4">
        <div className="p-2 rounded-lg bg-blue-500/20">
          <Shield className="w-5 h-5 text-blue-400" />
        </div>
        <div>
          <h2 className="text-lg font-medium text-foreground">Permission Checker</h2>
          <p className="text-sm text-muted-foreground">Check if you can perform actions on cluster resources</p>
        </div>
      </div>

      <div className="space-y-4">
        {/* Cluster Selection */}
        <div>
          <label htmlFor="cluster-select" className="block text-sm font-medium text-foreground mb-1">
            Cluster
          </label>
          <select
            id="cluster-select"
            value={cluster || clusters[0] || ''}
            onChange={(e) => setCluster(e.target.value)}
            className="w-full p-2 rounded-lg bg-secondary border border-border text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="can-i-cluster"
          >
            {clusters.map((c) => (
              <option key={c} value={c}>{c}</option>
            ))}
          </select>
        </div>

        {/* Verb Selection */}
        <div>
          <label htmlFor="verb-select" className="block text-sm font-medium text-foreground mb-1">
            Action (Verb)
          </label>
          <select
            id="verb-select"
            value={verb}
            onChange={(e) => setVerb(e.target.value)}
            className="w-full p-2 rounded-lg bg-secondary border border-border text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="can-i-verb"
          >
            {COMMON_VERBS.map((v) => (
              <option key={v} value={v}>{v}</option>
            ))}
            <option value="custom">Custom...</option>
          </select>
          {verb === 'custom' && (
            <input
              type="text"
              value={customVerb}
              onChange={(e) => setCustomVerb(e.target.value)}
              placeholder="Enter custom verb"
              className="mt-2 w-full p-2 rounded-lg bg-secondary border border-border text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
              data-testid="can-i-custom-verb"
            />
          )}
        </div>

        {/* Resource Selection */}
        <div>
          <label htmlFor="resource-select" className="block text-sm font-medium text-foreground mb-1">
            Resource
          </label>
          <select
            id="resource-select"
            value={resource}
            onChange={(e) => setResource(e.target.value)}
            className="w-full p-2 rounded-lg bg-secondary border border-border text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="can-i-resource"
          >
            {COMMON_RESOURCES.map((r) => (
              <option key={r} value={r}>{r}</option>
            ))}
            <option value="custom">Custom...</option>
          </select>
          {resource === 'custom' && (
            <input
              type="text"
              value={customResource}
              onChange={(e) => setCustomResource(e.target.value)}
              placeholder="Enter custom resource"
              className="mt-2 w-full p-2 rounded-lg bg-secondary border border-border text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
              data-testid="can-i-custom-resource"
            />
          )}
        </div>

        {/* Namespace (optional) */}
        <div>
          <label htmlFor="namespace-input" className="block text-sm font-medium text-foreground mb-1">
            Namespace <span className="text-muted-foreground">(optional, leave empty for cluster-scoped)</span>
          </label>
          <input
            id="namespace-input"
            type="text"
            value={namespace}
            onChange={(e) => setNamespace(e.target.value)}
            placeholder="e.g., default"
            className="w-full p-2 rounded-lg bg-secondary border border-border text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="can-i-namespace"
          />
        </div>

        {/* API Group - shown by default since it's required for accurate checks */}
        <div>
          <label htmlFor="group-input" className="block text-sm font-medium text-foreground mb-1">
            API Group <span className="text-muted-foreground">(auto-detected for common resources)</span>
          </label>
          <input
            id="group-input"
            type="text"
            value={group}
            onChange={(e) => setGroup(e.target.value)}
            placeholder={
              resource !== 'custom' && RESOURCE_API_GROUPS[resource] !== undefined
                ? `Auto: ${RESOURCE_API_GROUPS[resource] || '(core)'}`
                : 'e.g., apps, rbac.authorization.k8s.io'
            }
            className="w-full p-2 rounded-lg bg-secondary border border-border text-foreground focus:outline-none focus:ring-2 focus:ring-blue-500"
            data-testid="can-i-group"
          />
          {resource !== 'custom' && RESOURCE_API_GROUPS[resource] !== undefined && (
            <p className="mt-1 text-xs text-muted-foreground">
              Auto-detected: <code className="px-1 py-0.5 rounded bg-secondary">{RESOURCE_API_GROUPS[resource] || '(core API)'}</code>
            </p>
          )}
        </div>

        {/* Advanced Options */}
        <button
          type="button"
          onClick={() => setShowAdvanced(!showAdvanced)}
          className="text-sm text-muted-foreground hover:text-foreground"
        >
          {showAdvanced ? '- Hide' : '+ Show'} advanced options
        </button>

        {showAdvanced && (
          <div className="text-xs text-muted-foreground p-3 bg-secondary/30 rounded-lg">
            <p className="font-medium mb-2">Common API Groups:</p>
            <ul className="space-y-1">
              <li><code className="text-blue-400">""</code> - Core API (pods, services, secrets, configmaps, namespaces)</li>
              <li><code className="text-blue-400">apps</code> - Deployments, StatefulSets, DaemonSets, ReplicaSets</li>
              <li><code className="text-blue-400">rbac.authorization.k8s.io</code> - Roles, ClusterRoles, Bindings</li>
              <li><code className="text-blue-400">batch</code> - Jobs, CronJobs</li>
              <li><code className="text-blue-400">networking.k8s.io</code> - Ingresses, NetworkPolicies</li>
            </ul>
          </div>
        )}

        {/* Check Button */}
        <div className="flex gap-2">
          <button
            onClick={handleCheck}
            disabled={checking || clusters.length === 0}
            className="flex-1 flex items-center justify-center gap-2 px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-700 text-white font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            data-testid="can-i-check"
          >
            {checking ? (
              <>
                <Loader2 className="w-4 h-4 animate-spin" />
                Checking...
              </>
            ) : (
              <>
                <Shield className="w-4 h-4" />
                Check Permission
              </>
            )}
          </button>
          {result && (
            <button
              onClick={handleReset}
              className="px-4 py-2 rounded-lg bg-secondary hover:bg-secondary/80 text-foreground font-medium transition-colors"
              data-testid="can-i-reset"
            >
              Reset
            </button>
          )}
        </div>

        {/* Result */}
        {result && (
          <div
            className={`p-4 rounded-lg border ${
              result.allowed
                ? 'bg-green-500/10 border-green-500/30'
                : 'bg-red-500/10 border-red-500/30'
            }`}
            data-testid="can-i-result"
          >
            <div className="flex items-center gap-2">
              {result.allowed ? (
                <>
                  <Check className="w-5 h-5 text-green-500" />
                  <span className="font-medium text-green-500">Allowed</span>
                </>
              ) : (
                <>
                  <X className="w-5 h-5 text-red-500" />
                  <span className="font-medium text-red-500">Denied</span>
                </>
              )}
            </div>
            <p className="mt-2 text-sm text-muted-foreground">
              You {result.allowed ? 'can' : 'cannot'}{' '}
              <code className="px-1 py-0.5 rounded bg-secondary">{verb === 'custom' ? customVerb : verb}</code>{' '}
              <code className="px-1 py-0.5 rounded bg-secondary">{resource === 'custom' ? customResource : resource}</code>
              {namespace && (
                <>
                  {' '}in namespace <code className="px-1 py-0.5 rounded bg-secondary">{namespace}</code>
                </>
              )}
            </p>
            {result.reason && (
              <p className="mt-1 text-xs text-muted-foreground">{result.reason}</p>
            )}
          </div>
        )}

        {/* Error */}
        {error && (
          <div className="p-4 rounded-lg bg-red-500/10 border border-red-500/30" data-testid="can-i-error">
            <div className="flex items-center gap-2">
              <AlertCircle className="w-5 h-5 text-red-500" />
              <span className="font-medium text-red-500">Error</span>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">{error}</p>
          </div>
        )}

        {/* No clusters warning */}
        {clusters.length === 0 && (
          <div className="p-4 rounded-lg bg-yellow-500/10 border border-yellow-500/30">
            <div className="flex items-center gap-2">
              <AlertCircle className="w-5 h-5 text-yellow-500" />
              <span className="font-medium text-yellow-500">No clusters available</span>
            </div>
            <p className="mt-1 text-sm text-muted-foreground">
              Connect to a cluster to check permissions.
            </p>
          </div>
        )}
      </div>
    </div>
  )
}
