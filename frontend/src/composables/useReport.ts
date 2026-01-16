import { ref } from 'vue'
import { message } from 'ant-design-vue'
import { ExportReport } from '../../wailsjs/go/backend/App'
import type { services } from '../../wailsjs/go/models'
import html2canvas from 'html2canvas'

export function useReport() {
  const exporting = ref(false)

  // 截取元素截图
  const captureElement = async (element: HTMLElement): Promise<string> => {
    try {
      // 滚动到元素可见区域
      element.scrollIntoView({ behavior: 'smooth', block: 'center' })
      // 等待滚动完成
      await new Promise(resolve => setTimeout(resolve, 300))
      
      const canvas = await html2canvas(element, {
        backgroundColor: '#ffffff',
        scale: 2, // 提高清晰度
        logging: false,
        useCORS: true,
        allowTaint: true,
        foreignObjectRendering: false,
      })
      return canvas.toDataURL('image/png')
    } catch (error) {
      console.error('截图失败:', error)
      return ''
    }
  }

  // 导出报告
  const exportReport = async (
    connectionIds: string[],
    vulnerabilities?: services.VulnerabilityData[]
  ) => {
    if (connectionIds.length === 0) {
      message.warning('请选择要导出的连接')
      return
    }

    exporting.value = true
    try {
      const req: services.ExportReportRequest = {
        connectionIds,
        vulnerabilities: vulnerabilities || [],
      }

      const filePath = await ExportReport(req)
      message.success(`报告导出成功: ${filePath}`)
      return filePath
    } catch (error: any) {
      message.error(`导出失败: ${error.message || error}`)
      throw error
    } finally {
      exporting.value = false
    }
  }

  // 从连接卡片导出报告（带截图）
  const exportReportWithScreenshots = async (
    connectionIds: string[],
    getCardElement: (id: string) => HTMLElement | null
  ) => {
    if (connectionIds.length === 0) {
      message.warning('请选择要导出的连接')
      return
    }

    exporting.value = true
    let loadingMessage: any = null

    try {
      loadingMessage = message.loading('正在生成截图...', 0)
      const vulnerabilities: services.VulnerabilityData[] = []

      // 为每个连接生成截图和数据
      for (let i = 0; i < connectionIds.length; i++) {
        const id = connectionIds[i]
        loadingMessage()
        loadingMessage = message.loading(`正在截图 ${i + 1}/${connectionIds.length}...`, 0)
        
        const cardElement = getCardElement(id)
        let screenshot = ''
        
        if (cardElement) {
          console.log(`开始截图连接 ${id}`)
          screenshot = await captureElement(cardElement)
          if (screenshot) {
            console.log(`截图成功: ${id}, 大小: ${screenshot.length} 字符`)
          } else {
            console.warn(`截图失败: ${id}`)
          }
        } else {
          console.warn(`未找到卡片元素: ${id}`)
        }

        // 这里需要从连接数据中提取信息
        // 实际实现中应该从 connections 数组中获取对应的连接信息
        vulnerabilities.push({
          name: '', // 将由后端根据连接类型生成
          level: '', // 将由后端根据连接类型生成
          target: '', // 将由后端根据连接信息生成
          describe: '', // 将由后端根据连接类型生成
          images: screenshot ? [screenshot] : [],
          repair: '', // 将由后端根据连接类型生成
        })
      }

      // 关闭截图 loading
      if (loadingMessage) {
        loadingMessage()
      }
      
      // 显示生成报告 loading
      loadingMessage = message.loading('正在生成报告文档...', 0)

      const req: services.ExportReportRequest = {
        connectionIds,
        vulnerabilities,
      }

      console.log('调用后端 ExportReport API:', req)
      const filePath = await ExportReport(req)
      console.log('报告生成成功:', filePath)
      
      // 关闭 loading
      if (loadingMessage) {
        loadingMessage()
      }
      
      message.success(`报告导出成功: ${filePath}`)
      
      // 打开文件所在目录
      if (window.runtime) {
        // 可以调用系统命令打开文件夹
        console.log('报告已保存到:', filePath)
      }
      
      return filePath
    } catch (error: any) {
      console.error('导出报告失败:', error)
      
      // 关闭 loading
      if (loadingMessage) {
        loadingMessage()
      }
      
      message.error(`导出失败: ${error.message || error}`)
      throw error
    } finally {
      exporting.value = false
    }
  }

  return {
    exporting,
    exportReport,
    exportReportWithScreenshots,
    captureElement,
  }
}
