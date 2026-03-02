<template>
  <div class="endpoints-list">
    <div class="search-bar">
      <div class="search-bar-left">
        <a-input v-model="searchName" placeholder="搜索 Endpoints 名称..." allow-clear class="search-input" @input="handleSearch">
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
        <a-button v-permission="'k8s-endpoints:create'" type="primary" @click="handleCreateYAML">
          <icon-file /> YAML创建
        </a-button>
      </div>
    </div>

    <div class="table-wrapper">
      <a-table :data="filteredEndpoints" :loading="loading" class="modern-table" size="default" :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell">
              <icon-link />
              <div>
                <div class="name-text">{{ record.name }}</div>
                <div class="namespace-text">{{ record.namespace }}</div>
              </div>
            </div>
          </template>
          <template #col_6900="{ record }">
            <div v-if="record.subsets.length > 0">
              <div v-for="(subset, idx) in record.subsets" :key="idx" class="subset-item">
                <a-tag size="small" color="green" class="endpoint-tag">
                  {{ subset.addresses.length }} 就绪
                </a-tag>
                <a-tag v-if="subset.notReadyAddresses.length > 0" size="small" color="orangered" class="endpoint-tag">
                  {{ subset.notReadyAddresses.length }} 未就绪
                </a-tag>
                <div class="ports-display">
                  {{ subset.ports.map(p => `${p.port}/${p.protocol}`).join(', ') }}
                </div>
              </div>
            </div>
            <a-tag v-else color="gray" size="small">无端点</a-tag>
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="编辑 YAML" placement="top">
                <a-button v-permission="'k8s-endpoints:update'" type="text" class="action-btn" @click="handleEditYAML(record)">
                  <icon-file />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button v-permission="'k8s-endpoints:delete'" type="text" class="action-btn danger" @click="handleDelete(record)">
                  <icon-delete />
                </a-button>
              </a-tooltip>
            </div>
          </template>
        </a-table>
    </div>

    <a-modal v-model:visible="yamlDialogVisible" :title="`Endpoints YAML - ${selectedEndpoint?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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

    <a-modal v-model:visible="detailDialogVisible" :title="`Endpoints 详情 - ${selectedEndpoint?.name}`" width="800px">
      <div v-if="selectedEndpoint">
        <div v-for="(subset, idx) in selectedEndpoint.subsets" :key="idx" class="detail-subset">
          <h4>Subset {{ idx + 1 }}</h4>
          <div><strong>就绪地址:</strong></div>
          <div v-for="(addr, i) in subset.addresses" :key="i" class="address-item">
            {{ addr.ip }} <span v-if="addr.targetRef">({{ addr.targetRef }})</span>
          </div>
          <div v-if="!subset.addresses.length">无</div>

          <div style="margin-top: 10px;"><strong>未就绪地址:</strong></div>
          <div v-for="(addr, i) in subset.notReadyAddresses" :key="i" class="address-item">
            {{ addr.ip }} <span v-if="addr.targetRef">({{ addr.targetRef }})</span>
          </div>
          <div v-if="!subset.notReadyAddresses.length">无</div>

          <div style="margin-top: 10px;"><strong>端口:</strong></div>
          <div>{{ subset.ports.map(p => `${p.name || '-'}: ${p.port}/${p.protocol}`).join(', ') || '-' }}</div>
        </div>
      </div>
      <template #footer>
        <a-button @click="detailDialogVisible = false">关闭</a-button>
      </template>
    </a-modal>

    <!-- YAML 创建弹窗 -->
    <a-modal v-model:visible="createYamlDialogVisible" title="YAML 创建 Endpoints" width="900px" :lock-scroll="false" class="yaml-dialog">
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
  { title: '名称', dataIndex: 'name', slotName: 'name', width: 200 },
  { title: '端点', slotName: 'col_6900', width: 300 },
  { title: '存活时间', dataIndex: 'age', width: 120 },
  { title: '操作', slotName: 'actions', width: 120, fixed: 'right', align: 'center' }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { getEndpoints, getEndpointsDetail, createEndpointYAML, getEndpointYAML, updateEndpointYAML, deleteEndpoint, getNamespaces, type EndpointsInfo } from '@/api/kubernetes'
import { load, dump } from 'js-yaml'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['refresh', 'count-update'])

const loading = ref(false)
const endpointsList = ref<EndpointsInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterNamespace = ref('')
const detailDialogVisible = ref(false)
const selectedEndpoint = ref<EndpointsInfo | null>(null)

// YAML 编辑相关
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null)
const saving = ref(false)

// YAML 创建相关
const createYamlDialogVisible = ref(false)
const creating = ref(false)
const createYamlContent = ref('')
const createYamlTextarea = ref<HTMLTextAreaElement | null>(null)

// 使用 js-yaml 库解析 YAML
const yamlToJson = (yaml: string): any => {
  try {
    return load(yaml)
  } catch (error) {
    throw error
  }
}

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const createYamlLineCount = computed(() => {
  if (!createYamlContent.value) return 1
  return createYamlContent.value.split('\n').length
})

const filteredEndpoints = computed(() => {
  let result = endpointsList.value
  if (searchName.value) {
    result = result.filter(e => e.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterNamespace.value) {
    result = result.filter(e => e.namespace === filterNamespace.value)
  }
  return result
})

const loadEndpoints = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getEndpoints(props.clusterId, props.namespace || undefined)
    endpointsList.value = data || []
    if (showSuccess) {
      Message.success('刷新成功')
    }
  } catch (error) {
    Message.error('获取 Endpoints 列表失败')
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

const handleDetail = (endpoint: EndpointsInfo) => {
  selectedEndpoint.value = endpoint
  detailDialogVisible.value = true
}

const handleEditYAML = async (endpoint: EndpointsInfo) => {
  if (!props.clusterId) return
  selectedEndpoint.value = endpoint
  try {
    const response = await getEndpointYAML(props.clusterId, endpoint.namespace, endpoint.name)
    // 保存原始 JSON 数据
    originalJsonData.value = response
    // 使用 js-yaml 转换为 YAML 格式
    const yaml = dump(originalJsonData.value, { indent: 2, lineWidth: -1 })
    yamlContent.value = yaml
    yamlDialogVisible.value = true
  } catch (error) {
    Message.error('获取 YAML 失败')
  }
}

const handleDelete = async (endpoint: EndpointsInfo) => {
  if (!props.clusterId) return
  try {
    await confirmModal(`确定要删除 Endpoint ${endpoint.name} 吗？`, '删除确认', { type: 'error' })
    await deleteEndpoint(props.clusterId, endpoint.namespace, endpoint.name)
    Message.success('删除成功')
    emit('refresh')
    await loadEndpoints()
  } catch (error) {
    if (error !== 'cancel') {
      Message.error('删除失败')
    }
  }
}

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedEndpoint.value) return

  saving.value = true
  try {
    // 尝试将 YAML 转回 JSON
    let jsonData
    try {
      jsonData = yamlToJson(yamlContent.value)
      // 确保基本的元数据存在
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedEndpoint.value) {
        jsonData.metadata.name = selectedEndpoint.value.name
      }
      if (!jsonData.metadata.namespace && selectedEndpoint.value) {
        jsonData.metadata.namespace = selectedEndpoint.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'Endpoints'
      }
    } catch (e) {
      Message.error('YAML 格式错误，请检查缩进和语法')
      saving.value = false
      return
    }

    await updateEndpointYAML(
      props.clusterId,
      selectedEndpoint.value.namespace,
      selectedEndpoint.value.name,
      jsonData
    )
    Message.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadEndpoints()
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

const handleCreateYAML = () => {
  const defaultNamespace = props.namespace || 'default'
  // 设置默认 YAML 模板
  createYamlContent.value = `apiVersion: v1
kind: Endpoints
metadata:
  name: my-endpoints
  namespace: ${defaultNamespace}
subsets:
  - addresses:
      - ip: 192.168.1.1
    ports:
      - port: 80
        protocol: TCP
        name: http
`
  createYamlDialogVisible.value = true
}

const handleSaveCreateYAML = async () => {
  if (!props.clusterId) return

  creating.value = true
  try {
    const jsonData = yamlToJson(createYamlContent.value)
    // 确保基本的元数据存在
    if (!jsonData.apiVersion) {
      jsonData.apiVersion = 'v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'Endpoints'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    // 从 YAML 中提取命名空间
    const namespace = jsonData.metadata.namespace || props.namespace || 'default'
    jsonData.metadata.namespace = namespace

    await createEndpointYAML(
      props.clusterId,
      namespace,
      jsonData
    )
    Message.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadEndpoints()
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
  const lineNumbers = document.querySelector('.yaml-line-numbers') as HTMLElement
  if (lineNumbers) {
    lineNumbers.scrollTop = target.scrollTop
  }
}

watch(() => props.clusterId, () => {
  loadEndpoints()
  loadNamespaces()
})

watch(() => props.namespace, () => {
  filterNamespace.value = props.namespace || ''
  loadEndpoints()
})

// 监听筛选后的数据变化，更新计数
watch(filteredEndpoints, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  loadEndpoints()
  loadNamespaces()
})

// 暴露方法给父组件
defineExpose({
  loadData: () => loadEndpoints(true)
})
</script>

<style scoped>
.endpoints-list {
  width: 100%;
}


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
  width: 180px;
}

.search-icon {
  color: #165dff;
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
  border: 1px solid #e5e6eb;
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

.subset-item {
  margin-bottom: 8px;
}

.endpoint-tag {
  margin-right: 4px;
}

.ports-display {
  font-size: 12px;
  color: #606266;
  margin-top: 4px;
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
  transition: all 0.3s;
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

.detail-subset {
  margin-bottom: 20px;
  padding-bottom: 20px;
  border-bottom: 1px solid #eee;
}

.detail-subset:last-child {
  border-bottom: none;
}

.address-item {
  font-size: 13px;
  color: #606266;
  padding: 4px 0;
}

/* YAML 编辑弹窗 */
.yaml-editor-wrapper {
  display: flex;
  border: 1px solid #e5e6eb;
  border-radius: 6px;
  overflow: hidden;
  background-color: #1e1e1e;
}

.yaml-line-numbers {
  background-color: #0d0d0d;
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
  background-color: #1e1e1e;
  color: #d4d4d4;
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
  padding: 0;
  background-color: #1a1a1a;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
