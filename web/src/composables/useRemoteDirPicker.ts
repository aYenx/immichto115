import { ref, computed } from 'vue'
import { handleAuthFailure, getErrorMessage, type DirEntry } from '../api'
import { showToast } from './toast'

/** Normalize remote path: ensure leading `/`, collapse multiple slashes. */
export function normalizeRemotePath(p: string): string {
  const cleaned = ('/' + p.replace(/\\/g, '/')).replace(/\/+/g, '/')
  return cleaned || '/'
}

/**
 * 远端目录选择 composable.
 * 封装远端（115 Open / WebDAV）目录浏览逻辑。
 *
 * Caller provides a `listFn` callback at open-time that resolves the
 * correct API call (open115List or listWebDAV) based on current config state.
 */
export function useRemoteDirPicker() {
  const showModal = ref(false)
  const currentPath = ref('/')
  const entries = ref<DirEntry[]>([])
  const loading = ref(false)

  /** The listing function provided by the caller at open-time. */
  let listFn: ((path: string) => Promise<DirEntry[]>) | null = null

  // ---------- Computed ----------

  const pathSegments = computed(() =>
    currentPath.value.split('/').filter(s => s !== '')
  )

  const canGoUp = computed(() => currentPath.value !== '/')

  // ---------- Actions ----------

  const loadDir = async (path: string) => {
    if (!listFn) return
    loading.value = true
    const normalizedPath = normalizeRemotePath(path)
    try {
      const items = await listFn(normalizedPath)
      currentPath.value = normalizedPath
      entries.value = items.filter(i => i.IsDir).sort((a, b) => a.Name.localeCompare(b.Name))
    } catch (err) {
      entries.value = []
      if (handleAuthFailure(err)) return
      showToast('error', '浏览失败', getErrorMessage(err))
    } finally {
      loading.value = false
    }
  }

  /**
   * Open the picker and start browsing.
   * @param initialPath  Starting directory (e.g. current remote_dir value)
   * @param fn           Listing function — `(path) => Promise<DirEntry[]>`
   */
  const open = async (
    initialPath: string,
    fn: (path: string) => Promise<DirEntry[]>,
  ) => {
    listFn = fn
    showModal.value = true
    currentPath.value = normalizeRemotePath(initialPath)
    await loadDir(currentPath.value)
  }

  const close = () => {
    showModal.value = false
    listFn = null
  }

  const enterDir = (entry: DirEntry) => {
    const newPath = normalizeRemotePath(currentPath.value + '/' + entry.Name)
    void loadDir(newPath)
  }

  const goUp = () => {
    const parts = currentPath.value.split('/').filter(Boolean)
    parts.pop()
    const parent = parts.length > 0 ? '/' + parts.join('/') : '/'
    void loadDir(parent)
  }

  const navigateToSegment = (index: number) => {
    const segs = pathSegments.value.slice(0, index + 1)
    void loadDir('/' + segs.join('/'))
  }

  const confirm = (): string => {
    const selected = currentPath.value
    showModal.value = false
    listFn = null
    return selected
  }

  return {
    showModal,
    currentPath,
    entries,
    loading,
    pathSegments,
    canGoUp,
    open,
    close,
    loadDir,
    enterDir,
    goUp,
    navigateToSegment,
    confirm,
  }
}
