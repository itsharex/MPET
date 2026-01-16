import { ref } from 'vue'

export function useConnectionForm(serviceTypes: any) {
  const addModalVisible = ref(false)
  const editingConnection = ref<any>(null)
  const testingConnection = ref(false)
  const form = ref({
    type: '',
    ip: '',
    port: '',
    user: '',
    pass: '',
  })

  // 显示添加对话框
  function showAddModal() {
    editingConnection.value = null
    resetForm()
    addModalVisible.value = true
  }

  // 显示编辑对话框
  function showEditModal(record: any) {
    editingConnection.value = record
    form.value = {
      type: record.type,
      ip: record.ip,
      port: record.port,
      user: record.user,
      pass: '',
    }
    addModalVisible.value = true
  }

  // 重置表单
  function resetForm() {
    form.value = {
      type: '',
      ip: '',
      port: '',
      user: '',
      pass: '',
    }
  }

  // 处理类型变更
  function handleTypeChange(value: string) {
    const type = serviceTypes.value.find((t: any) => t.value === value)
    if (type && type.port) {
      form.value.port = type.port
    }
  }

  return {
    addModalVisible,
    editingConnection,
    testingConnection,
    form,
    showAddModal,
    showEditModal,
    resetForm,
    handleTypeChange,
  }
}
