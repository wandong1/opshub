<template>
  <div class="network-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Connection /></el-icon>
        </div>
        <div>
          <h2 class="page-title">网络管理</h2>
          <p class="page-subtitle">管理 Kubernetes 网络资源</p>
        </div>
      </div>
      <div class="header-actions">
        <el-select
          v-model="selectedClusterId"
          placeholder="选择集群"
          class="cluster-select"
          @change="handleClusterChange"
        >
          <template #prefix>
            <el-icon class="search-icon"><Platform /></el-icon>
          </template>
          <el-option
            v-for="cluster in clusterList"
            :key="cluster.id"
            :label="cluster.alias || cluster.name"
            :value="cluster.id"
          />
        </el-select>
        <el-button class="black-button" @click="loadCurrentResources">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 网络类型标签 -->
    <div class="network-types-bar">
      <div
        v-for="type in networkTypes"
        :key="type.value"
        :class="['type-tab', { active: activeTab === type.value }]"
        @click="handleTabChange(type.value)"
      >
        <el-icon class="type-icon">
          <component :is="type.icon" />
        </el-icon>
        <span class="type-label">{{ type.label }}</span>
        <span v-if="type.count !== undefined" class="type-count">({{ type.count }})</span>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="content-wrapper">
      <!-- Services -->
      <ServiceList
        v-show="activeTab === 'services' && selectedClusterId"
        ref="serviceListRef"
        :clusterId="selectedClusterId"
        :namespace="selectedNamespace"
        @edit="handleEditService"
        @yaml="handleEditServiceYAML"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('services', count)"
        @terminal="handleTerminal"
        @logs="handleLogs"
      />

      <!-- Ingress -->
      <IngressList
        v-show="activeTab === 'ingresses' && selectedClusterId"
        ref="ingressListRef"
        :clusterId="selectedClusterId"
        :namespace="selectedNamespace"
        @edit="handleEditIngress"
        @yaml="handleEditIngressYAML"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('ingresses', count)"
      />

      <!-- Network Policies -->
      <NetworkPolicyList
        v-show="activeTab === 'networkpolicies' && selectedClusterId"
        ref="networkPolicyListRef"
        :clusterId="selectedClusterId"
        :namespace="selectedNamespace"
        @edit="handleEditNetworkPolicy"
        @yaml="handleEditNetworkPolicyYAML"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('networkpolicies', count)"
      />

      <!-- Endpoints -->
      <EndpointsList
        v-show="activeTab === 'endpoints' && selectedClusterId"
        ref="endpointsListRef"
        :clusterId="selectedClusterId"
        :namespace="selectedNamespace"
        @refresh="loadCurrentResources"
        @count-update="(count) => updateCount('endpoints', count)"
      />
    </div>

    <!-- 终端对话框 -->
    <el-dialog
      v-model="terminalDialogVisible"
      :title="`终端 - Pod: ${terminalData.pod} | 容器: ${terminalData.container}`"
      width="90%"
      :close-on-click-modal="false"
      class="terminal-dialog"
      @close="handleCloseTerminal"
      @opened="handleTerminalDialogOpened"
    >
      <div class="terminal-container">
        <div v-if="!terminalConnected" class="terminal-loading-overlay">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>正在连接终端...</span>
        </div>
        <div class="terminal-wrapper" ref="terminalWrapper"></div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="terminalDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 日志对话框 -->
    <el-dialog
      v-model="logsDialogVisible"
      :title="`日志 - Pod: ${logsData.pod} | 容器: ${logsData.container}`"
      width="90%"
      :close-on-click-modal="false"
      class="logs-dialog"
      @opened="handleLogsDialogOpened"
    >
      <div class="logs-toolbar">
        <el-button size="small" @click="handleRefreshLogs" :loading="logsLoading">
          <el-icon><Refresh /></el-icon>
          刷新
        </el-button>
        <el-button size="small" @click="handleDownloadLogs">
          <el-icon><Download /></el-icon>
          下载
        </el-button>
        <el-button size="small" @click="logsAutoScroll = !logsAutoScroll" :type="logsAutoScroll ? 'primary' : 'default'">
          <el-icon><Bottom /></el-icon>
          {{ logsAutoScroll ? '自动滚动' : '停止滚动' }}
        </el-button>
        <el-select v-model="logsTailLines" size="small" style="width: 120px; margin-left: 10px;">
          <el-option label="最近100行" :value="100" />
          <el-option label="最近500行" :value="500" />
          <el-option label="最近1000行" :value="1000" />
          <el-option label="全部" :value="0" />
        </el-select>
      </div>
      <div class="logs-wrapper" ref="logsWrapper">
        <pre v-if="logsContent" class="logs-content">{{ logsContent }}</pre>
        <el-empty v-else-if="!logsLoading" description="暂无日志" />
        <div v-if="logsLoading" class="logs-loading">
          <el-icon class="is-loading"><Loading /></el-icon>
          <span>正在加载日志...</span>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="logsDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick, onUnmounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import {
  Platform,
  Refresh,
  Connection,
  Service,
  Link,
  Lock,
  Position,
  Loading,
  Download,
  Bottom
} from '@element-plus/icons-vue'
import { getClusterList, type Cluster } from '@/api/kubernetes'
import axios from 'axios'
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import { WebLinksAddon } from '@xterm/addon-web-links'
import '@xterm/xterm/css/xterm.css'
import ServiceList from './network-components/ServiceList.vue'
import IngressList from './network-components/IngressList.vue'
import NetworkPolicyList from './network-components/NetworkPolicyList.vue'
import EndpointsList from './network-components/EndpointsList.vue'

// 网络类型定义
interface NetworkType {
  label: string
  value: string
  icon: any
  count: number
}

const networkTypes = ref<NetworkType[]>([
  { label: 'Services', value: 'services', icon: Service, count: 0 },
  { label: 'Ingress', value: 'ingresses', icon: Link, count: 0 },
  { label: 'Network Policies', value: 'networkpolicies', icon: Lock, count: 0 },
  { label: 'Endpoints', value: 'endpoints', icon: Position, count: 0 },
])

const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const selectedNamespace = ref('')
const activeTab = ref('services')

// 子组件引用
const serviceListRef = ref()
const ingressListRef = ref()
const networkPolicyListRef = ref()
const endpointsListRef = ref()

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('network_selected_cluster_id')
      const savedNs = localStorage.getItem('network_selected_namespace')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      if (savedNs) {
        selectedNamespace.value = savedNs
      }
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('network_selected_cluster_id', selectedClusterId.value.toString())
  }
}

// Tab 切换
const handleTabChange = (tab: string) => {
  activeTab.value = tab
  localStorage.setItem('network_active_tab', tab)
}

// 更新资源数量
const updateCount = (type: string, count: number) => {
  const networkType = networkTypes.value.find(t => t.value === type)
  if (networkType) {
    networkType.count = count
  }
}

// 加载当前资源
const loadCurrentResources = () => {
  // 根据当前活动的 tab 刷新对应的资源
  switch (activeTab.value) {
    case 'services':
      serviceListRef.value?.loadData()
      break
    case 'ingresses':
      ingressListRef.value?.loadData()
      break
    case 'networkpolicies':
      networkPolicyListRef.value?.loadData()
      break
    case 'endpoints':
      endpointsListRef.value?.loadData()
      break
  }
}

// Service 操作
const handleEditService = (service: any) => {
  ElMessage.info('编辑 Service 功能开发中...')
}

const handleEditServiceYAML = (service: any) => {
  ElMessage.info('编辑 Service YAML 功能开发中...')
}

// Ingress 操作
const handleEditIngress = (ingress: any) => {
  ElMessage.info('编辑 Ingress 功能开发中...')
}

const handleEditIngressYAML = (ingress: any) => {
  ElMessage.info('编辑 Ingress YAML 功能开发中...')
}

// NetworkPolicy 操作
const handleEditNetworkPolicy = (policy: any) => {
  ElMessage.info('编辑 NetworkPolicy 功能开发中...')
}

const handleEditNetworkPolicyYAML = (policy: any) => {
  ElMessage.info('编辑 NetworkPolicy YAML 功能开发中...')
}

// 终端相关
const terminalDialogVisible = ref(false)
const terminalConnected = ref(false)
const terminalData = ref({
  pod: '',
  container: '',
  namespace: ''
})
const terminalWrapper = ref<HTMLDivElement | null>(null)
let terminal: Terminal | null = null
let terminalWebSocket: WebSocket | null = null

// 日志相关
const logsDialogVisible = ref(false)
const logsContent = ref('')
const logsLoading = ref(false)
const logsAutoScroll = ref(true)
const logsTailLines = ref(500)
const logsData = ref({
  pod: '',
  container: '',
  namespace: ''
})
const logsWrapper = ref<HTMLDivElement | null>(null)
let logsRefreshTimer: number | null = null

// 处理终端请求
const handleTerminal = async (data: { namespace: string; name: string }) => {
  // 获取Pod详情以获取容器名称
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/pods/${data.namespace}/${data.name}`, {
      params: { clusterId: selectedClusterId.value },
      headers: { Authorization: `Bearer ${token}` }
    })
    const pod = response.data.data
    const containers = pod.spec?.containers || []
    if (containers.length === 0) {
      ElMessage.error('Pod没有容器')
      return
    }
    // 默认使用第一个容器
    terminalData.value = {
      pod: data.name,
      container: containers[0].name,
      namespace: data.namespace
    }
    terminalConnected.value = false
    terminalDialogVisible.value = true
  } catch (error) {
    console.error('获取Pod详情失败:', error)
    ElMessage.error('获取Pod详情失败')
  }
}

// 处理日志请求
const handleLogs = async (data: { namespace: string; name: string }) => {
  // 获取Pod详情以获取容器名称
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/pods/${data.namespace}/${data.name}`, {
      params: { clusterId: selectedClusterId.value },
      headers: { Authorization: `Bearer ${token}` }
    })
    const pod = response.data.data
    const containers = pod.spec?.containers || []
    if (containers.length === 0) {
      ElMessage.error('Pod没有容器')
      return
    }
    // 默认使用第一个容器
    logsData.value = {
      pod: data.name,
      container: containers[0].name,
      namespace: data.namespace
    }
    logsContent.value = ''
    logsDialogVisible.value = true
  } catch (error) {
    console.error('获取Pod详情失败:', error)
    ElMessage.error('获取Pod详情失败')
  }
}

// 终端对话框打开后的回调
const handleTerminalDialogOpened = async () => {
  await nextTick()
  await initTerminal()
}

// 初始化终端
const initTerminal = async () => {
  console.log('🔍 initTerminal 被调用')
  console.log('🔍 terminalWrapper.value:', terminalWrapper.value)

  // 等待 DOM 元素准备好，最多重试 10 次
  let retries = 0
  while (!terminalWrapper.value && retries < 10) {
    console.log(`⏳ 等待 terminalWrapper 准备好... (${retries + 1}/10)`)
    await new Promise(resolve => setTimeout(resolve, 100))
    retries++
  }

  if (!terminalWrapper.value) {
    console.error('❌ terminalWrapper 仍然为 null，无法初始化终端')
    return
  }

  console.log('✅ terminalWrapper 已准备好，开始初始化终端')

  // 清空容器
  terminalWrapper.value.innerHTML = ''

  // 创建终端实例
  terminal = new Terminal({
    cursorBlink: true,
    fontSize: 14,
    fontFamily: 'Menlo, Monaco, "Courier New", monospace',
    theme: {
      background: '#1e1e1e',
      foreground: '#d4d4d4',
      cursor: '#d4d4d4',
      black: '#000000',
      red: '#cd3131',
      green: '#0dbc79',
      yellow: '#e5e510',
      blue: '#2472c8',
      magenta: '#bc3fbc',
      cyan: '#11a8cd',
      white: '#e5e5e5',
      brightBlack: '#666666',
      brightRed: '#f14c4c',
      brightGreen: '#23d18b',
      brightYellow: '#f5f543',
      brightBlue: '#3b8eea',
      brightMagenta: '#d670d6',
      brightCyan: '#29b8db',
      brightWhite: '#ffffff'
    }
  })

  // 加载插件
  const fitAddon = new FitAddon()
  const webLinksAddon = new WebLinksAddon()
  terminal.loadAddon(fitAddon)
  terminal.loadAddon(webLinksAddon)

  // 打开终端
  terminal.open(terminalWrapper.value)
  fitAddon.fit()

  // 欢迎信息
  terminal.writeln('\x1b[1;32m正在连接到容器...\x1b[0m')

  // 获取token
  const token = localStorage.getItem('token')
  const clusterId = selectedClusterId.value

  console.log('🔍 终端连接参数:', {
    clusterId,
    namespace: terminalData.value.namespace,
    pod: terminalData.value.pod,
    container: terminalData.value.container
  })

  // 构建WebSocket URL - 在开发环境直接连接后端，生产环境使用当前域名
  const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const host = window.location.hostname
  // 开发环境直接连接9876端口，生产环境使用当前端口
  const isDev = import.meta.env.DEV
  const port = isDev ? '9876' : (window.location.port || (window.location.protocol === 'https:' ? '443' : '9876'))
  const wsUrl = `${protocol}//${host}:${port}/api/v1/plugins/kubernetes/shell/pods?` +
    `clusterId=${clusterId}&` +
    `namespace=${terminalData.value.namespace}&` +
    `podName=${terminalData.value.pod}&` +
    `container=${terminalData.value.container}&` +
    `token=${token}`

  console.log('🔍 WebSocket URL:', wsUrl)

  try {
    // 建立WebSocket连接
    terminalWebSocket = new WebSocket(wsUrl)

    terminalWebSocket.onopen = () => {
      console.log('✅ WebSocket 已连接')
      terminalConnected.value = true
      terminal.clear()
      terminal.writeln('\x1b[1;32m✓ 已连接到容器 ' + terminalData.value.container + '\x1b[0m')
      terminal.writeln('')
    }

    terminalWebSocket.onmessage = (event) => {
      terminal.write(event.data)
    }

    terminalWebSocket.onerror = (error) => {
      console.error('❌ WebSocket错误:', error)
      terminal.writeln('\x1b[1;31m✗ 连接错误\x1b[0m')
      terminal.writeln('请检查:')
      terminal.writeln('1. 集群连接是否正常')
      terminal.writeln('2. Pod是否正在运行')
      terminal.writeln('3. 浏览器控制台是否有错误信息')
    }

    terminalWebSocket.onclose = (event) => {
      console.log('🔌 WebSocket 已关闭:', event.code, event.reason)
      terminalConnected.value = false
      // 安全检查：terminal 可能已经被销毁
      if (terminal) {
        try {
          terminal.writeln('\x1b[1;33m连接已关闭\x1b[0m')
        } catch (e) {
          console.warn('写入终端消息失败（可能已销毁）:', e)
        }
      }
    }

    // 处理用户输入
    terminal.onData((data: string) => {
      if (terminalWebSocket && terminalWebSocket.readyState === WebSocket.OPEN) {
        terminalWebSocket.send(data)
      }
    })

    // 处理窗口大小变化
    terminal.onResize(({ cols, rows }) => {
      if (terminalWebSocket && terminalWebSocket.readyState === WebSocket.OPEN) {
        terminalWebSocket.send(JSON.stringify({ type: 'resize', cols, rows }))
      }
    })

  } catch (error: any) {
    console.error('❌ 创建终端失败:', error)
    terminal.writeln('\x1b[1;31m✗ 连接失败: ' + error.message + '\x1b[0m')
  }
}

// 关闭终端
const handleCloseTerminal = () => {
  if (terminalWebSocket) {
    terminalWebSocket.close()
    terminalWebSocket = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  terminalConnected.value = false
}

// 日志对话框打开后的回调
const handleLogsDialogOpened = async () => {
  await handleLoadLogs()

  // 启动自动刷新定时器（每3秒刷新一次）
  if (logsRefreshTimer) clearInterval(logsRefreshTimer)
  logsRefreshTimer = window.setInterval(() => {
    handleLoadLogs()
  }, 3000)
}

// 加载日志
const handleLoadLogs = async () => {
  logsLoading.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const { pod, container, namespace } = logsData.value

    const response = await axios.get('/api/v1/plugins/kubernetes/resources/pods/logs', {
      params: {
        clusterId,
        namespace,
        podName: pod,
        container,
        tailLines: logsTailLines.value
      },
      headers: { Authorization: `Bearer ${token}` }
    })

    logsContent.value = response.data.data?.logs || ''

    // 自动滚动到底部 - 使用 setTimeout 确保 DOM 完全渲染
    if (logsAutoScroll.value) {
      setTimeout(() => {
        if (logsWrapper.value) {
          console.log('滚动到底部，scrollHeight:', logsWrapper.value.scrollHeight)
          logsWrapper.value.scrollTop = logsWrapper.value.scrollHeight
        } else {
          console.log('logsWrapper.value 为 null')
        }
      }, 100)
    }
  } catch (error: any) {
    console.error('获取日志失败:', error)
    ElMessage.error(`获取日志失败: ${error.response?.data?.message || error.message}`)
  } finally {
    logsLoading.value = false
  }
}

// 刷新日志
const handleRefreshLogs = () => {
  handleLoadLogs()
}

// 下载日志
const handleDownloadLogs = () => {
  const blob = new Blob([logsContent.value], { type: 'text/plain' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${logsData.value.pod}-${logsData.value.container}-${Date.now()}.log`
  a.click()
  URL.revokeObjectURL(url)
}

// 监听日志对话框关闭
watch(logsDialogVisible, (newVal) => {
  if (!newVal) {
    if (logsRefreshTimer) {
      clearInterval(logsRefreshTimer)
      logsRefreshTimer = null
    }
  }
})

onMounted(async () => {
  await loadClusters()
  const savedTab = localStorage.getItem('network_active_tab')
  if (savedTab) {
    activeTab.value = savedTab
  }
})

onUnmounted(() => {
  // 清理终端资源
  if (terminalWebSocket) {
    terminalWebSocket.close()
    terminalWebSocket = null
  }
  if (terminal) {
    terminal.dispose()
    terminal = null
  }
  // 清理日志定时器
  if (logsRefreshTimer) {
    clearInterval(logsRefreshTimer)
    logsRefreshTimer = null
  }
})
</script>

<style scoped>
.network-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 16px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 48px;
  height: 48px;
  background: linear-gradient(135deg, #000 0%, #1a1a1a 100%);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #d4af37;
  font-size: 22px;
  flex-shrink: 0;
  border: 1px solid #d4af37;
}

.page-title-icon .el-icon {
  font-size: 22px;
  color: #d4af37;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: #303133;
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: #909399;
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

.cluster-select {
  width: 280px;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

.search-icon {
  color: #d4af37;
}

/* 网络类型标签栏 */
.network-types-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  flex-wrap: wrap;
}

.type-tab {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  background: #1a1a1a;
  color: #fff;
  border-radius: 8px;
  cursor: pointer;
  transition: all 0.3s ease;
  font-size: 14px;
  font-weight: 500;
}

.type-tab:hover {
  background: #2a2a2a;
}

.type-tab.active {
  background: #d4af37;
  color: #000;
  border: 1px solid #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
}

.type-icon {
  font-size: 16px;
}

.type-label {
  white-space: nowrap;
}

.type-count {
  font-size: 12px;
  opacity: 0.8;
  margin-left: 2px;
}

/* 内容区域 */
.content-wrapper {
  background: transparent;
}

.cluster-select :deep(.el-input__wrapper) {
  border-radius: 8px;
}

/* 终端对话框样式 */
.terminal-dialog :deep(.el-dialog__body) {
  padding: 0;
  height: 60vh;
}

.terminal-container {
  position: relative;
  height: 100%;
  background: #1a1a1a;
}

.terminal-loading-overlay {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: rgba(26, 26, 26, 0.9);
  color: #fff;
  z-index: 10;
}

.terminal-wrapper {
  height: 100%;
  padding: 10px;
}

/* 日志对话框样式 */
.logs-dialog :deep(.el-dialog__body) {
  padding: 0;
  display: flex;
  flex-direction: column;
  height: 60vh;
}

.logs-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-bottom: 1px solid #e8e8e8;
  background: #fafbfc;
}

.logs-wrapper {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
  background: #1a1a1a;
}

.logs-content {
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
  line-height: 1.5;
  color: #ffffff;
  white-space: pre-wrap;
  word-break: break-all;
}

.logs-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  height: 100%;
  color: #909399;
}
</style>
