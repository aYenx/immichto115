import { ref, onUnmounted } from 'vue'

export interface LogLine {
  stream: 'stdout' | 'stderr'
  text: string
}

export function useWebSocket() {
  const logs = ref<LogLine[]>([])
  const connected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null

  function connect() {
    if (ws && ws.readyState <= WebSocket.OPEN) return

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/ws/logs`

    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
      console.log('[ws] connected')
    }

    ws.onmessage = (event) => {
      try {
        const line: LogLine = JSON.parse(event.data)
        logs.value.push(line)
        // 保留最近 2000 行
        if (logs.value.length > 2000) {
          logs.value = logs.value.slice(-1500)
        }
      } catch {
        // 忽略非 JSON 消息
      }
    }

    ws.onclose = () => {
      connected.value = false
      console.log('[ws] disconnected, reconnecting in 3s...')
      scheduleReconnect()
    }

    ws.onerror = () => {
      ws?.close()
    }
  }

  function scheduleReconnect() {
    if (reconnectTimer) return
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connect()
    }, 3000)
  }

  function disconnect() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
    if (ws) {
      ws.onclose = null // 阻止自动重连
      ws.close()
      ws = null
    }
    connected.value = false
  }

  function clearLogs() {
    logs.value = []
  }

  onUnmounted(() => {
    disconnect()
  })

  return { logs, connected, connect, disconnect, clearLogs }
}
