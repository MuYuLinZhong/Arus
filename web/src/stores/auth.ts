import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userUUID = ref(localStorage.getItem('user_uuid') || '')
  const role = ref(localStorage.getItem('role') || '')
  const name = ref(localStorage.getItem('user_name') || '')

  const isLoggedIn = computed(() => !!token.value)
  const isAdmin = computed(() => role.value === 'admin')

  function setAuth(data: { token: string; user_uuid: string; role: string; name: string }) {
    token.value = data.token
    userUUID.value = data.user_uuid
    role.value = data.role
    name.value = data.name

    localStorage.setItem('token', data.token)
    localStorage.setItem('user_uuid', data.user_uuid)
    localStorage.setItem('role', data.role)
    localStorage.setItem('user_name', data.name)
  }

  function clearAuth() {
    token.value = ''
    userUUID.value = ''
    role.value = ''
    name.value = ''

    localStorage.removeItem('token')
    localStorage.removeItem('user_uuid')
    localStorage.removeItem('role')
    localStorage.removeItem('user_name')
  }

  return { token, userUUID, role, name, isLoggedIn, isAdmin, setAuth, clearAuth }
})
