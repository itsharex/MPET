import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { GetProxySettings, UpdateProxySettings } from '../../wailsjs/go/backend/App'

export function useProxy() {
  const proxyModalVisible = ref(false)
  const proxyForm = ref({
    enabled: false,
    type: 'socks5',
    host: '127.0.0.1',
    port: '1080',
    user: '',
    pass: '',
  })

  // 加载代理设置
  async function loadProxySettings() {
    try {
      const settings = await GetProxySettings()
      proxyForm.value = { ...settings }
    } catch (error) {
      console.error('加载代理设置失败:', error)
    }
  }

  // 显示代理设置对话框
  function showProxyModal() {
    proxyModalVisible.value = true
  }

  // 保存代理设置
  async function handleSaveProxy() {
    try {
      await UpdateProxySettings(proxyForm.value)
      message.success('代理设置已保存')
      proxyModalVisible.value = false
    } catch (error) {
      message.error('保存失败: ' + error)
    }
  }

  return {
    proxyModalVisible,
    proxyForm,
    loadProxySettings,
    showProxyModal,
    handleSaveProxy,
  }
}
