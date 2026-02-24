<template>
  <div>
    <div class="page-header">
      <h2>审计日志</h2>
    </div>

    <a-space style="margin-bottom: 16px" wrap>
      <a-input v-model:value="filterDeviceID" placeholder="设备ID" style="width: 160px" />
      <a-select v-model:value="filterAction" placeholder="操作类型" allow-clear style="width: 150px">
        <a-select-option value="unlock_success">开锁成功</a-select-option>
        <a-select-option value="unlock_fail">开锁失败</a-select-option>
        <a-select-option value="challenge_request">挑战请求</a-select-option>
        <a-select-option value="challenge_denied">挑战被拒绝</a-select-option>
      </a-select>
      <a-range-picker v-model:value="dateRange" show-time />
      <a-button type="primary" @click="handleSearch">查询</a-button>
    </a-space>

    <a-table
      :columns="columns"
      :data-source="logs"
      :loading="loading"
      :pagination="false"
      row-key="id"
      :row-class-name="rowClassName"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'action'">
          <a-tag :color="record.action.includes('fail') || record.action.includes('denied') ? 'red' : 'green'">
            {{ auditActionMap[record.action] || record.action }}
          </a-tag>
        </template>
        <template v-if="column.key === 'occurred_at'">
          {{ formatTime(record.occurred_at) }}
        </template>
      </template>
    </a-table>

    <div style="text-align: center; margin-top: 16px">
      <a-button v-if="hasMore" :loading="loading" @click="loadMore">加载更多</a-button>
      <span v-else-if="logs.length > 0" style="color: #999">没有更多数据了</span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { getAuditLogs } from '@/api/admin'
import { formatTime, auditActionMap } from '@/utils/format'
import type { AuditLog } from '@/types'

const logs = ref<AuditLog[]>([])
const loading = ref(false)
const hasMore = ref(true)
const cursor = ref('')

const filterDeviceID = ref('')
const filterAction = ref<string>()
const dateRange = ref<any>(null)

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 80 },
  { title: '用户ID', dataIndex: 'user_id', key: 'user_id', width: 80 },
  { title: '设备ID', dataIndex: 'device_id', key: 'device_id', width: 140 },
  { title: '操作', key: 'action', width: 120 },
  { title: '结果码', dataIndex: 'result_code', key: 'result_code', width: 80 },
  { title: 'IP', dataIndex: 'client_ip', key: 'client_ip', width: 140 },
  { title: '设备型号', dataIndex: 'device_model', key: 'device_model', width: 130 },
  { title: '时间', key: 'occurred_at', width: 170 },
]

onMounted(() => fetchLogs())

function buildParams() {
  const params: Record<string, any> = { limit: 20 }
  if (cursor.value) params.cursor = cursor.value
  if (filterDeviceID.value) params.device_id = filterDeviceID.value
  if (filterAction.value) params.action = filterAction.value
  if (dateRange.value?.[0]) params.start_time = dateRange.value[0].toISOString()
  if (dateRange.value?.[1]) params.end_time = dateRange.value[1].toISOString()
  return params
}

async function fetchLogs() {
  loading.value = true
  try {
    const data = await getAuditLogs(buildParams())
    logs.value = [...logs.value, ...data.items]
    cursor.value = data.next_cursor
    hasMore.value = data.has_more
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  logs.value = []
  cursor.value = ''
  hasMore.value = true
  fetchLogs()
}

function loadMore() {
  fetchLogs()
}

function rowClassName(record: AuditLog) {
  if (record.action === 'unlock_fail') return 'row-fail'
  if (record.action === 'challenge_denied') return 'row-denied'
  return ''
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.page-header h2 { margin: 0; font-size: 20px; }

:deep(.row-fail) {
  background-color: #fff2e8 !important;
}
:deep(.row-denied) {
  background-color: #fff1f0 !important;
}
</style>
