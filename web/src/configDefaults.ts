import type { AppConfig } from './api'

/**
 * 创建默认的应用配置。
 * Settings.vue 和 Wizard.vue 共用此函数，确保默认值始终一致。
 */
export function createDefaultConfig(): AppConfig {
  return {
    provider: 'webdav',
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
    open115: {
      enabled: false,
      client_id: '',
      access_token: '',
      refresh_token: '',
      root_id: '0',
      token_expires_at: 0,
      user_id: '',
    },
    backup: {
      library_dir: '',
      backups_dir: '',
      remote_dir: '/immich-backup',
      mode: 'copy' as 'copy' | 'sync',
    },
    encrypt: {
      enabled: false,
      password: '',
      salt: '',
    },
    cron: {
      enabled: false,
      expression: '0 2 * * *',
    },
    notify: {
      enabled: false,
      bark_url: '',
    },
  }
}
