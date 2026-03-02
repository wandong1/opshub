<template>
  <div class="cluster-detail-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <div class="header-top">
          <a-button class="back-btn" @click="handleBack" :icon="ArrowLeft">返回列表</a-button>
        </div>
        <div class="cluster-name-section">
          <h1 class="cluster-title">
            <icon-apps />
            {{ clusterInfo?.alias || clusterInfo?.name }}
          </h1>
          <a-tag :color="getStatusType(clusterInfo?.status || 1)" size="large" class="status-tag">
            {{ getStatusText(clusterInfo?.status || 1) }}
          </a-tag>
        </div>
        <div class="cluster-meta">
          <span class="meta-item">
            <icon-link />
            {{ clusterInfo?.apiEndpoint }}
          </span>
          <span class="meta-item">
            <icon-info-circle />
            {{ clusterInfo?.version }}
          </span>
          <span class="meta-item" v-if="clusterInfo?.provider">
            <icon-apps />
            {{ getProviderText(clusterInfo.provider) }}
          </span>
        </div>
      </div>
    </div>

    <!-- 快速统计卡片 -->
    <div class="quick-stats">
      <div class="stat-card" v-for="(stat, index) in quickStats" :key="index" :style="{ '--delay': index * 0.1 + 's' }">
        <div class="stat-icon-wrapper" :style="{ background: stat.color }">
          <span style="font-size: 32px; color: stat.iconColor">
            <component :is="stat.icon" />
          </span>
        </div>
        <div class="stat-info">
          <div class="stat-value">{{ stat.value }}</div>
          <div class="stat-label">{{ stat.label }}</div>
        </div>
        <div class="stat-trend" v-if="stat.trend">
          <icon-rise />
        </div>
      </div>
    </div>

    <!-- 主内容区 -->
    <div class="main-content">
      <!-- 左侧列 -->
      <div class="left-column">
        <!-- 资源使用率 -->
        <a-card shadow="hover" class="modern-card">
          <div class="resource-usage">
            <div class="usage-item">
              <div class="usage-header">
                <div class="usage-label">
                  <icon-thunderbolt />
                  <span>CPU 使用率</span>
                </div>
                <span class="usage-value">{{ Math.round(clusterStats.cpuUsage) }}%</span>
              </div>
              <div class="progress-wrapper">
                <a-progress
                  :percentage="Math.round(clusterStats.cpuUsage)"
                  :color="getProgressColor(clusterStats.cpuUsage)"
                  :show-text="false"
                  :stroke-width="8"
                />
              </div>
              <div class="usage-detail">
                <span class="detail-text">已使用: {{ clusterStats.cpuUsed.toFixed(2) }} 核</span>
                <span class="detail-text">可分配: {{ clusterStats.cpuAllocatable.toFixed(2) }} 核</span>
              </div>
            </div>

            <div class="usage-item">
              <div class="usage-header">
                <div class="usage-label">
                  <icon-common />
                  <span>内存使用率</span>
                </div>
                <span class="usage-value">{{ Math.round(clusterStats.memoryUsage) }}%</span>
              </div>
              <div class="progress-wrapper">
                <a-progress
                  :percentage="Math.round(clusterStats.memoryUsage)"
                  :color="getProgressColor(clusterStats.memoryUsage)"
                  :show-text="false"
                  :stroke-width="8"
                />
              </div>
              <div class="usage-detail">
                <span class="detail-text">已使用: {{ formatBytes(clusterStats.memoryUsed) }}</span>
                <span class="detail-text">可分配: {{ formatBytes(clusterStats.memoryAllocatable) }}</span>
              </div>
            </div>
          </div>
        </a-card>

        <!-- 网络配置 -->
        <a-card shadow="hover" class="modern-card">
          <div class="network-config">
            <div class="config-grid">
              <div class="config-item">
                <div class="config-label">API Server</div>
                <a-tag type="primary" size="large">{{ networkInfo.apiServerAddress || '-' }}</a-tag>
              </div>
              <div class="config-item">
                <div class="config-label">Service CIDR</div>
                <a-tag color="gray" size="large">{{ networkInfo.serviceCIDR || '-' }}</a-tag>
              </div>
              <div class="config-item">
                <div class="config-label">Pod CIDR</div>
                <a-tag color="green" size="large">{{ networkInfo.podCIDR || '-' }}</a-tag>
              </div>
              <div class="config-item">
                <div class="config-label">网络插件</div>
                <a-tag color="orangered" size="large">{{ networkInfo.networkPlugin || '-' }}</a-tag>
              </div>
              <div class="config-item">
                <div class="config-label">Proxy 模式</div>
                <a-tag type="primary" size="large">{{ networkInfo.proxyMode || '-' }}</a-tag>
              </div>
              <div class="config-item">
                <div class="config-label">DNS 服务</div>
                <a-tag color="green" size="large">{{ networkInfo.dnsService || '-' }}</a-tag>
              </div>
            </div>
          </div>
        </a-card>

        <!-- 集群信息 -->
        <a-card shadow="hover" class="modern-card">
          <div class="cluster-info-grid">
            <div class="info-row">
              <span class="info-label">集群名称</span>
              <span class="info-value">{{ clusterInfo?.name }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">别名</span>
              <span class="info-value">{{ clusterInfo?.alias || '-' }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">服务商</span>
              <span class="info-value">{{ clusterInfo?.provider ? getProviderText(clusterInfo.provider) : '-' }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">区域</span>
              <span class="info-value">{{ clusterInfo?.region || '-' }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">创建时间</span>
              <span class="info-value">{{ clusterInfo?.createdAt }}</span>
            </div>
            <div class="info-row">
              <span class="info-label">更新时间</span>
              <span class="info-value">{{ clusterInfo?.updatedAt }}</span>
            </div>
            <div class="info-row full-width">
              <span class="info-label">备注</span>
              <span class="info-value">{{ clusterInfo?.description || '-' }}</span>
            </div>
          </div>
        </a-card>
      </div>

      <!-- 右侧列 -->
      <div class="right-column">
        <!-- 组件信息 -->
        <a-card shadow="hover" class="modern-card">

          <!-- 运行时环境 -->
          <div class="component-section">
            <div class="section-header">
              <icon-desktop />
              <span>运行时环境</span>
            </div>
            <div class="runtime-cards">
              <div class="runtime-card">
                <div class="runtime-label">容器运行时</div>
                <div class="runtime-value">{{ componentInfo.runtime.containerRuntime || '-' }}</div>
              </div>
              <div class="runtime-card">
                <div class="runtime-label">Kubelet 版本</div>
                <div class="runtime-value">{{ componentInfo.runtime.version || '-' }}</div>
              </div>
            </div>
          </div>

          <!-- 控制平面组件 -->
          <div class="component-section" v-if="componentInfo.components.length > 0">
            <div class="section-header">
              <icon-settings />
              <span>控制平面组件</span>
            </div>
            <div class="component-list">
              <div
                v-for="component in componentInfo.components"
                :key="component.name"
                class="component-item"
              >
                <div class="component-main">
                  <icon-check-circle />
                  <div class="component-info">
                    <div class="component-name">{{ component.name }}</div>
                    <a-tag size="small" color="gray">{{ component.version }}</a-tag>
                  </div>
                </div>
                <a-tag
                  :type="component.status === 'Running' ? 'success' : 'danger'"
                  size="small"
                >
                  {{ component.status }}
                </a-tag>
              </div>
            </div>
          </div>

          <!-- 存储类 -->
          <div class="component-section" v-if="componentInfo.storage.length > 0">
            <div class="section-header">
              <icon-folder />
              <span>存储类</span>
            </div>
            <div class="storage-list">
              <div
                v-for="storage in componentInfo.storage"
                :key="storage.name"
                class="storage-item"
              >
                <div class="storage-main">
                  <icon-folder />
                  <div class="storage-info">
                    <div class="storage-name">{{ storage.name }}</div>
                    <div class="storage-provisioner">{{ storage.provisioner }}</div>
                  </div>
                </div>
                <a-tag
                  :type="storage.reclaimPolicy === 'Delete' ? 'danger' : 'warning'"
                  size="small"
                >
                  {{ storage.reclaimPolicy }}
                </a-tag>
              </div>
            </div>
          </div>
        </a-card>
      </div>
    </div>

    <!-- 节点信息 -->
    <a-card shadow="hover" class="modern-card full-width-card">

      <!-- 搜索框 -->
      <div class="node-search-bar">
        <a-input
          v-model="nodeSearchKeyword"
          placeholder="搜索节点名称、IP地址..."
          clearable
          @clear="handleNodeSearch"
          @keyup.enter="handleNodeSearch"
          class="search-input"
        >
          <template #prefix>
            <icon-search />
          </template>
        </a-input>
        <a-button class="search-button" @click="handleNodeSearch">搜索</a-button>
      </div>

      <a-table
        :data="paginatedNodeList"
        stripe
        style="width: 100%"
        v-loading="nodesLoading"
       :columns="tableColumns2">
          <template #name="{ record }">
            <span class="node-name-link">{{ record.name }}</span>
          </template>
          <template #role="{ record }">
            <a-tag :type="getNodeRoleType(record.roles)" size="small">
              {{ record.roles || 'Worker' }}
            </a-tag>
          </template>
          <template #status="{ record }">
            <a-tag :type="record.status === 'Ready' ? 'success' : 'danger'" size="small">
              {{ record.status }}
            </a-tag>
          </template>
          <template #externalIP="{ record }">
            {{ record.externalIP || '无' }}
          </template>
        </a-table>

      <!-- 分页 -->
      <div class="node-pagination-wrapper">
        <a-pagination
          v-model:current="nodeCurrentPage"
          v-model:page-size="nodePageSize"
          :page-size-options="[10, 20, 50, 100]"
          :total="filteredNodeList.length"
          layout="total, sizes, prev, pager, next"
          @current-change="handleNodePageChange"
          @size-change="handleNodeSizeChange"
        />
      </div>
    </a-card>

    <!-- 最近事件 -->
    <a-card shadow="hover" class="modern-card full-width-card">
      <a-table
        :data="eventList"
        stripe
        style="width: 100%"
        v-loading="eventsLoading"
        :empty-text="'No Data'"
       :columns="tableColumns">
          <template #type="{ record }">
            <a-tag :type="record.type === 'Normal' ? 'success' : 'warning'" size="small">
              {{ record.type }}
            </a-tag>
          </template>
        </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
const tableColumns2 = [
  { title: '节点名称', dataIndex: 'name', slotName: 'name', width: 150 },
  { title: '角色', slotName: 'role', width: 100 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '内部IP', dataIndex: 'internalIP', width: 150 },
  { title: '外部IP', dataIndex: 'externalIP', slotName: 'externalIP', width: 150 },
  { title: 'K8s版本', dataIndex: 'version', width: 120 },
  { title: '操作系统', dataIndex: 'osImage', width: 180, ellipsis: true, tooltip: true },
  { title: '创建时间', dataIndex: 'age', width: 180 }
]

const tableColumns = [
  { title: '类型', slotName: 'type', width: 100 },
  { title: '原因', dataIndex: 'reason', width: 150, ellipsis: true, tooltip: true },
  { title: '消息', dataIndex: 'message', width: 300, ellipsis: true, tooltip: true },
  { title: '来源', dataIndex: 'source', width: 200, ellipsis: true, tooltip: true },
  { title: '次数', dataIndex: 'count', width: 80, align: 'center' },
  { title: '最后发生时间', dataIndex: 'lastTimestamp', width: 180 }
]

import { ref, onMounted, computed, nextTick } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Message } from '@arco-design/web-vue'
import axios from 'axios'
import {
  getClusterDetail,
  getClusterStats,
  getClusterNetworkInfo,
  getClusterComponentInfo,
  getNodes,
  getClusterEvents,
  type Cluster,
  type ClusterStats,
  type ClusterNetworkInfo,
  type ClusterComponentInfo,
  type NodeInfo,
  type EventInfo
} from '@/api/kubernetes'

const route = useRoute()
const router = useRouter()

const clusterId = ref<number>(parseInt(route.params.id as string))
const clusterInfo = ref<Cluster>()
let errorMessageShown = false // 跟踪是否已显示错误消息
const clusterStats = ref<ClusterStats>({
  nodeCount: 0,
  workloadCount: 0,
  podCount: 0,
  cpuUsage: 0,
  memoryUsage: 0,
  cpuCapacity: 0,
  memoryCapacity: 0,
  cpuAllocatable: 0,
  memoryAllocatable: 0,
  cpuUsed: 0,
  memoryUsed: 0
})

const networkInfo = ref<ClusterNetworkInfo>({
  serviceCIDR: '',
  podCIDR: '',
  apiServerAddress: '',
  networkPlugin: '',
  proxyMode: '',
  dnsService: ''
})

const componentInfo = ref<ClusterComponentInfo>({
  components: [],
  runtime: {
    containerRuntime: '',
    version: ''
  },
  storage: []
})

const nodeList = ref<NodeInfo[]>([])
const eventList = ref<EventInfo[]>([])
const nodesLoading = ref(false)
const eventsLoading = ref(false)

// 节点搜索和分页
const nodeSearchKeyword = ref('')
const nodeCurrentPage = ref(1)
const nodePageSize = ref(10)

// 过滤后的节点列表
const filteredNodeList = computed(() => {
  if (!nodeSearchKeyword.value) {
    return nodeList.value
  }
  const keyword = nodeSearchKeyword.value.toLowerCase()
  return nodeList.value.filter(node => {
    return (
      node.name.toLowerCase().includes(keyword) ||
      (node.internalIP && node.internalIP.toLowerCase().includes(keyword)) ||
      (node.externalIP && node.externalIP.toLowerCase().includes(keyword))
    )
  })
})

// 分页后的节点列表
const paginatedNodeList = computed(() => {
  const start = (nodeCurrentPage.value - 1) * nodePageSize.value
  const end = start + nodePageSize.value
  return filteredNodeList.value.slice(start, end)
})

// 处理节点搜索
const handleNodeSearch = () => {
  nodeCurrentPage.value = 1
}

// 处理节点分页变化
const handleNodePageChange = (page: number) => {
  nodeCurrentPage.value = page
}

const handleNodeSizeChange = (size: number) => {
  nodePageSize.value = size
  nodeCurrentPage.value = 1
}

// 快速统计卡片数据
const quickStats = computed(() => [
  {
    label: '节点数量',
    value: clusterStats.value.nodeCount,
    icon: Monitor,
    color: 'linear-gradient(135deg, #2c3e50 0%, #000000 100%)',
    iconColor: '#D4AF37',
    trend: true
  },
  {
    label: '工作负载',
    value: clusterStats.value.workloadCount,
    icon: Box,
    color: 'linear-gradient(135deg, #2c3e50 0%, #000000 100%)',
    iconColor: '#D4AF37',
    trend: true
  },
  {
    label: 'Pod 总数',
    value: clusterStats.value.podCount,
    icon: Files,
    color: 'linear-gradient(135deg, #2c3e50 0%, #000000 100%)',
    iconColor: '#D4AF37',
    trend: true
  },
  {
    label: 'CPU 使用率',
    value: Math.round(clusterStats.value.cpuUsage) + '%',
    icon: Cpu,
    color: 'linear-gradient(135deg, #2c3e50 0%, #000000 100%)',
    iconColor: '#D4AF37',
    trend: false
  }
])

// 加载集群详情
const loadClusterDetail = async () => {
  try {
    const data = await getClusterDetail(clusterId.value)
    clusterInfo.value = data
    // 并行加载所有数据
    await Promise.all([
      loadClusterStats(),
      loadNetworkInfo(),
      loadComponentInfo(),
      loadNodes(),
      loadEvents()
    ])
  } catch (error: any) {
    // 只显示一次错误消息
    if (!errorMessageShown) {
      errorMessageShown = true
      if (error.response?.status === 403 || error.response?.status === 401) {
        Message.error({
          content: '您没有权限访问该集群，请联系管理员授权',
          duration: 5000,
          showClose: true
        })
      } else {
        Message.error(error.response?.data?.message || '获取集群信息失败')
      }
    }
  }
}

// 加载集群统计信息
const loadClusterStats = async () => {
  try {
    const data = await getClusterStats(clusterId.value)
    clusterStats.value = data
  } catch (error: any) {
    throw error // 抛出错误，让 Promise.all 捕获
  }
}

// 加载网络信息
const loadNetworkInfo = async () => {
  try {
    const data = await getClusterNetworkInfo(clusterId.value)
    networkInfo.value = data
  } catch (error: any) {
    throw error
  }
}

// 加载组件信息
const loadComponentInfo = async () => {
  try {
    const data = await getClusterComponentInfo(clusterId.value)

    // 手动触发响应式更新
    componentInfo.value = {
      components: data?.components || [],
      runtime: data?.runtime || { containerRuntime: '', version: '' },
      storage: data?.storage || []
    }


    // 强制触发重新渲染
    await nextTick()
  } catch (error: any) {
    throw error
  }
}

// 加载节点列表
const loadNodes = async () => {
  nodesLoading.value = true
  try {
    const data = await getNodes(clusterId.value)
    nodeList.value = data || []
  } catch (error: any) {
    throw error
  } finally {
    nodesLoading.value = false
  }
}

// 加载事件列表
const loadEvents = async () => {
  eventsLoading.value = true
  try {
    const data = await getClusterEvents(clusterId.value)
    eventList.value = data || []
  } catch (error: any) {
    throw error
  } finally {
    eventsLoading.value = false
  }
}

// 格式化字节数
const formatBytes = (bytes: number): string => {
  if (bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i]
}

// 获取进度条颜色
const getProgressColor = (percentage: number) => {
  if (percentage < 50) return '#67C23A'
  if (percentage < 80) return '#E6A23C'
  return '#F56C6C'
}

// 返回列表
const handleBack = () => {
  router.push('/kubernetes/clusters')
}

// 获取状态类型
const getStatusType = (status: number) => {
  const statusMap: Record<number, string> = {
    1: 'success',
    2: 'danger',
    3: 'info'
  }
  return statusMap[status] || 'info'
}

// 获取状态文本
const getStatusText = (status: number) => {
  const statusMap: Record<number, string> = {
    1: '正常',
    2: '连接失败',
    3: '不可用'
  }
  return statusMap[status] || '未知'
}

// 获取服务商文本
const getProviderText = (provider: string) => {
  const providerMap: Record<string, string> = {
    native: '自建集群',
    aliyun: '阿里云 ACK',
    tencent: '腾讯云 TKE',
    aws: 'AWS EKS'
  }
  return providerMap[provider] || provider
}

// 获取节点角色类型
const getNodeRoleType = (roles: string) => {
  if (!roles) return 'info'
  if (roles.toLowerCase().includes('master') || roles.toLowerCase().includes('control-plane')) {
    return 'danger'
  }
  return 'info'
}

onMounted(() => {
  loadClusterDetail()
})
</script>

<style scoped lang="scss">
.cluster-detail-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #f5f7fa 0%, #c3cfe2 100%);
  padding: 0;
}

/* 页面头部 */
.page-header {
  margin-bottom: 24px;

  .header-content {
    background: #fff;
    border-radius: 12px;
    padding: 24px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  }

  .header-top {
    margin-bottom: 20px;

    .back-btn {
      background: linear-gradient(135deg, #2c3e50 0%, #000000 100%);
      color: #D4AF37;
      border: 1px solid rgba(212, 175, 55, 0.3);
      font-weight: 500;
      padding: 12px 24px;
      font-size: 14px;
      border-radius: 8px;
      transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
      display: inline-flex;
      align-items: center;
      gap: 6px;
      letter-spacing: 0.5px;
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.1);

      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 6px 20px rgba(212, 175, 55, 0.4);
        border-color: rgba(212, 175, 55, 0.5);
        background: linear-gradient(135deg, #34495e 0%, #1a1a1a 100%);
      }

      &:active {
        transform: translateY(0);
        box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
      }

      :deep(.arco-icon) {
        font-size: 16px;
        transition: transform 0.3s;
      }

      &:hover :deep(.arco-icon) {
        transform: translateX(-3px);
      }
    }
  }

  .cluster-name-section {
    display: flex;
    align-items: center;
    gap: 16px;
    margin-bottom: 16px;

    .cluster-title {
      display: flex;
      align-items: center;
      gap: 12px;
      margin: 0;
      font-size: 28px;
      font-weight: 600;
      color: #303133;

      .title-icon {
        color: #D4AF37;
      }
    }

    .status-tag {
      font-size: 14px;
      padding: 8px 16px;
      border-radius: 20px;
    }
  }

  .cluster-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 24px;

    .meta-item {
      display: flex;
      align-items: center;
      gap: 6px;
      color: #606266;
      font-size: 14px;

      .el-icon {
        color: #D4AF37;
      }
    }
  }
}

/* 快速统计卡片 */
.quick-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 24px;

  .stat-card {
    position: relative;
    background: #fff;
    border-radius: 12px;
    padding: 24px;
    display: flex;
    align-items: center;
    gap: 20px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
    overflow: hidden;
    transition: all 0.3s ease;
    animation: slideInUp 0.5s ease-out var(--delay) backwards;

    &::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 4px;
      background: #D4AF37;
    }

    &:hover {
      transform: translateY(-4px);
      box-shadow: 0 8px 24px rgba(212, 175, 55, 0.3);
    }

    .stat-icon-wrapper {
      width: 64px;
      height: 64px;
      border-radius: 12px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
    }

    .stat-info {
      flex: 1;

      .stat-value {
        font-size: 32px;
        font-weight: 600;
        color: #303133;
        line-height: 1.2;
        margin-bottom: 4px;
      }

      .stat-label {
        font-size: 14px;
        color: #909399;
      }
    }

    .stat-trend {
      color: #D4AF37;
      font-size: 20px;
    }
  }
}

/* 主内容区 */
.main-content {
  display: grid;
  grid-template-columns: 1.2fr 0.8fr;
  gap: 20px;
}

/* 卡片通用样式 */
.modern-card {
  border-radius: 12px;
  border: none;
  margin-bottom: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.08);
  transition: all 0.3s;

  &:hover {
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  }

  :deep(.arco-card__header) {
    padding: 20px 24px;
    border-bottom: 1px solid #f0f0f0;
  }

  :deep(.arco-card__body) {
    padding: 24px;
  }

  .card-title-section {
    display: flex;
    align-items: center;
    gap: 10px;

    .card-icon {
      flex-shrink: 0;
    }

    .card-title {
      font-size: 16px;
      font-weight: 600;
      color: #303133;
    }

    .node-count,
    .event-count {
      margin-left: auto;
      font-size: 13px;
      color: #909399;
      background: #f5f7fa;
      padding: 4px 12px;
      border-radius: 12px;
    }
  }
}

/* 全宽卡片 */
.full-width-card {
  grid-column: 1 / -1;
  margin-bottom: 20px;
}

/* 节点搜索栏 */
.node-search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;

  .search-input {
    flex: 1;
    max-width: 400px;
  }

  .search-button {
    background-color: #000;
    color: #d4af37;
    border: 1px solid #d4af37;
    border-radius: 8px;
    padding: 10px 20px;
    font-weight: 500;

    &:hover {
      background-color: #d4af37;
      color: #000;
    }
  }
}

/* 节点名称链接样式 */
.node-name-link {
  color: #d4af37;
  font-weight: 500;
  cursor: pointer;
}

/* 节点分页 */
.node-pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0;
  border-top: 1px solid #f0f0f0;
  margin-top: 16px;
}

/* 资源使用率 */
.resource-usage {
  .usage-item {
    margin-bottom: 24px;

    &:last-child {
      margin-bottom: 0;
    }

    .usage-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 12px;

      .usage-label {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 15px;
        font-weight: 500;
        color: #303133;
      }

      .usage-value {
        font-size: 24px;
        font-weight: 600;
        color: #D4AF37;
      }
    }

    .progress-wrapper {
      margin-bottom: 8px;
    }

    .usage-detail {
      display: flex;
      justify-content: space-between;
      font-size: 13px;
      color: #909399;

      .detail-text {
        padding: 4px 12px;
        background: #f5f7fa;
        border-radius: 4px;
      }
    }
  }
}

/* 网络配置 */
.network-config {
  .config-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 16px;

    .config-item {
      .config-label {
        font-size: 13px;
        color: #909399;
        margin-bottom: 8px;
      }

      .config-value {
        font-size: 14px;
        font-weight: 500;
        color: #303133;
        word-break: break-all;

        &.primary {
          color: #D4AF37;
        }
      }

      .arco-tag {
        width: 100%;
        justify-content: center;
      }
    }
  }
}

/* 集群信息 */
.cluster-info-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;

  .info-row {
    display: flex;
    flex-direction: column;
    gap: 6px;
    padding: 12px;
    background: #f5f7fa;
    border-radius: 8px;

    &.full-width {
      grid-column: 1 / -1;
    }

    .info-label {
      font-size: 13px;
      color: #909399;
    }

    .info-value {
      font-size: 14px;
      font-weight: 500;
      color: #303133;
      word-break: break-all;
    }
  }
}

/* 组件信息 */
.component-section {
  margin-bottom: 24px;

  &:last-child {
    margin-bottom: 0;
  }

  .section-header {
    display: flex;
    align-items: center;
    gap: 8px;
    font-size: 15px;
    font-weight: 600;
    color: #303133;
    margin-bottom: 16px;
    padding-bottom: 12px;
    border-bottom: 2px solid #f0f0f0;
  }

  .runtime-cards {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;

    .runtime-card {
      background: linear-gradient(135deg, #2c3e50 0%, #000000 100%);
      color: #D4AF37;
      padding: 16px;
      border-radius: 8px;
      transition: all 0.3s;
      border: 1px solid rgba(212, 175, 55, 0.2);

      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(212, 175, 55, 0.3);
        border-color: rgba(212, 175, 55, 0.4);
      }

      .runtime-label {
        font-size: 12px;
        opacity: 0.9;
        margin-bottom: 6px;
      }

      .runtime-value {
        font-size: 15px;
        font-weight: 600;
      }
    }
  }

  .component-list {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .component-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 16px;
      background: #fafafa;
      border-radius: 8px;
      border-left: 3px solid #D4AF37;
      transition: all 0.3s;

      &:hover {
        background: #fefcf5;
        box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
      }

      .component-main {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 1;

        .component-icon {
          flex-shrink: 0;
        }

        .component-info {
          display: flex;
          align-items: center;
          gap: 12px;

          .component-name {
            font-size: 14px;
            font-weight: 500;
            color: #303133;
          }
        }
      }
    }
  }

  .storage-list {
    display: flex;
    flex-direction: column;
    gap: 12px;

    .storage-item {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 16px;
      background: #fafafa;
      border-radius: 8px;
      border-left: 3px solid #D4AF37;
      transition: all 0.3s;

      &:hover {
        background: #fefcf5;
        box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
      }

      .storage-main {
        display: flex;
        align-items: center;
        gap: 12px;
        flex: 1;

        .storage-icon {
          flex-shrink: 0;
        }

        .storage-info {
          .storage-name {
            font-size: 14px;
            font-weight: 500;
            color: #303133;
            margin-bottom: 4px;
          }

          .storage-provisioner {
            font-size: 12px;
            color: #909399;
          }
        }
      }
    }
  }
}

/* 动画 */
@keyframes slideInUp {
  from {
    opacity: 0;
    transform: translateY(20px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 响应式设计 */
@media (max-width: 1400px) {
  .quick-stats {
    grid-template-columns: repeat(2, 1fr);
  }

  .main-content {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 768px) {
  .cluster-detail-container {
    padding: 16px;
  }

  .page-header {
    flex-direction: column;

    .header-content {
      width: 100%;
    }

    .cluster-meta {
      flex-direction: column;
      gap: 12px;
    }
  }

  .quick-stats {
    grid-template-columns: 1fr;
  }
}
</style>
