<template>
  <div class="explore-container">
    <div class="header">
      <div class="title-group">
        <h1>云端文件浏览</h1>
        <p>浏览云端备份的文件与目录结构（恢复功能开发中）</p>
      </div>
      <div class="actions">
        <button class="btn secondary" @click="fetchList" :disabled="isLoading">
          <LucideRefreshCw :size="16" :class="{ 'spin': isLoading }" />
          刷新列表
        </button>
      </div>
    </div>

    <div class="explorer-card">
      <nav class="breadcrumb" aria-label="目录导航">
        <div class="path-item" role="button" tabindex="0" @click="navigateTo('')" @keydown.enter="navigateTo('')" :class="{ active: currentPath === '' }">
          <LucideCloud :size="16" />
          <span>根目录</span>
        </div>
        <template v-for="(segment, idx) in breadcrumbs" :key="idx">
          <LucideChevronRight :size="16" class="separator" aria-hidden="true" />
          <div class="path-item" role="button" tabindex="0" @click="navigateTo(segment.path)" @keydown.enter="navigateTo(segment.path)" :class="{ active: idx === breadcrumbs.length - 1 }">
            <span>{{ segment.name }}</span>
          </div>
        </template>
      </nav>

      <div class="file-table">
        <div class="table-header">
          <div class="col-name" style="padding-left: 16px;">名称</div>
          <div class="col-size">大小</div>
          <div class="col-date">修改日期</div>
          <div class="col-actions"></div>
        </div>

        <div class="table-body">
          <div v-if="isLoading" class="table-empty">
            <LucideRefreshCw :size="32" class="empty-icon spin" />
            <span class="empty-title">加载中...</span>
          </div>
          <div v-else-if="items.length === 0" class="table-empty">
            <LucideFolderOpen :size="40" class="empty-icon" />
            <span class="empty-title">暂无文件</span>
            <span class="empty-desc">当前目录下没有文件或文件夹</span>
          </div>
          
          <template v-else>
            <div v-if="canGoUp" class="table-row folder go-up" @click="goUp()">
              <div class="col-name" style="padding-left: 16px;">
                <LucideCornerLeftUp :size="20" class="icon text-gray" />
                <span>返回上级目录</span>
              </div>
              <div class="col-size">--</div>
              <div class="col-date">--</div>
              <div class="col-actions">
                <button class="action-btn"><LucideChevronRight :size="18" /></button>
              </div>
            </div>

            <div class="table-row folder" v-for="folder in folders" :key="folder.Path || folder.Name" @click="navigateToFolder(folder)">
              <div class="col-name" style="padding-left: 16px;">
                <LucideFolder :size="20" class="icon text-blue" />
                <span>{{ folder.Name }}</span>
              </div>
              <div class="col-size">--</div>
              <div class="col-date">{{ formatDate(folder.ModTime ?? '') }}</div>
              <div class="col-actions">
                <button class="action-btn"><LucideChevronRight :size="18" /></button>
              </div>
            </div>

            <div class="table-row file" v-for="file in files" :key="file.Path || file.Name">
              <div class="col-name" style="padding-left: 16px;">
                <LucideImage :size="20" class="icon text-gray" v-if="isImage(file.Name)" />
                <LucideFileJson :size="20" class="icon text-yellow" v-else-if="isJson(file.Name)" />
                <LucideFile :size="20" class="icon text-gray" v-else />
                <span>{{ file.Name }}</span>
              </div>
              <div class="col-size">{{ formatSize(file.Size ?? 0) }}</div>
              <div class="col-date">{{ formatDate(file.ModTime ?? '') }}</div>
              <div class="col-actions"></div>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { 
  LucideRefreshCw, 
  LucideCloud, 
  LucideChevronRight, 
  LucideCornerLeftUp,
  LucideFolder, 
  LucideFolderOpen,
  LucideImage, 
  LucideFile,
  LucideFileJson
} from 'lucide-vue-next'
import { api, getErrorMessage, handleAuthFailure, type DirEntry } from '../api'
import { showToast } from '../composables/toast'

const currentPath = ref('')
const isLoading = ref(false)
const items = ref<DirEntry[]>([])


const breadcrumbs = computed(() => {
  if (!currentPath.value) return []
  const parts = currentPath.value.split('/').filter(Boolean)
  return parts.map((part, index) => {
    return {
      name: part,
      path: parts.slice(0, index + 1).join('/')
    }
  })
})

const folders = computed(() => items.value.filter(i => i.IsDir).sort((a, b) => a.Name.localeCompare(b.Name)))
const files = computed(() => items.value.filter(i => !i.IsDir).sort((a, b) => a.Name.localeCompare(b.Name)))
const canGoUp = computed(() => normalizePath(currentPath.value) !== '')

const fetchList = async () => {
  isLoading.value = true
  try {
    const normalizedPath = currentPath.value ? `/${normalizePath(currentPath.value)}` : '/'
    const data = await api.listRemote(normalizedPath)
    items.value = Array.isArray(data) ? data : []
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    showToast('error', '请求文件列表失败', getErrorMessage(err))
    items.value = []
  } finally {
    isLoading.value = false
  }
}

const navigateTo = (path: string) => {
  currentPath.value = normalizePath(path)
  fetchList()
}

const normalizePath = (value: string) => value.replace(/^\/+|\/+$/g, '')

const goUp = () => {
  const parts = normalizePath(currentPath.value).split('/').filter(Boolean)
  parts.pop()
  navigateTo(parts.join('/'))
}

const navigateToFolder = (folder: DirEntry) => {
  const folderPath = normalizePath(folder.Path || folder.Name || '')
  if (folderPath) {
    navigateTo(folderPath)
    return
  }

  const folderName = normalizePath(folder.Name || '')
  if (currentPath.value) {
    navigateTo(`${normalizePath(currentPath.value)}/${folderName}`)
  } else {
    navigateTo(folderName)
  }
}


const formatDate = (dateStr: string) => {
  if (!dateStr) return '--'
  const d = new Date(dateStr)
  return d.toLocaleString()
}

const formatSize = (bytes: number) => {
  if (bytes === undefined || bytes === null) return '--'
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i]
}

const isImage = (name: string) => /\.(jpg|jpeg|png|gif|webp)$/i.test(name)
const isJson = (name: string) => /\.json$/i.test(name)

onMounted(() => {
  fetchList()
})
</script>

<style scoped>
.explore-container {
  display: flex;
  flex-direction: column;
  padding: 48px 64px;
  gap: 32px;
  max-width: 1400px;
  margin: 0 auto;
}

.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.title-group h1 {
  font-family: var(--font-primary);
  font-weight: 800;
  font-size: 32px;
  color: var(--text-primary);
  letter-spacing: -0.5px;
  margin-bottom: 8px;
}

.title-group p {
  color: var(--text-secondary);
  font-size: 16px;
}

.actions {
  display: flex;
  gap: 16px;
}

/* .btn styles inherited from global style.css */

.explorer-card {
  display: flex;
  flex-direction: column;
  background-color: var(--bg-card);
  border: 1px solid var(--border-strong);
  border-radius: 16px;
  overflow: hidden;
}

.breadcrumb {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 20px 24px;
  border-bottom: 1px solid var(--border-subtle);
  background-color: #0F172A;
  color: #E2E8F0;
}

.breadcrumb .path-item {
  color: #94A3B8;
}

.breadcrumb .path-item:hover {
  color: #E2E8F0;
}

.breadcrumb .path-item.active {
  color: #F8FAFC;
  font-weight: 600;
}

.breadcrumb .separator {
  color: #475569;
}

.path-item {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
  cursor: pointer;
  transition: color 0.2s;
}

.path-item:hover {
  color: var(--text-primary);
}

.path-item.active {
  color: var(--text-primary);
  font-weight: 600;
}

.separator {
  color: var(--text-tertiary);
}

.file-table {
  display: flex;
  flex-direction: column;
}

.table-header {
  display: flex;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-strong);
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 600;
}

.table-row {
  display: flex;
  align-items: center;
  padding: 16px 24px;
  border-bottom: 1px solid var(--border-subtle);
  transition: background-color 0.2s;
  cursor: pointer;
}

.table-row:hover {
  background-color: var(--border-subtle);
}

.table-row:last-child {
  border-bottom: none;
}

.col-checkbox {
  width: 48px;
  display: flex;
  align-items: center;
}

.cb-input {
  width: 16px;
  height: 16px;
  cursor: pointer;
}

.col-name {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-primary);
  font-weight: 500;
  font-size: 14px;
}

.col-size {
  width: 120px;
  color: var(--text-secondary);
  font-size: 14px;
}

.col-date {
  width: 180px;
  color: var(--text-secondary);
  font-size: 14px;
}

.col-actions {
  width: 80px;
  display: flex;
  justify-content: flex-end;
}

.action-btn {
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

.action-btn:hover {
  background-color: var(--border-strong);
  color: var(--text-primary);
}

.text-blue {
  color: #3B82F6;
}

.text-yellow {
  color: #F59E0B;
}

.text-gray {
  color: #94A3B8;
}

.table-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 64px 24px;
  gap: 12px;
}

.empty-icon {
  color: var(--text-tertiary);
  opacity: 0.5;
}

.empty-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-secondary);
}

.empty-desc {
  font-size: 14px;
  color: var(--text-tertiary);
}

.spin {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>
