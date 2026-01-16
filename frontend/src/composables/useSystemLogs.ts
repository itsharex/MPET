import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { GetSystemLogs, ClearSystemLogs } from '../../wailsjs/go/backend/App'

export function useSystemLogs() {
  const logModalVisible = ref(false)
  const logLoading = ref(false)
  const systemLogs = ref<string[]>([])

  // 显示日志对话框
  function showLogModal() {
    logModalVisible.value = true
    loadLogs()
  }

  // 加载日志
  async function loadLogs() {
    try {
      logLoading.value = true
      const logs = await GetSystemLogs()
      systemLogs.value = logs
    } catch (error) {
      message.error('加载日志失败: ' + error)
    } finally {
      logLoading.value = false
    }
  }

  // 清空日志显示
  async function clearLogDisplay() {
    try {
      await ClearSystemLogs()
      systemLogs.value = []
      message.success('日志已清空')
    } catch (error) {
      message.error('清空日志失败: ' + error)
    }
  }

  return {
    logModalVisible,
    logLoading,
    systemLogs,
    showLogModal,
    loadLogs,
    clearLogDisplay,
  }
}
