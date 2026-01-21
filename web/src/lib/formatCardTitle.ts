// Known acronyms that should stay uppercase
const ACRONYMS = new Set([
  'opa',
  'gpu',
  'pvc',
  'pv',
  'crd',
  'api',
  'cpu',
  'ram',
  'ssd',
  'hdd',
  'rbac',
  'iam',
  'dns',
  'url',
  'uri',
  'http',
  'https',
  'tcp',
  'udp',
  'ip',
  'vpc',
  'eks',
  'aks',
  'gke',
  'olm',
  'lcp',
  'argocd',
])

/**
 * Formats a card_type string into a proper title
 * Handles acronyms properly (e.g., "opa_policies" -> "OPA Policies")
 */
export function formatCardTitle(cardType: string): string {
  return cardType
    .replace(/_/g, ' ')
    .split(' ')
    .map(word => {
      const lower = word.toLowerCase()
      if (ACRONYMS.has(lower)) {
        // Special case for ArgoCD
        if (lower === 'argocd') return 'ArgoCD'
        return word.toUpperCase()
      }
      // Capitalize first letter
      return word.charAt(0).toUpperCase() + word.slice(1).toLowerCase()
    })
    .join(' ')
}
