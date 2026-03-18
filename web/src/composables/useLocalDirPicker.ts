import { ref, computed } from 'vue'
import { api, handleAuthFailure, getErrorMessage, type DirEntry } from '../api'
import { showToast } from './toast'

/**
 * 本地目录选择 composable.
 * 封装目录浏览、导航（含 Windows 驱动器根）、路径面包屑等逻辑。
 */
export function useLocalDirPicker() {
  const showModal = ref(false)
  const currentPath = ref('')
  const entries = ref<DirEntry[]>([])
  const loading = ref(false)
  const showPathInput = ref(false)

  // ---------- Computed ----------

  const isWindowsPath = computed(() => currentPath.value.includes('\\'))

  const pathSegments = computed(() => {
    const p = currentPath.value
    if (!p) return []
    const sep = p.includes('\\') ? '\\' : '/'
    const parts = p.split(sep).filter(Boolean)
    // On Unix, paths start with '/' so first segment should be kept as-is
    return parts
  })

  const canGoUp = computed(() => {
    const p = currentPath.value
    return p !== '/' && p !== ''
  })

  // ---------- Actions ----------

  const loadDir = async (path: string) => {
    loading.value = true
    try {
      const items = await api.listLocal(path)
      entries.value = items.filter(i => i.IsDir).sort((a, b) => a.Name.localeCompare(b.Name))
      if (path === '') {
        currentPath.value = ''
      }
    } catch (err) {
      entries.value = []
      if (handleAuthFailure(err)) return
      showToast('error', '加载目录失败', getErrorMessage(err))
    } finally {
      loading.value = false
    }
  }

  const open = async (initialPath: string = '') => {
    showModal.value = true
    showPathInput.value = false
    currentPath.value = initialPath
    await loadDir(initialPath)
  }

  const close = () => {
    showModal.value = false
    showPathInput.value = false
  }

  const resolveEntryPath = (item: DirEntry): string => {
    const candidate = (item.Path || '').trim()
    if (candidate.startsWith('/') || /^[A-Za-z]:[/\\]/.test(candidate)) {
      return candidate
    }
    const sep = currentPath.value.includes('\\') ? '\\' : '/'
    let newPath = currentPath.value
    if (newPath === '' || newPath.endsWith(sep)) newPath += item.Name
    else newPath += sep + item.Name
    return newPath
  }

  const enterDir = (item: DirEntry) => {
    const newPath = resolveEntryPath(item)
    currentPath.value = newPath
    void loadDir(newPath)
  }

  const goUp = () => {
    const sep = currentPath.value.includes('\\') ? '\\' : '/'
    const parts = currentPath.value.split(sep)
    // Remove trailing empty segment (e.g. "C:\foo\" → ["C:", "foo", ""])
    if (parts.length > 0 && parts[parts.length - 1] === '') parts.pop()
    parts.pop()
    let newPath = parts.join(sep)

    // On Windows, empty result means we were already at drive root → go to drive list
    if (sep === '\\' && newPath === '') {
      currentPath.value = ''
      void loadDir('')
      return
    }

    // Normalize Windows drive-only path "C:" → "C:\"
    if (sep === '\\' && /^[A-Za-z]:$/.test(newPath)) newPath += '\\'
    if (newPath === '' || (sep === '\\' && !newPath.includes('\\'))) newPath += sep

    currentPath.value = newPath
    void loadDir(newPath)
  }

  const navigateToSegment = (index: number) => {
    const sep = isWindowsPath.value ? '\\' : '/'
    const segs = pathSegments.value.slice(0, index + 1)
    let newPath = segs.join(sep)
    if (isWindowsPath.value) {
      if (!newPath.endsWith('\\')) newPath += '\\'
    } else {
      newPath = '/' + newPath
    }
    currentPath.value = newPath
    void loadDir(newPath)
  }

  const confirm = (): string => {
    const selected = currentPath.value
    showModal.value = false
    showPathInput.value = false
    return selected
  }

  return {
    showModal,
    currentPath,
    entries,
    loading,
    showPathInput,
    isWindowsPath,
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
