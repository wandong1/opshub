<template>
  <div class="workloads-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon">
          <el-icon><Tools /></el-icon>
        </div>
        <div>
          <h2 class="page-title">工作负载</h2>
          <p class="page-subtitle">管理 Kubernetes 工作负载资源</p>
        </div>
      </div>
      <div class="header-actions">
        <el-button class="black-button" @click="loadWorkloads">
          <el-icon style="margin-right: 6px;"><Refresh /></el-icon>
          刷新
        </el-button>
      </div>
    </div>

    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-inputs">
        <el-input
          v-model="searchName"
          placeholder="搜索工作负载名称..."
          clearable
          @clear="handleSearch"
          @keyup.enter="handleSearch"
          @input="handleSearch"
          class="search-input"
        >
          <template #prefix>
            <el-icon class="search-icon"><Search /></el-icon>
          </template>
        </el-input>

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

        <el-select
          v-model="selectedNamespace"
          placeholder="命名空间"
          clearable
          @change="handleSearch"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><FolderOpened /></el-icon>
          </template>
          <el-option
            v-for="ns in namespaceList"
            :key="ns.name"
            :label="ns.name"
            :value="ns.name"
          />
        </el-select>

        <el-select
          v-model="selectedType"
          placeholder="工作负载类型"
          @change="handleTypeChange"
          class="filter-select"
        >
          <template #prefix>
            <el-icon class="search-icon"><Grid /></el-icon>
          </template>
          <el-option label="所有" value="" />
          <el-option label="Deployment" value="Deployment" />
          <el-option label="StatefulSet" value="StatefulSet" />
          <el-option label="DaemonSet" value="DaemonSet" />
          <el-option label="Job" value="Job" />
          <el-option label="CronJob" value="CronJob" />
        </el-select>
      </div>
    </div>

    <!-- 工作负载列表 -->
    <div class="table-wrapper">
      <el-table
        :data="paginatedWorkloadList"
        v-loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
        :row-style="{ height: '56px' }"
        :cell-style="{ padding: '8px 0' }"
      >
        <el-table-column label="名称" min-width="220" fixed="left">
          <template #header>
            <span class="header-with-icon">
              <el-icon class="header-icon header-icon-blue"><Tools /></el-icon>
              名称
            </span>
          </template>
          <template #default="{ row }">
            <div class="workload-name-cell">
              <div class="workload-name-content">
                <div class="workload-name golden-text">{{ row.name }}</div>
                <div class="workload-namespace">{{ row.namespace }}</div>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="标签" width="120" align="center">
          <template #default="{ row }">
            <div class="label-cell" @click="showLabels(row)">
              <div class="label-badge-wrapper">
                <span class="label-count">{{ Object.keys(row.labels || {}).length }}</span>
                <el-icon class="label-icon"><PriceTag /></el-icon>
              </div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="容器组" width="150" align="center">
          <template #default="{ row }">
            <div class="pod-count-cell">
              <span class="pod-count">{{ row.readyPods || 0 }}/{{ row.desiredPods || 0 }}</span>
              <span class="pod-label">Pods</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="Requests/Limits" min-width="200">
          <template #default="{ row }">
            <div class="resource-cell">
              <div v-if="row.requests?.cpu || row.limits?.cpu" class="resource-item">
                <span class="resource-label">CPU:</span>
                <span v-if="row.requests?.cpu" class="resource-value requests-value">{{ row.requests.cpu }}</span>
                <span v-if="row.requests?.cpu && row.limits?.cpu" class="resource-separator">/</span>
                <span v-if="row.limits?.cpu" class="resource-value limits-value">{{ row.limits.cpu }}</span>
              </div>
              <div v-if="row.requests?.memory || row.limits?.memory" class="resource-item">
                <span class="resource-label">Mem:</span>
                <span v-if="row.requests?.memory" class="resource-value requests-value">{{ row.requests.memory }}</span>
                <span v-if="row.requests?.memory && row.limits?.memory" class="resource-separator">/</span>
                <span v-if="row.limits?.memory" class="resource-value limits-value">{{ row.limits.memory }}</span>
              </div>
              <div v-if="!row.requests?.cpu && !row.requests?.memory && !row.limits?.cpu && !row.limits?.memory" class="resource-empty">-</div>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="镜像" min-width="300">
          <template #default="{ row }">
            <div class="image-cell">
              <el-tooltip
                v-if="row.images && row.images.length > 0"
                :content="row.images.join('\n')"
                placement="top"
              >
                <div class="image-list">
                  <span v-for="(image, index) in getDisplayImages(row.images)" :key="index" class="image-item">
                    {{ image }}
                  </span>
                  <span v-if="row.images.length > 2" class="image-more">
                    +{{ row.images.length - 2 }}
                  </span>
                </div>
              </el-tooltip>
              <span v-else class="image-empty">-</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="存活时间" width="150">
          <template #default="{ row }">
            <div class="age-cell">
              <el-icon class="age-icon"><Clock /></el-icon>
              <span>{{ formatAge(row.createdAt) }}</span>
            </div>
          </template>
        </el-table-column>

        <el-table-column label="操作" width="80" fixed="right" align="center">
          <template #default="{ row }">
            <el-dropdown trigger="click" @command="(command: string) => handleActionCommand(command, row)">
              <el-button link class="action-btn">
                <el-icon :size="18"><Edit /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu class="action-dropdown-menu">
                  <el-dropdown-item command="edit">
                    <el-icon><Edit /></el-icon>
                    <span>编辑</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="yaml">
                    <el-icon><Document /></el-icon>
                    <span>YAML</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="pods">
                    <el-icon><Monitor /></el-icon>
                    <span>Pods</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="restart" divided>
                    <el-icon><RefreshRight /></el-icon>
                    <span>重启</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="scale">
                    <el-icon><Rank /></el-icon>
                    <span>扩缩容</span>
                  </el-dropdown-item>
                  <el-dropdown-item command="delete" divided class="danger-item">
                    <el-icon><Delete /></el-icon>
                    <span>删除</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <el-pagination
          v-model:current-page="currentPage"
          v-model:page-size="pageSize"
          :page-sizes="[10, 20, 50, 100]"
          :total="filteredWorkloadList.length"
          layout="total, sizes, prev, pager, next, jumper"
          @current-change="handlePageChange"
          @size-change="handleSizeChange"
        />
      </div>
    </div>

    <!-- 标签弹窗 -->
    <el-dialog
      v-model="labelDialogVisible"
      title="工作负载标签"
      width="700px"
      class="label-dialog"
    >
      <div class="label-dialog-content">
        <el-table :data="labelList" class="label-table" max-height="500">
          <el-table-column prop="key" label="Key" min-width="280">
            <template #default="{ row }">
              <div class="label-key-wrapper" @click="copyToClipboard(row.key, 'Key')">
                <span class="label-key-text">{{ row.key }}</span>
                <el-icon class="copy-icon"><CopyDocument /></el-icon>
              </div>
            </template>
          </el-table-column>
          <el-table-column prop="value" label="Value" min-width="350">
            <template #default="{ row }">
              <span class="label-value">{{ row.value }}</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="labelDialogVisible = false">关闭</el-button>
        </div>
      </template>
    </el-dialog>

    <!-- YAML 编辑弹窗 -->
    <el-dialog
      v-model="yamlDialogVisible"
      :title="`工作负载 YAML - ${selectedWorkload?.name || ''}`"
      width="900px"
      class="yaml-dialog"
    >
      <div class="yaml-dialog-content">
        <div class="yaml-editor-wrapper">
          <div class="yaml-line-numbers">
            <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
          </div>
          <textarea
            v-model="yamlContent"
            class="yaml-textarea"
            placeholder="YAML 内容"
            spellcheck="false"
            @input="handleYamlInput"
            @scroll="handleYamlScroll"
            ref="yamlTextarea"
          ></textarea>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="yamlDialogVisible = false">取消</el-button>
          <el-button type="primary" class="black-button" @click="handleSaveYAML" :loading="yamlSaving">
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>

    <!-- 工作负载编辑对话框 -->
    <el-dialog
      v-model="editDialogVisible"
      title="编辑工作负载"
      width="90%"
      :close-on-click-modal="false"
      class="workload-edit-dialog"
    >
      <div class="workload-edit-content" v-if="editWorkloadData">
        <!-- 左侧：基础信息 -->
        <div class="edit-sidebar">
          <BasicInfo
            :formData="editWorkloadData"
            @add-label="handleAddLabel"
            @remove-label="handleRemoveLabel"
            @add-annotation="handleAddAnnotation"
            @remove-annotation="handleRemoveAnnotation"
          />
        </div>

        <!-- 右侧：详细配置 -->
        <div class="edit-main">
          <el-tabs v-model="activeEditTab" type="border-card">
            <el-tab-pane label="容器配置" name="containers">
              <div class="tab-content">
                <ContainerConfig
                  :containers="editWorkloadData.containers || []"
                  :initContainers="editWorkloadData.initContainers || []"
                  :volumes="editWorkloadData.volumes || []"
                  @updateContainers="updateContainers"
                  @updateInitContainers="updateInitContainers"
                />
              </div>
            </el-tab-pane>
            <el-tab-pane label="存储" name="storage">
              <div class="tab-content">
                <VolumeConfig
                  :volumes="editWorkloadData.volumes || []"
                  @addVolume="handleAddVolume"
                  @removeVolume="handleRemoveVolume"
                  @update="handleUpdateVolumes"
                />
              </div>
            </el-tab-pane>
            <el-tab-pane label="调度" name="scheduling">
              <div class="tab-content scheduling-tab-content">
                <!-- 调度类型 -->
                <div class="info-panel">
                  <div class="panel-header">
                    <span class="panel-icon">🎯</span>
                    <span class="panel-title">调度类型</span>
                  </div>
                  <div class="panel-content">
                    <NodeSelector
                      :formData="editWorkloadData"
                      :nodeList="nodeList"
                      :commonNodeLabels="[]"
                      @addMatchRule="handleAddMatchRule"
                      @removeMatchRule="handleRemoveMatchRule"
                      @update="handleUpdateScheduling"
                    />
                  </div>
                </div>

                <!-- 更新策略 -->
                <div class="info-panel">
                  <div class="panel-header">
                    <span class="panel-icon">🔄</span>
                    <span class="panel-title">更新策略</span>
                  </div>
                  <div class="panel-content">
                    <ScalingStrategy
                      :formData="scalingStrategyData"
                      @update="handleUpdateScalingStrategy"
                    />
                  </div>
                </div>

                <!-- 亲和性配置 -->
                <div class="info-panel">
                  <div class="panel-header">
                    <span class="panel-icon">🔗</span>
                    <span class="panel-title">亲和性配置</span>
                  </div>
                  <div class="panel-content">
                    <Affinity
                      :affinityRules="affinityRules"
                      :editingAffinityRule="editingAffinityRule"
                      :namespaceList="namespaceList"
                      @startAddAffinity="handleStartAddAffinity"
                      @cancelAffinityEdit="handleCancelAffinityEdit"
                      @saveAffinityRule="handleSaveAffinityRule"
                      @addMatchExpression="handleAddMatchExpression"
                      @removeMatchExpression="handleRemoveMatchExpression"
                      @addMatchLabel="handleAddMatchLabel"
                      @removeMatchLabel="handleRemoveMatchLabel"
                      @removeAffinityRule="handleRemoveAffinityRule"
                    />
                  </div>
                </div>

                <!-- 容忍度配置 -->
                <div class="info-panel">
                  <div class="panel-header">
                    <span class="panel-icon">✅</span>
                    <span class="panel-title">容忍度配置</span>
                  </div>
                  <div class="panel-content">
                    <Tolerations
                      :tolerations="editWorkloadData.tolerations || []"
                      @addToleration="handleAddToleration"
                      @removeToleration="handleRemoveToleration"
                    />
                  </div>
                </div>
              </div>
            </el-tab-pane>
            <el-tab-pane label="网络" name="network">
              <div class="tab-content">
                <Network :formData="editWorkloadData" />
              </div>
            </el-tab-pane>
            <el-tab-pane label="其他" name="others">
              <div class="tab-content">
                <Others :formData="editWorkloadData" />
              </div>
            </el-tab-pane>
          </el-tabs>
        </div>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="editDialogVisible = false">取消</el-button>
          <el-button type="primary" class="black-button" @click="handleSaveEdit" :loading="editSaving">
            保存
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, nextTick } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import axios from 'axios'
import {
  Search,
  Tools,
  Grid,
  Platform,
  FolderOpened,
  PriceTag,
  Clock,
  Refresh,
  Edit,
  View,
  Document,
  Monitor,
  RefreshRight,
  Rank,
  Delete,
  CopyDocument
} from '@element-plus/icons-vue'
import { getClusterList, updateWorkload, type Cluster } from '@/api/kubernetes'
// 导入工作负载编辑组件
import BasicInfo from './workload-components/BasicInfo.vue'
import ContainerConfig from './workload-components/ContainerConfig.vue'
import NodeSelector from './workload-components/spec/NodeSelector.vue'
import ScalingStrategy from './workload-components/spec/ScalingStrategy.vue'
import Affinity from './workload-components/spec/Affinity.vue'
import Tolerations from './workload-components/spec/Tolerations.vue'
import Network from './workload-components/spec/Network.vue'
import Others from './workload-components/spec/Others.vue'
import VolumeConfig from './workload-components/VolumeConfig.vue'

// 工作负载接口定义
interface Workload {
  name: string
  namespace: string
  type: string
  labels?: Record<string, string>
  readyPods?: number
  desiredPods?: number
  requests?: { cpu: string; memory: string }
  limits?: { cpu: string; memory: string }
  images?: string[]
  createdAt?: string
  updatedAt?: string
}

interface Namespace {
  name: string
}

const loading = ref(false)
const clusterList = ref<Cluster[]>([])
const namespaceList = ref<Namespace[]>([])
const selectedClusterId = ref<number>()
const selectedNamespace = ref<string>('')

// 计算属性：当前选中的集群对象
const selectedCluster = computed(() => {
  return clusterList.value.find(c => c.id === selectedClusterId.value)
})
const selectedType = ref<string>('')
const workloadList = ref<Workload[]>([])

// 搜索条件
const searchName = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)

// 标签弹窗
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])

// YAML 编辑弹窗
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const yamlSaving = ref(false)
const selectedWorkload = ref<Workload | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 工作负载编辑弹窗
const editDialogVisible = ref(false)
const editSaving = ref(false)
const editWorkloadData = ref<any>(null)
const activeEditTab = ref('containers')

// 亲和性规则
const affinityRules = ref<any[]>([])
const editingAffinityRule = ref<any>(null)

// 节点列表
const nodeList = ref<{ name: string }[]>([])

// 扩缩容策略
const scalingStrategyData = ref<any>({
  strategyType: 'RollingUpdate',
  maxSurge: '25%',
  maxUnavailable: '25%',
  minReadySeconds: 0,
  progressDeadlineSeconds: 600,
  revisionHistoryLimit: 10,
  timeoutSeconds: 600
})

// 过滤后的工作负载列表
const filteredWorkloadList = computed(() => {
  let result = workloadList.value

  if (searchName.value) {
    result = result.filter(workload =>
      workload.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  return result
})

// 分页后的工作负载列表
const paginatedWorkloadList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredWorkloadList.value.slice(start, end)
})

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 获取类型图标
const getTypeIcon = (type: string) => {
  return Tools
}

// 格式化资源显示
const formatResource = (resource: { cpu: string; memory: string }) => {
  const parts: string[] = []
  if (resource.cpu) parts.push(`cpu: ${resource.cpu}`)
  if (resource.memory) parts.push(`mem: ${resource.memory}`)
  return parts.join(' | ')
}

// 格式化存活时间
const formatAge = (createdAt: string | undefined): string => {
  if (!createdAt) return '-'

  const created = new Date(createdAt)
  const now = new Date()
  const diffMs = now.getTime() - created.getTime()
  const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays < 1) {
    const diffHours = Math.floor(diffMs / (1000 * 60 * 60))
    if (diffHours < 1) {
      const diffMinutes = Math.floor(diffMs / (1000 * 60))
      return diffMinutes < 1 ? '刚刚' : `${diffMinutes}分钟前`
    }
    return `${diffHours}小时前`
  }

  if (diffDays < 7) {
    return `${diffDays}天前`
  }

  const diffWeeks = Math.floor(diffDays / 7)
  if (diffWeeks < 4) {
    return `${diffWeeks}周前`
  }

  const diffMonths = Math.floor(diffDays / 30)
  if (diffMonths < 12) {
    return `${diffMonths}个月前`
  }

  const diffYears = Math.floor(diffDays / 365)
  return `${diffYears}年前`
}

// 获取显示的镜像（最多显示2个）
const getDisplayImages = (images?: string[]) => {
  if (!images || images.length === 0) return []
  return images.slice(0, 2).map(img => {
    // 只保留镜像名和tag，去掉registry部分
    const parts = img.split('/')
    const nameAndTag = parts[parts.length - 1]
    // 如果tag太长，截断显示
    if (nameAndTag.length > 50) {
      return nameAndTag.substring(0, 50) + '...'
    }
    return nameAndTag
  })
}

// 显示标签弹窗
const showLabels = (row: Workload) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({
    key,
    value: labels[key]
  }))
  labelDialogVisible.value = true
}

// 复制到剪贴板
const copyToClipboard = async (text: string, type: string) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(`${type} 已复制到剪贴板`)
  } catch (error) {
    // 降级方案：使用传统方法
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand('copy')
      ElMessage.success(`${type} 已复制到剪贴板`)
    } catch (err) {
      ElMessage.error('复制失败')
    }
    document.body.removeChild(textarea)
  }
}

// 处理页码变化
const handlePageChange = (page: number) => {
  currentPage.value = page
}

// 处理每页数量变化
const handleSizeChange = (size: number) => {
  pageSize.value = size
  const maxPage = Math.ceil(filteredWorkloadList.value.length / size)
  if (currentPage.value > maxPage) {
    currentPage.value = maxPage || 1
  }
}

// 加载集群列表
const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('workloads_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNamespaces()
      await loadWorkloads()
    }
  } catch (error) {
    console.error(error)
    ElMessage.error('获取集群列表失败')
  }
}

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!selectedClusterId.value) return

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/namespaces`,
      {
        params: { clusterId: selectedClusterId.value },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    namespaceList.value = response.data.data || []
  } catch (error) {
    console.error(error)
    namespaceList.value = []
  }
}

// 切换集群
const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('workloads_selected_cluster_id', selectedClusterId.value.toString())
  }
  selectedNamespace.value = ''
  currentPage.value = 1
  await loadNamespaces()
  await loadWorkloads()
}

// 切换工作负载类型
const handleTypeChange = () => {
  currentPage.value = 1
  loadWorkloads()
}

// 添加标签
const handleAddLabel = () => {
  if (!editWorkloadData.value) return
  editWorkloadData.value.labels.push({ key: '', value: '' })
}

// 删除标签
const handleRemoveLabel = (index: number) => {
  if (!editWorkloadData.value) return
  editWorkloadData.value.labels.splice(index, 1)
}

// 添加注解
const handleAddAnnotation = () => {
  if (!editWorkloadData.value) return
  editWorkloadData.value.annotations.push({ key: '', value: '' })
}

// 删除注解
const handleRemoveAnnotation = (index: number) => {
  if (!editWorkloadData.value) return
  editWorkloadData.value.annotations.splice(index, 1)
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
  loadWorkloads()
}

// 加载工作负载列表
const loadWorkloads = async () => {
  if (!selectedClusterId.value) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const params: any = { clusterId: selectedClusterId.value }
    if (selectedType.value) params.type = selectedType.value
    if (selectedNamespace.value) params.namespace = selectedNamespace.value

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/workloads`,
      {
        params,
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    workloadList.value = response.data.data || []
  } catch (error) {
    console.error(error)
    workloadList.value = []
    ElMessage.error('获取工作负载列表失败')
  } finally {
    loading.value = false
  }
}

// 处理下拉菜单命令
const handleActionCommand = (command: string, row: Workload) => {
  selectedWorkload.value = row

  switch (command) {
    case 'edit':
      handleShowEditDialog()
      break
    case 'yaml':
      handleShowYAML()
      break
    case 'pods':
      ElMessage.info('Pods 列表功能开发中...')
      break
    case 'restart':
      handleRestart()
      break
    case 'scale':
      handleScale()
      break
    case 'delete':
      handleDelete()
      break
  }
}

// 加载节点列表
const loadNodes = async () => {
  if (!selectedClusterId.value) {
    nodeList.value = []
    return
  }

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      '/api/v1/plugins/kubernetes/resources/nodes',
      {
        params: { clusterId: selectedClusterId.value },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    nodeList.value = response.data.data || []
    console.log('🔍 节点列表加载成功:', nodeList.value.length, '个节点')
  } catch (error: any) {
    console.error('获取节点列表失败:', error)
    nodeList.value = []
  }
}

// 添加匹配规则
const handleAddMatchRule = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.matchRules) {
    editWorkloadData.value.matchRules = []
  }
  // 自动切换到"调度规则匹配"类型
  editWorkloadData.value.schedulingType = 'match'
  editWorkloadData.value.matchRules.push({
    key: '',
    operator: 'In',
    value: ''
  })
  console.log('🔍 添加规则后 - schedulingType:', editWorkloadData.value.schedulingType)
  console.log('🔍 添加规则后 - matchRules:', editWorkloadData.value.matchRules)
}

// 删除匹配规则
const handleRemoveMatchRule = (index: number) => {
  if (!editWorkloadData.value || !editWorkloadData.value.matchRules) return
  editWorkloadData.value.matchRules.splice(index, 1)

  // 如果没有规则了，自动切换到"任意可用节点"
  if (editWorkloadData.value.matchRules.length === 0) {
    editWorkloadData.value.schedulingType = 'any'
    console.log('🔍 删除所有规则后，切换 schedulingType 为 any')
  }
}

// 更新调度配置
const handleUpdateScheduling = (data: { schedulingType: string; specifiedNode: string }) => {
  if (!editWorkloadData.value) {
    console.error('🔴 handleUpdateScheduling: editWorkloadData.value 是 null/undefined!')
    return
  }

  console.log('🔍 ====== handleUpdateScheduling 被调用 ======')
  console.log('🔍 接收到的数据:', data)
  console.log('🔍 更新前的 editWorkloadData.value.schedulingType:', editWorkloadData.value.schedulingType)
  console.log('🔍 更新前的 editWorkloadData.value.specifiedNode:', editWorkloadData.value.specifiedNode)

  // 使用 Object.assign 确保响应式更新
  Object.assign(editWorkloadData.value, {
    schedulingType: data.schedulingType,
    specifiedNode: data.specifiedNode
  })

  console.log('🔍 更新后的 editWorkloadData.value.schedulingType:', editWorkloadData.value.schedulingType)
  console.log('🔍 更新后的 editWorkloadData.value.specifiedNode:', editWorkloadData.value.specifiedNode)
  console.log('🔍 完整的 editWorkloadData.value:', editWorkloadData.value)
}

// 更新扩缩容策略
const handleUpdateScalingStrategy = (data: any) => {
  if (editWorkloadData.value) {
    editWorkloadData.value.strategyType = data.strategyType
    editWorkloadData.value.maxSurge = data.maxSurge
    editWorkloadData.value.maxUnavailable = data.maxUnavailable
    editWorkloadData.value.minReadySeconds = data.minReadySeconds
    editWorkloadData.value.progressDeadlineSeconds = data.progressDeadlineSeconds
    editWorkloadData.value.revisionHistoryLimit = data.revisionHistoryLimit
    editWorkloadData.value.timeoutSeconds = data.timeoutSeconds
  }
}

// 显示 YAML 编辑器
const handleShowYAML = async () => {
  if (!selectedWorkload.value) return

  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/workloads/${namespace}/${name}/yaml`,
      {
        params: { clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    yamlContent.value = response.data.data?.yaml || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    console.error('获取 YAML 失败:', error)
    ElMessage.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  } finally {
    yamlSaving.value = false
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  if (!selectedWorkload.value) return

  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace

    await axios.put(
      `/api/v1/plugins/kubernetes/resources/workloads/${namespace}/${name}/yaml`,
      {
        clusterId,
        yaml: yamlContent.value
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    await loadWorkloads()
  } catch (error) {
    console.error('保存 YAML 失败:', error)
    ElMessage.error('保存 YAML 失败')
  } finally {
    yamlSaving.value = false
  }
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 输入时自动调整滚动
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 重启工作负载
const handleRestart = async () => {
  if (!selectedWorkload.value) return

  try {
    await ElMessageBox.confirm(
      `确定要重启工作负载 ${selectedWorkload.value.name} 吗？`,
      '重启确认',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}/restart`,
      {
        clusterId: selectedClusterId.value,
        type: selectedWorkload.value.type
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('工作负载重启成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('重启失败:', error)
      ElMessage.error(`重启失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 扩缩容工作负载
const handleScale = async () => {
  if (!selectedWorkload.value) return

  try {
    const { value } = await ElMessageBox.prompt(
      `请输入 ${selectedWorkload.value.name} 的副本数：`,
      '扩缩容',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        inputValue: selectedWorkload.value.desiredPods?.toString() || '1',
        confirmButtonClass: 'black-button'
      }
    )

    const replicas = parseInt(value)
    if (isNaN(replicas) || replicas < 0) {
      ElMessage.error('请输入有效的副本数')
      return
    }

    const token = localStorage.getItem('token')
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}/scale`,
      {
        clusterId: selectedClusterId.value,
        type: selectedWorkload.value.type,
        replicas
      },
      {
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('扩缩容成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('扩缩容失败:', error)
      ElMessage.error(`扩缩容失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 显示编辑对话框
const handleShowEditDialog = async () => {
  if (!selectedWorkload.value) return

  editSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const clusterId = selectedClusterId.value
    const workloadType = selectedWorkload.value.type
    const name = selectedWorkload.value.name
    const namespace = selectedWorkload.value.namespace

    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/workloads/${namespace}/${name}/yaml`,
      {
        params: { clusterId, type: workloadType },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    // 获取返回的 JSON 数据
    const workloadData = response.data.data?.items
    if (workloadData) {
      console.log('🔍 获取到工作负载数据:', workloadData)
      console.log('🔍 副本数 replicas:', workloadData.spec?.replicas)
      console.log('🔍 完整的 spec:', workloadData.spec)

      // 转换 nodeSelector 为 matchRules 格式
      const nodeSelector = workloadData.spec?.template?.spec?.nodeSelector || {}
      console.log('🔍 从 Kubernetes 加载的 nodeSelector:', nodeSelector)

      const matchRules = Object.entries(nodeSelector).map(([key, value]) => {
        // 如果值是布尔值 true，则是 Exists 操作符
        if (value === true) {
          return {
            key,
            operator: 'Exists',
            value: ''
          }
        }
        // 否则是 In 操作符
        return {
          key,
          operator: 'In',
          value: String(value)
        }
      })

      console.log('🔍 转换后的 matchRules:', matchRules)
      console.log('🔍 matchRules 长度:', matchRules.length)

      // 解析 DNS 配置
      const dnsConfig = workloadData.spec?.template?.spec?.dnsConfig || {}
      const parsedDnsConfig = {
        nameservers: dnsConfig.nameservers || [],
        searches: dnsConfig.searches || [],
        options: (dnsConfig.options || []).map((opt: any) => ({
          name: opt.name || '',
          value: opt.value || ''
        }))
      }

      // 转换数据格式以适应组件
      const calculatedSchedulingType = workloadData.spec?.template?.spec?.nodeName ? 'specified' :
                                        (Object.keys(nodeSelector).length > 0 ? 'match' : 'any')

      console.log('🔍 ====== 加载工作负载调试信息 ======')
      console.log('🔍 workloadData.spec?.template?.spec:', workloadData.spec?.template?.spec)
      console.log('🔍 nodeName:', workloadData.spec?.template?.spec?.nodeName)
      console.log('🔍 nodeSelector:', nodeSelector)
      console.log('🔍 nodeSelector keys:', Object.keys(nodeSelector))
      console.log('🔍 计算的 schedulingType:', calculatedSchedulingType)

      editWorkloadData.value = {
        name: workloadData.metadata?.name || name,
        namespace: workloadData.metadata?.namespace || namespace,
        type: workloadData.kind || workloadType,
        replicas: workloadData.spec?.replicas || 0,
        labels: objectToKeyValueArray(workloadData.metadata?.labels || {}),
        annotations: objectToKeyValueArray(workloadData.metadata?.annotations || {}),
        nodeSelector: nodeSelector,
        nodeName: workloadData.spec?.template?.spec?.nodeName || '',
        specifiedNode: workloadData.spec?.template?.spec?.nodeName || '',
        schedulingType: calculatedSchedulingType,
        matchRules: matchRules,
        affinity: workloadData.spec?.template?.spec?.affinity || {},
        tolerations: workloadData.spec?.template?.spec?.tolerations || [],
        containers: parseContainers(workloadData.spec?.template?.spec?.containers || []),
        initContainers: parseContainers(workloadData.spec?.template?.spec?.initContainers || []),
        volumes: parseVolumesFromKubernetes(workloadData.spec?.template?.spec?.volumes || []),
        hostNetwork: workloadData.spec?.template?.spec?.hostNetwork || false,
        dnsPolicy: workloadData.spec?.template?.spec?.dnsPolicy || 'ClusterFirst',
        hostname: workloadData.spec?.template?.spec?.hostname || '',
        subdomain: workloadData.spec?.template?.spec?.subdomain || '',
        dnsConfig: parsedDnsConfig,
        terminationGracePeriodSeconds: workloadData.spec?.template?.spec?.terminationGracePeriodSeconds || 30,
        activeDeadlineSeconds: workloadData.spec?.template?.spec?.activeDeadlineSeconds,
        serviceAccountName: workloadData.spec?.template?.spec?.serviceAccountName || 'default',
        restartPolicy: workloadData.spec?.template?.spec?.restartPolicy || 'Always'
      }

      // 解析亲和性规则
      affinityRules.value = parseAffinityRules(workloadData.spec?.template?.spec?.affinity || {})
      editingAffinityRule.value = null

      // 解析扩缩容策略
      const strategy = workloadData.spec?.strategy || {}
      const rollingParams = strategy.rollingUpdate || {}
      scalingStrategyData.value = {
        strategyType: strategy.type || 'RollingUpdate',
        maxSurge: rollingParams.maxSurge || '25%',
        maxUnavailable: rollingParams.maxUnavailable || '25%',
        minReadySeconds: workloadData.spec?.minReadySeconds || 0,
        progressDeadlineSeconds: workloadData.spec?.progressDeadlineSeconds || 600,
        revisionHistoryLimit: workloadData.spec?.revisionHistoryLimit || 10,
        timeoutSeconds: 600
      }
      console.log('🔍 解析扩缩容策略:', scalingStrategyData.value)

      // 加载节点列表
      await loadNodes()

      editDialogVisible.value = true
    } else {
      ElMessage.warning('未获取到工作负载数据')
    }
  } catch (error: any) {
    console.error('获取工作负载详情失败:', error)
    ElMessage.error(`获取工作负载详情失败: ${error.response?.data?.message || error.message}`)
  } finally {
    editSaving.value = false
  }
}

// 将对象转换为键值对数组
const objectToKeyValueArray = (obj: Record<string, any>): { key: string; value: string }[] => {
  return Object.entries(obj).map(([key, value]) => ({
    key,
    value: String(value)
  }))
}

// 解析 Kubernetes Volumes 数据
const parseVolumesFromKubernetes = (volumes: any[]): any[] => {
  if (!volumes || !Array.isArray(volumes)) return []

  return volumes.map(volume => {
    const base = { name: volume.name }

    if (volume.emptyDir) {
      return {
        ...base,
        type: 'emptyDir',
        medium: volume.emptyDir.medium || '',
        sizeLimit: volume.emptyDir.sizeLimit || ''
      }
    }
    if (volume.hostPath) {
      return {
        ...base,
        type: 'hostPath',
        hostPath: {
          path: volume.hostPath.path || '',
          type: volume.hostPath.type || ''
        }
      }
    }
    if (volume.nfs) {
      return {
        ...base,
        type: 'nfs',
        nfs: {
          server: volume.nfs.server || '',
          path: volume.nfs.path || '',
          readOnly: volume.nfs.readOnly || false
        }
      }
    }
    if (volume.persistentVolumeClaim) {
      return {
        ...base,
        type: 'persistentVolumeClaim',
        persistentVolumeClaim: {
          claimName: volume.persistentVolumeClaim.claimName || '',
          readOnly: volume.persistentVolumeClaim.readOnly || false
        }
      }
    }
    if (volume.configMap) {
      return {
        ...base,
        type: 'configMap',
        configMap: {
          name: volume.configMap.name || '',
          defaultMode: volume.configMap.defaultMode,
          items: volume.configMap.items || []
        }
      }
    }
    if (volume.secret) {
      return {
        ...base,
        type: 'secret',
        secret: {
          secretName: volume.secret.secretName || '',
          defaultMode: volume.secret.defaultMode,
          items: volume.secret.items || []
        }
      }
    }

    return { ...base, type: 'unknown' }
  })
}

// 解析亲和性规则
const parseAffinityRules = (affinity: any): any[] => {
  const rules: any[] = []

  if (!affinity) return rules

  // Node Affinity
  if (affinity.nodeAffinity) {
    const nodeAff = affinity.nodeAffinity
    // Required
    if (nodeAff.requiredDuringSchedulingIgnoredDuringExecution) {
      const matchExpressions = nodeAff.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms?.[0]?.matchExpressions || []
      rules.push({
        type: 'nodeAffinity',
        priority: 'Required',
        weight: undefined,
        matchExpressions: matchExpressions.map((exp: any) => ({
          key: exp.key,
          operator: exp.operator,
          valueStr: exp.values?.join(',') || ''
        })),
        matchLabels: []
      })
    }
    // Preferred
    if (nodeAff.preferredDuringSchedulingIgnoredDuringExecution) {
      nodeAff.preferredDuringSchedulingIgnoredDuringExecution.forEach((pref: any) => {
        const matchExpressions = pref.preference.matchExpressions || []
        rules.push({
          type: 'nodeAffinity',
          priority: 'Preferred',
          weight: pref.weight,
          matchExpressions: matchExpressions.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })),
          matchLabels: []
        })
      })
    }
  }

  // Pod Affinity
  if (affinity.podAffinity) {
    const podAff = affinity.podAffinity
    // Required
    if (podAff.requiredDuringSchedulingIgnoredDuringExecution) {
      podAff.requiredDuringSchedulingIgnoredDuringExecution.forEach((rule: any) => {
        rules.push({
          type: 'podAffinity',
          priority: 'Required',
          namespaces: rule.labelSelector?.matchLabels ? Object.keys(rule.labelSelector.matchLabels) : [],
          matchExpressions: rule.labelSelector?.matchExpressions?.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })) || [],
          matchLabels: rule.labelSelector?.matchLabels ? Object.entries(rule.labelSelector.matchLabels).map(([k, v]) => ({ key: k, value: v })) : [],
          weight: undefined
        })
      })
    }
    // Preferred
    if (podAff.preferredDuringSchedulingIgnoredDuringExecution) {
      podAff.preferredDuringSchedulingIgnoredDuringExecution.forEach((pref: any) => {
        rules.push({
          type: 'podAffinity',
          priority: 'Preferred',
          weight: pref.weight,
          namespaces: pref.podAffinityTerm?.labelSelector?.matchLabels ? Object.keys(pref.podAffinityTerm.labelSelector.matchLabels) : [],
          matchExpressions: pref.podAffinityTerm?.labelSelector?.matchExpressions?.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })) || [],
          matchLabels: pref.podAffinityTerm?.labelSelector?.matchLabels ? Object.entries(pref.podAffinityTerm.labelSelector.matchLabels).map(([k, v]) => ({ key: k, value: v })) : [],
        })
      })
    }
  }

  // Pod Anti-Affinity
  if (affinity.podAntiAffinity) {
    const podAntiAff = affinity.podAntiAffinity
    // Required
    if (podAntiAff.requiredDuringSchedulingIgnoredDuringExecution) {
      podAntiAff.requiredDuringSchedulingIgnoredDuringExecution.forEach((rule: any) => {
        rules.push({
          type: 'podAntiAffinity',
          priority: 'Required',
          namespaces: rule.labelSelector?.matchLabels ? Object.keys(rule.labelSelector.matchLabels) : [],
          matchExpressions: rule.labelSelector?.matchExpressions?.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })) || [],
          matchLabels: rule.labelSelector?.matchLabels ? Object.entries(rule.labelSelector.matchLabels).map(([k, v]) => ({ key: k, value: v })) : [],
          weight: undefined
        })
      })
    }
    // Preferred
    if (podAntiAff.preferredDuringSchedulingIgnoredDuringExecution) {
      podAntiAff.preferredDuringSchedulingIgnoredDuringExecution.forEach((pref: any) => {
        rules.push({
          type: 'podAntiAffinity',
          priority: 'Preferred',
          weight: pref.weight,
          namespaces: pref.podAffinityTerm?.labelSelector?.matchLabels ? Object.keys(pref.podAffinityTerm.labelSelector.matchLabels) : [],
          matchExpressions: pref.podAffinityTerm?.labelSelector?.matchExpressions?.map((exp: any) => ({
            key: exp.key,
            operator: exp.operator,
            valueStr: exp.values?.join(',') || ''
          })) || [],
          matchLabels: pref.podAffinityTerm?.labelSelector?.matchLabels ? Object.entries(pref.podAffinityTerm.labelSelector.matchLabels).map(([k, v]) => ({ key: k, value: v })) : [],
        })
      })
    }
  }

  return rules
}

// 添加亲和性规则
const handleStartAddAffinity = (type: 'pod' | 'node') => {
  const isPod = type === 'pod'
  editingAffinityRule.value = {
    type: isPod ? 'podAffinity' : 'nodeAffinity',
    namespaces: [],
    topologyKey: isPod ? 'kubernetes.io/hostname' : undefined,
    priority: 'Required',
    weight: 50,
    matchExpressions: [],
    matchLabels: []
  }

  // 滚动到配置区域
  nextTick(() => {
    const configContainer = document.querySelector('.affinity-config-container')
    if (configContainer) {
      configContainer.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  })
}

// 取消编辑亲和性
const handleCancelAffinityEdit = () => {
  editingAffinityRule.value = null
}

// 保存亲和性规则
const handleSaveAffinityRule = () => {
  if (!editingAffinityRule.value) return

  // 验证 Pod 亲和性的拓扑键
  if (editingAffinityRule.value.type.includes('pod') && !editingAffinityRule.value.topologyKey) {
    ElMessage.warning('Pod 亲和性必须指定拓扑键')
    return
  }

  // 验证必填字段
  if (editingAffinityRule.value.matchExpressions.length === 0 &&
      editingAffinityRule.value.matchLabels.length === 0) {
    ElMessage.warning('请至少添加一个匹配表达式或标签')
    return
  }

  affinityRules.value.push({ ...editingAffinityRule.value })
  editingAffinityRule.value = null
  ElMessage.success('亲和性规则添加成功')
}

// 添加匹配表达式
const handleAddMatchExpression = () => {
  if (!editingAffinityRule.value) return
  editingAffinityRule.value.matchExpressions.push({
    key: '',
    operator: 'In',
    valueStr: ''
  })
}

// 删除匹配表达式
const handleRemoveMatchExpression = (index: number) => {
  if (!editingAffinityRule.value) return
  editingAffinityRule.value.matchExpressions.splice(index, 1)
}

// 添加匹配标签
const handleAddMatchLabel = () => {
  if (!editingAffinityRule.value) return
  editingAffinityRule.value.matchLabels.push({
    key: '',
    value: ''
  })
}

// 删除匹配标签
const handleRemoveMatchLabel = (index: number) => {
  if (!editingAffinityRule.value) return
  editingAffinityRule.value.matchLabels.splice(index, 1)
}

// 删除亲和性规则
const handleRemoveAffinityRule = (index: number) => {
  affinityRules.value.splice(index, 1)
  ElMessage.success('亲和性规则删除成功')
}

// 添加容忍度
const handleAddToleration = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.tolerations) {
    editWorkloadData.value.tolerations = []
  }
  editWorkloadData.value.tolerations.push({
    key: '',
    operator: 'Equal',
    value: '',
    effect: 'NoSchedule',
    tolerationSeconds: ''
  })
}

// 删除容忍度
const handleRemoveToleration = (index: number) => {
  if (!editWorkloadData.value?.tolerations) return
  editWorkloadData.value.tolerations.splice(index, 1)
}

// 将前端数据转换为 Kubernetes YAML 格式
const convertToKubernetesYaml = (data: any, cluster: string, namespace: string): string => {
  const kindMap: Record<string, string> = {
    'Deployment': 'Deployment',
    'StatefulSet': 'StatefulSet',
    'DaemonSet': 'DaemonSet',
    'Job': 'Job',
    'CronJob': 'CronJob'
  }

  const kind = kindMap[data.type] || data.type
  const apiVersion = data.type === 'CronJob' ? 'batch/v1' : 'apps/v1'

  // 构建 labels
  const labels: Record<string, string> = {}
  if (data.labels) {
    data.labels.forEach((l: any) => {
      if (l.key) labels[l.key] = l.value
    })
  }

  // 构建 annotations
  const annotations: Record<string, string> = {}
  if (data.annotations) {
    data.annotations.forEach((a: any) => {
      if (a.key) annotations[a.key] = a.value
    })
  }

  // 构建 affinity
  const affinity = buildAffinityFromRules(affinityRules.value)
  console.log('🔍 保存时 - affinityRules:', affinityRules.value)
  console.log('🔍 保存时 - 构建的 affinity:', affinity)

  // 构建 tolerations
  const tolerations = (data.tolerations || []).map((t: any) => {
    const toleration: any = {
      key: t.key,
      operator: t.operator,
      effect: t.effect
    }
    if (t.operator === 'Equal' && t.value) {
      toleration.value = t.value
    }
    if (t.effect === 'NoExecute' && t.tolerationSeconds) {
      toleration.tolerationSeconds = parseInt(t.tolerationSeconds)
    }
    return toleration
  })

  // 构建 volumes
  const volumes = (data.volumes || []).map((v: any) => {
    const volume: any = { name: v.name }
    if (v.type === 'emptyDir') {
      volume.emptyDir = {}
      if (v.medium) volume.emptyDir.medium = v.medium
      if (v.sizeLimit) volume.emptyDir.sizeLimit = v.sizeLimit
    } else if (v.type === 'hostPath' && v.hostPath) {
      volume.hostPath = {
        path: v.hostPath.path,
        type: v.hostPath.type || ''
      }
    } else if (v.type === 'nfs' && v.nfs) {
      volume.nfs = {
        server: v.nfs.server,
        path: v.nfs.path,
        readOnly: v.nfs.readOnly || false
      }
    } else if (v.type === 'configMap' && v.configMap) {
      const configMap: any = { name: v.configMap.name }
      if (v.configMap.defaultMode) configMap.defaultMode = v.configMap.defaultMode
      if (v.configMap.items && v.configMap.items.length > 0) {
        configMap.items = v.configMap.items
      }
      volume.configMap = configMap
    } else if (v.type === 'secret' && v.secret) {
      const secret: any = { secretName: v.secret.secretName }
      if (v.secret.defaultMode) secret.defaultMode = v.secret.defaultMode
      if (v.secret.items && v.secret.items.length > 0) {
        secret.items = v.secret.items
      }
      volume.secret = secret
    } else if (v.type === 'persistentVolumeClaim' && v.persistentVolumeClaim) {
      volume.persistentVolumeClaim = {
        claimName: v.persistentVolumeClaim.claimName,
        readOnly: v.persistentVolumeClaim.readOnly || false
      }
    }
    return volume
  })

  // 构建 containers
  const containers = (data.containers || []).map((c: any) => buildContainer(c, volumes))

  // 构建 initContainers
  const initContainers = (data.initContainers || []).map((c: any) => buildContainer(c, volumes))

  // 构建 pod template spec
  const podSpec: any = {
    containers,
    restartPolicy: 'Always',
    dnsPolicy: 'ClusterFirst'
  }

  if (initContainers.length > 0) {
    podSpec.initContainers = initContainers
  }

  if (volumes.length > 0) {
    podSpec.volumes = volumes
  }

  if (affinity && Object.keys(affinity).length > 0) {
    podSpec.affinity = affinity
  }

  if (tolerations.length > 0) {
    podSpec.tolerations = tolerations
  }

  // 明确删除 Pod 级别的 securityContext（包括 sysctls 等可能导致问题的配置）
  // 通过设置为 null 来确保删除旧配置
  podSpec.securityContext = null

  // 处理调度类型 - 关键：先完全删除调度相关字段，然后根据类型重新设置
  delete podSpec.nodeName
  delete podSpec.nodeSelector

  console.log('🔍 ====== 保存调度配置 ======')
  console.log('🔍 schedulingType:', data.schedulingType)
  console.log('🔍 specifiedNode:', data.specifiedNode)
  console.log('🔍 matchRules:', data.matchRules)

  if (data.schedulingType === 'specified' && data.specifiedNode) {
    // 指定节点 - 明确设置 nodeName
    podSpec.nodeName = data.specifiedNode
    console.log('🔍 设置 nodeName:', podSpec.nodeName)
  } else if (data.schedulingType === 'match') {
    // 调度规则匹配 - 构建 nodeSelector
    const nodeSelector: Record<string, any> = {}
    if (data.matchRules && data.matchRules.length > 0) {
      data.matchRules.forEach((rule: any) => {
        if (rule.key) {
          if (rule.operator === 'In' || rule.operator === 'NotIn') {
            if (rule.value) {
              const values = rule.value.split(',').map((v: string) => v.trim()).filter((v: string) => v)
              if (values.length > 0) {
                nodeSelector[rule.key] = values.length === 1 ? values[0] : values
              }
            }
          } else if (rule.operator === 'Exists') {
            nodeSelector[rule.key] = true
          }
        }
      })
    }

    if (Object.keys(nodeSelector).length > 0) {
      podSpec.nodeSelector = nodeSelector
      console.log('🔍 设置 nodeSelector:', nodeSelector)
    } else {
      console.log('🔍 nodeSelector 为空，不设置')
    }
  } else {
    // 任意可用节点 - 明确设置为 null 以删除 Kubernetes 中的字段
    podSpec.nodeName = null
    podSpec.nodeSelector = null
    console.log('🔍 任意可用节点 - nodeName 和 nodeSelector 设置为 null')
  }

  // 构建 Pod template
  const podTemplate = {
    metadata: {
      labels
    },
    spec: podSpec
  }

  console.log('🔍 构建的 podSpec:', JSON.stringify(podSpec, null, 2))
  console.log('🔍 podSpec.affinity:', podSpec.affinity)

  // 构建 Deployment spec
  const deploymentSpec: any = {
    replicas: data.replicas || 1,
    selector: {
      matchLabels: { app: labels.app || data.name }
    },
    template: podTemplate
  }

  // 添加扩缩容策略
  if (data.strategyType) {
    const strategy: any = {
      type: data.strategyType
    }

    if (data.strategyType === 'RollingUpdate') {
      strategy.rollingUpdate = {}
      if (data.maxSurge) strategy.rollingUpdate.maxSurge = data.maxSurge
      if (data.maxUnavailable) strategy.rollingUpdate.maxUnavailable = data.maxUnavailable
    }

    deploymentSpec.strategy = strategy
  }

  if (data.minReadySeconds) {
    deploymentSpec.minReadySeconds = data.minReadySeconds
  }

  if (data.progressDeadlineSeconds) {
    deploymentSpec.progressDeadlineSeconds = data.progressDeadlineSeconds
  }

  if (data.revisionHistoryLimit) {
    deploymentSpec.revisionHistoryLimit = data.revisionHistoryLimit
  }

  // 构建 metadata
  const metadata: any = {
    name: data.name,
    namespace,
    labels
  }

  if (Object.keys(annotations).length > 0) {
    metadata.annotations = annotations
  }

  // 构建完整的资源对象
  const resource: any = {
    apiVersion,
    kind,
    metadata,
    spec: deploymentSpec
  }

  // 转换为 JSON 字符串
  const jsonStr = JSON.stringify(resource)
  console.log('🔍 ====== 最终发送的 JSON ======')
  console.log('🔍 JSON 长度:', jsonStr.length)
  console.log('🔍 podSpec 部分:', JSON.stringify(podSpec, null, 2))

  return jsonStr
}

// 构建容器对象
const buildContainer = (container: any, volumes: any[]): any => {
  const c: any = {
    name: container.name,
    image: container.image,
    imagePullPolicy: container.imagePullPolicy || 'IfNotPresent'
  }

  // command 和 args
  if (container.command && container.command.length > 0) {
    c.command = container.command
  }
  if (container.args && container.args.length > 0) {
    c.args = container.args
  }

  // workingDir
  if (container.workingDir) {
    c.workingDir = container.workingDir
  }

  // ports
  if (container.ports && container.ports.length > 0) {
    c.ports = container.ports.map((p: any) => {
      const port: any = {
        containerPort: p.containerPort,
        protocol: p.protocol || 'TCP'
      }
      if (p.name) port.name = p.name
      if (p.hostPort) port.hostPort = p.hostPort
      if (p.hostIP) port.hostIP = p.hostIP
      return port
    })
  }

  // env
  if (container.env && container.env.length > 0) {
    c.env = container.env.map((e: any) => {
      const env: any = { name: e.name }
      if (e.valueFrom === 'configmap') {
        env.valueFrom = {
          configMapKeyRef: {
            name: e.configmapName,
            key: e.key
          }
        }
      } else if (e.valueFrom === 'secret') {
        env.valueFrom = {
          secretKeyRef: {
            name: e.secretName,
            key: e.key
          }
        }
      } else if (e.valueFrom === 'field') {
        env.valueFrom = {
          fieldRef: {
            fieldPath: e.fieldPath
          }
        }
      } else if (e.valueFrom === 'resource') {
        env.valueFrom = {
          resourceFieldRef: {
            container: container.name,
            resource: e.resourceField,
            divisor: e.divisor || '1'
          }
        }
      } else {
        env.value = e.value
      }
      return env
    })
  }

  // resources
  if (container.resources) {
    const resources: any = {}
    if (container.resources.requests && (container.resources.requests.cpu || container.resources.requests.memory)) {
      resources.requests = {}
      if (container.resources.requests.cpu) resources.requests.cpu = container.resources.requests.cpu
      if (container.resources.requests.memory) resources.requests.memory = container.resources.requests.memory
    }
    if (container.resources.limits && (container.resources.limits.cpu || container.resources.limits.memory)) {
      resources.limits = {}
      if (container.resources.limits.cpu) resources.limits.cpu = container.resources.limits.cpu
      if (container.resources.limits.memory) resources.limits.memory = container.resources.limits.memory
    }
    if (Object.keys(resources).length > 0) {
      c.resources = resources
    }
  }

  // volumeMounts
  if (container.volumeMounts && container.volumeMounts.length > 0) {
    c.volumeMounts = container.volumeMounts.map((vm: any) => {
      const mount: any = {
        name: vm.name,
        mountPath: vm.mountPath
      }
      if (vm.subPath) mount.subPath = vm.subPath
      if (vm.readOnly) mount.readOnly = true
      return mount
    })
  }

  // lifecycle (postStart, preStop)
  if (container.postStart || container.preStop) {
    c.lifecycle = {}
    if (container.postStart) {
      c.lifecycle.postStart = {
        exec: {
          command: container.postStart
        }
      }
    }
    if (container.preStop) {
      c.lifecycle.preStop = {
        exec: {
          command: container.preStop
        }
      }
    }
  }

  // probes
  if (container.livenessProbe) {
    c.livenessProbe = buildProbe(container.livenessProbe)
  }
  if (container.readinessProbe) {
    c.readinessProbe = buildProbe(container.readinessProbe)
  }
  if (container.startupProbe) {
    c.startupProbe = buildProbe(container.startupProbe)
  }

  return c
}

// 构建 probe 对象
const buildProbe = (probe: any): any => {
  if (!probe || !probe.enabled) return null

  const p: any = {
    initialDelaySeconds: probe.initialDelaySeconds || 0,
    timeoutSeconds: probe.timeoutSeconds || 3,
    periodSeconds: probe.periodSeconds || 10,
    successThreshold: probe.successThreshold || 1,
    failureThreshold: probe.failureThreshold || 3
  }

  // 根据类型构建探针
  if (probe.type === 'httpGet') {
    p.httpGet = {
      path: probe.path || '/',
      port: probe.port || 80,
      scheme: probe.scheme || 'HTTP'
    }
    if (probe.httpHeaders && probe.httpHeaders.length > 0) {
      p.httpGet.httpHeaders = probe.httpHeaders
    }
  } else if (probe.type === 'tcpSocket') {
    p.tcpSocket = {
      port: probe.port || 80
    }
  } else if (probe.type === 'exec') {
    if (probe.command && probe.command.length > 0) {
      p.exec = {
        command: probe.command
      }
    }
  } else if (probe.type === 'grpc') {
    p.grpc = {
      port: probe.port || 80,
      service: probe.service || null
    }
  }

  return p
}

// 从亲和性规则构建 Kubernetes affinity 对象
const buildAffinityFromRules = (rules: any[]): any => {
  console.log('🔍 buildAffinityFromRules - 输入的规则:', rules)
  console.log('🔍 buildAffinityFromRules - 规则数量:', rules?.length || 0)

  const affinity: any = {}

  for (const rule of rules) {
    if (rule.type === 'nodeAffinity') {
      if (!affinity.nodeAffinity) {
        affinity.nodeAffinity = {}
      }
      if (rule.priority === 'Required') {
        if (!affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
          affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution = {
            nodeSelectorTerms: []
          }
        }
        const term = buildNodeSelectorTerm(rule)
        affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.push(term)
      } else {
        if (!affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
          affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution.push({
          weight: rule.weight || 50,
          preference: buildNodeSelectorTerm(rule)
        })
      }
    } else if (rule.type === 'nodeAntiAffinity') {
      if (!affinity.nodeAffinity) {
        affinity.nodeAffinity = {}
      }
      if (rule.priority === 'Required') {
        if (!affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
          affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution = {
            nodeSelectorTerms: []
          }
        }
        const term = buildNodeSelectorTerm(rule)
        affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.push(term)
      } else {
        if (!affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
          affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution.push({
          weight: rule.weight || 50,
          preference: buildNodeSelectorTerm(rule)
        })
      }
    } else if (rule.type === 'podAffinity') {
      if (!affinity.podAffinity) {
        affinity.podAffinity = {}
      }
      const podAffinityTerm = buildPodAffinityTerm(rule)
      if (!podAffinityTerm) {
        console.warn('⚠️ buildPodAffinityTerm 返回 null，跳过此规则')
        continue
      }
      if (rule.priority === 'Required') {
        if (!affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
          affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution.push(podAffinityTerm)
      } else {
        if (!affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
          affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution.push({
          weight: rule.weight || 50,
          podAffinityTerm
        })
      }
    } else if (rule.type === 'podAntiAffinity') {
      if (!affinity.podAntiAffinity) {
        affinity.podAntiAffinity = {}
      }
      const podAffinityTerm = buildPodAffinityTerm(rule)
      if (!podAffinityTerm) {
        console.warn('⚠️ buildPodAffinityTerm 返回 null，跳过此规则')
        continue
      }
      if (rule.priority === 'Required') {
        if (!affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution) {
          affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution.push(podAffinityTerm)
      } else {
        if (!affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution) {
          affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution = []
        }
        affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution.push({
          weight: rule.weight || 50,
          podAffinityTerm
        })
      }
    }
  }

  // 清理空对象
  if (affinity.nodeAffinity && Object.keys(affinity.nodeAffinity).length === 0) {
    delete affinity.nodeAffinity
  }

  if (affinity.podAffinity && Object.keys(affinity.podAffinity).length === 0) {
    delete affinity.podAffinity
  }

  if (affinity.podAntiAffinity && Object.keys(affinity.podAntiAffinity).length === 0) {
    delete affinity.podAntiAffinity
  }

  console.log('🔍 buildAffinityFromRules - 构建的 affinity:', affinity)
  console.log('🔍 buildAffinityFromRules - affinity keys:', Object.keys(affinity))

  if (Object.keys(affinity).length === 0) return undefined
  return affinity
}

// 构建节点选择器条件
const buildNodeSelectorTerm = (rule: any): any => {
  const matchExpressions = (rule.matchExpressions || []).map((exp: any) => {
    const expression: any = {
      key: exp.key,
      operator: exp.operator
    }
    if (exp.operator !== 'Exists' && exp.operator !== 'DoesNotExist') {
      expression.values = exp.valueStr ? exp.valueStr.split(',').filter((v: string) => v) : []
    }
    return expression
  })

  // 添加 matchLabels
  const matchLabels: Record<string, string> = {}
  if (rule.matchLabels) {
    rule.matchLabels.forEach((l: any) => {
      if (l.key && l.value) matchLabels[l.key] = l.value
    })
  }

  const term: any = {}

  // 只有在有内容时才添加 matchExpressions
  if (matchExpressions.length > 0) {
    term.matchExpressions = matchExpressions
  }

  // 只有在有内容时才添加 matchLabels
  if (Object.keys(matchLabels).length > 0) {
    term.matchLabels = matchLabels
  }

  console.log('🔍 buildNodeSelectorTerm - 构建的 term:', term)

  return term
}

// 构建 Pod 亲和性条件
const buildPodAffinityTerm = (rule: any): any => {
  console.log('🔍 buildPodAffinityTerm - 输入的 rule:', rule)

  const matchExpressions = (rule.matchExpressions || []).map((exp: any) => {
    const expression: any = {
      key: exp.key,
      operator: exp.operator
    }
    if (exp.operator !== 'Exists' && exp.operator !== 'DoesNotExist') {
      expression.values = exp.valueStr ? exp.valueStr.split(',').filter((v: string) => v) : []
    }
    return expression
  })

  // 添加 matchLabels
  const matchLabels: Record<string, string> = {}
  if (rule.matchLabels) {
    rule.matchLabels.forEach((l: any) => {
      if (l.key && l.value) matchLabels[l.key] = l.value
    })
  }

  const labelSelector: any = {}

  // 只有在有内容时才添加 matchExpressions
  if (matchExpressions.length > 0) {
    labelSelector.matchExpressions = matchExpressions
  }

  // 只有在有内容时才添加 matchLabels
  if (Object.keys(matchLabels).length > 0) {
    labelSelector.matchLabels = matchLabels
  }

  // 如果 labelSelector 为空，返回 null 以表示无效配置
  if (Object.keys(labelSelector).length === 0) {
    console.warn('⚠️ buildPodAffinityTerm - labelSelector 为空，返回 null')
    return null
  }

  const podAffinityTerm: any = {
    labelSelector,
    topologyKey: rule.topologyKey || 'kubernetes.io/hostname'
  }

  console.log('🔍 buildPodAffinityTerm - 构建的 podAffinityTerm:', podAffinityTerm)
  console.log('🔍 buildPodAffinityTerm - labelSelector keys:', Object.keys(labelSelector))

  return podAffinityTerm
}

// 保存编辑
const handleSaveEdit = async () => {
  if (!editWorkloadData.value || !selectedWorkload.value) return

  editSaving.value = true

  try {
    const clusterName = selectedCluster.value?.name || ''
    const yaml = convertToKubernetesYaml(
      editWorkloadData.value,
      clusterName,
      editWorkloadData.value.namespace || 'default'
    )

    await updateWorkload({
      cluster: clusterName,
      namespace: editWorkloadData.value.namespace || 'default',
      type: editWorkloadData.value.type,
      name: editWorkloadData.value.name,
      yaml
    })

    ElMessage.success('工作负载更新成功')
    editDialogVisible.value = false

    // 重新加载列表
    await loadWorkloads()
  } catch (error: any) {
    console.error('更新工作负载失败:', error)
    ElMessage.error(error.response?.data?.message || '更新工作负载失败')
  } finally {
    editSaving.value = false
  }
}

// 解析容器数据
const parseContainers = (containers: any[]): any[] => {
  if (!containers || !Array.isArray(containers)) return []

  return containers.map(container => {
    // 解析环境变量
    let envs: any[] = []
    if (container.env) {
      for (const e of container.env) {
        if (e.valueFrom?.configMapKeyRef) {
          // ConfigMap 引用
          envs.push({
            name: e.name,
            configmapName: e.valueFrom.configMapKeyRef.name,
            key: e.valueFrom.configMapKeyRef.key,
            valueFrom: {
              type: 'configmap',
              configMapName: e.valueFrom.configMapKeyRef.name,
              key: e.valueFrom.configMapKeyRef.key
            }
          })
        } else if (e.valueFrom?.secretKeyRef) {
          // Secret 引用
          envs.push({
            name: e.name,
            secretName: e.valueFrom.secretKeyRef.name,
            key: e.valueFrom.secretKeyRef.key,
            valueFrom: {
              type: 'secret',
              secretName: e.valueFrom.secretKeyRef.name,
              key: e.valueFrom.secretKeyRef.key
            }
          })
        } else if (e.valueFrom?.fieldRef) {
          // Pod 字段引用
          envs.push({
            name: e.name,
            value: e.value || '',
            valueFrom: {
              type: 'fieldRef',
              fieldPath: e.valueFrom.fieldRef.fieldPath
            }
          })
        } else if (e.valueFrom?.resourceFieldRef) {
          // 资源字段引用
          envs.push({
            name: e.name,
            value: e.value || '',
            valueFrom: {
              type: 'resourceFieldRef',
              resource: e.valueFrom.resourceFieldRef.resource,
              containerName: e.valueFrom.resourceFieldRef.containerName,
              divisor: e.valueFrom.resourceFieldRef.divisor
            }
          })
        } else {
          // 普通变量
          envs.push({
            name: e.name,
            value: e.value || ''
          })
        }
      }
    }

    return {
      name: container.name || '',
      image: container.image || '',
      imagePullPolicy: container.imagePullPolicy || 'IfNotPresent',
      workingDir: container.workingDir || '',
      command: container.command || [],
      args: container.args || [],
      env: envs,
      resources: {
        requests: {
          cpu: container.resources?.requests?.cpu || '',
          memory: container.resources?.requests?.memory || ''
        },
        limits: {
          cpu: container.resources?.limits?.cpu || '',
          memory: container.resources?.limits?.memory || ''
        }
      },
      ports: (container.ports || []).map((p: any) => ({
        name: p.name || '',
        containerPort: p.containerPort || 0,
        protocol: p.protocol || 'TCP',
        hostPort: p.hostPort,
        hostIP: p.hostIP || ''
      })),
      volumeMounts: (container.volumeMounts || []).map((vm: any) => ({
        name: vm.name || '',
        mountPath: vm.mountPath || '',
        subPath: vm.subPath || '',
        readOnly: vm.readOnly || false
      })),

      // 解析探针配置
      livenessProbe: parseProbe(container.livenessProbe),
      readinessProbe: parseProbe(container.readinessProbe),
      startupProbe: parseProbe(container.startupProbe),

      stdin: container.stdin || false,
      tty: container.tty || false,
      activeTab: 'basic'
    }
  })
}

// 解析探针配置
const parseProbe = (probe: any): any => {
  if (!probe) return null

  const result: any = {
    enabled: true,
    type: 'httpGet',
    initialDelaySeconds: probe.initialDelaySeconds || 0,
    timeoutSeconds: probe.timeoutSeconds || 3,
    periodSeconds: probe.periodSeconds || 10,
    successThreshold: probe.successThreshold || 1,
    failureThreshold: probe.failureThreshold || 3
  }

  // 确定探针类型
  if (probe.httpGet) {
    result.type = 'httpGet'
    result.path = probe.httpGet.path || '/'
    result.port = probe.httpGet.port || 80
    result.scheme = probe.httpGet.scheme || 'HTTP'
    if (probe.httpGet.httpHeaders) {
      result.httpHeaders = probe.httpGet.httpHeaders
    }
  } else if (probe.tcpSocket) {
    result.type = 'tcpSocket'
    result.port = probe.tcpSocket.port || 80
  } else if (probe.exec) {
    result.type = 'exec'
    result.command = probe.exec.command || []
  }

  return result
}

// 更新容器列表
const updateContainers = (containers: any[]) => {
  if (editWorkloadData.value) {
    editWorkloadData.value.containers = containers
  }
}

// 更新初始化容器列表
const updateInitContainers = (initContainers: any[]) => {
  if (editWorkloadData.value) {
    editWorkloadData.value.initContainers = initContainers
  }
}

// 添加数据卷
const handleAddVolume = () => {
  if (!editWorkloadData.value) return
  if (!editWorkloadData.value.volumes) {
    editWorkloadData.value.volumes = []
  }
  editWorkloadData.value.volumes.push({
    name: '',
    type: 'emptyDir',
    medium: '',
    sizeLimit: ''
  })
}

// 删除数据卷
const handleRemoveVolume = (index: number) => {
  if (!editWorkloadData.value?.volumes) return
  editWorkloadData.value.volumes.splice(index, 1)
}

// 更新数据卷
const handleUpdateVolumes = (volumes: any[]) => {
  if (editWorkloadData.value) {
    editWorkloadData.value.volumes = volumes
  }
}

// 删除工作负载
const handleDelete = async () => {
  if (!selectedWorkload.value) return

  try {
    await ElMessageBox.confirm(
      `确定要删除工作负载 ${selectedWorkload.value.name} 吗？此操作不可恢复！`,
      '删除确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error',
        confirmButtonClass: 'black-button'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/workloads/${selectedWorkload.value.namespace}/${selectedWorkload.value.name}`,
      {
        params: {
          clusterId: selectedClusterId.value,
          type: selectedWorkload.value.type
        },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    ElMessage.success('删除成功')
    await loadWorkloads()
  } catch (error: any) {
    if (error !== 'cancel') {
      console.error('删除失败:', error)
      ElMessage.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

onMounted(() => {
  loadClusters()
})
</script>

<style scoped>
.workloads-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
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
  background: #d4af37;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #1a1a1a;
  font-size: 22px;
  flex-shrink: 0;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
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

.black-button {
  background: #d4af37 !important;
  color: #1a1a1a !important;
  border: none !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 600;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
}

.black-button:hover {
  background: #c9a227 !important;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.4);
}

/* 搜索栏 */
.search-bar {
  margin-bottom: 12px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  display: flex;
  gap: 16px;
}

.search-inputs {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-input {
  width: 280px;
}

.filter-select,
.cluster-select {
  width: 180px;
}

.search-icon {
  color: #d4af37;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

/* 搜索框样式优化 */
.search-bar :deep(.el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;
  background-color: #fff;
}

.search-bar :deep(.el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.15);
}

.search-bar :deep(.el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 2px 12px rgba(212, 175, 55, 0.25);
}

.search-bar :deep(.el-select .el-input__wrapper) {
  border-radius: 8px;
  border: 1px solid #dcdfe6;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
}

/* 表头图标 */
.header-with-icon {
  display: flex;
  align-items: center;
  gap: 6px;
}

.header-icon {
  font-size: 16px;
}

.header-icon-blue {
  color: #d4af37;
}

/* 现代表格 */
.modern-table {
  width: 100%;
}

.modern-table :deep(.el-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.el-table__row) {
  transition: background-color 0.2s ease;
  height: 56px !important;
}

.modern-table :deep(.el-table__row td) {
  height: 56px !important;
}

.modern-table :deep(.el-table__row:hover) {
  background-color: #f8fafc !important;
}

.modern-table :deep(.el-tag) {
  border-radius: 6px;
  padding: 4px 10px;
  font-weight: 500;
}

/* 工作负载名称单元格 */
.workload-name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.workload-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: #d4af37;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  box-shadow: 0 2px 6px rgba(212, 175, 55, 0.25);
}

.workload-icon {
  color: #1a1a1a;
  font-size: 18px;
}

.workload-name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.workload-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
}

.golden-text {
  color: #d4af37 !important;
}

.workload-namespace {
  font-size: 12px;
  color: #909399;
}

/* 标签单元格 */
.label-cell {
  display: flex;
  justify-content: center;
  align-items: center;
  cursor: pointer;
  padding: 5px 0;
}

.label-badge-wrapper {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.label-icon {
  color: #d4af37;
  font-size: 20px;
  transition: all 0.3s;
}

.label-count {
  position: absolute;
  top: -6px;
  right: -6px;
  background-color: #d4af37;
  color: #000;
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid #d4af37;
  z-index: 1;
}

.label-cell:hover .label-icon {
  color: #bfa13f;
  transform: scale(1.1);
}

.label-cell:hover .label-count {
  background-color: #bfa13f;
  border-color: #bfa13f;
}

/* Pod 数量 */
.pod-count-cell {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.pod-count {
  font-size: 18px;
  font-weight: 600;
  color: #d4af37;
}

.pod-label {
  font-size: 11px;
  color: #909399;
}

/* 资源单元格 */
.resource-cell {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.resource-item {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
}

.resource-label {
  color: #909399;
  font-weight: 500;
  min-width: 45px;
}

.resource-value {
  color: #303133;
  font-family: 'Monaco', 'Menlo', monospace;
  font-weight: 500;
}

.requests-value {
  color: #67c23a;
}

.limits-value {
  color: #e6a23c;
}

.resource-separator {
  color: #dcdfe6;
  margin: 0 4px;
}

.resource-empty {
  color: #909399;
}

/* 镜像单元格 */
.image-cell {
  display: flex;
  align-items: center;
}

.image-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-items: center;
}

.image-item {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 11px;
  color: #606266;
  background: #f5f7fa;
  padding: 2px 8px;
  border-radius: 4px;
  max-width: 280px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.image-more {
  font-size: 11px;
  color: #909399;
  background: #f5f7fa;
  padding: 2px 6px;
  border-radius: 4px;
  cursor: pointer;
}

.image-empty {
  color: #909399;
  font-size: 13px;
}

/* 时间单元格 */
.age-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #606266;
}

.age-icon {
  color: #d4af37;
}

/* 操作按钮 */
.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  color: #d4af37;
}

.action-btn:hover {
  color: #bfa13f;
}

/* 下拉菜单样式 */
.action-dropdown-menu {
  min-width: 140px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  font-size: 13px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item .el-icon) {
  color: #d4af37;
  font-size: 16px;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item) {
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item .el-icon) {
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item:hover) {
  background-color: #f5f5f5;
  color: #d4af37;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item:hover .el-icon) {
  color: #d4af37;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item:hover) {
  background-color: #fef0f0;
  color: #f56c6c;
}

.action-dropdown-menu :deep(.el-dropdown-menu__item.danger-item:hover .el-icon) {
  color: #f56c6c;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* 标签弹窗 */
.label-dialog :deep(.el-dialog__header) {
  background: #d4af37;
  color: #1a1a1a;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.label-dialog :deep(.el-dialog__title) {
  color: #1a1a1a;
  font-size: 16px;
  font-weight: 600;
}

.label-dialog-content {
  padding: 8px 0;
}

.label-table {
  width: 100%;
}

.label-table :deep(.el-table__cell) {
  padding: 8px 0;
}

.label-key-wrapper {
  display: inline-flex !important;
  align-items: center !important;
  gap: 6px !important;
  padding: 5px 12px !important;
  background: rgba(212, 175, 55, 0.1) !important;
  color: #d4af37 !important;
  border: 1px solid #d4af37 !important;
  border-radius: 6px !important;
  font-family: 'Monaco', 'Menlo', monospace !important;
  font-size: 12px !important;
  font-weight: 600 !important;
  cursor: pointer !important;
  transition: all 0.3s !important;
  user-select: none;
}

.label-key-wrapper:hover {
  background: rgba(212, 175, 55, 0.2) !important;
  border-color: #c9a227 !important;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3) !important;
  transform: translateY(-1px);
}

.label-key-wrapper:active {
  transform: translateY(0);
}

.label-key-text {
  flex: 1;
  word-break: break-all;
  line-height: 1.4;
  white-space: pre-wrap;
}

.copy-icon {
  font-size: 14px;
  flex-shrink: 0;
  opacity: 0.7;
  transition: opacity 0.3s;
}

.label-key-wrapper:hover .copy-icon {
  opacity: 1;
}

.label-value {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: #666;
  word-break: break-all;
  white-space: pre-wrap;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* YAML 编辑弹窗 */
.yaml-dialog :deep(.el-dialog__header) {
  background: #d4af37;
  color: #1a1a1a;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.yaml-dialog :deep(.el-dialog__title) {
  color: #1a1a1a;
  font-size: 16px;
  font-weight: 600;
}

.yaml-dialog :deep(.el-dialog__body) {
  padding: 24px;
  background-color: #ffffff;
}

.yaml-dialog-content {
  padding: 0;
}

.yaml-editor-wrapper {
  display: flex;
  border: 1px solid #e8e8e8;
  border-radius: 8px;
  overflow: hidden;
  background-color: #fafafa;
}

.yaml-line-numbers {
  background-color: #f5f5f5;
  color: #999;
  padding: 16px 8px;
  text-align: right;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  user-select: none;
  overflow: hidden;
  min-width: 40px;
  border-right: 1px solid #e8e8e8;
}

.line-number {
  height: 20.8px;
  line-height: 1.6;
}

.yaml-textarea {
  flex: 1;
  background-color: #fafafa;
  color: #333;
  border: none;
  outline: none;
  padding: 16px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
  min-height: 400px;
}

.yaml-textarea::placeholder {
  color: #aaa;
}

.yaml-textarea:focus {
  outline: none;
  background-color: #ffffff;
}

/* 响应式设计 */
@media (max-width: 1400px) {
  .search-inputs {
    flex-wrap: wrap;
  }
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .header-actions {
    width: 100%;
    flex-direction: column;
  }

  .cluster-select,
  .filter-select {
    width: 100%;
  }
}

/* 工作负载编辑对话框 - 白金风格 */
.workload-edit-dialog :deep(.el-dialog__wrapper) {
  overflow: hidden;
}

.workload-edit-dialog :deep(.el-dialog) {
  background: #ffffff;
  border: 1px solid #e8e8e8;
  border-radius: 16px;
  overflow: hidden;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.1);
  margin: auto;
  max-height: calc(100vh - 100px);
  display: flex;
  flex-direction: column;
}

.workload-edit-dialog :deep(.el-dialog__header) {
  background: #d4af37;
  border-bottom: 2px solid #c9a227;
  padding: 24px 32px;
  margin: 0;
  position: relative;
}

.workload-edit-dialog :deep(.el-dialog__header::before) {
  display: none;
}

.workload-edit-dialog :deep(.el-dialog__title) {
  font-size: 20px;
  font-weight: 700;
  color: #1a1a1a;
  letter-spacing: 0.5px;
  font-family: 'Helvetica Neue', Arial, sans-serif;
}

.workload-edit-dialog :deep(.el-dialog__headerbtn .el-dialog__close) {
  color: #1a1a1a;
  font-size: 20px;
  transition: all 0.3s ease;
  font-weight: bold;
}

.workload-edit-dialog :deep(.el-dialog__headerbtn .el-dialog__close:hover) {
  color: #000000;
  transform: rotate(90deg);
}

.workload-edit-dialog :deep(.el-dialog__body) {
  padding: 0;
  background: #ffffff;
  flex: 1;
  overflow: hidden;
  min-height: 0;
}

.workload-edit-dialog :deep(.el-dialog__footer) {
  padding: 16px 32px;
  background: #ffffff;
  border-top: 1px solid #e8e8e8;
  flex-shrink: 0;
}

.workload-edit-content {
  display: flex;
  height: calc(100vh - 200px);
  max-height: 800px;
  background: #ffffff;
}

.edit-sidebar {
  width: 360px;
  flex-shrink: 0;
  background: linear-gradient(135deg, #fafafa 0%, #f5f5f5 100%);
  border-right: 2px solid #e8e8e8;
  overflow-y: auto;
}

.edit-sidebar::-webkit-scrollbar {
  width: 8px;
}

.edit-sidebar::-webkit-scrollbar-track {
  background: #f5f5f5;
}

.edit-sidebar::-webkit-scrollbar-thumb {
  background: #d4af37;
  border-radius: 4px;
}

.edit-sidebar::-webkit-scrollbar-thumb:hover {
  background: #c9a227;
}

.edit-main {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  background: #ffffff;
}

.edit-main :deep(.el-tabs) {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: transparent;
}

.edit-main :deep(.el-tabs__header) {
  margin: 0;
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
  border-bottom: 2px solid #e8e8e8;
  padding: 0 32px;
}

.edit-main :deep(.el-tabs__nav-wrap::after) {
  display: none;
}

.edit-main :deep(.el-tabs__item) {
  color: #666;
  font-weight: 500;
  font-size: 15px;
  padding: 0 28px;
  height: 54px;
  line-height: 54px;
  border: none;
  transition: all 0.3s ease;
  letter-spacing: 0.3px;
}

.edit-main :deep(.el-tabs__item:hover) {
  color: #d4af37;
}

.edit-main :deep(.el-tabs__item.is-active) {
  color: #d4af37;
  background: transparent;
  font-weight: 600;
}

.edit-main :deep(.el-tabs__active-bar) {
  height: 3px;
  background: #d4af37;
}

.edit-main :deep(.el-tabs__content) {
  flex: 1;
  overflow-y: auto;
  padding: 0;
  background: transparent;
}

.edit-main :deep(.el-tabs__content)::-webkit-scrollbar {
  width: 10px;
}

.edit-main :deep(.el-tabs__content)::-webkit-scrollbar-track {
  background: #fafafa;
}

.edit-main :deep(.el-tabs__content)::-webkit-scrollbar-thumb {
  background: #d4af37;
  border-radius: 5px;
}

.edit-main :deep(.el-tabs__content)::-webkit-scrollbar-thumb:hover {
  background: #c9a227;
}

.tab-content {
  padding: 0;
  height: 100%;
  overflow-y: auto;
  background: #ffffff;
}

/* 调度页面样式 */
.scheduling-tab-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0;
}

.info-panel {
  background: #fff;
  border-radius: 4px;
  overflow: hidden;
}

.panel-header {
  display: flex;
  align-items: center;
  padding: 12px 20px;
  background: #d4af37;
  border-bottom: 1px solid #d4af37;
}

.panel-icon {
  font-size: 18px;
  margin-right: 8px;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: #ffffff;
  flex: 1;
}

.panel-content {
  padding: 16px;
  background: #ffffff;
}

.placeholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 450px;
  color: #999;
  font-size: 16px;
  gap: 20px;
  background: #fafafa;
  border-radius: 12px;
  border: 1px dashed #e0e0e0;
}

.placeholder :deep(.el-icon) {
  font-size: 64px;
  opacity: 0.4;
  color: #d4af37;
}

/* 白金风格按钮样式 */
.edit-main :deep(.el-button--primary),
.edit-sidebar :deep(.el-button--primary) {
  background: #d4af37;
  border: none;
  color: #1a1a1a;
  font-weight: 600;
  letter-spacing: 0.3px;
  box-shadow: 0 2px 8px rgba(212, 175, 55, 0.3);
  transition: all 0.3s ease;
}

.edit-main :deep(.el-button--primary:hover),
.edit-sidebar :deep(.el-button--primary:hover) {
  background: #c9a227;
  box-shadow: 0 4px 12px rgba(212, 175, 55, 0.4);
  transform: translateY(-1px);
}

.edit-main :deep(.el-button--primary:active),
.edit-sidebar :deep(.el-button--primary:active) {
  transform: translateY(0);
}

.edit-main :deep(.el-button--default),
.edit-sidebar :deep(.el-button--default) {
  background: #ffffff;
  border: 1px solid #e0e0e0;
  color: #666;
  font-weight: 500;
  transition: all 0.3s ease;
}

.edit-main :deep(.el-button--default:hover),
.edit-sidebar :deep(.el-button--default:hover) {
  background: #fafafa;
  border-color: #d4af37;
  color: #d4af37;
}

.edit-main :deep(.el-button--danger) {
  background: #ff4d4f;
  border: none;
  color: #ffffff;
  font-weight: 500;
}

.edit-main :deep(.el-button--danger:hover) {
  background: #ff7875;
}

/* 白金风格输入框 */
.edit-main :deep(.el-input__wrapper),
.edit-main :deep(.el-textarea__inner),
.edit-main :deep(.el-select .el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
  transition: all 0.3s ease;
}

.edit-main :deep(.el-input__wrapper:hover),
.edit-main :deep(.el-textarea__inner:hover),
.edit-main :deep(.el-select .el-input__wrapper:hover) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 3px rgba(212, 175, 55, 0.1);
}

.edit-main :deep(.el-input__wrapper.is-focus),
.edit-main :deep(.el-textarea__inner:focus),
.edit-main :deep(.el-select .el-input__wrapper.is-focus) {
  border-color: #d4af37;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05), 0 0 0 4px rgba(212, 175, 55, 0.15);
}

.edit-main :deep(.el-input__inner) {
  color: #333;
  font-weight: 500;
}

.edit-main :deep(.el-input__inner::placeholder) {
  color: #aaa;
}

.edit-main :deep(.el-textarea__inner) {
  color: #333;
  background: #fafafa;
}

.edit-main :deep(.el-select .el-input__inner) {
  color: #333;
}

/* 白金风格标签 */
.edit-main :deep(.el-tag) {
  background: rgba(212, 175, 55, 0.1);
  border: 1px solid #d4af37;
  color: #d4af37;
  font-weight: 600;
}

.edit-main :deep(.el-tag--success) {
  background: rgba(82, 196, 26, 0.1);
  border-color: #52c41a;
  color: #52c41a;
}

.edit-main :deep(.el-tag--warning) {
  background: rgba(250, 173, 20, 0.1);
  border-color: #faad14;
  color: #faad14;
}

.edit-main :deep(.el-tag--danger) {
  background: rgba(255, 77, 79, 0.1);
  border-color: #ff4d4f;
  color: #ff4d4f;
}

/* 白金风格表单 */
.edit-main :deep(.el-form-item__label) {
  color: #333;
  font-weight: 600;
  font-size: 14px;
  letter-spacing: 0.3px;
}

.edit-main :deep(.el-checkbox__label) {
  color: #333;
  font-weight: 500;
}

.edit-main :deep(.el-checkbox__input.is-checked .el-checkbox__inner) {
  background: #d4af37;
  border-color: #d4af37;
}

/* 白金风格表格 */
.edit-main :deep(.el-table) {
  background: #ffffff;
  color: #333;
}

.edit-main :deep(.el-table th) {
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
  color: #333;
  font-weight: 600;
  border-bottom: 2px solid #e8e8e8;
}

.edit-main :deep(.el-table tr) {
  transition: all 0.3s ease;
}

.edit-main :deep(.el-table tr:hover) {
  background: #fafafa;
}

.edit-main :deep(.el-table td) {
  border-bottom: 1px solid #f0f0f0;
}

/* 白金风格折叠面板 */
.edit-main :deep(.el-collapse-item__header) {
  background: linear-gradient(135deg, #fafafa 0%, #ffffff 100%);
  border: 1px solid #e8e8e8;
  color: #333;
  font-weight: 600;
  transition: all 0.3s ease;
}

.edit-main :deep(.el-collapse-item__header:hover) {
  background: #ffffff;
  border-color: #d4af37;
}

.edit-main :deep(.el-collapse-item__wrap) {
  background: #ffffff;
  border: none;
}

/* 白金风格开关 */
.edit-main :deep(.el-switch.is-checked .el-switch__core) {
  background: #d4af37;
  border-color: #d4af37;
}

/* 白金风格选择器下拉 */
.edit-main :deep(.el-select-dropdown) {
  background: #ffffff;
  border: 1px solid #e8e8e8;
}

.edit-main :deep(.el-select-dropdown__item) {
  color: #333;
}

.edit-main :deep(.el-select-dropdown__item:hover) {
  background: #fafafa;
  color: #d4af37;
}

.edit-main :deep(.el-select-dropdown__item.is-selected) {
  background: rgba(212, 175, 55, 0.1);
  color: #d4af37;
}

/* 白金风格数字输入框 */
.edit-main :deep(.el-input-number .el-input__wrapper) {
  background: #fafafa;
  border: 1px solid #e0e0e0;
}

.edit-main :deep(.el-input-number__decrease),
.edit-main :deep(.el-input-number__increase) {
  background: #f5f5f5;
  border-left: 1px solid #e0e0e0;
  color: #d4af37;
}

.edit-main :deep(.el-input-number__decrease:hover),
.edit-main :deep(.el-input-number__increase:hover) {
  color: #c9a227;
}

</style>
