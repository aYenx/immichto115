<template>
  <div class="explore-container">
    <div class="header">
      <div class="title-group">
        <h1>恢复照片</h1>
        <p>浏览并恢复云端备份的文件数据</p>
      </div>
      <div class="actions">
        <button class="btn secondary" @click="fetchList" :disabled="isLoading">
          <LucideRefreshCw :size="16" :class="{ 'spin': isLoading }" />
          刷新列表
        </button>
        <button class="btn primary" :disabled="selectedFiles.length === 0" @click="batchRestore">
          <LucideDownloadCloud :size="16" />
          批量恢复 ({{ selectedFiles.length }})
        </button>
      </div>
    </div>

    <div class="explorer-card">
      <div class="breadcrumb">
        <div class="path-item" @click="navigateTo('')" :class="{ active: currentPath === '' }">
          <LucideCloud :size="16" />
          <span>根目录</span>
        </div>
        <template v-for="(segment, idx) in breadcrumbs" :key="idx">
          <LucideChevronRight :size="16" class="separator" />
          <div class="path-item" @click="navigateTo(segment.path)" :class="{ active: idx === breadcrumbs.length - 1 }">
            <span>{{ segment.name }}</span>
          </div>
        </template>
      </div>

      <div class="file-table">
        <div class="table-header">
          <div class="col-checkbox">
            <input type="checkbox" class="cb-input" />
          </div>
          <div class="col-name">名称</div>
          <div class="col-size">大小</div>
          <div class="col-date">修改日期</div>
          <div class="col-actions">操作</div>
        </div>

        <div class="table-body">
          <div v-if="isLoading" class="table-empty">
            加载中...
          </div>
          <div v-else-if="items.length === 0" class="table-empty">
            无文件/文件夹
          </div>
          
          <template v-else>
            <div class="table-row folder" v-for="folder in folders" :key="folder.Name" @click="navigateToFolder(folder.Name)">
              <div class="col-checkbox" @click.stop>
                <input type="checkbox" class="cb-input" />
              </div>
              <div class="col-name">
                <LucideFolder :size="20" class="icon text-blue" />
                <span>{{ folder.Name }}</span>
              </div>
              <div class="col-size">--</div>
              <div class="col-date">{{ formatDate(folder.ModTime) }}</div>
              <div class="col-actions">
                <button class="action-btn"><LucideChevronRight :size="18" /></button>
              </div>
            </div>

            <div class="table-row file" v-for="file in files" :key="file.Name">
              <div class="col-checkbox">
                <input type="checkbox" class="cb-input" v-model="selectedFiles" :value="file.Name" />
              </div>
              <div class="col-name">
                <LucideImage :size="20" class="icon text-gray" v-if="isImage(file.Name)" />
                <LucideFileJson :size="20" class="icon text-yellow" v-else-if="isJson(file.Name)" />
                <LucideFile :size="20" class="icon text-gray" v-else />
                <span>{{ file.Name }}</span>
              </div>
              <div class="col-size">{{ formatSize(file.Size) }}</div>
              <div class="col-date">{{ formatDate(file.ModTime) }}</div>
              <div class="col-actions">
                <button class="action-btn" @click.stop="downloadFile(file)"><LucideDownload :size="18" /></button>
              </div>
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
  LucideDownloadCloud, 
  LucideCloud, 
  LucideChevronRight, 
  LucideFolder, 
  LucideImage, 
  LucideFile,
  LucideFileJson,
  LucideDownload
} from 'lucide-vue-next'
import { api } from '../api'

const selectedFiles = ref<string[]>([])
const currentPath = ref('')
const isLoading = ref(false)
const items = ref<any[]>([])

const breadcrumbs = computed(() => {
  if (!currentPath.value) return []
  const parts = currentPath.value.split('/')
  return parts.map((part, index) => {
    return {
      name: part,
      path: parts.slice(0, index + 1).join('/')
    }
  })
})

const folders = computed(() => items.value.filter(i => i.IsDir).sort((a,b)=>a.Name.localeCompare(b.Name)))
const files = computed(() => items.value.filter(i => !i.IsDir).sort((a,b)=>a.Name.localeCompare(b.Name)))

const fetchList = async () => {
  isLoading.value = true
  selectedFiles.value = []
  try {
    const data = await api.listRemote('/' + currentPath.value)
    items.value = Array.isArray(data) ? data : []
  } catch (err: any) {
    alert('请求文件列表失败: ' + err.message)
    items.value = []
  } finally {
    isLoading.value = false
  }
}

const navigateTo = (path: string) => {
  currentPath.value = path
  fetchList()
}

const navigateToFolder = (folderName: string) => {
  if (currentPath.value) {
    navigateTo(`${currentPath.value}/${folderName}`)
  } else {
    navigateTo(folderName)
  }
}

const downloadFile = (file: any) => {
  alert(`后端尚未提供单独文件的直接下载接口，文件: ${file.Name}`)
}

const batchRestore = () => {
  alert(`尝试恢复选中的 ${selectedFiles.value.length} 个文件。后端暂无处理该请求的接口。`)
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

.btn {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 20px;
  border-radius: 8px;
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn.primary {
  background-color: var(--text-primary);
  color: var(--text-inverted);
}

.btn.primary:not(:disabled):hover {
  opacity: 0.9;
}

.btn.secondary {
  background-color: var(--bg-card);
  border: 1px solid var(--border-strong);
  color: var(--text-primary);
}

.btn.secondary:not(:disabled):hover {
  background-color: var(--border-subtle);
}

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
  background-color: var(--bg-dark);
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
</style>
