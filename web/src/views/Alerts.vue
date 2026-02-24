<template>
  <div>
    <div class="page-header">
      <h2>告警管理</h2>
    </div>

    <a-space style="margin-bottom: 16px">
      <a-select v-model:value="filterStatus" placeholder="状态" allow-clear style="width: 120px" @change="fetchAlerts">
        <a-select-option :value="0">待处置</a-select-option>
        <a-select-option :value="1">已处置</a-select-option>
        <a-select-option :value="2">已忽略</a-select-option>
      </a-select>
      <a-select v-model:value="filterSeverity" placeholder="级别" allow-clear style="width: 120px" @change="fetchAlerts">
        <a-select-option :value="3">高</a-select-option>
        <a-select-option :value="2">中</a-select-option>
        <a-select-option :value="1">低</a-select-option>
      </a-select>
      <a-input v-model:value="filterDeviceID" placeholder="设备ID" style="width: 160px" @press-enter="fetchAlerts" />
    </a-space>

    <a-table
      :columns="columns"
      :data-source="alerts"
      :loading="loading"
      :pagination="{ current: page, pageSize, total, onChange: onPageChange }"
      row-key="id"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'alert_type'">
          {{ alertTypeMap[record.alert_type] || record.alert_type }}
        </template>
        <template v-if="column.key === 'severity'">
          <a-tag :color="alertSeverityMap[record.severity]?.color">
            {{ alertSeverityMap[record.severity]?.text }}
          </a-tag>
        </template>
        <template v-if="column.key === 'status'">
          <a-tag :color="alertStatusMap[record.status]?.color">
            {{ alertStatusMap[record.status]?.text }}
          </a-tag>
        </template>
        <template v-if="column.key === 'created_at'">
          {{ formatTime(record.created_at) }}
        </template>
        <template v-if="column.key === 'actions'">
          <a v-if="record.status === 0" @click="openHandleModal(record)">处置</a>
          <span v-else style="color: #999">{{ record.handle_note || '-' }}</span>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="showHandleModal" title="处置告警" @ok="submitHandle" :confirm-loading="handling">
      <a-form layout="vertical">
        <a-form-item label="处置备注" required>
          <a-textarea v-model:value="handleForm.handle_note" :rows="3" placeholder="请输入处置说明" />
        </a-form-item>
        <a-form-item v-if="currentAlert?.alert_type === 'consecutive_fail'">
          <a-checkbox v-model:checked="handleForm.unlock_device">同时解除设备告警锁定</a-checkbox>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getAlerts, handleAlert } from '@/api/admin'
import { formatTime, alertSeverityMap, alertStatusMap, alertTypeMap } from '@/utils/format'
import type { Alert } from '@/types'

const alerts = ref<Alert[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = 20
const total = ref(0)

const filterStatus = ref<number>()
const filterSeverity = ref<number>()
const filterDeviceID = ref('')

const showHandleModal = ref(false)
const handling = ref(false)
const currentAlert = ref<Alert | null>(null)
const handleForm = reactive({ handle_note: '', unlock_device: true })

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 60 },
  { title: '类型', key: 'alert_type', width: 140 },
  { title: '设备', dataIndex: 'device_id', key: 'device_id', width: 140 },
  { title: '级别', key: 'severity', width: 80 },
  { title: '状态', key: 'status', width: 90 },
  { title: '时间', key: 'created_at', width: 170 },
  { title: '操作/备注', key: 'actions' },
]

onMounted(() => fetchAlerts())

async function fetchAlerts() {
  loading.value = true
  try {
    const data = await getAlerts({
      page: page.value, page_size: pageSize,
      status: filterStatus.value,
      severity: filterSeverity.value,
      device_id: filterDeviceID.value || undefined,
    })
    alerts.value = data.items
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  fetchAlerts()
}

function openHandleModal(alert: Alert) {
  currentAlert.value = alert
  handleForm.handle_note = ''
  handleForm.unlock_device = alert.alert_type === 'consecutive_fail'
  showHandleModal.value = true
}

async function submitHandle() {
  if (!currentAlert.value) return
  handling.value = true
  try {
    await handleAlert(currentAlert.value.id, {
      handle_note: handleForm.handle_note,
      unlock_device: handleForm.unlock_device,
    })
    message.success('告警已处置')
    showHandleModal.value = false
    fetchAlerts()
  } finally {
    handling.value = false
  }
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
</style>
