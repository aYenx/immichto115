<script setup lang="ts">
import { ref, h } from 'vue'
import { NCard, NButton, NDataTable, NSpace, NBreadcrumb, NBreadcrumbItem, NSpin, useMessage, NIcon } from 'naive-ui'
import { api, type RemoteFile } from '../api'
import type { DataTableColumns } from 'naive-ui'

const message = useMessage()
const loading = ref(false)
const files = ref<RemoteFile[]>([])
const currentPath = ref('')
const pathParts = ref<string[]>([])

function formatSize(bytes: number): string {
  if (bytes === 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return (bytes / Math.pow(1024, i)).toFixed(1) + ' ' + units[i]
}

const columns: DataTableColumns<RemoteFile> = [
  {
    title: '名称',
    key: 'Name',
    render(row) {
      if (row.IsDir) {
        return h(
          'div',
          {
            class: 'flex items-center text-blue-600 cursor-pointer hover:text-blue-700 font-medium group',
            onClick: () => navigateTo(row.Path),
          },
          [
            h(NIcon, { class: 'mr-2 text-blue-400 group-hover:text-blue-500 transition-colors', size: 18 }, { default: () => h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }, [h('path', { d: 'M20 20a2 2 0 0 0 2-2V8a2 2 0 0 0-2-2h-7.9a2 2 0 0 1-1.69-.9L9.6 3.9A2 2 0 0 0 7.93 3H4a2 2 0 0 0-2 2v13a2 2 0 0 0 2 2Z' })]) }),
            h('span', { class: 'group-hover:underline underline-offset-2' }, row.Name)
          ]
        )
      }
      return h(
        'div',
        { class: 'flex items-center text-slate-600' },
        [
          h(NIcon, { class: 'mr-2 text-slate-400', size: 18 }, { default: () => h('svg', { xmlns: 'http://www.w3.org/2000/svg', viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', 'stroke-width': '2', 'stroke-linecap': 'round', 'stroke-linejoin': 'round' }, [h('path', { d: 'M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z' }), h('path', { d: 'M14 2v4a2 2 0 0 0 2 2h4' })]) }),
          h('span', {}, row.Name)
        ]
      )
    },
  },
  {
    title: '大小',
    key: 'Size',
    width: 150,
    render(row) {
      return h('span', { class: 'text-slate-500 font-mono text-sm' }, row.IsDir ? '-' : formatSize(row.Size))
    },
  },
  {
    title: '修改时间',
    key: 'ModTime',
    width: 200,
    render(row) {
      if (!row.ModTime) return h('span', { class: 'text-slate-400' }, '-')
      return h('span', { class: 'text-slate-500 font-mono text-sm' }, new Date(row.ModTime).toLocaleString('zh-CN'))
    },
  },
]

async function loadFiles(path: string = '') {
  loading.value = true
  try {
    files.value = await api.listRemote(path)
    currentPath.value = path
    pathParts.value = path ? path.split('/').filter(Boolean) : []
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error('加载失败: ' + msg)
    files.value = []
  } finally {
    loading.value = false
  }
}

function navigateTo(path: string) {
  loadFiles(path)
}

function navigateToIndex(index: number) {
  if (index < 0) {
    loadFiles('')
  } else {
    const path = pathParts.value.slice(0, index + 1).join('/')
    loadFiles(path)
  }
}
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-8">
      <h2 class="text-3xl font-extrabold tracking-tight text-slate-900">云端文件浏览</h2>
    </div>

    <NCard class="shadow-sm hover:shadow-md transition-shadow" :bordered="false">
      <NSpace vertical size="large">
        <!-- 面包屑导航与工具栏 -->
        <div class="flex flex-col sm:flex-row sm:items-center justify-between gap-4 bg-slate-50 p-4 rounded-lg border border-slate-100">
          <NBreadcrumb separator=">">
            <NBreadcrumbItem @click="navigateToIndex(-1)">
              <span class="flex items-center cursor-pointer hover:text-blue-600 transition-colors">
                <NIcon class="mr-1"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 9l9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg></NIcon>
                根目录
              </span>
            </NBreadcrumbItem>
            <NBreadcrumbItem
              v-for="(part, i) in pathParts"
              :key="i"
              @click="navigateToIndex(i)"
            >
              <span class="cursor-pointer hover:text-blue-600 transition-colors">{{ part }}</span>
            </NBreadcrumbItem>
          </NBreadcrumb>

          <div class="flex items-center gap-3">
            <transition name="fade">
              <div v-if="loading" class="flex items-center text-blue-500 text-sm font-medium bg-blue-50 px-3 py-1.5 rounded-full">
                <NSpin size="small" class="mr-2" /> 加载中...
              </div>
            </transition>
            <NSpace>
              <NButton size="small" secondary @click="loadFiles('')" class="hover:text-blue-600">
                <template #icon>
                  <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="m3 9 9-7 9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg></NIcon>
                </template>
                回根目录
              </NButton>
              <NButton size="small" secondary type="info" @click="loadFiles(currentPath)">
                <template #icon>
                  <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 12a9 9 0 1 0 9-9 9.75 9.75 0 0 0-6.74 2.74L3 8"/><path d="M3 3v5h5"/></svg></NIcon>
                </template>
                刷新
              </NButton>
            </NSpace>
          </div>
        </div>

        <!-- 文件表格 -->
        <div class="border border-slate-100 rounded-lg overflow-hidden">
          <NDataTable
            :columns="columns"
            :data="files"
            :loading="loading"
            :bordered="false"
            :single-line="false"
            :bottom-bordered="false"
            size="medium"
            class="[&_.n-data-table-th]:bg-slate-50 [&_.n-data-table-th]:text-slate-500 [&_.n-data-table-th]:font-semibold"
          />
        </div>

        <div v-if="!loading && files.length === 0" class="flex flex-col items-center justify-center py-16 text-slate-400 bg-slate-50/50 rounded-lg border border-slate-100 border-dashed">
          <NIcon size="48" class="mb-4 text-slate-300">
            <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"><rect width="18" height="18" x="3" y="3" rx="2" ry="2"/><circle cx="9" cy="9" r="2"/><path d="m21 15-3.086-3.086a2 2 0 0 0-2.828 0L6 21"/></svg>
          </NIcon>
          <p class="text-lg font-medium text-slate-500">当前目录为空或未加载</p>
          <p class="text-sm mt-1">点击右上角「回根目录」或「刷新」开始浏览</p>
        </div>
      </NSpace>
    </NCard>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.3s ease;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}
</style>
