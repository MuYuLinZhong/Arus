import dayjs from 'dayjs'

export function formatTime(time: string | undefined): string {
  if (!time) return '-'
  return dayjs(time).format('YYYY-MM-DD HH:mm:ss')
}

export function formatDate(time: string | undefined): string {
  if (!time) return '-'
  return dayjs(time).format('YYYY-MM-DD')
}

export function maskPhone(phone: string): string {
  if (!phone || phone.length < 7) return '****'
  return phone.slice(0, 3) + '****' + phone.slice(-4)
}

export const deviceStatusMap: Record<number, { text: string; color: string }> = {
  0: { text: '已禁用', color: '#999' },
  1: { text: '正常', color: '#52c41a' },
  2: { text: '告警锁定', color: '#ff4d4f' },
}

export const alertSeverityMap: Record<number, { text: string; color: string }> = {
  1: { text: '低', color: '#1890ff' },
  2: { text: '中', color: '#faad14' },
  3: { text: '高', color: '#ff4d4f' },
}

export const alertStatusMap: Record<number, { text: string; color: string }> = {
  0: { text: '待处置', color: '#ff4d4f' },
  1: { text: '已处置', color: '#52c41a' },
  2: { text: '已忽略', color: '#999' },
}

export const alertTypeMap: Record<string, string> = {
  consecutive_fail: '连续校验失败',
  challenge_flood: '挑战请求洪泛',
  off_hours_attempt: '非工作时段操作',
  device_offline: '设备离线',
}

export const auditActionMap: Record<string, string> = {
  unlock_success: '开锁成功',
  unlock_fail: '开锁失败',
  challenge_request: '挑战请求',
  challenge_denied: '挑战被拒绝',
  auth_fail: '登录失败',
}

export const riskLevelMap: Record<number, { text: string; color: string }> = {
  1: { text: '普通', color: '#1890ff' },
  2: { text: '重要', color: '#faad14' },
  3: { text: '关键', color: '#ff4d4f' },
}
