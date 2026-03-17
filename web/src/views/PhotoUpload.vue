<template>
  <div class="photo-upload-container">
    <div class="header">
      <div class="greeting">
        <h1>📷 摄影文件上传</h1>
        <p>扫描本地摄影文件，按拍摄日期自动分类上传到 115 网盘</p>
      </div>
      <div class="actions">
        <button class="btn secondary" @click="stopUpload" :disabled="uploadStatus !== 'running'">
          <LucideSquare :size="16" />
          停止上传
        </button>
        <button class="btn primary" @click="startUpload" :disabled="uploadStatus === 'running' || !config?.photo_upload?.watch_dir">
          <LucideUpload :size="16" />
          开始上传
        </button>
      </div>
    </div>

    <!-- Config Section -->
    <div class="config-section">
      <h2>上传配置</h2>
      <div class="config-grid">
        <div class="config-item">
          <label>本地监控目录</label>
          <div class="input-with-browse">
            <input type="text" v-model="localWatchDir" placeholder="例如 D:\摄影" />
            <button class="btn secondary browse-btn" @click="browseLocal" title="浏览目录">
              <LucideFolderOpen :size="16" />
            </button>
          </div>
        </div>

        <div class="config-item">
          <label>远端目标目录</label>
          <input type="text" v-model="remoteDir" placeholder="例如 /摄影" />
        </div>

        <div class="config-item">
          <label>文件扩展名</label>
          <input type="text" v-model="extensions" placeholder="逗号分隔，如 cr2,nef,arw,jpg" />
        </div>

        <div class="config-item">
          <label>日期目录格式</label>
          <select v-model="dateFormat">
            <option value="2006/01/02">年/月/日 (2026/03/17)</option>
            <option value="2006/01">年/月 (2026/03)</option>
            <option value="2006-01-02">年-月-日 (2026-03-17)</option>
            <option value="2006">按年 (2026)</option>
          </select>
        </div>

        <div class="config-item toggle-item">
          <label>上传后删除本地文件</label>
          <button :class="['toggle', { active: deleteAfterUpload }]" @click="deleteAfterUpload = !deleteAfterUpload">
            <span class="toggle-knob"></span>
          </button>
        </div>
      </div>

      <div class="config-actions">
        <button class="btn primary" @click="saveConfig" :disabled="saving">
          {{ saving ? '保存中...' : '保存配置' }}
        </button>
      </div>
    </div>

    <!-- Browse Modal -->
    <div v-if="showBrowseModal" class="modal-overlay" @click.self="showBrowseModal = false">
      <div class="modal">
        <div class="modal-header">
          <h3>选择目录</h3>
          <button class="icon-btn" @click="showBrowseModal = false">
            <LucideX :size="18" />
          </button>
        </div>
        <div class="browse-path">
          <span>{{ browsePath || '/' }}</span>
        </div>
        <div class="browse-list">
          <div v-if="browsePath" class="browse-item" @click="browseUp">
            <LucideFolderUp :size="16" />
            <span>..</span>
          </div>
          <div v-for="entry in browseEntries" :key="entry.Path" class="browse-item" @click="browseInto(entry)">
            <LucideFolder :size="16" />
            <span>{{ entry.Name }}</span>
          </div>
          <div v-if="browseEntries.length === 0 && !browseLoading" class="browse-empty">
            空目录
          </div>
          <div v-if="browseLoading" class="browse-empty">
            加载中...
          </div>
        </div>
        <div class="modal-footer">
          <button class="btn secondary" @click="showBrowseModal = false">取消</button>
          <button class="btn primary" @click="selectBrowsePath">选择此目录</button>
        </div>
      </div>
    </div>

    <!-- Status & Logs -->
    <div class="logs-section">
      <div class="logs-header">
        <h2>上传日志</h2>
        <div class="logs-meta">
          <span :class="['status-chip', uploadStatus === 'running' ? 'info' : 'neutral']">
            {{ uploadStatus === 'running' ? '上传中' : '空闲' }}
          </span>
          <span class="log-count">{{ logs.length }} / {{ MAX_LOGS }} 条</span>
          <button class="icon-btn" :class="{ active: autoScroll }" @click="autoScroll = !autoScroll"
            :title="autoScroll ? '自动滚动已开启' : '自动滚动已关闭'">
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
        <div class="log-line active">
          <span class="cursor">_</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import {
  LucideUpload,
  LucideSquare,
  LucideFolderOpen,
  LucideFolderUp,
  LucideFolder,
  LucideX,
  LucideArrowDownToLine,
  LucideTrash2
} from 'lucide-vue-next'
import { api, getErrorMessage, handleAuthFailure, type AppConfig, type DirEntry } from '../api'
import { showToast } from '../composables/toast'

const MAX_LOGS = 200
let logIdCounter = 0

// Config state
const config = ref<AppConfig | null>(null)
const localWatchDir = ref('')
const remoteDir = ref('/摄影')
const extensions = ref('cr2,cr3,nef,arw,dng,raf,rw2,orf,pef,srw,jpg,jpeg')
const dateFormat = ref('2006/01/02')
const deleteAfterUpload = ref(true)
const saving = ref(false)

// Upload status
const uploadStatus = ref<'idle' | 'running'>('idle')

// Logs
const logs = ref<Array<{ id: number; time: string; text: string }>>([])
const terminalRef = ref<HTMLElement | null>(null)
const autoScroll = ref(true)
let scrollRAF: number | null = null

// WebSocket
let ws: WebSocket | null = null
let reconnectTimer: ReturnType<typeof setTimeout> | null = null
let shouldReconnect = true

// Browse
const showBrowseModal = ref(false)
const browsePath = ref('')
const browseEntries = ref<DirEntry[]>([])
const browseLoading = ref(false)

// Status polling
let statusInterval: ReturnType<typeof setInterval> | null = null

const loadConfig = async () => {
  try {
    const cfg = await api.getConfig()
    config.value = cfg
    if (cfg.photo_upload) {
      localWatchDir.value = cfg.photo_upload.watch_dir || ''
      remoteDir.value = cfg.photo_upload.remote_dir || '/摄影'
      extensions.value = cfg.photo_upload.extensions || 'cr2,cr3,nef,arw,dng,raf,rw2,orf,pef,srw,jpg,jpeg'
      dateFormat.value = cfg.photo_upload.date_format || '2006/01/02'
      deleteAfterUpload.value = cfg.photo_upload.delete_after_upload ?? true
    }
  } catch (err) {
    if (handleAuthFailure(err)) return
    showToast('error', '加载配置失败', getErrorMessage(err))
  }
}

const saveConfig = async () => {
  if (!config.value) return
  saving.value = true
  try {
    const updated = { ...config.value }
    updated.photo_upload = {
      ...updated.photo_upload,
      enabled: true,
      watch_dir: localWatchDir.value,
      remote_dir: remoteDir.value,
      extensions: extensions.value,
      date_format: dateFormat.value,
      delete_after_upload: deleteAfterUpload.value
    }
    await api.saveConfig(updated)
    config.value = updated
    showToast('info', '配置已保存', '摄影上传配置已更新')
  } catch (err) {
    if (handleAuthFailure(err)) return
    showToast('error', '保存失败', getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

const fetchUploadStatus = async () => {
  try {
    const status = await api.photoUploadStatus()
    uploadStatus.value = status.status as 'idle' | 'running'
  } catch (err) {
    if (handleAuthFailure(err)) return
  }
}

const startUpload = async () => {
  try {
    logs.value = []
    const result = await api.photoUploadStart()
    await fetchUploadStatus()
    showToast('info', '上传已开始', result.message || '正在扫描并上传摄影文件...')
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    showToast('error', '启动失败', getErrorMessage(err))
  }
}

const stopUpload = async () => {
  try {
    const result = await api.photoUploadStop()
    await fetchUploadStatus()
    showToast('warning', '已停止上传', result.message || '上传任务已停止')
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    showToast('error', '停止失败', getErrorMessage(err))
  }
}

// Browse local directories
const browseLocal = async () => {
  showBrowseModal.value = true
  browsePath.value = localWatchDir.value || ''
  await loadBrowseDir(browsePath.value)
}

const loadBrowseDir = async (path: string) => {
  browseLoading.value = true
  try {
    const entries = await api.listLocal(path)
    browseEntries.value = entries.filter((e: DirEntry) => e.IsDir)
  } catch (err) {
    browseEntries.value = []
    if (handleAuthFailure(err)) return
  } finally {
    browseLoading.value = false
  }
}

const browseInto = (entry: DirEntry) => {
  browsePath.value = entry.Path
  loadBrowseDir(entry.Path)
}

const browseUp = () => {
  const parts = browsePath.value.replace(/\\/g, '/').split('/')
  parts.pop()
  const parent = parts.join('/') || (browsePath.value.includes(':') ? parts[0] + '/' : '/')
  browsePath.value = parent
  loadBrowseDir(parent)
}

const selectBrowsePath = () => {
  localWatchDir.value = browsePath.value
  showBrowseModal.value = false
}

// Log handling
const getLogLevelClass = (text: string) => {
  const upper = text.toUpperCase()
  if (upper.includes('ERROR') || upper.includes('FAILED') || text.includes('失败')) return 'error'
  if (upper.includes('WARN') || text.includes('停止') || text.includes('取消')) return 'warning'
  if (upper.includes('SUCCESS') || text.includes('成功') || text.includes('完成') || text.includes('已删除')) return 'success'
  return 'info'
}

const clearLogs = () => { logs.value = [] }

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
  autoScroll.value = el.scrollHeight - el.scrollTop - el.clientHeight < 50
}

// WebSocket for real-time logs
const disconnectRealtime = () => {
  shouldReconnect = false
  if (statusInterval) { clearInterval(statusInterval); statusInterval = null }
  if (reconnectTimer) { clearTimeout(reconnectTimer); reconnectTimer = null }
  if (ws) { ws.close(); ws = null }
}

const connectWebSocket = () => {
  if (!shouldReconnect) return
  if (ws && (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING)) return

  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${protocol}//${window.location.host}/ws/logs`

  ws = new WebSocket(wsUrl)
  ws.onopen = () => {}
  ws.onmessage = (ev) => {
    try {
      const data = JSON.parse(ev.data)
      const text = data.Text || data.text || ''
      // Only show photo-upload logs on this page
      if (!text.includes('[photo-upload]')) return
      const now = new Date()
      logs.value.push({
        id: ++logIdCounter,
        time: now.toLocaleTimeString(),
        text
      })
      if (logs.value.length > MAX_LOGS) {
        logs.value.splice(0, logs.value.length - MAX_LOGS)
      }
      scrollToBottom()
    } catch { /* ignore */ }
  }
  ws.onclose = () => {
    ws = null
    if (!shouldReconnect) return
    reconnectTimer = setTimeout(() => { reconnectTimer = null; connectWebSocket() }, 5000)
  }
}

onMounted(() => {
  shouldReconnect = true
  loadConfig()
  fetchUploadStatus()
  statusInterval = setInterval(fetchUploadStatus, 3000)
  connectWebSocket()
})

onUnmounted(() => {
  disconnectRealtime()
})
</script>

<style scoped>
.photo-upload-container {
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

/* Config Section */
.config-section {
  background-color: var(--bg-card);
  border-radius: 16px;
  border: 1px solid var(--border-subtle);
  padding: 32px;
}

.config-section h2 {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin-bottom: 24px;
}

.config-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.config-item label {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
}

.config-item input,
.config-item select {
  height: 42px;
  padding: 0 14px;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  background-color: var(--bg-primary);
  color: var(--text-primary);
  font-size: 14px;
  font-family: inherit;
  transition: border-color 0.2s;
}

.config-item input:focus,
.config-item select:focus {
  border-color: #3B82F6;
  outline: none;
}

.input-with-browse {
  display: flex;
  gap: 8px;
}

.input-with-browse input {
  flex: 1;
}

.browse-btn {
  height: 42px !important;
  padding: 0 12px !important;
}

.toggle-item {
  flex-direction: row !important;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
}

.toggle {
  position: relative;
  width: 48px;
  height: 26px;
  border-radius: 13px;
  background-color: var(--border-strong);
  border: none;
  cursor: pointer;
  transition: background-color 0.2s ease;
}

.toggle.active {
  background-color: #10B981;
}

.toggle-knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background-color: white;
  transition: transform 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.toggle.active .toggle-knob {
  transform: translateX(22px);
}

.config-actions {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
}

/* Browse Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-card);
  border-radius: 16px;
  width: 520px;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-strong);
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.25);
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-subtle);
}

.modal-header h3 {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
}

.browse-path {
  padding: 12px 24px;
  background: var(--bg-primary);
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border-subtle);
}

.browse-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
  max-height: 400px;
}

.browse-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 24px;
  cursor: pointer;
  color: var(--text-primary);
  font-size: 14px;
  transition: background-color 0.15s;
}

.browse-item:hover {
  background-color: var(--bg-primary);
}

.browse-empty {
  padding: 24px;
  text-align: center;
  color: var(--text-tertiary);
  font-size: 14px;
}

.modal-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 20px 24px;
  border-top: 1px solid var(--border-subtle);
}

/* Status chip */
.status-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 26px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.status-chip.info {
  background: rgba(37, 99, 235, 0.14);
  color: #2563eb;
}

.status-chip.neutral {
  background: var(--bg-card);
  color: var(--text-secondary);
}

/* Logs - same style as Dashboard */
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

.info .message { color: #E2E8F0; }
.success .message { color: #10B981; }
.warning .message { color: #FCD34D; }
.error .message { color: #FCA5A5; }

.cursor {
  color: #E2E8F0;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}
</style>
