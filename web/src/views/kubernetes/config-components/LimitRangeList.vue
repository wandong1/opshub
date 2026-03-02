<template>
  <div class="limitrange-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <a-input
          v-model="searchName"
          placeholder="搜索 LimitRange 名称..."
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
        <a-button v-permission="'k8s-limitranges:create'" type="primary" @click="handleCreate">
          <icon-plus />
          新增 LimitRange
        </a-button>
      </div>
    </div>

    <!-- LimitRange 列表 -->
    <div class="table-wrapper">
      <a-table
        :data="paginatedLimitRanges"
        :loading="loading"
        class="modern-table"
        size="default"
       :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <icon-settings />
              </div>
              <div class="name-content">
                <div class="name-text">{{ record.name }}</div>
                <div class="namespace-text">{{ record.namespace }}</div>
              </div>
            </div>
          </template>
          <template #type="{ record }">
            <a-tag color="gray" size="small">{{ record.type || '-' }}</a-tag>
          </template>
          <template #resource="{ record }">
            <span class="resource-value">{{ record.resource || '-' }}</span>
          </template>
          <template #min="{ record }">
            <span class="resource-value">{{ record.min || '-' }}</span>
          </template>
          <template #max="{ record }">
            <span class="resource-value">{{ record.max || '-' }}</span>
          </template>
          <template #defaultLimit="{ record }">
            <span class="resource-value">{{ record.defaultLimit || '-' }}</span>
          </template>
          <template #defaultRequest="{ record }">
            <span class="resource-value">{{ record.defaultRequest || '-' }}</span>
          </template>
          <template #maxLimitRequestRatio="{ record }">
            <span class="resource-value">{{ record.maxLimitRequestRatio || '-' }}</span>
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="编辑 YAML" placement="top">
                <a-button v-permission="'k8s-limitranges:update'" type="text" class="action-btn" @click="handleEditYAML(record)">
                  <icon-file />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button v-permission="'k8s-limitranges:delete'" type="text" class="action-btn danger" @click="handleDelete(record)">
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
          :total="filteredLimitRanges.length"
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
  { title: 'Type', dataIndex: 'type', slotName: 'type', width: 120, align: 'center' },
  { title: 'Resource', dataIndex: 'resource', slotName: 'resource', width: 120, align: 'center' },
  { title: 'Min', dataIndex: 'min', slotName: 'min', width: 120, align: 'center' },
  { title: 'Max', dataIndex: 'max', slotName: 'max', width: 120, align: 'center' },
  { title: 'Default', dataIndex: 'defaultLimit', slotName: 'defaultLimit', width: 120, align: 'center' },
  { title: 'Default Request', dataIndex: 'defaultRequest', slotName: 'defaultRequest', width: 140, align: 'center' },
  { title: 'Max Limit Request Ratio', dataIndex: 'maxLimitRequestRatio', slotName: 'maxLimitRequestRatio', width: 180, align: 'center' },
  { title: '操作', slotName: 'actions', width: 120, fixed: 'right', align: 'center' }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { getNamespaces } from '@/api/kubernetes'
import axios from 'axios'

interface LimitRangeInfo {
  name: string
  namespace: string
  type?: string
  resource?: string
  min?: string
  max?: string
  defaultLimit?: string
  defaultRequest?: string
  maxLimitRequestRatio?: string
  age: string
}

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update'])

const loading = ref(false)
const limitRangeList = ref<LimitRangeInfo[]>([])
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
const selectedLimitRange = ref<LimitRangeInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 LimitRange'
  }
  return `LimitRange YAML - ${selectedLimitRange.value?.name || ''}`
})

// 默认 LimitRange YAML 模板
const getDefaultLimitRangeYAML = () => `apiVersion: v1
kind: LimitRange
metadata:
  name: example-limitrange
  namespace: default
spec:
  limits:
  - type: Container
    default:
      cpu: "500m"
      memory: "512Mi"
    defaultRequest:
      cpu: "100m"
      memory: "256Mi"
    max:
      cpu: "1"
      memory: "1Gi"
    min:
      cpu: "50m"
      memory: "64Mi"
`

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 过滤后的列表
const filteredLimitRanges = computed(() => {
  let result = limitRangeList.value

  if (searchName.value) {
    result = result.filter(lr =>
      lr.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (filterNamespace.value) {
    result = result.filter(lr => lr.namespace === filterNamespace.value)
  }

  return result
})

// 分页后的列表
const paginatedLimitRanges = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredLimitRanges.value.slice(start, end)
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

// 加载 LimitRange 列表
const loadLimitRanges = async () => {
  if (!props.clusterId) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/limitranges`, {
      params: { clusterId: props.clusterId },
      headers: { Authorization: `Bearer ${token}` }
    })
    limitRangeList.value = response.data.data || []
  } catch (error) {
    limitRangeList.value = []
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// 编辑 YAML
const handleEditYAML = async (row: LimitRangeInfo) => {
  selectedLimitRange.value = row
  isCreateMode.value = false

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/limitranges/${row.namespace}/${row.name}/yaml`,
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

// 新增 LimitRange
const handleCreate = () => {
  isCreateMode.value = true
  selectedLimitRange.value = null
  yamlContent.value = getDefaultLimitRangeYAML()
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
        `/api/v1/plugins/kubernetes/resources/limitranges/${namespace}/yaml`,
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
      await loadLimitRanges()
      emit('refresh')
    } catch (error: any) {
      Message.error(`创建失败: ${error.response?.data?.message || error.message}`)
    } finally {
      saving.value = false
    }
  } else {
    // 编辑模式
    if (!selectedLimitRange.value) return

    saving.value = true
    try {
      const token = localStorage.getItem('token')
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/limitranges/${selectedLimitRange.value.namespace}/${selectedLimitRange.value.name}/yaml`,
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
      await loadLimitRanges()
      emit('refresh')
    } catch (error: any) {
      Message.error(`保存失败: ${error.response?.data?.message || error.message}`)
    } finally {
      saving.value = false
    }
  }
}

// 删除 LimitRange
const handleDelete = async (row: LimitRangeInfo) => {
  try {
    await confirmModal(
      `确定要删除 LimitRange ${row.name} 吗？此操作不可恢复！`,
      '删除 LimitRange 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/limitranges/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('删除成功')
    await loadLimitRanges()
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
    loadLimitRanges()
  }
})

// 监听筛选后的数据变化，更新计数
watch(filteredLimitRanges, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  if (props.clusterId) {
    loadNamespaces()
    loadLimitRanges()
  }
})

// 暴露方法给父组件
defineExpose({
  loadLimitRanges
})
</script>

<style scoped>
.limitrange-list {
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
