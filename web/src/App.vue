<template>
  <GlobalToast />
  <router-view></router-view>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import GlobalToast from './components/GlobalToast.vue'
import { initSession } from './api'

onMounted(() => {
  initSession()
})
</script>
