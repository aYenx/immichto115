<template>
  <div class="dashboard-container">
    <div class="header">
      <div class="greeting">
        <h1>{{ greeting }}，Administrator</h1>
        <p>系统环境良好。当前状态：{{ backupStatusText }}</p>
        <div class="backup-status-strip">
          <span :class="['status-chip', backupPhaseTone]">{{ backupPhaseLabel }}</span>
          <span v-if="backupStatusDetail" class="status-detail">{{ backupStatusDetail }}</span>
        </div>
      </div>
      <div v-if="!wsConnected || !apiReachable" class="connection-banner">
        <LucideWifiOff :size="16" />
        <span v-if="!apiReachable">后端连接失败，请检查服务是否运行</span>
        <span v-else>日志连接已断开，正在重连...</span>
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
        <div class="logs-meta">
          <span class="log-count">{{ logs.length }} / {{ MAX_LOGS }} 条</span>
          <button class="icon-btn" :class="{ active: autoScroll }" @click="autoScroll = !autoScroll" :title="autoScroll ? '自动滚动已开启' : '自动滚动已关闭'">
            <LucideArrowDownToLine :size="16" />
          </button>
          <button class="icon-btn" @click="clearLogs" title="清空日志">
            <LucideTrash2 :size="16" />
          </button>
        </div>
      </div>
      
      <div class="logs-terminal" ref="terminalRef" @scroll="onTerminalScroll">
        <div v-for="log in logs" :key="log.id" :class="['log-line', getLogLevelClass(log.text)]">
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
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { 
  LucidePlay, 
  LucidePause, 
  LucideClock, 
  LucideImage, 
  LucideDatabase, 
  LucideCloud,
  LucideArrowDownToLine,
  LucideTrash2,
  LucideWifiOff
} from 'lucide-vue-next'
import { api, getErrorMessage, handleAuthFailure, type SystemStatus } from '../api'
import { showToast } from '../composables/toast'

const MAX_LOGS = 200
let logIdCounter = 0

const systemStatus = ref<SystemStatus | null>(null)
const logs = ref<Array<{ id: number, time: string, text: string }>>([])
const lastStatusSnapshot = ref('')
const terminalRef = ref<HTMLElement | null>(null)
const autoScroll = ref(true)
const wsConnected = ref(false)
const apiReachable = ref(true)
let scrollRAF: number | null = null
const router = useRouter()

const greeting = computed(() => {
  const hour = new Date().getHours()
  if (hour < 6) return '夜深了'
  if (hour < 12) return '早上好'
  if (hour < 14) return '中午好'
  if (hour < 18) return '下午好'
  return '晚上好'
})

const latestLogText = computed(() => {
  for (let i = logs.value.length - 1; i >= 0; i -= 1) {
    const text = logs.value[i]!.text
    if (!text.includes('connected to log stream')) {
      return text
    }
  }
  return ''
})

const derivePhaseFromText = (text: string): 'idle' | 'preparing' | 'library' | 'database' | 'stopping' | 'success' | 'partial' | 'failed' | null => {
  const lower = text.toLowerCase()

  if (lower.includes('任务已被手动停止') || lower.includes('安全退出') || lower.includes('停止信号') || lower.includes('已发送停止指令')) return 'stopping'
  if (lower.includes('所有备份阶段执行完毕') || lower.includes('当前同步阶段已成功完成')) return 'success'
  if (lower.includes('数据库备份失败') || lower.includes('照片库备份失败') || lower.includes('生成 rclone 配置失败') || lower.includes('无法启动')) {
    const completedLibrary = logs.value.some(log => log.text.includes('照片库目录备份阶段已结束'))
    const completedDatabase = logs.value.some(log => log.text.includes('数据库备份目录同步阶段已结束'))
    if (completedLibrary || completedDatabase) return 'partial'
    return 'failed'
  }
  if (lower.includes('开始备份数据库备份目录')) return 'database'
  if (lower.includes('开始备份照片库目录')) return 'library'
  if (lower.includes('备份任务已启动') || lower.includes('正在生成临时 rclone 配置') || lower.includes('开始执行同步任务')) return 'preparing'
  return null
}

const latestStatusLogText = computed(() => {
  for (let i = logs.value.length - 1; i >= 0; i -= 1) {
    const text = logs.value[i]!.text
    if (derivePhaseFromText(text) !== null) {
      return text
    }
  }
  return lastStatusSnapshot.value
})

const backupPhase = computed<'idle' | 'preparing' | 'library' | 'database' | 'stopping' | 'success' | 'partial' | 'failed'>(() => {
  const phase = derivePhaseFromText(latestStatusLogText.value)
  if (phase) return phase
  if (systemStatus.value?.backup_status === 'running') return 'preparing'
  return 'idle'
})

const backupPhaseLabel = computed(() => {
  switch (backupPhase.value) {
    case 'preparing': return '准备中'
    case 'library': return '备份照片库中'
    case 'database': return '备份数据库中'
    case 'stopping': return '停止中'
    case 'success': return '已完成'
    case 'partial': return '部分失败'
    case 'failed': return '已失败'
    default: return '空闲'
  }
})

const backupPhaseTone = computed(() => {
  switch (backupPhase.value) {
    case 'success': return 'success'
    case 'failed':
    case 'partial': return 'error'
    case 'stopping': return 'warning'
    case 'preparing':
    case 'library':
    case 'database': return 'info'
    default: return 'neutral'
  }
})

const backupStatusText = computed(() => {
  if (backupPhase.value === 'idle') {
    return systemStatus.value?.backup_status === 'running' ? '备份中...' : '空闲'
  }
  return backupPhaseLabel.value
})

const backupStatusDetail = computed(() => {
  const sourceText = latestStatusLogText.value || latestLogText.value
  const text = sourceText.replace(/^\[immichto115\]\s*/, '').trim()
  if (text) {
    if (backupPhase.value === 'idle') return ''
    return text
  }

  if (backupPhase.value === 'preparing') {
    return '任务正在运行，等待实时日志同步...'
  }

  return ''
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
    apiReachable.value = true
  } catch (err) {
    if (handleAuthFailure(err)) {
      disconnectRealtime()
      return
    }
    apiReachable.value = false
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
  const previousSnapshot = lastStatusSnapshot.value
  const previousLogs = [...logs.value]
  try {
    lastStatusSnapshot.value = '[immichto115] 备份任务已启动，正在检查配置与目标路径...'
    logs.value = []
    const result = await api.startBackup()
    await fetchStatus()
    showToast('info', '备份已启动', result.message || '正在检查配置并准备同步任务，请留意实时日志。')
  } catch (err: any) {
    lastStatusSnapshot.value = previousSnapshot
    logs.value = previousLogs
    if (handleAuthFailure(err)) return
    showToast('error', '启动备份失败', getErrorMessage(err))
  }
}

const stopBackup = async () => {
  const previousSnapshot = lastStatusSnapshot.value
  const previousLogs = [...logs.value]
  try {
    lastStatusSnapshot.value = '[immichto115] 已发送停止指令，当前任务会在安全收尾后退出'
    logs.value = []
    const result = await api.stopBackup()
    await fetchStatus()
    showToast('warning', '已请求停止备份', result.message || '当前任务会在安全收尾后退出。')
  } catch (err: any) {
    lastStatusSnapshot.value = previousSnapshot
    logs.value = previousLogs
    if (handleAuthFailure(err)) return
    showToast('error', '停止备份失败', getErrorMessage(err))
  }
}

const getLogLevelClass = (text: string) => {
  const upper = text.toUpperCase()
  if (
    upper.includes('ERROR') ||
    upper.includes('FAILED') ||
    text.includes('失败') ||
    text.includes('异常退出') ||
    text.includes('无法启动')
  ) return 'error'

  if (
    upper.includes('WARN') ||
    text.includes('停止') ||
    text.includes('跳过') ||
    text.includes('请先')
  ) return 'warning'

  if (
    upper.includes('SUCCESS') ||
    text.includes('成功') ||
    text.includes('已完成') ||
    text.includes('执行完毕')
  ) return 'success'

  return 'info'
}

const clearLogs = () => {
  lastStatusSnapshot.value = latestStatusLogText.value || lastStatusSnapshot.value
  logs.value = []
}

const scrollToBottom = () => {
  if (scrollRAF) return
  scrollRAF = requestAnimationFrame(() => {
    scrollRAF = null
    if (autoScroll.value && terminalRef.value) {
      terminalRef.value.scrollTop = terminalRef.value.scrollHeight
    }
  })
}

const onTerminalScroll = () => {
  if (!terminalRef.value) return
  const el = terminalRef.value
  // If user scrolled up more than 50px from bottom, pause auto-scroll
  const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 50
  autoScroll.value = atBottom
}

const connectWebSocket = () => {
  if (!shouldReconnect) return
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) {
    return
  }

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws/logs`
  
  ws = new WebSocket(wsUrl)
  ws.onopen = () => {
    wsConnected.value = true
  }
  ws.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data)
      const now = new Date()
      const text = data.Text || data.text || ''
      logs.value.push({
        id: ++logIdCounter,
        time: now.toLocaleTimeString(),
        text
      })
      if (derivePhaseFromText(text) !== null) {
        lastStatusSnapshot.value = text
      }
      // Batch trim: remove 50 oldest when exceeding limit
      if (logs.value.length > MAX_LOGS) {
        logs.value.splice(0, logs.value.length - MAX_LOGS)
      }
      scrollToBottom()
    } catch {
      // Ignore parse errors
    }
  }
  ws.onerror = (e) => {
    console.error('WebSocket error', e)
  }
  ws.onclose = () => {
    ws = null
    wsConnected.value = false
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
  flex-wrap: wrap;
  gap: 16px;
}

.connection-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background-color: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.3);
  border-radius: 10px;
  color: #F59E0B;
  font-size: 13px;
  font-weight: 500;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
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

.backup-status-strip {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 14px;
}

.status-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 30px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.status-chip.success {
  background: rgba(22, 163, 74, 0.12);
  color: #15803d;
}

.status-chip.error {
  background: rgba(220, 38, 38, 0.12);
  color: #dc2626;
}

.status-chip.warning {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}

.status-chip.info {
  background: rgba(37, 99, 235, 0.14);
  color: #2563eb;
}

.status-chip.neutral {
  background: var(--bg-card);
  color: var(--text-secondary);
}

.status-detail {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.5;
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

.logs-meta {
  display: flex;
  align-items: center;
  gap: 8px;
}

.log-count {
  color: var(--text-secondary);
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  opacity: 0.7;
}

.icon-btn.active {
  color: #3B82F6;
  background-color: rgba(59, 130, 246, 0.1);
}

.logs-terminal {
  flex: 1;
  min-height: 300px;
  max-height: 500px;
  background-color: #0F172A;
  border-radius: 12px;
  padding: 24px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 14px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
  border: 1px solid var(--border-strong);
  scroll-behavior: smooth;
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
