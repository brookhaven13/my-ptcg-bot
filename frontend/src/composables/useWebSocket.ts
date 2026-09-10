import { ref, onUnmounted } from 'vue'

export interface WSMessage {
  type: string
  payload?: any
}

export function useWebSocket() {
  const connected = ref(false)
  const lastMessage = ref<WSMessage | null>(null)
  let ws: WebSocket | null = null
  let handlers: ((msg: WSMessage) => void)[] = []

  function connect(gameId: string) {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${window.location.host}/api/battle/ws/${gameId}`

    ws = new WebSocket(url)

    ws.onopen = () => {
      connected.value = true
    }

    ws.onclose = () => {
      connected.value = false
    }

    ws.onerror = (err) => {
      console.error('[ws] error:', err)
    }

    ws.onmessage = (event) => {
      try {
        const msg: WSMessage = JSON.parse(event.data)
        lastMessage.value = msg
        handlers.forEach(h => h(msg))
      } catch (e) {
        console.error('[ws] parse error:', e)
      }
    }
  }

  function send(msg: Record<string, unknown>) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(msg))
    }
  }

  function onMessage(handler: (msg: WSMessage) => void) {
    handlers.push(handler)
  }

  function disconnect() {
    if (ws) {
      ws.close()
      ws = null
    }
    handlers = []
    connected.value = false
  }

  onUnmounted(() => {
    disconnect()
  })

  return {
    connected,
    lastMessage,
    connect,
    send,
    onMessage,
    disconnect,
  }
}
