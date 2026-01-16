<template>
  <div class="hello-world">
    <!-- 欢迎标题 -->
    <div class="welcome-section glass-card">
      <h1 class="welcome-title">{{ msg }}</h1>
      <p class="welcome-subtitle">现代化的桌面应用开发模板</p>
    </div>

    <!-- 功能展示区域 -->
    <div class="features-grid">
      <!-- 计数器卡片 -->
      <div class="feature-card glass-card">
        <h3>🎯 交互计数器</h3>
        <div class="counter-section">
          <a-button type="primary" size="large" @click="count++">
            点击次数: {{ count }}
          </a-button>
        </div>
      </div>

      <!-- 按钮展示卡片 -->
      <div class="feature-card glass-card">
        <h3>🎨 按钮样式</h3>
        <a-space wrap>
          <a-button>默认</a-button>
          <a-button type="primary">主要</a-button>
          <a-button type="dashed">虚线</a-button>
          <a-button type="primary" danger>危险</a-button>
        </a-space>
      </div>

      <!-- 输入组件卡片 -->
      <div class="feature-card glass-card">
        <h3>📝 输入组件</h3>
        <a-space direction="vertical" style="width: 100%">
          <a-input v-model:value="inputValue" placeholder="输入一些文字..." />
          <a-select v-model:value="selectValue" style="width: 100%" placeholder="选择一个选项">
            <a-select-option value="option1">选项 1</a-select-option>
            <a-select-option value="option2">选项 2</a-select-option>
            <a-select-option value="option3">选项 3</a-select-option>
          </a-select>
        </a-space>
      </div>

      <!-- 通知卡片 -->
      <div class="feature-card glass-card">
        <h3>🔔 消息通知</h3>
        <a-space>
          <a-button @click="showSuccess">成功</a-button>
          <a-button @click="showWarning">警告</a-button>
          <a-button @click="showError">错误</a-button>
        </a-space>
      </div>
    </div>

    <!-- 技术栈信息 -->
    <div class="tech-stack glass-card">
      <h3>🚀 技术栈</h3>
      <div class="tech-tags">
        <a-tag color="blue">Wails v2</a-tag>
        <a-tag color="green">Vue 3</a-tag>
        <a-tag color="purple">TypeScript</a-tag>
        <a-tag color="orange">Ant Design Vue</a-tag>
        <a-tag color="red">Vite</a-tag>
        <a-tag color="cyan">Pinia</a-tag>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { message } from 'ant-design-vue'

defineProps<{ msg: string }>()

const count = ref(0)
const inputValue = ref('')
const selectValue = ref(undefined)

const showSuccess = () => {
  message.success('这是一个成功消息！')
}

const showWarning = () => {
  message.warning('这是一个警告消息！')
}

const showError = () => {
  message.error('这是一个错误消息！')
}
</script>

<style scoped>
.hello-world {
  padding: 20px;
  max-width: 1200px;
  margin: 0 auto;
}

.welcome-section {
  text-align: center;
  padding: 40px 20px;
  margin-bottom: 30px;
}

.welcome-title {
  font-size: 2.5rem;
  font-weight: 700;
  margin-bottom: 10px;
  background: linear-gradient(135deg, #1890ff, #722ed1);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.welcome-subtitle {
  font-size: 1.1rem;
  color: var(--text-color);
  opacity: 0.8;
  margin: 0;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 20px;
  margin-bottom: 30px;
}

.feature-card {
  padding: 24px;
  text-align: center;
}

.feature-card h3 {
  font-size: 1.2rem;
  margin-bottom: 16px;
  color: var(--text-color);
}

.counter-section {
  margin: 16px 0;
}

.tech-stack {
  padding: 24px;
  text-align: center;
}

.tech-stack h3 {
  font-size: 1.2rem;
  margin-bottom: 16px;
  color: var(--text-color);
}

.tech-tags {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
}

/* 响应式设计 */
@media (max-width: 768px) {
  .hello-world {
    padding: 10px;
  }
  
  .welcome-title {
    font-size: 2rem;
  }
  
  .features-grid {
    grid-template-columns: 1fr;
    gap: 15px;
  }
  
  .feature-card {
    padding: 20px;
  }
}
</style>
