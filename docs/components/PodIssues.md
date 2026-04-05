# PodIssues

## Purpose
Displays problematic pods across clusters to help users quickly identify workload failures, crash loops, and unhealthy states.

This card is primarily used for debugging and operational visibility, enabling operators to troubleshoot active cluster issues at a glance.

---

## Data Sources

- **Hooks**
  - `useCardData` (for filtering, sorting, pagination logic wrapper)
  - `useCachedPodIssues` (provides raw tracking, cache resolution, and continuous data polling for Pod problems)
  - `useCardLoadingState` (handles mapping raw fetch statuses into `CardWrapper` visual states)
  - `useDrillDownActions` (enables `drillToPod` exploration when elements are clicked)

- **Backend**
  - Kubernetes pod data aggregated across clusters.
  - Enriched pod lifecycle status (e.g., restart counts, container failure reasons, error statuses like `CrashLoopBackOff` or `OOMKilled`).

- **Provider**
  - Tracks live endpoints, local demo fallback modes (`isDemoFallback`), and limited-access warning states when backend synchronization experiences delays or RBAC permission drops.

---

## Configuration (Props)

| Prop | Type | Default | Description |
|------|------|--------|------------|
| `cluster` | `string` | `all` | Filter pods by individual cluster |
| `namespace` | `string` | `all` | Filter by Kubernetes namespace |
| `limit` | `number` | `5` | Number of items to display per page (configurable dynamically via `CardControlsRow`) |
| `sortBy` | `string` | `status` | Sorting field (options: `status`, `name`, `restarts`, `cluster`) |
| `order` | `"asc" \| "desc"` | `asc` | Sorting order (`asc` applies by default, pushing severe elements conditionally) |

> Note: Exact programmatic props map directly into `CardControlsRow` and `CardSearchInput`, adapting dynamically dependent upon `DynamicCard` configurations and Dashboard UI overrides.

---

## UI States

- **Loading**
  - Uses `CardSkeleton` with `rows={3}` and `rowHeight={80}` showing the header layout prior to mounting content.

- **Empty**
  - Uses a strict empty status displaying:
    > "All pods healthy - No issues detected" utilizing a green `CheckCircle` variant icon state.
  - Or "No pod issues" empty state message based conditionally via the internal loading context.

- **Error / Offline Data**
  - Renders `LimitedAccessWarning` appending to the bottom to verify cache stale thresholds.

---

## Logic & Patterns

- **Issue Triggers and Filtering:**
  Identifies problematic workloads based selectively via `useCachedPodIssues` parsing state:
  - Memory usage violations (`OOMKilled`) mapping to `MemoryStick` icons.
  - Registry failures (`ImagePullBackOff`) mapping to `ImageOff` icons.
  - Scheduling delays (`Pending`) mapping to `Clock` icons.
  - Generic loops/restarts (`CrashLoopBackOff`) rendering dynamically calculated thresholds using `RefreshCw` icons.

- **Sorting and Search (`useCardData` Integration):**
  - Searchable fields include: `name`, `namespace`, `cluster`, `status`. Needs string containment matches through an additional custom predicate scanning `issue.issues`.
  - Supports strict comparators ensuring restarts dynamically sort workloads numerically without custom inline comparisons, while names/clusters remain localized string sorted.

- **AI Diagnostic Tools:**
  - Inlines `CardAIActions`. Every troubled pod renders quick actions allowing administrators to trigger "AI Diagnose, Repair & Ask actions" directly tied to Kubernetes resource parameters (`name`, `namespace`, `cluster`, `status`, and `restarts` context payload).

- **Prioritization Workflow:**
  - Evaluates rendering background hues using a shared layout protocol via `getStatusColors` parsing raw backend container errors to `colors.bg` and `colors.text`.

---

## Notes

- This is one of the most frequently used debugging cards in the dashboard.
- Maintains strict consistency utilizing generic mapping architectures (like `ClusterBadge`, `StatusBadge`, and unified sorting controls) to match `ClusterHealth` patterns. Ensure `ClusterBadge` implementations remain strictly unified across these core cards.
- The interactive element utilizes `drillToPod`, enforcing deep navigability beyond just the shallow metrics.
