import { showToast } from './composables/toast'

const BASE_URL = '/api/v1'
const AUTH_ERROR_MESSAGE = '认证已失效，请刷新页面并重新输入管理员账号密码'

let authRecoveryTriggered = false
let csrfRecoveryInProgress = false

// ---------------------------------------------------------------------------
// CSRF token state — managed by login / recovery lifecycle
// ---------------------------------------------------------------------------
let csrfToken: string | null = null

export function setCsrfToken(token: string | null) {
  csrfToken = token
}

export function getCsrfToken(): string | null {
  return csrfToken
}

export interface ServerConfig {
  port: number
  auth_enabled: boolean
  auth_user: string
  auth_password?: string
}

export interface WebDAVConfig {
  url: string
  user: string
  password: string
  vendor?: string
}

export interface WebDAVListRequest {
  url: string
  user: string
  password: string
  vendor?: string
  path: string
}

export interface BackupConfig {
  library_dir: string
  backups_dir: string
  remote_dir: string
  mode: 'copy' | 'sync'
  manifest_path?: string
  allow_remote_delete?: boolean
}

export interface EncryptConfig {
  enabled: boolean
  password: string
  salt: string
}

export interface CronConfig {
  enabled: boolean
  expression: string
}

export interface Open115Config {
  enabled: boolean
  client_id: string
  access_token: string
  refresh_token: string
  root_id: string
  token_expires_at: number
  user_id: string
}

export interface Open115EncryptConfig {
  enabled: boolean
  password: string
  salt: string
  mode: string
  filename_mode: string
  algorithm: string
  temp_dir: string
  min_free_space_mb: number
}

export interface PhotoUploadConfig {
  enabled: boolean
  watch_dir: string
  remote_dir: string
  extensions: string
  date_format: string
  delete_after_upload: boolean
}

export interface NotifyConfig {
  enabled: boolean
  bark_url: string
}

// ---------------------------------------------------------------------------
// AppConfig — 本地编辑草稿类型（Vue 组件内部使用）
// ---------------------------------------------------------------------------
export interface AppConfig {
  provider: 'webdav' | 'open115'
  server: ServerConfig
  webdav: WebDAVConfig
  open115: Open115Config
  open115_encrypt: Open115EncryptConfig
  backup: BackupConfig
  encrypt: EncryptConfig
  cron: CronConfig
  notify: NotifyConfig
  photo_upload: PhotoUploadConfig
  updated_at?: number
}

// ---------------------------------------------------------------------------
// SafeConfigResponse — 后端返回的安全视图（敏感字段用 has_* 布尔值）
// ---------------------------------------------------------------------------
export interface SafeServerConfig {
  port: number
  auth_enabled: boolean
  auth_user: string
  has_auth_password: boolean
}

export interface SafeWebDAVConfig {
  url: string
  user: string
  has_password: boolean
  vendor: string
}

export interface SafeOpen115Config {
  enabled: boolean
  client_id: string
  has_access_token: boolean
  has_refresh_token: boolean
  root_id: string
  token_expires_at: number
  user_id: string
}

export interface SafeOpen115EncryptConfig {
  enabled: boolean
  has_password: boolean
  has_salt: boolean
  mode: string
  filename_mode: string
  algorithm: string
  temp_dir: string
  min_free_space_mb: number
}

export interface SafeEncryptConfig {
  enabled: boolean
  has_password: boolean
  has_salt: boolean
}

export interface SafeConfigResponse {
  provider: 'webdav' | 'open115'
  server: SafeServerConfig
  webdav: SafeWebDAVConfig
  open115: SafeOpen115Config
  open115_encrypt: SafeOpen115EncryptConfig
  backup: BackupConfig
  encrypt: SafeEncryptConfig
  cron: CronConfig
  notify: NotifyConfig
  photo_upload: PhotoUploadConfig
  updated_at: number
}

// ---------------------------------------------------------------------------
// ConfigUpdateRequest — 保存配置的请求体（敏感字段可选，null=保留旧值）
// ---------------------------------------------------------------------------
export interface ServerUpdateRequest {
  port: number
  auth_enabled: boolean
  auth_user: string
  password?: string | null  // null/undefined=keep, ""=clear, value=set
}

export interface WebDAVUpdateRequest {
  url: string
  user: string
  password?: string | null
  vendor: string
}

export interface Open115UpdateRequest {
  enabled: boolean
  client_id: string
  access_token?: string | null
  refresh_token?: string | null
  root_id: string
  token_expires_at: number
  user_id: string
}

export interface Open115EncryptUpdateRequest {
  enabled: boolean
  password?: string | null
  salt?: string | null
  mode: string
  filename_mode: string
  algorithm: string
  temp_dir: string
  min_free_space_mb: number
}

export interface EncryptUpdateRequest {
  enabled: boolean
  password?: string | null
  salt?: string | null
}

export interface ConfigUpdateRequest {
  provider: 'webdav' | 'open115'
  server: ServerUpdateRequest
  webdav: WebDAVUpdateRequest
  open115: Open115UpdateRequest
  open115_encrypt: Open115EncryptUpdateRequest
  backup: BackupConfig
  encrypt: EncryptUpdateRequest
  cron: CronConfig
  notify: NotifyConfig
  photo_upload: PhotoUploadConfig
  updated_at: number
}

// ---------------------------------------------------------------------------
// Helper: SafeConfigResponse → AppConfig 用于初始化编辑草稿
// ---------------------------------------------------------------------------
export function safeConfigToAppConfig(safe: SafeConfigResponse): AppConfig {
  return {
    provider: safe.provider,
    server: {
      port: safe.server.port,
      auth_enabled: safe.server.auth_enabled,
      auth_user: safe.server.auth_user,
      // 不设置密码 — 用户需要主动输入新密码才会发送
    },
    webdav: {
      url: safe.webdav.url,
      user: safe.webdav.user,
      password: '', // 不回传密码，编辑时留空表示保留
      vendor: safe.webdav.vendor,
    },
    open115: {
      enabled: safe.open115.enabled,
      client_id: safe.open115.client_id,
      access_token: '',
      refresh_token: '',
      root_id: safe.open115.root_id,
      token_expires_at: safe.open115.token_expires_at,
      user_id: safe.open115.user_id,
    },
    open115_encrypt: {
      enabled: safe.open115_encrypt.enabled,
      password: '',
      salt: '',
      mode: safe.open115_encrypt.mode,
      filename_mode: safe.open115_encrypt.filename_mode,
      algorithm: safe.open115_encrypt.algorithm,
      temp_dir: safe.open115_encrypt.temp_dir,
      min_free_space_mb: safe.open115_encrypt.min_free_space_mb,
    },
    backup: { ...safe.backup },
    encrypt: {
      enabled: safe.encrypt.enabled,
      password: '',
      salt: '',
    },
    cron: { ...safe.cron },
    notify: { ...safe.notify },
    photo_upload: { ...safe.photo_upload },
    updated_at: safe.updated_at,
  }
}

// ---------------------------------------------------------------------------
// Helper: AppConfig (draft) → ConfigUpdateRequest
// 只有用户修改过的敏感字段才会作为非 null 提交
// ---------------------------------------------------------------------------
export function appConfigToUpdateRequest(
  draft: AppConfig,
  safe: SafeConfigResponse
): ConfigUpdateRequest {
  return {
    provider: draft.provider,
    server: {
      port: draft.server.port,
      auth_enabled: draft.server.auth_enabled,
      auth_user: draft.server.auth_user,
      // 密码为空且已有密码 → null（保留旧值）；密码非空 → 设置新值
      password: draft.server.auth_password || safe.server.has_auth_password
        ? (draft.server.auth_password || null)
        : null,
    },
    webdav: {
      url: draft.webdav.url,
      user: draft.webdav.user,
      password: draft.webdav.password || safe.webdav.has_password
        ? (draft.webdav.password || null)
        : null,
      vendor: draft.webdav.vendor || 'other',
    },
    open115: {
      enabled: draft.open115.enabled,
      client_id: draft.open115.client_id,
      access_token: draft.open115.access_token || safe.open115.has_access_token
        ? (draft.open115.access_token || null)
        : null,
      refresh_token: draft.open115.refresh_token || safe.open115.has_refresh_token
        ? (draft.open115.refresh_token || null)
        : null,
      root_id: draft.open115.root_id,
      token_expires_at: draft.open115.token_expires_at,
      user_id: draft.open115.user_id,
    },
    open115_encrypt: {
      enabled: draft.open115_encrypt.enabled,
      password: draft.open115_encrypt.password || safe.open115_encrypt.has_password
        ? (draft.open115_encrypt.password || null)
        : null,
      salt: draft.open115_encrypt.salt || safe.open115_encrypt.has_salt
        ? (draft.open115_encrypt.salt || null)
        : null,
      mode: draft.open115_encrypt.mode,
      filename_mode: draft.open115_encrypt.filename_mode,
      algorithm: draft.open115_encrypt.algorithm,
      temp_dir: draft.open115_encrypt.temp_dir,
      min_free_space_mb: draft.open115_encrypt.min_free_space_mb,
    },
    backup: { ...draft.backup },
    encrypt: {
      enabled: draft.encrypt.enabled,
      password: draft.encrypt.password || safe.encrypt.has_password
        ? (draft.encrypt.password || null)
        : null,
      salt: draft.encrypt.salt || safe.encrypt.has_salt
        ? (draft.encrypt.salt || null)
        : null,
    },
    cron: { ...draft.cron },
    notify: { ...draft.notify },
    photo_upload: { ...draft.photo_upload },
    updated_at: draft.updated_at || 0,
  }
}

export interface WebDAVTestResponse {
  success: boolean
  message: string
}

export interface Open115AuthStartResponse {
  uid: string
  time: number
  sign: string
  qrcode: string
  created_at: string
}

export interface Open115AuthStatusResponse {
  status: number
  message: string
  authorized: boolean
}

export interface Open115AuthFinishResponse {
  message: string
  state: {
    enabled: boolean
    client_id: string
    root_id: string
    token_expires_at: number
    user_id: string
  }
}

export interface Open115TestResponse {
  success: boolean
  message: string
}

export interface SystemStatus {
  provider?: 'webdav' | 'open115'
  rclone_installed: boolean
  rclone_version: string
  backup_status: 'idle' | 'running'
  cron_enabled: boolean
  next_run: string | null
  setup_complete: boolean
  version?: string
  commit?: string
  build_time?: string
  dirty?: boolean
}

/** Represents a file or directory entry from backend listing APIs */
export interface DirEntry {
  Name: string
  Path: string
  IsDir: boolean
  Size?: number
  ModTime?: string
}

export class ApiError extends Error {
  status: number

  constructor(status: number, message: string) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

// ---------------------------------------------------------------------------
// Gallery (云盘相册) Types
// ---------------------------------------------------------------------------
export interface GalleryEntry {
  id: string
  name: string
  path: string
  is_dir: boolean
  size: number
  mod_time: string
  pick_code?: string
  thumbnail?: string
  original_url?: string
  file_type?: string
}

export interface GalleryListResponse {
  items: GalleryEntry[]
  total: number
  dir_id: string
}

export interface GalleryDownloadResponse {
  url: string
  file_name: string
  file_size: number
}

async function readErrorMessage(res: Response): Promise<string> {
  if (res.status === 401) {
    return AUTH_ERROR_MESSAGE
  }

  // Read body as text first, then attempt JSON parse to avoid double-consumption
  const text = (await res.text().catch(() => '')).trim()

  if (text) {
    try {
      const payload = JSON.parse(text) as Record<string, unknown>
      const message = payload?.error || payload?.message
      if (typeof message === 'string' && message.trim()) {
        return message
      }
    } catch {
      // Not JSON, use as plain text
    }
    return text
  }

  return `请求失败（${res.status}）`
}

async function requestJSON<T>(url: string, init?: RequestInit): Promise<T> {
  // Auto-inject CSRF token for mutating methods when using JWT session
  const method = init?.method?.toUpperCase()
  if (csrfToken && method && ['POST', 'PUT', 'PATCH', 'DELETE'].includes(method)) {
    const headers = new Headers(init?.headers)
    if (!headers.has('X-CSRF-Token')) {
      headers.set('X-CSRF-Token', csrfToken)
    }
    init = { ...init, headers }
  }

  const res = await fetch(url, init)
  if (!res.ok) {
    // Read body once for both CSRF detection and error message
    const bodyText = (await res.text().catch(() => '')).trim()

    // 403 + CSRF mismatch → one-shot recovery
    if (res.status === 403 && csrfToken !== null && bodyText.includes('CSRF') && !csrfRecoveryInProgress) {
      csrfRecoveryInProgress = true
      try {
        await api.getCSRFToken()
      } catch { /* no active session */ }
      csrfRecoveryInProgress = false

      if (csrfToken) {
        // Retry once with refreshed token
        const retryHeaders = new Headers(init?.headers)
        retryHeaders.set('X-CSRF-Token', csrfToken)
        const retryRes = await fetch(url, { ...init, headers: retryHeaders })
        if (!retryRes.ok) {
          throw new ApiError(retryRes.status, await readErrorMessage(retryRes))
        }
        return await retryRes.json() as T
      }
    }

    // Build error from already-consumed body text
    let errorMessage: string
    if (res.status === 401) {
      errorMessage = AUTH_ERROR_MESSAGE
    } else if (bodyText) {
      try {
        const payload = JSON.parse(bodyText) as Record<string, unknown>
        const msg = payload?.error || payload?.message
        errorMessage = (typeof msg === 'string' && msg.trim()) ? msg : bodyText
      } catch {
        errorMessage = bodyText
      }
    } else {
      errorMessage = `请求失败（${res.status}）`
    }
    throw new ApiError(res.status, errorMessage)
  }
  return await res.json() as T
}

export function getErrorMessage(error: unknown): string {
  if (error instanceof Error && error.message) {
    return error.message
  }
  return '请求失败，请稍后重试'
}

export function handleAuthFailure(error: unknown): boolean {
  if (!(error instanceof ApiError) || error.status !== 401) {
    return false
  }

  if (!authRecoveryTriggered && typeof window !== 'undefined') {
    authRecoveryTriggered = true
    showToast('warning', '认证已失效', AUTH_ERROR_MESSAGE, 2200)
    window.setTimeout(() => {
      window.location.reload()
    }, 250)
  }

  return true
}

export const api = {
  getSystemStatus: async (): Promise<SystemStatus> => {
    return await requestJSON<SystemStatus>(`${BASE_URL}/system/status`)
  },

  getConfig: async (): Promise<SafeConfigResponse> => {
    return await requestJSON<SafeConfigResponse>(`${BASE_URL}/config`)
  },

  saveConfig: async (req: ConfigUpdateRequest): Promise<{ message: string; updated_at: number }> => {
    return await requestJSON<{ message: string; updated_at: number }>(`${BASE_URL}/config`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(req)
    })
  },

  testWebDAV: async (data: { url: string; user: string; password: string; vendor?: string }): Promise<WebDAVTestResponse> => {
    return await requestJSON<WebDAVTestResponse>(`${BASE_URL}/webdav/test`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
  },

  listWebDAV: async (data: WebDAVListRequest): Promise<DirEntry[]> => {
    return await requestJSON<DirEntry[]>(`${BASE_URL}/webdav/ls`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
  },

  open115AuthStart: async (data: { client_id: string }): Promise<Open115AuthStartResponse> => {
    return await requestJSON<Open115AuthStartResponse>(`${BASE_URL}/open115/auth/start`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
  },

  open115AuthStatus: async (uid: string): Promise<Open115AuthStatusResponse> => {
    return await requestJSON<Open115AuthStatusResponse>(`${BASE_URL}/open115/auth/status?uid=${encodeURIComponent(uid)}`)
  },

  open115AuthFinish: async (data: { uid: string }): Promise<Open115AuthFinishResponse> => {
    return await requestJSON<Open115AuthFinishResponse>(`${BASE_URL}/open115/auth/finish`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
  },

  open115Test: async (tokens?: { access_token: string; refresh_token: string }): Promise<Open115TestResponse> => {
    return await requestJSON<Open115TestResponse>(`${BASE_URL}/open115/test`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: tokens ? JSON.stringify(tokens) : undefined
    })
  },

  open115List: async (path: string, tokens?: { access_token: string; refresh_token: string; root_id?: string }): Promise<DirEntry[]> => {
    const body: Record<string, string> = { path }
    if (tokens) {
      body.access_token = tokens.access_token
      body.refresh_token = tokens.refresh_token
      if (tokens.root_id) {
        body.root_id = tokens.root_id
      }
    }
    return await requestJSON<DirEntry[]>(`${BASE_URL}/open115/ls`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    })
  },

  startBackup: async (): Promise<{ message: string }> => {
    return await requestJSON<{ message: string }>(`${BASE_URL}/backup/start`, { method: 'POST' })
  },

  stopBackup: async (): Promise<{ message: string }> => {
    return await requestJSON<{ message: string }>(`${BASE_URL}/backup/stop`, { method: 'POST' })
  },

  listRemote: async (path: string): Promise<DirEntry[]> => {
    return await requestJSON<DirEntry[]>(`${BASE_URL}/remote/ls?path=${encodeURIComponent(path)}`)
  },

  listLocal: async (path: string = ''): Promise<DirEntry[]> => {
    const params = new URLSearchParams()
    if (path) {
      params.append('path', path)
    }
    return await requestJSON<DirEntry[]>(`${BASE_URL}/local/ls?${params.toString()}`)
  },

  testNotify: async (): Promise<{ message: string }> => {
    return await requestJSON<{ message: string }>(`${BASE_URL}/notify/test`, { method: 'POST' })
  },

  // 摄影文件上传
  photoUploadStart: async (): Promise<{ message: string }> => {
    return await requestJSON<{ message: string }>(`${BASE_URL}/photo-upload/start`, { method: 'POST' })
  },

  photoUploadStop: async (): Promise<{ message: string }> => {
    return await requestJSON<{ message: string }>(`${BASE_URL}/photo-upload/stop`, { method: 'POST' })
  },

  photoUploadStatus: async (): Promise<{ status: string }> => {
    return await requestJSON<{ status: string }>(`${BASE_URL}/photo-upload/status`)
  },

  // --- Auth (JWT session) ---

  login: async (username: string, password: string): Promise<{ csrf_token: string; expires_at: number }> => {
    const resp = await requestJSON<{ csrf_token: string; expires_at: number }>(`${BASE_URL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username, password }),
    })
    setCsrfToken(resp.csrf_token)
    return resp
  },

  logout: async (): Promise<{ message: string }> => {
    const resp = await requestJSON<{ message: string }>(`${BASE_URL}/auth/logout`, {
      method: 'POST',
    })
    setCsrfToken(null)
    return resp
  },

  getCSRFToken: async (): Promise<{ csrf_token: string }> => {
    const resp = await requestJSON<{ csrf_token: string }>(`${BASE_URL}/auth/csrf`)
    setCsrfToken(resp.csrf_token)
    return resp
  },

  // --- Gallery (云盘相册) ---

  galleryList: async (path: string, dirId?: string, offset = 0, limit = 50): Promise<GalleryListResponse> => {
    return await requestJSON<GalleryListResponse>(`${BASE_URL}/gallery/ls`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ path, dir_id: dirId || '', offset, limit }),
    })
  },

  galleryProxyURL: (rawUrl: string, type: 'thumb' | 'original' = 'thumb'): string => {
    return `${BASE_URL}/gallery/proxy?url=${encodeURIComponent(rawUrl)}&type=${type}`
  },

  galleryDownloadUrl: async (pickCode: string): Promise<GalleryDownloadResponse> => {
    return await requestJSON<GalleryDownloadResponse>(`${BASE_URL}/gallery/download-url`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ pick_code: pickCode }),
    })
  },
}

/**
 * Recover CSRF state from an existing JWT cookie after page refresh.
 * Call once at app startup (e.g. App.vue onMounted or main.ts).
 * If no JWT session exists, this silently no-ops.
 */
export async function initSession(): Promise<void> {
  if (csrfToken) return // already initialized
  try {
    await api.getCSRFToken()
  } catch {
    // no active session — normal for unauthenticated users
  }
}
