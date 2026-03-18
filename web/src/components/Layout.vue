<template>
  <div class="layout-container">
    <!-- Sidebar (desktop) -->
    <div class="sidebar">
      <div class="sidebar-header">
        <h1 class="logo">ImmichTo115</h1>
        <div class="sidebar-accent"></div>
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
        <router-link to="/photo-upload" class="nav-item" active-class="active">
          <LucideCamera :size="20" />
          <span>摄影上传</span>
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
      <button class="theme-fab" @click="toggleTheme" :title="isDark ? '切换到白天模式' : '切换到夜间模式'">
        <LucideMoon v-if="!isDark" :size="18" />
        <LucideSun v-else :size="18" />
      </button>
      <router-view />
    </div>

    <!-- Mobile Bottom Tab Bar -->
    <nav class="mobile-tab-bar">
      <router-link to="/dashboard" class="tab-item" active-class="active">
        <LucideLayout :size="20" />
        <span>仪表盘</span>
      </router-link>
      <router-link to="/explore" class="tab-item" active-class="active">
        <LucideHardDrive :size="20" />
        <span>云端</span>
      </router-link>
      <router-link to="/settings" class="tab-item" active-class="active">
        <LucideSettings :size="20" />
        <span>设置</span>
      </router-link>
      <router-link to="/photo-upload" class="tab-item" active-class="active">
        <LucideCamera :size="20" />
        <span>上传</span>
      </router-link>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { LucideLayout, LucideHardDrive, LucideSettings, LucideCamera, LucideSun, LucideMoon } from 'lucide-vue-next'
import { api, handleAuthFailure } from '../api'

const serviceHealthy = ref(true)
let statusTimer: ReturnType<typeof setInterval> | null = null

// Theme toggle
const isDark = ref(false)

const applyTheme = (dark: boolean) => {
  isDark.value = dark
  document.documentElement.classList.toggle('dark', dark)
}

const toggleTheme = () => {
  const newDark = !isDark.value
  applyTheme(newDark)
  localStorage.setItem('theme', newDark ? 'dark' : 'light')
}

const initTheme = () => {
  // main.ts 已经初始化过 class, 这里只需同步 isDark ref
  isDark.value = document.documentElement.classList.contains('dark')
}

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
  initTheme()
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

.sidebar-accent {
  height: 2px;
  margin-top: 16px;
  background: linear-gradient(90deg, #6366F1, #3B82F6, #06B6D4);
  border-radius: 1px;
  opacity: 0.6;
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

.theme-fab {
  position: fixed;
  top: 16px;
  right: 20px;
  z-index: 50;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  border-radius: 10px;
  background-color: var(--bg-card);
  border: 1px solid var(--border-strong);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
}

.theme-fab:hover {
  color: var(--text-primary);
  background-color: var(--border-subtle);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
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

.mobile-tab-bar {
  display: none;
}

@media (max-width: 768px) {
  .sidebar {
    display: none;
  }

  .main-content {
    width: 100%;
    padding-bottom: calc(64px + env(safe-area-inset-bottom, 0px));
  }

  .mobile-tab-bar {
    display: flex;
    position: fixed;
    bottom: 0;
    left: 0;
    right: 0;
    height: calc(64px + env(safe-area-inset-bottom, 0px));
    padding-bottom: env(safe-area-inset-bottom, 0px);
    background-color: var(--bg-card);
    border-top: 1px solid var(--border-strong);
    z-index: 100;
    align-items: stretch;
  }

  .tab-item {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;
    color: var(--text-tertiary);
    text-decoration: none;
    font-size: 11px;
    font-weight: 600;
    transition: color 0.2s ease;
    -webkit-tap-highlight-color: transparent;
  }

  .tab-item.active {
    color: var(--text-primary);
  }
}
</style>
