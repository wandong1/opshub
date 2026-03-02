<template>
  <div class="hpa-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <a-input
          v-model="searchName"
          placeholder="搜索 HPA 名称..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <icon-search />
          </template>
        </a-input>

        <a-select v-model="filterNamespace" placeholder="命名空间" allow-clear @change="handleSearch" class="filter-select">
          <a-option label="全部" value="" />
          <a-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </a-select>
      </div>

      <div class="search-bar-right">
        <a-button v-permission="'k8s-hpa:create'" type="primary" @click="handleCreate">
          <icon-plus />
          新增 HPA
        </a-button>
      </div>
    </div>

    <!-- HPA 列表 -->
    <div class="table-wrapper">
      <a-table
        :data="paginatedHPAs"
        :loading="loading"
        class="modern-table"
        size="default"
       :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <icon-rise />
              </div>
              <div class="name-content">
                <div class="name-text">{{ record.name }}</div>
                <div class="namespace-text">{{ record.namespace }}</div>
              </div>
            </div>
          </template>
          <template #referenceTarget="{ record }">
            <span class="resource-value">{{ record.referenceTarget || '-' }}</span>
          </template>
          <template #minReplicas="{ record }">
            <a-tag color="gray" size="small">{{ record.minReplicas || '-' }}</a-tag>
          </template>
          <template #maxReplicas="{ record }">
            <a-tag color="green" size="small">{{ record.maxReplicas || '-' }}</a-tag>
          </template>
          <template #currentReplicas="{ record }">
            <a-tag :type="getReplicasTagType(record.currentReplicas, record.maxReplicas)" size="small">
              {{ record.currentReplicas || '-' }}
            </a-tag>
          </template>
          <template #targetCPU="{ record }">
            <span class="resource-value">{{ record.targetCPU || '-' }}</span>
          </template>
          <template #targetMemory="{ record }">
            <span class="resource-value">{{ record.targetMemory || '-' }}</span>
          </template>
          <template #createdAt="{ record }">
            {{ record.createdAt || '-' }}
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="编辑 YAML" placement="top">
                <a-button v-permission="'k8s-hpa:update'" type="text" class="action-btn" @click="handleEditYAML(record)">
                  <icon-file />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button v-permission="'k8s-hpa:delete'" type="text" class="action-btn danger" @click="handleDelete(record)">
                  <icon-delete />
                </a-button>
              </a-tooltip>
            </div>
          </template>
        </a-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="currentPage"
          v-model:page-size="pageSize"
          :page-size-options="[10, 20, 50, 100]"
          :total="filteredHPAs.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- YAML 弹窗 -->
    <a-modal v-model:visible="yamlDialogVisible" :title="yamlDialogTitle" width="900px" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <div class="yaml-line-numbers">
          <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="yamlContent"
          class="yaml-textarea"
          spellcheck="false"
          @input="handleYamlInput"
          @scroll="handleYamlScroll"
          ref="yamlTextarea"
        ></textarea>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="yamlDialogVisible = false">关闭</a-button>
          <a-button type="primary" @click="handleSaveYAML" :loading="saving">保存</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns = [
  { title: '名称', dataIndex: 'name', slotName: 'name', width: 180 },
  { title: 'Reference Target', dataIndex: 'referenceTarget', slotName: 'referenceTarget', width: 160 },
  { title: 'Min Replicas', dataIndex: 'minReplicas', slotName: 'minReplicas', width: 120, align: 'center' },
  { title: 'Max Replicas', dataIndex: 'maxReplicas', slotName: 'maxReplicas', width: 120, align: 'center' },
  { title: 'Current Replicas', dataIndex: 'currentReplicas', slotName: 'currentReplicas', width: 140, align: 'center' },
  { title: 'Target CPU', dataIndex: 'targetCPU', slotName: 'targetCPU', width: 120, align: 'center' },
  { title: 'Target Memory', dataIndex: 'targetMemory', slotName: 'targetMemory', width: 130, align: 'center' },
  { title: '创建时间', dataIndex: 'createdAt', slotName: 'createdAt', width: 180 },
  { title: '操作', slotName: 'actions', width: 120, fixed: 'right', align: 'center' }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { getNamespaces } from '@/api/kubernetes'
import axios from 'axios'

interface HPAInfo {
  name: string
  namespace: string
  referenceTarget?: string
  minReplicas?: number
  maxReplicas?: number
  currentReplicas?: number
  targetCPU?: string
  targetMemory?: string
  age: string
  createdAt?: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update'])

const loading = ref(false)
const hpaList = ref<HPAInfo[]>([])
const namespaces = ref<{ name: string }[]>([])

// 搜索和筛选
const searchName = ref('')
const filterNamespace = ref('')

// 分页
const currentPage = ref(1)
const pageSize = ref(10)

// YAML 编辑
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedHPA = ref<HPAInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 HPA'
  }
  return `HPA YAML - ${selectedHPA.value?.name || ''}`
})

// 默认 HPA YAML 模板
const getDefaultHPAYAML = () => `apiVersion: autoscaling/v1
kind: HorizontalPodAutoscaler
metadata:
  name: example-hpa
  namespace: default
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: example-deployment
  minReplicas: 2
  maxReplicas: 5
  targetCPUUtilizationPercentage: 80
`

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 获取副本数标签类型
const getReplicasTagType = (current: number | undefined, max: number | undefined) => {
  if (!current || !max) return 'info'
  if (current >= max) return 'danger'
  if (current >= max * 0.8) return 'warning'
  return 'success'
}

// 过滤后的列表
const filteredHPAs = computed(() => {
  let result = hpaList.value

  if (searchName.value) {
    result = result.filter(h =>
      h.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (filterNamespace.value) {
    result = result.filter(h => h.namespace === filterNamespace.value)
  }

  return result
})

// 分页后的列表
const paginatedHPAs = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredHPAs.value.slice(start, end)
})

// 加载命名空间列表
const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
  }
}

// 加载 HPA 列表
const loadHPAs = async () => {
  if (!props.clusterId) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers`, {
      params: { clusterId: props.clusterId },
      headers: { Authorization: `Bearer ${token}` }
    })
    hpaList.value = response.data.data || []
  } catch (error) {
    hpaList.value = []
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// 编辑 YAML
const handleEditYAML = async (row: HPAInfo) => {
  selectedHPA.value = row
  isCreateMode.value = false

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 原生axios，需要用 response.data.data?.yaml
    yamlContent.value = response.data.data?.yaml || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    Message.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// 新增 HPA
const handleCreate = () => {
  isCreateMode.value = true
  selectedHPA.value = null
  yamlContent.value = getDefaultHPAYAML()
  yamlDialogVisible.value = true
}

// 保存 YAML
const handleSaveYAML = async () => {
  if (isCreateMode.value) {
    // 创建模式
    const nameMatch = yamlContent.value.match(/name:\s*(.+)/)
    const nsMatch = yamlContent.value.match(/namespace:\s*(.+)/)
    if (!nameMatch || !nsMatch) {
      Message.error('YAML中缺少name或namespace字段')
      return
    }
    const namespace = nsMatch[1].trim()

    saving.value = true
    try {
      const token = localStorage.getItem('token')
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/${namespace}/yaml`,
        {
          clusterId: props.clusterId,
          yaml: yamlContent.value
        },
        {
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      Message.success('创建成功')
      yamlDialogVisible.value = false
      await loadHPAs()
      emit('refresh')
    } catch (error: any) {
      Message.error(`创建失败: ${error.response?.data?.message || error.message}`)
    } finally {
      saving.value = false
    }
  } else {
    // 编辑模式
    if (!selectedHPA.value) return

    saving.value = true
    try {
      const token = localStorage.getItem('token')
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/${selectedHPA.value.namespace}/${selectedHPA.value.name}/yaml`,
        {
          clusterId: props.clusterId,
          yaml: yamlContent.value
        },
        {
          headers: { Authorization: `Bearer ${token}` }
        }
      )

      Message.success('保存成功')
      yamlDialogVisible.value = false
      await loadHPAs()
      emit('refresh')
    } catch (error: any) {
      Message.error(`保存失败: ${error.response?.data?.message || error.message}`)
    } finally {
      saving.value = false
    }
  }
}

// 删除 HPA
const handleDelete = async (row: HPAInfo) => {
  try {
    await confirmModal(
      `确定要删除 HPA ${row.name} 吗？此操作不可恢复！`,
      '删除 HPA 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/horizontalpodautoscalers/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('删除成功')
    await loadHPAs()
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// YAML编辑器输入处理
const handleYamlInput = () => {
  // 可以添加输入验证
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 监听 clusterId 变化
watch(() => props.clusterId, (newVal) => {
  if (newVal) {
    currentPage.value = 1
    loadNamespaces()
    loadHPAs()
  }
})

// 监听筛选后的数据变化，更新计数
watch(filteredHPAs, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  if (props.clusterId) {
    loadNamespaces()
    loadHPAs()
  }
})

// 暴露方法给父组件
defineExpose({
  loadHPAs
})
</script>

<style scoped>
.hpa-list {
  padding: 0;
}

/* 搜索栏 */
.search-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
  padding: 12px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-bar-left {
  display: flex;
  gap: 12px;
  flex: 1;
}

.search-bar-right {
  display: flex;
  gap: 12px;
}

.search-input {
  width: 280px;
}

.filter-select {
  width: 200px;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.arco-table__body-wrapper) {
  border-radius: 0 0 12px 12px;
}

.modern-table :deep(.arco-table__row) {
  transition: background-color 0.2s ease;
  height: 56px !important;
}

.modern-table :deep(.arco-table__row td) {
  height: 56px !important;
}

.modern-table :deep(.arco-table__row:hover) {
  background-color: #f8fafc !important;
}

.resource-value {
  font-size: 13px;
  color: #606266;
}

/* 名称单元格 */
.name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.name-icon-wrapper {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  flex-shrink: 0;
}

.name-icon {
  color: #165dff;
}

.name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.name-text {
  font-weight: 600;
  color: #303133;
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
  color: #165dff;
}

.namespace-text {
  font-size: 12px;
  color: #909399;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  justify-content: center;
}

.action-btn {
  color: #165dff;
  padding: 0;
  font-size: 16px;
}

.action-btn:hover {
  color: #4080ff;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn.danger:hover {
  color: #f78989;
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  background: #fff;
  border-top: 1px solid #f0f0f0;
}

/* YAML 编辑弹窗 */
.yaml-dialog :deep(.arco-dialog__header) {
  background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%);
  color: #165dff;
  border-radius: 8px 8px 0 0;
  padding: 20px 24px;
}

.yaml-dialog :deep(.arco-dialog__title) {
  color: #165dff;
  font-size: 16px;
  font-weight: 600;
}

.yaml-dialog :deep(.arco-dialog__body) {
  padding: 24px;
  background-color: #fafafa;
}

.yaml-editor-wrapper {
  display: flex;
  border: 1px solid #e5e6eb;
  border-radius: 6px;
  overflow: hidden;
  background-color: #fafafa;
}

.yaml-line-numbers {
  background-color: #f2f3f5;
  color: #86909c;
  padding: 16px 8px;
  text-align: right;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  user-select: none;
  overflow: hidden;
  min-width: 40px;
  border-right: 1px solid #e5e6eb;
}

.line-number {
  height: 20.8px;
  line-height: 1.6;
}

.yaml-textarea {
  flex: 1;
  background-color: #fafafa;
  color: #1d2129;
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
  color: #555;
}

.yaml-textarea:focus {
  outline: none;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

</style>
