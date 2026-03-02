<template>
  <div class="role-bindings-tab">
    <div class="table-wrapper">
      <a-table :data="paginatedList" :loading="loading" class="modern-table" :header-cell-style="{ background: '#fafbfc', color: '#606266', fontWeight: '600' }" :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell">
              <div class="name-icon-wrapper"><icon-link /></div>
              <span class="name-text">{{ record.name }}</span>
            </div>
          </template>
          <template #namespace="{ record }"><a-tag size="small" color="gray">{{ record.namespace }}</a-tag></template>
          <template #col_Role="{ record }">
            <span class="role-name">{{ record.roleName }}</span>
          </template>
          <template #col_Users="{ record }">
            <span class="subject-text">{{ getSubjectsByKind(record.subjects, 'User') }}</span>
          </template>
          <template #col_Groups="{ record }">
            <span class="subject-text">{{ getSubjectsByKind(record.subjects, 'Group') }}</span>
          </template>
          <template #col_ServiceAccounts="{ record }">
            <span class="subject-text">{{ getSubjectsByKind(record.subjects, 'ServiceAccount') }}</span>
          </template>
          <template #actions="{ record }">
            <a-button v-permission="'k8s-rolebindings:update'" type="text" @click="handleEdit(record)" class="action-btn">
              <icon-edit />
            </a-button>
            <a-button v-permission="'k8s-rolebindings:delete'" type="text" @click="handleDelete(record)" class="action-btn danger">
              <icon-delete />
            </a-button>
          </template>
        </a-table>
      <div class="pagination-wrapper">
        <a-pagination v-model:current="currentPage" v-model:page-size="pageSize" :page-size-options="[10, 20, 50, 100]" :total="filteredData.length" show-total show-page-size show-jumper />
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
  { title: '名称', slotName: 'name', width: 200, fixed: 'left' },
  { title: '命名空间', dataIndex: 'namespace', slotName: 'namespace', width: 180 },
  { title: 'Role', slotName: 'col_Role', width: 180 },
  { title: 'Users', slotName: 'col_Users', width: 150 },
  { title: 'Groups', slotName: 'col_Groups', width: 150 },
  { title: 'ServiceAccounts', slotName: 'col_ServiceAccounts', width: 200 },
  { title: '存活时间', dataIndex: 'age', width: 140 },
  { title: '操作', slotName: 'actions', width: 100, fixed: 'right', align: 'center' }
]

import { ref, computed, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import { getRoleBindings, createRoleBindingFromYAML, updateRoleBindingFromYAML, deleteRoleBinding, type RoleBindingInfo } from '@/api/kubernetes'
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
const roleBindings = ref<RoleBindingInfo[]>([])
const currentPage = ref(1)
const pageSize = ref(10)

// YAML 编辑
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedRoleBinding = ref<RoleBindingInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 RoleBinding (YAML)'
  }
  return `RoleBinding YAML - ${selectedRoleBinding.value?.name || ''}`
})

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const filteredData = computed(() => {
  let result = roleBindings.value
  if (props.searchName) {
    result = result.filter(item => item.name.toLowerCase().includes(props.searchName!.toLowerCase()))
  }
  return result
})

const paginatedList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredData.value.slice(start, start + pageSize.value)
})

const loadData = async () => {
  loading.value = true
  try {
    const data = await getRoleBindings(props.clusterId, props.namespace)
    roleBindings.value = data || []
  } catch (error) {
    Message.error('获取 RoleBinding 列表失败')
  } finally {
    loading.value = false
  }
}

// 根据 kind 获取 subjects 名称列表
const getSubjectsByKind = (subjects: any[] | undefined, kind: string) => {
  if (!subjects) return '-'
  const filtered = subjects.filter(s => s.kind === kind)
  if (filtered.length === 0) return '-'
  return filtered.map(s => s.name).join(', ')
}

// YAML 创建
const handleCreate = () => {
  isCreateMode.value = true
  selectedRoleBinding.value = null
  // 默认 RoleBinding YAML 模板
  const ns = props.namespace || 'default'
  yamlContent.value = `apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: example-rolebinding
  namespace: ${ns}
subjects:
- kind: ServiceAccount
  name: default
  namespace: ${ns}
roleRef:
  kind: Role
  name: example-role
  apiGroup: rbac.authorization.k8s.io
`
  yamlDialogVisible.value = true
}

// 编辑 YAML
const handleEdit = async (row: RoleBindingInfo) => {
  selectedRoleBinding.value = row
  isCreateMode.value = false

  try {
    const token = localStorage.getItem('token') || ''
    const response: any = await axios.get(
      `/api/v1/plugins/kubernetes/resources/rolebindings/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 清理不需要的字段并修复 apiVersion
    const cleanData = cleanK8sResource(response.data.data)
    // 将返回的JSON对象转换为YAML字符串
    yamlContent.value = yaml.dump(cleanData, {
      indent: 2,
      lineWidth: -1,
      noRefs: true,
      sortKeys: false
    })
    yamlDialogVisible.value = true
  } catch (error: any) {
    Message.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
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

// 删除 RoleBinding
const handleDelete = async (row: RoleBindingInfo) => {
  try {
    await confirmModal(
      `确定要删除 RoleBinding ${row.name} 吗？此操作不可恢复！`,
      '删除 RoleBinding 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )
    // 先执行删除
    await deleteRoleBinding(props.clusterId, row.namespace, row.name)
    Message.success('删除成功')
    // 删除成功后再刷新列表，刷新失败不影响删除结果
    try {
      await loadData()
    } catch (refreshError) {
    }
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(error.response?.data?.message || '删除失败')
    }
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  saving.value = true
  try {
    // 从 YAML 中解析对象
    const yamlObj: any = yaml.load(yamlContent.value)
    if (!yamlObj || !yamlObj.metadata || !yamlObj.metadata.name) {
      Message.error('YAML 中缺少必要的 metadata.name 字段')
      return
    }
    const name = yamlObj.metadata.name
    const namespace = yamlObj.metadata.namespace || props.namespace || 'default'

    if (isCreateMode.value) {
      // 创建模式
      await createRoleBindingFromYAML(props.clusterId, namespace, yamlObj)
      Message.success('创建成功')
    } else {
      // 编辑模式 - 调用更新 API
      await updateRoleBindingFromYAML(props.clusterId, selectedRoleBinding.value!.namespace, selectedRoleBinding.value!.name, yamlObj)
      Message.success('更新成功')
    }

    yamlDialogVisible.value = false
    await loadData()
  } catch (error: any) {
    Message.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    saving.value = false
  }
}

// YAML编辑器滚动处理（同步行号滚动）
const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

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
.role-bindings-tab { width: 100%; }
.search-bar { margin-bottom: 16px; }
.search-input { width: 300px; }

.table-wrapper { background: #fff; border-radius: 8px; overflow: hidden; }
.name-cell { display: flex; align-items: center; gap: 10px; }
.name-icon-wrapper { width: 32px; height: 32px; border-radius: 6px; background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%); display: flex; align-items: center; justify-content: center; border: none; }
.name-icon { color: #165dff; }
.name-text { font-weight: 600; color: #303133; }
.role-name { font-family: monospace; color: #606266; }
.subject-text { font-size: 13px; color: #606266; }
.action-btn { color: #165dff; margin: 0 4px; padding: 0; font-size: 16px; display: inline-flex; align-items: center; justify-content: center; }
.action-btn.danger { color: #f56c6c; }
.action-btn:hover { transform: scale(1.1); }
.pagination-wrapper { display: flex; justify-content: flex-end; padding: 16px 20px; border-top: 1px solid #f0f0f0; }

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
  color: #666;
  padding: 16px 8px;
  text-align: right;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  user-select: none;
  overflow: hidden;
  min-width: 40px;
  border-right: 1px solid #333;
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
