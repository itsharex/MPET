<template>
  <div class="file-browser">
    <!-- 路径导航 -->
    <div class="path-nav">
      <a-breadcrumb separator="/">
        <a-breadcrumb-item>
          <a @click="navigateTo('/', '')">
            <HomeOutlined /> {{ isSMB ? '共享' : '根目录' }}
          </a>
        </a-breadcrumb-item>
        <a-breadcrumb-item v-if="isSMB && currentShare">
          <a @click="navigateTo('/', currentShare)">{{ currentShare }}</a>
        </a-breadcrumb-item>
        <a-breadcrumb-item v-for="(part, index) in pathParts" :key="index">
          <a @click="navigateTo(getPathUpTo(index), currentShare)">{{ part }}</a>
        </a-breadcrumb-item>
      </a-breadcrumb>
      <a-button type="text" size="small" @click="refreshDirectory" :loading="loading">
        <ReloadOutlined />
      </a-button>
    </div>

    <!-- 文件表格 -->
    <div class="file-table">
      <a-spin :spinning="loading">
        <div class="table-header">
          <div class="col-name">名称</div>
          <div class="col-size">大小</div>
          <div class="col-time">修改时间</div>
          <div class="col-action">操作</div>
        </div>
        <div class="table-body">
          <div v-if="files.length === 0" class="empty-state">
            <a-empty description="暂无文件" />
          </div>
          <div
            v-for="file in files"
            :key="file.name"
            class="table-row"
            :class="{ 'is-folder': file.type === 'folder' || file.type === 'share' }"
            @click="handleFileClick(file)"
          >
            <div class="col-name">
              <FolderOutlined v-if="file.type === 'folder' || file.type === 'share'" class="icon folder-icon" />
              <FileOutlined v-else class="icon file-icon" />
              <span class="name-text">{{ file.name }}</span>
            </div>
            <div class="col-size">
              {{ (file.type === 'folder' || file.type === 'share') ? '-' : formatSize(file.size) }}
            </div>
            <div class="col-time">
              {{ file.modTime || '-' }}
            </div>
            <div class="col-action">
              <a-button
                v-if="file.type === 'file'"
                type="link"
                size="small"
                @click.stop="downloadFile(file)"
              >
                <DownloadOutlined /> 下载
              </a-button>
            </div>
          </div>
        </div>
      </a-spin>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { message } from 'ant-design-vue'
import {
  FolderOutlined,
  FileOutlined,
  HomeOutlined,
  ReloadOutlined,
  DownloadOutlined,
} from '@ant-design/icons-vue'
import { BrowseFTPDirectory, DownloadFTPFile, BrowseSMBDirectory, DownloadSMBFile, BrowseSFTPDirectory, DownloadSFTPFile } from '../../wailsjs/go/backend/App'

const props = defineProps<{
  connection: any
}>()

const loading = ref(false)
const currentPath = ref('/')
const currentShare = ref('')
const files = ref<any[]>([])

const isSMB = computed(() => props.connection.type === 'SMB')
const isSFTP = computed(() => props.connection.type === 'SFTP')

const pathParts = computed(() => {
  if (currentPath.value === '/') return []
  return currentPath.value.split('/').filter(p => p)
})

const getPathUpTo = (index: number) => {
  const parts = pathParts.value.slice(0, index + 1)
  return '/' + parts.join('/')
}

onMounted(() => {
  loadDirectory()
})

watch(() => props.connection.result, () => {
  if (props.connection.result) {
    try {
      const data = JSON.parse(props.connection.result)
      if (data.files) {
        files.value = data.files
        currentPath.value = data.currentPath || '/'
        if (data.currentShare) {
          currentShare.value = data.currentShare
        }
      }
    } catch (e) {
      // 忽略非 JSON 格式
    }
  }
}, { immediate: true })

async function loadDirectory() {
  try {
    loading.value = true
    let result
    
    if (isSMB.value) {
      result = await BrowseSMBDirectory(props.connection.id, currentShare.value, currentPath.value)
    } else if (isSFTP.value) {
      result = await BrowseSFTPDirectory(props.connection.id, currentPath.value)
    } else {
      result = await BrowseFTPDirectory(props.connection.id, currentPath.value)
    }
    
    const data = JSON.parse(result)
    files.value = data.files || []
    currentPath.value = data.currentPath || '/'
    if (data.currentShare) {
      currentShare.value = data.currentShare
    }
  } catch (error) {
    message.error('加载目录失败: ' + error)
  } finally {
    loading.value = false
  }
}

function navigateTo(path: string, share: string = '') {
  currentPath.value = path
  if (isSMB.value) {
    currentShare.value = share
  }
  loadDirectory()
}

function refreshDirectory() {
  loadDirectory()
}

function handleFileClick(file: any) {
  if (file.type === 'folder') {
    navigateTo(file.path, currentShare.value)
  } else if (file.type === 'share') {
    currentShare.value = file.name
    currentPath.value = '/'
    loadDirectory()
  }
}

async function downloadFile(file: any) {
  try {
    if (isSMB.value) {
      await DownloadSMBFile(props.connection.id, currentShare.value, file.path)
    } else if (isSFTP.value) {
      await DownloadSFTPFile(props.connection.id, file.path)
    } else {
      await DownloadFTPFile(props.connection.id, file.path)
    }
    message.success('文件下载成功')
  } catch (error) {
    message.error('文件下载失败: ' + error)
  }
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return (bytes / Math.pow(k, i)).toFixed(2) + ' ' + sizes[i]
}
</script>

<style scoped>
.file-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
}

html.dark .file-browser {
  background: #141414;
}

/* 路径导航 */
.path-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  background: #f5f5f5;
  border-bottom: 1px solid #e8e8e8;
  gap: 8px;
}

html.dark .path-nav {
  background: #1a1a1a;
  border-bottom-color: #303030;
}

:deep(.ant-breadcrumb a) {
  color: #1890ff;
}

:deep(.ant-breadcrumb a:hover) {
  color: #40a9ff;
}

/* 文件表格 */
.file-table {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

:deep(.ant-spin-nested-loading),
:deep(.ant-spin-container) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.table-header {
  display: flex;
  background: #fafafa;
  border-bottom: 2px solid #e8e8e8;
  padding: 12px;
  font-weight: 600;
  color: #262626;
  flex-shrink: 0;
}

html.dark .table-header {
  background: #1f1f1f;
  border-bottom-color: #303030;
  color: rgba(255, 255, 255, 0.85);
}

.table-body {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
}

.table-row {
  display: flex;
  padding: 10px 12px;
  border-bottom: 1px solid #f0f0f0;
  transition: background 0.2s;
}

.table-row:hover {
  background: #f5f5f5;
}

.table-row.is-folder {
  cursor: pointer;
}

.table-row.is-folder:hover {
  background: #e6f7ff;
}

html.dark .table-row {
  border-bottom-color: #303030;
}

html.dark .table-row:hover {
  background: #262626;
}

html.dark .table-row.is-folder:hover {
  background: #111b26;
}

/* 列宽 */
.col-name {
  flex: 1;
  min-width: 150px;
  display: flex;
  align-items: center;
  gap: 8px;
  padding-right: 16px;
}

.col-size {
  width: 100px;
  text-align: right;
  flex-shrink: 0;
  padding-right: 16px;
}

.col-time {
  width: 150px;
  flex-shrink: 0;
  padding-right: 1px;
}

.col-action {
  width: 80px;
  text-align: center;
  flex-shrink: 0;
}

/* 图标和文本 */
.icon {
  font-size: 18px;
  flex-shrink: 0;
}

.folder-icon {
  color: #faad14;
}

.file-icon {
  color: #1890ff;
}

.name-text {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  color: #262626;
}

html.dark .name-text {
  color: rgba(255, 255, 255, 0.85);
}

.col-size {
  font-family: 'Consolas', 'Monaco', monospace;
  color: #595959;
  font-size: 13px;
}

html.dark .col-size {
  color: rgba(255, 255, 255, 0.65);
}

.col-time {
  color: #8c8c8c;
  font-size: 13px;
}

html.dark .col-time {
  color: rgba(255, 255, 255, 0.45);
}

/* 空状态 */
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
}

/* 滚动条 */
.table-body {
  scrollbar-width: thin;
  scrollbar-color: #bfbfbf #f5f5f5;
}

.table-body::-webkit-scrollbar {
  width: 8px;
}

.table-body::-webkit-scrollbar-track {
  background: #f5f5f5;
}

.table-body::-webkit-scrollbar-thumb {
  background: #bfbfbf;
  border-radius: 4px;
}

.table-body::-webkit-scrollbar-thumb:hover {
  background: #999;
}

html.dark .table-body {
  scrollbar-color: #595959 #1a1a1a;
}

html.dark .table-body::-webkit-scrollbar-track {
  background: #1a1a1a;
}

html.dark .table-body::-webkit-scrollbar-thumb {
  background: #595959;
}

html.dark .table-body::-webkit-scrollbar-thumb:hover {
  background: #737373;
}

/* 响应式布局 - 小窗口优化 */
@media (max-width: 900px) {
  .col-time {
    display: none;
  }
  
  .col-size {
    width: 90px;
  }
}

@media (max-width: 600px) {
  .col-size {
    width: 80px;
    font-size: 12px;
  }
  
  .col-action {
    width: 70px;
  }
  
  .col-name {
    min-width: 120px;
  }
  
  .name-text {
    font-size: 13px;
  }
}
</style>
