<script setup lang="ts">
import { ref } from 'vue'
import { NCard, NButton, NDataTable, NSpace, NBreadcrumb, NBreadcrumbItem, NTag, NSpin, useMessage } from 'naive-ui'
import { api, type RemoteFile } from '../api'
import type { DataTableColumns } from 'naive-ui'
import { h } from 'vue'

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
          'a',
          {
            class: 'text-blue-600 cursor-pointer hover:underline font-medium',
            onClick: () => navigateTo(row.Path),
          },
          '📁 ' + row.Name
        )
      }
      return h('span', {}, '📄 ' + row.Name)
    },
  },
  {
    title: '大小',
    key: 'Size',
    width: 120,
    render(row) {
      return row.IsDir ? '-' : formatSize(row.Size)
    },
  },
  {
    title: '修改时间',
    key: 'ModTime',
    width: 200,
    render(row) {
      if (!row.ModTime) return '-'
      return new Date(row.ModTime).toLocaleString('zh-CN')
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
    <h2 class="text-2xl font-bold text-gray-800 mb-6">数据恢复 · 云端浏览</h2>

    <NCard>
      <NSpace vertical>
        <!-- 面包屑导航 -->
        <NBreadcrumb>
          <NBreadcrumbItem @click="navigateToIndex(-1)">
            <span class="cursor-pointer">🏠 根目录</span>
          </NBreadcrumbItem>
          <NBreadcrumbItem
            v-for="(part, i) in pathParts"
            :key="i"
            @click="navigateToIndex(i)"
          >
            <span class="cursor-pointer">{{ part }}</span>
          </NBreadcrumbItem>
        </NBreadcrumb>

        <!-- 刷新按钮 -->
        <NSpace>
          <NButton size="small" @click="loadFiles(currentPath)">🔄 刷新</NButton>
          <NButton size="small" @click="loadFiles('')">📂 加载根目录</NButton>
          <NTag v-if="loading" type="info">
            <NSpin size="small" /> 加载中...
          </NTag>
        </NSpace>

        <!-- 文件表格 -->
        <NDataTable
          :columns="columns"
          :data="files"
          :loading="loading"
          :bordered="false"
          :single-line="false"
          size="small"
        />

        <div v-if="!loading && files.length === 0" class="text-center text-gray-400 py-8">
          点击「加载根目录」开始浏览云端文件
        </div>
      </NSpace>
    </NCard>
  </div>
</template>
