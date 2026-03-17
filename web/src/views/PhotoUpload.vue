<template>
  <div class="photo-upload-container">
    <!-- Hero header with status strip -->
    <div class="header">
      <div class="greeting">
        <p class="eyebrow">Photo Upload</p>
        <h1>摄影文件上传</h1>
        <p class="subtitle">扫描本地摄影文件，按拍摄日期自动分类上传到 115 网盘</p>
        <div class="status-strip">
          <span :class="['status-chip', uploadStatusTone]">
            <span v-if="uploadStatus === 'running'" class="pulse-dot"></span>
            {{ uploadStatusLabel }}
          </span>
          <span v-if="uploadStatus === 'running' && lastLogSummary" class="status-detail">{{ lastLogSummary }}</span>
        </div>
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

    <!-- Stats Cards -->
    <div class="stats-grid">
      <div class="stat-card">
        <div class="stat-icon-wrapper blue">
          <LucideFolderInput :size="20" />
        </div>
        <div class="stat-info">
          <span class="stat-label">监控目录</span>
          <span class="stat-value">{{ localWatchDir || '未配置' }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-wrapper green">
          <LucideCloud :size="20" />
        </div>
        <div class="stat-info">
          <span class="stat-label">远端目录</span>
          <span class="stat-value">{{ remoteDir || '未配置' }}</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-wrapper yellow">
          <LucideFileType :size="20" />
        </div>
        <div class="stat-info">
          <span class="stat-label">监控格式</span>
          <span class="stat-value">{{ extensionCount }} 种</span>
        </div>
      </div>

      <div class="stat-card">
        <div class="stat-icon-wrapper" :class="deleteAfterUpload ? 'red' : 'neutral'">
          <LucideTrash2 v-if="deleteAfterUpload" :size="20" />
          <LucideArchive v-else :size="20" />
        </div>
        <div class="stat-info">
          <span class="stat-label">上传后操作</span>
          <span class="stat-value">{{ deleteAfterUpload ? '删除本地' : '保留本地' }}</span>
        </div>
      </div>
    </div>

    <!-- Config Section (collapsible) -->
    <div class="config-section">
      <button class="config-toggle" @click="configExpanded = !configExpanded">
        <div class="config-toggle-left">
          <LucideSettings :size="20" />
          <span>上传配置</span>
        </div>
        <LucideChevronDown :size="18" :class="['chevron', { expanded: configExpanded }]" />
      </button>

      <transition name="slide">
        <div v-if="configExpanded" class="config-body">
          <div class="config-grid">
            <div class="config-item">
              <label>本地监控目录</label>
              <div class="input-with-browse">
                <input type="text" v-model="localWatchDir" placeholder="例如 D:\摄影" />
                <button class="btn secondary browse-btn" @click="browseLocal" title="浏览目录">
                  <LucideFolderOpen :size="16" />
                </button>
              </div>
              <span class="input-hint">摄影文件所在的本地文件夹，支持子目录递归扫描</span>
            </div>

            <div class="config-item">
              <label>远端目标目录</label>
              <input type="text" v-model="remoteDir" placeholder="例如 /摄影" />
              <span class="input-hint">115 网盘上的根目录，文件将按日期子目录归类</span>
            </div>

            <div class="config-item full-width">
              <label>文件扩展名</label>
              <input type="text" v-model="extensions" placeholder="逗号分隔，如 cr2,nef,arw,jpg" />
              <span class="input-hint">
                支持的格式：RAW (CR2, CR3, NEF, ARW, DNG, RAF 等) + JPG/JPEG。同名的 RAW+JPG 文件对将使用 JPG 的 EXIF 日期归类
              </span>
            </div>

            <div class="config-item">
              <label>日期目录格式</label>
              <select v-model="dateFormat">
                <option value="2006/01/02">年/月/日 (2026/03/17)</option>
                <option value="2006/01">年/月 (2026/03)</option>
                <option value="2006-01-02">年-月-日 (2026-03-17)</option>
                <option value="2006">按年 (2026)</option>
              </select>
              <span class="input-hint">上传到远端后的目录层级结构</span>
            </div>

            <div class="config-item">
              <div class="toggle-field" @click="deleteAfterUpload = !deleteAfterUpload">
                <div class="toggle-info">
                  <span class="toggle-title">上传后删除本地文件</span>
                  <span class="toggle-desc">开启后，文件上传成功即从本地删除。请确保网盘文件完整后再启用。</span>
                </div>
                <div :class="['switch', deleteAfterUpload ? 'active' : '']">
                  <div class="thumb"></div>
                </div>
              </div>
              <div v-if="deleteAfterUpload" class="warning-hint">
                <LucideTriangleAlert :size="14" />
                <span>已启用自动删除，上传成功的本地文件将被永久删除</span>
              </div>
            </div>
          </div>

          <div class="config-actions">
            <span v-if="configDirty" class="unsaved-hint">有未保存的更改</span>
            <button class="btn primary" @click="saveConfig" :disabled="saving || !configDirty">
              {{ saving ? '保存中...' : '保存配置' }}
            </button>
          </div>
        </div>
      </transition>
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

    <!-- Logs Section -->
    <div class="logs-section">
      <div class="logs-header">
        <h2>上传日志</h2>
        <div class="logs-meta">
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
        <div v-if="logs.length === 0" class="log-empty-state">
          <LucideTerminal :size="32" />
          <p>暂无日志</p>
          <span>点击「开始上传」后，实时上传日志将显示在这里</span>
        </div>
        <template v-else>
          <div v-for="log in logs" :key="log.id" :class="['log-line', getLogLevelClass(log.text)]">
            <span class="timestamp">[{{ log.time }}]</span>
            <span class="message">{{ formatLogText(log.text) }}</span>
          </div>
          <div class="log-line active">
            <span class="cursor">_</span>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  LucideUpload,
  LucideSquare,
  LucideFolderOpen,
  LucideFolderUp,
  LucideFolder,
  LucideFolderInput,
  LucideCloud,
  LucideFileType,
  LucideX,
  LucideArrowDownToLine,
  LucideTrash2,
  LucideArchive,
  LucideSettings,
  LucideChevronDown,
  LucideTriangleAlert,
  LucideTerminal
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
const configExpanded = ref(false)

// Config dirty tracking
const savedSnapshot = ref('')
const currentSnapshot = computed(() =>
  JSON.stringify({ localWatchDir: localWatchDir.value, remoteDir: remoteDir.value, extensions: extensions.value, dateFormat: dateFormat.value, deleteAfterUpload: deleteAfterUpload.value })
)
const configDirty = computed(() => savedSnapshot.value !== '' && savedSnapshot.value !== currentSnapshot.value)

// Computed for stats cards
const extensionCount = computed(() => {
  if (!extensions.value) return 0
  return extensions.value.split(',').filter(e => e.trim()).length
})

// Upload status
const uploadStatus = ref<'idle' | 'running'>('idle')
const uploadStatusTone = computed(() => uploadStatus.value === 'running' ? 'info' : 'neutral')
const uploadStatusLabel = computed(() => uploadStatus.value === 'running' ? '上传中' : '空闲')

// Last log summary for status strip
const lastLogSummary = computed(() => {
  if (logs.value.length === 0) return ''
  const last = logs.value[logs.value.length - 1]
  if (!last) return ''
  return formatLogText(last.text)
})

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
    // Take initial snapshot after loading
    savedSnapshot.value = currentSnapshot.value

    // Auto-expand config if not yet configured
    if (!cfg.photo_upload?.watch_dir) {
      configExpanded.value = true
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
    savedSnapshot.value = currentSnapshot.value
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
const formatLogText = (text: string) => {
  // Strip the [photo-upload] prefix for cleaner display
  return text.replace(/\[photo-upload\]\s*/g, '')
}

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

      // Auto-detect when upload finishes
      if (text.includes('任务完成') || text.includes('已被取消')) {
        fetchUploadStatus()
      }
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
  statusInterval = setInterval(fetchUploadStatus, 5000)
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
  gap: 32px;
  max-width: 1400px;
  margin: 0 auto;
}

/* ===== Header ===== */
.header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 16px;
}

.eyebrow {
  font-size: 13px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  color: var(--text-tertiary);
  margin-bottom: 4px;
}

.greeting h1 {
  font-family: var(--font-primary);
  font-weight: 800;
  font-size: 32px;
  color: var(--text-primary);
  letter-spacing: -0.5px;
  margin-bottom: 6px;
}

.subtitle {
  color: var(--text-secondary);
  font-size: 15px;
  margin-bottom: 12px;
}

.status-strip {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.status-detail {
  font-size: 13px;
  color: var(--text-secondary);
  max-width: 400px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.actions {
  display: flex;
  gap: 12px;
  flex-shrink: 0;
}

/* ===== Status Chips ===== */
.status-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 26px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.status-chip.info {
  background: rgba(37, 99, 235, 0.14);
  color: #2563eb;
}

.status-chip.neutral {
  background: var(--border-subtle);
  color: var(--text-secondary);
}

.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #2563eb;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.8); }
}

/* ===== Stats Grid ===== */
.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
}

.stat-card {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 20px;
  background-color: var(--bg-card);
  border-radius: 14px;
  border: 1px solid var(--border-subtle);
  transition: transform 0.15s ease, box-shadow 0.15s ease;
}

.stat-card:hover {
  transform: translateY(-1px);
  box-shadow: 0 4px 12px rgba(0,0,0,0.06);
}

.stat-icon-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  flex-shrink: 0;
}

.stat-icon-wrapper.blue { background: rgba(59, 130, 246, 0.12); color: #3B82F6; }
.stat-icon-wrapper.green { background: rgba(16, 185, 129, 0.12); color: #10B981; }
.stat-icon-wrapper.yellow { background: rgba(245, 158, 11, 0.12); color: #F59E0B; }
.stat-icon-wrapper.red { background: rgba(239, 68, 68, 0.12); color: #EF4444; }
.stat-icon-wrapper.neutral { background: var(--border-subtle); color: var(--text-secondary); }

.stat-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.stat-label {
  font-size: 12px;
  color: var(--text-tertiary);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.stat-value {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* ===== Config Section ===== */
.config-section {
  background-color: var(--bg-card);
  border-radius: 16px;
  border: 1px solid var(--border-subtle);
  overflow: hidden;
}

.config-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 20px 28px;
  cursor: pointer;
  color: var(--text-primary);
  transition: background-color 0.15s;
}

.config-toggle:hover {
  background-color: var(--border-subtle);
}

.config-toggle-left {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 16px;
  font-weight: 700;
}

.chevron {
  transition: transform 0.25s ease;
  color: var(--text-tertiary);
}

.chevron.expanded {
  transform: rotate(180deg);
}

/* Slide animation */
.slide-enter-active,
.slide-leave-active {
  transition: all 0.25s ease;
  overflow: hidden;
}

.slide-enter-from,
.slide-leave-to {
  max-height: 0;
  opacity: 0;
  padding-top: 0;
  padding-bottom: 0;
}

.slide-enter-to,
.slide-leave-from {
  max-height: 800px;
  opacity: 1;
}

.config-body {
  padding: 0 28px 28px;
  border-top: 1px solid var(--border-subtle);
  padding-top: 24px;
}

.config-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
}

.config-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.config-item.full-width {
  grid-column: 1 / -1;
}

.config-item label {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
}

.input-hint {
  font-size: 12px;
  color: var(--text-tertiary);
  line-height: 1.4;
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
  box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.08);
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

/* Toggle field (same as Settings.vue) */
.toggle-field {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 14px 18px;
  border-radius: 12px;
  background: var(--bg-primary);
  border: 1px solid var(--border-subtle);
  cursor: pointer;
  transition: background-color 0.15s;
}

.toggle-field:hover {
  background-color: var(--border-subtle);
}

.toggle-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
}

.toggle-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}

.toggle-desc {
  font-size: 12px;
  color: var(--text-tertiary);
  line-height: 1.4;
}

/* Switch (copied from Settings.vue for consistency) */
.switch {
  position: relative;
  width: 44px;
  height: 24px;
  border-radius: 12px;
  background-color: var(--border-strong);
  transition: background-color 0.2s ease;
  flex-shrink: 0;
}

.switch.active {
  background-color: #10B981;
}

.thumb {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 18px;
  height: 18px;
  border-radius: 50%;
  background-color: white;
  transition: transform 0.2s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.2);
}

.switch.active .thumb {
  transform: translateX(20px);
}

.warning-hint {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border-radius: 8px;
  background: rgba(245, 158, 11, 0.08);
  color: #D97706;
  font-size: 12px;
  font-weight: 500;
  margin-top: 4px;
}

.config-actions {
  margin-top: 24px;
  display: flex;
  justify-content: flex-end;
  align-items: center;
  gap: 16px;
}

.unsaved-hint {
  font-size: 13px;
  font-weight: 600;
  color: #F59E0B;
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; }
  to { opacity: 1; }
}

/* ===== Browse Modal ===== */
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  backdrop-filter: blur(2px);
}

.modal {
  background: var(--bg-card);
  border-radius: 16px;
  width: min(520px, calc(100vw - 32px));
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border-strong);
  box-shadow: 0 25px 50px rgba(0, 0, 0, 0.25);
  animation: modalIn 0.2s ease;
}

@keyframes modalIn {
  from { opacity: 0; transform: scale(0.97) translateY(8px); }
  to { opacity: 1; transform: scale(1) translateY(0); }
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

/* ===== Logs Section ===== */
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
  border-radius: 8px;
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
  font-size: 13px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 4px;
  border: 1px solid var(--border-strong);
  scroll-behavior: smooth;
}

.log-empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: #475569;
  min-height: 200px;
}

.log-empty-state p {
  font-size: 15px;
  font-weight: 600;
  color: #64748B;
}

.log-empty-state span {
  font-size: 13px;
  color: #475569;
}

.log-line {
  display: flex;
  gap: 12px;
  line-height: 1.6;
}

.timestamp {
  color: #475569;
  flex-shrink: 0;
}

.info .message { color: #CBD5E1; }
.success .message { color: #34D399; }
.warning .message { color: #FCD34D; }
.error .message { color: #FCA5A5; }

.cursor {
  color: #CBD5E1;
  animation: blink 1s step-end infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

/* ===== Responsive ===== */
@media (max-width: 1024px) {
  .photo-upload-container {
    padding: 32px 24px;
  }

  .stats-grid {
    grid-template-columns: repeat(2, 1fr);
  }

  .config-grid {
    grid-template-columns: 1fr;
  }

  .config-item.full-width {
    grid-column: 1;
  }
}

@media (max-width: 640px) {
  .photo-upload-container {
    padding: 24px 16px;
  }

  .header {
    flex-direction: column;
  }

  .actions {
    width: 100%;
  }

  .actions .btn {
    flex: 1;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }
}
</style>
