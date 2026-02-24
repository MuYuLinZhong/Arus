import { ref, type Ref } from 'vue'

interface UsePagedListOptions<T> {
  fetchFn: (params: Record<string, any>) => Promise<{ items: T[]; next_cursor: string; has_more: boolean }>
  defaultParams?: Record<string, any>
  limit?: number
}

export function usePagedList<T>(options: UsePagedListOptions<T>) {
  const items: Ref<T[]> = ref([])
  const loading = ref(false)
  const hasMore = ref(true)
  const cursor = ref('')

  async function loadMore(params: Record<string, any> = {}) {
    if (loading.value || !hasMore.value) return

    loading.value = true
    try {
      const data = await options.fetchFn({
        ...options.defaultParams,
        ...params,
        cursor: cursor.value,
        limit: options.limit || 20,
      })

      items.value = [...items.value, ...data.items] as any
      cursor.value = data.next_cursor
      hasMore.value = data.has_more
    } finally {
      loading.value = false
    }
  }

  function reset() {
    items.value = []
    cursor.value = ''
    hasMore.value = true
  }

  async function refresh(params: Record<string, any> = {}) {
    reset()
    await loadMore(params)
  }

  return { items, loading, hasMore, loadMore, reset, refresh }
}
