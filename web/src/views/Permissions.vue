<template>
  <div>
    <div class="page-header">
      <h2>权限管理</h2>
      <a-button type="primary" @click="showGrantModal = true">授权</a-button>
    </div>

    <a-space style="margin-bottom: 16px">
      <a-select v-model:value="filterStatus" placeholder="状态" allow-clear style="width: 120px" @change="fetchPermissions">
        <a-select-option :value="1">有效</a-select-option>
        <a-select-option :value="0">已撤销</a-select-option>
      </a-select>
    </a-space>

    <a-table
      :columns="columns"
      :data-source="permissions"
      :loading="loading"
      :pagination="{ current: page, pageSize, total, onChange: onPageChange }"
      row-key="id"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'user'">
          {{ record.user?.name || '-' }}
        </template>
        <template v-if="column.key === 'device'">
          {{ record.device?.name || '-' }} ({{ record.device?.device_id || '-' }})
        </template>
        <template v-if="column.key === 'status'">
          <a-tag :color="record.status === 1 ? 'green' : 'red'">
            {{ record.status === 1 ? '有效' : '已撤销' }}
          </a-tag>
        </template>
        <template v-if="column.key === 'valid_from'">
          {{ formatTime(record.valid_from) }}
        </template>
        <template v-if="column.key === 'valid_until'">
          {{ record.valid_until ? formatTime(record.valid_until) : '永久' }}
        </template>
        <template v-if="column.key === 'actions'">
          <a-popconfirm
            v-if="record.status === 1"
            title="确认撤销该授权？将实时生效"
            @confirm="handleRevoke(record.id)"
          >
            <a style="color: #ff4d4f">撤销</a>
          </a-popconfirm>
          <span v-else style="color: #999">-</span>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="showGrantModal" title="授权" @ok="handleGrant" :confirm-loading="granting">
      <a-form :model="grantForm" layout="vertical">
        <a-form-item label="用户 ID" required>
          <a-input-number v-model:value="grantForm.user_id" style="width: 100%" placeholder="填用户管理表格中的 ID 列数字" />
        </a-form-item>
        <a-form-item label="设备 ID" required>
          <a-input-number v-model:value="grantForm.device_id" style="width: 100%" placeholder="填锁具管理表格中的 ID 列数字" />
        </a-form-item>
        <a-form-item label="生效时间" required>
          <a-date-picker v-model:value="grantForm.valid_from" show-time style="width: 100%" />
        </a-form-item>
        <a-form-item label="失效时间（留空为永久）">
          <a-date-picker v-model:value="grantForm.valid_until" show-time style="width: 100%" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getPermissions, grantPermission, revokePermission } from '@/api/admin'
import { formatTime } from '@/utils/format'
import type { Permission } from '@/types'

const permissions = ref<Permission[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const filterStatus = ref<number>()

const showGrantModal = ref(false)
const granting = ref(false)
const grantForm = reactive({
  user_id: undefined as number | undefined,
  device_id: undefined as number | undefined,
  valid_from: null as any,
  valid_until: null as any,
})

const columns = [
  { title: '用户', key: 'user' },
  { title: '设备', key: 'device' },
  { title: '状态', key: 'status', width: 80 },
  { title: '生效时间', key: 'valid_from', width: 170 },
  { title: '失效时间', key: 'valid_until', width: 170 },
  { title: '操作', key: 'actions', width: 80 },
]

onMounted(() => fetchPermissions())

async function fetchPermissions() {
  loading.value = true
  try {
    const data = await getPermissions({
      page: page.value, page_size: pageSize,
      status: filterStatus.value,
    })
    permissions.value = data.items
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  fetchPermissions()
}

async function handleGrant() {
  granting.value = true
  try {
    await grantPermission({
      user_id: grantForm.user_id,
      device_id: grantForm.device_id,
      valid_from: grantForm.valid_from?.toISOString(),
      valid_until: grantForm.valid_until?.toISOString() || null,
    })
    message.success('授权成功')
    showGrantModal.value = false
    fetchPermissions()
  } finally {
    granting.value = false
  }
}

async function handleRevoke(id: number) {
  await revokePermission(id)
  message.success('已撤销授权')
  fetchPermissions()
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
