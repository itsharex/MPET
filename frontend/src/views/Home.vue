<template>
  <div 
    class="home-container"
    @drop="handleFileDrop"
    @dragover="handleDragOver"
    @dragenter="handleDragEnter"
    @dragleave="handleDragLeave"
  >
    <!-- 拖拽提示遮罩 -->
    <div v-if="isDragging" class="drag-overlay">
      <div class="drag-overlay-content">
        <ImportOutlined style="font-size: 64px; color: #1890ff; margin-bottom: 16px" />
        <div style="font-size: 24px; font-weight: 600; color: #1890ff; margin-bottom: 8px">
          释放文件以导入
        </div>
        <div style="font-size: 14px; color: #8c8c8c">
          支持 CSV、Fscan 和 Lightx 结果文件 (*.csv, *.txt)
        </div>
      </div>
    </div>
    
    <!-- 自定义标题栏 -->
    <TitleBar title="MPET - Multi-Protocol Exploitation Toolkit" />
    
    <a-layout style="min-height: calc(100vh - 32px)">
      <!-- 侧边栏 -->
      <a-layout-sider 
        v-model:collapsed="collapsed" 
        :collapsible="false"
        theme="light" 
        width="200" 
        class="custom-sider"
      >
        <div class="logo">
          <img src="/icon.svg" alt="MPET" style="width: 32px; height: 32px" />
          <span v-if="!collapsed" style="margin-left: 12px; font-weight: bold">MPET</span>
        </div>
        <a-menu 
          v-model:selectedKeys="selectedKeys" 
          mode="inline" 
          @click="handleMenuClick"
          class="scrollable-menu"
        >
          <a-menu-item key="all">
            <template #icon>
              <AppstoreOutlined />
            </template>
            <span>全部</span>
            <span v-if="!collapsed" class="menu-badge" style="background: #1890ff; color: white;">{{ allConnections.length }}</span>
          </a-menu-item>
          <a-menu-divider />
          <a-menu-item v-for="type in serviceTypes" :key="type.value" :class="`service-${type.value.toLowerCase()}`">
            <template #icon>
              <component :is="getServiceIcon(type.value)" style="width: 20px; height: 20px;" />
            </template>
            <span>{{ type.label }}</span>
            <span v-if="!collapsed" class="menu-badge">{{ getTypeCount(type.value) }}</span>
          </a-menu-item>
        </a-menu>
        
        <!-- 底部伸缩按钮 -->
        <div class="sider-trigger" @click="collapsed = !collapsed">
          <DoubleLeftOutlined v-if="!collapsed" />
          <DoubleRightOutlined v-else />
        </div>
      </a-layout-sider>

      <!-- 主内容区 -->
      <a-layout :style="{ marginLeft: collapsed ? '80px' : '200px', transition: 'margin-left 0.2s' }">
        <a-layout-header style="background: #fff; padding: 0 20px; display: flex; align-items: center; justify-content: space-between; height: 56px; border-bottom: 1px solid #f0f0f0">
          <a-space :size="12">
            <!-- 主要操作 -->
            <a-space :size="8">
              <a-button type="primary" size="small" @click="showAddModal">
                <PlusOutlined /> 添加
              </a-button>
              <a-button size="small" @click="handleImportCSV">
                <ImportOutlined /> 导入
              </a-button>
              <a-button size="small" @click="handleClipboardImport">
                <CopyOutlined /> 剪贴板导入
              </a-button>
            </a-space>
            
            <!-- 批量操作 -->
            <a-divider type="vertical" style="height: 24px; margin: 0" />
            <a-space :size="8">
              <a-button 
                type="primary" 
                size="small"
                :disabled="selectedRowKeys.length === 0"
                @click="handleBatchConnectWrapper"
              >
                <ApiOutlined /> 批量连接 <span v-if="selectedRowKeys.length > 0">({{ selectedRowKeys.length }})</span>
              </a-button>
              <a-button 
                size="small"
                :disabled="selectedRowKeys.length === 0"
                :loading="exporting"
                @click="handleExportReport"
              >
                <FileTextOutlined /> 导出报告 <span v-if="selectedRowKeys.length > 0">({{ selectedRowKeys.length }})</span>
              </a-button>
              <a-button 
                danger 
                size="small"
                :disabled="selectedRowKeys.length === 0"
                @click="handleBatchDelete"
              >
                <DeleteOutlined /> 批量删除 <span v-if="selectedRowKeys.length > 0">({{ selectedRowKeys.length }})</span>
              </a-button>
            </a-space>
            
            <!-- 设置 -->
            <a-divider type="vertical" style="height: 24px; margin: 0" />
            <a-space :size="8">
              <a-button size="small" @click="showProxyModal">
                <SettingOutlined /> 代理
              </a-button>
              <a-button size="small" @click="showVulnModal">
                <FileTextOutlined /> 漏洞信息
              </a-button>
            </a-space>
          </a-space>
          
          <!-- 右侧工具按钮 -->
          <a-space :size="8">
            <a-button type="text" size="small" @click="showLogModal">
              <FileTextOutlined /> 日志
            </a-button>
            <a-button type="text" size="small" @click="loadConnections">
              <ReloadOutlined :spin="loading" /> 刷新
            </a-button>
          </a-space>
        </a-layout-header>

        <a-layout-content style="margin: 16px; background: #fff; padding: 16px; overflow-y: auto; height: calc(100vh - 32px - 56px - 32px)">
         <!-- 筛选栏和分页 -->
          <div style="margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; justify-content: space-between; align-items: center">
            <div style="display: flex; gap: 8px; flex-wrap: wrap; align-items: center">
              <a-select 
                v-model:value="filters.types" 
                mode="multiple"
                placeholder="服务类型" 
                style="width: 140px" 
                size="small" 
                allow-clear
                :max-tag-count="1"
                :max-tag-text-length="6"
                class="compact-select"
              >
                <a-select-option v-for="type in serviceTypes" :key="type.value" :value="type.value">
                  <span style="display: flex; align-items: center; gap: 6px">
                    <component :is="getServiceIcon(type.value)" style="width: 16px; height: 16px;" />
                    {{ type.label }}
                  </span>
                </a-select-option>
              </a-select>
              <a-input v-model:value="filters.ip" placeholder="IP 地址" style="width: 120px" size="small" />
              <a-input v-model:value="filters.user" placeholder="用户名" style="width: 100px" size="small" />
              <a-select v-model:value="filters.status" placeholder="状态" style="width: 90px" size="small" allow-clear>
                <a-select-option value="success">成功</a-select-option>
                <a-select-option value="failed">失败</a-select-option>
                <a-select-option value="pending">待连接</a-select-option>
              </a-select>
              <a-input v-model:value="filters.message" placeholder="消息内容" style="width: 120px" size="small" />
              <a-button size="small" @click="resetFilters">重置</a-button>
            </div>
            <a-pagination
              v-model:current="pagination.current"
              v-model:page-size="pagination.pageSize"
              :total="filteredConnections.length"
              :show-size-changer="true"
              :show-total="(total) => `共 ${total} 条`"
              :page-size-options="['15', '30', '50', '100']"
              size="small"
              style="margin: 0"
            />
          </div>

          <!-- 连接列表 -->
          <a-table
            :columns="columns"
            :data-source="paginatedConnections"
            :loading="loading"
            :row-selection="{ selectedRowKeys, onChange: onSelectChange }"
            :pagination="false"
            :expandedRowKeys="expandedRowKeys"
            :scroll="{ x: 900 }"
            @expand="handleExpand"
            row-key="id"
            size="small"
          >
            <template #bodyCell="{ column, record }">
              <template v-if="column.key === 'type'">
                <span :class="`service-${record.type.toLowerCase()}`" style="display: flex; align-items: center; gap: 6px">
                  <component :is="getServiceIcon(record.type)" style="width: 18px; height: 18px;" />
                  <span>{{ record.type }}</span>
                </span>
              </template>
              <template v-if="column.key === 'status'">
                <a-tag :color="getStatusColor(record.status)">
                  {{ getStatusText(record.status) }}
                </a-tag>
              </template>
              <template v-if="column.key === 'action'">
                <a-space :size="2">
                  <a-button type="link" size="small" @click="handleConnectWrapper(record.id)">
                    <ApiOutlined /> 重连
                  </a-button>
                  <a-button type="link" size="small" @click="showEditModal(record)">
                    <EditOutlined /> 编辑
                  </a-button>
                  <a-button type="link" size="small" @click="toggleDetail(record.id)">
                    <EyeOutlined /> {{ expandedRowKeys.includes(record.id) ? '收起' : '详情' }}
                  </a-button>
                  <a-popconfirm title="确定删除?" @confirm="handleDelete(record.id)">
                    <a-button type="link" danger size="small">
                      <DeleteOutlined /> 删除
                    </a-button>
                  </a-popconfirm>
                </a-space>
              </template>
            </template>

            <!-- 展开行内容 -->
            <template #expandedRowRender="{ record }">
              <div class="expanded-row-container expanded-content" :data-connection-id="record.id">
                <!-- 上方：连接信息 -->
                <div class="expanded-card" style="margin-bottom: 12px">
                  <div class="expanded-card-title expanded-card-title-blue">
                    📋 连接信息
                  </div>
                  <a-row :gutter="[12, 8]">
                    <!-- 第一行：服务类型、IP、端口 -->
                    <a-col :span="8">
                      <div class="info-field">
                        <span class="info-label">服务类型</span>
                        <span :class="`service-${record.type.toLowerCase()}`" class="info-value" style="display: flex; align-items: center; gap: 6px">
                          <component :is="getServiceIcon(record.type)" style="width: 18px; height: 18px; flex-shrink: 0;" />
                          <a-tag :color="getServiceColor(record.type)" style="margin: 0; font-size: 12px">{{ record.type }}</a-tag>
                        </span>
                      </div>
                    </a-col>
                    <a-col :span="8">
                      <div class="info-field">
                        <span class="info-label">IP 地址</span>
                        <span class="info-value info-value-primary">{{ record.ip }}</span>
                      </div>
                    </a-col>
                    <a-col :span="8">
                      <div class="info-field">
                        <span class="info-label">端口</span>
                        <span class="info-value info-value-primary">{{ record.port }}</span>
                      </div>
                    </a-col>
                    
                    <!-- 第二行：状态、用户名、密码 -->
                    <a-col :span="8">
                      <div class="info-field">
                        <span class="info-label">状态</span>
                        <a-tag :color="getStatusColor(record.status)" style="font-size: 12px; margin: 0">
                          {{ getStatusText(record.status) }}
                        </a-tag>
                      </div>
                    </a-col>
                    <a-col :span="8">
                      <div class="info-field">
                        <span class="info-label">用户名</span>
                        <span class="info-value">{{ record.user || '-' }}</span>
                      </div>
                    </a-col>
                    <a-col :span="8">
                      <div class="info-field">
                        <span class="info-label">密码</span>
                        <span class="info-value">{{ record.pass ? '******' : '-' }}</span>
                      </div>
                    </a-col>
                    
                    <!-- 第三行：创建时间、连接时间、消息 -->
                    <a-col :span="8">
                      <div class="info-field">
                        <span class="info-label">创建时间</span>
                        <span class="info-value info-value-time">{{ formatTime(record.created_at) }}</span>
                      </div>
                    </a-col>
                    <a-col :span="8" v-if="record.connected_at">
                      <div class="info-field">
                        <span class="info-label">连接时间</span>
                        <span class="info-value info-value-time">{{ formatTime(record.connected_at) }}</span>
                      </div>
                    </a-col>
                    <a-col :span="record.connected_at ? 8 : 16">
                      <div class="info-field">
                        <span class="info-label">消息</span>
                        <span class="info-value info-value-message">{{ record.message || '-' }}</span>
                      </div>
                    </a-col>
                  </a-row>
                </div>

                <!-- 下方：日志和结果 -->
                <div class="resizable-container" :style="{ height: panelHeights[record.id] || '450px' }">
                  <div class="resizable-panels">
                    <!-- 左侧：连接日志 -->
                    <div class="resizable-panel" :style="{ width: panelWidths[record.id] || '40%' }">
                      <div class="expanded-card" style="height: 100%; display: flex; flex-direction: column">
                        <div class="expanded-card-title expanded-card-title-green">
                          📝 连接日志
                        </div>
                        <div class="expanded-log-container" :ref="el => setLogRef(record.id, el)">
                          <div v-for="(log, index) in record.logs" :key="index" class="expanded-log-item">
                            {{ log }}
                          </div>
                          <div v-if="!record.logs || record.logs.length === 0" class="expanded-empty-text">
                            暂无日志
                          </div>
                        </div>
                      </div>
                    </div>

                    <!-- 垂直分隔条 -->
                    <div 
                      class="resizer resizer-vertical" 
                      @mousedown="startVerticalResize($event, record.id)"
                    ></div>

                    <!-- 右侧：执行结果 / 文件浏览器 -->
                    <div class="resizable-panel" style="flex: 1">
                      <div class="expanded-card" style="height: 100%; display: flex; flex-direction: column">
                        <div class="expanded-card-title expanded-card-title-blue">
                          <template v-if="record.type === 'FTP' || record.type === 'SMB' || record.type === 'SFTP'">
                            📁 文件浏览器
                          </template>
                          <template v-else>
                            💻 执行结果
                          </template>
                        </div>
                        
                        <!-- FTP/SMB/SFTP 文件浏览器 -->
                        <div v-if="(record.type === 'FTP' || record.type === 'SMB' || record.type === 'SFTP') && record.result" style="flex: 1; overflow: hidden;">
                          <FTPFileBrowser :connection="record" @refresh="loadConnections" />
                        </div>
                        
                        <!-- VNC/RDP 截图显示 -->
                        <div 
                          v-else-if="record.type === 'VNC' || record.type === 'RDP'" 
                          class="expanded-log-container"
                          style="overflow: auto;"
                          :ref="el => setResultRef(record.id, el)"
                        >
                          <div v-if="record.result" v-html="renderVNCResult(record.result)"></div>
                          <div v-else style="display: flex; align-items: center; justify-content: center; height: 100%;">
                            <a-button type="primary" size="large" @click="handleVNCScreenshot(record)" :loading="commandLoading[record.id]">
                              <CameraOutlined /> 获取屏幕截图
                            </a-button>
                          </div>
                        </div>
                        
                        <!-- 普通执行结果显示 -->
                        <div 
                          v-else-if="record.result && record.type !== 'FTP' && record.type !== 'SMB' && record.type !== 'SFTP' && record.type !== 'VNC' && record.type !== 'RDP'" 
                          class="expanded-log-container"
                          :ref="el => setResultRef(record.id, el)"
                        >
                          <pre class="expanded-log-item" style="margin: 0; white-space: pre-wrap; word-wrap: break-word">{{ record.result }}</pre>
                        </div>
                        
                        <!-- 空状态 -->
                        <div v-else class="expanded-log-container" style="display: flex; align-items: center; justify-content: center">
                          <span class="expanded-empty-text" style="padding: 0">
                            <template v-if="record.type === 'FTP' || record.type === 'SMB' || record.type === 'SFTP'">
                              等待文件列表加载...
                            </template>
                            <template v-else>
                              暂无执行结果
                            </template>
                          </span>
                        </div>
                      </div>
                    </div>
                  </div>

                  <!-- 水平分隔条（底部） -->
                  <div 
                    class="resizer resizer-horizontal" 
                    @mousedown="startHorizontalResize($event, record.id)"
                  ></div>
                </div>
                
                <!-- 命令执行区域 -->
                <div v-if="supportsCommand(record.type)" class="expanded-card" style="margin-top: 12px">
                  <div class="expanded-card-title expanded-card-title-purple">
                    ⚡ 命令执行
                  </div>
                  <a-row :gutter="12">
                    <!-- 常用命令下拉选择 -->
                    <a-col :span="record.type === 'Docker' ? 6 : 8">
                      <a-select
                        v-model:value="selectedCommand[record.id]"
                        placeholder="选择常用命令"
                        size="large"
                        style="width: 100%"
                        @change="(value) => onCommandSelect(record, value)"
                      >
                        <a-select-option 
                          v-for="cmd in getCommonCommands(record.type)" 
                          :key="cmd.command"
                          :value="cmd.command"
                        >
                          {{ cmd.label }} - {{ cmd.description }}
                        </a-select-option>
                      </a-select>
                    </a-col>
                    
                    <!-- 命令输入 -->
                    <a-col :span="record.type === 'Docker' ? 12 : 16">
                      <a-input-search
                        v-model:value="commandInputs[record.id]"
                        placeholder="输入自定义命令或从左侧选择常用命令"
                        enter-button="执行"
                        size="large"
                        @search="handleExecuteCommand(record)"
                        :loading="commandLoading[record.id]"
                      >
                        <template #prefix>
                          <CodeOutlined style="color: #722ed1" />
                        </template>
                      </a-input-search>
                    </a-col>

                    <!-- Docker Shell 按钮 -->
                    <a-col v-if="record.type === 'Docker'" :span="6">
                      <a-button
                        type="primary"
                        size="large"
                        block
                        @click="showDockerShell(record)"
                        :disabled="record.status !== 'success'"
                      >
                        <CodeOutlined /> 容器 Shell
                      </a-button>
                    </a-col>
                  </a-row>
                </div>
              </div>
            </template>
          </a-table>
        </a-layout-content>
      </a-layout>
    </a-layout>

    <!-- 添加/编辑连接对话框 -->
    <a-modal
      v-model:open="addModalVisible"
      :title="editingConnection ? '编辑连接' : '添加连接'"
      @ok="handleAddConnection"
      @cancel="resetForm"
      width="600px"
    >
      <a-form :model="form" layout="vertical">
        <a-form-item label="服务类型" required>
          <a-select 
            v-model:value="form.type" 
            placeholder="选择或搜索服务类型" 
            @change="handleTypeChange"
            show-search
            option-filter-prop="label"
          >
            <a-select-option 
              v-for="type in serviceTypes" 
              :key="type.value" 
              :value="type.value"
              :label="type.label"
            >
              {{ type.label }}
            </a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="IP 地址" required>
          <a-input v-model:value="form.ip" placeholder="192.168.1.100" />
        </a-form-item>
        <a-form-item label="端口" required>
          <a-input v-model:value="form.port" placeholder="默认端口" />
        </a-form-item>
        <a-form-item label="用户名">
          <a-input v-model:value="form.user" placeholder="可选，留空尝试未授权访问" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model:value="form.pass" placeholder="可选" />
        </a-form-item>
        
        <!-- 测试连接按钮 -->
        <a-form-item>
          <a-button 
            type="dashed" 
            block 
            @click="handleTestConnectionWrapper" 
            :loading="testingConnection"
            :disabled="!form.type || !form.ip || !form.port"
          >
            <ApiOutlined v-if="!testingConnection" /> 测试连接
          </a-button>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 代理设置对话框 -->
    <a-modal
      v-model:open="proxyModalVisible"
      title="代理设置"
      @ok="handleSaveProxy"
      width="500px"
    >
      <a-form :model="proxyForm" layout="vertical">
        <a-form-item label="启用代理">
          <a-switch v-model:checked="proxyForm.enabled" />
        </a-form-item>
        <a-form-item label="代理类型">
          <a-select v-model:value="proxyForm.type" :disabled="!proxyForm.enabled">
            <a-select-option value="socks5">SOCKS5</a-select-option>
          </a-select>
        </a-form-item>
        <a-form-item label="代理主机" required>
          <a-input v-model:value="proxyForm.host" placeholder="127.0.0.1" :disabled="!proxyForm.enabled" />
        </a-form-item>
        <a-form-item label="代理端口" required>
          <a-input v-model:value="proxyForm.port" placeholder="1080" :disabled="!proxyForm.enabled" />
        </a-form-item>
        <a-form-item label="用户名">
          <a-input v-model:value="proxyForm.user" placeholder="可选" :disabled="!proxyForm.enabled" />
        </a-form-item>
        <a-form-item label="密码">
          <a-input-password v-model:value="proxyForm.pass" placeholder="可选" :disabled="!proxyForm.enabled" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 系统日志对话框 -->
    <a-modal
      v-model:open="logModalVisible"
      title="系统日志"
      :footer="null"
      width="800px"
      :bodyStyle="{ padding: '16px', maxHeight: '600px', overflow: 'auto' }"
    >
      <div style="margin-bottom: 12px; display: flex; justify-content: space-between; align-items: center">
        <a-space>
          <a-button size="small" @click="loadLogs" :loading="logLoading">
            <ReloadOutlined /> 刷新日志
          </a-button>
          <a-button size="small" @click="clearLogDisplay">
            <ClearOutlined /> 清空显示
          </a-button>
        </a-space>
        <span style="color: #666; font-size: 12px">共 {{ systemLogs.length }} 条日志</span>
      </div>
      <div style="background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-family: 'Consolas', 'Monaco', monospace; font-size: 12px; line-height: 1.6; white-space: pre-wrap; word-break: break-all">
        <div v-if="systemLogs.length === 0" style="color: #888; text-align: center; padding: 20px">
          暂无日志
        </div>
        <div v-else v-for="(log, index) in systemLogs" :key="index" style="margin-bottom: 4px">
          {{ log }}
        </div>
      </div>
    </a-modal>

    <!-- Docker Shell 弹窗 -->
    <DockerShellModal
      v-if="dockerShellVisible"
      :connection-id="dockerShellConnectionId"
      :containers="dockerShellContainers"
      @close="dockerShellVisible = false"
    />

    <!-- 漏洞信息管理弹窗 -->
    <VulnerabilityModal
      :open="vulnModalVisible"
      @close="vulnModalVisible = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { message } from 'ant-design-vue'
import {
  AppstoreOutlined,
  PlusOutlined,
  ImportOutlined,
  CopyOutlined,
  ApiOutlined,
  DeleteOutlined,
  SettingOutlined,
  ReloadOutlined,
  EditOutlined,
  EyeOutlined,
  CodeOutlined,
  FileTextOutlined,
  DoubleLeftOutlined,
  DoubleRightOutlined,
  ClearOutlined,
  CameraOutlined,
} from '@ant-design/icons-vue'
import { GetServiceTypes, BrowseFTPDirectory, DownloadFTPFile, BrowseSMBDirectory, DownloadSMBFile, BrowseSFTPDirectory, DownloadSFTPFile, ImportCSV } from '../../wailsjs/go/backend/App'
import TitleBar from '../components/TitleBar.vue'
import FTPFileBrowser from '../components/FTPFileBrowser.vue'
import DockerShellModal from '../components/DockerShellModal.vue'
import VulnerabilityModal from '../components/VulnerabilityModal.vue'
import { useThemeStore } from '../stores/theme'
import { getServiceColor, getStatusColor, getStatusText, getServiceIcon, formatTime, supportsCommand, getCommonCommands, renderVNCResult } from '../utils/formatters'
import { useConnections } from '../composables/useConnections'
import { useImport } from '../composables/useImport'
import { useCommand } from '../composables/useCommand'
import { useProxy } from '../composables/useProxy'
import { usePanel } from '../composables/usePanel'
import { useFilters } from '../composables/useFilters'
import { useSystemLogs } from '../composables/useSystemLogs'
import { useConnectionForm } from '../composables/useConnectionForm'
import { useReport } from '../composables/useReport'

const themeStore = useThemeStore()

// 使用连接管理 composable
const {
  loading,
  connections,
  allConnections,
  selectedRowKeys,
  expandedRowKeys,
  loadConnections,
  handleAddConnection: addConnection,
  handleUpdateConnection: updateConnection,
  handleConnect,
  handleBatchConnect,
  handleDelete,
  handleBatchDelete,
  handleTestConnection: testConnection,
  toggleDetail,
  onSelectChange,
  getTypeCount,
} = useConnections()

// 使用导入功能 composable
const {
  isDragging,
  handleImportCSV,
  handleClipboardImport,
  handleDragOver,
  handleDragEnter,
  handleDragLeave,
  handleFileDrop,
} = useImport(() => loadConnections())

// 使用命令执行 composable
const {
  commandInputs,
  commandLoading,
  selectedCommand,
  onCommandSelect,
  handleExecuteCommand: executeCommand,
  handleVNCScreenshot: vncScreenshot,
} = useCommand((id) => {
  scrollResultToBottom(id)
  scrollLogToBottom(id)
})

// 使用代理配置 composable
const {
  proxyModalVisible,
  proxyForm,
  loadProxySettings,
  showProxyModal,
  handleSaveProxy,
} = useProxy()

// 使用面板调整 composable
const {
  panelWidths,
  panelHeights,
  startVerticalResize,
  startHorizontalResize,
} = usePanel()

// 使用筛选和分页 composable
const {
  filters,
  pagination,
  filteredConnections,
  paginatedConnections,
  resetFilters,
} = useFilters(connections)

// 使用系统日志 composable
const {
  logModalVisible,
  logLoading,
  systemLogs,
  showLogModal,
  loadLogs,
  clearLogDisplay,
} = useSystemLogs()

// 使用报告导出 composable
const {
  exporting,
  exportReportWithScreenshots,
} = useReport()

const collapsed = ref(false)
const selectedKeys = ref(['all'])
const serviceTypes = ref<any[]>([])

// 使用表单和模态框 composable
const {
  addModalVisible,
  editingConnection,
  testingConnection,
  form,
  showAddModal,
  showEditModal,
  resetForm,
  handleTypeChange,
} = useConnectionForm(serviceTypes)

let dragCounterRemoved = 0 // 移除旧的 dragCounter

// 结果容器 ref
const resultRefs = ref<Record<string, HTMLElement>>({})
const logRefs = ref<Record<string, HTMLElement>>({})

// 自动滚动标记（用于区分自动操作和手动展开）
const autoScrollIds = ref<Set<string>>(new Set())

// Docker Shell 相关
const dockerShellVisible = ref(false)
const dockerShellConnectionId = ref('')
const dockerShellContainers = ref<any[]>([])

// 漏洞信息管理
const vulnModalVisible = ref(false)

function showVulnModal() {
  vulnModalVisible.value = true
}

// 设置结果容器 ref
function setResultRef(id: string, el: any) {
  if (el) {
    resultRefs.value[id] = el
  }
}

// 设置日志容器 ref
function setLogRef(id: string, el: any) {
  if (el) {
    logRefs.value[id] = el
  }
}

// 滚动到结果容器底部 - 始终自动滚动
function scrollResultToBottom(id: string) {
  const el = resultRefs.value[id]
  if (el) {
    // 使用 nextTick 确保 DOM 已更新
    setTimeout(() => {
      el.scrollTop = el.scrollHeight
    }, 100)
  }
}

// 滚动到日志容器底部 - 始终自动滚动
function scrollLogToBottom(id: string) {
  const el = logRefs.value[id]
  if (el) {
    setTimeout(() => {
      el.scrollTop = el.scrollHeight
    }, 100)
  }
}

const columns = [
  { title: '类型', dataIndex: 'type', key: 'type', width: 120 },
  { title: 'IP', dataIndex: 'ip', key: 'ip', width: 120 },
  { title: '端口', dataIndex: 'port', key: 'port', width: 70 },
  { title: '用户名', dataIndex: 'user', key: 'user', width: 92 },
  { title: '状态', key: 'status', width: 60 },
  { title: '消息', dataIndex: 'message', key: 'message', width: 200, ellipsis: true },
  { title: '操作', key: 'action', width: 250, fixed: 'right', align: 'center' },
]

onMounted(async () => {
  // 初始化主题
  themeStore.initTheme()
  
  await loadServiceTypes()
  await loadConnections()
  await loadProxySettings()
})

async function loadServiceTypes() {
  try {
    serviceTypes.value = await GetServiceTypes()
  } catch (error) {
    console.error('加载服务类型失败:', error)
  }
}

function handleMenuClick({ key }: any) {
  selectedKeys.value = [key]
  loadConnections(key)
}

async function handleAddConnection() {
  if (editingConnection.value) {
    await updateConnection(editingConnection.value.id, form.value, () => {
      addModalVisible.value = false
      resetForm()
    })
  } else {
    await addConnection(form.value, () => {
      addModalVisible.value = false
      resetForm()
    })
  }
}

async function handleTestConnectionWrapper() {
  testingConnection.value = true
  await testConnection(form.value)
  testingConnection.value = false
}

async function handleConnectWrapper(id: string) {
  // 标记为自动滚动
  autoScrollIds.value.add(id)
  
  await handleConnect(id, (id) => {
    scrollResultToBottom(id)
    scrollLogToBottom(id)
  })
}

async function handleBatchConnectWrapper() {
  // 标记所有选中的连接为自动滚动
  selectedRowKeys.value.forEach(id => {
    autoScrollIds.value.add(id as string)
  })
  
  await handleBatchConnect((ids) => {
    ids.forEach(id => {
      if (expandedRowKeys.value.includes(id)) {
        scrollResultToBottom(id)
        scrollLogToBottom(id)
      }
    })
  })
}

// 导出报告
async function handleExportReport() {
  if (selectedRowKeys.value.length === 0) {
    return
  }

  // 先展开所有选中的行
  const idsToExpand = selectedRowKeys.value.filter(id => !expandedRowKeys.value.includes(id))
  if (idsToExpand.length > 0) {
    expandedRowKeys.value.push(...idsToExpand)
    // 等待 DOM 更新
    await nextTick()
    // 再等待一下确保内容渲染完成
    await new Promise(resolve => setTimeout(resolve, 500))
  }

  // 获取卡片元素的函数
  const getCardElement = (id: string): HTMLElement | null => {
    // 查找对应的表格行
    const row = document.querySelector(`tr[data-row-key="${id}"]`)
    if (!row) {
      console.warn(`未找到行: ${id}`)
      return null
    }
    
    // 查找展开的内容区域
    const expandedRow = row.nextElementSibling
    if (expandedRow && expandedRow.classList.contains('ant-table-expanded-row')) {
      const content = expandedRow.querySelector('.expanded-content') as HTMLElement
      if (content) {
        console.log(`找到展开内容: ${id}`, content)
        return content
      }
    }
    
    console.warn(`未找到展开内容: ${id}`)
    return null
  }

  await exportReportWithScreenshots(selectedRowKeys.value as string[], getCardElement)
}

function handleExpand(expanded: boolean, record: any) {
  if (expanded) {
    if (!expandedRowKeys.value.includes(record.id)) {
      expandedRowKeys.value.push(record.id)
    }
    // 手动展开时，只有在自动滚动列表中的才滚动到底部
    if (autoScrollIds.value.has(record.id)) {
      setTimeout(() => {
        scrollResultToBottom(record.id)
        scrollLogToBottom(record.id)
      }, 200)
      // 滚动后移除标记
      autoScrollIds.value.delete(record.id)
    }
    // 手动展开时不滚动，保持用户当前位置
  } else {
    const index = expandedRowKeys.value.indexOf(record.id)
    if (index > -1) {
      expandedRowKeys.value.splice(index, 1)
    }
  }
}

// 包装命令执行函数以传递 loadConnections
async function handleExecuteCommand(record: any) {
  await executeCommand(record, loadConnections)
}

async function handleVNCScreenshot(record: any) {
  await vncScreenshot(record, loadConnections)
}

// 显示 Docker Shell 弹窗
async function showDockerShell(record: any) {
  if (record.type !== 'Docker' || record.status !== 'success') {
    message.warning('仅支持已成功连接的 Docker 服务')
    return
  }

  try {
    // 直接从 API 获取容器列表
    const { GetDockerContainers } = await import('../../wailsjs/go/backend/App')
    const containersJSON = await GetDockerContainers(record.id)
    const containers = JSON.parse(containersJSON)
    
    if (containers.length === 0) {
      message.warning('未找到容器')
      return
    }

    dockerShellConnectionId.value = record.id
    dockerShellContainers.value = containers
    dockerShellVisible.value = true
  } catch (error) {
    message.error(`获取容器列表失败: ${error}`)
  }
}

// 从 Docker 结果中解析容器列表（已废弃，保留用于兼容）
function parseDockerContainers(result: string): any[] {
  if (!result) return []

  const containers: any[] = []
  const lines = result.split('\n')
  
  let inContainerSection = false
  for (const line of lines) {
    if (line.includes('【容器列表】')) {
      inContainerSection = true
      continue
    }
    if (line.includes('【镜像列表】') || line.includes('【安全建议】')) {
      inContainerSection = false
      continue
    }
    
    if (inContainerSection && line.trim().startsWith('[')) {
      // 解析容器信息
      const nameMatch = line.match(/\[\d+\]\s+(.+)/)
      if (nameMatch) {
        const name = nameMatch[1].trim()
        
        // 查找后续行获取更多信息
        const nextLines = lines.slice(lines.indexOf(line) + 1, lines.indexOf(line) + 4)
        let image = ''
        let state = 'unknown'
        let id = ''
        
        for (const nextLine of nextLines) {
          if (nextLine.includes('镜像:')) {
            image = nextLine.split('镜像:')[1]?.trim() || ''
          }
          if (nextLine.includes('状态:')) {
            const statusMatch = nextLine.match(/状态:\s+(\w+)/)
            if (statusMatch) {
              state = statusMatch[1]
            }
          }
          if (nextLine.includes('ID:')) {
            id = nextLine.split('ID:')[1]?.trim() || ''
          }
        }
        
        // 如果没有 ID，尝试从名称生成一个临时 ID
        if (!id) {
          id = name.replace(/[^a-zA-Z0-9]/g, '_')
        }
        
        containers.push({
          Id: id,
          Names: [name],
          Image: image,
          State: state,
          Status: state
        })
      }
    }
  }
  
  return containers
}

</script>

<style scoped>
.home-container {
  height: 100vh;
  overflow: hidden;
}

/* 自定义侧边栏 */
.custom-sider {
  height: calc(100vh - 32px) !important;
  position: fixed !important;
  left: 0;
  top: 32px;
  bottom: 0;
  overflow: hidden !important;
  display: flex;
  flex-direction: column;
}

.logo {
  height: 57px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 18px;
  border-bottom: 1px solid #f0f0f0;
  flex-shrink: 0;
}

/* 可滚动菜单 */
.scrollable-menu {
  flex: 1;
  overflow-y: auto !important;
  overflow-x: hidden !important;
  height: calc(100vh - 32px - 57px - 48px) !important;
}

:deep(.ant-table-cell) {
  padding: 6px 4px !important;
}




:deep(.ant-btn-sm) {
  padding: 0 4px;
  font-size: 13px;
  height: 24px;
  line-height: 22px;
}

:deep(.ant-btn-link) {
  padding: 0 6px;
}

:deep(.ant-space-item) {
  margin-right: 0 !important;
}

/* 侧边栏滚动条样式 */
.scrollable-menu {
  scrollbar-width: thin;
  scrollbar-color: #d9d9d9 transparent;
}

.scrollable-menu::-webkit-scrollbar {
  width: 6px;
}

.scrollable-menu::-webkit-scrollbar-track {
  background: transparent;
}

.scrollable-menu::-webkit-scrollbar-thumb {
  background-color: #d9d9d9;
  border-radius: 3px;
}

.scrollable-menu::-webkit-scrollbar-thumb:hover {
  background-color: #bfbfbf;
}

/* 暗色主题滚动条 */
html.dark .scrollable-menu {
  scrollbar-color: #434343 transparent;
}

html.dark .scrollable-menu::-webkit-scrollbar-thumb {
  background-color: #434343;
}

html.dark .scrollable-menu::-webkit-scrollbar-thumb:hover {
  background-color: #595959;
}

/* 确保菜单不会被遮挡 */
:deep(.ant-menu) {
  border-right: none;
}

/* 侧边栏底部伸缩按钮 */
.sider-trigger {
  height: 48px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fafafa;
  border-top: 1px solid #f0f0f0;
  cursor: pointer;
  transition: all 0.3s;
  color: #595959;
  font-size: 16px;
  flex-shrink: 0;
  margin-top: auto;
}

.sider-trigger:hover {
  background: #f0f0f0;
  color: #1890ff;
}

html.dark .sider-trigger {
  background: #1f1f1f;
  border-top-color: #303030;
  color: #8c8c8c;
}

html.dark .sider-trigger:hover {
  background: #262626;
  color: #1890ff;
}

/* 侧边栏收缩时图标放大 */
:deep(.ant-layout-sider-collapsed .ant-menu-item .anticon) {
  font-size: 20px !important;
}

:deep(.ant-layout-sider-collapsed .logo img) {
  width: 50px !important;
  height: 50px !important;
}

/* 菜单项数量徽章 */
.menu-badge {
  position: absolute;
  right: 16px;
  top: 50%;
  transform: translateY(-50%);
  min-width: 20px;
  height: 20px;
  line-height: 20px;
  padding: 0 6px;
  font-size: 12px;
  text-align: center;
  background: #d9d9d9;
  color: #595959;
  border-radius: 10px;
  font-weight: 500;
}

:deep(.ant-menu-item) {
  position: relative;
}

html.dark .menu-badge {
  background: #434343;
  color: #d9d9d9;
}

/* 暗色主题 - Logo 边框 */
html.dark .logo {
  border-bottom-color: #303030;
}

/* 暗色主题 - 侧边栏收缩时图标 */
html.dark :deep(.ant-layout-sider-collapsed .logo img) {
  filter: brightness(0.9);
}

/* 展开行样式类 */
.expanded-row-container {
  background: #fafafa;
  padding: 12px;
  border-radius: 4px;
}

.expanded-card {
  background: #fff;
  padding: 12px;
  border-radius: 4px;
  border: 1px solid #e8e8e8;
}

.expanded-card-title {
  font-weight: 600;
  margin-bottom: 10px;
  color: rgba(0, 0, 0, 0.85);
  font-size: 14px;
  padding-bottom: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.expanded-card-title-blue {
  border-bottom: 2px solid #1890ff;
}

.expanded-card-title-green {
  border-bottom: 2px solid #52c41a;
}

.expanded-card-title-purple {
  border-bottom: 2px solid #722ed1;
}

.expanded-field-label {
  color: #8c8c8c;
  font-size: 12px;
}

.expanded-field-value {
  font-family: 'Consolas', monospace;
  font-size: 13px;
}

/* 新的信息字段样式 - 紧凑布局 */
.info-field {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 28px;
  padding: 4px 0;
}

.info-label {
  color: #8c8c8c;
  font-size: 12px;
  white-space: nowrap;
  flex-shrink: 0;
  min-width: 60px;
}

.info-value {
  font-size: 13px;
  color: rgba(0, 0, 0, 0.85);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.info-value-primary {
  font-weight: 600;
  font-family: 'Consolas', monospace;
  color: #1890ff;
}

.info-value-time {
  font-size: 12px;
  font-family: 'Consolas', monospace;
  color: #595959;
}

.info-value-message {
  font-size: 12px;
  color: #595959;
}

/* 暗色主题 - 信息字段 */
html.dark .info-label {
  color: rgba(255, 255, 255, 0.45);
}

html.dark .info-value {
  color: rgba(255, 255, 255, 0.85);
}

html.dark .info-value-primary {
  color: #40a9ff;
}

html.dark .info-value-time {
  color: rgba(255, 255, 255, 0.65);
}

html.dark .info-value-message {
  color: rgba(255, 255, 255, 0.65);
}

.expanded-log-container {
  flex: 1;
  overflow-y: auto;
  background: #f5f5f5;
  padding: 10px;
  border-radius: 4px;
  border: 1px solid #e8e8e8;
}

.expanded-log-item {
  margin-bottom: 4px;
  font-family: 'Consolas', 'Monaco', monospace;
  font-size: 12px;
  line-height: 1.6;
  color: #262626;
}

.expanded-empty-text {
  color: #999;
  text-align: center;
  padding: 60px 20px;
}

/* 暗色主题 - 展开行 */
html.dark .expanded-row-container {
  background: #141414;
}

html.dark .expanded-card {
  background: #1f1f1f;
  border-color: #303030;
}

html.dark .expanded-card-title {
  color: rgba(255, 255, 255, 0.85);
}

html.dark .expanded-field-label {
  color: rgba(255, 255, 255, 0.45);
}

html.dark .expanded-field-value {
  color: rgba(255, 255, 255, 0.85);
}

html.dark .expanded-log-container {
  background: #0a0a0a;
  border-color: #303030;
}

html.dark .expanded-log-item {
  color: rgba(255, 255, 255, 0.85);
}

html.dark .expanded-empty-text {
  color: rgba(255, 255, 255, 0.45);
}

/* 紧凑型多选框 - 防止选中多项时变高 */
.compact-select :deep(.ant-select-selector) {
  max-height: 24px !important;
  overflow: hidden !important;
  flex-wrap: nowrap !important;
}

.compact-select :deep(.ant-select-selection-overflow) {
  flex-wrap: nowrap !important;
  overflow: hidden !important;
}

.compact-select :deep(.ant-select-selection-overflow-item) {
  flex-shrink: 0 !important;
}

/* 可调整大小的容器 */
.resizable-container {
  position: relative;
  min-height: 200px;
  max-height: 800px;
}

.resizable-panels {
  display: flex;
  height: calc(100% - 4px);
  gap: 0;
}

.resizable-panel {
  min-width: 200px;
  overflow: hidden;
}

/* 分隔条样式 */
.resizer {
  background: #e8e8e8;
  position: relative;
  z-index: 1;
  transition: background 0.2s;
}

.resizer:hover {
  background: #1890ff;
}

.resizer-vertical {
  width: 4px;
  cursor: col-resize;
  flex-shrink: 0;
}

.resizer-horizontal {
  height: 4px;
  cursor: row-resize;
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
}

/* 暗色主题 - 分隔条 */
html.dark .resizer {
  background: #303030;
}

html.dark .resizer:hover {
  background: #1890ff;
}

/* 拖拽遮罩层 */
.drag-overlay {
  position: fixed;
  top: 32px;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(24, 144, 255, 0.1);
  backdrop-filter: blur(4px);
  z-index: 9999;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  animation: fadeIn 0.2s ease-in-out;
}

.drag-overlay-content {
  background: #fff;
  padding: 48px 64px;
  border-radius: 12px;
  border: 3px dashed #1890ff;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  text-align: center;
  animation: scaleIn 0.2s ease-in-out;
}

html.dark .drag-overlay {
  background: rgba(24, 144, 255, 0.15);
}

html.dark .drag-overlay-content {
  background: #1f1f1f;
  border-color: #1890ff;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

@keyframes fadeIn {
  from {
    opacity: 0;
  }
  to {
    opacity: 1;
  }
}

@keyframes scaleIn {
  from {
    transform: scale(0.9);
    opacity: 0;
  }
  to {
    transform: scale(1);
    opacity: 1;
  }
}

</style>
