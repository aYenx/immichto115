<template>
  <div class="photo-upload-container">

    <!-- Compact Header -->
    <div class="page-header">
      <div class="header-left">
        <h2>摄影文件上传</h2>
        <span :class="['status-badge', uploadStatusTone]">
          <span v-if="uploadStatus === BackupStatus.Running" class="pulse-dot"></span>
          {{ uploadStatusLabel }}
        </span>
      </div>
      <div class="header-actions">
        <button class="btn secondary" @click="stopUpload" :disabled="uploadStatus !== 'running'">
          <LucideSquare :size="16" />
          停止
        </button>
        <button class="btn primary" @click="startUpload"
          :disabled="uploadStatus === BackupStatus.Running || !config?.photo_upload?.watch_dir">
          <LucideUpload :size="16" />
          开始上传
        </button>
      </div>
    </div>

    <!-- Running status detail -->
    <div v-if="uploadStatus === BackupStatus.Running && lastLogSummary" class="running-detail">
      <LucideActivity :size="14" class="running-icon" />
      <span>{{ lastLogSummary }}</span>
    </div>

    <!-- Compact Info Bar -->
    <div class="info-bar">
      <div class="info-item">
        <LucideFolderInput :size="16" class="info-icon" />
        <span class="info-label">监控</span>
        <span class="info-value" :title="localWatchDir || '未配置'">{{ localWatchDir || '未配置' }}</span>
      </div>
      <div class="info-divider"></div>
      <div class="info-item">
        <LucideCloud :size="16" class="info-icon" />
        <span class="info-label">远端</span>
        <span class="info-value" :title="remoteDir || '未配置'">{{ remoteDir || '未配置' }}</span>
      </div>
      <div class="info-divider"></div>
      <div class="info-item">
        <LucideFileType :size="16" class="info-icon" />
        <span class="info-label">格式</span>
        <span class="info-value">{{ extensionCount }} 种</span>
      </div>
      <div class="info-divider"></div>
      <div class="info-item">
        <component :is="deleteAfterUpload ? LucideTrash2 : LucideArchive" :size="16"
          :class="['info-icon', deleteAfterUpload ? 'text-red' : '']" />
        <span class="info-label">上传后</span>
        <span class="info-value">{{ deleteAfterUpload ? '删除本地' : '保留本地' }}</span>
      </div>
    </div>

    <!-- Unconfigured banner -->
    <div v-if="!config?.photo_upload?.watch_dir && !configExpanded" class="setup-banner" @click="configExpanded = true">
      <LucideInfo :size="16" />
      <span>尚未配置监控目录，点击此处展开配置面板开始设置</span>
      <LucideChevronRight :size="16" />
    </div>

    <!-- Config Section (collapsible) -->
    <div class="config-section">
      <button class="config-toggle" @click="configExpanded = !configExpanded">
        <div class="config-toggle-left">
          <LucideSettings :size="18" />
          <span>上传配置</span>
        </div>
        <LucideChevronDown :size="16" :class="['chevron', { expanded: configExpanded }]" />
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
              <div class="input-with-browse">
                <input type="text" v-model="remoteDir" placeholder="例如 /摄影" />
                <button class="btn secondary browse-btn" @click="browseRemote" title="浏览远端目录">
                  <LucideFolderOpen :size="16" />
                </button>
              </div>
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

    <!-- Local Browse Modal -->
    <div v-if="showBrowseModal" class="modal-overlay" @click.self="showBrowseModal = false">
      <div class="modal">
        <div class="modal-header">
          <h3>选择本地目录</h3>
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

    <!-- Remote Browse Modal -->
    <div v-if="showRemoteBrowseModal" class="modal-overlay" @click.self="showRemoteBrowseModal = false">
      <div class="modal">
        <div class="modal-header">
          <h3>选择远端目录</h3>
          <button class="icon-btn" @click="showRemoteBrowseModal = false">
            <LucideX :size="18" />
          </button>
        </div>
        <div class="browse-path">
          <span>{{ remoteBrowsePath || '/' }}</span>
        </div>
        <div class="browse-list">
          <div v-if="remoteBrowseLoading" class="browse-empty">
            <LucideLoader2 :size="20" class="spin-icon" />
            加载中...
          </div>
          <template v-else>
            <div v-if="remoteBrowsePath && remoteBrowsePath !== '/'" class="browse-item" @click="remoteBrowseUp">
              <LucideFolderUp :size="16" />
              <span>..</span>
            </div>
            <div v-for="entry in remoteBrowseEntries" :key="entry.Path || entry.Name" class="browse-item" @click="remoteBrowseInto(entry)">
              <LucideFolder :size="16" />
              <span>{{ entry.Name }}</span>
            </div>
            <div v-if="remoteBrowseEntries.length === 0" class="browse-empty">
              该目录下没有子文件夹
            </div>
          </template>
        </div>
        <div class="modal-footer">
          <button class="btn secondary" @click="showRemoteBrowseModal = false">取消</button>
          <button class="btn primary" @click="selectRemoteBrowsePath">选择此目录</button>
        </div>
      </div>
    </div>

    <!-- Logs Section -->
    <div class="logs-section">
      <div class="logs-header">
        <div class="logs-title-row">
          <h3>上传日志</h3>
          <span class="log-count">{{ logs.length }} / {{ MAX_LOGS }}</span>
        </div>
        <div class="logs-actions">
          <button class="icon-btn" :class="{ active: autoScroll }" @click="autoScroll = !autoScroll"
            :title="autoScroll ? '自动滚动已开启' : '自动滚动已关闭'">
            <LucideArrowDownToLine :size="15" />
          </button>
          <button class="icon-btn" @click="clearLogs" title="清空日志">
            <LucideTrash2 :size="15" />
          </button>
        </div>
      </div>

      <div class="logs-terminal" ref="terminalRef" @scroll="onTerminalScroll">
        <div v-if="logs.length === 0" class="log-empty-state">
          <LucideTerminal :size="28" />
          <p>暂无日志</p>
          <span>点击「开始上传」后，实时日志将显示在这里</span>
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
  LucideChevronRight,
  LucideTriangleAlert,
  LucideTerminal,
  LucideActivity,
  LucideInfo,
  LucideLoader2
} from 'lucide-vue-next'
import { api, getErrorMessage, handleAuthFailure, safeConfigToAppConfig, appConfigToUpdateRequest, type AppConfig, type SafeConfigResponse, type DirEntry } from '../api'
import { BackupStatus } from '../constants'
import { showToast } from '../composables/toast'

const MAX_LOGS = 200
let logIdCounter = 0

// Config state
const config = ref<AppConfig | null>(null)
const safeResponse = ref<SafeConfigResponse | null>(null)
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
const uploadStatusTone = computed(() => uploadStatus.value === BackupStatus.Running ? 'info' : 'neutral')
const uploadStatusLabel = computed(() => uploadStatus.value === BackupStatus.Running ? '上传中' : '空闲')

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

// Browse local
const showBrowseModal = ref(false)
const browsePath = ref('')
const browseEntries = ref<DirEntry[]>([])

// Browse remote
const showRemoteBrowseModal = ref(false)
const remoteBrowsePath = ref('/')
const remoteBrowseEntries = ref<DirEntry[]>([])
const remoteBrowseLoading = ref(false)
const browseLoading = ref(false)

// Status polling
let statusInterval: ReturnType<typeof setInterval> | null = null

const loadConfig = async () => {
  try {
    const safe = await api.getConfig()
    safeResponse.value = safe
    const cfg = safeConfigToAppConfig(safe)
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
  if (!config.value || !safeResponse.value) return
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
    const result = await api.saveConfig(appConfigToUpdateRequest(updated, safeResponse.value))
    safeResponse.value = { ...safeResponse.value, updated_at: result.updated_at }
    config.value = updated
    config.value.updated_at = result.updated_at
    savedSnapshot.value = currentSnapshot.value
    showToast('info', '配置已保存', '摄影上传配置已更新')
  } catch (err) {
    if (handleAuthFailure(err)) throw err
    showToast('error', '保存失败', getErrorMessage(err))
    throw err // 向调用方传播（startUpload 需要感知保存失败）
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
    // 先保存当前界面配置，确保后端使用的是用户看到的值
    await saveConfig()
    await saveConfig()
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
  let parent = parts.join('/')
  // Only jump to drive list when parent is truly empty (was at root)
  if (parent === '') {
    browsePath.value = ''
    loadBrowseDir('')
    return
  }
  // Normalize "C:" → "C:/" for proper directory loading
  if (/^[A-Za-z]:$/.test(parent)) parent += '/'
  browsePath.value = parent
  loadBrowseDir(parent)
}

const selectBrowsePath = () => {
  localWatchDir.value = browsePath.value
  showBrowseModal.value = false
}

// Browse remote directories
const browseRemote = async () => {
  if (!config.value) {
    showToast('warning', '配置未加载', '请等待配置加载完成后再试')
    return
  }
  const provider = config.value.provider
  if (provider === 'open115') {
    if (!config.value.open115?.access_token?.trim() || !config.value.open115?.refresh_token?.trim()) {
      showToast('warning', '请先完成授权', '需要先在设置中完成 115 Open 授权，才能浏览远端目录。')
      return
    }
  } else {
    if (!config.value.webdav?.url?.trim() || !config.value.webdav?.user?.trim() || !config.value.webdav?.password?.trim()) {
      showToast('warning', '请先完善连接信息', '需要先在设置中填写 WebDAV 地址、用户名和密码。')
      return
    }
  }
  showRemoteBrowseModal.value = true
  remoteBrowsePath.value = remoteDir.value || '/'
  await loadRemoteBrowseDir(remoteBrowsePath.value)
}

const normalizeRemotePath = (p: string) => {
  const cleaned = ('/' + p.replace(/\\/g, '/')).replace(/\/+/g, '/')
  return cleaned || '/'
}

const loadRemoteBrowseDir = async (path: string) => {
  if (!config.value) return
  remoteBrowseLoading.value = true
  const normalizedPath = normalizeRemotePath(path)
  try {
    const items = config.value.provider === 'open115'
      ? await api.open115List(normalizedPath, {
          access_token: config.value.open115.access_token,
          refresh_token: config.value.open115.refresh_token,
          root_id: config.value.open115.root_id,
        })
      : await api.listWebDAV({
          url: config.value.webdav.url,
          user: config.value.webdav.user,
          password: config.value.webdav.password,
          vendor: config.value.webdav.vendor,
          path: normalizedPath,
        })
    remoteBrowseEntries.value = items.filter((e: any) => e.IsDir)
    remoteBrowsePath.value = normalizedPath
  } catch (err) {
    remoteBrowseEntries.value = []
    if (handleAuthFailure(err)) return
    showToast('error', '浏览失败', getErrorMessage(err))
  } finally {
    remoteBrowseLoading.value = false
  }
}

const remoteBrowseInto = (entry: any) => {
  const newPath = normalizeRemotePath(remoteBrowsePath.value + '/' + entry.Name)
  loadRemoteBrowseDir(newPath)
}

const remoteBrowseUp = () => {
  const parts = remoteBrowsePath.value.split('/').filter(Boolean)
  parts.pop()
  const parent = parts.length > 0 ? '/' + parts.join('/') : '/'
  loadRemoteBrowseDir(parent)
}

const selectRemoteBrowsePath = () => {
  remoteDir.value = remoteBrowsePath.value
  showRemoteBrowseModal.value = false
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
  padding: 0 64px 48px;
  gap: 20px;
  max-width: 1400px;
  margin: 0 auto;
}



/* ===== Compact Header ===== */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-top: 32px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 14px;
}

.header-left h2 {
  font-family: var(--font-primary);
  font-weight: 800;
  font-size: 24px;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}

.header-actions {
  display: flex;
  gap: 10px;
}

/* ===== Status Badge ===== */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 26px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.status-badge.info {
  background: rgba(37, 99, 235, 0.14);
  color: #2563eb;
}

.status-badge.neutral {
  background: var(--border-subtle);
  color: var(--text-secondary);
}

.pulse-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background-color: #2563eb;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.8); }
}

/* ===== Running Detail ===== */
.running-detail {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: rgba(37, 99, 235, 0.06);
  border: 1px solid rgba(37, 99, 235, 0.15);
  border-radius: 10px;
  font-size: 13px;
  color: var(--text-secondary);
  animation: fadeSlideIn 0.3s ease;
}

.running-icon {
  color: #3B82F6;
  flex-shrink: 0;
  animation: breathe 2s ease-in-out infinite;
}

@keyframes breathe {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

@keyframes fadeSlideIn {
  from { opacity: 0; transform: translateY(-4px); }
  to { opacity: 1; transform: translateY(0); }
}

/* ===== Compact Info Bar ===== */
.info-bar {
  display: flex;
  align-items: center;
  gap: 0;
  padding: 14px 20px;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.04), rgba(59, 130, 246, 0.06));
  border: 1px solid var(--border-subtle);
  border-radius: 12px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 0;
}

.info-icon {
  color: var(--text-tertiary);
  flex-shrink: 0;
}

.info-icon.text-red {
  color: #EF4444;
}

.info-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-tertiary);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  flex-shrink: 0;
}

.info-value {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}

.info-divider {
  width: 1px;
  height: 24px;
  background: var(--border-strong);
  margin: 0 16px;
  flex-shrink: 0;
}

/* ===== Setup Banner ===== */
.setup-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px 20px;
  background: rgba(245, 158, 11, 0.08);
  border: 1px dashed rgba(245, 158, 11, 0.3);
  border-radius: 12px;
  color: #D97706;
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s;
}

.setup-banner:hover {
  background: rgba(245, 158, 11, 0.12);
}

/* ===== Config Section ===== */
.config-section {
  background-color: var(--bg-card);
  border-radius: 14px;
  border: 1px solid var(--border-subtle);
  overflow: hidden;
}

.config-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 16px 24px;
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
  gap: 10px;
  font-size: 15px;
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
  padding: 0 24px 24px;
  border-top: 1px solid var(--border-subtle);
  padding-top: 20px;
}

.config-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 18px;
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
  height: 40px;
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
  height: 40px !important;
  padding: 0 12px !important;
}

/* Toggle field */
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

/* Switch */
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
  margin-top: 20px;
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

.spin-icon { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

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
  gap: 12px;
  flex: 1;
}

.logs-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.logs-title-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

.logs-header h3 {
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
}

.logs-actions {
  display: flex;
  gap: 4px;
}

.log-count {
  color: var(--text-tertiary);
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
}

.icon-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
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
  min-height: 180px;
  max-height: 360px;
  background-color: #0F172A;
  border-radius: 10px;
  padding: 18px 20px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 13px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 3px;
  border: 1px solid var(--border-strong);
  scroll-behavior: smooth;
}

.log-empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  color: #475569;
  min-height: 140px;
}

.log-empty-state p {
  font-size: 14px;
  font-weight: 600;
  color: #64748B;
}

.log-empty-state span {
  font-size: 12px;
  color: #475569;
}

.log-line {
  display: flex;
  gap: 10px;
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
    padding: 0 24px 32px;
  }



  .config-grid {
    grid-template-columns: 1fr;
  }

  .config-item.full-width {
    grid-column: 1;
  }

  .info-bar {
    flex-wrap: wrap;
    gap: 0;
  }

  .info-item {
    flex: 0 0 calc(50% - 16px);
    padding: 4px 0;
  }

  .info-divider {
    display: none;
  }
}

@media (max-width: 640px) {
  .photo-upload-container {
    padding: 0 16px 24px;
    gap: 16px;
  }



  .page-header {
    flex-direction: column;
    align-items: stretch;
    gap: 12px;
    padding-top: 20px;
  }

  .header-left {
    flex-wrap: wrap;
  }

  .header-left h2 {
    font-size: 20px;
  }

  .header-actions {
    width: 100%;
  }

  .header-actions .btn {
    flex: 1;
  }

  .info-bar {
    flex-direction: column;
    gap: 0;
    padding: 12px 16px;
  }

  .info-item {
    flex: 1 0 100%;
    padding: 6px 0;
  }

  .info-divider {
    display: none;
  }

  .logs-terminal {
    min-height: 150px;
    max-height: 280px;
    padding: 14px 16px;
    font-size: 12px;
  }
}
</style>
