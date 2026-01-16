<template>
  <div class="title-bar" :class="{ 'dark': isDark }" style="--wails-draggable: drag">
    <div class="title-bar-drag-region">
      <div class="title-bar-title">
        <img src="/icon.svg" alt="MPET" class="title-bar-icon" />
        {{ title }}
      </div>
    </div>
    
    <div class="title-bar-controls">
      <!-- 主题切换按钮 -->
      <div class="title-bar-button theme-button" @click="toggleTheme" title="切换主题">
        <span class="theme-icon">{{ isDark ? '☀️' : '🌙' }}</span>
      </div>
      
      <!-- 最小化按钮 -->
      <div class="title-bar-button minimize-button" @click="minimizeWindow" title="最小化">
        <MinusOutlined />
      </div>
      
      <!-- 最大化/还原按钮 -->
      <div class="title-bar-button maximize-button" @click="toggleMaximize" :title="isMaximized ? '还原' : '最大化'">
        <component :is="isMaximized ? FullscreenExitOutlined : FullscreenOutlined" />
      </div>
      
      <!-- 关闭按钮 -->
      <div class="title-bar-button close-button" @click="closeWindow" title="关闭">
        <CloseOutlined />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { 
  MinusOutlined, 
  FullscreenOutlined, 
  FullscreenExitOutlined, 
  CloseOutlined
} from '@ant-design/icons-vue'
import { useThemeStore } from '../stores/theme'
import { WindowMinimize, WindowMaximize, WindowUnmaximize, WindowClose, WindowIsMaximized } from '../../wailsjs/go/main/App'

interface Props {
  title?: string
}

const props = withDefaults(defineProps<Props>(), {
  title: 'MPET - Multi-Protocol Exploitation Toolkit'
})

const themeStore = useThemeStore()
const isMaximized = ref(false)

const isDark = computed(() => themeStore.isDark)

const toggleTheme = () => {
  themeStore.toggleTheme()
  themeStore.saveTheme()
}

const minimizeWindow = async () => {
  try {
    await WindowMinimize()
  } catch (error) {
    console.error('Failed to minimize window:', error)
  }
}

const toggleMaximize = async () => {
  try {
    if (isMaximized.value) {
      await WindowUnmaximize()
    } else {
      await WindowMaximize()
    }
    isMaximized.value = !isMaximized.value
  } catch (error) {
    console.error('Failed to toggle maximize:', error)
  }
}

const closeWindow = async () => {
  try {
    await WindowClose()
  } catch (error) {
    console.error('Failed to close window:', error)
  }
}

const checkWindowState = async () => {
  try {
    isMaximized.value = await WindowIsMaximized()
  } catch (error) {
    console.error('Failed to check window state:', error)
  }
}

onMounted(() => {
  checkWindowState()
  window.addEventListener('resize', checkWindowState)
})
</script>

<style scoped>
.title-bar {
  display: flex;
  align-items: center;
  height: 32px;
  background: rgba(255, 255, 255, 0.95);
  backdrop-filter: blur(20px);
  -webkit-backdrop-filter: blur(20px);
  border-bottom: 1px solid #f0f0f0;
  user-select: none;
  position: relative;
  z-index: 1000;
  transition: all 0.3s ease;
  -webkit-app-region: drag;
  app-region: drag;
}

.title-bar.dark {
  background: rgba(20, 20, 20, 0.95);
  border-bottom-color: #303030;
  color: #ffffff;
}

.title-bar-drag-region {
  flex: 1;
  height: 100%;
  display: flex;
  align-items: center;
  padding-left: 12px;
  -webkit-app-region: drag;
  app-region: drag;
}

.title-bar-title {
  font-size: 13px;
  font-weight: 600;
  color: #333;
  transition: color 0.3s ease;
  display: flex;
  align-items: center;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.title-bar-icon {
  width: 18px;
  height: 18px;
  margin-right: 8px;
}

.title-bar.dark .title-bar-title {
  color: #ffffff;
}

.title-bar-controls {
  display: flex;
  height: 100%;
  -webkit-app-region: no-drag;
  app-region: no-drag;
}

.title-bar-button {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 46px;
  height: 100%;
  cursor: pointer;
  transition: background-color 0.2s ease;
  font-size: 12px;
}

.title-bar-button:hover {
  background-color: #f5f5f5;
}

.title-bar.dark .title-bar-button:hover {
  background-color: #333;
}

.theme-button:hover {
  background-color: #e6f7ff !important;
}

.title-bar.dark .theme-button:hover {
  background-color: rgba(24, 144, 255, 0.2) !important;
}

.minimize-button:hover {
  background-color: #fff7e6 !important;
}

.title-bar.dark .minimize-button:hover {
  background-color: rgba(250, 140, 22, 0.2) !important;
}

.maximize-button:hover {
  background-color: #f6ffed !important;
}

.title-bar.dark .maximize-button:hover {
  background-color: rgba(82, 196, 26, 0.2) !important;
}

.close-button:hover {
  background-color: #ff4d4f !important;
  color: white !important;
}

.theme-icon {
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.title-bar-button svg {
  width: 14px;
  height: 14px;
}

@media (max-width: 768px) {
  .title-bar-title {
    font-size: 12px;
  }
  
  .title-bar-button {
    width: 40px;
  }
  
  .title-bar-button svg {
    width: 12px;
    height: 12px;
  }
}
</style>
