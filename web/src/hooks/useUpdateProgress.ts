import { useEffect, useState, useRef } from 'react'
import type { UpdateProgress } from '../types/updates'
import { LOCAL_AGENT_WS_URL } from '../lib/constants/network'

const WS_RECONNECT_MS = 5000 // Reconnect interval after WebSocket disconnect

/**
 * Hook that listens for update_progress WebSocket broadcasts from kc-agent.
 * Uses a separate WebSocket connection to avoid interfering with the shared one.
 */
export function useUpdateProgress() {
  const [progress, setProgress] = useState<UpdateProgress | null>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const progressRef = useRef<UpdateProgress | null>(null)

  // Keep ref in sync so the connect closure always sees the latest value
  progressRef.current = progress

  useEffect(() => {
    let reconnectTimer: ReturnType<typeof setTimeout>

    function connect() {
      try {
        const ws = new WebSocket(LOCAL_AGENT_WS_URL)
        wsRef.current = ws

        ws.onopen = () => {
          // If we reconnected while showing "restarting", the restart succeeded
          const cur = progressRef.current
          if (cur && cur.status === 'restarting') {
            setProgress({ status: 'done', message: 'Update complete — restarted successfully', progress: 100 })
          }
        }

        ws.onmessage = (event) => {
          try {
            const msg = JSON.parse(event.data)
            if (msg.type === 'update_progress' && msg.payload) {
              setProgress(msg.payload as UpdateProgress)
            }
          } catch {
            // Ignore parse errors
          }
        }

        ws.onclose = () => {
          wsRef.current = null
          // Reconnect after 5 seconds (faster during restarts)
          reconnectTimer = setTimeout(connect, WS_RECONNECT_MS)
        }

        ws.onerror = () => {
          ws.close()
        }
      } catch {
        // Agent not available, retry later
        reconnectTimer = setTimeout(connect, WS_RECONNECT_MS)
      }
    }

    connect()

    return () => {
      clearTimeout(reconnectTimer)
      if (wsRef.current) {
        wsRef.current.close()
        wsRef.current = null
      }
    }
  }, [])

  const dismiss = () => setProgress(null)

  return { progress, dismiss }
}
