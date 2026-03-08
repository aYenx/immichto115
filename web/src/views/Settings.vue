<template>
  <div class="settings-page">
    <div class="settings-hero">
      <div>
        <p class="settings-eyebrow">Settings</p>
        <h1 class="settings-title">配置中心</h1>
        <p class="settings-subtitle">把常用配置拆成四个模块，按需打开弹窗编辑，避免沿用首次引导流程。</p>
        <div class="settings-status-strip">
          <span :class="['status-chip', systemStatus?.rclone_installed ? 'healthy' : 'warning']">
            {{ systemStatus?.rclone_installed ? 'Rclone 已就绪' : 'Rclone 未安装' }}
          </span>
          <span :class="['status-chip', systemStatus?.backup_status === 'running' ? 'info' : 'neutral']">
            {{ systemStatus?.backup_status === 'running' ? '备份进行中' : '当前空闲' }}
          </span>
          <span :class="['status-chip', systemStatus?.cron_enabled ? 'healthy' : 'warning']">
            {{ systemStatus?.cron_enabled ? '自动备份已开启' : '自动备份未开启' }}
          </span>
        </div>
      </div>
      <button class="btn secondary" @click="refreshConfig" :disabled="isRefreshing">
        {{ isRefreshing ? '刷新中...' : '刷新配置' }}
      </button>
    </div>

    <div class="settings-grid">
      <button class="settings-card" @click="openSection('webdav')">
        <div class="settings-card-icon webdav">
          <LucideGlobe :size="28" />
        </div>
        <div class="settings-card-body">
          <div class="settings-card-head">
            <div>
              <h2>WebDAV 与远端目录</h2>
              <span>连接配置</span>
            </div>
            <span :class="['card-badge', webdavCardState.tone]">{{ webdavCardState.label }}</span>
          </div>
          <p>{{ webdavSummary }}</p>
          <ul class="settings-card-signals">
            <li v-for="signal in webdavSignals" :key="signal">{{ signal }}</li>
          </ul>
        </div>
      </button>

      <button class="settings-card" @click="openSection('backup')">
        <div class="settings-card-icon backup">
          <LucideFolder :size="28" />
        </div>
        <div class="settings-card-body">
          <div class="settings-card-head">
            <div>
              <h2>备份路径</h2>
              <span>本地数据源</span>
            </div>
            <span :class="['card-badge', backupCardState.tone]">{{ backupCardState.label }}</span>
          </div>
          <p>{{ backupSummary }}</p>
          <ul class="settings-card-signals">
            <li v-for="signal in backupSignals" :key="signal">{{ signal }}</li>
          </ul>
        </div>
      </button>

      <button class="settings-card" @click="openSection('encrypt')">
        <div class="settings-card-icon encrypt">
          <LucideLock :size="28" />
        </div>
        <div class="settings-card-body">
          <div class="settings-card-head">
            <div>
              <h2>加密配置</h2>
              <span>传输保护</span>
            </div>
            <span :class="['card-badge', encryptCardState.tone]">{{ encryptCardState.label }}</span>
          </div>
          <p>{{ encryptSummary }}</p>
          <ul class="settings-card-signals">
            <li v-for="signal in encryptSignals" :key="signal">{{ signal }}</li>
          </ul>
        </div>
      </button>

      <button class="settings-card" @click="openSection('automation')">
        <div class="settings-card-icon automation">
          <LucideClock :size="28" />
        </div>
        <div class="settings-card-body">
          <div class="settings-card-head">
            <div>
              <h2>定时任务与访问保护</h2>
              <span>自动化与安全</span>
            </div>
            <span :class="['card-badge', automationCardState.tone]">{{ automationCardState.label }}</span>
          </div>
          <p>{{ automationSummary }}</p>
          <ul class="settings-card-signals">
            <li v-for="signal in automationSignals" :key="signal">{{ signal }}</li>
          </ul>
        </div>
      </button>

      <button class="settings-card" @click="openSection('notify')">
        <div class="settings-card-icon notify">
          <LucideBell :size="28" />
        </div>
        <div class="settings-card-body">
          <div class="settings-card-head">
            <div>
              <h2>推送通知</h2>
              <span>Bark 通知</span>
            </div>
            <span :class="['card-badge', config.notify.enabled ? 'good' : 'neutral']">{{ config.notify.enabled ? '已启用' : '未启用' }}</span>
          </div>
          <p>{{ config.notify.enabled ? '备份成功/失败时推送通知到手机' : '启用后可接收备份结果推送' }}</p>
        </div>
      </button>
    </div>

    <div v-if="activeSection" class="settings-modal-overlay" @click.self="closeSection">
      <div class="settings-modal-card">
        <div class="settings-modal-header">
          <div>
            <p class="settings-modal-kicker">{{ sectionMeta[activeSection].kicker }}</p>
            <h3>{{ sectionMeta[activeSection].title }}</h3>
            <p>{{ sectionMeta[activeSection].description }}</p>
          </div>
          <button class="settings-close" @click="closeSection"><LucideX :size="20" /></button>
        </div>

        <div class="settings-modal-body">
          <template v-if="activeSection === 'webdav'">
            <div class="input-field">
              <span class="input-label">接入方式</span>
              <div class="radio-group">
                <label class="radio-option" :class="{ active: draftConfig.provider === 'webdav' }">
                  <input type="radio" v-model="draftConfig.provider" value="webdav" />
                  <div class="radio-option-text">
                    <strong>WebDAV</strong>
                    <span>继续使用现有的 WebDAV + rclone 模式</span>
                  </div>
                </label>
                <label class="radio-option" :class="{ active: draftConfig.provider === 'open115' }">
                  <input type="radio" v-model="draftConfig.provider" value="open115" />
                  <div class="radio-option-text">
                    <strong>115 Open</strong>
                    <span>通过二维码授权，后续走 115 Open API</span>
                  </div>
                </label>
              </div>
            </div>

            <template v-if="draftConfig.provider === 'webdav'">
              <div class="input-field">
                <span class="input-label">服务器地址</span>
                <input class="input-control" v-model="draftConfig.webdav.url" type="text" placeholder="请输入 WebDAV 地址，例如 https://dav.example.com" />
              </div>

              <div class="input-field">
                <span class="input-label">用户名</span>
                <input class="input-control" v-model="draftConfig.webdav.user" type="text" placeholder="请输入 WebDAV 用户名" />
              </div>

              <div class="input-field">
                <span class="input-label">密码或授权码</span>
                <input class="input-control" v-model="draftConfig.webdav.password" type="password" placeholder="••••••••••••" autocomplete="off" />
              </div>

              <div class="input-field">
                <span class="input-label">远端目录</span>
                <div class="path-input-row">
                  <button class="btn secondary browse-btn" @click="openRemoteFolderPicker">
                    <LucideFolderOpen :size="16" />
                    WebDAV
                  </button>
                </div>
                <span class="input-hint">备份会写入这里指定的 WebDAV 目录。</span>
              </div>
            </template>

            <template v-else>
              <div class="input-field">
                <span class="input-label">Client ID</span>
                <input class="input-control" v-model="draftConfig.open115.client_id" type="text" placeholder="请输入 115 Open Client ID" />
              </div>

              <div class="input-field">
                <span class="input-label">远端目录</span>
                <input class="input-control" v-model="draftConfig.backup.remote_dir" type="text" placeholder="例如 /immich-backup（逻辑目录）" />
                <span class="input-hint">Open115 目录浏览后续补充，当前先直接填写逻辑目录。</span>
              </div>

              <div class="input-field" v-if="draftConfig.open115.user_id">
                <span class="input-label">当前授权用户</span>
                <span class="input-hint">User ID: {{ draftConfig.open115.user_id }}</span>
              </div>
            </template>

            <div class="settings-inline-actions">
              <template v-if="draftConfig.provider === 'webdav'">
                <button class="btn secondary" @click="testConnection" :disabled="isTesting">
                  {{ isTesting ? '测试中...' : '测试连接' }}
                </button>
              </template>
              <template v-else>
                <button class="btn secondary" @click="startOpen115Auth" :disabled="isOpen115AuthLoading">
                  {{ isOpen115AuthLoading ? '生成中...' : '开始扫码授权' }}
                </button>
                <button class="btn secondary" @click="finishOpen115Auth" :disabled="isOpen115Finishing || !open115Auth.uid || open115Authorized !== true">
                  {{ isOpen115Finishing ? '确认中...' : '完成授权' }}
                </button>
                <button class="btn secondary" @click="testConnection" :disabled="isTesting">
                  {{ isTesting ? '测试中...' : '测试连接' }}
                </button>
              </template>
              <span v-if="testResult" :class="['settings-inline-message', testSuccess ? 'success' : 'error']">
                {{ testResult }}
              </span>
            </div>

            <div v-if="draftConfig.provider === 'open115' && open115Auth.qrcode" class="qrcode-panel">
              <p class="input-label">扫码二维码</p>
              <img :src="open115Auth.qrcode" alt="115 QR Code" class="qrcode-image" />
              <p class="input-hint">请使用 115 App 扫码并确认授权。</p>
              <p class="input-hint">当前状态：{{ open115AuthStatusText }}</p>
            </div>
          </template>

          <template v-else-if="activeSection === 'backup'">
            <div class="input-field">
              <span class="input-label">照片库路径</span>
              <div class="path-input-row">
                <input class="input-control" v-model="draftConfig.backup.library_dir" type="text" placeholder="例如 /data/library 或 D:\\Immich\\library" style="flex: 1" />
                <button class="btn secondary browse-btn" @click="openFolderPicker('library_dir')">
                  <LucideFolderOpen :size="16" />
                  浏览
                </button>
              </div>
            </div>

            <div class="input-field">
              <span class="input-label">数据库备份路径</span>
              <div class="path-input-row">
                <input class="input-control" v-model="draftConfig.backup.backups_dir" type="text" placeholder="例如 /data/backups 或 D:\\Immich\\backups" style="flex: 1" />
                <button class="btn secondary browse-btn" @click="openFolderPicker('backups_dir')">
                  <LucideFolderOpen :size="16" />
                  浏览
                </button>
              </div>
            </div>

            <div class="input-field">
              <span class="input-label">备份模式</span>
              <div class="radio-group">
                <label class="radio-option" :class="{ active: draftConfig.backup.mode === 'copy' }">
                  <input type="radio" v-model="draftConfig.backup.mode" value="copy" />
                  <div class="radio-option-text">
                    <strong>增量备份 (copy)</strong>
                    <span>只复制新增/修改的文件，不删除远端已有文件</span>
                  </div>
                </label>
                <label class="radio-option" :class="{ active: draftConfig.backup.mode === 'sync' }">
                  <input type="radio" v-model="draftConfig.backup.mode" value="sync" />
                  <div class="radio-option-text">
                    <strong>镜像同步 (sync)</strong>
                    <span>保持远端与本地完全一致，会删除远端多余文件</span>
                  </div>
                </label>
              </div>
            </div>
          </template>

          <template v-else-if="activeSection === 'encrypt'">
            <div class="toggle-field" @click="draftConfig.encrypt.enabled = !draftConfig.encrypt.enabled">
              <div class="toggle-info">
                <span class="toggle-title">启用加密 (Rclone Crypt)</span>
                <span class="toggle-desc">开启后，上传前会在本地对文件内容和文件名进行加密。</span>
              </div>
              <div :class="['switch', draftConfig.encrypt.enabled ? 'active' : '']">
                <div class="thumb"></div>
              </div>
            </div>

            <div v-if="draftConfig.encrypt.enabled" class="input-field">
              <span class="input-label">加密密码</span>
              <input class="input-control" v-model="draftConfig.encrypt.password" type="password" placeholder="用于文件内容的加密" autocomplete="new-password" />
            </div>

            <div v-if="draftConfig.encrypt.enabled" class="input-field">
              <span class="input-label">加密混淆盐</span>
              <input class="input-control" v-model="draftConfig.encrypt.salt" type="password" placeholder="用于文件名的加密" autocomplete="new-password" />
            </div>
          </template>

          <template v-else-if="activeSection === 'automation'">
            <div class="toggle-field" @click="draftConfig.server.auth_enabled = !draftConfig.server.auth_enabled">
              <div class="toggle-info">
                <span class="toggle-title">启用访问保护</span>
                <span class="toggle-desc">开启后，管理页面、接口和实时日志都会受管理员账号密码保护；保存后会立即重新验证身份。</span>
              </div>
              <div :class="['switch', draftConfig.server.auth_enabled ? 'active' : '']">
                <div class="thumb"></div>
              </div>
            </div>

            <div v-if="draftConfig.server.auth_enabled" class="input-field">
              <span class="input-label">管理员用户名</span>
              <input class="input-control" v-model="draftConfig.server.auth_user" type="text" placeholder="例如：admin" />
            </div>

            <div v-if="draftConfig.server.auth_enabled" class="input-field">
              <span class="input-label">管理员密码</span>
              <input class="input-control" v-model="draftConfig.server.auth_password" type="password" placeholder="留空则保持当前密码不变" autocomplete="new-password" />
            </div>

            <div class="toggle-field" @click="draftConfig.cron.enabled = !draftConfig.cron.enabled">
              <div class="toggle-info">
                <span class="toggle-title">开启自动备份</span>
                <span class="toggle-desc">启用后将按设定的时间表自动执行备份。</span>
              </div>
              <div :class="['switch', draftConfig.cron.enabled ? 'active' : '']">
                <div class="thumb"></div>
              </div>
            </div>

            <CronScheduler v-if="draftConfig.cron.enabled" v-model="draftConfig.cron.expression" />
          </template>

          <template v-else-if="activeSection === 'notify'">
            <div class="toggle-field" @click="draftConfig.notify.enabled = !draftConfig.notify.enabled">
              <div class="toggle-info">
                <span class="toggle-title">启用 Bark 推送通知</span>
                <span class="toggle-desc">备份成功或失败时，通过 Bark 推送通知到你的 iPhone。</span>
              </div>
              <div :class="['switch', draftConfig.notify.enabled ? 'active' : '']">
                <div class="thumb"></div>
              </div>
            </div>

            <div v-if="draftConfig.notify.enabled" class="input-field">
              <span class="input-label">Bark 推送地址</span>
              <input class="input-control" v-model="draftConfig.notify.bark_url" type="text" placeholder="https://api.day.app/YOUR_DEVICE_KEY" />
              <span class="input-hint">在 Bark App 中复制推送地址，格式如 https://api.day.app/xxxxxx</span>
            </div>

            <div v-if="draftConfig.notify.enabled && draftConfig.notify.bark_url" class="notify-test-row">
              <button class="btn secondary" @click="testNotify" :disabled="isTestingNotify">
                <LucideSend :size="16" />
                {{ isTestingNotify ? '发送中...' : '发送测试通知' }}
              </button>
            </div>
          </template>

          <p v-if="validationError" class="validation-error">{{ validationError }}</p>
        </div>

        <div class="settings-modal-footer">
          <button class="btn secondary" @click="closeSection">取消</button>
          <button class="btn primary" @click="saveSection" :disabled="isSaving">
            {{ isSaving ? '保存中...' : '保存设置' }}
          </button>
        </div>
      </div>
    </div>

    <div v-if="showFolderPicker" class="picker-overlay" @click.self="showFolderPicker = false">
      <div class="picker-modal">
        <div class="picker-header">
          <h3>选择本地文件夹</h3>
          <button class="settings-close" @click="showFolderPicker = false"><LucideX :size="20" /></button>
        </div>
        <div class="picker-body">
          <div class="breadcrumb-bar">
            <div class="breadcrumb-inner">
              <button class="breadcrumb-item" @click="loadLocalDir(isWindowsPath ? 'C:\\' : '/')">
                <LucideHardDrive :size="14" />
              </button>
              <template v-for="(seg, idx) in pathSegments" :key="idx">
                <span class="breadcrumb-sep">/</span>
                <button class="breadcrumb-item" @click="navigateToSegment(idx)">{{ seg }}</button>
              </template>
            </div>
            <button class="breadcrumb-edit-btn" @click="showPathInput = !showPathInput" title="手动输入路径">
              <LucidePencil :size="14" />
            </button>
          </div>

          <input
            v-if="showPathInput"
            v-model="currentLocalPath"
            type="text"
            class="input-control path-manual-input"
            placeholder="输入路径后按 Enter"
            @keydown.enter="loadLocalDir(currentLocalPath); showPathInput = false"
          />

          <div class="folder-list">
            <div v-if="isLoadingLocal" class="folder-empty">
              <LucideLoader2 :size="20" class="spin-icon" />
              <span>加载中...</span>
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
              <div v-if="localDirs.length === 0" class="folder-empty">
                <LucideFolderOpen :size="20" />
                <span>该目录下没有子文件夹</span>
              </div>
            </div>
          </div>
        </div>
        <div class="picker-footer">
          <div class="selected-path-preview" v-if="currentLocalPath">
            <LucideCheck :size="14" />
            <span>{{ currentLocalPath }}</span>
          </div>
          <div class="picker-footer-actions">
            <button class="btn secondary" @click="showFolderPicker = false">取消</button>
            <button class="btn primary" @click="confirmFolder">选择此目录</button>
          </div>
        </div>
      </div>
    </div>

    <div v-if="showRemoteFolderPicker" class="picker-overlay" @click.self="showRemoteFolderPicker = false">
      <div class="picker-modal">
        <div class="picker-header">
          <h3>选择 WebDAV 备份目录</h3>
          <button class="settings-close" @click="showRemoteFolderPicker = false"><LucideX :size="20" /></button>
        </div>
        <div class="picker-body">
          <div class="breadcrumb-bar">
            <div class="breadcrumb-inner">
              <button class="breadcrumb-item" @click="loadRemoteDir('/')">
                <LucideFolder :size="14" />
              </button>
              <template v-for="(seg, idx) in remotePathSegments" :key="idx">
                <span class="breadcrumb-sep">/</span>
                <button class="breadcrumb-item" @click="navigateToRemoteSegment(idx)">{{ seg }}</button>
              </template>
            </div>
          </div>

          <div class="folder-list">
            <div v-if="isLoadingRemote" class="folder-empty">
              <LucideLoader2 :size="20" class="spin-icon" />
              <span>加载中...</span>
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
              <div v-if="remoteDirs.length === 0" class="folder-empty">
                <LucideFolderOpen :size="20" />
                <span>该目录下没有子文件夹</span>
              </div>
            </div>
          </div>
        </div>
        <div class="picker-footer">
          <div class="selected-path-preview">
            <LucideCheck :size="14" />
            <span>{{ currentRemotePath }}</span>
          </div>
          <div class="picker-footer-actions">
            <button class="btn secondary" @click="showRemoteFolderPicker = false">取消</button>
            <button class="btn primary" @click="confirmRemoteFolder">选择此目录</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue'
import { api, getErrorMessage, handleAuthFailure, type AppConfig, type DirEntry, type SystemStatus, type Open115AuthStartResponse } from '../api'
import CronScheduler from '../components/CronScheduler.vue'
import { showToast } from '../composables/toast'
import {
  LucideGlobe,
  LucideFolder,
  LucideLock,
  LucideClock,
  LucideFolderOpen,
  LucideLoader2,
  LucideCornerLeftUp,
  LucideX,
  LucideHardDrive,
  LucidePencil,
  LucideCheck,
  LucideBell,
  LucideSend
} from 'lucide-vue-next'

type SectionKey = 'webdav' | 'backup' | 'encrypt' | 'automation' | 'notify'
type LocalField = 'library_dir' | 'backups_dir'

import { createDefaultConfig } from '../configDefaults'

const cloneConfig = (value: AppConfig): AppConfig => JSON.parse(JSON.stringify(value)) as AppConfig

const config = reactive<AppConfig>(createDefaultConfig())
const draftConfig = ref<AppConfig>(createDefaultConfig())
const systemStatus = ref<SystemStatus | null>(null)

const activeSection = ref<SectionKey | null>(null)
const isRefreshing = ref(false)
const isSaving = ref(false)
const isTesting = ref(false)
const isTestingNotify = ref(false)
const isOpen115AuthLoading = ref(false)
const isOpen115Finishing = ref(false)
const testResult = ref('')
const testSuccess = ref(false)
const validationError = ref('')
const open115Auth = reactive<Open115AuthStartResponse>({ uid: '', time: 0, sign: '', qrcode: '', created_at: '' })
const open115AuthStatusText = ref('未开始')
const open115Authorized = ref<boolean | null>(null)
let authPollTimer: number | null = null

const sectionMeta: Record<SectionKey, { title: string; kicker: string; description: string }> = {
  webdav: {
    title: 'WebDAV 与远端目录',
    kicker: '连接配置',
    description: '管理 WebDAV 连接信息和云端备份目录。',
  },
  backup: {
    title: '备份路径',
    kicker: '本地数据源',
    description: '指定需要同步的照片库目录和数据库备份目录。',
  },
  encrypt: {
    title: '加密配置',
    kicker: '传输保护',
    description: '控制是否启用 Rclone Crypt 以及对应密钥。',
  },
  automation: {
    title: '定时任务与访问保护',
    kicker: '自动化与安全',
    description: '管理自动备份计划，以及后台的访问鉴权。',
  },
  notify: {
    title: '推送通知',
    kicker: 'Bark 通知',
    description: '备份完成或失败时，通过 Bark 推送通知到手机。',
  },
}

const refreshConfig = async () => {
  isRefreshing.value = true
  try {
    const [data, status] = await Promise.all([api.getConfig(), api.getSystemStatus()])
    Object.assign(config, data)
    systemStatus.value = status
  } catch (error) {
    if (handleAuthFailure(error)) return
    showToast('error', '刷新失败', getErrorMessage(error))
  } finally {
    isRefreshing.value = false
  }
}

const stopAuthPolling = () => {
  if (authPollTimer != null) {
    window.clearInterval(authPollTimer)
    authPollTimer = null
  }
}

const pollOpen115Auth = async () => {
  if (!open115Auth.uid) return
  try {
    const status = await api.open115AuthStatus(open115Auth.uid)
    open115AuthStatusText.value = status.message || `status=${status.status}`
    open115Authorized.value = status.authorized
    if (status.authorized) {
      stopAuthPolling()
      showToast('success', '扫码已确认', '已收到 115 授权确认，请点击“完成授权”。')
    }
  } catch (error) {
    if (handleAuthFailure(error)) return
    open115AuthStatusText.value = '状态查询失败：' + getErrorMessage(error)
    stopAuthPolling()
  }
}

const startOpen115Auth = async () => {
  if (!draftConfig.value.open115.client_id.trim()) {
    validationError.value = '请输入 115 Open Client ID'
    return
  }
  validationError.value = ''
  isOpen115AuthLoading.value = true
  open115Authorized.value = null
  open115AuthStatusText.value = '正在生成二维码...'
  try {
    const result = await api.open115AuthStart({ client_id: draftConfig.value.open115.client_id.trim() })
    Object.assign(open115Auth, result)
    open115AuthStatusText.value = '等待扫码'
    stopAuthPolling()
    authPollTimer = window.setInterval(() => {
      void pollOpen115Auth()
    }, 2500)
  } catch (error) {
    if (handleAuthFailure(error)) return
    showToast('error', '启动扫码失败', getErrorMessage(error))
    open115AuthStatusText.value = '启动失败'
  } finally {
    isOpen115AuthLoading.value = false
  }
}

const finishOpen115Auth = async () => {
  if (!open115Auth.uid) {
    validationError.value = '请先开始扫码授权'
    return
  }
  isOpen115Finishing.value = true
  try {
    const result = await api.open115AuthFinish({ uid: open115Auth.uid })
    draftConfig.value.open115 = { ...draftConfig.value.open115, ...result.state }
    open115Authorized.value = true
    open115AuthStatusText.value = '授权完成'
    showToast('success', '授权成功', '115 Open token 已保存，可以点击保存设置。')
  } catch (error) {
    if (handleAuthFailure(error)) return
    showToast('error', '完成授权失败', getErrorMessage(error))
  } finally {
    isOpen115Finishing.value = false
  }
}

onMounted(async () => {
  await refreshConfig()
})

onUnmounted(() => stopAuthPolling())

const openSection = (section: SectionKey) => {
  activeSection.value = section
  draftConfig.value = cloneConfig(config)
  validationError.value = ''
  testResult.value = ''
  testSuccess.value = false
}

const closeSection = () => {
  activeSection.value = null
  validationError.value = ''
  testResult.value = ''
}

const webdavSummary = computed(() => {
  if (config.provider === 'open115') {
    const user = config.open115.user_id.trim() || '未授权'
    const remoteDir = config.backup.remote_dir.trim() || '/'
    return `115 Open · 用户 ${user} · 远端目录 ${remoteDir}`
  }
  const url = config.webdav.url.trim() || '未配置服务器地址'
  const remoteDir = config.backup.remote_dir.trim() || '/'
  return `${url} · 远端目录 ${remoteDir}`
})

const backupSummary = computed(() => {
  const parts: string[] = []
  if (config.backup.library_dir.trim()) parts.push('照片库已配置')
  if (config.backup.backups_dir.trim()) parts.push('数据库备份已配置')
  return parts.length > 0 ? parts.join(' · ') : '尚未设置任何本地备份目录'
})

const encryptSummary = computed(() => {
  return config.encrypt.enabled ? '已启用加密传输与文件名混淆' : '当前未启用 Rclone Crypt 加密'
})

const automationSummary = computed(() => {
  const auth = config.server.auth_enabled ? '访问保护已开启' : '访问保护未开启'
  const cron = config.cron.enabled ? `自动备份: ${config.cron.expression}` : '自动备份未开启'
  return `${auth} · ${cron}`
})

const createCardState = (label: string, tone: 'healthy' | 'warning' | 'info' | 'neutral') => ({ label, tone })

const webdavCardState = computed(() => {
  if (config.provider === 'open115') {
    if (!config.open115.access_token.trim() || !config.open115.refresh_token.trim()) return createCardState('待授权', 'warning')
    return createCardState('Open115 已连接', 'healthy')
  }
  if (!config.webdav.url.trim() || !config.webdav.user.trim()) return createCardState('待配置', 'warning')
  if (systemStatus.value && !systemStatus.value.rclone_installed) return createCardState('需修复', 'warning')
  return createCardState('已连接', 'healthy')
})

const backupCardState = computed(() => {
  if (!config.backup.library_dir.trim() && !config.backup.backups_dir.trim()) return createCardState('待配置', 'warning')
  if (!config.backup.library_dir.trim() || !config.backup.backups_dir.trim()) return createCardState('待完善', 'warning')
  return createCardState('已配置', 'healthy')
})

const encryptCardState = computed(() => {
  if (!config.encrypt.enabled) return createCardState('未启用', 'neutral')
  if (!config.encrypt.password.trim() || !config.encrypt.salt.trim()) return createCardState('需补全', 'warning')
  return createCardState('已保护', 'healthy')
})

const automationCardState = computed(() => {
  if (systemStatus.value?.backup_status === 'running') return createCardState('运行中', 'info')
  if (!config.cron.enabled && !config.server.auth_enabled) return createCardState('基础模式', 'neutral')
  if (config.cron.enabled && systemStatus.value?.cron_enabled) return createCardState('已自动化', 'healthy')
  return createCardState('待检查', 'warning')
})

const webdavSignals = computed(() => {
  const signals: string[] = []
  if (config.provider === 'open115') {
    signals.push(config.open115.client_id.trim() ? `Client ID: ${config.open115.client_id.trim()}` : '尚未填写 115 Open Client ID')
    signals.push(config.open115.user_id.trim() ? `用户: ${config.open115.user_id.trim()}` : '尚未完成扫码授权')
    signals.push(config.backup.remote_dir.trim() ? `写入目录: ${config.backup.remote_dir.trim()}` : '尚未选择远端目录')
    return signals
  }
  signals.push(config.webdav.url.trim() ? `地址: ${config.webdav.url.trim()}` : '尚未填写 WebDAV 地址')
  signals.push(config.backup.remote_dir.trim() ? `写入目录: ${config.backup.remote_dir.trim()}` : '尚未选择远端目录')
  signals.push(systemStatus.value?.rclone_installed ? 'Rclone 可用，可直接执行同步' : 'Rclone 未就绪，备份无法启动')
  return signals
})

const backupSignals = computed(() => {
  const signals: string[] = []
  signals.push(config.backup.library_dir.trim() ? `照片库: ${config.backup.library_dir.trim()}` : '照片库路径未设置')
  signals.push(config.backup.backups_dir.trim() ? `数据库备份: ${config.backup.backups_dir.trim()}` : '数据库备份路径未设置')
  signals.push(config.backup.library_dir.trim() && config.backup.backups_dir.trim() ? '两类数据都会进入备份任务' : '建议同时配置两类路径，避免备份不完整')
  return signals
})

const encryptSignals = computed(() => {
  if (!config.encrypt.enabled) {
    return ['当前上传为明文目录结构', '适合可信存储环境', '如需保护文件内容，建议开启加密']
  }

  return [
    config.encrypt.password.trim() ? '已填写内容加密密码' : '内容加密密码缺失',
    config.encrypt.salt.trim() ? '已填写文件名混淆盐' : '文件名混淆盐缺失',
    '请妥善保管密钥，丢失后无法恢复已加密数据',
  ]
})

const automationSignals = computed(() => {
  const signals: string[] = []
  signals.push(config.cron.enabled ? `自动备份: ${config.cron.expression}` : '自动备份当前关闭')
  signals.push(config.server.auth_enabled ? `访问保护: ${config.server.auth_user || '已启用（默认账号）'}` : '访问保护未开启')
  if (systemStatus.value?.backup_status === 'running') {
    signals.push('当前有备份任务在运行，修改配置后建议下次任务生效')
  } else if (systemStatus.value?.next_run) {
    signals.push(`下次执行: ${systemStatus.value.next_run}`)
  } else {
    signals.push('暂无计划中的自动备份任务')
  }
  return signals
})

const validateSection = (section: SectionKey): string | null => {
  if (section === 'webdav') {
    if (draftConfig.value.provider === 'open115') {
      if (!draftConfig.value.open115.client_id.trim()) return '请输入 115 Open Client ID'
      if (!draftConfig.value.open115.access_token.trim() || !draftConfig.value.open115.refresh_token.trim()) {
        return '请先完成 115 Open 扫码授权'
      }
      if (!draftConfig.value.backup.remote_dir.trim()) return '请填写远端目录'
    } else {
      if (!draftConfig.value.webdav.url.trim()) return '请输入 WebDAV 服务器地址'
      if (!draftConfig.value.webdav.user.trim()) return '请输入 WebDAV 用户名'
      if (!draftConfig.value.webdav.password.trim()) return '请输入 WebDAV 密码或授权码'
      if (!draftConfig.value.backup.remote_dir.trim()) return '请选择远端备份目录'
    }
  }

  if (section === 'backup') {
    if (!draftConfig.value.backup.library_dir.trim() && !draftConfig.value.backup.backups_dir.trim()) {
      return '请至少填写一个备份路径（照片库或数据库备份路径）'
    }
  }

  if (section === 'encrypt' && draftConfig.value.encrypt.enabled) {
    if (!draftConfig.value.encrypt.password.trim()) return '请输入加密密码'
    if (!draftConfig.value.encrypt.salt.trim()) return '请输入加密混淆盐'
  }

  if (section === 'automation' && draftConfig.value.server.auth_enabled) {
    if (!draftConfig.value.server.auth_user.trim()) return '请输入管理员用户名'
  }

  return null
}

const requiresAuthRefresh = (previous: AppConfig, next: AppConfig) => {
  if (!previous.server.auth_enabled && !next.server.auth_enabled) return false
  if (previous.server.auth_enabled !== next.server.auth_enabled) return true
  if (previous.server.auth_user !== next.server.auth_user) return true

  const nextPassword = (next.server.auth_password ?? '').trim()
  const previousPassword = (previous.server.auth_password ?? '').trim()
  return nextPassword !== '' && nextPassword !== previousPassword
}

const testNotify = async () => {
  isTestingNotify.value = true
  try {
    // 先保存当前配置以确保后端有最新的 Bark 地址
    await api.saveConfig(draftConfig.value)
    Object.assign(config, cloneConfig(draftConfig.value))
    await api.testNotify()
    showToast('success', '测试通知已发送', '请检查 Bark App；你会看到更明确的状态字段，例如触发方式、当前阶段和结果。')
  } catch (error) {
    if (handleAuthFailure(error)) return
    showToast('error', '发送失败', getErrorMessage(error))
  } finally {
    isTestingNotify.value = false
  }
}

const saveSection = async () => {
  if (!activeSection.value) return

  const error = validateSection(activeSection.value)
  if (error) {
    validationError.value = error
    return
  }

  validationError.value = ''
  isSaving.value = true
  const authRefreshNeeded = requiresAuthRefresh(config, draftConfig.value)
  try {
    await api.saveConfig(draftConfig.value)
    Object.assign(config, cloneConfig(draftConfig.value))

    if (authRefreshNeeded) {
      showToast('info', '正在重新验证身份', '访问保护配置已更新，页面将立即刷新。', 1200)
      window.setTimeout(() => {
        window.location.replace(window.location.pathname || '/')
      }, 180)
      return
    }

    await refreshConfig()
    closeSection()
    showToast('success', '保存成功', '配置已更新，新的状态摘要已经同步刷新。')
  } catch (error) {
    if (handleAuthFailure(error)) return
    showToast('error', '保存失败', getErrorMessage(error))
  } finally {
    isSaving.value = false
  }
}

const testConnection = async () => {
  isTesting.value = true
  testResult.value = '测试中...'
  testSuccess.value = false
  try {
    if (draftConfig.value.provider === 'open115') {
      const result = await api.open115Test()
      if (!result.success) {
        throw new Error(result.message || '115 Open 连接失败')
      }
      testSuccess.value = true
      testResult.value = '连接成功!'
      showToast('success', '连接成功', '115 Open Token 可用。')
    } else {
      const result = await api.testWebDAV({
        url: draftConfig.value.webdav.url,
        user: draftConfig.value.webdav.user,
        password: draftConfig.value.webdav.password,
      })

      if (!result.success) {
        throw new Error(result.message || 'WebDAV 连接失败')
      }

      testSuccess.value = true
      testResult.value = '连接成功!'
      showToast('success', '连接成功', 'WebDAV 可用，可以正常浏览并写入远端目录。')
    }
  } catch (error) {
    if (handleAuthFailure(error)) return
    testSuccess.value = false
    testResult.value = '连接失败: ' + getErrorMessage(error)
    showToast('error', '连接失败', getErrorMessage(error))
  } finally {
    isTesting.value = false
  }
}

const showFolderPicker = ref(false)
const showPathInput = ref(false)
const targetLocalField = ref<LocalField>('library_dir')
const currentLocalPath = ref('')
const localDirs = ref<DirEntry[]>([])
const isLoadingLocal = ref(false)

const showRemoteFolderPicker = ref(false)
const currentRemotePath = ref('/')
const remoteDirs = ref<DirEntry[]>([])
const isLoadingRemote = ref(false)

const isWindowsPath = computed(() => currentLocalPath.value.includes('\\'))
const pathSegments = computed(() => {
  const path = currentLocalPath.value
  if (!path) return []
  const separator = path.includes('\\') ? '\\' : '/'
  return path.split(separator).filter((segment) => segment !== '')
})
const canGoUp = computed(() => {
  const path = currentLocalPath.value
  return path !== '/' && path !== 'C:\\' && path !== ''
})
const remotePathSegments = computed(() => currentRemotePath.value.split('/').filter((segment) => segment !== ''))
const remoteCanGoUp = computed(() => currentRemotePath.value !== '/')

const normalizeRemotePath = (path: string) => {
  if (!path || path.trim() === '') return '/'
  const normalized = path.replace(/\\/g, '/').trim()
  if (normalized === '/') return '/'
  return normalized.startsWith('/') ? normalized : `/${normalized}`
}

const openFolderPicker = (field: LocalField) => {
  targetLocalField.value = field
  showFolderPicker.value = true
  showPathInput.value = false
  currentLocalPath.value = draftConfig.value.backup[field] || ''
  void loadLocalDir(currentLocalPath.value)
}

const openRemoteFolderPicker = () => {
  if (!draftConfig.value.webdav.url.trim() || !draftConfig.value.webdav.user.trim() || !draftConfig.value.webdav.password.trim()) {
    showToast('warning', '请先完善连接信息', '需要先填写 WebDAV 地址、用户名和密码，才能浏览远端目录。')
    return
  }
  showRemoteFolderPicker.value = true
  currentRemotePath.value = normalizeRemotePath(draftConfig.value.backup.remote_dir)
  void loadRemoteDir(currentRemotePath.value)
}

const loadLocalDir = async (path: string) => {
  isLoadingLocal.value = true
  try {
    const items = await api.listLocal(path)
    localDirs.value = items.filter((item) => item.IsDir).sort((left, right) => left.Name.localeCompare(right.Name))
    if (path === '') {
      currentLocalPath.value = items.length > 0 && items[0]?.Path.includes('\\') ? 'C:\\' : '/'
    }
  } catch (error) {
    if (handleAuthFailure(error)) return
    showToast('error', '加载目录失败', getErrorMessage(error))
  } finally {
    isLoadingLocal.value = false
  }
}

const loadRemoteDir = async (path: string) => {
  isLoadingRemote.value = true
  try {
    const normalizedPath = normalizeRemotePath(path)
    const items = await api.listWebDAV({
      url: draftConfig.value.webdav.url,
      user: draftConfig.value.webdav.user,
      password: draftConfig.value.webdav.password,
      path: normalizedPath,
    })
    currentRemotePath.value = normalizedPath
    remoteDirs.value = items.filter((item) => item.IsDir).sort((left, right) => left.Name.localeCompare(right.Name))
  } catch (error) {
    if (handleAuthFailure(error)) return
    showToast('error', '加载远端目录失败', getErrorMessage(error))
  } finally {
    isLoadingRemote.value = false
  }
}

const resolveLocalEntryPath = (item: DirEntry) => {
  const candidate = (item.Path || '').trim()
  if (candidate.startsWith('/') || /^[A-Za-z]:[\\/]/.test(candidate)) {
    return candidate
  }

  const separator = currentLocalPath.value.includes('\\') ? '\\' : '/'
  let nextPath = currentLocalPath.value
  if (nextPath === '' || nextPath.endsWith(separator)) {
    nextPath += item.Name
  } else {
    nextPath += separator + item.Name
  }
  return nextPath
}

const enterLocalDir = (item: DirEntry) => {
  const nextPath = resolveLocalEntryPath(item)
  currentLocalPath.value = nextPath
  void loadLocalDir(nextPath)
}

const resolveRemoteEntryPath = (item: DirEntry) => {
  const candidate = normalizeRemotePath((item.Path || '').trim())
  if (candidate !== '/' || (item.Path || '').trim().startsWith('/')) {
    return candidate
  }

  return currentRemotePath.value === '/' ? `/${item.Name}` : `${currentRemotePath.value}/${item.Name}`
}

const enterRemoteDir = (item: DirEntry) => {
  const nextPath = resolveRemoteEntryPath(item)
  void loadRemoteDir(nextPath)
}

const goUpLocalDir = () => {
  const separator = currentLocalPath.value.includes('\\') ? '\\' : '/'
  const parts = currentLocalPath.value.split(separator)
  if (parts.length > 0 && parts[parts.length - 1] === '') parts.pop()
  parts.pop()
  let nextPath = parts.join(separator)
  if (nextPath === '' || (separator === '\\' && !nextPath.includes('\\'))) nextPath += separator
  currentLocalPath.value = nextPath
  void loadLocalDir(nextPath)
}

const goUpRemoteDir = () => {
  if (currentRemotePath.value === '/') return
  const parts = currentRemotePath.value.split('/').filter(Boolean)
  parts.pop()
  const nextPath = parts.length === 0 ? '/' : `/${parts.join('/')}`
  void loadRemoteDir(nextPath)
}

const navigateToSegment = (index: number) => {
  const separator = isWindowsPath.value ? '\\' : '/'
  const segments = pathSegments.value.slice(0, index + 1)
  let nextPath = segments.join(separator)
  if (isWindowsPath.value) {
    if (!nextPath.endsWith('\\')) nextPath += '\\'
  } else {
    nextPath = '/' + nextPath
  }
  currentLocalPath.value = nextPath
  void loadLocalDir(nextPath)
}

const navigateToRemoteSegment = (index: number) => {
  const segments = remotePathSegments.value.slice(0, index + 1)
  const nextPath = segments.length === 0 ? '/' : `/${segments.join('/')}`
  void loadRemoteDir(nextPath)
}

const confirmFolder = () => {
  draftConfig.value.backup[targetLocalField.value] = currentLocalPath.value
  showFolderPicker.value = false
}

const confirmRemoteFolder = () => {
  draftConfig.value.backup.remote_dir = normalizeRemotePath(currentRemotePath.value)
  showRemoteFolderPicker.value = false
}
</script>

<style scoped>
.settings-page {
  display: flex;
  flex-direction: column;
  padding: 48px 64px;
  gap: 40px;
  max-width: 1400px;
  margin: 0 auto;
}

.settings-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 24px;
}

.settings-eyebrow,
.settings-modal-kicker {
  color: var(--text-tertiary);
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  margin-bottom: 10px;
}

.settings-title {
  font-family: var(--font-primary);
  font-weight: 800;
  font-size: 32px;
  color: var(--text-primary);
  letter-spacing: -0.5px;
  margin-bottom: 8px;
}

.settings-subtitle,
.settings-modal-header p {
  color: var(--text-secondary);
  font-size: 16px;
  max-width: 720px;
}

.settings-status-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 18px;
}

.status-chip,
.card-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 30px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 700;
}

.status-chip.healthy,
.card-badge.healthy {
  background: rgba(22, 163, 74, 0.12);
  color: #15803d;
}

.status-chip.warning,
.card-badge.warning {
  background: rgba(245, 158, 11, 0.14);
  color: #b45309;
}

.status-chip.info,
.card-badge.info {
  background: rgba(37, 99, 235, 0.14);
  color: #1d4ed8;
}

.status-chip.neutral,
.card-badge.neutral {
  background: var(--bg-card);
  color: var(--text-secondary);
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 24px;
}

.settings-card {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  width: 100%;
  padding: 24px;
  border-radius: 16px;
  background-color: var(--bg-card);
  border: 1px solid var(--border-subtle);
  text-align: left;
  transition: all 0.2s ease;
}

.settings-card:hover {
  transform: translateY(-2px);
  border-color: var(--border-strong);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.05);
}

.settings-card-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  flex-shrink: 0;
}

.settings-card-icon.webdav {
  color: #3b82f6;
  background: rgba(59, 130, 246, 0.1);
}

.settings-card-icon.backup {
  color: #10b981;
  background: rgba(16, 185, 129, 0.1);
}

.settings-card-icon.encrypt {
  color: #f59e0b;
  background: rgba(245, 158, 11, 0.1);
}

.settings-card-icon.automation {
  color: #8b5cf6;
  background: rgba(139, 92, 246, 0.1);
}

.settings-card-icon.notify {
  color: #ec4899;
  background: rgba(236, 72, 153, 0.1);
}

.notify-test-row {
  display: flex;
  align-items: center;
  gap: 12px;
}

.notify-test-row .btn {
  display: flex;
  align-items: center;
  gap: 6px;
}

.input-hint {
  font-size: 12px;
  color: var(--text-tertiary);
  margin-top: 4px;
}

.settings-card-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
}

.settings-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.settings-card-head > div {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.settings-card-head h2 {
  font-size: 20px;
  font-weight: 800;
  font-family: var(--font-primary);
  color: #ffffff;
}

.settings-card-head span {
  color: var(--text-tertiary);
  font-size: 13px;
  font-weight: 600;
}

.settings-card p {
  color: var(--text-secondary);
  font-size: 14px;
  line-height: 1.6;
}

.settings-card-signals {
  display: flex;
  flex-direction: column;
  gap: 8px;
  list-style: none;
}

.settings-card-signals li {
  position: relative;
  padding-left: 16px;
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.settings-card-signals li::before {
  content: '';
  position: absolute;
  left: 0;
  top: 8px;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--text-tertiary);
}

.settings-modal-overlay,
.picker-overlay {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(9, 9, 11, 0.55);
  padding: 24px;
  z-index: 50;
}

.settings-modal-card,
.picker-modal {
  width: min(760px, 100%);
  max-height: min(88vh, 920px);
  display: flex;
  flex-direction: column;
  border-radius: 16px;
  background: var(--bg-primary);
  border: 1px solid var(--border-strong);
  box-shadow: 0 8px 32px rgba(0,0,0,0.3);
  overflow: hidden;
}

.picker-modal {
  width: min(680px, 100%);
}

.settings-modal-header,
.picker-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding: 28px 28px 20px;
  border-bottom: 1px solid var(--border-strong);
}

.settings-modal-header h3,
.picker-header h3 {
  font-size: 28px;
  margin-bottom: 8px;
}

.settings-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 18px;
  background: var(--bg-card);
  color: var(--text-primary);
  border: none;
  cursor: pointer;
}

.settings-close:hover {
  background: var(--border-subtle);
}

.settings-modal-body,
.picker-body {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 24px 28px;
  overflow: auto;
}

.settings-modal-footer,
.picker-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 20px 28px 28px;
  border-top: 1px solid var(--border-strong);
}

.picker-footer-actions,
.settings-inline-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.input-field,
.toggle-field {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
  border-radius: 12px;
  background: var(--bg-card);
}

.input-label,
.toggle-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-primary);
}

.toggle-field {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 20px;
}

.toggle-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.toggle-desc,
.input-hint,
.selected-path-preview,
.settings-inline-message {
  color: var(--text-secondary);
  font-size: 13px;
}

.settings-inline-message.success {
  color: #16a34a;
}

.settings-inline-message.error,
.validation-error {
  color: #dc2626;
}

.input-control {
  width: 100%;
  height: 48px;
  padding: 0 16px;
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  background-color: transparent;
  color: var(--text-primary);
  font-size: 16px;
}

.path-input-row {
  display: flex;
  gap: 10px;
}

.browse-btn {
  flex-shrink: 0;
  height: 48px;
  border-radius: 12px;
  padding: 0 16px !important;
  font-size: 16px;
}

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

.switch {
  width: 44px;
  height: 24px;
  border-radius: 12px;
  background: var(--border-strong);
  position: relative;
  transition: all 0.2s ease;
}

.switch.active {
  background: var(--text-primary);
}

.thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  border-radius: 10px;
  background: var(--text-inverted);
  transition: all 0.2s ease;
}

.switch.active .thumb {
  left: 22px;
}

.breadcrumb-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 14px;
  background: var(--bg-card);
}

.breadcrumb-inner {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.breadcrumb-item,
.breadcrumb-edit-btn {
  padding: 6px 10px;
  border-radius: 10px;
  background: transparent;
  color: var(--text-primary);
  font-size: 13px;
}

.breadcrumb-item:hover,
.breadcrumb-edit-btn:hover,
.folder-item:hover {
  background: var(--border-subtle);
}

.breadcrumb-sep {
  color: var(--text-tertiary);
}

.folder-list {
  min-height: 280px;
  max-height: 48vh;
  border: 1px solid var(--border-strong);
  border-radius: 10px;
  overflow: hidden;
  background: var(--bg-primary);
}

.folder-scroll {
  overflow: auto;
}

.folder-item,
.folder-empty {
  display: flex;
  align-items: center;
  min-height: 48px;
  padding: 0 16px;
  color: var(--text-primary);
  border-bottom: 1px solid var(--border-subtle);
}

.folder-item {
  cursor: pointer;
  gap: 10px;
}

.folder-item.go-up {
  color: var(--text-secondary);
  font-size: 13px;
  font-weight: 500;
}

.folder-empty {
  justify-content: center;
  gap: 8px;
  color: var(--text-tertiary);
}

.spin-icon {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.path-manual-input {
  margin-top: 6px;
}

.radio-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.radio-option {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 14px 16px;
  border-radius: 10px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-card);
  cursor: pointer;
  transition: all 0.2s ease;
}

.radio-option:hover {
  border-color: var(--border-strong);
}

.radio-option.active {
  border-color: var(--accent);
  background: rgba(99, 102, 241, 0.06);
}

.radio-option input[type="radio"] {
  margin-top: 3px;
  accent-color: var(--accent);
}

.radio-option-text {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.radio-option-text strong {
  font-size: 14px;
  color: #ffffff;
}

.radio-option-text span {
  font-size: 12px;
  color: var(--text-secondary);
}

@media (max-width: 960px) {
  .settings-page {
    padding: 24px;
  }

  .settings-grid {
    grid-template-columns: 1fr;
  }

  .settings-hero,
  .settings-modal-footer,
  .picker-footer,
  .path-input-row {
    flex-direction: column;
    align-items: stretch;
  }

  .settings-card-head {
    flex-direction: column;
  }
}
</style>
