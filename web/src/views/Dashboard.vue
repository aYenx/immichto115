<template>
  <div class="dashboard-container">
    <div class="header">
      <div class="greeting">
        <h1>{{ greeting }}，Administrator</h1>
        <p>系统环境良好。当前状态：{{ systemStatus?.backup_status === 'running' ? '备份中...' : '空闲' }}</p>
      </div>
      <div class="actions">
        <button class="btn secondary" @click="openSettings">
          编辑配置
        </button>
        <button class="btn secondary" @click="stopBackup" :disabled="systemStatus?.backup_status !== 'running'">
          <LucidePause :size="16" />
          停止备份
        </button>
        <button class="btn primary" @click="startBackup" :disabled="systemStatus?.backup_status === 'running'">
          <LucidePlay :size="16" />
          立即备份
        </button>
      </div>
    </div>

    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon-wrapper blue">
          <LucideClock :size="20" class="icon" />
        </div>
        <div class="stat-info">
          <span class="stat-label">定时开启状态</span>
          <span class="stat-value" style="font-size: 16px;">{{ systemStatus?.cron_enabled ? '已开启' : '未开启' }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-wrapper green">
          <LucideImage :size="20" class="icon" />
        </div>
        <div class="stat-info">
          <span class="stat-label">下次备份时间</span>
          <span class="stat-value" style="font-size: 16px;">{{ formatNextRun(systemStatus?.next_run) }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-wrapper yellow">
          <LucideDatabase :size="20" class="icon" />
        </div>
        <div class="stat-info">
          <span class="stat-label">配置已完成</span>
          <span class="stat-value" style="font-size: 16px;">{{ systemStatus?.setup_complete ? '是' : '否' }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-wrapper purple">
          <LucideCloud :size="20" class="icon" />
        </div>
        <div class="stat-info">
          <span class="stat-label">Rclone 状态</span>
          <span class="stat-value" style="font-size: 16px;">{{ systemStatus?.rclone_installed ? '可用' : '未检测到' }}</span>
        </div>
      </div>
    </div>

    <!-- Logs Section -->
    <div class="logs-section">
      <div class="logs-header">
        <h2>实时日志监控</h2>
        <div class="logs-actions">
          <button class="icon-btn" @click="logs = []">
            <LucideRefreshCcw :size="16" />
          </button>
        </div>
      </div>
      
      <div class="logs-terminal" ref="terminalRef">
        <div v-for="(log, idx) in logs" :key="idx" :class="['log-line', getLogLevelClass(log.text)]">
          <span class="timestamp">[{{ log.time }}]</span>
          <span class="message">{{ log.text }}</span>
        </div>
        <!-- Active log cursor -->
        <div class="log-line active">
          <span class="cursor">_</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { 
  LucidePlay, 
  LucidePause, 
  LucideClock, 
  LucideImage, 
  LucideDatabase, 
  LucideCloud,
  LucideRefreshCcw
} from 'lucide-vue-next'
import { api, getErrorMessage, handleAuthFailure, type SystemStatus } from '../api'

const systemStatus = ref<SystemStatus | null>(null)
const logs = ref<Array<{ time: string, text: string }>>([])
const terminalRef = ref<HTMLElement | null>(null)
const router = useRouter()

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '夜深了'
  if (hour < 12) return '早上好'
  if (hour < 14) return '中午好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

let ws: WebSocket | null = null
let statusInterval: ReturnType<typeof setInterval> | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let shouldReconnect = true

const disconnectRealtime = () => {
  shouldReconnect = false

  if (statusInterval) {
    clearInterval(statusInterval)
    statusInterval = null
  }

  if (reconnectTimer) {
    clearTimeout(reconnectTimer)
    reconnectTimer = null
  }

  if (ws) {
    ws.close()
    ws = null
  }
}

const fetchStatus = async () => {
  try {
    systemStatus.value = await api.getSystemStatus()
  } catch (err) {
    if (handleAuthFailure(err)) {
      disconnectRealtime()
      return
    }
    console.error('Failed to get status', err)
  }
}

const formatNextRun = (dateStr: string | null | undefined) => {
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return 'N/A'
  const d = new Date(dateStr)
  return d.toLocaleString()
}

const openSettings = () => {
  router.push('/settings')
}

const startBackup = async () => {
  try {
    await api.startBackup()
    await fetchStatus()
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    alert(getErrorMessage(err))
  }
}

const stopBackup = async () => {
  try {
    await api.stopBackup()
    await fetchStatus()
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    alert(getErrorMessage(err))
  }
}

const getLogLevelClass = (text: string) => {
  const upper = text.toUpperCase()
  if (upper.includes('ERROR') || upper.includes('FAILED')) return 'error'
  if (upper.includes('WARN')) return 'warning'
  if (upper.includes('SUCCESS')) return 'success'
  return 'info'
}

const connectWebSocket = () => {
  if (!shouldReconnect) return
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws/logs`
  
  ws = new WebSocket(wsUrl)
  ws.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data)
      const now = new Date()
      logs.value.push({
        time: now.toLocaleTimeString(),
        text: data.Text || data.text || ''
      })
      if (logs.value.length > 500) {
        logs.value.shift()
      }
      nextTick(() => {
        if (terminalRef.value) {
          terminalRef.value.scrollTop = terminalRef.value.scrollHeight
        }
      })
    } catch {
      // Ignore parse errors
    }
  }
  ws.onerror = (e) => {
    console.error('WebSocket error', e)
  }
  ws.onclose = () => {
    ws = null
    if (!shouldReconnect) {
      return
    }

    console.log('WebSocket closed, reconnecting in 5s...')
    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      connectWebSocket()
    }, 5000)
  }
}

onMounted(() => {
  shouldReconnect = true
  fetchStatus()
  statusInterval = setInterval(fetchStatus, 3000)
  connectWebSocket()
})

onUnmounted(() => {
  disconnectRealtime()
})
</script>

<style scoped>
.dashboard-container {
  display: flex;
  flex-direction: column;
  padding: 48px 64px;
  gap: 40px;
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.greeting h1 {
  font-family: var(--font-primary);
  font-weight: 800;
  font-size: 32px;
  color: var(--text-primary);
  letter-spacing: -0.5px;
  margin-bottom: 8px;
}

.greeting p {
  color: var(--text-secondary);
  font-size: 16px;
}

.actions {
  display: flex;
  gap: 16px;
}

/* .btn styles inherited from global style.css */

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 24px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  background-color: var(--bg-card);
  padding: 24px;
  border-radius: 16px;
  border: 1px solid var(--border-subtle);
}

.stat-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
}

.stat-icon-wrapper.blue {
  background-color: rgba(59, 130, 246, 0.1);
  color: #3B82F6;
}

.stat-icon-wrapper.green {
  background-color: rgba(16, 185, 129, 0.1);
  color: #10B981;
}

.stat-icon-wrapper.yellow {
  background-color: rgba(245, 158, 11, 0.1);
  color: #F59E0B;
}

.stat-icon-wrapper.purple {
  background-color: rgba(139, 92, 246, 0.1);
  color: #8B5CF6;
}

.stat-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.stat-label {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
}

.stat-value {
  color: var(--text-primary);
  font-size: 20px;
  font-weight: 800;
  font-family: var(--font-primary);
}

.logs-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
  flex: 1;
}

.logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logs-header h2 {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
}

.logs-actions {
  display: flex;
  gap: 8px;
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 6px;
  color: var(--text-secondary);
  background-color: transparent;
  transition: all 0.2s ease;
}

.icon-btn:hover {
  background-color: var(--border-subtle);
  color: var(--text-primary);
}

.logs-terminal {
  flex: 1;
  min-height: 400px;
  background-color: #0F172A; /* Dark theme for terminal */
  border-radius: 12px;
  padding: 24px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border: 1px solid var(--border-strong);
}

.log-line {
  display: flex;
  gap: 12px;
  line-height: 1.5;
}

.timestamp {
  color: #64748B;
}

.level {
  font-weight: 600;
  min-width: 60px;
}

.info .level {
  color: #3B82F6;
}

.success .level {
  color: #10B981;
}

.warning .level {
  color: #F59E0B;
}

.error .level {
  color: #EF4444;
}

.info .message {
  color: #E2E8F0;
}

.success .message {
  color: #10B981;
}

.warning .message {
  color: #FCD34D;
}

.error .message {
  color: #FCA5A5;
}

.cursor {
  color: #E2E8F0;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}
</style>
