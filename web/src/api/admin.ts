import request from '@/utils/request'
import type { PaginatedData, User, Device, Permission, Alert, AuditLog, DashboardData, PagedData } from '@/types'

// Dashboard
export function getDashboard(): Promise<DashboardData> {
  return request.get('/admin/dashboard')
}

// Users
export function getUsers(params: Record<string, any>): Promise<PaginatedData<User>> {
  return request.get('/admin/users', { params })
}

export function createUser(data: Record<string, any>): Promise<User> {
  return request.post('/admin/users', data)
}

export function updateUser(uuid: string, data: Record<string, any>): Promise<void> {
  return request.put(`/admin/users/${uuid}`, data)
}

export function resetPassword(uuid: string): Promise<void> {
  return request.post(`/admin/users/${uuid}/reset-pwd`)
}

// Devices
export function getDevices(params: Record<string, any>): Promise<PaginatedData<Device>> {
  return request.get('/admin/devices', { params })
}

export function createDevice(data: Record<string, any>): Promise<Device> {
  return request.post('/admin/devices', data)
}

// Permissions
export function getPermissions(params: Record<string, any>): Promise<PaginatedData<Permission>> {
  return request.get('/admin/permissions', { params })
}

export function grantPermission(data: Record<string, any>): Promise<void> {
  return request.post('/admin/permissions', data)
}

export function batchGrantPermissions(data: { permissions: any[] }): Promise<void> {
  return request.post('/admin/permissions/batch', data)
}

export function revokePermission(id: number): Promise<void> {
  return request.delete(`/admin/permissions/${id}`)
}

// Audit Logs
export function getAuditLogs(params: Record<string, any>): Promise<PagedData<AuditLog>> {
  return request.get('/admin/audit-logs', { params })
}

// Alerts
export function getAlerts(params: Record<string, any>): Promise<PaginatedData<Alert>> {
  return request.get('/admin/alerts', { params })
}

export function handleAlert(id: number, data: { handle_note: string; unlock_device: boolean }): Promise<void> {
  return request.put(`/admin/alerts/${id}`, data)
}
