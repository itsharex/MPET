import { defineStore } from 'pinia'
import { ref, watch } from 'vue'

export const useThemeStore = defineStore('theme', () => {
  const isDark = ref(false)

  const toggleTheme = () => {
    isDark.value = !isDark.value
    updateTheme()
    saveTheme()
  }

  const setTheme = (dark: boolean) => {
    isDark.value = dark
    updateTheme()
    saveTheme()
  }

  const updateTheme = () => {
    if (isDark.value) {
      document.documentElement.classList.add('dark')
      document.documentElement.setAttribute('data-theme', 'dark')
    } else {
      document.documentElement.classList.remove('dark')
      document.documentElement.setAttribute('data-theme', 'light')
    }
  }

  // 初始化主题
  const initTheme = () => {
    const savedTheme = localStorage.getItem('theme')
    if (savedTheme === 'dark') {
      isDark.value = true
    } else {
      isDark.value = false
    }
    updateTheme()
  }

  // 保存主题到 localStorage
  const saveTheme = () => {
    localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
  }

  // 监听主题变化
  watch(isDark, () => {
    updateTheme()
  })

  return {
    isDark,
    toggleTheme,
    setTheme,
    initTheme,
    saveTheme,
  }
})
