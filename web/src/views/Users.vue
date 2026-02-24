<template>
  <div>
    <div class="page-header">
      <h2>用户管理</h2>
      <a-button type="primary" @click="showCreateModal = true">新建用户</a-button>
    </div>

    <a-space style="margin-bottom: 16px">
      <a-input-search v-model:value="searchText" placeholder="搜索用户名" @search="fetchUsers" style="width: 250px" />
      <a-select v-model:value="filterRole" placeholder="角色" allow-clear style="width: 120px" @change="fetchUsers">
        <a-select-option value="admin">管理员</a-select-option>
        <a-select-option value="user">普通用户</a-select-option>
      </a-select>
      <a-select v-model:value="filterStatus" placeholder="状态" allow-clear style="width: 120px" @change="fetchUsers">
        <a-select-option value="1">启用</a-select-option>
        <a-select-option value="0">禁用</a-select-option>
      </a-select>
    </a-space>

    <a-table
      :columns="columns"
      :data-source="users"
      :loading="loading"
      :pagination="{ current: page, pageSize, total, onChange: onPageChange }"
      row-key="uuid"
    >
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'role'">
          <a-tag :color="record.role === 'admin' ? 'blue' : 'default'">
            {{ record.role === 'admin' ? '管理员' : '普通用户' }}
          </a-tag>
        </template>
        <template v-if="column.key === 'status'">
          <a-badge :status="record.status === 1 ? 'success' : 'error'" :text="record.status === 1 ? '启用' : '禁用'" />
        </template>
        <template v-if="column.key === 'created_at'">
          {{ formatTime(record.created_at) }}
        </template>
        <template v-if="column.key === 'actions'">
          <a-space>
            <a @click="editUser(record)">编辑</a>
            <a-popconfirm title="确认重置密码？新密码将通过短信发送" @confirm="handleResetPwd(record.uuid)">
              <a>重置密码</a>
            </a-popconfirm>
            <a-popconfirm
              :title="record.status === 1 ? '确认禁用该用户？将立即踢出所有登录设备' : '确认启用该用户？'"
              @confirm="toggleStatus(record)"
            >
              <a :style="{ color: record.status === 1 ? '#ff4d4f' : '#52c41a' }">
                {{ record.status === 1 ? '禁用' : '启用' }}
              </a>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="showCreateModal" title="新建用户" @ok="handleCreate" :confirm-loading="creating">
      <a-form :model="createForm" layout="vertical">
        <a-form-item label="手机号" required>
          <a-input v-model:value="createForm.phone" placeholder="请输入手机号" />
        </a-form-item>
        <a-form-item label="姓名" required>
          <a-input v-model:value="createForm.name" placeholder="请输入姓名" />
        </a-form-item>
        <a-form-item label="部门">
          <a-input v-model:value="createForm.department" placeholder="请输入部门" />
        </a-form-item>
        <a-form-item label="角色" required>
          <a-select v-model:value="createForm.role">
            <a-select-option value="user">普通用户</a-select-option>
            <a-select-option value="admin">管理员</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>

    <a-modal v-model:open="showEditModal" title="编辑用户" @ok="handleUpdate" :confirm-loading="updating">
      <a-form :model="editForm" layout="vertical">
        <a-form-item label="姓名">
          <a-input v-model:value="editForm.name" />
        </a-form-item>
        <a-form-item label="部门">
          <a-input v-model:value="editForm.department" />
        </a-form-item>
        <a-form-item label="角色">
          <a-select v-model:value="editForm.role">
            <a-select-option value="user">普通用户</a-select-option>
            <a-select-option value="admin">管理员</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { message } from 'ant-design-vue'
import { getUsers, createUser, updateUser, resetPassword } from '@/api/admin'
import { formatTime } from '@/utils/format'
import type { User } from '@/types'

const users = ref<User[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = 20
const total = ref(0)
const searchText = ref('')
const filterRole = ref<string>()
const filterStatus = ref<string>()

const showCreateModal = ref(false)
const creating = ref(false)
const createForm = reactive({ phone: '', name: '', department: '', role: 'user' })

const showEditModal = ref(false)
const updating = ref(false)
const editForm = reactive({ uuid: '', name: '', department: '', role: '' })

const columns = [
  { title: 'ID', dataIndex: 'id', key: 'id', width: 70 },
  { title: '姓名', dataIndex: 'name', key: 'name' },
  { title: '手机号', dataIndex: 'phone', key: 'phone' },
  { title: '部门', dataIndex: 'department', key: 'department' },
  { title: '角色', key: 'role', width: 100 },
  { title: '状态', key: 'status', width: 80 },
  { title: '创建时间', key: 'created_at', width: 170 },
  { title: '操作', key: 'actions', width: 200 },
]

onMounted(() => fetchUsers())

async function fetchUsers() {
  loading.value = true
  try {
    const data = await getUsers({
      page: page.value, page_size: pageSize,
      search: searchText.value || undefined,
      role: filterRole.value, status: filterStatus.value,
    })
    users.value = data.items
    total.value = data.total
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) {
  page.value = p
  fetchUsers()
}

async function handleCreate() {
  creating.value = true
  try {
    await createUser(createForm)
    message.success('用户创建成功，初始密码已通过短信发送')
    showCreateModal.value = false
    Object.assign(createForm, { phone: '', name: '', department: '', role: 'user' })
    fetchUsers()
  } finally {
    creating.value = false
  }
}

function editUser(record: User) {
  editForm.uuid = record.uuid
  editForm.name = record.name
  editForm.department = record.department || ''
  editForm.role = record.role
  showEditModal.value = true
}

async function handleUpdate() {
  updating.value = true
  try {
    await updateUser(editForm.uuid, {
      name: editForm.name,
      department: editForm.department,
      role: editForm.role,
    })
    message.success('更新成功')
    showEditModal.value = false
    fetchUsers()
  } finally {
    updating.value = false
  }
}

async function handleResetPwd(uuid: string) {
  await resetPassword(uuid)
  message.success('密码已重置，新密码已通过短信发送')
}

async function toggleStatus(record: User) {
  const newStatus = record.status === 1 ? 0 : 1
  await updateUser(record.uuid, { status: newStatus })
  message.success(newStatus === 1 ? '已启用' : '已禁用')
  fetchUsers()
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.page-header h2 {
  margin: 0;
  font-size: 20px;
}
</style>
