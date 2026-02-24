import request from '@/utils/request'
import type { Device } from '@/types'

export function getAuthorizedDevices(): Promise<Device[]> {
  return request.get('/lock/devices')
}

export function submitChallenge(data: {
  device_id: string
  challenge_c: string
  timestamp: number
}): Promise<{ response: string }> {
  return request.post('/lock/challenge', data)
}

export function reportResult(data: {
  device_id: string
  result: 'success' | 'fail'
  fail_reason?: string
  occurred_at: number
  device_model?: string
}): Promise<void> {
  return request.post('/lock/report', data)
}
