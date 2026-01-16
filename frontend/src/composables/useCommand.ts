import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { ExecuteCommand } from '../../wailsjs/go/backend/App'

export function useCommand(
  onCommandComplete?: (id: string) => void
) {
  const commandInputs = ref<Record<string, string>>({})
  const commandLoading = ref<Record<string, boolean>>({})
  const selectedCommand = ref<Record<string, string>>({})

  // 选择常用命令
  function onCommandSelect(record: any, command: string) {
    // 将选中的命令填充到输入框
    commandInputs.value[record.id] = command
    // 不自动执行，等待用户点击执行按钮
  }

  // 执行命令
  async function handleExecuteCommand(record: any, onReload?: () => Promise<void>) {
    const command = commandInputs.value[record.id]
    if (!command || !command.trim()) {
      message.warning('请输入命令')
      return
    }
    
    try {
      commandLoading.value[record.id] = true
      
      // 调用后端执行命令（后端会自动追加结果和日志）
      await ExecuteCommand(record.id, command.trim())
      
      message.success('命令执行成功')
      
      // 清空输入框和选择
      commandInputs.value[record.id] = ''
      selectedCommand.value[record.id] = undefined
      
      // 重新加载连接数据以获取最新的日志和结果
      if (onReload) {
        await onReload()
      }
      
      // 通知父组件滚动到底部
      if (onCommandComplete) {
        onCommandComplete(record.id)
      }
      
    } catch (error) {
      message.error('命令执行失败: ' + error)
    } finally {
      commandLoading.value[record.id] = false
    }
  }

  // VNC/RDP 截图
  async function handleVNCScreenshot(record: any, onReload?: () => Promise<void>) {
    try {
      commandLoading.value[record.id] = true
      
      // 调用后端执行截图命令
      await ExecuteCommand(record.id, 'screenshot')
      
      message.success('截图获取成功')
      
      // 重新加载连接数据以获取最新的截图
      if (onReload) {
        await onReload()
      }
      
      // 通知父组件滚动到底部
      if (onCommandComplete) {
        onCommandComplete(record.id)
      }
      
    } catch (error) {
      message.error('截图获取失败: ' + error)
    } finally {
      commandLoading.value[record.id] = false
    }
  }

  return {
    commandInputs,
    commandLoading,
    selectedCommand,
    onCommandSelect,
    handleExecuteCommand,
    handleVNCScreenshot,
  }
}
