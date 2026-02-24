import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getAlerts } from '@/api/admin'
import type { Alert } from '@/types'

export const useAlertStore = defineStore('alert', () => {
  const pendingAlerts = ref<Alert[]>([])
  const pendingCount = ref(0)

  async function fetchPendingAlerts() {
    try {
      const data = await getAlerts({ status: 0, page: 1, page_size: 20 })
      pendingAlerts.value = data.items
      pendingCount.value = data.total
    } catch {
      // silently fail for polling
    }
  }

  return { pendingAlerts, pendingCount, fetchPendingAlerts }
})
