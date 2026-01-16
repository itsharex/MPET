import { ref, computed } from 'vue'
import type { Connection } from '../types/connection'

export function useFilters(connections: any) {
  const filters = ref({
    types: [] as string[],
    ip: '',
    user: '',
    status: undefined,
    message: '',
  })

  const pagination = ref({
    current: 1,
    pageSize: 30,
  })

  // 筛选后的连接
  const filteredConnections = computed(() => {
    return connections.value.filter((conn: Connection) => {
      if (filters.value.types.length > 0 && !filters.value.types.includes(conn.type)) return false
      if (filters.value.ip && !conn.ip?.toLowerCase().includes(filters.value.ip.toLowerCase())) return false
      if (filters.value.user && !conn.user?.toLowerCase().includes(filters.value.user.toLowerCase())) return false
      if (filters.value.status && conn.status !== filters.value.status) return false
      if (filters.value.message && !conn.message?.toLowerCase().includes(filters.value.message.toLowerCase())) return false
      return true
    })
  })

  // 分页后的数据
  const paginatedConnections = computed(() => {
    const start = (pagination.value.current - 1) * pagination.value.pageSize
    const end = start + pagination.value.pageSize
    return filteredConnections.value.slice(start, end)
  })

  // 重置筛选
  function resetFilters() {
    filters.value = {
      types: [],
      ip: '',
      user: '',
      status: undefined,
      message: '',
    }
  }

  return {
    filters,
    pagination,
    filteredConnections,
    paginatedConnections,
    resetFilters,
  }
}
