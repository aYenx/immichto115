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

export interface AppConfig {
  server: ServerConfig
  webdav: WebDAVConfig
  backup: BackupConfig
  encrypt: EncryptConfig
  cron: CronConfig
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

  return `Request failed (${res.status})`
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

  testWebDAV: async (data: { url: string; user: string; password: string }): Promise<{ success: boolean; message: string }> => {
    return await requestJSON<{ success: boolean; message: string }>(`${BASE_URL}/webdav/test`, {
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
  }
}
