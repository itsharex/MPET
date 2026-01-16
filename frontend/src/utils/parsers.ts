import { AddConnection } from '../../wailsjs/go/backend/App'

// 服务类型标准化映射
const FSCAN_TYPE_MAP: Record<string, string> = {
  'redis': 'Redis',
  'mysql': 'MySQL',
  'mssql': 'SQLServer',
  'sqlserver': 'SQLServer',
  'oracle': 'Oracle',
  'postgres': 'PostgreSQL',
  'postgresql': 'PostgreSQL',
  'mongodb': 'MongoDB',
  'mongo': 'MongoDB',
  'ssh': 'SSH',
  'ftp': 'FTP',
  'smb': 'SMB',
  'rdp': 'RDP',
  'memcached': 'Memcached',
  'elasticsearch': 'Elasticsearch',
  'es': 'Elasticsearch',
  'rabbitmq': 'RabbitMQ',
  'mqtt': 'MQTT',
  'zookeeper': 'Zookeeper',
  'etcd': 'Etcd',
  'kafka': 'Kafka',
  'vnc': 'VNC',
}

const LIGHTX_TYPE_MAP: Record<string, string> = {
  ...FSCAN_TYPE_MAP,
  // 'sybase': 'SQLServer',
  'sftp': 'SFTP',
  // 'clickhouse': 'MySQL',
  // 'influxdb': 'Elasticsearch',
  // 'iotdb': 'MySQL',
  'docker': 'Docker',
  'kubernetes': 'Kubernetes',
  // 'coap': 'MQTT',
  // 'hazelcast': 'Redis',
  // 'consul': 'Etcd',
  // 'couchdb': 'MongoDB',
  // 'radius': 'SSH',
  // 'winrm': 'SSH',
  'adb': 'ADB',
  // 'telnet': 'SSH',
  'jdwp': 'JDWP',
}

// 标准化 fscan 服务类型名称
export function normalizeFscanServiceType(fscanType: string): string {
  const lowerType = fscanType.toLowerCase()
  
  if (FSCAN_TYPE_MAP[lowerType]) {
    return FSCAN_TYPE_MAP[lowerType]
  }
  
  if (fscanType.length > 0) {
    return fscanType.charAt(0).toUpperCase() + fscanType.slice(1).toLowerCase()
  }
  
  return ''
}

// 标准化 lightx 服务类型名称
export function normalizeLightxServiceType(lightxType: string): string {
  const lowerType = lightxType.toLowerCase()
  
  if (LIGHTX_TYPE_MAP[lowerType]) {
    return LIGHTX_TYPE_MAP[lowerType]
  }
  
  if (lightxType.length > 0) {
    return lightxType.charAt(0).toUpperCase() + lightxType.slice(1).toLowerCase()
  }
  
  return ''
}

// 解析 CSV 内容
export async function parseCSVContent(text: string): Promise<{ success: number; failed: number }> {
  const lines = text.split('\n').filter(line => line.trim())
  if (lines.length === 0) {
    throw new Error('CSV 文件为空')
  }
  
  const startIndex = lines[0].toLowerCase().includes('type') || lines[0].toLowerCase().includes('ip') ? 1 : 0
  let successCount = 0
  let failCount = 0
  
  for (let i = startIndex; i < lines.length; i++) {
    const line = lines[i].trim()
    if (!line) continue
    
    const parts = line.split(',').map(p => p.trim())
    if (parts.length < 3) continue
    
    try {
      const connData = {
        type: parts[0],
        ip: parts[1],
        port: parts[2],
        user: parts[3] || '',
        pass: parts[4] || '',
      }
      
      await AddConnection(connData)
      successCount++
    } catch (error) {
      console.error(`导入第 ${i + 1} 行失败:`, error)
      failCount++
    }
  }
  
  return { success: successCount, failed: failCount }
}

// 解析 fscan 1.8.4 结果文件
export async function parseFscanResult(text: string): Promise<{ success: number; failed: number }> {
  const lines = text.split('\n').filter(line => line.trim())
  let successCount = 0
  let failCount = 0
  
  for (const line of lines) {
    const trimmedLine = line.trim()
    
    if (!trimmedLine.startsWith('[+]')) {
      continue
    }
    
    try {
      const content = trimmedLine.substring(3).trim()
      const parts = content.split(/\s+/)
      if (parts.length < 1) {
        failCount++
        continue
      }
      
      const serviceInfo = parts[0]
      const serviceParts = serviceInfo.split(':')
      if (serviceParts.length < 3) {
        failCount++
        continue
      }
      
      let type = serviceParts[0].trim()
      const ip = serviceParts[1].trim()
      const port = serviceParts[2].trim()
      
      type = normalizeFscanServiceType(type)
      if (!type) {
        failCount++
        continue
      }
      
      let user = ''
      let pass = ''
      
      if (parts.length >= 2) {
        const credInfo = parts.slice(1).join(' ')
        
        if (credInfo.toLowerCase().includes('unauthorized') || 
            credInfo.toLowerCase().includes('unauth') ||
            credInfo.toLowerCase().includes('no auth')) {
          // 未授权访问
        } else if (credInfo.includes(':')) {
          const credParts = credInfo.split(':')
          if (credParts.length >= 2) {
            user = credParts[0].trim()
            pass = credParts.slice(1).join(':').trim()
          }
        } else {
          user = credInfo.trim()
        }
      }
      
      await AddConnection({ type, ip, port, user, pass })
      successCount++
    } catch (error) {
      console.error('解析 fscan 行失败:', line, error)
      failCount++
    }
  }
  
  return { success: successCount, failed: failCount }
}

// 解析 fscan 2.1.1 结果文件
export async function parseFscan21Result(text: string): Promise<{ success: number; failed: number }> {
  const lines = text.split('\n').filter(line => line.trim())
  let successCount = 0
  let failCount = 0
  let inVulnSection = false
  
  const seen = new Set<string>()
  
  for (const line of lines) {
    const trimmedLine = line.trim()
    
    if (trimmedLine.includes('# ===== 漏洞信息 =====')) {
      inVulnSection = true
      continue
    }
    
    if (trimmedLine.startsWith('# =====') && !trimmedLine.includes('漏洞信息')) {
      inVulnSection = false
      continue
    }
    
    if (!inVulnSection || trimmedLine === '' || trimmedLine.startsWith('#')) {
      continue
    }
    
    try {
      const parts = trimmedLine.split(/\s+/)
      if (parts.length < 2) {
        failCount++
        continue
      }
      
      const ipPort = parts[0]
      const ipPortParts = ipPort.split(':')
      if (ipPortParts.length !== 2) {
        failCount++
        continue
      }
      
      const ip = ipPortParts[0].trim()
      const port = ipPortParts[1].trim()
      
      let type = parts[1].trim()
      type = normalizeFscanServiceType(type)
      if (!type) {
        failCount++
        continue
      }
      
      let user = ''
      let pass = ''
      
      if (parts.length >= 3) {
        const credInfo = parts[2]
        
        if (trimmedLine.toLowerCase().includes('未授权') || 
            trimmedLine.toLowerCase().includes('unauthorized') ||
            credInfo === '/') {
          // 未授权访问
        } else if (credInfo.includes('/')) {
          const credParts = credInfo.split('/')
          if (credParts.length >= 1) {
            user = credParts[0].trim()
          }
          if (credParts.length >= 2) {
            pass = credParts.slice(1).join('/').trim()
          }
        } else {
          user = credInfo.trim()
        }
      }
      
      const uniqueKey = `${type}:${ip}:${port}:${user}:${pass}`
      if (seen.has(uniqueKey)) {
        continue
      }
      seen.add(uniqueKey)
      
      await AddConnection({ type, ip, port, user, pass })
      successCount++
    } catch (error) {
      console.error('解析 fscan 2.1.1 行失败:', line, error)
      failCount++
    }
  }
  
  return { success: successCount, failed: failCount }
}

// 解析 lightx 结果文件
export async function parseLightxResult(text: string): Promise<{ success: number; failed: number }> {
  const lines = text.split('\n').filter(line => line.trim())
  let successCount = 0
  let failCount = 0
  
  const seen = new Set<string>()
  
  for (const line of lines) {
    const trimmedLine = line.trim()
    
    if (!trimmedLine.includes('[Plugin:') || !trimmedLine.includes(':SUCCESS]')) {
      continue
    }
    
    try {
      const pluginStart = trimmedLine.indexOf('[Plugin:')
      const pluginEnd = trimmedLine.indexOf(':SUCCESS]')
      if (pluginStart === -1 || pluginEnd === -1) {
        failCount++
        continue
      }
      
      const serviceType = trimmedLine.substring(pluginStart + 8, pluginEnd)
      
      if (serviceType === 'NetInfo') {
        continue
      }
      
      const contentStart = pluginEnd + 9
      if (contentStart >= trimmedLine.length) {
        failCount++
        continue
      }
      const mainContent = trimmedLine.substring(contentStart).trim()
      
      const parts = mainContent.split(/\s+/)
      if (parts.length < 1) {
        failCount++
        continue
      }
      
      const serviceInfo = parts[0]
      const serviceParts = serviceInfo.split(':')
      if (serviceParts.length < 3) {
        failCount++
        continue
      }
      
      let type = serviceParts[0].trim()
      const ip = serviceParts[1].trim()
      const port = serviceParts[2].trim()
      
      type = normalizeLightxServiceType(type)
      if (!type) {
        failCount++
        continue
      }
      
      let user = ''
      let pass = ''
      
      if (parts.length >= 2) {
        const credInfo = parts[1]
        
        if (trimmedLine.toLowerCase().includes('未授权访问') || 
            trimmedLine.toLowerCase().includes('unauthorized') ||
            trimmedLine.toLowerCase().includes('匿名登录')) {
          if (credInfo.includes('anonymous')) {
            user = 'anonymous'
          }
        } else if (credInfo.includes('/')) {
          const credParts = credInfo.split('/')
          if (credParts.length >= 1) {
            user = credParts[0].trim()
          }
          if (credParts.length >= 2) {
            pass = credParts.slice(1).join('/').trim()
          }
        }
      }
      
      const uniqueKey = `${type}:${ip}:${port}:${user}:${pass}`
      if (seen.has(uniqueKey)) {
        continue
      }
      seen.add(uniqueKey)
      
      await AddConnection({ type, ip, port, user, pass })
      successCount++
    } catch (error) {
      console.error('解析 lightx 行失败:', line, error)
      failCount++
    }
  }
  
  return { success: successCount, failed: failCount }
}
