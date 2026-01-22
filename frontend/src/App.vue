<template>
  <a-config-provider :locale="zhCN" :theme="themeConfig">
    <Home />
  </a-config-provider>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ConfigProvider as AConfigProvider, theme } from 'ant-design-vue'
import zhCN from 'ant-design-vue/es/locale/zh_CN'
import Home from './views/Home.vue'
import { useThemeStore } from './stores/theme'
import 'ant-design-vue/dist/reset.css'

const themeStore = useThemeStore()

const themeConfig = computed(() => ({
  algorithm: themeStore.isDark ? theme.darkAlgorithm : theme.defaultAlgorithm,
  token: {
    colorPrimary: '#1890ff',
    borderRadius: 4,
  },
}))
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

html, body {
  overflow: hidden;
  height: 100vh;
  width: 100vw;
}

:root {
  --glass-bg: rgba(255, 255, 255, 0.95);
  --glass-border: rgba(0, 0, 0, 0.1);
  --shadow: rgba(0, 0, 0, 0.05);
}

:root.dark {
  --glass-bg: rgba(20, 20, 20, 0.95);
  --glass-border: rgba(255, 255, 255, 0.1);
  --shadow: rgba(0, 0, 0, 0.3);
}

body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
}

#app {
  height: 100vh;
  overflow: hidden;
}

/* 暗色主题基础样式 */
html.dark {
  color-scheme: dark;
}

html.dark body {
  background-color: #141414;
  color: rgba(255, 255, 255, 0.85);
}

/* 暗色主题 - 布局 */
html.dark .ant-layout {
  background: #141414;
}

html.dark .ant-layout-sider {
  background: #1f1f1f !important;
}

html.dark .ant-layout-header {
  background: #1f1f1f !important;
  border-bottom: 1px solid #303030;
}

html.dark .ant-layout-content {
  background: #141414 !important;
}

/* 暗色主题 - 菜单 */
html.dark .ant-menu {
  background: #1f1f1f !important;
}

html.dark .ant-menu-item-selected {
  background-color: rgba(24, 144, 255, 0.15) !important;
}

html.dark .logo {
  border-bottom: 1px solid #303030;
}

/* 暗色主题 - 表格细节优化 */
html.dark .ant-table-wrapper {
  background: transparent;
}

html.dark .ant-table-container {
  border-color: #303030;
}

html.dark .ant-table-expanded-row > td {
  background: #141414 !important;
}

/* 暗色主题 - 展开行内部卡片 */
html.dark .ant-table-expanded-row .ant-row > div > div {
  background: #1f1f1f !important;
  border-color: #303030 !important;
}

/* 暗色主题 - 展开行内的标题和文本 */
html.dark .ant-table-expanded-row div[style*="color: #8c8c8c"] {
  color: rgba(255, 255, 255, 0.45) !important;
}

html.dark .ant-table-expanded-row div[style*="color: rgba(0, 0, 0, 0.85)"] {
  color: rgba(255, 255, 255, 0.85) !important;
}

html.dark .ant-table-expanded-row div[style*="color: #262626"] {
  color: rgba(255, 255, 255, 0.85) !important;
}

html.dark .ant-table-expanded-row div[style*="color: #999"] {
  color: rgba(255, 255, 255, 0.45) !important;
}

/* 暗色主题 - 展开行内的背景色 */
html.dark .ant-table-expanded-row div[style*="background: #fafafa"] {
  background: #141414 !important;
}

html.dark .ant-table-expanded-row div[style*="background: #fff"] {
  background: #1f1f1f !important;
}

html.dark .ant-table-expanded-row div[style*="background: #f5f5f5"] {
  background: #0a0a0a !important;
}

/* 暗色主题 - 展开行内的边框 */
html.dark .ant-table-expanded-row div[style*="border: 1px solid #e8e8e8"] {
  border-color: #303030 !important;
}

html.dark .ant-table-expanded-row div[style*="border-bottom: 2px solid"] {
  border-bottom-color: #1890ff !important;
}

/* 暗色主题 - 输入框占位符 */
html.dark .ant-input::placeholder,
html.dark .ant-input-password input::placeholder,
html.dark .ant-select-selection-placeholder {
  color: rgba(255, 255, 255, 0.3);
}

/* 暗色主题 - 下拉菜单 */
html.dark .ant-select-dropdown {
  background: #1f1f1f;
}

html.dark .ant-select-item {
  color: rgba(255, 255, 255, 0.85);
}

html.dark .ant-select-item-option-selected {
  background: rgba(24, 144, 255, 0.15);
}

html.dark .ant-select-item-option-active {
  background: #262626;
}

/* 暗色主题 - 模态框 */
html.dark .ant-modal-footer {
  border-top: 1px solid #303030;
}

html.dark .ant-modal-close-x {
  color: rgba(255, 255, 255, 0.45);
}

/* 暗色主题 - Popconfirm */
html.dark .ant-popover-inner {
  background: #1f1f1f;
}

html.dark .ant-popover-arrow-content {
  background: #1f1f1f;
}

/* 暗色主题 - 分页器 */
html.dark .ant-pagination-item {
  background: #1f1f1f;
  border-color: #434343;
}

html.dark .ant-pagination-item a {
  color: rgba(255, 255, 255, 0.85);
}

html.dark .ant-pagination-item-active {
  background: rgba(24, 144, 255, 0.15);
  border-color: #1890ff;
}

html.dark .ant-pagination-prev .ant-pagination-item-link,
html.dark .ant-pagination-next .ant-pagination-item-link {
  background: #1f1f1f;
  border-color: #434343;
  color: rgba(255, 255, 255, 0.85);
}

html.dark .ant-pagination-disabled .ant-pagination-item-link {
  color: rgba(255, 255, 255, 0.25);
}

/* 暗色主题 - 表单标签 */
html.dark .ant-form-item-label > label {
  color: rgba(255, 255, 255, 0.85);
}

/* 暗色主题 - 标签 */
html.dark .ant-tag {
  border-color: transparent;
}

/* 暗色主题 - 复选框 */
html.dark .ant-checkbox-wrapper {
  color: rgba(255, 255, 255, 0.85);
}

/* 暗色主题 - 滚动条 */
html.dark ::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

html.dark ::-webkit-scrollbar-track {
  background: #1f1f1f;
}

html.dark ::-webkit-scrollbar-thumb {
  background: #434343;
  border-radius: 4px;
}

html.dark ::-webkit-scrollbar-thumb:hover {
  background: #595959;
}
</style>
