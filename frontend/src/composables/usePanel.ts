import { ref } from 'vue'

export function usePanel() {
  const panelWidths = ref<Record<string, string>>({})
  const panelHeights = ref<Record<string, string>>({})
  const resizing = ref<{ 
    id: string
    type: 'vertical' | 'horizontal'
    startX: number
    startY: number
    startWidth?: number
    startHeight?: number 
  } | null>(null)

  // 开始垂直调整大小（左右拖动）
  function startVerticalResize(event: MouseEvent, id: string) {
    event.preventDefault()
    const container = (event.target as HTMLElement).parentElement
    if (!container) return
    
    const leftPanel = container.querySelector('.resizable-panel') as HTMLElement
    if (!leftPanel) return
    
    resizing.value = {
      id,
      type: 'vertical',
      startX: event.clientX,
      startY: 0,
      startWidth: leftPanel.offsetWidth,
    }
    
    document.addEventListener('mousemove', handleVerticalResize)
    document.addEventListener('mouseup', stopResize)
    document.body.style.cursor = 'col-resize'
    document.body.style.userSelect = 'none'
  }

  // 开始水平调整大小（上下拖动）
  function startHorizontalResize(event: MouseEvent, id: string) {
    event.preventDefault()
    const container = (event.target as HTMLElement).parentElement
    if (!container) return
    
    resizing.value = {
      id,
      type: 'horizontal',
      startX: 0,
      startY: event.clientY,
      startHeight: container.offsetHeight,
    }
    
    document.addEventListener('mousemove', handleHorizontalResize)
    document.addEventListener('mouseup', stopResize)
    document.body.style.cursor = 'row-resize'
    document.body.style.userSelect = 'none'
  }

  // 处理垂直调整
  function handleVerticalResize(event: MouseEvent) {
    if (!resizing.value || resizing.value.type !== 'vertical') return
    
    const deltaX = event.clientX - resizing.value.startX
    const newWidth = (resizing.value.startWidth || 0) + deltaX
    const minWidth = 200
    const maxWidth = window.innerWidth * 0.6
    
    if (newWidth >= minWidth && newWidth <= maxWidth) {
      const percentage = (newWidth / window.innerWidth) * 100
      panelWidths.value[resizing.value.id] = `${percentage}%`
    }
  }

  // 处理水平调整
  function handleHorizontalResize(event: MouseEvent) {
    if (!resizing.value || resizing.value.type !== 'horizontal') return
    
    const deltaY = event.clientY - resizing.value.startY
    const newHeight = (resizing.value.startHeight || 0) + deltaY
    const minHeight = 200
    const maxHeight = 800
    
    if (newHeight >= minHeight && newHeight <= maxHeight) {
      panelHeights.value[resizing.value.id] = `${newHeight}px`
    }
  }

  // 停止调整大小
  function stopResize() {
    document.removeEventListener('mousemove', handleVerticalResize)
    document.removeEventListener('mousemove', handleHorizontalResize)
    document.removeEventListener('mouseup', stopResize)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    resizing.value = null
  }

  return {
    panelWidths,
    panelHeights,
    resizing,
    startVerticalResize,
    startHorizontalResize,
    handleVerticalResize,
    handleHorizontalResize,
    stopResize,
  }
}
