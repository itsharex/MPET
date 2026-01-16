import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { ImportCSV } from '../../wailsjs/go/backend/App'
import { parseCSVContent, parseFscanResult, parseFscan21Result, parseLightxResult } from '../utils/parsers'

export function useImport(onImportComplete?: () => void) {
  const isDragging = ref(false)
  let dragCounter = 0

  // 导入 CSV 文件
  async function handleImportCSV() {
    try {
      const count = await ImportCSV()
      message.success(`成功导入 ${count} 条记录`)
      if (onImportComplete) onImportComplete()
    } catch (error) {
      message.error('导入失败: ' + error)
    }
  }

  // 剪贴板导入
  async function handleClipboardImport() {
    try {
      const text = await navigator.clipboard.readText()
      
      if (!text || !text.trim()) {
        message.warning('剪贴板内容为空')
        return
      }
      
      message.loading({ content: '正在导入...', key: 'clipboard-import', duration: 0 })
      
      let successCount = 0
      let failCount = 0
      
      // 自动识别格式并解析
      if (text.includes('[Plugin:') && text.includes(':SUCCESS]')) {
        const result = await parseLightxResult(text)
        successCount = result.success
        failCount = result.failed
      } else if (text.includes('# ===== 漏洞信息 =====')) {
        const result = await parseFscan21Result(text)
        successCount = result.success
        failCount = result.failed
      } else if (text.includes('[+]') && /\[[\+\-\*]\]\s+\w+:/m.test(text)) {
        const result = await parseFscanResult(text)
        successCount = result.success
        failCount = result.failed
      } else if (text.includes(',')) {
        const result = await parseCSVContent(text)
        successCount = result.success
        failCount = result.failed
      } else {
        message.error({ 
          content: '无法识别剪贴板内容格式，支持 CSV、Fscan 和 Lightx 格式', 
          key: 'clipboard-import' 
        })
        return
      }
      
      message.success({ 
        content: `成功导入 ${successCount} 条记录${failCount > 0 ? `，失败 ${failCount} 条` : ''}`, 
        key: 'clipboard-import' 
      })
      
      if (onImportComplete) onImportComplete()
      
    } catch (error) {
      if (error instanceof Error && error.name === 'NotAllowedError') {
        message.error({ 
          content: '无法访问剪贴板，请授予权限', 
          key: 'clipboard-import' 
        })
      } else {
        message.error({ 
          content: '剪贴板导入失败: ' + error, 
          key: 'clipboard-import' 
        })
      }
    }
  }

  // 拖拽相关处理函数
  function handleDragOver(event: DragEvent) {
    event.preventDefault()
    event.stopPropagation()
  }

  function handleDragEnter(event: DragEvent) {
    event.preventDefault()
    event.stopPropagation()
    dragCounter++
    if (dragCounter === 1) {
      isDragging.value = true
    }
  }

  function handleDragLeave(event: DragEvent) {
    event.preventDefault()
    event.stopPropagation()
    dragCounter--
    if (dragCounter === 0) {
      isDragging.value = false
    }
  }

  async function handleFileDrop(event: DragEvent) {
    event.preventDefault()
    event.stopPropagation()
    
    // 重置拖拽状态
    isDragging.value = false
    dragCounter = 0
    
    const files = event.dataTransfer?.files
    if (!files || files.length === 0) {
      return
    }
    
    const file = files[0]
    
    // 检查文件类型
    const fileName = file.name.toLowerCase()
    if (!fileName.endsWith('.csv') && !fileName.endsWith('.txt')) {
      message.error('仅支持 CSV 和 TXT (fscan结果) 格式文件')
      return
    }
    
    try {
      const text = await file.text()
      
      message.loading({ content: '正在导入...', key: 'import', duration: 0 })
      
      let successCount = 0
      let failCount = 0
      
      // 判断文件类型并解析
      if (fileName.endsWith('.txt')) {
        if (text.includes('[Plugin:') && text.includes(':SUCCESS]')) {
          const result = await parseLightxResult(text)
          successCount = result.success
          failCount = result.failed
        } else if (text.includes('# ===== 漏洞信息 =====')) {
          const result = await parseFscan21Result(text)
          successCount = result.success
          failCount = result.failed
        } else {
          const result = await parseFscanResult(text)
          successCount = result.success
          failCount = result.failed
        }
      } else {
        const result = await parseCSVContent(text)
        successCount = result.success
        failCount = result.failed
      }
      
      message.success({ 
        content: `成功导入 ${successCount} 条记录${failCount > 0 ? `，失败 ${failCount} 条` : ''}`, 
        key: 'import' 
      })
      
      if (onImportComplete) onImportComplete()
      
    } catch (error) {
      message.error({ content: '文件读取失败: ' + error, key: 'import' })
    }
  }

  return {
    isDragging,
    handleImportCSV,
    handleClipboardImport,
    handleDragOver,
    handleDragEnter,
    handleDragLeave,
    handleFileDrop,
  }
}
