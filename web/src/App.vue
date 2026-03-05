<script setup lang="ts">
import { NConfigProvider, NMessageProvider, zhCN, dateZhCN, type GlobalThemeOverrides } from 'naive-ui'
import AppLayout from './components/AppLayout.vue'

const themeOverrides: GlobalThemeOverrides = {
  common: {
    primaryColor: '#0ea5e9',
    primaryColorHover: '#38bdf8',
    primaryColorPressed: '#0284c7',
    primaryColorSuppl: '#38bdf8',
    infoColor: '#0ea5e9',
    successColor: '#10b981',
    warningColor: '#f59e0b',
    errorColor: '#ef4444',
    borderRadius: '8px'
  },
  Card: {
    borderRadius: '12px',
    borderColor: '#e2e8f0',
    boxShadow: '0 4px 6px -1px rgb(0 0 0 / 0.1), 0 2px 4px -2px rgb(0 0 0 / 0.1)'
  },
  Button: {
    borderRadius: '8px',
    fontWeight: '500'
  }
}
</script>

<template>
  <NConfigProvider :locale="zhCN" :date-locale="dateZhCN" :theme-overrides="themeOverrides">
    <NMessageProvider>
      <AppLayout />
    </NMessageProvider>
  </NConfigProvider>
</template>
