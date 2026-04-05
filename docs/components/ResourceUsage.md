# ResourceUsage

## Purpose
Displays aggregated resource usage (CPU, memory, and accelerators like GPU/TPU/AIU/XPU) across clusters, enabling users to monitor utilization and quickly identify capacity saturation issues.

Unlike list-based operational cards, `ResourceUsage` provides high-level observability via visual `Gauge` charts.

---

## Data Sources

- **Hooks**
  - `useChartFilters` (Handles cluster isolation filtering specific for visual components outside the standard list-paginations).
  - `useClusters` (Supplies core cluster topology including CPU counts, Memory capacities, and requests).
  - `useCachedGPUNodes` (Polls extensive accelerator allocations including alternative architectures like TPU/AIU/XPU).
  - `useCardLoadingState` (Translates raw fetch hooks appropriately to map empty/error/demo conditions into `CardWrapper` contexts).
  - `useDrillDownActions` (enables `drillToResources` navigation on click).

- **Backend**
  - Uses direct Kubernetes node capacities combined with requested workload utilization (Metrics aggregated generally without relying on Prometheus explicitly to ensure core capacity mapping is highly available).

- **Provider**
  - Inherits multi-provider support rendering mock values conditionally via `isDemoFallback` checks, but falls back gracefully upon errors. 

---

## Configuration & Filtering

Instead of mapping strictly to `limit` or `sortBy` grid parameters via standard dynamic card definitions, this component heavily prioritizes aggregated mathematical layout configurations natively:

| Filter Feature | Interaction Origin | Storage Key / Prop | Description |
|------|------|--------|------------|
| `selectedClusters` | `CardClusterFilter` dropdown | `resource-usage` | Internal state mapping clusters globally or locally limiting mathematical footprint boundaries |
| `acceleratorType` | Dynamic Auto-detection | None | Instantiates additional Gauge displays natively if `TPU`, `AIU`, or `XPU` node footprints contain capacities `> 0`. 

> Note: Props like `limit` and `sortBy` listed in generic architecture templates are generally dismissed in `ResourceUsage.tsx` because it evaluates full capacities sequentially via visual Gauges rather than lists.

---

## UI States

- **Loading**
  - Renders a bespoke `min-h-[200px]` loader showcasing structural skeletons for 2 mock circular `Gauge` outlines and bottom metrics layout, contrasting from typical `CardSkeleton` lists to avoid jarring visual resizes (`layout shift`).

- **Empty**
  - Displays localized text:
    > "No clusters configured" along with a prompt hint (e.g., "Add environments to track capacity").

- **Error**
  - Reverts generically to empty components or inherits the `CardWrapper` offline warnings.

---

## Logic & Patterns

- **Capacity & Saturation Math:**
  - Standardizes raw core counts and `GB` conversions automatically iterating over all fetched clusters. E.g., `cpuPercent = Math.round((usedCPUs / totalCPUs) * 100)`.
- **Dynamic Render Architecture (`Flex` / `Grid`):**
  - Evaluates how many unique families of computational acceleration are accessible in the infrastructure layer. The footer scales `gridTemplateColumns: repeat(footerCols, minmax(0, 1fr))` to render metrics columns evenly for CPU + RAM + dynamically found GPU/TPU variants.
- **Charts and Color Thresholds:**
  - Leverages standard `Gauge` outputs.
  - Hardcodes implicit alerts: CPU warning loops at `70%`, critical at `90%`. Memory thresholds sit slightly higher (`75%`, `90%`). Accelerators carry extremely strict `critical` breakpoints up to `95%`.

---

## Notes

- Critical for observability and capacity planning. Its bespoke internal layouts reflect complex metric consolidations spanning global data. 
- Overcommitted cluster detection automatically embeds error texts under `Gauge` displays if mathematical requests (`accel.data.used`) surpass tangible hardware nodes (`accel.data.total`).
- `ResourceUsage` completely centralizes unified capacities vs. allocated footprint without navigating directly to Grafana dashboard frames.
