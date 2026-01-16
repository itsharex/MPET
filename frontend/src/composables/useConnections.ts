import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { 
  GetConnections, 
  AddConnection, 
  ConnectSingle, 
  ConnectBatch, 
  UpdateConnection, 
  DeleteConnection, 
  DeleteBatchConnections,
  TestConnection,
  ExecuteCommand
} from '../../wailsjs/go/backend/App'
import type { Connection, ConnectionRequest } from '../types/connection'

export function useConnections() {
  const loading = ref(false)
  const connections = ref<Connection[]>([])
  const allConnections = ref<Connection[]>([])
  const selectedRowKeys = ref<string[]>([])
  const expandedRowKeys = ref<string[]>([])
  
  // 加载连接列表
  async function loadConnections(type: string = 'all') {
    try {
      loading.value = true
      allConnections.value = await GetConnections('')
      
      if (type === 'all') {
        connections.value = allConnections.value
      } else {
        connections.value = allConnections.value.filter(conn => conn.type === type)
      }
    } catch (error) {
      message.error('加载连接失败: ' + error)
    } finally {
      loading.value = false
    }
  }
  
  // 添加连接
  async function handleAddConnection(req: ConnectionRequest, onSuccess?: () => void) {
    if (!req.type || !req.ip || !req.port) {
      message.error('请填写必填字段')
      return
    }

    try {
      const newConn = await AddConnection(req)
      message.success('添加成功，正在连接...')
      
      if (onSuccess) onSuccess()
      
      await loadConnections()
      
      // 智能轮询：等待连接状态更新
      let pollCount = 0
      const maxPolls = 10
      const pollInterval = setInterval(async () => {
        pollCount++
        await loadConnections()
        
        const currentConn = connections.value.find(c => c.id === newConn.id)
        if (currentConn && currentConn.status !== 'pending') {
          clearInterval(pollInterval)
          
          // 如果是 VNC 或 RDP 且连接成功，自动执行截图
          if (currentConn.status === 'success' && (currentConn.type === 'VNC' || currentConn.type === 'RDP')) {
            setTimeout(async () => {
              try {
                await ExecuteCommand(currentConn.id, 'screenshot')
                await loadConnections()
              } catch (error) {
                console.error('自动截图失败:', error)
              }
            }, 500)
          }
          return
        }
        
        if (pollCount >= maxPolls) {
          clearInterval(pollInterval)
        }
      }, 1000)
    } catch (error) {
      message.error('操作失败: ' + error)
    }
  }
  
  // 更新连接
  async function handleUpdateConnection(id: string, req: ConnectionRequest, onSuccess?: () => void) {
    if (!req.type || !req.ip || !req.port) {
      message.error('请填写必填字段')
      return
    }

    try {
      await UpdateConnection(id, req)
      message.success('更新成功')
      
      if (onSuccess) onSuccess()
      await loadConnections()
    } catch (error) {
      message.error('操作失败: ' + error)
    }
  }
  
  // 单个连接
  async function handleConnect(id: string, onComplete?: (id: string) => void) {
    try {
      await ConnectSingle(id)
      message.success('连接任务已启动')
      
      await loadConnections()
      
      // 智能轮询
      let pollCount = 0
      const maxPolls = 10
      const pollInterval = setInterval(async () => {
        pollCount++
        await loadConnections()
        
        const currentConn = connections.value.find(c => c.id === id)
        if (currentConn && currentConn.status !== 'pending') {
          clearInterval(pollInterval)
          
          // 如果是 VNC 或 RDP 且连接成功，自动执行截图
          if (currentConn.status === 'success' && (currentConn.type === 'VNC' || currentConn.type === 'RDP')) {
            setTimeout(async () => {
              try {
                await ExecuteCommand(currentConn.id, 'screenshot')
                await loadConnections()
                if (onComplete) onComplete(id)
              } catch (error) {
                console.error('自动截图失败:', error)
              }
            }, 500)
          } else {
            if (onComplete) onComplete(id)
          }
          return
        }
        
        if (pollCount >= maxPolls) {
          clearInterval(pollInterval)
        }
      }, 1000)
    } catch (error) {
      message.error('连接失败: ' + error)
    }
  }
  
  // 批量连接
  async function handleBatchConnect(onComplete?: (ids: string[]) => void) {
    try {
      const count = await ConnectBatch(selectedRowKeys.value)
      message.success(`已启动 ${count} 个连接任务`)
      const connectedIds = [...selectedRowKeys.value]
      selectedRowKeys.value = []
      
      await loadConnections()
      
      // 智能轮询
      let pollCount = 0
      const maxPolls = 20
      const pollInterval = setInterval(async () => {
        pollCount++
        await loadConnections()
        
        const allCompleted = connectedIds.every(id => {
          const conn = connections.value.find(c => c.id === id)
          return conn && conn.status !== 'pending'
        })
        
        if (allCompleted) {
          clearInterval(pollInterval)
          message.success('所有连接已完成')
          
          // 对成功的 VNC/RDP 连接自动执行截图
          const vncRdpConns = connections.value.filter(c => 
            connectedIds.includes(c.id) && 
            c.status === 'success' && 
            (c.type === 'VNC' || c.type === 'RDP')
          )
          
          if (vncRdpConns.length > 0) {
            for (let i = 0; i < vncRdpConns.length; i++) {
              setTimeout(async () => {
                try {
                  await ExecuteCommand(vncRdpConns[i].id, 'screenshot')
                  if (i === vncRdpConns.length - 1) {
                    await loadConnections()
                    if (onComplete) onComplete(connectedIds)
                  }
                } catch (error) {
                  console.error('自动截图失败:', error)
                }
              }, i * 1000)
            }
          } else {
            if (onComplete) onComplete(connectedIds)
          }
          return
        }
        
        if (pollCount >= maxPolls) {
          clearInterval(pollInterval)
          message.warning('部分连接仍在进行中，请稍后刷新查看')
        }
      }, 1000)
    } catch (error) {
      message.error('批量连接失败: ' + error)
    }
  }
  
  // 删除连接
  async function handleDelete(id: string) {
    try {
      await DeleteConnection(id)
      message.success('删除成功')
      await loadConnections()
    } catch (error) {
      message.error('删除失败: ' + error)
    }
  }
  
  // 批量删除
  async function handleBatchDelete() {
    try {
      const count = await DeleteBatchConnections(selectedRowKeys.value)
      message.success(`已删除 ${count} 条记录`)
      selectedRowKeys.value = []
      await loadConnections()
    } catch (error) {
      message.error('批量删除失败: ' + error)
    }
  }
  
  // 测试连接
  async function handleTestConnection(req: ConnectionRequest) {
    if (!req.type || !req.ip || !req.port) {
      message.error('请填写必填字段')
      return false
    }

    try {
      const result = await TestConnection(req)
      message.success('连接测试成功: ' + result)
      return true
    } catch (error) {
      message.error('连接测试失败: ' + error)
      return false
    }
  }
  
  // 切换详情展开
  function toggleDetail(id: string) {
    const index = expandedRowKeys.value.indexOf(id)
    if (index > -1) {
      expandedRowKeys.value.splice(index, 1)
    } else {
      expandedRowKeys.value.push(id)
    }
  }
  
  // 选择变更
  function onSelectChange(keys: string[]) {
    selectedRowKeys.value = keys
  }
  
  // 获取类型数量
  function getTypeCount(type: string): number {
    return allConnections.value.filter(conn => conn.type === type).length
  }
  
  return {
    loading,
    connections,
    allConnections,
    selectedRowKeys,
    expandedRowKeys,
    loadConnections,
    handleAddConnection,
    handleUpdateConnection,
    handleConnect,
    handleBatchConnect,
    handleDelete,
    handleBatchDelete,
    handleTestConnection,
    toggleDetail,
    onSelectChange,
    getTypeCount,
  }
}
