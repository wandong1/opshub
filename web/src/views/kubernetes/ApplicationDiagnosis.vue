<template>
  <div class="app-diagnosis-container">
    <!-- 页面标题 -->
    <a-card class="page-header-card">
      <div class="page-header">
        <div class="page-title-group">
          <div class="page-title-icon">
            <icon-thunderbolt />
          </div>
          <div>
            <h2 class="page-title">应用诊断</h2>
            <p class="page-subtitle">基于 Arthas 的 Java 应用诊断工具</p>
          </div>
        </div>
      </div>
    </a-card>

    <!-- 选择器栏 -->
    <a-card class="search-card">
      <div class="selector-bar">
      <div class="selector-inputs">
        <div class="selector-item">
          <span class="selector-label">集群</span>
          <a-select
            v-model="selectedCluster"
            placeholder="请选择集群"
            @change="handleClusterChange"
            class="selector-select"
          >
            <a-option
              v-for="cluster in clusters"
              :key="cluster.id"
              :label="cluster.name"
              :value="cluster.id"
            />
          </a-select>
        </div>

        <div class="selector-item">
          <span class="selector-label">命名空间</span>
          <a-select
            v-model="selectedNamespace"
            placeholder="请选择命名空间"
            @change="handleNamespaceChange"
            :disabled="!selectedCluster"
            class="selector-select"
          >
            <a-option
              v-for="ns in namespaces"
              :key="ns.name"
              :label="ns.name"
              :value="ns.name"
            />
          </a-select>
        </div>

        <div class="selector-item">
          <span class="selector-label">Pod</span>
          <a-select
            v-model="selectedPod"
            placeholder="请选择Pod"
            @change="handlePodChange"
            :disabled="!selectedNamespace"
            class="selector-select"
          >
            <a-option
              v-for="pod in pods"
              :key="pod.name"
              :label="pod.name"
              :value="pod.name"
            />
          </a-select>
        </div>

        <div class="selector-item">
          <span class="selector-label">容器</span>
          <a-select
            v-model="selectedContainer"
            placeholder="请选择容器"
            @change="handleContainerChange"
            :disabled="!selectedPod"
            class="selector-select"
          >
            <a-option
              v-for="container in containers"
              :key="container"
              :label="container"
              :value="container"
            />
          </a-select>
        </div>

        <div class="selector-item">
          <span class="selector-label">进程</span>
          <a-select
            v-model="selectedProcess"
            placeholder="请选择进程"
            :disabled="!selectedContainer"
            :loading="loadingProcesses"
            class="selector-select process-select"
            @change="handleProcessChange"
          >
            <a-option
              v-for="proc in processes"
              :key="proc.pid"
              :label="`${proc.pid} - ${proc.mainClass}`"
              :value="proc.pid"
            />
          </a-select>
        </div>
      </div>

      <div class="selector-actions">
        <a-button
          type="primary"
          :icon="attached ? Link : Download"
          @click="handleAttach"
          :disabled="!selectedProcess"
          :loading="attaching"
          class="attach-btn"
        >
          {{ attached ? '已连接' : '连接' }}
        </a-button>
      </div>
      </div>
    </a-card>

    <!-- Tab 内容区 -->
    <a-card class="table-card">
      <a-tabs v-model:active-key="activeTab" class="diagnosis-tabs" @tab-change="handleTabChange">
        <a-tab-pane title="控制面板" key="dashboard">
          <DashboardPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>

        <a-tab-pane title="线程清单" key="threads">
          <ThreadListPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>

        <a-tab-pane title="JVM信息" key="jvm">
          <JvmInfoPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>

        <a-tab-pane title="系统信息" key="sysinfo">
          <SystemInfoPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>

        <a-tab-pane title="线程堆栈" key="stack">
          <ThreadStackPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>

        <a-tab-pane title="火焰图" key="flame">
          <FlameGraphPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>

        <a-tab-pane title="方法追踪" key="trace">
          <MethodTracePanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>

        <a-tab-pane title="方法监测" key="watch">
          <MethodWatchPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>

        <a-tab-pane title="方法监控" key="monitor">
          <MethodMonitorPanel
            :cluster-id="selectedCluster"
            :namespace="selectedNamespace"
            :pod="selectedPod"
            :container="selectedContainer"
            :process-id="selectedProcess"
            :attached="attached"
          />
        </a-tab-pane>
      </a-tabs>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import { getClusterList, getNamespaces, getPods, getPodDetail } from '@/api/kubernetes'
import { listJavaProcesses, checkArthasInstalled, installArthas, type JavaProcess } from '@/api/arthas'

// 导入子组件
import DashboardPanel from './diagnosis-components/DashboardPanel.vue'
import ThreadListPanel from './diagnosis-components/ThreadListPanel.vue'
import JvmInfoPanel from './diagnosis-components/JvmInfoPanel.vue'
import SystemInfoPanel from './diagnosis-components/SystemInfoPanel.vue'
import ThreadStackPanel from './diagnosis-components/ThreadStackPanel.vue'
import FlameGraphPanel from './diagnosis-components/FlameGraphPanel.vue'
import MethodTracePanel from './diagnosis-components/MethodTracePanel.vue'
import MethodWatchPanel from './diagnosis-components/MethodWatchPanel.vue'
import MethodMonitorPanel from './diagnosis-components/MethodMonitorPanel.vue'

const route = useRoute()
const router = useRouter()

// 状态存储的 key
const STORAGE_KEY = 'arthas_diagnosis_state'

// 选择器数据
const clusters = ref<any[]>([])
const namespaces = ref<any[]>([])
const pods = ref<any[]>([])
const containers = ref<string[]>([])
const processes = ref<JavaProcess[]>([])

// 选中的值
const selectedCluster = ref<number | null>(null)
const selectedNamespace = ref<string>('')
const selectedPod = ref<string>('')
const selectedContainer = ref<string>('')
const selectedProcess = ref<string>('')

// 状态
const activeTab = ref('dashboard')
const attaching = ref(false)
const attached = ref(false)
const loadingProcesses = ref(false)

// 保存状态到 localStorage
const saveState = () => {
  const state = {
    clusterId: selectedCluster.value,
    namespace: selectedNamespace.value,
    pod: selectedPod.value,
    container: selectedContainer.value,
    processId: selectedProcess.value,
    activeTab: activeTab.value,
    attached: attached.value
  }
  localStorage.setItem(STORAGE_KEY, JSON.stringify(state))
}

// 从 localStorage 恢复状态
const restoreState = async () => {
  const savedState = localStorage.getItem(STORAGE_KEY)
  if (!savedState) return

  try {
    const state = JSON.parse(savedState)

    // 恢复 Tab 状态
    if (state.activeTab) {
      activeTab.value = state.activeTab
    }

    // 恢复选择状态（需要按顺序恢复，因为有依赖关系）
    if (state.clusterId) {
      selectedCluster.value = state.clusterId
      await loadNamespaces()

      if (state.namespace) {
        selectedNamespace.value = state.namespace
        await loadPods()

        if (state.pod) {
          selectedPod.value = state.pod
          await loadContainers()

          if (state.container) {
            selectedContainer.value = state.container
            await loadProcesses()

            if (state.processId) {
              selectedProcess.value = state.processId
              attached.value = state.attached || false
            }
          }
        }
      }
    }
  } catch (e) {
  }
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const res = await getClusterList()
    clusters.value = res || []
  } catch (error) {
  }
}

// 加载命名空间
const loadNamespaces = async () => {
  if (!selectedCluster.value) return
  try {
    const res = await getNamespaces(selectedCluster.value)
    namespaces.value = res || []
  } catch (error) {
  }
}

// 加载 Pods
const loadPods = async () => {
  if (!selectedCluster.value || !selectedNamespace.value) return
  try {
    const res = await getPods(selectedCluster.value, selectedNamespace.value)
    pods.value = res || []
  } catch (error) {
  }
}

// 加载容器
const loadContainers = async () => {
  if (!selectedCluster.value || !selectedNamespace.value || !selectedPod.value) return
  try {
    const res = await getPodDetail(selectedCluster.value, selectedNamespace.value, selectedPod.value)
    const containerList = res?.spec?.containers?.map((c: any) => c.name) || []
    const initContainers = res?.spec?.initContainers?.map((c: any) => c.name) || []
    containers.value = [...containerList, ...initContainers]
  } catch (error) {
  }
}

// 加载进程
const loadProcesses = async () => {
  if (!selectedCluster.value || !selectedNamespace.value || !selectedPod.value || !selectedContainer.value) return

  loadingProcesses.value = true
  try {
    const res = await listJavaProcesses({
      clusterId: selectedCluster.value,
      namespace: selectedNamespace.value,
      pod: selectedPod.value,
      container: selectedContainer.value
    })
    processes.value = Array.isArray(res) ? res : (res?.data || [])
    if (processes.value.length === 0) {
      Message.warning('未检测到Java进程，请确保容器中有运行的Java应用')
    }
  } catch (error: any) {
    processes.value = []
    if (error.message && !error.message.includes('exit code')) {
      Message.error('获取Java进程失败: ' + (error.message || '未知错误'))
    } else {
      Message.warning('该容器未检测到Java环境')
    }
  } finally {
    loadingProcesses.value = false
  }
}

// 集群变更
const handleClusterChange = async () => {
  selectedNamespace.value = ''
  selectedPod.value = ''
  selectedContainer.value = ''
  selectedProcess.value = ''
  namespaces.value = []
  pods.value = []
  containers.value = []
  processes.value = []
  attached.value = false

  await loadNamespaces()
  saveState()
}

// 命名空间变更
const handleNamespaceChange = async () => {
  selectedPod.value = ''
  selectedContainer.value = ''
  selectedProcess.value = ''
  pods.value = []
  containers.value = []
  processes.value = []
  attached.value = false

  await loadPods()
  saveState()
}

// Pod变更
const handlePodChange = async () => {
  selectedContainer.value = ''
  selectedProcess.value = ''
  containers.value = []
  processes.value = []
  attached.value = false

  await loadContainers()
  saveState()
}

// 容器变更
const handleContainerChange = async () => {
  selectedProcess.value = ''
  processes.value = []
  attached.value = false

  await loadProcesses()
  saveState()
}

// 进程变更
const handleProcessChange = () => {
  attached.value = false
  saveState()
}

// Tab 变更
const handleTabChange = (tab: string) => {
  saveState()
}

// 安装/连接 Arthas
const handleAttach = async () => {
  if (!selectedProcess.value) {
    Message.warning('请先选择要诊断的进程')
    return
  }

  if (!selectedCluster.value || !selectedNamespace.value || !selectedPod.value || !selectedContainer.value) {
    Message.warning('请先选择集群、命名空间、Pod和容器')
    return
  }

  attaching.value = true
  try {
    // 先检查Arthas是否已安装
    const checkRes = await checkArthasInstalled({
      clusterId: selectedCluster.value,
      namespace: selectedNamespace.value,
      pod: selectedPod.value,
      container: selectedContainer.value
    })

    const checkData = checkRes?.hasJava !== undefined ? checkRes : checkRes?.data

    if (!checkData?.hasJava) {
      Message.error('容器中未检测到Java环境，无法使用Arthas诊断')
      return
    }

    if (!checkData?.hasArthas) {
      Message.info('正在安装Arthas...')
      await installArthas({
        clusterId: selectedCluster.value,
        namespace: selectedNamespace.value,
        pod: selectedPod.value,
        container: selectedContainer.value
      })
    }

    attached.value = true
    saveState()
    Message.success('连接成功')
  } catch (error: any) {
    Message.error('连接失败: ' + (error.message || '未知错误'))
  } finally {
    attaching.value = false
  }
}

onMounted(async () => {
  await loadClusters()
  await restoreState()
})
</script>

<style scoped>
.app-diagnosis-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部卡片 */
.page-header-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  background: var(--ops-primary, #165dff);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  line-height: 1.4;
}

/* 搜索卡片 */
.search-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

/* 选择器栏 */
.selector-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
}

.selector-inputs {
  display: flex;
  gap: 16px;
  flex: 1;
  flex-wrap: wrap;
  align-items: center;
}

.selector-item {
  display: flex;
  align-items: center;
  gap: 8px;
}
.selector-label {
  font-size: 13px;
  color: var(--ops-text-secondary, #4e5969);
  white-space: nowrap;
  font-weight: 500;
}

.selector-select {
  width: 150px;
}

.process-select {
  width: 200px;
}

.selector-actions {
  display: flex;
  gap: 10px;
}

/* 表格卡片 */
.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

/* Tab 样式 */
.diagnosis-tabs {
  padding: 0;
}

:deep(.diagnosis-tabs .arco-tabs__header) {
  margin: 0;
  padding: 0;
  background: var(--ops-content-bg, #f7f8fa);
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
}

:deep(.diagnosis-tabs .arco-tabs__nav-wrap::after) {
  display: none;
}

:deep(.diagnosis-tabs .arco-tabs__item) {
  padding: 0 20px;
  height: 48px;
  line-height: 48px;
  font-size: 13px;
  color: var(--ops-text-secondary, #4e5969);
  font-weight: 500;
  transition: all 0.2s ease;
}

:deep(.diagnosis-tabs .arco-tabs__item:hover) {
  color: var(--ops-primary, #165dff);
}

:deep(.diagnosis-tabs .arco-tabs__content) {
  padding: 20px;
}

/* 响应式布局 */
@media (max-width: 1200px) {
  .selector-inputs {
    gap: 12px;
  }

  .selector-select {
    width: 130px;
  }

  .process-select {
    width: 180px;
  }
}

@media (max-width: 992px) {
  .selector-bar {
    flex-direction: column;
    align-items: flex-start;
  }

  .selector-inputs {
    width: 100%;
  }

  .selector-actions {
    width: 100%;
    justify-content: flex-end;
  }
}

@media (max-width: 768px) {
  .selector-item {
    flex-direction: column;
    align-items: flex-start;
    gap: 4px;
    width: calc(50% - 8px);
  }

  .selector-select,
  .process-select {
    width: 100%;
  }

  :deep(.diagnosis-tabs .arco-tabs__item) {
    padding: 0 12px;
    font-size: 12px;
  }
}
</style>
