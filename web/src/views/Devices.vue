<template>
  <div>
    <div class="page-header">
      <h2>锁具管理</h2>
      <a-button type="primary" @click="showCreateModal = true">添加锁具</a-button>
    </div>

    <a-space style="margin-bottom: 16px">
      <a-input-search v-model:value="searchText" placeholder="搜索设备ID或名称" @search="fetchDevices" style="width: 280px" />
      <a-select v-model:value="filterStatus" placeholder="状态" allow-clear style="width: 140px" @change="fetchDevices">
        <a-select-option :value="1">正常</a-select-option>
        <a-select-option :value="0">已禁用</a-select-option>
        <a-select-option :value="2">告警锁定</a-select-option>
      </a-select>
    </a-space>

    <a-table
      :columns="columns"
      :data-source="devices"
      :loading="loading"
      :pagination="{ current: page, pageSize, total, onChange: onPageChange }"
      row-key="id"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'status'">
          <a-tag :color="deviceStatusMap[record.status]?.color">
            {{ deviceStatusMap[record.status]?.text }}
          </a-tag>
        </template>
        <template v-if="column.key === 'risk_level'">
          <a-tag :color="riskLevelMap[record.risk_level]?.color">
            {{ riskLevelMap[record.risk_level]?.text }}
          </a-tag>
        </template>
        <template v-if="column.key === 'last_active_at'">
          {{ record.last_active_at ? formatTime(record.last_active_at) : '从未活跃' }}
        </template>
        <template v-if="column.key === 'created_at'">
          {{ formatTime(record.created_at) }}
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="showCreateModal" title="添加锁具" @ok="handleCreate" :confirm-loading="creating" width="600px">
      <a-form :model="createForm" layout="vertical">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="设备ID" required>
              <a-input v-model:value="createForm.device_id" placeholder="出厂唯一编号" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="设备名称" required>
              <a-input v-model:value="createForm.name" placeholder="设备名称" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="安装位置" required>
          <a-input v-model:value="createForm.location_text" placeholder="详细位置描述" />
        </a-form-item>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="经度">
              <a-input-number v-model:value="createForm.longitude" style="width: 100%" :precision="7" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="纬度">
              <a-input-number v-model:value="createForm.latitude" style="width: 100%" :precision="7" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="管线标签">
              <a-input v-model:value="createForm.pipeline_tag" placeholder="归属管线" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="风险等级" required>
              <a-select v-model:value="createForm.risk_level">
                <a-select-option :value="1">普通</a-select-option>
                <a-select-option :value="2">重要</a-select-option>
                <a-select-option :value="3">关键</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="设备密钥 (K_d)" required>
          <a-input-password
            v-model:value="createForm.device_key"
            placeholder="32 位十六进制，例如 0123456789abcdef0123456789abcdef"
            :maxlength="32"
          />
          <template #extra>
            <span class="form-hint">AES-128 密钥，仅可填 0-9、a-f，共 32 个字符</span>
          </template>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getDevices, createDevice } from '@/api/admin'
import { formatTime, deviceStatusMap, riskLevelMap } from '@/utils/format'
import type { Device } from '@/types'

const devices = ref<Device[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const searchText = ref('')
const filterStatus = ref<number>()

const showCreateModal = ref(false)
const creating = ref(false)
const createForm = reactive({
  device_id: '', name: '', location_text: '',
  longitude: undefined as number | undefined,
  latitude: undefined as number | undefined,
  pipeline_tag: '', risk_level: 1, device_key: '',
})

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '设备ID', dataIndex: 'device_id', key: 'device_id', width: 140 },
  { title: '名称', dataIndex: 'name', key: 'name' },
  { title: '位置', dataIndex: 'location_text', key: 'location_text', ellipsis: true },
  { title: '管线', dataIndex: 'pipeline_tag', key: 'pipeline_tag', width: 100 },
  { title: '风险等级', key: 'risk_level', width: 90 },
  { title: '状态', key: 'status', width: 100 },
  { title: '最后活跃', key: 'last_active_at', width: 170 },
  { title: '创建时间', key: 'created_at', width: 170 },
]

onMounted(() => fetchDevices())

async function fetchDevices() {
  loading.value = true
  try {
    const data = await getDevices({
      page: page.value, page_size: pageSize,
      search: searchText.value || undefined,
      status: filterStatus.value,
    })
    devices.value = data.items
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  fetchDevices()
}

async function handleCreate() {
  creating.value = true
  try {
    await createDevice(createForm)
    message.success('锁具添加成功')
    showCreateModal.value = false
    fetchDevices()
  } finally {
    creating.value = false
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
.form-hint { font-size: 12px; color: #8c8c8c; }
</style>
