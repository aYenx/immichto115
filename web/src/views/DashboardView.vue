<script setup lang="ts">
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { NCard, NButton, NSpace, NTag, NAlert, NSpin, useMessage } from 'naive-ui'
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
    <h2 class="text-2xl font-bold text-gray-800 mb-6">仪表盘</h2>

    <!-- 状态卡片 -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
      <NCard size="small">
        <div class="text-sm text-gray-500 mb-1">备份状态</div>
        <NTag v-if="status" :type="status.backup_status === 'running' ? 'warning' : 'success'" size="large">
          {{ status.backup_status === 'running' ? '⏳ 运行中' : '✅ 空闲' }}
        </NTag>
        <NSpin v-else size="small" />
      </NCard>

      <NCard size="small">
        <div class="text-sm text-gray-500 mb-1">Rclone</div>
        <div v-if="status">
          <NTag v-if="status.rclone_installed" type="success">已安装</NTag>
          <NTag v-else type="error">未安装</NTag>
        </div>
        <NSpin v-else size="small" />
      </NCard>

      <NCard size="small">
        <div class="text-sm text-gray-500 mb-1">下次运行</div>
        <div v-if="status">
          <span v-if="status.cron_enabled && status.next_run" class="text-sm font-mono">
            {{ status.next_run }}
          </span>
          <NTag v-else type="default">未设置定时</NTag>
        </div>
        <NSpin v-else size="small" />
      </NCard>
    </div>

    <!-- 未配置提示 -->
    <NAlert v-if="status && !status.setup_complete" type="warning" title="未完成配置" class="mb-4">
      请先前往「任务配置」页面完成 WebDAV 连接和备份路径设置。
    </NAlert>

    <!-- 控制按钮 -->
    <NSpace class="mb-4">
      <NButton
        type="primary"
        :loading="loading"
        :disabled="status?.backup_status === 'running'"
        @click="startBackup"
      >
        ▶ 手动备份
      </NButton>
      <NButton
        type="error"
        :disabled="status?.backup_status !== 'running'"
        @click="stopBackup"
      >
        ⏹ 停止备份
      </NButton>
      <NButton quaternary @click="clearLogs">清空日志</NButton>
      <NTag :type="connected ? 'success' : 'error'" size="small">
        WS: {{ connected ? '已连接' : '断开' }}
      </NTag>
    </NSpace>

    <!-- 实时日志 -->
    <NCard title="📋 实时日志" size="small">
      <div
        ref="logContainer"
        class="bg-gray-900 text-green-400 font-mono text-xs p-4 rounded overflow-y-auto"
        style="height: 400px; white-space: pre-wrap; word-break: break-all;"
      >
        <div v-if="logs.length === 0" class="text-gray-500">等待日志输出...</div>
        <div
          v-for="(line, i) in logs"
          :key="i"
          :class="line.stream === 'stderr' ? 'text-red-400' : 'text-green-400'"
        >{{ line.text }}</div>
      </div>
    </NCard>
  </div>
</template>
