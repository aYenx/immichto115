import type { AppConfig } from './api'

/**
 * 创建默认的应用配置。
 * Settings.vue 和 Wizard.vue 共用此函数，确保默认值始终一致。
 */
export function createDefaultConfig(): AppConfig {
  return {
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
      mode: 'copy' as 'copy' | 'sync',
    },
    encrypt: {
      enabled: false,
      password: '',
      salt: '',
    },
    cron: {
      enabled: true,
      expression: '0 3 * * *',
    },
    notify: {
      enabled: false,
      bark_url: '',
    },
  }
}
