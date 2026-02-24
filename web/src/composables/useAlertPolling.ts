import { onMounted, onUnmounted } from 'vue'
import { useAlertStore } from '@/stores/alert'

export function useAlertPolling(intervalMs = 30000) {
  const alertStore = useAlertStore()
  let timer: ReturnType<typeof setInterval> | null = null

  onMounted(() => {
    alertStore.fetchPendingAlerts()
    timer = setInterval(() => {
      alertStore.fetchPendingAlerts()
    }, intervalMs)
  })

  onUnmounted(() => {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  })

  return { alertStore }
}
