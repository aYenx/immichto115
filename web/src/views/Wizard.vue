<template>
  <div :class="['wizard-container', isSettingsMode ? 'settings-mode' : '']">
    <div class="left-col">
      <div class="title-box">
        <h1 class="main-title">ImmichTo115</h1>
        <h2 class="sub-title">{{ isSettingsMode ? '配置设置' : 'Setup Wizard' }}</h2>
      </div>

      <div class="step-list">
        <!-- Step 1 Navigation -->
        <div class="step-item">
          <div :class="['step-icon', step > 1 ? 'completed' : step === 1 ? 'active' : 'pending']">
            <LucideCheck v-if="step > 1" :size="16" />
            <span v-else>1</span>
          </div>
          <span :class="['step-text', step >= 1 ? 'active-text' : 'pending-text']">WebDAV 账号</span>
        </div>

        <!-- Step 2 Navigation -->
        <div class="step-item">
          <div :class="['step-icon', step > 2 ? 'completed' : step === 2 ? 'active' : 'pending']">
            <LucideCheck v-if="step > 2" :size="16" />
            <span v-else>2</span>
          </div>
          <span :class="['step-text', step >= 2 ? 'active-text' : 'pending-text']">备份路径</span>
        </div>

        <!-- Step 3 Navigation -->
        <div class="step-item">
          <div :class="['step-icon', step > 3 ? 'completed' : step === 3 ? 'active' : 'pending']">
            <LucideCheck v-if="step > 3" :size="16" />
            <span v-else>3</span>
          </div>
          <span :class="['step-text', step >= 3 ? 'active-text' : 'pending-text']">加密配置</span>
        </div>

        <!-- Step 4 Navigation -->
        <div class="step-item">
          <div :class="['step-icon', step > 4 ? 'completed' : step === 4 ? 'active' : 'pending']">
            <LucideCheck v-if="step > 4" :size="16" />
            <span v-else>4</span>
          </div>
          <span :class="['step-text', step >= 4 ? 'active-text' : 'pending-text']">定时任务</span>
        </div>
      </div>
    </div>

    <!-- Right Content Area -->
    <div class="right-col">
      <!-- Step 1: WebDAV Account -->
      <div v-if="step === 1" class="step-content">
        <div class="header-group">
          <h2 class="content-title">WebDAV 账号</h2>
          <p class="content-desc">连接您的配置以确保文件和数据库可以上传到云端</p>
        </div>

        <div class="form-group">
          <div class="input-field">
            <span class="input-label">服务器地址</span>
            <input class="input-control" type="text" v-model="config.webdav.url" placeholder="https://dav.example.com" />
          </div>

          <div class="input-field">
            <span class="input-label">用户名</span>
            <input class="input-control" type="text" v-model="config.webdav.user" placeholder="admin" />
          </div>

          <div class="input-field">
            <span class="input-label">密码或授权码</span>
            <input class="input-control" type="password" v-model="config.webdav.password" placeholder="••••••••••••" autocomplete="off" />
          </div>

          <div class="input-field">
            <span class="input-label">Remote Dir</span>
            <div class="path-input-row">
              <input class="input-control" type="text" v-model="config.backup.remote_dir" placeholder="/immich-backup" style="flex: 1;" />
              <button class="btn secondary browse-btn" @click="openRemoteFolderPicker">
                <LucideFolderOpen :size="16" />
                WebDAV
              </button>
            </div>
            <span class="input-hint">WebDAV 用户的根目录只是登录后的起点，真正备份会写入这里选择的云端目录。</span>
          </div>
        </div>

        <div class="buttons">
          <span v-if="testResult" :style="{ color: testSuccess ? 'var(--text-primary)' : 'red', alignSelf: 'center', marginRight: '16px' }">{{ testResult }}</span>
          <span v-if="validationError && step === 1" class="validation-error">{{ validationError }}</span>
          <button class="btn secondary" @click="testConnection" :disabled="isTesting">测试连接</button>
          <button class="btn primary" @click="nextStep">下一步</button>
        </div>
      </div>
      
      <!-- Step 2: Backup Path -->
      <div v-else-if="step === 2" class="step-content">
        <div class="header-group">
          <h2 class="content-title">备份路径</h2>
          <p class="content-desc">指定需本地备份的 Immich 照片库和数据库目录</p>
        </div>

        <div class="form-group">
          <div class="input-field">
            <span class="input-label">照片库路径 (Library Dir)</span>
            <div class="path-input-row">
              <input class="input-control" type="text" v-model="config.backup.library_dir" placeholder="/path/to/library" style="flex: 1;" />
              <button class="btn secondary browse-btn" @click="openFolderPicker('library_dir')">
                <LucideFolderOpen :size="16" />
                浏览
              </button>
            </div>
          </div>

          <div class="input-field">
            <span class="input-label">数据库备份路径 (DB Dump Dir)</span>
            <div class="path-input-row">
              <input class="input-control" type="text" v-model="config.backup.backups_dir" placeholder="/path/to/db_dumps" style="flex: 1;" />
              <button class="btn secondary browse-btn" @click="openFolderPicker('backups_dir')">
                <LucideFolderOpen :size="16" />
                浏览
              </button>
            </div>
          </div>
        </div>

        <div class="buttons space-between">
          <button class="btn secondary" @click="prevStep">上一步</button>
          <span v-if="validationError && step === 2" class="validation-error">{{ validationError }}</span>
          <button class="btn primary" @click="nextStep">下一步</button>
        </div>
      </div>
      
      <!-- Step 3: Encryption Configuration -->
      <div v-else-if="step === 3" class="step-content">
        <div class="header-group">
          <h2 class="content-title">加密配置</h2>
          <p class="content-desc">保护您的隐私数据，以防数据泄露</p>
        </div>

        <div class="form-group">
          <div class="toggle-field" @click="config.encrypt.enabled = !config.encrypt.enabled">
            <div class="toggle-info">
              <span class="toggle-title">启用加密 (Rclone Crypt)</span>
              <span class="toggle-desc">如果启用，所有的文件上传之前都会被本地加密</span>
            </div>
            <div :class="['switch', config.encrypt.enabled ? 'active' : '']">
              <div class="thumb"></div>
            </div>
          </div>

          <div class="input-field" v-if="config.encrypt.enabled">
            <span class="input-label">加密密码 (Password)</span>
            <input class="input-control" type="password" v-model="config.encrypt.password" placeholder="用于文件内容的加密" autocomplete="new-password" />
          </div>

          <div class="input-field" v-if="config.encrypt.enabled">
            <span class="input-label">加密混淆盐 (Salt)</span>
            <input class="input-control" type="password" v-model="config.encrypt.salt" placeholder="用于文件名的加密" autocomplete="new-password" />
          </div>
        </div>

        <div class="buttons space-between">
          <button class="btn secondary" @click="prevStep">上一步</button>
          <span v-if="validationError && step === 3" class="validation-error">{{ validationError }}</span>
          <button class="btn primary" @click="nextStep">下一步</button>
        </div>
      </div>
      
      <!-- Step 4: Cron Job Configuration -->
      <div v-else-if="step === 4" class="step-content">
        <div class="header-group">
          <h2 class="content-title">定时任务</h2>
          <p class="content-desc">配置自动备份的时间表</p>
        </div>

        <div class="form-group">
          <div class="toggle-field" @click="config.server.auth_enabled = !config.server.auth_enabled">
            <div class="toggle-info">
              <span class="toggle-title">启用访问保护</span>
              <span class="toggle-desc">启用后，访问管理界面和 API 需要输入管理员账号密码</span>
            </div>
            <div :class="['switch', config.server.auth_enabled ? 'active' : '']">
              <div class="thumb"></div>
            </div>
          </div>

          <div v-if="config.server.auth_enabled" class="input-field">
            <span class="input-label">管理员用户名</span>
            <input class="input-control" type="text" v-model="config.server.auth_user" placeholder="admin" />
          </div>

          <div v-if="config.server.auth_enabled" class="input-field">
            <span class="input-label">管理员密码</span>
            <input class="input-control" type="password" v-model="config.server.auth_password" placeholder="留空则保持当前密码不变" autocomplete="new-password" />
          </div>

          <div class="toggle-field" @click="config.cron.enabled = !config.cron.enabled">
            <div class="toggle-info">
              <span class="toggle-title">开启自动备份</span>
              <span class="toggle-desc">启用后将按设定的时间表自动执行备份</span>
            </div>
            <div :class="['switch', config.cron.enabled ? 'active' : '']">
              <div class="thumb"></div>
            </div>
          </div>

          <CronScheduler v-if="config.cron.enabled" v-model="config.cron.expression" />
        </div>

        <div class="buttons space-between">
          <button class="btn secondary" @click="prevStep">上一步</button>
          <button class="btn primary" @click="finishSetup" :disabled="isSaving">{{ isSaving ? '保存中...' : '完成并开始使用' }}</button>
        </div>
      </div>
    <!-- Local Folder Picker Modal -->
    <div v-if="showFolderPicker" class="modal-overlay" @click.self="showFolderPicker = false">
      <div class="modal">
        <div class="modal-header">
          <h3 style="margin: 0; font-size: 16px;">选择本地文件夹</h3>
          <button class="btn-icon" @click="showFolderPicker = false" style="background:none; border:none; cursor:pointer; color: var(--text-primary);"><LucideX :size="20" /></button>
        </div>
        <div class="modal-body">
          <!-- Breadcrumb Navigation -->
          <div class="breadcrumb-bar">
            <div class="breadcrumb-inner">
              <button class="breadcrumb-item" @click="loadLocalDir(isWindowsPath ? 'C:\\' : '/')">
                <LucideHardDrive :size="14" />
              </button>
              <template v-for="(seg, idx) in pathSegments" :key="idx">
                <span class="breadcrumb-sep">/</span>
                <button class="breadcrumb-item" @click="navigateToSegment(idx)" :class="{ last: idx === pathSegments.length - 1 }">
                  {{ seg }}
                </button>
              </template>
            </div>
            <button class="breadcrumb-edit-btn" @click="showPathInput = !showPathInput" title="手动输入路径">
              <LucidePencil :size="14" />
            </button>
          </div>
          <!-- Optional manual path input -->
          <input v-if="showPathInput" type="text" v-model="currentLocalPath" class="input-control path-manual-input" @keydown.enter="loadLocalDir(currentLocalPath); showPathInput = false" placeholder="输入路径后按 Enter" />
          <!-- Folder List -->
          <div class="folder-list">
            <div v-if="isLoadingLocal" class="folder-empty">
              <LucideLoader2 :size="20" class="spin-icon" />
              <span>加载中...</span>
            </div>
            <div v-else-if="localDirs.length === 0" class="folder-empty">
              <LucideFolderOpen :size="20" />
              <span>该目录下没有子文件夹</span>
            </div>
            <div v-else class="folder-scroll">
              <div v-if="canGoUp" class="folder-item go-up" @click="goUpLocalDir">
                <LucideCornerLeftUp :size="16" />
                <span>返回上级目录</span>
              </div>
              <div v-for="item in localDirs" :key="item.Path" class="folder-item" @click="enterLocalDir(item)">
                <LucideFolder :size="16" />
                <span>{{ item.Name }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <div class="selected-path-preview" v-if="currentLocalPath">
            <LucideCheck :size="14" />
            <span>{{ currentLocalPath }}</span>
          </div>
          <div class="modal-footer-btns">
            <button class="btn secondary" style="height: 36px; padding: 0 16px;" @click="showFolderPicker = false">取消</button>
            <button class="btn primary" style="height: 36px; padding: 0 16px;" @click="confirmFolder">选择此目录</button>
          </div>
        </div>
      </div>
    </div>
    <div v-if="showRemoteFolderPicker" class="modal-overlay" @click.self="showRemoteFolderPicker = false">
      <div class="modal">
        <div class="modal-header">
          <h3 style="margin: 0; font-size: 16px;">选择 WebDAV 备份目录</h3>
          <button class="btn-icon" @click="showRemoteFolderPicker = false" style="background:none; border:none; cursor:pointer; color: var(--text-primary);"><LucideX :size="20" /></button>
        </div>
        <div class="modal-body">
          <div class="breadcrumb-bar">
            <div class="breadcrumb-inner">
              <button class="breadcrumb-item" @click="loadRemoteDir('/')">
                <LucideFolder :size="14" />
              </button>
              <template v-for="(seg, idx) in remotePathSegments" :key="idx">
                <span class="breadcrumb-sep">/</span>
                <button class="breadcrumb-item" @click="navigateToRemoteSegment(idx)" :class="{ last: idx === remotePathSegments.length - 1 }">
                  {{ seg }}
                </button>
              </template>
            </div>
          </div>
          <div class="folder-list">
            <div v-if="isLoadingRemote" class="folder-empty">
              <LucideLoader2 :size="20" class="spin-icon" />
              <span>加载中...</span>
            </div>
            <div v-else-if="remoteDirs.length === 0" class="folder-empty">
              <LucideFolderOpen :size="20" />
              <span>该目录下没有子文件夹</span>
            </div>
            <div v-else class="folder-scroll">
              <div v-if="remoteCanGoUp" class="folder-item go-up" @click="goUpRemoteDir">
                <LucideCornerLeftUp :size="16" />
                <span>返回上级目录</span>
              </div>
              <div v-for="item in remoteDirs" :key="item.Path || item.Name" class="folder-item" @click="enterRemoteDir(item)">
                <LucideFolder :size="16" />
                <span>{{ item.Name }}</span>
              </div>
            </div>
          </div>
        </div>
        <div class="modal-footer">
          <div class="selected-path-preview">
            <LucideCheck :size="14" />
            <span>{{ currentRemotePath }}</span>
          </div>
          <div class="modal-footer-btns">
            <button class="btn secondary" style="height: 36px; padding: 0 16px;" @click="showRemoteFolderPicker = false">取消</button>
            <button class="btn primary" style="height: 36px; padding: 0 16px;" @click="confirmRemoteFolder">选择此目录</button>
          </div>
        </div>
      </div>
    </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref, computed, onMounted, reactive } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LucideCheck, LucideFolder, LucideFolderOpen, LucideX, LucideHardDrive, LucidePencil, LucideLoader2, LucideCornerLeftUp } from 'lucide-vue-next'
import { api, getErrorMessage, handleAuthFailure, type AppConfig, type DirEntry } from '../api'
import { showToast } from '../composables/toast'
import { markSetupComplete } from '../router'
import CronScheduler from '../components/CronScheduler.vue'

const router = useRouter()
const route = useRoute()
const step = ref(1)

const testResult = ref('')
const testSuccess = ref(false)
const isTesting = ref(false)
const isSaving = ref(false)
const isSettingsMode = computed(() => route.name === 'settings')

const config = reactive<AppConfig>({
  server: {
    port: 8096,
    auth_enabled: false,
    auth_user: 'admin',
    auth_password: '',
  },
  webdav: {
    url: '',
    user: '',
    password: '',
  },
  backup: {
    library_dir: '',
    backups_dir: '',
    remote_dir: '/immich-backup',
  },
  encrypt: {
    enabled: false,
    password: '',
    salt: '',
  },
  cron: {
    enabled: true,
    expression: '0 3 * * *',
  }
})

onMounted(async () => {
  try {
    const data = await api.getConfig()
    Object.assign(config, data)
  } catch (error) {
    console.warn('Could not fetch existing config, using defaults.', error)
  }
})

// ============== Folder Picker ===============
const showFolderPicker = ref(false)
const showPathInput = ref(false)
const targetLocalField = ref<'library_dir' | 'backups_dir'>('library_dir')
const currentLocalPath = ref('')
const localDirs = ref<DirEntry[]>([])
const isLoadingLocal = ref(false)
const showRemoteFolderPicker = ref(false)
const currentRemotePath = ref('/')
const remoteDirs = ref<DirEntry[]>([])
const isLoadingRemote = ref(false)
const validationError = ref('')

const isWindowsPath = computed(() => currentLocalPath.value.includes('\\'))

const pathSegments = computed(() => {
  const p = currentLocalPath.value
  if (!p) return []
  const sep = p.includes('\\') ? '\\' : '/'
  return p.split(sep).filter(s => s !== '')
})

const canGoUp = computed(() => {
  const p = currentLocalPath.value
  return p !== '/' && p !== 'C:\\' && p !== ''
})

const remotePathSegments = computed(() => currentRemotePath.value.split('/').filter(s => s !== ''))

const remoteCanGoUp = computed(() => currentRemotePath.value !== '/')

const navigateToSegment = (idx: number) => {
  const sep = isWindowsPath.value ? '\\' : '/'
  const segs = pathSegments.value.slice(0, idx + 1)
  let newPath = segs.join(sep)
  if (isWindowsPath.value) {
    // e.g. "C:" + "\\" + "Users" ...
    if (!newPath.endsWith('\\')) newPath += '\\'
  } else {
    newPath = '/' + newPath
  }
  currentLocalPath.value = newPath
  loadLocalDir(newPath)
}

const openFolderPicker = (field: 'library_dir' | 'backups_dir') => {
  targetLocalField.value = field
  showFolderPicker.value = true
  showPathInput.value = false
  currentLocalPath.value = config.backup[field] || ''
  loadLocalDir(currentLocalPath.value)
}

const normalizeRemotePath = (path: string) => {
  if (!path || path.trim() === '') return '/'
  const normalized = path.replace(/\\/g, '/').trim()
  if (normalized === '/') return '/'
  return normalized.startsWith('/') ? normalized : `/${normalized}`
}

const openRemoteFolderPicker = () => {
  showRemoteFolderPicker.value = true
  currentRemotePath.value = normalizeRemotePath(config.backup.remote_dir)
  loadRemoteDir(currentRemotePath.value)
}

const loadLocalDir = async (path: string) => {
  isLoadingLocal.value = true
  try {
    const items = await api.listLocal(path)
    localDirs.value = items.filter(i => i.IsDir).sort((a, b) => a.Name.localeCompare(b.Name))
    if (path === '') {
      currentLocalPath.value = items.length > 0 && items[0]!.Path.includes('\\') ? 'C:\\' : '/'
    }
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    alert('加载目录失败: ' + getErrorMessage(err))
  } finally {
    isLoadingLocal.value = false
  }
}

const loadRemoteDir = async (path: string) => {
  isLoadingRemote.value = true
  try {
    const normalizedPath = normalizeRemotePath(path)
    const items = await api.listWebDAV({
      url: config.webdav.url,
      user: config.webdav.user,
      password: config.webdav.password,
      path: normalizedPath,
    })
    currentRemotePath.value = normalizedPath
    remoteDirs.value = items.filter(i => i.IsDir).sort((a, b) => a.Name.localeCompare(b.Name))
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    alert('加载 WebDAV 目录失败: ' + getErrorMessage(err))
  } finally {
    isLoadingRemote.value = false
  }
}

const enterLocalDir = (item: any) => {
  const sep = currentLocalPath.value.includes('\\') ? '\\' : '/'
  let newPath = currentLocalPath.value
  if (newPath === '' || newPath.endsWith(sep)) {
    newPath += item.Name
  } else {
    newPath += sep + item.Name
  }
  currentLocalPath.value = newPath
  loadLocalDir(newPath)
}

const enterRemoteDir = (item: any) => {
  const newPath = currentRemotePath.value === '/' ? `/${item.Name}` : `${currentRemotePath.value}/${item.Name}`
  loadRemoteDir(newPath)
}

const goUpLocalDir = () => {
  const sep = currentLocalPath.value.includes('\\') ? '\\' : '/'
  let parts = currentLocalPath.value.split(sep)
  if (parts.length > 0 && parts[parts.length - 1] === '') parts.pop()
  parts.pop()
  let newPath = parts.join(sep)
  if (newPath === '' || (sep === '\\' && !newPath.includes('\\'))) newPath += sep
  currentLocalPath.value = newPath
  loadLocalDir(newPath)
}

const goUpRemoteDir = () => {
  if (currentRemotePath.value === '/') return
  const parts = currentRemotePath.value.split('/').filter(Boolean)
  parts.pop()
  const newPath = parts.length === 0 ? '/' : `/${parts.join('/')}`
  loadRemoteDir(newPath)
}

const navigateToRemoteSegment = (idx: number) => {
  const segs = remotePathSegments.value.slice(0, idx + 1)
  const newPath = segs.length === 0 ? '/' : `/${segs.join('/')}`
  loadRemoteDir(newPath)
}

const confirmFolder = () => {
  config.backup[targetLocalField.value] = currentLocalPath.value
  showFolderPicker.value = false
}

const confirmRemoteFolder = () => {
  config.backup.remote_dir = normalizeRemotePath(currentRemotePath.value)
  showRemoteFolderPicker.value = false
}

const validateCurrentStep = (): string | null => {
  if (step.value === 1) {
    if (!config.webdav.url.trim()) return '请输入 WebDAV 服务器地址'
    if (!config.webdav.user.trim()) return '请输入用户名'
    if (!config.webdav.password.trim()) return '请输入密码'
    if (!config.backup.remote_dir.trim()) return '请选择远端备份目录'
  } else if (step.value === 2) {
    if (!config.backup.library_dir.trim()) return '请输入照片库路径'
    if (!config.backup.backups_dir.trim()) return '请输入数据库备份路径'
  } else if (step.value === 3) {
    if (config.encrypt.enabled) {
      if (!config.encrypt.password.trim()) return '请输入加密密码'
      if (!config.encrypt.salt.trim()) return '请输入加密混淆盐'
    }
  }
  return null
}

const nextStep = () => {
  const error = validateCurrentStep()
  if (error) {
    validationError.value = error
    return
  }
  validationError.value = ''
  if (step.value < 4) step.value++
}

const prevStep = () => {
  validationError.value = ''
  if (step.value > 1) step.value--
}

const testConnection = async () => {
  isTesting.value = true
  testResult.value = '测试中...'
  testSuccess.value = false
  try {
    await api.testWebDAV({
      url: config.webdav.url,
      user: config.webdav.user,
      password: config.webdav.password
    })
    testSuccess.value = true
    testResult.value = '连接成功!'
    showToast('success', '连接成功', 'WebDAV 已通过测试，可以继续下一步配置。')
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    testSuccess.value = false
    testResult.value = '连接失败: ' + getErrorMessage(err)
    showToast('error', '连接失败', getErrorMessage(err))
  } finally {
    isTesting.value = false
  }
}

const finishSetup = async () => {
  isSaving.value = true
  try {
    await api.saveConfig(config)
    markSetupComplete()
    if (isSettingsMode.value) {
      showToast('success', '保存成功', '配置已保存并立即生效。')
    }
    router.replace('/dashboard')
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    showToast('error', '保存失败', getErrorMessage(err))
  } finally {
    isSaving.value = false
  }
}
</script>

<style scoped>
.wizard-container {
  display: flex;
  width: 100vw;
  height: 100vh;
  justify-content: center;
  align-items: center;
  background-color: var(--bg-primary);
  padding: 64px;
  gap: 120px;
}

.wizard-container.settings-mode {
  width: 100%;
  height: 100%;
}

.left-col {
  display: flex;
  flex-direction: column;
  gap: 64px;
  width: 360px;
}

.title-box {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.main-title {
  color: var(--text-primary);
  font-family: var(--font-primary);
  font-size: 40px;
  font-weight: 900;
  letter-spacing: -1px;
}

.sub-title {
  color: var(--text-secondary);
  font-family: var(--font-secondary);
  font-size: 20px;
  font-weight: 400;
}

.step-list {
  display: flex;
  flex-direction: column;
  gap: 32px;
}

.step-item {
  display: flex;
  align-items: center;
  gap: 16px;
}

.step-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: 16px;
  font-family: var(--font-primary);
  font-weight: 700;
  font-size: 14px;
}

.step-icon.completed {
  background-color: var(--text-primary);
  color: var(--text-inverted);
}

.step-icon.active {
  background-color: var(--bg-dark);
  color: var(--text-inverted);
}

.step-icon.pending {
  background-color: var(--bg-card);
  color: var(--text-secondary);
}

.step-text {
  font-family: var(--font-primary);
  font-size: 20px;
}

.active-text {
  color: var(--text-primary);
  font-weight: 800;
}

.pending-text {
  color: var(--text-secondary);
  font-weight: 500;
}

.right-col {
  display: flex;
  flex-direction: column;
  width: 600px;
  padding: 48px;
  background-color: var(--bg-card);
  border-radius: 24px;
  gap: 32px;
}

.step-content {
  display: flex;
  flex-direction: column;
  gap: 32px;
  width: 100%;
}

.header-group {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.content-title {
  color: var(--text-primary);
  font-family: var(--font-primary);
  font-size: 32px;
  font-weight: 800;
}

.content-desc {
  color: var(--text-secondary);
  font-family: var(--font-secondary);
  font-size: 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.input-field {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.input-label {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.input-hint {
  color: var(--text-secondary);
  font-size: 12px;
  line-height: 1.5;
}

.input-control {
  height: 48px;
  border-radius: 12px;
  border: 1px solid var(--border-strong);
  background-color: transparent;
  padding: 0 16px;
  color: var(--text-primary);
  font-size: 16px;
}

.input-control::placeholder {
  color: var(--text-tertiary);
}

.input-with-action {
  display: flex;
  height: 48px;
  border-radius: 12px;
  border: 1px solid var(--border-strong);
  padding: 0 8px 0 16px;
  align-items: center;
  justify-content: space-between;
}

.input-value {
  color: var(--text-primary);
  font-size: 16px;
}

.action-btn-small {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
  padding: 0 12px;
  border-radius: 6px;
  background-color: var(--bg-dark);
  color: var(--text-inverted);
  font-size: 12px;
  font-weight: 500;
}

.toggle-field {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border-subtle);
}

.toggle-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.toggle-title {
  color: var(--text-primary);
  font-size: 14px;
  font-weight: 600;
}

.toggle-desc {
  color: var(--text-secondary);
  font-size: 12px;
  max-width: 80%;
}

.switch {
  width: 44px;
  height: 24px;
  background-color: var(--border-strong);
  border-radius: 12px;
  position: relative;
  cursor: pointer;
  transition: all 0.2s ease;
}

.switch.active {
  background-color: var(--text-primary);
}

.thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  background-color: var(--text-inverted);
  border-radius: 10px;
  transition: all 0.2s ease;
}

.switch.active .thumb {
  left: 22px;
}

.visual-selector {
  display: flex;
  gap: 12px;
  align-items: center;
}

.visual-box {
  display: flex;
  height: 48px;
  width: 240px;
  border-radius: 12px;
  border: 1px solid var(--border-strong);
  padding: 0 16px;
  align-items: center;
  justify-content: space-between;
  color: var(--text-primary);
  font-size: 14px;
  cursor: pointer;
}

.visual-box.small {
  width: 160px;
}

.advanced-toggle {
  display: flex;
  gap: 8px;
  align-items: center;
  padding-top: 8px;
  color: var(--text-secondary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.buttons {
  display: flex;
  gap: 16px;
  padding-top: 16px;
  justify-content: flex-end;
}

.buttons.space-between {
  justify-content: space-between;
}

/* .btn base styles inherited from global style.css */
.btn.primary {
  height: 48px;
  border-radius: 12px;
  font-size: 16px;
  padding: 0 32px;
}

.btn.secondary {
  height: 48px;
  border-radius: 12px;
  font-size: 16px;
  padding: 0 24px;
}

.validation-error {
  color: #EF4444;
  font-size: 13px;
  font-weight: 500;
  align-self: center;
  margin-right: auto;
}

.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal {
  background: var(--bg-card);
  width: 520px;
  border-radius: 16px;
  padding: 24px;
  border: 1px solid var(--border-strong);
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
  color: var(--text-primary);
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.modal-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.modal-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.modal-footer {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.modal-footer-btns {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.selected-path-preview {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background-color: var(--bg-primary);
  border-radius: 8px;
  color: var(--text-secondary);
  font-size: 12px;
  font-family: 'Consolas', 'Monaco', monospace;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Breadcrumb Bar */
.breadcrumb-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  background-color: var(--bg-primary);
  border-radius: 10px;
  border: 1px solid var(--border-strong);
  min-height: 40px;
}

.breadcrumb-inner {
  display: flex;
  align-items: center;
  gap: 4px;
  flex: 1;
  overflow-x: auto;
  white-space: nowrap;
}

.breadcrumb-inner::-webkit-scrollbar {
  display: none;
}

.breadcrumb-item {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
  padding: 4px 8px;
  border-radius: 6px;
  transition: all 0.15s ease;
  cursor: pointer;
  display: flex;
  align-items: center;
  flex-shrink: 0;
}

.breadcrumb-item:hover {
  background-color: var(--border-subtle);
  color: var(--text-primary);
}

.breadcrumb-item.last {
  color: var(--text-primary);
  font-weight: 600;
}

.breadcrumb-sep {
  color: var(--text-tertiary);
  font-size: 12px;
  flex-shrink: 0;
}

.breadcrumb-edit-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  color: var(--text-tertiary);
  transition: all 0.15s ease;
  cursor: pointer;
  flex-shrink: 0;
}

.breadcrumb-edit-btn:hover {
  background-color: var(--border-subtle);
  color: var(--text-primary);
}

.path-manual-input {
  width: 100%;
  height: 36px !important;
  font-size: 13px !important;
  font-family: 'Consolas', 'Monaco', monospace;
}

/* Folder List */
.folder-list {
  max-height: 280px;
  overflow-y: auto;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
}

.folder-scroll {
  display: flex;
  flex-direction: column;
}

.folder-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 14px;
  cursor: pointer;
  color: var(--text-primary);
  font-size: 14px;
  transition: background-color 0.12s ease;
  border-bottom: 1px solid var(--border-subtle);
}

.folder-item:last-child {
  border-bottom: none;
}

.folder-item:hover {
  background-color: var(--border-subtle);
}

.folder-item.go-up {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
}

.folder-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 32px 16px;
  color: var(--text-tertiary);
  font-size: 14px;
}

/* Browse button in path input row */
.path-input-row {
  display: flex;
  gap: 8px;
}

.browse-btn {
  padding: 0 16px !important;
  gap: 6px;
  white-space: nowrap;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.spin-icon {
  animation: spin 1s linear infinite;
}
</style>
