const BASE = '/api/v1'

export interface SystemStatus {
  rclone_installed: boolean
  rclone_version: string
  backup_status: 'idle' | 'running'
  cron_enabled: boolean
  next_run: string
  setup_complete: boolean
}

export interface AppConfig {
  server: { port: number }
  webdav: { url: string; user: string; password: string; vendor: string }
  backup: { library_dir: string; backups_dir: string; remote_dir: string }
  encrypt: { enabled: boolean; password: string; salt: string }
  cron: { enabled: boolean; expression: string }
}

export interface WebDAVTestResult {
  success: boolean
  message: string
}

export interface RemoteFile {
  Path: string
  Name: string
  Size: number
  MimeType: string
  ModTime: string
  IsDir: boolean
}

async function request<T>(url: string, options?: RequestInit): Promise<T> {
  const res = await fetch(BASE + url, {
    headers: { 'Content-Type': 'application/json' },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error || res.statusText)
  }
  return res.json()
}

export const api = {
  // 系统状态
  getStatus: () => request<SystemStatus>('/system/status'),

  // 配置管理
  getConfig: () => request<AppConfig>('/config'),
  saveConfig: (config: AppConfig) =>
    request<{ message: string }>('/config', {
      method: 'POST',
      body: JSON.stringify(config),
    }),

  // WebDAV 测试
  testWebDAV: (data: { url: string; user: string; password: string }) =>
    request<WebDAVTestResult>('/webdav/test', {
      method: 'POST',
      body: JSON.stringify(data),
    }),

  // 备份控制
  startBackup: () =>
    request<{ message: string }>('/backup/start', { method: 'POST' }),
  stopBackup: () =>
    request<{ message: string }>('/backup/stop', { method: 'POST' }),

  // 云端文件浏览
  listRemote: (path: string = '') =>
    request<RemoteFile[]>(`/remote/ls?path=${encodeURIComponent(path)}`),
}
