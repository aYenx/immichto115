<template>
  <div class="gallery-page">
    <!-- Provider gate -->
    <div v-if="providerError" class="gallery-unsupported">
      <LucideShieldAlert :size="48" />
      <h2>功能不可用</h2>
      <p>云盘相册仅支持 115 Open 模式，请在设置中切换到 open115 后使用。</p>
    </div>

    <template v-else>
      <!-- Header -->
      <div class="gallery-header">
        <div class="gallery-title-row">
          <h1 class="gallery-title">
            <LucideImage :size="28" />
            云盘相册
          </h1>
          <div class="gallery-actions">
            <span v-if="total >= 0" class="item-count">{{ total }} 项</span>
            <button class="action-btn" @click="refresh" :disabled="loading" title="刷新">
              <LucideRefreshCw :size="18" :class="{ spinning: loading }" />
            </button>
            <button class="action-btn" @click="showSettings = !showSettings" title="设置">
              <LucideSettings :size="18" />
            </button>
          </div>

          <!-- Settings popover -->
          <Transition name="popover">
            <div v-if="showSettings" class="settings-popover">
              <label class="settings-label">浏览根目录</label>
              <p class="settings-hint">设置Gallery默认浏览的目录路径</p>
              <div class="settings-input-row">
                <input
                  v-model="galleryRootInput"
                  type="text"
                  class="settings-input"
                  placeholder="/ (默认根目录)"
                  @keydown.enter="applyGalleryRoot"
                />
                <button class="settings-apply-btn" @click="openDirBrowser">浏览…</button>
                <button class="settings-apply-btn" @click="applyGalleryRoot">应用</button>
              </div>
            </div>
          </Transition>
        </div>

        <!-- Breadcrumb -->
        <nav class="breadcrumb" v-if="pathSegments.length > 0">
          <button class="crumb" @click="navigateTo('/')">
            <LucideHome :size="14" />
            <span>根目录</span>
          </button>
          <template v-for="(seg, i) in pathSegments" :key="i">
            <LucideChevronRight :size="14" class="crumb-sep" />
            <button
              class="crumb"
              :class="{ current: i === pathSegments.length - 1 }"
              @click="navigateToSegment(i)"
            >{{ seg }}</button>
          </template>
        </nav>
      </div>

      <!-- Loading -->
      <div v-if="loading && items.length === 0" class="gallery-loading">
        <div class="loading-spinner"></div>
        <p>加载中…</p>
      </div>

      <!-- Error -->
      <div v-else-if="error" class="gallery-error">
        <LucideAlertCircle :size="40" />
        <p>{{ error }}</p>
        <button class="retry-btn" @click="refresh">重试</button>
      </div>

      <!-- Empty -->
      <div v-else-if="items.length === 0 && !loading" class="gallery-empty">
        <LucideImageOff :size="48" />
        <h3>空目录</h3>
        <p>此目录下没有图片或文件夹</p>
      </div>

      <!-- Grid -->
      <div v-else class="gallery-grid" ref="gridRef">
        <div
          v-for="(item, index) in items"
          :key="item.id"
          class="gallery-card"
          :class="{ folder: item.is_dir }"
          @click="onCardClick(item, index)"
        >
          <!-- Folder -->
          <template v-if="item.is_dir">
            <div class="card-thumb folder-icon">
              <LucideFolder :size="40" />
            </div>
            <div class="card-info">
              <span class="card-name" :title="item.name">{{ item.name }}</span>
            </div>
          </template>

          <!-- Previewable image -->
          <template v-else-if="isPreviewable(item)">
            <div class="card-thumb">
              <img
                :data-src="getCardImagePrimaryUrl(item)"
                :data-fallback-src="getCardImageFallbackUrl(item)"
                :alt="item.name"
                class="thumb-img lazy"
                loading="lazy"
                @load="onThumbLoad"
                @error="onThumbError"
              />
              <div class="thumb-placeholder">
                <LucideImage :size="24" />
              </div>
            </div>
            <div class="card-info">
              <span class="card-name" :title="item.name">{{ item.name }}</span>
              <span class="card-size">{{ formatSize(item.size) }}</span>
            </div>
          </template>

          <!-- No thumbnail (encrypted) -->
          <template v-else>
            <div class="card-thumb no-thumb">
              <LucideFileLock :size="36" />
            </div>
            <div class="card-info">
              <span class="card-name" :title="item.name">{{ item.name }}</span>
              <span class="card-size">{{ formatSize(item.size) }}</span>
            </div>
          </template>
        </div>

        <!-- Infinite scroll sentinel -->
        <div ref="sentinelRef" class="scroll-sentinel" v-if="hasMore"></div>

        <!-- Loading more indicator -->
        <div v-if="loadingMore" class="loading-more">
          <div class="loading-spinner small"></div>
          <span>加载更多…</span>
        </div>
      </div>
    </template>

    <!-- Lightbox -->
    <Teleport to="body">
      <!-- Directory Browser Modal -->
      <Transition name="lightbox">
        <div v-if="dirBrowserOpen" class="lightbox-overlay" @click.self="closeDirBrowser">
          <div class="dir-browser-modal">
            <div class="dir-browser-header">
              <h3>选择目录</h3>
              <button class="lb-close-inline" @click="closeDirBrowser"><LucideX :size="20" /></button>
            </div>
            <div class="dir-browser-path">
              <LucideHome :size="14" class="path-icon" @click="dirBrowserNavigate('/')" />
              <template v-for="(seg, i) in dirBrowserSegments" :key="i">
                <LucideChevronRight :size="12" class="path-sep" />
                <span class="path-seg" @click="dirBrowserNavigateToSegment(i)">{{ seg }}</span>
              </template>
            </div>
            <div class="dir-browser-list">
              <div v-if="dirBrowserLoading" class="dir-browser-loading"><div class="loading-spinner small"></div></div>
              <div v-else-if="dirBrowserItems.length === 0" class="dir-browser-empty">此目录下没有子文件夹</div>
              <div
                v-for="folder in dirBrowserItems"
                :key="folder.name"
                class="dir-browser-item"
                @click="dirBrowserNavigate(dirBrowserPath + (dirBrowserPath.endsWith('/') ? '' : '/') + folder.name)"
              >
                <LucideFolder :size="18" />
                <span>{{ folder.name }}</span>
              </div>
            </div>
            <div class="dir-browser-footer">
              <button class="settings-apply-btn" @click="selectDirBrowserPath">选择当前目录</button>
              <button class="dir-browser-cancel" @click="closeDirBrowser">取消</button>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>

    <Teleport to="body">
      <Transition name="lightbox">
        <div v-if="lightboxOpen" class="lightbox-overlay" @click.self="closeLightbox">
          <button class="lb-close" @click="closeLightbox" title="关闭 (ESC)">
            <LucideX :size="24" />
          </button>

          <button class="lb-nav lb-prev" @click.stop="lightboxPrev" v-if="lightboxIndex > 0" title="上一张 (←)">
            <LucideChevronLeft :size="28" />
          </button>
          <button class="lb-nav lb-next" @click.stop="lightboxNext" v-if="lightboxIndex < imageItems.length - 1" title="下一张 (→)">
            <LucideChevronRight :size="28" />
          </button>

          <div class="lb-content">
            <div v-if="lightboxLoading" class="lb-loading">
              <div class="loading-spinner"></div>
            </div>
            <img
              v-if="lightboxSrc"
              :src="lightboxSrc"
              :alt="lightboxItem?.name"
              class="lb-image"
              @load="onLightboxLoad"
              @error="onLightboxError"
            />
          </div>

          <div class="lb-toolbar">
            <span class="lb-filename">{{ lightboxItem?.name }}</span>
            <span class="lb-filesize">{{ lightboxItem ? formatSize(lightboxItem.size) : '' }}</span>
            <span class="lb-counter">{{ lightboxIndex + 1 }} / {{ imageItems.length }}</span>
            <button class="lb-download" @click.stop="downloadCurrent" :disabled="downloading" title="下载">
              <LucideDownload :size="18" />
              <span>{{ downloading ? '获取中…' : '下载' }}</span>
            </button>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, watch, nextTick } from 'vue'
import {
  LucideImage, LucideImageOff, LucideFolder, LucideFileLock, LucideRefreshCw,
  LucideHome, LucideChevronRight, LucideChevronLeft, LucideX, LucideDownload,
  LucideAlertCircle, LucideShieldAlert, LucideSettings
} from 'lucide-vue-next'
import { api, handleAuthFailure } from '../api'
import type { GalleryEntry } from '../api'

// ---------------------------------------------------------------------------
// State
// ---------------------------------------------------------------------------
const currentPath = ref('/')
const currentDirId = ref('')
const items = ref<GalleryEntry[]>([])
const total = ref(-1)
const loading = ref(false)
const loadingMore = ref(false)
const error = ref('')
const providerError = ref(false)
const offset = ref(0)
const PAGE_SIZE = 50


const sentinelRef = ref<HTMLElement | null>(null)

const lightboxOpen = ref(false)
const lightboxIndex = ref(0)
const lightboxLoading = ref(false)
const downloading = ref(false)
const lightboxSrc = ref('')
const lightboxFallbackSrc = ref('')

// Settings
const GALLERY_ROOT_KEY = 'gallery_root_path'
const showSettings = ref(false)
const galleryRoot = ref(localStorage.getItem(GALLERY_ROOT_KEY) || '/')
const galleryRootInput = ref(galleryRoot.value)

// Directory browser
const dirBrowserOpen = ref(false)
const dirBrowserPath = ref('/')
const dirBrowserItems = ref<{name: string; is_dir: boolean}[]>([])
const dirBrowserLoading = ref(false)

// ---------------------------------------------------------------------------
// Computed
// ---------------------------------------------------------------------------
const hasMore = computed(() => items.value.length < total.value)

const pathSegments = computed(() => {
  const p = currentPath.value.replace(/^\/+|\/+$/g, '')
  return p ? p.split('/') : []
})

const imageItems = computed(() => items.value.filter(i => !i.is_dir && isPreviewable(i)))

const lightboxItem = computed(() => {
  const imgs = imageItems.value
  return imgs[lightboxIndex.value] ?? null
})

const dirBrowserSegments = computed(() => {
  const p = dirBrowserPath.value.replace(/^\/+|\/+$/g, '')
  return p ? p.split('/') : []
})

function isPreviewable(item: GalleryEntry): boolean {
  return Boolean(item.thumbnail || item.original_url)
}

function shouldProxyImage(rawUrl: string): boolean {
  return typeof window !== 'undefined'
    && window.location.protocol === 'https:'
    && rawUrl.startsWith('http://')
}

function getRenderableImageUrl(rawUrl: string, type: 'thumb' | 'original'): string {
  if (!rawUrl) return ''
  return shouldProxyImage(rawUrl) ? api.galleryProxyURL(rawUrl, type) : rawUrl
}

function getCardImagePrimaryUrl(item: GalleryEntry): string {
  if (item.thumbnail) {
    return getRenderableImageUrl(item.thumbnail, 'thumb')
  }
  if (item.original_url) {
    return getRenderableImageUrl(item.original_url, 'original')
  }
  return ''
}

function getCardImageFallbackUrl(item: GalleryEntry): string {
  if (!item.thumbnail || !item.original_url) {
    return ''
  }
  return getRenderableImageUrl(item.original_url, 'original')
}

// ---------------------------------------------------------------------------
// Data Loading
// ---------------------------------------------------------------------------
async function loadPage(append = false) {
  if (!append) {
    loading.value = true
    error.value = ''
  } else {
    loadingMore.value = true
  }

  try {
    const resp = await api.galleryList(
      currentPath.value,
      currentDirId.value || undefined,
      offset.value,
      PAGE_SIZE
    )
    if (append) {
      items.value.push(...resp.items)
    } else {
      items.value = resp.items
    }
    total.value = resp.total
    currentDirId.value = resp.dir_id

    // Lazy load images after DOM update
    await nextTick()
    observeImages()
  } catch (e) {
    if (handleAuthFailure(e)) return
    error.value = e instanceof Error ? e.message : '加载失败'
  } finally {
    loading.value = false
    loadingMore.value = false
  }
}

function refresh() {
  offset.value = 0
  currentDirId.value = ''
  loadPage()
}

function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  offset.value = items.value.length
  loadPage(true)
}

// ---------------------------------------------------------------------------
// Navigation
// ---------------------------------------------------------------------------
function navigateTo(path: string) {
  currentPath.value = path
  currentDirId.value = ''
  offset.value = 0
  items.value = []
  total.value = -1
  loadPage()
}

function navigateToSegment(index: number) {
  const segs = pathSegments.value.slice(0, index + 1)
  navigateTo('/' + segs.join('/'))
}

function applyGalleryRoot() {
  const raw = galleryRootInput.value.trim()
  const newRoot = raw.startsWith('/') ? raw : '/' + raw
  galleryRoot.value = newRoot
  localStorage.setItem(GALLERY_ROOT_KEY, newRoot)
  showSettings.value = false
  navigateTo(newRoot)
}

// ---------------------------------------------------------------------------
// Directory Browser
// ---------------------------------------------------------------------------
function openDirBrowser() {
  dirBrowserPath.value = galleryRoot.value
  dirBrowserOpen.value = true
  loadDirBrowserItems()
}

function closeDirBrowser() {
  dirBrowserOpen.value = false
}

async function loadDirBrowserItems() {
  dirBrowserLoading.value = true
  dirBrowserItems.value = []
  try {
    // dir_only=true: 后端只返回文件夹，不过滤图片类型
    const resp = await api.galleryList(dirBrowserPath.value, undefined, 0, 500, true)
    dirBrowserItems.value = resp.items
  } catch (e) {
    if (handleAuthFailure(e)) return
  } finally {
    dirBrowserLoading.value = false
  }
}

function dirBrowserNavigate(path: string) {
  dirBrowserPath.value = path
  loadDirBrowserItems()
}

function dirBrowserNavigateToSegment(index: number) {
  const segs = dirBrowserSegments.value.slice(0, index + 1)
  dirBrowserNavigate('/' + segs.join('/'))
}

function selectDirBrowserPath() {
  galleryRootInput.value = dirBrowserPath.value
  closeDirBrowser()
  applyGalleryRoot()
}

// ---------------------------------------------------------------------------
// Lightbox image loading
// ---------------------------------------------------------------------------
watch(lightboxItem, (item) => {
  lightboxSrc.value = ''
  lightboxFallbackSrc.value = ''
  if (!item) return
  lightboxLoading.value = true

  if (item.original_url) {
    lightboxSrc.value = getRenderableImageUrl(item.original_url, 'original')
    if (item.thumbnail) {
      lightboxFallbackSrc.value = getRenderableImageUrl(item.thumbnail, 'thumb')
    }
    return
  }

  if (item.thumbnail) {
    lightboxSrc.value = getRenderableImageUrl(item.thumbnail, 'thumb')
  } else {
    lightboxLoading.value = false
  }
})

function onCardClick(item: GalleryEntry, _index: number) {
  if (item.is_dir) {
    const target = currentPath.value.replace(/\/+$/, '') + '/' + item.name
    navigateTo(target)
  } else if (isPreviewable(item)) {
    // Previewable: open in lightbox
    const imgs = imageItems.value
    const imgIdx = imgs.findIndex(i => i.id === item.id)
    if (imgIdx >= 0) {
      openLightbox(imgIdx)
    }
  } else if (item.pick_code) {
    // Non-previewable (e.g. encrypted): trigger download directly
    downloadByPickCode(item.pick_code, item.name)
  }
}

async function downloadByPickCode(pickCode: string, fileName: string) {
  try {
    const resp = await api.galleryDownloadUrl(pickCode)
    if (resp.url) {
      const a = document.createElement('a')
      a.href = resp.url
      a.download = resp.file_name || fileName
      a.target = '_blank'
      a.rel = 'noopener'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
    }
  } catch (e) {
    if (handleAuthFailure(e)) return
    error.value = e instanceof Error ? e.message : '获取下载链接失败'
  }
}

// ---------------------------------------------------------------------------
// Lightbox
// ---------------------------------------------------------------------------
function openLightbox(imgIndex: number) {
  lightboxIndex.value = imgIndex
  lightboxLoading.value = true
  lightboxOpen.value = true
}

function closeLightbox() {
  lightboxOpen.value = false
  lightboxSrc.value = ''
  lightboxFallbackSrc.value = ''
}

function lightboxPrev() {
  if (lightboxIndex.value > 0) {
    lightboxIndex.value--
    lightboxLoading.value = true
  }
}

function lightboxNext() {
  if (lightboxIndex.value < imageItems.value.length - 1) {
    lightboxIndex.value++
    lightboxLoading.value = true
  }
}

async function downloadCurrent() {
  const item = lightboxItem.value
  if (!item?.pick_code) return
  downloading.value = true
  try {
    const resp = await api.galleryDownloadUrl(item.pick_code)
    if (resp.url) {
      const a = document.createElement('a')
      a.href = resp.url
      a.download = resp.file_name || item.name
      a.target = '_blank'
      a.rel = 'noopener'
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
    }
  } catch (e) {
    if (handleAuthFailure(e)) return
    error.value = e instanceof Error ? e.message : '获取下载链接失败'
  } finally {
    downloading.value = false
  }
}

function onKeyDown(e: KeyboardEvent) {
  if (!lightboxOpen.value) return
  switch (e.key) {
    case 'Escape': closeLightbox(); break
    case 'ArrowLeft': lightboxPrev(); break
    case 'ArrowRight': lightboxNext(); break
  }
}

// ---------------------------------------------------------------------------
// Image Loading
// ---------------------------------------------------------------------------

function markImageLoadError(img: HTMLElement) {
  img.classList.add('error')
  img.classList.remove('lazy')
}

function loadCardImage(img: HTMLImageElement) {
  const primaryUrl = img.dataset.src
  if (!primaryUrl) return

  delete img.dataset.src
  img.src = primaryUrl
}

function onLightboxLoad() {
  lightboxLoading.value = false
}

function onLightboxError(e: Event) {
  const img = e.target as HTMLImageElement
  if (lightboxFallbackSrc.value) {
    const fallbackUrl = lightboxFallbackSrc.value
    lightboxFallbackSrc.value = ''
    img.src = fallbackUrl
    return
  }
  lightboxLoading.value = false
  lightboxSrc.value = ''
}

// ---------------------------------------------------------------------------
// Lazy Loading (IntersectionObserver)
// ---------------------------------------------------------------------------
let imageObserver: IntersectionObserver | null = null
let sentinelObserver: IntersectionObserver | null = null

function getScrollRoot(): Element | null {
  return document.querySelector('.main-content')
}

function observeImages() {
  if (!imageObserver) return
  const imgs = document.querySelectorAll('.thumb-img.lazy[data-src]')
  imgs.forEach(img => {
    if (!(img as HTMLImageElement).src || (img as HTMLImageElement).src === window.location.href) {
      imageObserver!.observe(img)
    }
  })
}

function onThumbLoad(e: Event) {
  const img = e.target as HTMLElement
  img.classList.add('loaded')
  img.classList.remove('lazy')
}

function onThumbError(e: Event) {
  const img = e.target as HTMLImageElement
  const fallbackUrl = img.dataset.fallbackSrc
  if (fallbackUrl) {
    delete img.dataset.fallbackSrc
    img.src = fallbackUrl
    return
  }
  markImageLoadError(img)
}

// ---------------------------------------------------------------------------
// Formatting
// ---------------------------------------------------------------------------
function formatSize(bytes: number): string {
  if (!bytes || bytes <= 0) return ''
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------
async function checkProvider() {
  try {
    const status = await api.getSystemStatus()
    if (status.provider !== 'open115') {
      providerError.value = true
    }
  } catch (e) {
    if (handleAuthFailure(e)) return
  }
}

onMounted(async () => {
  await checkProvider()
  if (providerError.value) return

  // Use persisted gallery root as initial path
  currentPath.value = galleryRoot.value

  // Image lazy loader
  imageObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          const img = entry.target as HTMLImageElement
          if (img.dataset.src) {
            loadCardImage(img)
          }
          imageObserver?.unobserve(img)
        }
      })
    },
    { root: getScrollRoot(), rootMargin: '200px' }
  )

  // Infinite scroll sentinel
  await loadPage()

  await nextTick()
  if (sentinelRef.value) {
    sentinelObserver = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          loadMore()
        }
      },
      { root: getScrollRoot(), rootMargin: '400px' }
    )
    sentinelObserver.observe(sentinelRef.value)
  }

  document.addEventListener('keydown', onKeyDown)
})

// Re-observe sentinel when it appears/disappears
watch(sentinelRef, (el) => {
  if (el && sentinelObserver) {
    sentinelObserver.observe(el)
  }
})

onUnmounted(() => {
  document.removeEventListener('keydown', onKeyDown)
  imageObserver?.disconnect()
  sentinelObserver?.disconnect()
})
</script>

<style scoped>
.gallery-page {
  padding: 24px 32px;
  max-width: 1600px;
  margin: 0 auto;
}

/* Header */
.gallery-header {
  margin-bottom: 24px;
}

.gallery-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.gallery-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 24px;
  font-weight: 800;
  color: var(--text-primary);
  margin: 0;
}

.gallery-actions {
  display: flex;
  align-items: center;
  gap: 12px;
}

.item-count {
  font-size: 14px;
  color: var(--text-secondary);
  font-weight: 500;
}

.action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  border: 1px solid var(--border-strong);
  background: var(--bg-card);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-btn:hover {
  color: var(--text-primary);
  background: var(--bg-primary);
}

.spinning {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* Settings popover */
.settings-popover {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 8px;
  background: var(--bg-card);
  border: 1px solid var(--border-strong);
  border-radius: 12px;
  padding: 16px;
  min-width: 320px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.3);
  z-index: 50;
}

.settings-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
  display: block;
  margin-bottom: 4px;
}

.settings-hint {
  font-size: 12px;
  color: var(--text-secondary);
  margin: 0 0 12px;
}

.settings-input-row {
  display: flex;
  gap: 8px;
}

.settings-input {
  flex: 1;
  padding: 8px 12px;
  border-radius: 8px;
  border: 1px solid var(--border-strong);
  background: var(--bg-primary);
  color: var(--text-primary);
  font-size: 13px;
  outline: none;
  transition: border-color 0.2s;
}

.settings-input:focus {
  border-color: var(--accent);
}

.settings-apply-btn {
  padding: 8px 16px;
  border-radius: 8px;
  border: none;
  background: var(--accent);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.2s;
  white-space: nowrap;
}

.settings-apply-btn:hover {
  opacity: 0.85;
}

.popover-enter-active, .popover-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.popover-enter-from, .popover-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.gallery-title-row {
  position: relative;
}

/* Directory Browser Modal */
.dir-browser-modal {
  background: var(--bg-card);
  border-radius: 16px;
  width: 480px;
  max-width: 90vw;
  max-height: 70vh;
  display: flex;
  flex-direction: column;
  box-shadow: 0 12px 48px rgba(0, 0, 0, 0.4);
}

.dir-browser-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--border-strong);
}

.dir-browser-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--text-primary);
}

.lb-close-inline {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  border-radius: 6px;
  transition: color 0.2s;
}
.lb-close-inline:hover { color: var(--text-primary); }

.dir-browser-path {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 12px 20px;
  font-size: 13px;
  border-bottom: 1px solid var(--border-strong);
  flex-wrap: wrap;
}

.dir-browser-path .path-icon {
  color: var(--text-secondary);
  cursor: pointer;
  flex-shrink: 0;
}
.dir-browser-path .path-icon:hover { color: var(--accent); }
.dir-browser-path .path-sep { color: var(--text-muted); flex-shrink: 0; }
.dir-browser-path .path-seg {
  color: var(--text-secondary);
  cursor: pointer;
}
.dir-browser-path .path-seg:hover { color: var(--accent); }

.dir-browser-list {
  flex: 1;
  overflow-y: auto;
  min-height: 200px;
  max-height: 400px;
}

.dir-browser-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 20px;
  cursor: pointer;
  color: var(--text-primary);
  font-size: 14px;
  transition: background 0.15s;
}
.dir-browser-item:hover {
  background: var(--bg-primary);
}
.dir-browser-item svg { color: var(--accent); flex-shrink: 0; }

.dir-browser-loading, .dir-browser-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;
  color: var(--text-secondary);
  font-size: 14px;
}

.dir-browser-footer {
  display: flex;
  gap: 8px;
  padding: 12px 20px;
  border-top: 1px solid var(--border-strong);
  justify-content: flex-end;
}

.dir-browser-cancel {
  padding: 8px 16px;
  border-radius: 8px;
  border: 1px solid var(--border-strong);
  background: transparent;
  color: var(--text-secondary);
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.dir-browser-cancel:hover {
  color: var(--text-primary);
  background: var(--bg-primary);
}

/* Breadcrumb */
.breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 13px;
}

.crumb {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  border-radius: 6px;
  background: transparent;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  font-size: 13px;
  font-weight: 500;
  transition: all 0.15s ease;
}

.crumb:hover {
  background: var(--bg-card);
  color: var(--text-primary);
}

.crumb.current {
  color: var(--text-primary);
  font-weight: 600;
  cursor: default;
}

.crumb-sep {
  color: var(--text-tertiary);
  flex-shrink: 0;
}

/* Grid */
.gallery-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
}

.gallery-card {
  background: var(--bg-card);
  border: 1px solid var(--border-subtle);
  border-radius: 14px;
  overflow: hidden;
  cursor: pointer;
  transition: all 0.2s ease;
}

.gallery-card:hover {
  border-color: var(--border-strong);
  transform: translateY(-2px);
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.08);
}

.card-thumb {
  position: relative;
  width: 100%;
  aspect-ratio: 1;
  background: var(--bg-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}

.thumb-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  opacity: 0;
  transition: opacity 0.3s ease;
}

.thumb-img.loaded {
  opacity: 1;
}

.thumb-img.loaded + .thumb-placeholder {
  display: none;
}

.thumb-placeholder {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--text-tertiary);
}

.thumb-img.error + .thumb-placeholder {
  display: flex;
}

.folder-icon {
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.08), rgba(59, 130, 246, 0.08));
  color: #6366F1;
}

.no-thumb {
  background: linear-gradient(135deg, rgba(245, 158, 11, 0.08), rgba(239, 68, 68, 0.08));
  color: var(--text-tertiary);
}

.card-info {
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.card-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.card-size {
  font-size: 11px;
  color: var(--text-tertiary);
}

/* States */
.gallery-loading,
.gallery-error,
.gallery-empty,
.gallery-unsupported {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 20px;
  text-align: center;
  color: var(--text-secondary);
  gap: 12px;
}

.gallery-unsupported h2 {
  font-size: 20px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.gallery-empty h3 {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-primary);
  margin: 0;
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--border-subtle);
  border-top-color: #6366F1;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

.loading-spinner.small {
  width: 20px;
  height: 20px;
  border-width: 2px;
}

.retry-btn {
  margin-top: 8px;
  padding: 8px 20px;
  border-radius: 10px;
  border: 1px solid var(--border-strong);
  background: var(--bg-card);
  color: var(--text-primary);
  cursor: pointer;
  font-weight: 600;
  transition: all 0.2s ease;
}

.retry-btn:hover {
  background: var(--text-primary);
  color: var(--text-inverted);
}

/* Infinite scroll */
.scroll-sentinel {
  height: 1px;
  grid-column: 1 / -1;
}

.loading-more {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 20px;
  color: var(--text-secondary);
  font-size: 13px;
}

/* Lightbox */
.lightbox-overlay {
  position: fixed;
  inset: 0;
  z-index: 9999;
  background: rgba(0, 0, 0, 0.92);
  display: flex;
  align-items: center;
  justify-content: center;
  backdrop-filter: blur(8px);
}

.lb-close {
  position: absolute;
  top: 16px;
  right: 16px;
  z-index: 10;
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s ease;
}

.lb-close:hover {
  background: rgba(255, 255, 255, 0.2);
}

.lb-nav {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  z-index: 10;
  width: 48px;
  height: 48px;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.1);
  border: 1px solid rgba(255, 255, 255, 0.12);
  color: #fff;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.2s ease;
}

.lb-nav:hover {
  background: rgba(255, 255, 255, 0.2);
}

.lb-prev { left: 16px; }
.lb-next { right: 16px; }

.lb-content {
  max-width: calc(100vw - 120px);
  max-height: calc(100vh - 120px);
  display: flex;
  align-items: center;
  justify-content: center;
}

.lb-image {
  max-width: 100%;
  max-height: calc(100vh - 120px);
  object-fit: contain;
  border-radius: 4px;
  user-select: none;
}

.lb-loading {
  position: absolute;
}

.lb-toolbar {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 16px;
  padding: 16px 24px;
  background: linear-gradient(transparent, rgba(0, 0, 0, 0.6));
  color: rgba(255, 255, 255, 0.85);
  font-size: 13px;
}

.lb-filename {
  font-weight: 600;
  max-width: 300px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.lb-filesize {
  color: rgba(255, 255, 255, 0.5);
}

.lb-counter {
  color: rgba(255, 255, 255, 0.5);
}

.lb-download {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.12);
  border: 1px solid rgba(255, 255, 255, 0.15);
  color: #fff;
  cursor: pointer;
  font-size: 13px;
  font-weight: 600;
  transition: background 0.2s ease;
}

.lb-download:hover {
  background: rgba(255, 255, 255, 0.22);
}

.lb-download:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

/* Lightbox transition */
.lightbox-enter-active, .lightbox-leave-active {
  transition: opacity 0.25s ease;
}

.lightbox-enter-from, .lightbox-leave-to {
  opacity: 0;
}

/* Responsive */
@media (max-width: 768px) {
  .gallery-page {
    padding: 16px;
  }

  .gallery-grid {
    grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
    gap: 10px;
  }

  .gallery-title {
    font-size: 20px;
  }

  .lb-nav {
    width: 40px;
    height: 40px;
  }

  .lb-prev { left: 8px; }
  .lb-next { right: 8px; }

  .lb-toolbar {
    flex-wrap: wrap;
    gap: 8px;
    font-size: 12px;
  }
}

@media (max-width: 480px) {
  .gallery-grid {
    grid-template-columns: repeat(2, 1fr);
    gap: 8px;
  }
}
</style>
