import request from '@/utils/request'
import type { LoginForm, LoginResult } from '@/types'

export function login(data: LoginForm): Promise<LoginResult> {
  return request.post('/auth/login', data)
}

export function logout(): Promise<void> {
  return request.post('/auth/logout')
}
