// 连接类型定义
export interface Connection {
  id: string
  type: string
  ip: string
  port: string
  user: string
  pass: string
  status: 'success' | 'failed' | 'pending'
  message: string
  logs: string[]
  result: string
  created_at: string
  connected_at?: string
}

export interface ConnectionRequest {
  type: string
  ip: string
  port: string
  user: string
  pass: string
}

export interface ProxyConfig {
  enabled: boolean
  type: string
  host: string
  port: string
  user: string
  pass: string
}

export interface ServiceType {
  value: string
  label: string
  port: string
}

export interface CommandInfo {
  label: string
  command: string
  description: string
}
