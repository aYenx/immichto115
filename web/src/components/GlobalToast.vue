<template>
  <div v-if="toastState.visible" class="toast-wrap">
    <div :class="['toast-card', toastState.tone]">
      <div class="toast-title-row">
        <strong>{{ toastState.title }}</strong>
        <button class="toast-close" @click="hideToast">×</button>
      </div>
      <p>{{ toastState.message }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { hideToast, toastState } from '../composables/toast'
</script>

<style scoped>
.toast-wrap {
  position: fixed;
  top: 24px;
  right: 24px;
  z-index: 200;
}

.toast-card {
  width: min(380px, calc(100vw - 32px));
  padding: 16px 16px 14px;
  border-radius: 18px;
  border: 1px solid var(--border-strong);
  background: var(--bg-primary);
  box-shadow: 0 18px 48px rgba(15, 23, 42, 0.14);
}

.toast-card.success {
  border-color: rgba(22, 163, 74, 0.28);
}

.toast-card.error {
  border-color: rgba(220, 38, 38, 0.28);
}

.toast-card.warning {
  border-color: rgba(245, 158, 11, 0.28);
}

.toast-card.info {
  border-color: rgba(37, 99, 235, 0.28);
}

.toast-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 6px;
}

.toast-title-row strong {
  font-size: 14px;
}

.toast-card p {
  color: var(--text-secondary);
  font-size: 13px;
  line-height: 1.6;
}

.toast-close {
  width: 24px;
  height: 24px;
  border-radius: 12px;
  color: var(--text-secondary);
  font-size: 18px;
  line-height: 1;
}
</style>
