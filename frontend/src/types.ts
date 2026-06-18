export interface Profile {
  name: string
  listen_ip: string
  port: number
  hosts_file?: string
  running: boolean
  hosts: Record<string, string>
  system_hosts_active?: boolean
  proxy_active?: boolean
  proxy_error?: string
  type?: string
  subscription_url?: string
  subscription_interval?: number
  subscription_enabled?: boolean
  subscription_last_fetch?: string
}

export interface SubscriptionResult {
  status: string
  message: string
  last_fetch: string
  entry_count: number
}

export interface ParsedLine {
  lineNo: number
  raw: string
  type: 'mapping' | 'comment' | 'blank' | 'invalid'
  enabled?: boolean
  ip?: string
  domain?: string
}

export interface BackupInfo {
  file_name: string
  path: string
  size: number
  modified: string
}
