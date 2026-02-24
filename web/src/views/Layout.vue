<template>
  <a-layout class="app-layout">
    <a-layout-sider v-model:collapsed="collapsed" :trigger="null" collapsible width="240" theme="dark">
      <div class="logo">
        <span v-if="!collapsed">NFC 管控平台</span>
        <span v-else>NFC</span>
      </div>
      <a-menu
        v-model:selectedKeys="selectedKeys"
        theme="dark"
        mode="inline"
      >
        <a-menu-item key="dashboard" @click="$router.push('/dashboard')">
          <template #icon><DashboardOutlined /></template>
          <span>总览</span>
        </a-menu-item>
        <a-menu-item key="users" @click="$router.push('/users')">
          <template #icon><UserOutlined /></template>
          <span>用户管理</span>
        </a-menu-item>
        <a-menu-item key="devices" @click="$router.push('/devices')">
          <template #icon><LockOutlined /></template>
          <span>锁具管理</span>
        </a-menu-item>
        <a-menu-item key="permissions" @click="$router.push('/permissions')">
          <template #icon><SafetyOutlined /></template>
          <span>权限管理</span>
        </a-menu-item>
        <a-menu-item key="audit-logs" @click="$router.push('/audit-logs')">
          <template #icon><FileTextOutlined /></template>
          <span>审计日志</span>
        </a-menu-item>
        <a-menu-item key="alerts" @click="$router.push('/alerts')">
          <template #icon><AlertOutlined /></template>
          <span>
            告警管理
            <a-badge v-if="alertStore.pendingCount > 0" :count="alertStore.pendingCount" :offset="[10, -2]" />
          </span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="app-header">
        <div class="header-left">
          <MenuFoldOutlined v-if="!collapsed" class="trigger" @click="collapsed = true" />
          <MenuUnfoldOutlined v-else class="trigger" @click="collapsed = false" />
          <a-breadcrumb class="breadcrumb">
            <a-breadcrumb-item>{{ currentRoute?.meta?.title || '总览' }}</a-breadcrumb-item>
          </a-breadcrumb>
        </div>
        <div class="header-right">
          <a-badge :count="alertStore.pendingCount" :offset="[-5, 5]">
            <BellOutlined class="header-icon" @click="$router.push('/alerts')" />
          </a-badge>
          <a-dropdown>
            <span class="user-info">
              <a-avatar size="small" style="background-color: #1890ff">
                {{ authStore.name?.charAt(0) }}
              </a-avatar>
              <span class="user-name">{{ authStore.name }}</span>
            </span>
            <template #overlay>
              <a-menu>
                <a-menu-item @click="handleLogout">退出登录</a-menu-item>
              </a-menu>
            </template>
          </a-dropdown>
        </div>
      </a-layout-header>

      <a-layout-content class="app-content">
        <router-view />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  DashboardOutlined, UserOutlined, LockOutlined, SafetyOutlined,
  FileTextOutlined, AlertOutlined, BellOutlined,
  MenuFoldOutlined, MenuUnfoldOutlined,
} from '@ant-design/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useAlertStore } from '@/stores/alert'
import { logout } from '@/api/auth'
import { useAlertPolling } from '@/composables/useAlertPolling'

const authStore = useAuthStore()
const { alertStore } = useAlertPolling()
const route = useRoute()
const router = useRouter()

const collapsed = ref(false)
const selectedKeys = ref<string[]>(['dashboard'])

const currentRoute = computed(() => route)

watch(() => route.path, (path) => {
  const key = path.replace('/', '') || 'dashboard'
  selectedKeys.value = [key]
}, { immediate: true })

async function handleLogout() {
  try {
    await logout()
  } catch {
    // continue even if logout API fails
  }
  authStore.clearAuth()
  message.success('已退出登录')
  router.push('/login')
}
</script>

<style scoped>
.app-layout {
  min-height: 100vh;
}

.logo {
  height: 64px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  font-weight: 600;
  background: rgba(255, 255, 255, 0.05);
}

.app-header {
  background: #fff;
  padding: 0 24px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  box-shadow: 0 1px 4px rgba(0, 0, 0, 0.08);
  height: 64px;
  line-height: 64px;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 16px;
}

.trigger {
  font-size: 18px;
  cursor: pointer;
  transition: color 0.3s;
}

.trigger:hover {
  color: #1890ff;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 24px;
}

.header-icon {
  font-size: 18px;
  cursor: pointer;
}

.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.user-name {
  font-size: 14px;
}

.app-content {
  margin: 24px;
  padding: 24px;
  background: #fff;
  border-radius: 8px;
  min-height: calc(100vh - 112px);
}

.breadcrumb {
  font-size: 14px;
}
</style>
