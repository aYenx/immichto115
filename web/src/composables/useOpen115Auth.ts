import { ref, reactive, onUnmounted } from 'vue'
import { api, handleAuthFailure, getErrorMessage, type Open115AuthStartResponse, type Open115AuthFinishResponse } from '../api'
import { showToast } from './toast'

/**
 * 115 Open 扫码授权流程 composable.
 * 封装 QR 生成 → 轮询 → 完成授权的完整生命周期。
 */
export function useOpen115Auth() {
  const authData = reactive<Open115AuthStartResponse>({
    uid: '',
    time: 0,
    sign: '',
    qrcode: '',
    created_at: '',
  })
  const statusText = ref('未开始')
  const authorized = ref<boolean | null>(null)
  const isLoading = ref(false)
  const isFinishing = ref(false)

  let pollTimer: number | null = null

  // ---------- Internal ----------

  const stopPolling = () => {
    if (pollTimer != null) {
      window.clearInterval(pollTimer)
      pollTimer = null
    }
  }

  const poll = async () => {
    if (!authData.uid) return
    try {
      const status = await api.open115AuthStatus(authData.uid)
      statusText.value = status.message || `status=${status.status}`
      authorized.value = status.authorized
      if (status.authorized) {
        stopPolling()
        showToast('success', '扫码已确认', '已收到 115 授权确认，请点击"完成授权"。')
      }
    } catch (err) {
      if (handleAuthFailure(err)) return
      statusText.value = '状态查询失败：' + getErrorMessage(err)
      stopPolling()
    }
  }

  // ---------- Actions ----------

  /**
   * Start a new QR code auth flow.
   * @param clientId  The 115 Open client_id.
   * @throws          Re-throws if there is a validation issue for the caller to handle.
   */
  const start = async (clientId: string) => {
    isLoading.value = true
    authorized.value = null
    statusText.value = '正在生成二维码...'
    try {
      const result = await api.open115AuthStart({ client_id: clientId })
      Object.assign(authData, result)
      statusText.value = '等待扫码'
      stopPolling()
      pollTimer = window.setInterval(() => {
        void poll()
      }, 2500)
    } catch (err) {
      if (handleAuthFailure(err)) return
      showToast('error', '启动扫码失败', getErrorMessage(err))
      statusText.value = '启动失败'
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Finish the auth flow after user has scanned the QR code.
   * @returns The `state` object containing tokens/config from the backend.
   */
  const finish = async (): Promise<Open115AuthFinishResponse['state'] | null> => {
    if (!authData.uid) return null
    isFinishing.value = true
    try {
      const result = await api.open115AuthFinish({ uid: authData.uid })
      authorized.value = true
      statusText.value = '授权完成'
      return result.state
    } catch (err) {
      if (handleAuthFailure(err)) return null
      showToast('error', '完成授权失败', getErrorMessage(err))
      return null
    } finally {
      isFinishing.value = false
    }
  }

  // Auto-cleanup on unmount
  onUnmounted(() => stopPolling())

  return {
    authData,
    statusText,
    authorized,
    isLoading,
    isFinishing,
    start,
    finish,
    stopPolling,
  }
}
