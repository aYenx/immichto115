/**
 * Frontend state constants — eliminate magic string proliferation.
 */

/** Backup job status reported by /api/v1/system/status */
export const BackupStatus = {
  Idle: 'idle',
  Running: 'running',
} as const

export type BackupStatusType = (typeof BackupStatus)[keyof typeof BackupStatus]

/** Storage provider type */
export const Provider = {
  WebDAV: 'webdav',
  Open115: 'open115',
} as const

export type ProviderType = (typeof Provider)[keyof typeof Provider]
