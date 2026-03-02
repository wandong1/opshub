<template>
  <a-modal
    v-model="dialogVisible"
    :title="`Service 详情: ${serviceData?.name || ''}`"
    width="1400px"
    :close-on-click-modal="false"
    @close="handleClose"
    class="service-detail-dialog"
  >
    <div v-if="loading" class="loading-container">
      <icon-loading />
      <span>加载中...</span>
    </div>

    <div v-else-if="serviceData" class="service-detail-container">
      <!-- 基本信息 -->
      <div class="info-section">
        <div class="section-header">
          <span class="section-icon">📋</span>
          <span class="section-title">基本信息</span>
        </div>
        <div class="info-grid">
          <div class="info-item">
            <label>名称</label>
            <span>{{ serviceData.name }}</span>
          </div>
          <div class="info-item">
            <label>命名空间</label>
            <span>{{ serviceData.namespace }}</span>
          </div>
          <div class="info-item">
            <label>类型</label>
            <a-tag :type="getTypeTagType(serviceData.type)" size="small">
              {{ serviceData.type }}
            </a-tag>
          </div>
          <div class="info-item">
            <label>Cluster IP</label>
            <span>{{ serviceData.clusterIP || '-' }}</span>
          </div>
          <div class="info-item">
            <label>外部 IP</label>
            <span>{{ serviceData.externalIP || '-' }}</span>
          </div>
          <div class="info-item">
            <label>存活时间</label>
            <span>{{ serviceData.age }}</span>
          </div>
        </div>

        <!-- 标签 -->
        <div class="tags-section" v-if="hasLabels">
          <label>标签</label>
          <div class="tags-container">
            <a-tag
              v-for="(value, key) in serviceData.labels"
              :key="key"
              size="small"
              class="tag-item"
            >
              {{ key }}: {{ value }}
            </a-tag>
          </div>
        </div>

        <!-- Selector -->
        <div class="selector-section" v-if="hasSelector">
          <label>选择器</label>
          <div class="tags-container">
            <a-tag
              v-for="(value, key) in serviceData.selector"
              :key="key"
              size="small"
              type="success"
              class="tag-item"
            >
              {{ key }}: {{ value }}
            </a-tag>
          </div>
        </div>

        <!-- 注解 -->
        <div class="annotations-section" v-if="hasAnnotations">
          <label>注解</label>
          <div class="annotations-container">
            <div
              v-for="(value, key) in serviceData.annotations"
              :key="key"
              class="annotation-item"
            >
              <span class="annotation-key">{{ key }}:</span>
              <a-tooltip :content="value" placement="top" effect="light" :show-after="500">
                <span class="annotation-value truncated">{{ value }}</span>
              </a-tooltip>
            </div>
          </div>
        </div>
      </div>

      <!-- Tab 内容 -->
      <a-tabs v-model:active-key="activeTab" class="detail-tabs">
        <!-- 容器组 -->
        <a-tab-pane title="容器组" key="pods">
          <div v-if="podsLoading" class="loading-container">
            <icon-loading />
            <span>加载中...</span>
          </div>
          <div v-else-if="pods.length > 0">
            <a-table :data="pods" size="default" class="modern-table" :columns="tableColumns2">
          <template #name="{ record }">
                  <div class="name-cell">
                    <icon-storage />
                    <span class="name-text">{{ record.name }}</span>
                  </div>
                </template>
          <template #image="{ record }">
                  <span class="image-text">{{ record.image }}</span>
                </template>
          <template #status="{ record }">
                  <a-tag :color="getStatusType(record.status)" size="small">
                    {{ record.status }}
                  </a-tag>
                </template>
          <template #cpu="{ record }">
                  <span>{{ record.cpu || '-' }}</span>
                </template>
          <template #col_Memory="{ record }">
                  <span>{{ record.memory || '-' }}</span>
                </template>
          <template #actions="{ record }">
                  <div class="action-buttons">
                    <a-tooltip content="终端" placement="top">
                      <a-button type="text" class="action-btn" @click="handleTerminal(record)">
                        <icon-desktop />
                      </a-button>
                    </a-tooltip>
                    <a-tooltip content="日志" placement="top">
                      <a-button type="text" class="action-btn" @click="handleLogs(record)">
                        <icon-file />
                      </a-button>
                    </a-tooltip>
                  </div>
                </template>
        </a-table>
          </div>
          <div v-else class="empty-container">
            <a-empty description="暂无容器组" :image-size="80" />
          </div>
        </a-tab-pane>

        <!-- 端口 -->
        <a-tab-pane title="端口" key="ports">
          <div v-if="serviceData.ports && serviceData.ports.length > 0">
            <a-table :data="serviceData.ports" size="default" class="modern-table" :columns="tableColumns">
          <template #name="{ record }">
                  <span>{{ record.name || '-' }}</span>
                </template>
          <template #protocol="{ record }">
                  <a-tag size="small" color="gray">{{ record.protocol }}</a-tag>
                </template>
          <template #col_TargetPort="{ record }">
                  <span>{{ record.targetPort || '-' }}</span>
                </template>
          <template #col_NodePort="{ record }">
                  <span>{{ record.nodePort || '-' }}</span>
                </template>
        </a-table>
          </div>
          <div v-else class="empty-container">
            <a-empty description="暂无端口配置" :image-size="80" />
          </div>
        </a-tab-pane>
      </a-tabs>
    </div>
  </a-modal>
</template>

<script setup lang="ts">
const tableColumns2 = [
  { title: '名称', dataIndex: 'name', slotName: 'name', width: 200 },
  { title: '镜像', slotName: 'image', width: 200 },
  { title: '状态', slotName: 'status', width: 100 },
  { title: '重启次数', dataIndex: 'restarts', width: 100, align: 'center' },
  { title: '节点', dataIndex: 'node', width: 150 },
  { title: 'CPU', slotName: 'cpu', width: 100 },
  { title: 'Memory', slotName: 'col_Memory', width: 100 },
  { title: '存活时间', dataIndex: 'age', width: 120 },
  { title: '操作', slotName: 'actions', width: 120, fixed: 'right', align: 'center' }
]

const tableColumns = [
  { title: '名称', dataIndex: 'name', slotName: 'name', width: 150 },
  { title: '端口', dataIndex: 'port', width: 100, align: 'center' },
  { title: '协议', dataIndex: 'protocol', slotName: 'protocol', width: 100, align: 'center' },
  { title: 'TargetPort', slotName: 'col_TargetPort', width: 120, align: 'center' },
  { title: 'NodePort', slotName: 'col_NodePort', width: 120, align: 'center' }
]

import { ref, computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getPods, getPodDetail, getServiceYAML, type PodInfo } from '@/api/kubernetes'

interface ServiceDetail {
  name: string
  namespace: string
  type: string
  clusterIP: string
  externalIP: string
  age: string
  labels: Record<string, string>
  selector: Record<string, string>
  annotations: Record<string, string>
  ports: ServicePort[]
}

interface ServicePort {
  name?: string
  port: number
  protocol: string
  targetPort?: number | string
  nodePort?: number
}

interface PodDetail extends PodInfo {
  image?: string
  cpu?: string
  memory?: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['terminal', 'logs'])

const dialogVisible = ref(false)
const loading = ref(false)
const podsLoading = ref(false)
const activeTab = ref('pods')
const serviceData = ref<ServiceDetail | null>(null)
const pods = ref<PodDetail[]>([])

const hasLabels = computed(() => {
  return serviceData.value?.labels && Object.keys(serviceData.value.labels).length > 0
})

const hasSelector = computed(() => {
  return serviceData.value?.selector && Object.keys(serviceData.value.selector).length > 0
})

const hasAnnotations = computed(() => {
  return serviceData.value?.annotations && Object.keys(serviceData.value.annotations).length > 0
})

const getTypeTagType = (type: string) => {
  const map: Record<string, string> = {
    ClusterIP: 'success',
    NodePort: 'warning',
    LoadBalancer: 'danger'
  }
  return map[type] || 'info'
}

const getStatusType = (status: string) => {
  const map: Record<string, string> = {
    Running: 'success',
    Pending: 'warning',
    Failed: 'danger',
    Succeeded: 'info',
    Unknown: 'info'
  }
  return map[status] || 'info'
}

const loadServiceDetail = async (namespace: string, name: string) => {
  if (!props.clusterId) return

  loading.value = true
  try {
    const response: any = await getServiceYAML(props.clusterId, namespace, name)
    // response 就是 {items: Service对象}
    const data = response.items

    serviceData.value = {
      name: data.metadata.name,
      namespace: data.metadata.namespace,
      type: data.spec.type,
      clusterIP: data.spec.clusterIP || '',
      externalIP: data.spec.externalIPs?.join(', ') || '',
      age: formatAge(data.metadata.creationTimestamp),
      labels: data.metadata.labels || {},
      selector: data.spec.selector || {},
      annotations: data.metadata.annotations || {},
      ports: (data.spec.ports || []).map((p: any) => ({
        name: p.name,
        port: p.port,
        protocol: p.protocol,
        targetPort: p.targetPort,
        nodePort: p.nodePort
      }))
    }

    // 加载关联的Pods
    await loadPods()
  } catch (error) {
    Message.error('加载Service详情失败: ' + (error as any).message)
  } finally {
    loading.value = false
  }
}

const loadPods = async () => {
  if (!props.clusterId || !serviceData.value) return

  podsLoading.value = true
  try {
    const podList = await getPods(props.clusterId, serviceData.value.namespace)

    // 根据selector筛选pods
    const selector = serviceData.value.selector
    if (!selector || Object.keys(selector).length === 0) {
      pods.value = []
      podsLoading.value = false
      return
    }

    const matchedPods = podList.filter(pod => {
      return Object.entries(selector).every(([key, value]) => {
        return pod.labels?.[key] === value
      })
    })

    // 逐个获取Pod详细信息
    const detailedPods = await Promise.all(
      matchedPods.map(async (pod) => {
        try {
          // getPodDetail返回的是Pod对象（因为request拦截器返回了res.data）
          const podDetail: any = await getPodDetail(props.clusterId, pod.namespace, pod.name)
          const containers = podDetail.spec?.containers || []
          const containerStatuses = podDetail.status?.containerStatuses || []

          // 获取第一个容器的主容器信息
          const mainContainer = containers[0] || {}
          const mainStatus = containerStatuses[0]

          // 获取CPU和内存信息
          const cpu = mainContainer.resources?.requests?.cpu ||
                     mainContainer.resources?.limits?.cpu || '-'

          const memory = mainContainer.resources?.requests?.memory ||
                       mainContainer.resources?.limits?.memory || '-'

          return {
            ...pod,
            image: mainContainer.image || '-',
            cpu: formatResource(cpu),
            memory: formatResource(memory),
            restarts: mainStatus?.restartCount || pod.restarts || 0
          }
        } catch (error) {
          // 失败时返回基本信息
          return {
            ...pod,
            image: '-',
            cpu: '-',
            memory: '-'
          }
        }
      })
    )

    pods.value = detailedPods
  } catch (error) {
    Message.error('加载Pod列表失败')
  } finally {
    podsLoading.value = false
  }
}

const formatResource = (value: string): string => {
  if (!value || value === '-') return '-'
  return value
}

const formatAge = (timestamp: string): string => {
  if (!timestamp) return '-'
  const now = new Date()
  const past = new Date(timestamp)
  const diff = now.getTime() - past.getTime()

  const seconds = Math.floor(diff / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  const days = Math.floor(hours / 24)

  if (days > 0) {
    return `${days}d`
  } else if (hours > 0) {
    return `${hours}h`
  } else if (minutes > 0) {
    return `${minutes}m`
  } else {
    return `${seconds}s`
  }
}

const handleTerminal = (pod: PodDetail) => {
  emit('terminal', {
    namespace: pod.namespace,
    name: pod.name
  })
}

const handleLogs = (pod: PodDetail) => {
  emit('logs', {
    namespace: pod.namespace,
    name: pod.name
  })
}

const handleClose = () => {
  dialogVisible.value = false
  serviceData.value = null
  pods.value = []
  activeTab.value = 'pods'
}

const open = (namespace: string, name: string) => {
  dialogVisible.value = true
  loadServiceDetail(namespace, name)
}

defineExpose({
  open
})
</script>

<style scoped>
.service-detail-dialog :deep(.arco-dialog__body) {
  padding: 0;
  max-height: 70vh;
  overflow-y: auto;
}

.loading-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 60px 20px;
  gap: 12px;
  color: #909399;
}

.service-detail-container {
  padding: 20px;
}

/* 基本信息区域 */
.info-section {
  background: #fff;
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 20px;
  border: 1px solid #e8e8e8;
}

.section-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding-bottom: 12px;
  border-bottom: 2px solid #165dff;
}

.section-icon {
  font-size: 18px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #1d2129;
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px 24px;
  margin-bottom: 16px;
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.info-item label {
  font-size: 12px;
  color: #909399;
  font-weight: 500;
}

.info-item span {
  font-size: 14px;
  color: #1d2129;
  font-weight: 500;
}

/* 标签区域 */
.tags-section,
.selector-section {
  margin-top: 16px;
}

.tags-section label,
.selector-section label,
.annotations-section label {
  font-size: 13px;
  color: #606266;
  font-weight: 600;
  margin-bottom: 8px;
  display: block;
}

.tags-container {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.tag-item {
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 12px;
}

/* 注解区域 */
.annotations-section {
  margin-top: 16px;
}

.annotations-container {
  background: #f5f7fa;
  border-radius: 6px;
  padding: 12px;
  max-height: 120px;
  overflow-y: auto;
}

.annotation-item {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 12px;
  line-height: 1.6;
}

.annotation-item:last-child {
  margin-bottom: 0;
}

.annotation-key {
  color: #909399;
  font-weight: 600;
  min-width: 120px;
  flex-shrink: 0;
}

.annotation-value {
  color: #606266;
  flex: 1;
}

.annotation-value.truncated {
  max-width: 400px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  cursor: help;
}

/* Tab区域 */
.detail-tabs {
  background: #fff;
  border-radius: 8px;
  padding: 16px;
  border: 1px solid #e8e8e8;
}

.detail-tabs :deep(.arco-tabs__header) {
  margin-bottom: 16px;
}

.detail-tabs :deep(.arco-tabs__item) {
  font-size: 14px;
  font-weight: 500;
}

/* 表格样式 */
.modern-table {
  background: #fff;
}

.modern-table :deep(.arco-table__header-wrapper) {
  background: #fafbfc;
}

.modern-table :deep(.arco-table__header th) {
  background: #fafbfc;
  color: #606266;
  font-weight: 600;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.name-icon {
  color: #165dff;
  font-size: 16px;
}

.name-text {
  font-weight: 500;
  color: #1d2129;
}

.image-text {
  font-size: 13px;
  color: #606266;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  color: #165dff;
}

.action-btn:hover {
  color: #4080ff;
}

/* 空状态 */
.empty-container {
  padding: 40px 20px;
  text-align: center;
}
</style>
