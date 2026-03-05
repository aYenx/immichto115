<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { NCard, NButton, NSpace, NTag, NAlert, NSpin, useMessage, NIcon } from 'naive-ui'
import { api, type SystemStatus } from '../api'
import { useWebSocket } from '../composables/useWebSocket'

const message = useMessage()
const status = ref<SystemStatus | null>(null)
const loading = ref(false)
const logContainer = ref<HTMLElement | null>(null)

const { logs, connected, connect, clearLogs } = useWebSocket()

// 自动滚到底部
watch(
  () => logs.value.length,
  async () => {
    await nextTick()
    if (logContainer.value) {
      logContainer.value.scrollTop = logContainer.value.scrollHeight
    }
  }
)

async function fetchStatus() {
  try {
    status.value = await api.getStatus()
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error('获取状态失败: ' + msg)
  }
}

async function startBackup() {
  loading.value = true
  try {
    await api.startBackup()
    message.success('备份已启动')
    await fetchStatus()
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error('启动失败: ' + msg)
  } finally {
    loading.value = false
  }
}

async function stopBackup() {
  try {
    await api.stopBackup()
    message.warning('已发送停止信号')
    await fetchStatus()
  } catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error('停止失败: ' + msg)
  }
}

let statusTimer: ReturnType<typeof setInterval> | null = null

onMounted(() => {
  fetchStatus()
  connect()
  statusTimer = setInterval(fetchStatus, 10000)
})

onUnmounted(() => {
  if (statusTimer) clearInterval(statusTimer)
})
</script>

<template>
  <div>
    <div class="flex items-center justify-between mb-8">
      <h2 class="text-3xl font-extrabold tracking-tight text-slate-900">仪表盘</h2>
      <NTag :type="connected ? 'success' : 'error'" :bordered="false" class="font-medium px-3 py-1 rounded-full shadow-sm">
        <template #icon>
          <div :class="['w-2 h-2 rounded-full mr-1', connected ? 'bg-emerald-500 animate-pulse' : 'bg-rose-500']"></div>
        </template>
        WebSocket {{ connected ? '已连接' : '已断开' }}
      </NTag>
    </div>

    <!-- 状态卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
      <NCard size="medium" class="hover:shadow-md transition-shadow duration-300 group">
        <div class="flex items-start justify-between mb-4">
          <div class="text-sm font-semibold text-slate-500 uppercase tracking-wider">备份状态</div>
          <div class="p-2 rounded-lg bg-blue-50 text-blue-500 group-hover:bg-blue-100 transition-colors">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" x2="12" y1="3" y2="15"/></svg>
          </div>
        </div>
        <div v-if="status" class="flex items-center mt-1">
          <div v-if="status.backup_status === 'running'" class="flex items-center text-amber-600 font-bold text-lg">
            <svg class="animate-spin -ml-1 mr-2 h-5 w-5 text-amber-500" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4"></circle><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path></svg>
            运行中
          </div>
          <div v-else class="flex items-center text-emerald-600 font-bold text-lg">
            <svg xmlns="http://www.w3.org/2000/svg" class="mr-2 h-5 w-5" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><path d="M22 11.08V12a10 10 0 1 1-5.93-9.14"/><polyline points="22 4 12 14.01 9 11.01"/></svg>
            空闲
          </div>
        </div>
        <NSpin v-else size="small" />
      </NCard>

      <NCard size="medium" class="hover:shadow-md transition-shadow duration-300 group">
        <div class="flex items-start justify-between mb-4">
          <div class="text-sm font-semibold text-slate-500 uppercase tracking-wider">Rclone 引擎</div>
          <div class="p-2 rounded-lg bg-indigo-50 text-indigo-500 group-hover:bg-indigo-100 transition-colors">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polygon points="12 2 2 7 12 12 22 7 12 2"/><polyline points="2 17 12 22 22 17"/><polyline points="2 12 12 17 22 12"/></svg>
          </div>
        </div>
        <div v-if="status" class="mt-1">
          <div v-if="status.rclone_installed" class="flex items-center text-slate-800 font-bold text-lg">
            <div class="w-2.5 h-2.5 rounded-full bg-emerald-500 mr-2.5"></div>
            已就绪
          </div>
          <div v-else class="flex items-center text-rose-600 font-bold text-lg">
            <div class="w-2.5 h-2.5 rounded-full bg-rose-500 mr-2.5"></div>
            未安装
          </div>
        </div>
        <NSpin v-else size="small" />
      </NCard>

      <NCard size="medium" class="hover:shadow-md transition-shadow duration-300 group">
        <div class="flex items-start justify-between mb-4">
          <div class="text-sm font-semibold text-slate-500 uppercase tracking-wider">下次计划任务</div>
          <div class="p-2 rounded-lg bg-teal-50 text-teal-500 group-hover:bg-teal-100 transition-colors">
            <svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/></svg>
          </div>
        </div>
        <div v-if="status" class="mt-1">
          <div v-if="status.cron_enabled && status.next_run" class="text-lg font-bold text-slate-800 tracking-tight">
            {{ status.next_run }}
          </div>
          <div v-else class="flex items-center text-slate-400 font-medium text-base">
            <svg xmlns="http://www.w3.org/2000/svg" class="mr-2 h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"/><line x1="4.93" y1="4.93" x2="19.07" y2="19.07"/></svg>
            未启用定时任务
          </div>
        </div>
        <NSpin v-else size="small" />
      </NCard>
    </div>

    <!-- 未配置提示 -->
    <NAlert v-if="status && !status.setup_complete" type="warning" class="mb-8 rounded-lg border border-amber-200">
      <template #header>
        <span class="font-bold">系统尚未完成初始化配置</span>
      </template>
      请先前往 <router-link to="/setup" class="font-bold text-amber-700 hover:underline">任务配置</router-link> 页面完成 WebDAV 连接和备份路径设置。
    </NAlert>

    <!-- 控制栏 -->
    <div class="bg-white p-4 rounded-xl shadow-sm border border-slate-200 mb-6 flex flex-wrap items-center justify-between gap-4">
      <NSpace>
        <NButton
          type="primary"
          size="large"
          :loading="loading"
          :disabled="status?.backup_status === 'running'"
          @click="startBackup"
          class="shadow-sm shadow-blue-500/30"
        >
          <template #icon>
            <NIcon v-if="!loading"><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><polygon points="5 3 19 12 5 21 5 3"/></svg></NIcon>
          </template>
          手动启动备份
        </NButton>
        <NButton
          type="error"
          size="large"
          secondary
          :disabled="status?.backup_status !== 'running'"
          @click="stopBackup"
        >
          <template #icon>
            <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/></svg></NIcon>
          </template>
          紧急停止
        </NButton>
      </NSpace>
      
      <NButton quaternary size="small" @click="clearLogs" class="text-slate-500 hover:text-slate-800">
        <template #icon>
          <NIcon><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M3 6h18"/><path d="M19 6v14c0 1-1 2-2 2H7c-1 0-2-1-2-2V6"/><path d="M8 6V4c0-1 1-2 2-2h4c1 0 2 1 2 2v2"/></svg></NIcon>
        </template>
        清空终端日志
      </NButton>
    </div>

    <!-- 实时日志终端 -->
    <div class="rounded-xl overflow-hidden shadow-lg border border-slate-800 bg-slate-950 flex flex-col" style="height: 500px;">
      <div class="flex items-center px-4 py-3 bg-slate-900 border-b border-slate-800">
        <div class="flex space-x-2 mr-4">
          <div class="w-3 h-3 rounded-full bg-rose-500/80"></div>
          <div class="w-3 h-3 rounded-full bg-amber-500/80"></div>
          <div class="w-3 h-3 rounded-full bg-emerald-500/80"></div>
        </div>
        <div class="text-xs font-medium text-slate-400 font-mono flex items-center">
          <svg xmlns="http://www.w3.org/2000/svg" class="mr-2 h-4 w-4" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 17 10 11 4 5"/><line x1="12" y1="19" x2="20" y2="19"/></svg>
          rclone_sync.log
        </div>
      </div>
      
      <div
        ref="logContainer"
        class="flex-1 overflow-y-auto p-4 font-mono text-sm leading-relaxed"
        style="white-space: pre-wrap; word-break: break-all; font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace;"
      >
        <div v-if="logs.length === 0" class="text-slate-500 italic flex items-center h-full justify-center">
          等待备份进程输出日志...
        </div>
        <div
          v-for="(line, i) in logs"
          :key="i"
          class="flex hover:bg-slate-800/50 py-0.5 px-2 -mx-2 rounded transition-colors"
        >
          <span class="text-slate-600 mr-4 select-none text-xs leading-5 w-8 text-right shrink-0">{{ i + 1 }}</span>
          <span :class="line.stream === 'stderr' ? 'text-rose-400' : 'text-emerald-400/90'">{{ line.text }}</span>
        </div>
      </div>
    </div>
  </div>
</template>
