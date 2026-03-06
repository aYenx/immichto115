const BASE_URL = '/api/v1'

export interface SystemStatus {
  rclone_installed: boolean
  rclone_version: string
  backup_status: 'idle' | 'running'
  cron_enabled: boolean
  next_run: string | null
  setup_complete: boolean
}

export const api = {
  getSystemStatus: async (): Promise<SystemStatus> => {
    const res = await fetch(`${BASE_URL}/system/status`)
    if (!res.ok) throw new Error(await res.text())
    return (await res.json()) as SystemStatus
  },

  getConfig: async (): Promise<any> => {
    const res = await fetch(`${BASE_URL}/config`)
    if (!res.ok) throw new Error(await res.text())
    return await res.json()
  },

  saveConfig: async (config: any): Promise<any> => {
    const res = await fetch(`${BASE_URL}/config`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(config)
    })
    if (!res.ok) throw new Error(await res.text())
    return await res.json()
  },

  testWebDAV: async (data: { url: string; user: string; password: string }): Promise<{ success: boolean; message: string }> => {
    const res = await fetch(`${BASE_URL}/webdav/test`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data)
    })
    if (!res.ok) {
      const resp = await res.json()
      throw new Error(resp.error || 'Request failed')
    }
    return await res.json()
  },

  startBackup: async (): Promise<any> => {
    const res = await fetch(`${BASE_URL}/backup/start`, { method: 'POST' })
    if (!res.ok) {
      const resp = await res.json()
      throw new Error(resp.error || 'Failed to start backup')
    }
    return await res.json()
  },

  stopBackup: async (): Promise<any> => {
    const res = await fetch(`${BASE_URL}/backup/stop`, { method: 'POST' })
    if (!res.ok) {
      const resp = await res.json()
      throw new Error(resp.error || 'Failed to stop backup')
    }
    return await res.json()
  },

  listRemote: async (path: string): Promise<any> => {
    const res = await fetch(`${BASE_URL}/remote/ls?path=${encodeURIComponent(path)}`)
    if (!res.ok) {
      throw new Error('Failed to fetch remote directory')
    }
    return await res.json()
  },

  listLocal: async (path: string = ''): Promise<any> => {
    const params = new URLSearchParams()
    if (path) {
      params.append('path', path)
    }
    const res = await fetch(`${BASE_URL}/local/ls?${params.toString()}`)
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || 'Failed to list local directory')
    }
    return await res.json()
  }
}
