<template>
  <div class="pvc-list">
    <div class="search-bar">
      <div class="search-bar-left">
        <a-input v-model="searchName" placeholder="搜索 PVC 名称..." allow-clear class="search-input" @input="handleSearch">
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
        <a-button v-permission="'k8s-pvc:create'" type="primary" @click="handleCreateYAML">
          <icon-file /> YAML创建
        </a-button>
      </div>
    </div>

    <div class="table-wrapper">
      <a-table :data="filteredPVCs" :loading="loading" class="modern-table" :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <icon-common />
              </div>
              <div>
                <div class="name-text">{{ record.name }}</div>
                <div class="namespace-text">{{ record.namespace }}</div>
              </div>
            </div>
          </template>
          <template #status="{ record }">
            <a-tag :type="getStatusTagType(record.status)" size="small">{{ record.status }}</a-tag>
          </template>
          <template #accessModes="{ record }">
            <div v-for="mode in record.accessModes" :key="mode" class="access-mode-item">
              {{ formatAccessMode(mode) }}
            </div>
          </template>
          <template #storageClass="{ record }">
            {{ record.storageClass || '-' }}
          </template>
          <template #volumeName="{ record }">
            {{ record.volumeName || '-' }}
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="编辑 YAML" placement="top">
                <a-button v-permission="'k8s-pvc:update'" type="text" class="action-btn" @click="handleEditYAML(record)">
                  <icon-file />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button v-permission="'k8s-pvc:delete'" type="text" class="action-btn danger" @click="handleDelete(record)">
                  <icon-delete />
                </a-button>
              </a-tooltip>
            </div>
          </template>
        </a-table>
    </div>

    <a-modal v-model:visible="yamlDialogVisible" :title="`PVC YAML - ${selectedPVC?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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
          <a-button @click="yamlDialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleSaveYAML" :loading="saving">保存</a-button>
        </div>
      </template>
    </a-modal>

    <!-- YAML 创建弹窗 -->
    <a-modal v-model:visible="createYamlDialogVisible" title="YAML 创建 PVC" width="900px" :lock-scroll="false" class="yaml-dialog">
      <div class="yaml-editor-wrapper">
        <div class="yaml-line-numbers">
          <div v-for="line in createYamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="createYamlContent"
          class="yaml-textarea"
          spellcheck="false"
          @input="handleCreateYamlInput"
          @scroll="handleCreateYamlScroll"
          ref="createYamlTextarea"
        ></textarea>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="createYamlDialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleSaveCreateYAML" :loading="creating">创建</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns = [
  { title: '名称', dataIndex: 'name', slotName: 'name', width: 280 },
  { title: '状态', dataIndex: 'status', slotName: 'status', width: 120 },
  { title: '容量', dataIndex: 'capacity', width: 120 },
  { title: '访问模式', slotName: 'accessModes', width: 140 },
  { title: '存储类', dataIndex: 'storageClass', slotName: 'storageClass', width: 150 },
  { title: '卷名称', dataIndex: 'volumeName', slotName: 'volumeName', width: 180 },
  { title: '存活时间', dataIndex: 'age', width: 100 },
  { title: '操作', slotName: 'actions', width: 160, fixed: 'right', align: 'center' }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { load, dump } from 'js-yaml'
import {
  getPersistentVolumeClaims,
  getPersistentVolumeClaimYAML,
  updatePersistentVolumeClaimYAML,
  createPersistentVolumeClaimYAML,
  deletePersistentVolumeClaim,
  getNamespaces,
  type PVCInfo
} from '@/api/kubernetes'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['refresh', 'count-update'])

const loading = ref(false)
const saving = ref(false)
const pvcList = ref<PVCInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterNamespace = ref('')
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedPVC = ref<PVCInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null)

// YAML 创建相关
const createYamlDialogVisible = ref(false)
const creating = ref(false)
const createYamlContent = ref('')
const createYamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const createYamlLineCount = computed(() => {
  if (!createYamlContent.value) return 1
  return createYamlContent.value.split('\n').length
})

const filteredPVCs = computed(() => {
  let result = pvcList.value
  if (searchName.value) {
    result = result.filter(p => p.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterNamespace.value) {
    result = result.filter(p => p.namespace === filterNamespace.value)
  }
  return result
})

const loadPVCs = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getPersistentVolumeClaims(props.clusterId, props.namespace || undefined)
    pvcList.value = data || []
    if (showSuccess) {
      Message.success('刷新成功')
    }
  } catch (error) {
    Message.error('获取 PVC 列表失败')
  } finally {
    loading.value = false
  }
}

const loadNamespaces = async () => {
  if (!props.clusterId) return
  try {
    const data = await getNamespaces(props.clusterId)
    namespaces.value = data || []
  } catch (error) {
  }
}

const handleSearch = () => {
  // 本地过滤
}

const formatAccessMode = (mode: string) => {
  const modeMap: Record<string, string> = {
    'ReadWriteOnce': 'RWO',
    'ReadOnlyMany': 'ROX',
    'ReadWriteMany': 'RWX',
    'ReadWriteOncePod': 'RWOP'
  }
  return modeMap[mode] || mode
}

const getStatusTagType = (status: string) => {
  const map: Record<string, string> = {
    'Bound': 'success',
    'Pending': 'warning',
    'Lost': 'danger',
    'Released': 'info'
  }
  return map[status] || 'info'
}

const handleCreateYAML = () => {
  const defaultNamespace = props.namespace || 'default'
  createYamlContent.value = `apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: my-pvc
  namespace: ${defaultNamespace}
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
  storageClassName: standard
`
  createYamlDialogVisible.value = true
}

const handleEditYAML = async (pvc: PVCInfo) => {
  if (!props.clusterId) return
  selectedPVC.value = pvc
  try {
    const response = await getPersistentVolumeClaimYAML(props.clusterId, pvc.namespace, pvc.name)
    originalJsonData.value = response
    const yaml = dump(originalJsonData.value, { indent: 2, lineWidth: -1 })
    yamlContent.value = yaml
    yamlDialogVisible.value = true
  } catch (error) {
    Message.error('获取 YAML 失败')
  }
}


const yamlToJson = (yaml: string): any => {
  try {
    return load(yaml)
  } catch (error) {
    throw error
  }
}

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedPVC.value) return

  saving.value = true
  try {
    let jsonData
    try {
      jsonData = yamlToJson(yamlContent.value)
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedPVC.value) {
        jsonData.metadata.name = selectedPVC.value.name
      }
      if (!jsonData.metadata.namespace && selectedPVC.value) {
        jsonData.metadata.namespace = selectedPVC.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'PersistentVolumeClaim'
      }
    } catch (e) {
      Message.error('YAML 格式错误，请检查缩进和语法')
      saving.value = false
      return
    }

    await updatePersistentVolumeClaimYAML(
      props.clusterId,
      selectedPVC.value.namespace,
      selectedPVC.value.name,
      jsonData
    )
    Message.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadPVCs()
  } catch (error) {
    Message.error('保存失败')
  } finally {
    saving.value = false
  }
}

const handleYamlInput = () => {
  // 处理输入
}

const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

const handleDelete = async (pvc: PVCInfo) => {
  if (!props.clusterId) return
  try {
    await confirmModal(`确定要删除 PVC ${pvc.name} 吗？`, '删除确认', { type: 'error' })
    await deletePersistentVolumeClaim(props.clusterId, pvc.namespace, pvc.name)
    Message.success('删除成功')
    emit('refresh')
    await loadPVCs()
  } catch (error) {
    if (error !== 'cancel') {
      Message.error('删除失败')
    }
  }
}

const handleSaveCreateYAML = async () => {
  if (!props.clusterId) return

  creating.value = true
  try {
    const jsonData = yamlToJson(createYamlContent.value)
    if (!jsonData.apiVersion) {
      jsonData.apiVersion = 'v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'PersistentVolumeClaim'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    const namespace = jsonData.metadata.namespace || props.namespace || 'default'
    jsonData.metadata.namespace = namespace

    await createPersistentVolumeClaimYAML(
      props.clusterId,
      namespace,
      jsonData
    )
    Message.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadPVCs()
  } catch (error) {
    Message.error('创建失败')
  } finally {
    creating.value = false
  }
}

const handleCreateYamlInput = () => {
  // 处理输入
}

const handleCreateYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.create-yaml .yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

// 修复页面偏移
watch(yamlDialogVisible, (val) => {
  if (val) {
    const scrollBarWidth = window.innerWidth - document.documentElement.clientWidth
    if (scrollBarWidth > 0) {
      document.body.style.paddingRight = `${scrollBarWidth}px`
    }
  } else {
    document.body.style.paddingRight = ''
  }
})

watch(() => props.clusterId, () => {
  loadPVCs()
  loadNamespaces()
})

watch(() => props.namespace, () => {
  filterNamespace.value = props.namespace || ''
  loadPVCs()
})

// 监听筛选后的数据变化，更新计数
watch(filteredPVCs, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  loadPVCs()
  loadNamespaces()
})

defineExpose({
  loadData: () => loadPVCs(true)
})
</script>

<style scoped>
.pvc-list {
  width: 100%;
}

.search-bar {
  display: flex;
  justify-content: space-between;
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
  width: 180px;
}

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}

.name-icon-wrapper {
  width: 36px;
  height: 36px;
  background: linear-gradient(135deg, #e8f3ff 0%, #d6e8ff 100%);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #165dff;
  font-size: 18px;
  flex-shrink: 0;
}

.name-text {
  font-weight: 500;
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

.access-mode-item {
  font-size: 12px;
  padding: 2px 6px;
  background: #e8f3ff;
  border-radius: 3px;
  color: #165dff;
  margin-bottom: 4px;
  display: inline-block;
}

.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  color: #165dff;
  transition: all 0.3s;
  padding: 0;
  font-size: 16px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
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

.yaml-dialog :deep(.arco-dialog__body) {
  padding: 24px;
  background-color: #fafafa;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
