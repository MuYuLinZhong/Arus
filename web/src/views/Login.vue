<template>
  <div class="login-container">
    <div class="login-card">
      <div class="login-header">
        <h1>石油阀门 NFC 智能管控系统</h1>
        <p>防盗安全预警系列 · Web 管控平台</p>
      </div>
      <a-form :model="form" layout="vertical" class="login-form">
        <a-form-item label="手机号">
          <a-input
            v-model:value="form.phone"
            placeholder="请输入手机号"
            size="large"
            :prefix="h(PhoneOutlined)"
          />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password
            v-model:value="form.password"
            placeholder="请输入密码"
            size="large"
            :prefix="h(LockOutlined)"
          />
        </a-form-item>
        <a-form-item>
          <a-button
            type="primary"
            :loading="loading"
            size="large"
            block
            @click="handleLogin"
          >
            登录
          </a-button>
        </a-form-item>
      </a-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, h } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import { PhoneOutlined, LockOutlined } from '@ant-design/icons-vue'
import { login } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()
const loading = ref(false)

const form = reactive({
  phone: '',
  password: '',
})

async function handleLogin() {
  const phone = form.phone?.trim() ?? ''
  const password = form.password ?? ''
  if (!phone) {
    message.warning('请输入手机号')
    return
  }
  if (!password) {
    message.warning('请输入密码')
    return
  }
  loading.value = true
  try {
    const result = await login({ phone, password })
    authStore.setAuth(result)
    message.success(`欢迎回来，${result.name}`)
    router.push('/dashboard')
  } catch (err) {
    console.error('[Login]', err)
    // message 已由 request 拦截器统一展示
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0c1e3c 0%, #1a3a5c 50%, #0d2137 100%);
}

.login-card {
  width: 420px;
  padding: 48px 40px;
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
}

.login-header {
  text-align: center;
  margin-bottom: 40px;
}

.login-header h1 {
  font-size: 22px;
  font-weight: 600;
  color: #1a1a2e;
  margin-bottom: 8px;
}

.login-header p {
  font-size: 14px;
  color: #8c8c8c;
}

.login-form :deep(.ant-btn-primary) {
  height: 44px;
  font-size: 16px;
  background: linear-gradient(135deg, #1890ff, #096dd9);
  border: none;
}
</style>
