<template>
  <div class="layout-container">
    <!-- Sidebar -->
    <div class="sidebar">
      <div class="sidebar-header">
        <h1 class="logo">ImmichTo115</h1>
      </div>
      
      <div class="nav-menu">
        <router-link to="/dashboard" class="nav-item" active-class="active">
          <LucideLayout :size="20" />
          <span>仪表盘</span>
        </router-link>
        <router-link to="/explore" class="nav-item" active-class="active">
          <LucideHardDrive :size="20" />
          <span>云端目录</span>
        </router-link>
        <router-link to="/settings" class="nav-item" active-class="active">
          <LucideSettings :size="20" />
          <span>设置</span>
        </router-link>
      </div>

      <div class="sidebar-footer">
        <div class="status-indicator">
          <div :class="['status-dot', serviceHealthy ? 'healthy' : 'offline']"></div>
          <span>{{ serviceHealthy ? '服务运行中' : '服务连接异常' }}</span>
        </div>
      </div>
    </div>

    <!-- Main Content -->
    <div class="main-content">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { LucideLayout, LucideHardDrive, LucideSettings } from 'lucide-vue-next'
import { api, handleAuthFailure } from '../api'

const serviceHealthy = ref(true)
let statusTimer: ReturnType<typeof setInterval> | null = null

const checkServiceHealth = async () => {
  try {
    await api.getSystemStatus()
    serviceHealthy.value = true
  } catch (error) {
    if (handleAuthFailure(error)) {
      return
    }
    serviceHealthy.value = false
  }
}

onMounted(() => {
  checkServiceHealth()
  statusTimer = setInterval(checkServiceHealth, 5000)
})

onUnmounted(() => {
  if (statusTimer) {
    clearInterval(statusTimer)
    statusTimer = null
  }
})
</script>

<style scoped>
.layout-container {
  display: flex;
  width: 100vw;
  height: 100vh;
  background-color: var(--bg-primary);
}

.sidebar {
  display: flex;
  flex-direction: column;
  width: 280px;
  height: 100%;
  background-color: var(--bg-card);
  border-right: 1px solid var(--border-strong);
  justify-content: space-between;
}

.sidebar-header {
  padding: 32px 24px 16px 24px;
}

.logo {
  font-family: var(--font-primary);
  font-size: 24px;
  font-weight: 900;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}

.nav-menu {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 0 16px;
  flex: 1;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 16px;
  border-radius: 12px;
  color: var(--text-secondary);
  font-size: 16px;
  font-weight: 500;
  text-decoration: none;
  transition: all 0.2s ease;
}

.nav-item:hover {
  background-color: var(--bg-primary);
}

.nav-item.active {
  background-color: var(--text-primary);
  color: var(--text-inverted);
}

.sidebar-footer {
  padding: 24px;
  border-top: 1px solid var(--border-subtle);
}

.status-indicator {
  display: flex;
  align-items: center;
  gap: 12px;
  color: var(--text-secondary);
  font-size: 14px;
  font-weight: 500;
}

.status-dot {
  width: 10px;
  height: 10px;
  border-radius: 5px;
}

.status-dot.healthy {
  background-color: #10B981;
  box-shadow: 0 0 8px rgba(16, 185, 129, 0.4);
}

.status-dot.offline {
  background-color: #EF4444;
  box-shadow: 0 0 8px rgba(239, 68, 68, 0.35);
}

.main-content {
  flex: 1;
  overflow: auto;
  position: relative;
}
</style>
