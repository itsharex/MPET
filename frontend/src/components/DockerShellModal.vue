<template>
  <a-modal
    v-model:open="visible"
    title="Docker 容器 Shell"
    width="900px"
    :footer="null"
    @cancel="handleClose"
  >
    <div class="docker-shell-container">
      <!-- 容器选择 -->
      <div class="container-selector">
        <a-select
          v-model:value="selectedContainer"
          placeholder="选择容器"
          style="width: 100%"
          size="large"
          @change="handleContainerChange"
        >
          <a-select-option
            v-for="container in containers"
            :key="container.Id"
            :value="container.Id"
          >
            <div style="display: flex; align-items: center; justify-content: space-between">
              <span>{{ getContainerName(container) }}</span>
              <a-tag :color="container.State === 'running' ? 'success' : 'default'" size="small">
                {{ container.State }}
              </a-tag>
            </div>
          </a-select-option>
        </a-select>
      </div>

      <!-- 终端输出区域 -->
      <div class="terminal-output" ref="terminalRef">
        <div v-for="(item, index) in history" :key="index" class="terminal-line">
          <div v-if="item.type === 'command'" class="command-line">
            <span class="prompt">root@{{ getContainerName(getCurrentContainer()) }}:~#</span>
            <span class="command-text">{{ item.content }}</span>
          </div>
          <div v-else-if="item.type === 'output'" class="output-line">
            <pre>{{ item.content }}</pre>
          </div>
          <div v-else-if="item.type === 'error'" class="error-line">
            <pre>{{ item.content }}</pre>
          </div>
        </div>
        <div v-if="executing" class="executing-indicator">
          <LoadingOutlined /> 执行中...
        </div>
      </div>

      <!-- 命令输入区域 -->
      <div class="command-input-area">
        <div class="input-wrapper">
          <span class="prompt">root@{{ getContainerName(getCurrentContainer()) }}:~#</span>
          <a-input
            v-model:value="command"
            placeholder="输入命令..."
            size="large"
            :disabled="!selectedContainer || executing"
            @pressEnter="executeCommand"
            ref="commandInputRef"
            class="command-input"
          />
        </div>
        <a-button
          type="primary"
          size="large"
          :disabled="!selectedContainer || !command.trim() || executing"
          :loading="executing"
          @click="executeCommand"
        >
          执行
        </a-button>
      </div>

      <!-- 快捷命令 -->
      <div class="quick-commands">
        <span style="color: #8c8c8c; margin-right: 8px">快捷命令:</span>
        <a-space :size="4" wrap>
          <a-button
            v-for="cmd in quickCommands"
            :key="cmd.command"
            size="small"
            :disabled="!selectedContainer || executing"
            @click="insertCommand(cmd.command)"
          >
            {{ cmd.label }}
          </a-button>
        </a-space>
      </div>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import { LoadingOutlined } from '@ant-design/icons-vue'
import { ExecuteContainerCommand } from '../../wailsjs/go/backend/App'

interface DockerContainer {
  Id: string
  Names: string[]
  Image: string
  State: string
  Status: string
}

interface HistoryItem {
  type: 'command' | 'output' | 'error'
  content: string
}

const props = defineProps<{
  connectionId: string
  containers: DockerContainer[]
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const visible = ref(true)
const selectedContainer = ref<string>('')
const command = ref('')
const executing = ref(false)
const history = ref<HistoryItem[]>([])
const terminalRef = ref<HTMLElement>()
const commandInputRef = ref()

// 快捷命令
const quickCommands = [
  { label: 'ls -la', command: 'ls -la' },
  { label: 'pwd', command: 'pwd' },
  { label: 'whoami', command: 'whoami' },
  { label: 'id', command: 'id' },
  { label: 'uname -a', command: 'uname -a' },
  { label: 'ps aux', command: 'ps aux' },
  { label: 'env', command: 'env' },
  { label: 'cat /etc/passwd', command: 'cat /etc/passwd' },
]

// 获取容器名称
function getContainerName(container: DockerContainer | undefined): string {
  if (!container) return 'container'
  if (container.Names && container.Names.length > 0) {
    return container.Names[0].replace(/^\//, '')
  }
  return container.Id.substring(0, 12)
}

// 获取当前选中的容器
function getCurrentContainer(): DockerContainer | undefined {
  return props.containers.find(c => c.Id === selectedContainer.value)
}

// 容器切换
function handleContainerChange() {
  history.value = []
  const container = getCurrentContainer()
  if (container) {
    history.value.push({
      type: 'output',
      content: `已连接到容器: ${getContainerName(container)}\n容器 ID: ${container.Id}\n镜像: ${container.Image}\n状态: ${container.State}\n`
    })
  }
  scrollToBottom()
  focusInput()
}

// 插入命令
function insertCommand(cmd: string) {
  command.value = cmd
  focusInput()
}

// 执行命令
async function executeCommand() {
  if (!selectedContainer.value || !command.value.trim() || executing.value) {
    return
  }

  const cmd = command.value.trim()
  
  // 添加命令到历史
  history.value.push({
    type: 'command',
    content: cmd
  })

  // 清空输入
  command.value = ''
  executing.value = true

  try {
    const result = await ExecuteContainerCommand(
      props.connectionId,
      selectedContainer.value,
      cmd
    )

    // 添加输出到历史
    history.value.push({
      type: 'output',
      content: result || '(命令执行成功，无输出)'
    })
  } catch (error) {
    // 添加错误到历史
    history.value.push({
      type: 'error',
      content: `错误: ${error}`
    })
    message.error(`命令执行失败: ${error}`)
  } finally {
    executing.value = false
    scrollToBottom()
    focusInput()
  }
}

// 滚动到底部
function scrollToBottom() {
  nextTick(() => {
    if (terminalRef.value) {
      terminalRef.value.scrollTop = terminalRef.value.scrollHeight
    }
  })
}

// 聚焦输入框
function focusInput() {
  nextTick(() => {
    commandInputRef.value?.focus()
  })
}

// 关闭弹窗
function handleClose() {
  visible.value = false
  emit('close')
}

// 监听容器列表变化，自动选择第一个运行中的容器
watch(() => props.containers, (newContainers) => {
  if (newContainers && newContainers.length > 0 && !selectedContainer.value) {
    const runningContainer = newContainers.find(c => c.State === 'running')
    if (runningContainer) {
      selectedContainer.value = runningContainer.Id
      handleContainerChange()
    } else {
      selectedContainer.value = newContainers[0].Id
      handleContainerChange()
    }
  }
}, { immediate: true })

// 组件挂载后聚焦输入框
watch(visible, (newVal) => {
  if (newVal) {
    focusInput()
  }
})
</script>

<style scoped>
.docker-shell-container {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.container-selector {
  padding-bottom: 12px;
  border-bottom: 1px solid #f0f0f0;
}

.terminal-output {
  background: #1e1e1e;
  color: #d4d4d4;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  padding: 16px;
  border-radius: 4px;
  height: 400px;
  overflow-y: auto;
  overflow-x: hidden;
}

.terminal-line {
  margin-bottom: 8px;
}

.command-line {
  display: flex;
  gap: 8px;
  color: #4ec9b0;
}

.prompt {
  color: #4ec9b0;
  font-weight: 600;
  flex-shrink: 0;
}

.command-text {
  color: #ce9178;
}

.output-line pre {
  margin: 0;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.error-line pre {
  margin: 0;
  color: #f48771;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.executing-indicator {
  color: #569cd6;
  display: flex;
  align-items: center;
  gap: 8px;
}

.command-input-area {
  display: flex;
  gap: 8px;
  align-items: center;
}

.input-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  background: #f5f5f5;
  padding: 4px 12px;
  border-radius: 4px;
}

.input-wrapper .prompt {
  color: #4ec9b0;
  font-weight: 600;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  flex-shrink: 0;
}

.command-input {
  flex: 1;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
}

.command-input :deep(.ant-input) {
  background: transparent;
  border: none;
  box-shadow: none;
  padding: 0;
}

.command-input :deep(.ant-input:focus) {
  background: transparent;
  border: none;
  box-shadow: none;
}

.quick-commands {
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}

/* 暗色主题 */
html.dark .container-selector {
  border-bottom-color: #303030;
}

html.dark .input-wrapper {
  background: #262626;
}

html.dark .quick-commands {
  border-top-color: #303030;
}

/* 滚动条样式 */
.terminal-output::-webkit-scrollbar {
  width: 8px;
}

.terminal-output::-webkit-scrollbar-track {
  background: #2d2d2d;
}

.terminal-output::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 4px;
}

.terminal-output::-webkit-scrollbar-thumb:hover {
  background: #666;
}
</style>
