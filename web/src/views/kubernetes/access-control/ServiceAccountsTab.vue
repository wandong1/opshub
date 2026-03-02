<template>
  <div class="service-accounts-tab">
    <!-- 表格 -->
    <div class="table-wrapper">
      <a-table
        :data="paginatedList"
        :loading="loading"
        class="modern-table"
        size="default"
        :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }"
       :columns="tableColumns2">
          <template #name="{ record }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <icon-user />
              </div>
              <span class="name-text">{{ record.name }}</span>
            </div>
          </template>
          <template #namespace="{ record }">
            <a-tag size="small" color="gray">{{ record.namespace }}</a-tag>
          </template>
          <template #actions="{ record }">
            <a-button v-permission="'k8s-serviceaccounts:update'" type="text" @click="handleEdit(record)" class="action-btn">
              <icon-edit />
            </a-button>
            <a-button v-permission="'k8s-serviceaccounts:delete'" type="text" @click="handleDelete(record)" class="action-btn danger">
              <icon-delete />
            </a-button>
          </template>
        </a-table>

      <!-- 分页 -->
      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="currentPage"
          v-model:page-size="pageSize"
          :page-size-options="[10, 20, 50, 100]"
          :total="filteredData.length"
          show-total show-page-size show-jumper
        />
      </div>
    </div>

    <!-- 标签弹窗 -->
    <a-modal
      v-model:visible="labelDialogVisible"
      title="标签"
      width="600px"
    >
      <a-table :data="labelList" max-height="400" :columns="tableColumns">

        </a-table>
    </a-modal>

    <!-- YAML 编辑弹窗 -->
    <a-modal
      v-model:visible="yamlDialogVisible"
      :title="yamlDialogTitle"
      width="900px"
      :mask-closable="false"
      class="yaml-dialog"
    >
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
          <a-button type="primary" @click="handleSaveYaml" :loading="yamlSaving">保存</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns2 = [
  { title: '名称', slotName: 'name', width: 200, fixed: 'left' },
  { title: '命名空间', dataIndex: 'namespace', slotName: 'namespace', width: 180 },
  { title: '存活时间', dataIndex: 'age', width: 140 },
  { title: '操作', slotName: 'actions', width: 100, fixed: 'right', align: 'center' }
]

const tableColumns = [
  { title: 'Key', dataIndex: 'key', width: 150 },
  { title: 'Value', dataIndex: 'value', width: 150 }
]

import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getServiceAccounts, type ServiceAccountInfo } from '@/api/kubernetes'
import axios from 'axios'
import * as yaml from 'js-yaml'

interface Props {
  clusterId: number
  namespace?: string
  searchName?: string
}

const props = defineProps<Props>()
const emit = defineEmits<{
  'count-update': [count: number]
}>()
const loading = ref(false)
const serviceAccounts = ref<ServiceAccountInfo[]>([])
const currentPage = ref(1)
const pageSize = ref(10)
const labelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])
const yamlDialogVisible = ref(false)
const yamlDialogTitle = ref('')
const yamlContent = ref('')
const yamlSaving = ref(false)
const editingItem = ref<ServiceAccountInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 默认 ServiceAccount YAML 模板
const defaultServiceAccountYaml = `apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-serviceaccount
  namespace: default
`.trim()

// 过滤后的数据
const filteredData = computed(() => {
  let result = serviceAccounts.value
  if (props.searchName) {
    result = result.filter(item =>
      item.name.toLowerCase().includes(props.searchName!.toLowerCase())
    )
  }
  return result
})

// 分页后的数据
const paginatedList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredData.value.slice(start, end)
})

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const data = await getServiceAccounts(props.clusterId, props.namespace)
    serviceAccounts.value = data || []
  } catch (error) {
    Message.error('获取 ServiceAccount 列表失败')
  } finally {
    loading.value = false
  }
}

// 新增
const handleCreate = () => {
  yamlDialogTitle.value = '新增 ServiceAccount'
  yamlContent.value = defaultServiceAccountYaml
  editingItem.value = null
  yamlDialogVisible.value = true
}

// 编辑
const handleEdit = async (row: ServiceAccountInfo) => {
  try {
    const token = localStorage.getItem('token') || ''
    const response: any = await axios.get(
      `/api/v1/plugins/kubernetes/resources/serviceaccounts/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 清理不需要的字段并修复 apiVersion
    const cleanData = cleanK8sResource(response.data.data || response.data)
    // 将返回的JSON对象转换为YAML字符串
    yamlContent.value = yaml.dump(cleanData, {
      indent: 2,
      lineWidth: -1,
      noRefs: true,
      sortKeys: false
    })
    yamlDialogTitle.value = `ServiceAccount YAML - ${row.name}`
    editingItem.value = row
    yamlDialogVisible.value = true
  } catch (error) {
    Message.error('获取 ServiceAccount YAML 失败')
  }
}

// 清理 K8s 资源对象，移除不需要的字段
const cleanK8sResource = (data: any): any => {
  if (!data || typeof data !== 'object') return data

  // 深拷贝避免修改原数据
  const cleaned = JSON.parse(JSON.stringify(data))

  // 移除 metadata 中的不需要字段
  if (cleaned.metadata) {
    delete cleaned.metadata.managedFields
    delete cleaned.metadata.creationTimestamp
    delete cleaned.metadata.resourceVersion
    delete cleaned.metadata.selfLink
    delete cleaned.metadata.uid
    delete cleaned.metadata.generation
  }

  // 修复 apiVersion 空串问题
  if (!cleaned.apiVersion || cleaned.apiVersion === '') {
    // 根据 kind 设置默认 apiVersion
    const kindToApiVersion: Record<string, string> = {
      ServiceAccount: 'v1',
      Role: 'rbac.authorization.k8s.io/v1',
      RoleBinding: 'rbac.authorization.k8s.io/v1',
      ClusterRole: 'rbac.authorization.k8s.io/v1',
      ClusterRoleBinding: 'rbac.authorization.k8s.io/v1',
      PodSecurityPolicy: 'policy/v1beta1'
    }
    cleaned.apiVersion = kindToApiVersion[cleaned.kind] || 'v1'
  }

  return cleaned
}

// 删除
const handleDelete = async (row: ServiceAccountInfo) => {
  try {
    await confirmModal(
      `确定要删除 ServiceAccount "${row.name}" 吗？此操作不可撤销。`,
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )

    const token = localStorage.getItem('token') || ''
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/serviceaccounts/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('删除成功')
    await loadData()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error('删除失败: ' + (error.response?.data?.message || error.message))
    }
  }
}

// 保存 YAML
const handleSaveYaml = async () => {
  yamlSaving.value = true
  try {
    const token = localStorage.getItem('token') || ''
    const namespace = props.namespace || 'default'

    if (editingItem.value) {
      // 编辑模式 - 使用 YAML 更新 API
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/serviceaccounts/${editingItem.value.namespace}/${editingItem.value.name}/yaml`,
        { clusterId: props.clusterId, yaml: yamlContent.value },
        { headers: { Authorization: `Bearer ${token}` } }
      )
      Message.success('更新成功')
    } else {
      // 新增模式
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/serviceaccounts/${namespace}/yaml`,
        yamlContent.value,
        {
          params: { clusterId: props.clusterId },
          headers: {
            'Content-Type': 'application/yaml',
            Authorization: `Bearer ${token}`
          }
        }
      )
      Message.success('创建成功')
    }
    yamlDialogVisible.value = false
    await loadData()
  } catch (error: any) {
    Message.error('保存失败: ' + (error.response?.data?.message || error.message))
  } finally {
    yamlSaving.value = false
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

// 监听 props 变化
watch(() => [props.clusterId, props.namespace], () => {
  if (props.clusterId) {
    loadData()
  }
}, { immediate: true })

// 监听筛选后的数据变化，更新计数
watch(filteredData, (newData) => {
  emit('count-update', newData?.length || 0)
})

// 暴露方法给父组件
defineExpose({
  handleCreate,
  loadData
})
</script>

<style scoped>
.service-accounts-tab {
  width: 100%;
}

.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
}

.search-section {
  flex: 1;
}

.search-input {
  width: 300px;
}

.action-buttons {
  display: flex;
  gap: 12px;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  overflow: hidden;
}

.modern-table {
  width: 100%;
}

.modern-table :deep(.arco-table__row) {
  transition: background-color 0.2s;
}

.modern-table :deep(.arco-table__row:hover) {
  background-color: #f8fafc !important;
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.name-icon-wrapper {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  border: none;
  flex-shrink: 0;
}

.name-icon {
  color: #165dff;
  font-size: 14px;
}

.name-text {
  font-weight: 600;
  color: #303133;
}

.secrets-count {
  color: #606266;
  font-weight: 500;
}

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
  gap: 6px;
}

.label-icon {
  color: #165dff;
  font-size: 18px;
}

.label-count {
  position: absolute;
  top: -6px;
  right: -6px;
  background-color: #165dff;
  color: #000;
  font-size: 10px;
  font-weight: 600;
  min-width: 16px;
  height: 16px;
  line-height: 16px;
  padding: 0 4px;
  border-radius: 8px;
  text-align: center;
  border: 1px solid #165dff;
  z-index: 1;
}

.label-cell:hover .label-icon {
  color: #4080ff;
  transform: scale(1.1);
}

.label-cell:hover .label-count {
  background-color: #4080ff;
}

.action-btn {
  color: #165dff;
  margin: 0 4px;
  padding: 0;
  font-size: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.action-btn.danger {
  color: #f56c6c;
}

.action-btn:hover {
  transform: scale(1.1);
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

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 20px;
  border-top: 1px solid #f0f0f0;
}
</style>
