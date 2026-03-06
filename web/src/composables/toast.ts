import { reactive } from 'vue'

export type ToastTone = 'success' | 'error' | 'warning' | 'info'

export const toastState = reactive({
  visible: false,
  title: '',
  message: '',
  tone: 'info' as ToastTone,
})

let toastTimer: number | null = null

export function hideToast() {
  toastState.visible = false
  if (toastTimer !== null) {
    window.clearTimeout(toastTimer)
    toastTimer = null
  }
}

export function showToast(tone: ToastTone, title: string, message: string, duration = 3200) {
  toastState.visible = true
  toastState.tone = tone
  toastState.title = title
  toastState.message = message

  if (toastTimer !== null) {
    window.clearTimeout(toastTimer)
  }

  toastTimer = window.setTimeout(() => {
    toastState.visible = false
    toastTimer = null
  }, duration)
}
