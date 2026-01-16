import { SERVICE_COLORS, STATUS_COLORS, STATUS_TEXTS, COMMAND_SUPPORTED_TYPES, COMMON_COMMANDS } from './constants'
import type { CommandInfo } from '../types/connection'
import { DatabaseOutlined } from '@ant-design/icons-vue'
import RedisIcon from '../assets/icons/redis.svg'
import MemcachedIcon from '../assets/icons/memcached.svg'
import MySQLIcon from '../assets/icons/mysql.svg'
import PostgreSQLIcon from '../assets/icons/postgresql.svg'
import MongoDBIcon from '../assets/icons/mongodb.svg'
import SQLServerIcon from '../assets/icons/sqlserver.svg'
import OracleIcon from '../assets/icons/oracle.svg'
import ElasticsearchIcon from '../assets/icons/elasticsearch.svg'
import RabbitMQIcon from '../assets/icons/rabbitmq.svg'
import ZookeeperIcon from '../assets/icons/zookeeper.svg'
import EtcdIcon from '../assets/icons/etcd.svg'
import FTPIcon from '../assets/icons/ftp.svg'
import SFTPIcon from '../assets/icons/sftp.svg'
import SSHIcon from '../assets/icons/ssh.svg'
import SMBIcon from '../assets/icons/smb.svg'
import RDPIcon from '../assets/icons/rdp.svg'
import MQTTIcon from '../assets/icons/mqtt.svg'
import WMIIcon from '../assets/icons/wmi.svg'
import ADBIcon from '../assets/icons/adb.svg'
import KafkaIcon from '../assets/icons/kafka.svg'
import JDWPIcon from '../assets/icons/jdwp.svg'
import RMIIcon from '../assets/icons/rmi.svg'
import VNCIcon from '../assets/icons/vnc.svg'
import DockerIcon from '../assets/icons/docker.svg'
import KubernetesIcon from '../assets/icons/kubernetes.svg'

// 获取服务颜色
export function getServiceColor(type: string): string {
  return SERVICE_COLORS[type] || '#1890ff'
}

// 获取状态颜色
export function getStatusColor(status: string): string {
  return STATUS_COLORS[status] || 'default'
}

// 获取状态文本
export function getStatusText(status: string): string {
  return STATUS_TEXTS[status] || status
}

// 获取服务图标
export function getServiceIcon(type: string) {
  const icons: Record<string, any> = {
    Redis: RedisIcon,
    Memcached: MemcachedIcon,
    MySQL: MySQLIcon,
    PostgreSQL: PostgreSQLIcon,
    SQLServer: SQLServerIcon,
    Oracle: OracleIcon,
    MongoDB: MongoDBIcon,
    SSH: SSHIcon,
    FTP: FTPIcon,
    SFTP: SFTPIcon,
    SMB: SMBIcon,
    RDP: RDPIcon,
    RabbitMQ: RabbitMQIcon,
    MQTT: MQTTIcon,
    Kafka: KafkaIcon,
    Elasticsearch: ElasticsearchIcon,
    Zookeeper: ZookeeperIcon,
    Etcd: EtcdIcon,
    WMI: WMIIcon,
    ADB: ADBIcon,
    JDWP: JDWPIcon,
    RMI: RMIIcon,
    VNC: VNCIcon,
    Docker: DockerIcon,
    Kubernetes: KubernetesIcon,
  }
  return icons[type] || DatabaseOutlined
}

// 格式化时间
export function formatTime(time: string): string {
  if (!time) return '-'
  return new Date(time).toLocaleString('zh-CN')
}

// 检查是否支持命令执行
export function supportsCommand(type: string): boolean {
  return COMMAND_SUPPORTED_TYPES.includes(type)
}

// 获取常用命令
export function getCommonCommands(type: string): CommandInfo[] {
  return COMMON_COMMANDS[type] || []
}

// 渲染 VNC 结果（包含 Base64 图片）
export function renderVNCResult(result: string): string {
  const regex = /\[BASE64_IMAGE\](.*?)\[\/BASE64_IMAGE\]/g
  let html = result
  
  html = html.replace(regex, (match, base64Data) => {
    return `<img src="data:image/png;base64,${base64Data}" style="max-width: 100%; height: auto; border: 1px solid #e8e8e8; border-radius: 4px; margin: 10px 0;" alt="VNC Screenshot" />`
  })
  
  html = html.replace(/\n/g, '<br>')
  
  return html
}
