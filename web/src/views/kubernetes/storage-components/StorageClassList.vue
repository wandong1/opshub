<template>
  <div class="storageclass-list">
    <div class="search-bar">
      <div class="search-bar-left">
        <a-input v-model="searchName" placeholder="搜索 StorageClass 名称..." allow-clear class="search-input" @input="handleSearch">
          <template #prefix>
            <icon-search />
          </template>
        </a-input>
      </div>

      <div class="search-bar-right">
        <a-button v-permission="'k8s-storageclasses:create'" type="primary" @click="handleCreateYAML">
          <icon-file /> YAML创建
        </a-button>

        <a-button type="primary" @click="loadStorageClasses">
          <icon-refresh /> 刷新
        </a-button>
      </div>
    </div>

    <div class="table-wrapper">
      <a-table :data="filteredStorageClasses" :loading="loading" class="modern-table" :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell">
              <icon-storage />
              <div class="name-text">{{ record.name }}</div>
            </div>
          </template>
          <template #reclaimPolicy="{ record }">
            {{ formatReclaimPolicy(record.reclaimPolicy) }}
          </template>
          <template #volumeBindingMode="{ record }">
            {{ formatVolumeBindingMode(record.volumeBindingMode) }}
          </template>
          <template #allowExpansion="{ record }">
            <a-tag :type="record.allowVolumeExpansion ? 'success' : 'info'" size="small">
              {{ record.allowVolumeExpansion ? '是' : '否' }}
            </a-tag>
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="编辑 YAML" placement="top">
                <a-button v-permission="'k8s-storageclasses:update'" type="text" class="action-btn" @click="handleEditYAML(record)">
                  <icon-file />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button v-permission="'k8s-storageclasses:delete'" type="text" class="action-btn danger" @click="handleDelete(record)">
                  <icon-delete />
                </a-button>
              </a-tooltip>
            </div>
          </template>
        </a-table>
    </div>

    <a-modal v-model:visible="yamlDialogVisible" :title="`StorageClass YAML - ${selectedStorageClass?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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
    <a-modal v-model:visible="createYamlDialogVisible" title="YAML 创建 StorageClass" width="900px" :lock-scroll="false" class="yaml-dialog">
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
  { title: 'Provisioner', dataIndex: 'provisioner', width: 250 },
  { title: '回收策略', dataIndex: 'reclaimPolicy', slotName: 'reclaimPolicy', width: 120 },
  { title: '绑定模式', dataIndex: 'volumeBindingMode', slotName: 'volumeBindingMode', width: 140 },
  { title: '允许卷扩展', slotName: 'allowExpansion', width: 120, align: 'center' },
  { title: '存活时间', dataIndex: 'age', width: 100 },
  { title: '操作', slotName: 'actions', width: 160, fixed: 'right', align: 'center' }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { load, dump } from 'js-yaml'
import {
  getStorageClasses,
  getStorageClassYAML,
  updateStorageClassYAML,
  createStorageClassYAML,
  deleteStorageClass,
  type StorageClassInfo
} from '@/api/kubernetes'

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['refresh', 'count-update'])

const loading = ref(false)
const saving = ref(false)
const storageClassList = ref<StorageClassInfo[]>([])
const searchName = ref('')
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedStorageClass = ref<StorageClassInfo | null>(null)
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

const filteredStorageClasses = computed(() => {
  let result = storageClassList.value
  if (searchName.value) {
    result = result.filter(s => s.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  return result
})

const loadStorageClasses = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getStorageClasses(props.clusterId)
    storageClassList.value = data || []
    if (showSuccess) {
      Message.success('刷新成功')
    }
  } catch (error) {
    Message.error('获取 StorageClass 列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  // 本地过滤
}

const formatReclaimPolicy = (policy: string) => {
  const policyMap: Record<string, string> = {
    'Retain': '保留',
    'Delete': '删除'
  }
  return policyMap[policy] || policy
}

const formatVolumeBindingMode = (mode: string) => {
  const modeMap: Record<string, string> = {
    'Immediate': '立即绑定',
    'WaitForFirstConsumer': '等待消费者'
  }
  return modeMap[mode] || mode
}

const handleCreateYAML = () => {
  createYamlContent.value = `apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard
provisioner: kubernetes.io/aws-ebs
parameters:
  type: gp2
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
`
  createYamlDialogVisible.value = true
}

const handleEditYAML = async (sc: StorageClassInfo) => {
  if (!props.clusterId) return
  selectedStorageClass.value = sc
  try {
    const response = await getStorageClassYAML(props.clusterId, sc.name)
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
  if (!props.clusterId || !selectedStorageClass.value) return

  saving.value = true
  try {
    let jsonData
    try {
      jsonData = yamlToJson(yamlContent.value)
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedStorageClass.value) {
        jsonData.metadata.name = selectedStorageClass.value.name
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'storage.k8s.io/v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'StorageClass'
      }
    } catch (e) {
      Message.error('YAML 格式错误，请检查缩进和语法')
      saving.value = false
      return
    }

    await updateStorageClassYAML(
      props.clusterId,
      selectedStorageClass.value.name,
      jsonData
    )
    Message.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadStorageClasses()
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
      jsonData.apiVersion = 'storage.k8s.io/v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'StorageClass'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    await createStorageClassYAML(
      props.clusterId,
      jsonData
    )
    Message.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadStorageClasses()
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

const handleDelete = async (sc: StorageClassInfo) => {
  if (!props.clusterId) return
  try {
    await confirmModal(`确定要删除 StorageClass ${sc.name} 吗？`, '删除确认', { type: 'error' })
    await deleteStorageClass(props.clusterId, sc.name)
    Message.success('删除成功')
    emit('refresh')
    await loadStorageClasses()
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
  loadStorageClasses()
})

// 监听筛选后的数据变化，更新计数
watch(filteredStorageClasses, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  loadStorageClasses()
})

defineExpose({
  loadData: () => loadStorageClasses(true)
})
</script>

<style scoped>
.storageclass-list {
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
