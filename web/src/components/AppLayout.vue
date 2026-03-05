<script setup lang="ts">
import { h } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import {
  NLayout,
  NLayoutSider,
  NLayoutContent,
  NMenu,
  NIcon,
  type MenuOption,
} from 'naive-ui'

const router = useRouter()
const route = useRoute()

// 简单的 SVG 图标渲染
function renderIcon(svg: string) {
  return () => h(NIcon, null, { default: () => h('span', { innerHTML: svg }) })
}

const menuOptions: MenuOption[] = [
  {
    label: '仪表盘',
    key: '/dashboard',
    icon: renderIcon(
      '<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor"><path d="M3 13h8V3H3v10zm0 8h8v-6H3v6zm10 0h8V11h-8v10zm0-18v6h8V3h-8z"/></svg>'
    ),
  },
  {
    label: '任务配置',
    key: '/setup',
    icon: renderIcon(
      '<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor"><path d="M19.14 12.94c.04-.3.06-.61.06-.94 0-.32-.02-.64-.07-.94l2.03-1.58a.49.49 0 0 0 .12-.61l-1.92-3.32a.49.49 0 0 0-.59-.22l-2.39.96c-.5-.38-1.03-.7-1.62-.94l-.36-2.54a.484.484 0 0 0-.48-.41h-3.84c-.24 0-.43.17-.47.41l-.36 2.54c-.59.24-1.13.57-1.62.94l-2.39-.96a.49.49 0 0 0-.59.22L2.74 8.87c-.12.21-.08.47.12.61l2.03 1.58c-.05.3-.07.62-.07.94s.02.64.07.94l-2.03 1.58a.49.49 0 0 0-.12.61l1.92 3.32c.12.22.37.29.59.22l2.39-.96c.5.38 1.03.7 1.62.94l.36 2.54c.05.24.24.41.48.41h3.84c.24 0 .44-.17.47-.41l.36-2.54c.59-.24 1.13-.56 1.62-.94l2.39.96c.22.08.47 0 .59-.22l1.92-3.32c.12-.22.07-.47-.12-.61l-2.01-1.58zM12 15.6A3.6 3.6 0 1 1 12 8.4a3.6 3.6 0 0 1 0 7.2z"/></svg>'
    ),
  },
  {
    label: '数据恢复',
    key: '/restore',
    icon: renderIcon(
      '<svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor"><path d="M19 3H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm-2 10h-4v4h-2v-4H7v-2h4V7h2v4h4v2z"/></svg>'
    ),
  },
]

function handleMenuSelect(key: string) {
  router.push(key)
}
</script>

<template>
  <NLayout has-sider class="h-screen">
    <NLayoutSider
      bordered
      :width="220"
      :collapsed-width="64"
      collapse-mode="width"
      show-trigger
      class="shadow-sm"
    >
      <div class="flex items-center gap-2 px-4 py-4 border-b border-gray-200">
        <div class="text-xl font-bold text-blue-600">📦</div>
        <span class="text-sm font-semibold text-gray-800 whitespace-nowrap">ImmichTo115</span>
      </div>
      <NMenu
        :options="menuOptions"
        :value="route.path"
        @update:value="handleMenuSelect"
        class="mt-2"
      />
    </NLayoutSider>
    <NLayoutContent class="bg-gray-50">
      <div class="p-6 max-w-5xl mx-auto">
        <router-view />
      </div>
    </NLayoutContent>
  </NLayout>
</template>
