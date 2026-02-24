import axios, { type AxiosInstance, type AxiosResponse, type InternalAxiosRequestConfig } from 'axios'
import { message } from 'ant-design-vue'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import type { ApiResponse } from '@/types'

function requestId(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  const hex = '0123456789abcdef'
  let s = ''
  for (let i = 0; i < 36; i++) {
    if (i === 8 || i === 13 || i === 18 || i === 23) s += '-'
    else if (i === 14) s += '4'
    else s += hex[Math.floor(Math.random() * 16)]
  }
  return s
}

const request: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 10000,
})

request.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = typeof localStorage !== 'undefined' ? localStorage.getItem('token') : null
    if (token) {
      config.headers.Authorization = `Bearer ${token}`
    }
    config.headers['X-Request-ID'] = requestId()
    return config
  },
  (error) => Promise.reject(error)
)

request.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    const { data } = response

    if (data.code === 0) {
      return data.data
    }

    if (data.code === 1001 || data.code === 1003) {
      const authStore = useAuthStore()
      authStore.clearAuth()
      router.push('/login')
      message.error('会话已过期，请重新登录')
      return Promise.reject(new Error(data.message))
    }

    if (data.code === 1002) {
      const authStore = useAuthStore()
      authStore.clearAuth()
      router.push('/login')
      message.error('账号已被禁用')
      return Promise.reject(new Error(data.message))
    }

    if (data.code >= 2000 && data.code < 3000) {
      message.error('权限不足：' + data.message)
      return Promise.reject(new Error(data.message))
    }

    message.error(data.message || '请求失败')
    return Promise.reject(new Error(data.message))
  },
  (error) => {
    if (error.response) {
      const status = error.response.status
      const msg = error.response.data?.message
      if (status === 401) {
        const authStore = useAuthStore()
        authStore.clearAuth()
        router.push('/login')
      } else if (status === 400) {
        message.error(msg || '请求参数错误')
      } else if (status === 429) {
        message.warning('请求过于频繁，请稍后再试')
      } else if (status >= 500) {
        message.error('服务异常，请稍后重试')
      }
    } else if (error.code === 'ECONNABORTED') {
      message.error('请求超时，请检查网络')
    } else {
      message.error('网络异常')
    }
    return Promise.reject(error)
  }
)

export default request
