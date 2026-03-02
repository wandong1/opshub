<template>
  <div class="pv-list">
    <div class="search-bar">
      <div class="search-bar-left">
        <a-input v-model="searchName" placeholder="搜索 PV 名称..." allow-clear class="search-input" @input="handleSearch">
          <template #prefix>
            <icon-search />
          </template>
        </a-input>
      </div>

      <div class="search-bar-right">
        <a-button v-permission="'k8s-pv:create'" type="primary" @click="handleCreateYAML">
          <icon-file /> YAML创建
        </a-button>

        <a-button type="primary" @click="loadPVs">
          <icon-refresh /> 刷新
        </a-button>
      </div>
    </div>

    <div class="table-wrapper">
      <a-table :data="filteredPVs" :loading="loading" class="modern-table" :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell">
              <icon-folder />
              <div class="name-text">{{ record.name }}</div>
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
          <template #reclaimPolicy="{ record }">
            {{ formatReclaimPolicy(record.reclaimPolicy) }}
          </template>
          <template #claim="{ record }">
            {{ record.claim || '-' }}
          </template>
          <template #storageClass="{ record }">
            {{ record.storageClass || '-' }}
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="编辑 YAML" placement="top">
                <a-button v-permission="'k8s-pv:update'" type="text" class="action-btn" @click="handleEditYAML(record)">
                  <icon-file />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button v-permission="'k8s-pv:delete'" type="text" class="action-btn danger" @click="handleDelete(record)">
                  <icon-delete />
                </a-button>
              </a-tooltip>
            </div>
          </template>
        </a-table>
    </div>

    <a-modal v-model:visible="yamlDialogVisible" :title="`PV YAML - ${selectedPV?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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
    <a-modal v-model:visible="createYamlDialogVisible" title="YAML 创建 PV" width="900px" :lock-scroll="false" class="yaml-dialog">
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
  { title: '回收策略', dataIndex: 'reclaimPolicy', slotName: 'reclaimPolicy', width: 140 },
  { title: '声明', dataIndex: 'claim', slotName: 'claim', width: 180 },
  { title: '存储类', dataIndex: 'storageClass', slotName: 'storageClass', width: 150 },
  { title: '存活时间', dataIndex: 'age', width: 100 },
  { title: '操作', slotName: 'actions', width: 160, fixed: 'right', align: 'center' }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { load, dump } from 'js-yaml'
import {
  getPersistentVolumes,
  getPersistentVolumeYAML,
  updatePersistentVolumeYAML,
  createPersistentVolumeYAML,
  deletePersistentVolume,
  type PVInfo
} from '@/api/kubernetes'

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['refresh', 'count-update'])

const loading = ref(false)
const saving = ref(false)
const pvList = ref<PVInfo[]>([])
const searchName = ref('')
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedPV = ref<PVInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null)

// YAML 创建相关
const createYamlDialogVisible = ref(false)
const creating = ref(false)
const createYamlContent = ref('')
const createYamlTextarea = ref<HTMLTextAreaElement | null>(null)

const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const createYamlLineCount = computed(() => {
  if (!createYamlContent.value) return 1
  return createYamlContent.value.split('\n').length
})

const filteredPVs = computed(() => {
  let result = pvList.value
  if (searchName.value) {
    result = result.filter(p => p.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  return result
})

const loadPVs = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getPersistentVolumes(props.clusterId)
    pvList.value = data || []
    if (showSuccess) {
      Message.success('刷新成功')
    }
  } catch (error) {
    Message.error('获取 PV 列表失败')
  } finally {
    loading.value = false
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

const formatReclaimPolicy = (policy: string) => {
  const policyMap: Record<string, string> = {
    'Retain': '保留',
    'Delete': '删除',
    'Recycle': '回收'
  }
  return policyMap[policy] || policy
}

const getStatusTagType = (status: string) => {
  const map: Record<string, string> = {
    'Available': 'success',
    'Bound': 'warning',
    'Released': 'info',
    'Failed': 'danger'
  }
  return map[status] || 'info'
}

const handleCreateYAML = () => {
  createYamlContent.value = `apiVersion: v1
kind: PersistentVolume
metadata:
  name: my-pv
spec:
  capacity:
    storage: 10Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  storageClassName: local-storage
  local:
    path: /mnt/data
  nodeAffinity:
    required:
      nodeSelectorTerms:
        - matchExpressions:
            - key: kubernetes.io/hostname
              operator: In
              values:
                - node-1
`
  createYamlDialogVisible.value = true
}

const handleEditYAML = async (pv: PVInfo) => {
  if (!props.clusterId) return
  selectedPV.value = pv
  try {
    const response = await getPersistentVolumeYAML(props.clusterId, pv.name)
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
  if (!props.clusterId || !selectedPV.value) return

  saving.value = true
  try {
    let jsonData
    try {
      jsonData = yamlToJson(yamlContent.value)
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedPV.value) {
        jsonData.metadata.name = selectedPV.value.name
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'PersistentVolume'
      }
    } catch (e) {
      Message.error('YAML 格式错误，请检查缩进和语法')
      saving.value = false
      return
    }

    await updatePersistentVolumeYAML(
      props.clusterId,
      selectedPV.value.name,
      jsonData
    )
    Message.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadPVs()
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

const handleSaveCreateYAML = async () => {
  if (!props.clusterId) return

  creating.value = true
  try {
    const jsonData = yamlToJson(createYamlContent.value)
    if (!jsonData.apiVersion) {
      jsonData.apiVersion = 'v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'PersistentVolume'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    await createPersistentVolumeYAML(
      props.clusterId,
      jsonData
    )
    Message.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadPVs()
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

const handleDelete = async (pv: PVInfo) => {
  if (!props.clusterId) return
  try {
    await confirmModal(`确定要删除 PV ${pv.name} 吗？`, '删除确认', { type: 'error' })
    await deletePersistentVolume(props.clusterId, pv.name)
    Message.success('删除成功')
    emit('refresh')
    await loadPVs()
  } catch (error) {
    if (error !== 'cancel') {
      Message.error('删除失败')
    }
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
  loadPVs()
})

// 监听筛选后的数据变化，更新计数
watch(filteredPVs, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  loadPVs()
})

defineExpose({
  loadData: () => loadPVs(true)
})
</script>

<style scoped>
.pv-list {
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

.table-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.name-icon {
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
  border: none;
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
  background-color: #fafafa;
  padding: 24px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
