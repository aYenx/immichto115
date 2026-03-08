import { showToast } from './composables/toast'

const BASE_URL = '/api/v1'
const AUTH_ERROR_MESSAGE = '认证已失效，请刷新页面并重新输入管理员账号密码'

let authRecoveryTriggered = false

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

export interface AppConfig {
  provider: 'webdav' | 'open115'
  server: ServerConfig
  webdav: WebDAVConfig
  open115: Open115Config
  backup: BackupConfig
  encrypt: EncryptConfig
  cron: CronConfig
  notify: NotifyConfig
}

export interface NotifyConfig {
  enabled: boolean
  bark_url: string
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
  state: Open115Config
}

export interface Open115TestResponse {
  success: boolean
  message: string
}

export interface SystemStatus {
  rclone_installed: boolean
  rclone_version: string
  backup_status: 'idle' | 'running'
  cron_enabled: boolean
  next_run: string | null
  setup_complete: boolean
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
  const res = await fetch(url, init)
  if (!res.ok) {
    throw new ApiError(res.status, await readErrorMessage(res))
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
      authRecoveryTriggered = false // Reset flag so future 401s can be handled
      window.location.reload()
    }, 250)
  }

  return true
}

export const api = {
  getSystemStatus: async (): Promise<SystemStatus> => {
    return await requestJSON<SystemStatus>(`${BASE_URL}/system/status`)
  },

  getConfig: async (): Promise<AppConfig> => {
    return await requestJSON<AppConfig>(`${BASE_URL}/config`)
  },

  saveConfig: async (config: AppConfig): Promise<{ message: string }> => {
    return await requestJSON<{ message: string }>(`${BASE_URL}/config`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    })
  },

  testWebDAV: async (data: { url: string; user: string; password: string }): Promise<WebDAVTestResponse> => {
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

  open115Test: async (): Promise<Open115TestResponse> => {
    return await requestJSON<Open115TestResponse>(`${BASE_URL}/open115/test`, { method: 'POST' })
  },

  open115List: async (path: string): Promise<DirEntry[]> => {
    return await requestJSON<DirEntry[]>(`${BASE_URL}/open115/ls?path=${encodeURIComponent(path)}`)
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
  }
}
