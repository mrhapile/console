import { useState, useMemo } from 'react'
import { TimeSeriesChart } from '../charts'
import { useClusters } from '../../hooks/useMCP'
import { useGlobalFilters } from '../../hooks/useGlobalFilters'
import { Server } from 'lucide-react'

// Generate demo time series data with a seed for consistency per cluster
function generateTimeSeriesData(points: number, baseValue: number, variance: number, seed: number) {
  const now = new Date()
  // Simple seeded random for consistency
  const seededRandom = (i: number) => {
    const x = Math.sin(seed + i) * 10000
    return x - Math.floor(x)
  }
  return Array.from({ length: points }, (_, i) => {
    const time = new Date(now.getTime() - (points - i - 1) * 60000)
    return {
      time: time.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }),
      value: Math.max(0, baseValue + (seededRandom(i) - 0.5) * variance),
    }
  })
}

type MetricType = 'cpu' | 'memory' | 'network' | 'requests'

const metricConfig = {
  cpu: { label: 'CPU Usage', color: '#9333ea', unit: '%', baseValue: 65, variance: 30 },
  memory: { label: 'Memory Usage', color: '#3b82f6', unit: '%', baseValue: 72, variance: 20 },
  network: { label: 'Network I/O', color: '#10b981', unit: ' MB/s', baseValue: 150, variance: 100 },
  requests: { label: 'Requests/min', color: '#f59e0b', unit: '', baseValue: 1200, variance: 500 },
}

// Generate a numeric hash from a string for seeding
function stringToSeed(str: string): number {
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = ((hash << 5) - hash) + str.charCodeAt(i)
    hash |= 0
  }
  return Math.abs(hash)
}

export function ClusterMetrics() {
  const { clusters: rawClusters } = useClusters()
  const { selectedClusters, isAllClustersSelected } = useGlobalFilters()
  const [selectedMetric, setSelectedMetric] = useState<MetricType>('cpu')

  // Filter clusters based on global selection
  const clusters = useMemo(() => {
    if (isAllClustersSelected) return rawClusters
    return rawClusters.filter(c => selectedClusters.includes(c.name))
  }, [rawClusters, selectedClusters, isAllClustersSelected])

  // Generate aggregated metrics data based on filtered clusters
  const data = useMemo(() => {
    const config = metricConfig[selectedMetric]
    const points = 20

    if (clusters.length === 0) {
      // No clusters selected - return empty data
      return generateTimeSeriesData(points, 0, 0, 0)
    }

    // Generate data for each cluster and aggregate (average)
    const clusterData = clusters.map(cluster =>
      generateTimeSeriesData(points, config.baseValue, config.variance, stringToSeed(cluster.name + selectedMetric))
    )

    // Aggregate by averaging all cluster values at each time point
    return Array.from({ length: points }, (_, i) => {
      const avgValue = clusterData.reduce((sum, cd) => sum + cd[i].value, 0) / clusterData.length
      return {
        time: clusterData[0][i].time,
        value: avgValue,
      }
    })
  }, [clusters, selectedMetric])

  const config = metricConfig[selectedMetric]
  const currentValue = data[data.length - 1]?.value || 0

  return (
    <div className="h-full flex flex-col">
      {/* Header with metric selector */}
      <div className="flex items-center justify-between mb-4">
        <div>
          <div className="flex items-center gap-2">
            <h4 className="text-sm font-medium text-foreground">{config.label}</h4>
            {!isAllClustersSelected && clusters.length > 0 && (
              <span className="flex items-center gap-1 text-xs text-muted-foreground bg-secondary/50 px-1.5 py-0.5 rounded">
                <Server className="w-3 h-3" />
                {clusters.length}
              </span>
            )}
          </div>
          <p className="text-2xl font-bold text-foreground">
            {Math.round(currentValue)}<span className="text-sm text-muted-foreground">{config.unit}</span>
          </p>
        </div>
        <div className="flex gap-1">
          {(Object.keys(metricConfig) as MetricType[]).map((key) => (
            <button
              key={key}
              onClick={() => setSelectedMetric(key)}
              className={`px-2 py-1 text-xs rounded transition-colors ${
                selectedMetric === key
                  ? 'bg-purple-500/20 text-purple-400'
                  : 'text-muted-foreground hover:text-foreground hover:bg-secondary/50'
              }`}
            >
              {metricConfig[key].label.split(' ')[0]}
            </button>
          ))}
        </div>
      </div>

      {/* Chart */}
      <div className="flex-1 min-h-0">
        {clusters.length === 0 ? (
          <div className="h-full flex items-center justify-center text-muted-foreground text-sm">
            No clusters selected
          </div>
        ) : (
          <TimeSeriesChart
            data={data}
            color={config.color}
            height={160}
            unit={config.unit}
            showGrid
          />
        )}
      </div>

      {/* Stats */}
      <div className="mt-3 pt-3 border-t border-border/50 grid grid-cols-3 gap-4">
        <div>
          <p className="text-xs text-muted-foreground">Min</p>
          <p className="text-sm font-medium text-foreground">
            {data.length > 0 ? Math.round(Math.min(...data.map((d) => d.value))) : 0}{config.unit}
          </p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Avg</p>
          <p className="text-sm font-medium text-foreground">
            {data.length > 0 ? Math.round(data.reduce((a, b) => a + b.value, 0) / data.length) : 0}{config.unit}
          </p>
        </div>
        <div>
          <p className="text-xs text-muted-foreground">Max</p>
          <p className="text-sm font-medium text-foreground">
            {data.length > 0 ? Math.round(Math.max(...data.map((d) => d.value))) : 0}{config.unit}
          </p>
        </div>
      </div>
    </div>
  )
}
