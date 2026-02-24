export interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
  request_id: string
  timestamp: number
}

export interface PagedData<T = any> {
  items: T[]
  next_cursor: string
  has_more: boolean
}

export interface PaginatedData<T = any> {
  items: T[]
  total: number
}

export interface User {
  id: number
  uuid: string
  name: string
  phone: string
  department: string
  role: 'user' | 'admin'
  status: number
  created_at: string
  updated_at: string
}

export interface Device {
  id: number
  device_id: string
  name: string
  location_text: string
  longitude?: number
  latitude?: number
  pipeline_tag?: string
  risk_level: number
  key_version: number
  status: number
  last_active_at?: string
  created_at: string
}

export interface Permission {
  id: number
  user_id: number
  device_id: number
  granted_by: number
  valid_from: string
  valid_until?: string
  status: number
  revoked_by?: number
  revoked_at?: string
  created_at: string
  user?: User
  device?: Device
}

export interface AuditLog {
  id: number
  user_id: number
  device_id: string
  action: string
  result_code: number
  client_ip: string
  device_model: string
  extra?: Record<string, any>
  occurred_at: string
}

export interface Alert {
  id: number
  alert_type: string
  device_id: string
  user_id?: number
  severity: number
  status: number
  handled_by?: number
  handle_note?: string
  extra?: Record<string, any>
  created_at: string
  handled_at?: string
}

export interface DashboardData {
  total_users: number
  total_devices: number
  active_sessions: number
  pending_alerts: number
  recent_alerts: Alert[]
  devices_by_status: Record<string, number>
}

export interface LoginForm {
  phone: string
  password: string
}

export interface LoginResult {
  token: string
  expires_at: string
  user_uuid: string
  role: string
  name: string
}
