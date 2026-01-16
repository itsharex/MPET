import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface Activity {
  time: string
  user: string
  action: string
  status: string
  statusType: 'success' | 'warning' | 'error' | 'info'
}

export const useDashboardStore = defineStore('dashboard', () => {
  // 状态数据
  const userCount = ref(1284)
  const growthRate = ref(23.5)
  const revenue = ref(456789)
  const projectCount = ref(42)
  const recentActivities = ref<Activity[]>([
    {
      time: '2024-01-15 14:30',
      user: '张三',
      action: '创建了新的项目 "企业管理系统"',
      status: '成功',
      statusType: 'success'
    },
    {
      time: '2024-01-15 13:45',
      user: '李四',
      action: '更新了用户权限设置',
      status: '成功',
      statusType: 'success'
    },
    {
      time: '2024-01-15 12:20',
      user: '王五',
      action: '导出了月度报表',
      status: '处理中',
      statusType: 'warning'
    },
    {
      time: '2024-01-15 11:00',
      user: '赵六',
      action: '删除了过期数据',
      status: '成功',
      statusType: 'success'
    }
  ])

  // 计算属性
  const formattedRevenue = computed(() => {
    return `¥${revenue.value.toLocaleString()}`
  })

  const totalActivities = computed(() => recentActivities.value.length)

  // 操作方法
  function updateUserCount(count: number) {
    userCount.value = count
  }

  function updateGrowthRate(rate: number) {
    growthRate.value = rate
  }

  function updateRevenue(amount: number) {
    revenue.value = amount
  }

  function updateProjectCount(count: number) {
    projectCount.value = count
  }

  function addActivity(activity: Activity) {
    recentActivities.value.unshift(activity)
  }

  function removeActivity(index: number) {
    recentActivities.value.splice(index, 1)
  }

  function clearActivities() {
    recentActivities.value = []
  }

  // 模拟数据更新
  function simulateDataUpdate() {
    // 模拟用户数量增长
    updateUserCount(userCount.value + Math.floor(Math.random() * 10))
    
    // 模拟增长率变化
    updateGrowthRate(Math.round((growthRate.value + (Math.random() - 0.5) * 2) * 10) / 10)
    
    // 模拟收入变化
    updateRevenue(revenue.value + Math.floor(Math.random() * 10000))
    
    // 模拟项目数量变化
    updateProjectCount(projectCount.value + Math.floor(Math.random() * 3))
  }

  return {
    // 状态
    userCount,
    growthRate,
    revenue,
    projectCount,
    recentActivities,
    
    // 计算属性
    formattedRevenue,
    totalActivities,
    
    // 方法
    updateUserCount,
    updateGrowthRate,
    updateRevenue,
    updateProjectCount,
    addActivity,
    removeActivity,
    clearActivities,
    simulateDataUpdate
  }
})