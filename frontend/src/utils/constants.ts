import type { CommandInfo } from '../types/connection'

// 服务颜色映射
export const SERVICE_COLORS: Record<string, string> = {
  Redis: '#dc382d',
  Memcached: '#00a8e1',
  MySQL: '#00758f',
  PostgreSQL: '#336791',
  SQLServer: '#cc2927',
  Oracle: '#f80000',
  MongoDB: '#47a248',
  SSH: '#000000',
  FTP: '#ffa500',
  SFTP: '#4CAF50',
  SMB: '#0078d4',
  RabbitMQ: '#ff6600',
  MQTT: '#660066',
  Elasticsearch: '#005571',
  Zookeeper: '#d22128',
  Etcd: '#419EDA',
  WMI: '#0078d4',
  ADB: '#3DDC84',
  Kafka: '#231F20',
  JDWP: '#EA2D2E',
  RMI: '#EA2D2E',
  RDP: '#0078D4',
  VNC: '#E74C3C',
  Docker: '#2496ED',
  Kubernetes: '#326CE5',
}

// 状态颜色映射
export const STATUS_COLORS: Record<string, string> = {
  success: 'success',
  failed: 'error',
  pending: 'default',
}

// 状态文本映射
export const STATUS_TEXTS: Record<string, string> = {
  success: '成功',
  failed: '失败',
  pending: '待连接',
}

// 支持命令执行的服务类型
export const COMMAND_SUPPORTED_TYPES = [
  'Redis', 'Memcached', 'MySQL', 'PostgreSQL', 'MongoDB', 
  'SSH', 'SQLServer', 'Oracle', 'Zookeeper', 'Elasticsearch', 
  'ADB', 'Kafka', 'Etcd', 'JDWP', 'VNC', 'RDP', 'Docker', 'Kubernetes'
]

// 常用命令映射
export const COMMON_COMMANDS: Record<string, CommandInfo[]> = {
  Redis: [
    { label: 'INFO', command: 'INFO', description: '查看服务器信息' },
    { label: 'KEYS *', command: 'KEYS *', description: '列出所有键' },
    { label: 'DBSIZE', command: 'DBSIZE', description: '查看键数量' },
    { label: 'CONFIG GET *', command: 'CONFIG GET *', description: '查看配置' },
  ],
  Memcached: [
    { label: 'stats', command: 'stats', description: '查看服务器统计信息' },
    { label: 'stats items', command: 'stats items', description: '查看项目统计' },
    { label: 'stats slabs', command: 'stats slabs', description: '查看 slab 统计' },
    { label: 'stats sizes', command: 'stats sizes', description: '查看大小统计' },
    { label: 'version', command: 'version', description: '查看版本信息' },
  ],
  MySQL: [
    { label: 'SHOW DATABASES', command: 'SHOW DATABASES', description: '显示所有数据库' },
    { label: 'SHOW TABLES', command: 'SHOW TABLES', description: '显示所有表' },
    { label: 'SELECT VERSION()', command: 'SELECT VERSION()', description: '查看版本' },
    { label: 'SHOW VARIABLES', command: 'SHOW VARIABLES', description: '查看变量' },
  ],
  PostgreSQL: [
    { label: '\\l', command: '\\l', description: '列出数据库' },
    { label: '\\dt', command: '\\dt', description: '列出表' },
    { label: 'SELECT version()', command: 'SELECT version()', description: '查看版本' },
    { label: '\\du', command: '\\du', description: '列出用户' },
  ],
  MongoDB: [
    { label: 'show dbs', command: 'show dbs', description: '显示所有数据库' },
    { label: 'show collections', command: 'show collections', description: '显示集合' },
    { label: 'db.version()', command: 'db.version()', description: '查看版本' },
    { label: 'db.stats()', command: 'db.stats()', description: '数据库统计' },
  ],
  SSH: [
    { label: 'whoami', command: 'whoami', description: '当前用户' },
    { label: 'pwd', command: 'pwd', description: '当前目录' },
    { label: 'ls -la', command: 'ls -la', description: '列出文件' },
    { label: 'uname -a', command: 'uname -a', description: '系统信息' },
  ],
  SQLServer: [
    { label: 'SELECT @@VERSION', command: 'SELECT @@VERSION', description: '查看版本' },
    { label: 'SELECT name FROM sys.databases', command: 'SELECT name FROM sys.databases', description: '列出数据库' },
    { label: 'SELECT @@SERVERNAME', command: 'SELECT @@SERVERNAME', description: '服务器名称' },
  ],
  Oracle: [
    { label: 'SELECT * FROM v$version', command: 'SELECT * FROM v$version', description: '查看版本' },
    { label: 'SELECT username FROM all_users', command: 'SELECT username FROM all_users', description: '列出用户' },
    { label: 'SELECT table_name FROM user_tables', command: 'SELECT table_name FROM user_tables', description: '列出表' },
  ],
  Zookeeper: [
    { label: 'stat', command: 'stat', description: '服务器统计信息' },
    { label: 'srvr', command: 'srvr', description: '服务器详细信息' },
    { label: 'mntr', command: 'mntr', description: '监控信息' },
    { label: 'conf', command: 'conf', description: '配置信息' },
    { label: 'envi', command: 'envi', description: '环境变量' },
    { label: 'ruok', command: 'ruok', description: '服务器是否运行' },
    { label: 'cons', command: 'cons', description: '连接信息' },
    { label: 'dump', command: 'dump', description: '会话和临时节点' },
  ],
  Elasticsearch: [
    { label: 'health', command: 'health', description: '集群健康状态' },
    { label: 'stats', command: 'stats', description: '集群统计信息' },
    { label: 'nodes', command: 'nodes', description: '节点信息' },
    { label: 'indices', command: 'indices', description: '索引列表' },
    { label: 'shards', command: 'shards', description: '分片信息' },
    { label: 'allocation', command: 'allocation', description: '分配信息' },
    { label: 'count', command: 'count', description: '文档数量' },
    { label: 'settings', command: 'settings', description: '集群设置' },
    { label: 'version', command: 'version', description: '版本信息' },
    { label: 'tasks', command: 'tasks', description: '任务列表' },
  ],
  ADB: [
    { label: 'getprop', command: 'getprop', description: '查看所有系统属性' },
    { label: 'pm list packages', command: 'pm list packages', description: '列出所有应用包' },
    { label: 'dumpsys battery', command: 'dumpsys battery', description: '电池信息' },
    { label: 'dumpsys meminfo', command: 'dumpsys meminfo', description: '内存信息' },
    { label: 'ps', command: 'ps', description: '进程列表' },
    { label: 'netstat', command: 'netstat', description: '网络连接' },
    { label: 'df', command: 'df', description: '磁盘使用情况' },
    { label: 'ls /sdcard', command: 'ls /sdcard', description: '列出SD卡文件' },
  ],
  Kafka: [
    { label: 'cluster-info', command: 'cluster-info', description: '查看集群信息' },
    { label: 'version', command: 'version', description: '查看 Broker 版本' },
    { label: 'topics', command: 'topics', description: '列出所有 Topic' },
    { label: 'brokers', command: 'brokers', description: '列出所有 Broker' },
    { label: 'broker-info', command: 'broker-info <broker-id>', description: '查看 Broker 详细信息' },
    { label: 'config', command: 'config <broker-id>', description: '查看 Broker 配置' },
    { label: 'consumer-groups', command: 'consumer-groups', description: '列出消费者组' },
    { label: 'describe-topic', command: 'describe-topic <topic-name>', description: '描述指定 Topic' },
  ],
  Etcd: [
    { label: 'status', command: 'status', description: '获取状态信息' },
    { label: 'version', command: 'version', description: '获取版本信息' },
    { label: 'member', command: 'member', description: '列出集群成员' },
    { label: 'list', command: 'list /', description: '列出所有键' },
    { label: 'get', command: 'get <key>', description: '获取键值' },
    { label: 'put', command: 'put <key> <value>', description: '设置键值' },
    { label: 'del', command: 'del <key>', description: '删除键' },
    { label: 'compact', command: 'compact <revision>', description: '压缩历史版本' },
  ],
  JDWP: [
    { label: 'whoami (Linux)', command: 'whoami', description: '查看当前用户' },
    { label: 'id (Linux)', command: 'id', description: '查看用户ID信息' },
    { label: 'pwd (Linux)', command: 'pwd', description: '查看当前目录' },
    { label: 'ls -la (Linux)', command: 'ls -la', description: '列出文件' },
    { label: 'calc (Windows)', command: 'calc', description: '打开计算器' },
    { label: 'notepad (Windows)', command: 'notepad', description: '打开记事本' },
    { label: 'cmd /c whoami (Windows)', command: 'cmd /c whoami', description: 'Windows查看用户' },
    { label: 'bash -c "id" (Linux)', command: 'bash -c "id"', description: 'Bash执行命令' },
  ],
  VNC: [
    { label: 'screenshot', command: 'screenshot', description: '获取屏幕截图' },
  ],
  RDP: [
    { label: 'screenshot', command: 'screenshot', description: '获取屏幕截图' },
  ],
  Docker: [
    { label: 'ps', command: 'ps', description: '列出所有容器' },
    { label: 'containers', command: 'containers', description: '列出所有容器（详细）' },
    { label: 'images', command: 'images', description: '列出所有镜像' },
    { label: 'info', command: 'info', description: '查看系统信息' },
    { label: 'version', command: 'version', description: '查看版本信息' },
  ],
  Kubernetes: [
    { label: 'namespaces', command: 'namespaces', description: '列出所有命名空间' },
    { label: 'pods', command: 'pods', description: '列出所有 Pod' },
    { label: 'secrets', command: 'secrets', description: '列出 Secret（前10个）' },
    { label: 'version', command: 'version', description: '查看 Kubernetes 版本' },
  ],
}
