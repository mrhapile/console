# GitOpsDrift

## Purpose
Displays the drift between the desired state (defined in Git repositories) and the actual state of resources in the clusters.

This card is central to maintaining GitOps consistency, utilizing both high-level aggregated severity states and clear granular itemized configurations to identify out-of-sync resources actively.

---

## Data Sources

- **Hooks**
  - `useCachedGitOpsDrifts` (Core hook supplying aggregated drift states mapping resource, namespace, kind, severity, driftType, and gitVersion).
  - `useGlobalFilters` (Ingests dashboard-level `selectedSeverities` constraints and `customFilter` global text searches).
  - `useCardData` (Provides local card-level generic filtering, sorting, and pagination logic wrapping the drifted items).
  - `useCardLoadingState` (Translates the raw data readiness into uniform loading skeletons/empty/offline UI contexts).

- **Backend**
  - GitOps controllers (e.g., Argo CD, Flux).
  - Returns generalized structures detailing out-of-sync status and specific sync discrepancies (`driftType`), ensuring users can identify what specific properties are deviating from source control.

- **Provider**
  - Actively supports Local Demo fallback (`isDemoData`) for dashboard evaluation periods before establishing integrations natively onto backend controllers.

---

## Configuration & Filters

| Target Feature | Interaction Origin | Storage Key / Prop | Description |
|------|------|--------|------------|
| `cluster` & `namespace` | Direct Prop | None | Card instantiated optionally pre-constrained (e.g. `<GitOpsDrift config={{cluster: 'prod'}} />`). |
| `selectedSeverities` | Dashboard Global | None | Limits list elements based on dashboard-level toggles mapping `high`, `medium`, `low` drifts matching global values (`critical`, `info`, etc). |
| `limit` | Card Controls | `5` | Pagination bounds controlled via `useCardData`. |
| `sortBy` | Card Controls | `severity` | Sorts drifts visually relying on custom comparators. (Options: `severity`, `type`, `resource`, `cluster`). |

---

## UI States

- **Loading**
  - Employs a basic animated spin loop (`Loader2`) rendering in the center when drifts are explicitly polled. 

- **Empty / Synced State**
  - Evaluates when arrays return `length === 0`:
    > A success banner containing a green `GitBranch` icon reads: "No drift detected" or "All resources are in sync".
  - Validates `drifts.length > 0` conditionally during strict global filtering constraints declaring "All filtered" rather than completely "in sync".

- **Error**
  - Degrades into a generic fallback text structure conveying directly the returned query errors strings without wrapper chrome.

---

## Logic & Patterns

- **Drift Comparisons & Visual Indicators:**
  Classifies differences into simple semantic mapping constants:
  - `modified`: Yields a Yellow color scheme with a `RefreshCw` icon ("Modified").
  - `deleted`: Yields a Red color scheme with a `Minus` icon ("Missing in cluster").
  - `added`: Yields a Blue color scheme with a `Plus` icon ("Not in Git").

- **Severity Ordering:**
  Custom sorting forces `high` → `medium` → `low` explicitly via dictionary logic mapping prior to component rendering. `High` severity counts aggressively attach `AlertTriangle` `StatusBadge` notifications directly into the inner header of the card.

- **Interactive Drill-Down (`DriftItem`):**
  Instead of utilizing the dashboard-wide `useDrillDownActions` linking to different routes, clicking drifts mounts the `GitOpsDriftDetailModal` component. This encapsulates deeper modal comparisons preventing users from rapidly context-switching out of the dashboard.

---

## Notes

- Relies heavily on external GitOps controllers parsing definitions reliably.
- Accurate validation enforces rigorous local string inclusions evaluating `resource`, `kind`, `cluster`, `namespace` text structures across both global (`customFilter`) and localized queries natively. 
- Heavily focuses upon visual severity bands (`border-l-red-500`, etc) ensuring rapid recognition of failing sync states.
