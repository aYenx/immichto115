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
          </div>
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

          <!-- Image with thumbnail -->
          <template v-else-if="item.thumbnail">
            <div class="card-thumb">
              <img
                :data-src="api.galleryProxyURL(item.thumbnail, 'thumb')"
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
              @load="lightboxLoading = false"
              @error="lightboxLoading = false"
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
  LucideAlertCircle, LucideShieldAlert
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

// Lightbox
const lightboxOpen = ref(false)
const lightboxIndex = ref(0)
const lightboxLoading = ref(false)
const downloading = ref(false)

// ---------------------------------------------------------------------------
// Computed
// ---------------------------------------------------------------------------
const hasMore = computed(() => items.value.length < total.value)

const pathSegments = computed(() => {
  const p = currentPath.value.replace(/^\/+|\/+$/g, '')
  return p ? p.split('/') : []
})

const imageItems = computed(() => items.value.filter(i => !i.is_dir && (i.thumbnail || i.original_url)))

const lightboxItem = computed(() => {
  const imgs = imageItems.value
  return imgs[lightboxIndex.value] ?? null
})

const lightboxSrc = computed(() => {
  const item = lightboxItem.value
  if (!item) return ''
  if (item.original_url) {
    return api.galleryProxyURL(item.original_url, 'original')
  }
  if (item.thumbnail) {
    return api.galleryProxyURL(item.thumbnail, 'original')
  }
  return ''
})

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

function onCardClick(item: GalleryEntry, _index: number) {
  if (item.is_dir) {
    const target = currentPath.value.replace(/\/+$/, '') + '/' + item.name
    navigateTo(target)
  } else if (item.thumbnail || item.original_url) {
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
  const img = e.target as HTMLElement
  img.classList.add('error')
  img.classList.remove('lazy')
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

  // Image lazy loader
  imageObserver = new IntersectionObserver(
    (entries) => {
      entries.forEach(entry => {
        if (entry.isIntersecting) {
          const img = entry.target as HTMLImageElement
          const src = img.dataset.src
          if (src) {
            img.src = src
            delete img.dataset.src
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
