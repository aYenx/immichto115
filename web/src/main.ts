import { createApp } from 'vue'
import './style.css'
import App from './App.vue'
import router from './router'

// 全局主题初始化 —— 在 Vue 挂载前同步执行，确保所有路由（包括 /setup）均生效
;(function initTheme() {
  const saved = localStorage.getItem('theme')
  if (saved === 'dark') {
    document.documentElement.classList.add('dark')
  } else if (!saved && window.matchMedia('(prefers-color-scheme: dark)').matches) {
    document.documentElement.classList.add('dark')
  }
})()

const app = createApp(App)

app.use(router)
app.mount('#app')

