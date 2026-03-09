<template>
  <div :class="['wizard-container', isSettingsMode ? 'settings-mode' : '']">
    <div class="left-col">
      <div class="title-box">
        <h1 class="main-title">ImmichTo115</h1>
        <h2 class="sub-title">{{ isSettingsMode ? '配置设置' : 'Setup Wizard' }}</h2>
      </div>

      <div class="step-list">
        <div class="step-item">
          <div :class="['step-icon', step > 1 ? 'completed' : step === 1 ? 'active' : 'pending']">
            <LucideCheck v-if="step > 1" :size="16" />
            <span v-else>1</span>
          </div>
          <span :class="['step-text', step >= 1 ? 'active-text' : 'pending-text']">接入方式</span>
        </div>
        <div class="step-item">
          <div :class="['step-icon', step > 2 ? 'completed' : step === 2 ? 'active' : 'pending']">
            <LucideCheck v-if="step > 2" :size="16" />
            <span v-else>2</span>
          </div>
          <span :class="['step-text', step >= 2 ? 'active-text' : 'pending-text']">备份路径</span>
        </div>
        <div class="step-item">
          <div :class="['step-icon', step > 3 ? 'completed' : step === 3 ? 'active' : 'pending']">
            <LucideCheck v-if="step > 3" :size="16" />
            <span v-else>3</span>
          </div>
          <span :class="['step-text', step >= 3 ? 'active-text' : 'pending-text']">加密配置</span>
        </div>
        <div class="step-item">
          <div :class="['step-icon', step > 4 ? 'completed' : step === 4 ? 'active' : 'pending']">
            <LucideCheck v-if="step > 4" :size="16" />
            <span v-else>4</span>
          </div>
          <span :class="['step-text', step >= 4 ? 'active-text' : 'pending-text']">定时任务</span>
        </div>
      </div>
    </div>

    <div class="right-col">
      <div v-if="step === 1" class="step-content">
        <div class="header-group">
          <h2 class="content-title">接入方式</h2>
          <p class="content-desc">选择继续使用 WebDAV，或切换到 115 Open 扫码授权。</p>
        </div>

        <div class="form-group">
          <div class="radio-group provider-group">
            <label class="radio-option" :class="{ active: config.provider === 'webdav' }">
              <input type="radio" v-model="config.provider" value="webdav" />
              <div class="radio-option-text">
                <strong>WebDAV</strong>
                <span>兼容现有 rclone + WebDAV 备份模式</span>
              </div>
            </label>
            <label class="radio-option" :class="{ active: config.provider === 'open115' }">
              <input type="radio" v-model="config.provider" value="open115" />
              <div class="radio-option-text">
                <strong>115 Open</strong>
                <span>通过二维码授权，后续走 115 Open API</span>
              </div>
            </label>
          </div>

          <template v-if="config.provider === 'webdav'">
            <div class="input-field">
              <span class="input-label">服务器地址</span>
              <input class="input-control" type="text" v-model="config.webdav.url" placeholder="请输入 WebDAV 地址，例如 https://dav.example.com" />
            </div>

            <div class="input-field">
              <span class="input-label">用户名</span>
              <input class="input-control" type="text" v-model="config.webdav.user" placeholder="请输入 WebDAV 用户名" />
            </div>

            <div class="input-field">
              <span class="input-label">密码或授权码</span>
              <input class="input-control" type="password" v-model="config.webdav.password" placeholder="••••••••••••" autocomplete="off" />
            </div>

            <div class="input-field">
              <span class="input-label">远端目录</span>
              <div class="path-input-row">
                <input class="input-control" type="text" v-model="config.backup.remote_dir" placeholder="例如 /immich-backup（云端目录）" style="flex: 1;" />
                <button class="btn secondary browse-btn" @click="openRemoteFolderPicker">
                  <LucideFolderOpen :size="16" />
                  WebDAV
                </button>
              </div>
              <span class="input-hint">WebDAV 用户的根目录只是登录后的起点，真正备份会写入这里选择的云端目录。</span>
            </div>
          </template>

          <template v-else>
            <div class="input-field">
              <span class="input-label">Access Token</span>
              <input class="input-control" type="password" v-model="config.open115.access_token" placeholder="直接填写 OpenList / 115 Open 获取到的 access_token" autocomplete="off" />
            </div>

            <div class="input-field">
              <span class="input-label">Refresh Token</span>
              <input class="input-control" type="password" v-model="config.open115.refresh_token" placeholder="直接填写 OpenList / 115 Open 获取到的 refresh_token" autocomplete="off" />
            </div>

            <div class="input-field">
              <span class="input-label">Client ID（可选）</span>
              <input class="input-control" type="text" v-model="config.open115.client_id" placeholder="只有你要在项目内扫码授权时才需要填写" />
              <span class="input-hint">推荐直接填写 token；如果你后面要走项目内扫码授权，再填写 client_id。</span>
            </div>

            <div class="input-field">
              <span class="input-label">远端目录</span>
              <div class="path-input-row">
                <input class="input-control" type="text" v-model="config.backup.remote_dir" placeholder="例如 /immich-backup（逻辑目录）" style="flex: 1;" />
                <button class="btn secondary browse-btn" @click="openRemoteFolderPicker">
                  <LucideFolderOpen :size="16" />
                  115 目录
                </button>
              </div>
              <span class="input-hint">可以手动输入路径，或点击"115 目录"按钮浏览选择。</span>
            </div>

            <div class="settings-inline-actions">
              <button class="btn secondary" @click="openOpenListTokenPage">
                获取 Token（OpenList）
              </button>
              <button class="btn secondary" @click="startOpen115Auth" :disabled="isOpen115AuthLoading || !config.open115.client_id.trim()">
                {{ isOpen115AuthLoading ? '生成中...' : '开始扫码授权（可选）' }}
              </button>
              <button class="btn secondary" @click="finishOpen115Auth" :disabled="isOpen115Finishing || !open115Auth.uid || open115Authorized !== true">
                {{ isOpen115Finishing ? '确认中...' : '完成授权' }}
              </button>
              <button class="btn secondary" @click="testConnection" :disabled="isTesting">
                {{ isTesting ? '测试中...' : '测试连接' }}
              </button>
            </div>
            <div class="input-hint">
              推荐先点“获取 Token（OpenList）”。在打开的页面中选择 <strong>115 Network Disk Verification</strong>，勾选 <strong>Use parameters provided by OpenList</strong>，留空 Client ID / Secret，获取到 <code>access_token</code> / <code>refresh_token</code> 后再粘贴回来。若你已有自己的开放平台应用，也可以继续使用项目内扫码授权。
            </div>

            <div v-if="open115Auth.qrcode" class="qrcode-panel">
              <p class="input-label">扫码二维码</p>
              <img :src="open115Auth.qrcode" alt="115 QR Code" class="qrcode-image" />
              <p class="input-hint">请使用 115 App 扫码并确认授权。</p>
              <p class="input-hint">当前状态：{{ open115AuthStatusText }}</p>
            </div>

            <div v-if="config.open115.user_id" class="input-hint success-hint">
              当前已授权用户 ID：{{ config.open115.user_id }}
            </div>
          </template>
        </div>

        <div class="buttons">
          <span v-if="testResult" :style="{ color: testSuccess ? 'var(--text-primary)' : 'red', alignSelf: 'center', marginRight: '16px' }">{{ testResult }}</span>
          <span v-if="validationError && step === 1" class="validation-error">{{ validationError }}</span>
          <button class="btn primary" @click="nextStep">下一步</button>
        </div>
      </div>

      <div v-else-if="step === 2" class="step-content">
        <div class="header-group">
          <h2 class="content-title">备份路径</h2>
          <p class="content-desc">指定需本地备份的 Immich 照片库和数据库目录</p>
        </div>

        <div class="form-group">
          <div class="input-field">
            <span class="input-label">照片库路径 (Library Dir)</span>
            <div class="path-input-row">
              <input class="input-control" type="text" v-model="config.backup.library_dir" placeholder="例如 /data/library 或 D:\\Immich\\library" style="flex: 1;" />
              <button class="btn secondary browse-btn" @click="openFolderPicker('library_dir')">
                <LucideFolderOpen :size="16" />
                浏览
              </button>
            </div>
          </div>

          <div class="input-field">
            <span class="input-label">数据库备份路径 (DB Dump Dir)</span>
            <div class="path-input-row">
              <input class="input-control" type="text" v-model="config.backup.backups_dir" placeholder="例如 /data/backups 或 D:\\Immich\\backups" style="flex: 1;" />
              <button class="btn secondary browse-btn" @click="openFolderPicker('backups_dir')">
                <LucideFolderOpen :size="16" />
                浏览
              </button>
            </div>
          </div>

          <div class="input-field">
            <span class="input-label">备份模式</span>
            <div class="radio-group provider-group">
              <label class="radio-option" :class="{ active: config.backup.mode === 'copy' }">
                <input type="radio" v-model="config.backup.mode" value="copy" />
                <div class="radio-option-text">
                  <strong>增量备份 (copy)</strong>
                  <span>只上传新增或修改的文件，不删除远端已有文件</span>
                </div>
              </label>
              <label class="radio-option" :class="{ active: config.backup.mode === 'sync' }">
                <input type="radio" v-model="config.backup.mode" value="sync" />
                <div class="radio-option-text">
                  <strong>镜像同步 (sync)</strong>
                  <span>保持远端与本地一致，可删除远端多余文件</span>
                </div>
              </label>
            </div>
          </div>

          <div v-if="config.backup.mode === 'sync'" class="toggle-field" @click="config.backup.allow_remote_delete = !config.backup.allow_remote_delete">
            <div class="toggle-info">
              <span class="toggle-title">允许删除远端多余文件</span>
              <span class="toggle-desc">默认关闭。开启后，sync 模式会尝试删除远端存在但本地已删除的文件。</span>
            </div>
            <div :class="['switch', config.backup.allow_remote_delete ? 'active' : '']">
              <div class="thumb"></div>
            </div>
          </div>
        </div>

        <div class="buttons space-between">
          <button class="btn secondary" @click="prevStep">上一步</button>
          <span v-if="validationError && step === 2" class="validation-error">{{ validationError }}</span>
          <button class="btn primary" @click="nextStep">下一步</button>
        </div>
      </div>

      <div v-else-if="step === 3" class="step-content">
        <div class="header-group">
          <h2 class="content-title">加密配置</h2>
          <p class="content-desc">保护您的隐私数据，以防数据泄露</p>
        </div>

        <div class="form-group">
          <template v-if="config.provider === 'open115'">
            <div class="toggle-field" @click="config.open115_encrypt.enabled = !config.open115_encrypt.enabled">
              <div class="toggle-info">
                <span class="toggle-title">启用 Open115 本地加密</span>
                <span class="toggle-desc">支持 <code>temp</code> 和 <code>stream</code> 两种模式；正式建议优先使用 <code>temp</code>，验证稳定后再尝试 <code>stream</code>。</span>
              </div>
              <div :class="['switch', config.open115_encrypt.enabled ? 'active' : '']">
                <div class="thumb"></div>
              </div>
            </div>

            <div v-if="config.open115_encrypt.enabled" class="input-field">
              <span class="input-label">加密模式</span>
              <div class="radio-group provider-group">
                <label class="radio-option" :class="{ active: config.open115_encrypt.mode === 'temp' }">
                  <input type="radio" v-model="config.open115_encrypt.mode" value="temp" />
                  <div class="radio-option-text">
                    <strong>temp</strong>
                    <span>先生成临时 `.enc` 文件后上传，更稳</span>
                  </div>
                </label>
                <label class="radio-option" :class="{ active: config.open115_encrypt.mode === 'stream' }">
                  <input type="radio" v-model="config.open115_encrypt.mode" value="stream" />
                  <div class="radio-option-text">
                    <strong>stream</strong>
                    <span>流式加密上传，更省空间；当前已完成 debug 闭环验证，但正式使用仍建议先从小目录开始</span>
                  </div>
                </label>
              </div>
            </div>

            <div class="input-field" v-if="config.open115_encrypt.enabled">
              <span class="input-label">加密密码</span>
              <input class="input-control" type="password" v-model="config.open115_encrypt.password" placeholder="用于 Open115 内容加密" autocomplete="new-password" />
            </div>

            <div class="input-field" v-if="config.open115_encrypt.enabled">
              <span class="input-label">加密盐</span>
              <input class="input-control" type="password" v-model="config.open115_encrypt.salt" placeholder="可留空使用自动盐" autocomplete="new-password" />
            </div>

            <div class="input-field" v-if="config.open115_encrypt.enabled">
              <span class="input-label">临时目录</span>
              <input class="input-control" type="text" v-model="config.open115_encrypt.temp_dir" placeholder="例如 /tmp/immichto115-open115-encrypt" />
            </div>

            <div class="input-field" v-if="config.open115_encrypt.enabled">
              <span class="input-label">最小剩余空间（MB）</span>
              <input class="input-control" type="number" min="0" v-model.number="config.open115_encrypt.min_free_space_mb" placeholder="1024" />
            </div>
          </template>
          <template v-else>
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
          </template>
        </div>

        <div class="buttons space-between">
          <button class="btn secondary" @click="prevStep">上一步</button>
          <span v-if="validationError && step === 3" class="validation-error">{{ validationError }}</span>
          <button class="btn primary" @click="nextStep">下一步</button>
        </div>
      </div>

      <div v-else-if="step === 4" class="step-content">
        <div class="header-group">
          <h2 class="content-title">定时任务</h2>
          <p class="content-desc">配置自动备份的时间表</p>
        </div>

        <div class="form-group">
          <div class="toggle-field" @click="config.server.auth_enabled = !config.server.auth_enabled">
            <div class="toggle-info">
              <span class="toggle-title">启用访问保护</span>
              <span class="toggle-desc">启用后，管理页面、接口和实时日志都会受管理员账号密码保护；完成配置后会立即重新验证身份</span>
            </div>
            <div :class="['switch', config.server.auth_enabled ? 'active' : '']">
              <div class="thumb"></div>
            </div>
          </div>

          <div v-if="config.server.auth_enabled" class="input-field">
            <span class="input-label">管理员用户名</span>
            <input class="input-control" type="text" v-model="config.server.auth_user" placeholder="例如：admin" />
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

      <div v-if="showFolderPicker" class="modal-overlay" @click.self="showFolderPicker = false">
        <div class="modal">
          <div class="modal-header">
            <h3 style="margin: 0; font-size: 16px;">选择本地文件夹</h3>
            <button class="btn-icon" @click="showFolderPicker = false" style="background:none; border:none; cursor:pointer; color: var(--text-primary);"><LucideX :size="20" /></button>
          </div>
          <div class="modal-body">
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
            <input v-if="showPathInput" type="text" v-model="currentLocalPath" class="input-control path-manual-input" @keydown.enter="loadLocalDir(currentLocalPath); showPathInput = false" placeholder="输入路径后按 Enter" />
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
            <h3 style="margin: 0; font-size: 16px;">选择{{ config.provider === 'open115' ? ' 115 Open ' : ' WebDAV ' }}备份目录</h3>
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
                  <button class="breadcrumb-item" @click="navigateToRemoteSegment(idx)" :class="{ last: idx === remotePathSegments.length - 1 }">{{ seg }}</button>
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
import { ref, computed, onMounted, reactive, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { LucideCheck, LucideFolder, LucideFolderOpen, LucideX, LucideHardDrive, LucidePencil, LucideLoader2, LucideCornerLeftUp } from 'lucide-vue-next'
import { api, getErrorMessage, handleAuthFailure, type AppConfig, type DirEntry, type Open115AuthStartResponse } from '../api'
import { showToast } from '../composables/toast'
import { markSetupComplete } from '../router'
import CronScheduler from '../components/CronScheduler.vue'
import { createDefaultConfig } from '../configDefaults'

const router = useRouter()
const route = useRoute()
const step = ref(1)
const testResult = ref('')
const testSuccess = ref(false)
const isTesting = ref(false)
const isSaving = ref(false)
const isSettingsMode = computed(() => route.name === 'settings')
const config = reactive<AppConfig>(createDefaultConfig())

const isOpen115AuthLoading = ref(false)
const isOpen115Finishing = ref(false)
const open115Auth = reactive<Open115AuthStartResponse>({ uid: '', time: 0, sign: '', qrcode: '', created_at: '' })
const open115AuthStatusText = ref('未开始')
const open115Authorized = ref<boolean | null>(null)
let authPollTimer: number | null = null

const OPENLIST_TOKEN_URL = 'https://api.oplist.org/'

const openOpenListTokenPage = () => {
  if (typeof window !== 'undefined') {
    window.open(OPENLIST_TOKEN_URL, '_blank', 'noopener,noreferrer')
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
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    open115AuthStatusText.value = '状态查询失败：' + getErrorMessage(err)
    stopAuthPolling()
  }
}

const startOpen115Auth = async () => {
  if (!config.open115.client_id.trim()) {
    validationError.value = '请输入 115 Open Client ID'
    return
  }
  validationError.value = ''
  isOpen115AuthLoading.value = true
  open115Authorized.value = null
  open115AuthStatusText.value = '正在生成二维码...'
  try {
    const result = await api.open115AuthStart({ client_id: config.open115.client_id.trim() })
    Object.assign(open115Auth, result)
    open115AuthStatusText.value = '等待扫码'
    stopAuthPolling()
    authPollTimer = window.setInterval(() => {
      void pollOpen115Auth()
    }, 2500)
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    showToast('error', '启动扫码失败', getErrorMessage(err))
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
    config.open115 = { ...config.open115, ...result.state }
    open115Authorized.value = true
    open115AuthStatusText.value = '授权完成'
    showToast('success', '授权成功', '115 Open token 已保存，可以继续下一步。')
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    showToast('error', '完成授权失败', getErrorMessage(err))
  } finally {
    isOpen115Finishing.value = false
  }
}

onMounted(async () => {
  try {
    const data = await api.getConfig()
    Object.assign(config, data)
  } catch (error) {
    console.warn('无法读取现有配置，已回退到默认值。', error)
  }
})

onUnmounted(() => stopAuthPolling())

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
    if (!newPath.endsWith('\\')) newPath += '\\'
  } else {
    newPath = '/' + newPath
  }
  currentLocalPath.value = newPath
  void loadLocalDir(newPath)
}

const openFolderPicker = (field: 'library_dir' | 'backups_dir') => {
  targetLocalField.value = field
  showFolderPicker.value = true
  showPathInput.value = false
  currentLocalPath.value = config.backup[field] || ''
  void loadLocalDir(currentLocalPath.value)
}

const normalizeRemotePath = (path: string) => {
  if (!path || path.trim() === '') return '/'
  const normalized = path.replace(/\\/g, '/').trim()
  if (normalized === '/') return '/'
  return normalized.startsWith('/') ? normalized : `/${normalized}`
}

const openRemoteFolderPicker = () => {
  if (config.provider === 'webdav') {
    if (!config.webdav.url.trim() || !config.webdav.user.trim() || !config.webdav.password.trim()) {
      showToast('warning', '请先完善连接信息', '需要先填写 WebDAV 地址、用户名和密码，才能浏览远端目录。')
      return
    }
  } else {
    if (!config.open115.access_token.trim() || !config.open115.refresh_token.trim()) {
      showToast('warning', '请先完成授权', '需要先完成 115 Open 扫码授权，才能浏览远端目录。')
      return
    }
  }
  showRemoteFolderPicker.value = true
  currentRemotePath.value = normalizeRemotePath(config.backup.remote_dir)
  void loadRemoteDir(currentRemotePath.value)
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
    showToast('error', '加载目录失败', getErrorMessage(err))
  } finally {
    isLoadingLocal.value = false
  }
}

const loadRemoteDir = async (path: string) => {
  isLoadingRemote.value = true
  try {
    const normalizedPath = normalizeRemotePath(path)
    const items = config.provider === 'open115'
      ? await api.open115List(normalizedPath)
      : await api.listWebDAV({
          url: config.webdav.url,
          user: config.webdav.user,
          password: config.webdav.password,
          vendor: config.webdav.vendor,
          path: normalizedPath,
        })
    currentRemotePath.value = normalizedPath
    remoteDirs.value = items.filter(i => i.IsDir).sort((a, b) => a.Name.localeCompare(b.Name))
  } catch (err: any) {
    if (handleAuthFailure(err)) return
    showToast('error', config.provider === 'open115' ? '加载 115 目录失败' : '加载 WebDAV 目录失败', getErrorMessage(err))
  } finally {
    isLoadingRemote.value = false
  }
}

const resolveLocalEntryPath = (item: DirEntry) => {
  const candidate = (item.Path || '').trim()
  if (candidate.startsWith('/') || /^[A-Za-z]:[\\/]/.test(candidate)) {
    return candidate
  }
  const sep = currentLocalPath.value.includes('\\') ? '\\' : '/'
  let newPath = currentLocalPath.value
  if (newPath === '' || newPath.endsWith(sep)) newPath += item.Name
  else newPath += sep + item.Name
  return newPath
}

const enterLocalDir = (item: DirEntry) => {
  const newPath = resolveLocalEntryPath(item)
  currentLocalPath.value = newPath
  void loadLocalDir(newPath)
}

const resolveRemoteEntryPath = (item: DirEntry) => {
  const candidate = normalizeRemotePath((item.Path || '').trim())
  if (candidate !== '/' || (item.Path || '').trim().startsWith('/')) return candidate
  return currentRemotePath.value === '/' ? `/${item.Name}` : `${currentRemotePath.value}/${item.Name}`
}

const enterRemoteDir = (item: DirEntry) => {
  const newPath = resolveRemoteEntryPath(item)
  void loadRemoteDir(newPath)
}

const goUpLocalDir = () => {
  const sep = currentLocalPath.value.includes('\\') ? '\\' : '/'
  let parts = currentLocalPath.value.split(sep)
  if (parts.length > 0 && parts[parts.length - 1] === '') parts.pop()
  parts.pop()
  let newPath = parts.join(sep)
  if (newPath === '' || (sep === '\\' && !newPath.includes('\\'))) newPath += sep
  currentLocalPath.value = newPath
  void loadLocalDir(newPath)
}

const goUpRemoteDir = () => {
  if (currentRemotePath.value === '/') return
  const parts = currentRemotePath.value.split('/').filter(Boolean)
  parts.pop()
  const newPath = parts.length === 0 ? '/' : `/${parts.join('/')}`
  void loadRemoteDir(newPath)
}

const navigateToRemoteSegment = (idx: number) => {
  const segs = remotePathSegments.value.slice(0, idx + 1)
  const newPath = segs.length === 0 ? '/' : `/${segs.join('/')}`
  void loadRemoteDir(newPath)
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
    if (config.provider === 'webdav') {
      if (!config.webdav.url.trim()) return '请输入 WebDAV 服务器地址'
      if (!config.webdav.user.trim()) return '请输入用户名'
      if (!config.webdav.password.trim()) return '请输入密码'
    } else {
      if (!config.open115.access_token.trim() || !config.open115.refresh_token.trim()) {
        return '请填写 access_token 和 refresh_token，或先完成扫码授权'
      }
    }
    if (!config.backup.remote_dir.trim()) return '请选择远端备份目录'
  } else if (step.value === 2) {
    if (!config.backup.library_dir.trim() && !config.backup.backups_dir.trim()) return '请至少填写一个备份路径（照片库或数据库备份路径）'
  } else if (step.value === 3) {
    if (config.provider === 'open115') {
      if (config.open115_encrypt.enabled && !config.open115_encrypt.password.trim()) {
        return '请输入 Open115 加密密码'
      }
    } else {
      if (config.encrypt.enabled) {
        if (!config.encrypt.password.trim()) return '请输入加密密码'
        if (!config.encrypt.salt.trim()) return '请输入加密混淆盐'
      }
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
    if (config.provider === 'webdav') {
      const result = await api.testWebDAV({
        url: config.webdav.url,
        user: config.webdav.user,
        password: config.webdav.password,
        vendor: config.webdav.vendor,
      })
      if (!result.success) throw new Error(result.message || 'WebDAV 连接失败')
      testSuccess.value = true
      testResult.value = '连接成功!'
      showToast('success', '连接成功', 'WebDAV 已通过测试，可以继续下一步配置。')
    } else {
      const result = await api.open115Test()
      if (!result.success) throw new Error(result.message || '115 Open 连接失败')
      testSuccess.value = true
      testResult.value = '连接成功!'
      showToast('success', '连接成功', '115 Open Token 可用。')
    }
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
    if (config.server.auth_enabled) {
      window.location.replace('/dashboard')
      return
    }
    if (isSettingsMode.value) showToast('success', '保存成功', '配置已保存并立即生效。')
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
.wizard-container { display:flex; width:100vw; height:100vh; justify-content:center; align-items:center; background-color:var(--bg-primary); padding:64px; gap:120px; }
.wizard-container.settings-mode { width:100%; height:100%; }
.left-col { display:flex; flex-direction:column; gap:64px; width:360px; }
.title-box { display:flex; flex-direction:column; gap:12px; }
.main-title { color:var(--text-primary); font-family:var(--font-primary); font-size:40px; font-weight:900; letter-spacing:-1px; }
.sub-title { color:var(--text-secondary); font-family:var(--font-secondary); font-size:20px; font-weight:400; }
.step-list { display:flex; flex-direction:column; gap:32px; }
.step-item { display:flex; align-items:center; gap:16px; }
.step-icon { width:32px; height:32px; border-radius:999px; display:flex; align-items:center; justify-content:center; font-weight:700; }
.step-icon.active { background:#111827; color:#fff; }
.step-icon.completed { background:#16a34a; color:#fff; }
.step-icon.pending { background:#e5e7eb; color:#6b7280; }
.step-text { font-size:15px; }
.active-text { color:var(--text-primary); }
.pending-text { color:var(--text-secondary); }
.right-col { width:min(780px, 100%); background:#fff; border-radius:24px; box-shadow:0 20px 50px rgba(0,0,0,.08); padding:40px; }
.step-content { display:flex; flex-direction:column; gap:28px; }
.header-group { display:flex; flex-direction:column; gap:8px; }
.content-title { font-size:28px; font-weight:800; color:var(--text-primary); }
.content-desc { color:var(--text-secondary); }
.form-group { display:flex; flex-direction:column; gap:18px; }
.input-field { display:flex; flex-direction:column; gap:8px; }
.input-label { font-size:14px; font-weight:700; color:var(--text-primary); }
.input-control { width:100%; border:1px solid #d1d5db; border-radius:14px; padding:12px 14px; font-size:14px; }
.input-hint { color:var(--text-secondary); font-size:13px; }
.success-hint { color:#15803d; }
.path-input-row, .buttons, .buttons.space-between, .settings-inline-actions { display:flex; align-items:center; gap:12px; flex-wrap:wrap; }
.buttons.space-between { justify-content:space-between; }
.btn { border:none; border-radius:14px; padding:12px 18px; cursor:pointer; font-weight:700; }
.btn.primary { background:#111827; color:#fff; }
.btn.secondary { background:#eef2ff; color:#111827; }
.validation-error { color:#dc2626; font-size:14px; }
.radio-group { display:flex; flex-direction:column; gap:12px; }
.radio-option { display:flex; gap:12px; border:1px solid #d1d5db; border-radius:16px; padding:14px; cursor:pointer; }
.radio-option.active { border-color:#111827; background:#f9fafb; }
.radio-option-text { display:flex; flex-direction:column; gap:4px; }
.toggle-field { display:flex; align-items:center; justify-content:space-between; gap:18px; border:1px solid #e5e7eb; border-radius:18px; padding:16px; cursor:pointer; }
.toggle-info { display:flex; flex-direction:column; gap:6px; }
.switch { width:52px; height:30px; border-radius:999px; background:#d1d5db; position:relative; transition:.2s; }
.switch.active { background:#111827; }
.thumb { width:24px; height:24px; border-radius:999px; background:#fff; position:absolute; top:3px; left:3px; transition:.2s; }
.switch.active .thumb { left:25px; }
.modal-overlay { position:fixed; inset:0; background:rgba(17,24,39,.45); display:flex; align-items:center; justify-content:center; padding:24px; z-index:50; }
.modal { width:min(760px, 100%); max-height:min(88vh, 920px); background:#fff; border-radius:24px; overflow:hidden; display:flex; flex-direction:column; }
.modal-header, .modal-footer { padding:20px 24px; border-bottom:1px solid #f3f4f6; }
.modal-footer { border-top:1px solid #f3f4f6; border-bottom:none; display:flex; align-items:center; justify-content:space-between; gap:16px; }
.modal-body { padding:20px 24px; display:flex; flex-direction:column; gap:16px; overflow:auto; flex:1; min-height:0; }
.btn-icon, .breadcrumb-edit-btn { border:none; background:none; cursor:pointer; }
.breadcrumb-bar { display:flex; align-items:center; justify-content:space-between; gap:10px; padding:10px 12px; background:#f9fafb; border-radius:14px; }
.breadcrumb-inner { display:flex; align-items:center; gap:8px; flex-wrap:wrap; }
.breadcrumb-item { border:none; background:none; cursor:pointer; color:#111827; display:flex; align-items:center; gap:6px; }
.breadcrumb-sep { color:#9ca3af; }
.folder-list { min-height:240px; max-height:48vh; border:1px solid #e5e7eb; border-radius:16px; padding:12px; overflow:hidden; }
.folder-scroll { display:flex; flex-direction:column; gap:8px; max-height:calc(48vh - 24px); overflow-y:auto; }
.folder-item { display:flex; align-items:center; gap:10px; padding:10px 12px; border-radius:12px; cursor:pointer; }
.folder-item:hover { background:#f3f4f6; }
.folder-empty { min-height:200px; display:flex; align-items:center; justify-content:center; gap:10px; color:#6b7280; }
.selected-path-preview { display:flex; align-items:center; gap:8px; color:#111827; font-size:14px; }
.modal-footer-btns { display:flex; gap:10px; }
.spin-icon { animation:spin 1s linear infinite; }
.qrcode-panel { display:flex; flex-direction:column; gap:10px; align-items:flex-start; }
.qrcode-image { width:220px; height:220px; object-fit:contain; border:1px solid #e5e7eb; border-radius:16px; padding:12px; background:#fff; }
@keyframes spin { from { transform:rotate(0deg) } to { transform:rotate(360deg) } }
@media (max-width: 1100px) {
  .wizard-container { flex-direction:column; height:auto; min-height:100vh; gap:32px; padding:24px; }
  .left-col, .right-col { width:100%; }
}
</style>
