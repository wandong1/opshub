<template>
  <div class="service-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <a-input
          v-model="searchName"
          placeholder="搜索服务名称..."
          clearable
          class="search-input"
          @input="handleSearch"
        >
          <template #prefix>
            <icon-search />
          </template>
        </a-input>

        <a-select v-model="filterType" placeholder="服务类型" allow-clear @change="handleSearch" class="filter-select">
          <a-option label="全部" value="" />
          <a-option label="ClusterIP" value="ClusterIP" />
          <a-option label="NodePort" value="NodePort" />
          <a-option label="LoadBalancer" value="LoadBalancer" />
        </a-select>

        <a-select v-model="filterNamespace" placeholder="命名空间" allow-clear @change="handleSearch" class="filter-select">
          <a-option label="全部" value="" />
          <a-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
        </a-select>
      </div>

      <div class="search-bar-right">
        <a-button v-permission="'k8s-services:create'" type="primary" @click="handleCreate">创建服务</a-button>
        <a-button v-permission="'k8s-services:create'" type="primary" @click="handleCreateYAML">
          <icon-file /> YAML创建
        </a-button>
      </div>
    </div>

    <!-- 服务列表 -->
    <div class="table-wrapper">
      <a-table
        :data="paginatedServices"
        :loading="loading"
        class="modern-table"
        size="default"
       :columns="tableColumns">
          <template #name="{ record }">
            <div class="name-cell" @click="handleShowDetail(record)" style="cursor: pointer;">
              <icon-link />
              <div>
                <div class="name-text">{{ record.name }}</div>
                <div class="namespace-text">{{ record.namespace }}</div>
              </div>
            </div>
          </template>
          <template #type="{ record }">
            <a-tag :type="getTypeTagType(record.type)" size="small">{{ record.type }}</a-tag>
          </template>
          <template #externalIP="{ record }">
            {{ record.externalIP || '-' }}
          </template>
          <template #ports="{ record }">
            <div v-for="port in record.ports" :key="port.port" class="port-item">
              {{ port.protocol }}: {{ port.port }}
              <span v-if="port.targetPort">→ {{ port.targetPort }}</span>
              <span v-if="port.nodePort"> ({{ port.nodePort }})</span>
            </div>
          </template>
          <template #endpoints="{ record }">
            <a-tag v-if="record.endpoints > 0" color="green" size="small">{{ record.endpoints }}</a-tag>
            <a-tag v-else color="gray" size="small">0</a-tag>
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="编辑 YAML" placement="top">
                <a-button v-permission="'k8s-services:update'" type="text" class="action-btn" @click="handleEditYAML(record)">
                  <icon-file />
                </a-button>
              </a-tooltip>
              <a-tooltip content="编辑" placement="top">
                <a-button v-permission="'k8s-services:update'" type="text" class="action-btn" @click="handleEdit(record)">
                  <icon-edit />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button v-permission="'k8s-services:delete'" type="text" class="action-btn danger" @click="handleDelete(record)">
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
          :page-size-options="[10, 20, 50]"
          :total="filteredServices.length"
          layout="total, sizes, prev, pager, next"
        />
      </div>
    </div>

    <!-- YAML 弹窗 -->
    <a-modal v-model:visible="yamlDialogVisible" :title="`Service YAML - ${selectedService?.name}`" width="900px" :lock-scroll="false" class="yaml-dialog">
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
    <a-modal v-model:visible="createYamlDialogVisible" title="YAML 创建 Service" width="900px" :lock-scroll="false" class="yaml-dialog">
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

    <!-- 编辑对话框 -->
    <ServiceEditDialog
      ref="editDialogRef"
      :clusterId="clusterId"
      @success="handleEditSuccess"
    />

    <!-- Service详情对话框 -->
    <ServiceDetailDialog
      ref="detailDialogRef"
      :clusterId="clusterId"
      @terminal="handleTerminal"
      @logs="handleLogs"
    />
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns = [
  { title: '名称', dataIndex: 'name', slotName: 'name', width: 180 },
  { title: '类型', dataIndex: 'type', slotName: 'type', width: 130 },
  { title: 'Cluster IP', dataIndex: 'clusterIP', width: 140 },
  { title: '外部 IP', dataIndex: 'externalIP', slotName: 'externalIP', width: 140 },
  { title: '端口', slotName: 'ports', width: 200 },
  { title: '端点', dataIndex: 'endpoints', slotName: 'endpoints', width: 80, align: 'center' },
  { title: '存活时间', dataIndex: 'age', width: 120 },
  { title: '操作', slotName: 'actions', width: 160, fixed: 'right', align: 'center' }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { load, dump } from 'js-yaml'
import { getServices, getServiceYAML, updateServiceYAML, createServiceYAML, deleteService, getNamespaces, type ServiceInfo } from '@/api/kubernetes'
import ServiceEditDialog from './ServiceEditDialog.vue'
import ServiceDetailDialog from './ServiceDetailDialog.vue'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update', 'terminal', 'logs'])

const loading = ref(false)
const saving = ref(false)
const serviceList = ref<ServiceInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterType = ref('')
const filterNamespace = ref('')
const currentPage = ref(1)
const pageSize = ref(10)
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedService = ref<ServiceInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null) // 保存原始 JSON 数据
const editDialogRef = ref<any>(null)
const detailDialogRef = ref<any>(null)

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

const filteredServices = computed(() => {
  let result = serviceList.value
  if (searchName.value) {
    result = result.filter(s => s.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterType.value) {
    result = result.filter(s => s.type === filterType.value)
  }
  if (filterNamespace.value) {
    result = result.filter(s => s.namespace === filterNamespace.value)
  }
  return result
})

const paginatedServices = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredServices.value.slice(start, end)
})

const loadServices = async (showSuccess = false) => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getServices(props.clusterId, props.namespace || undefined)
    serviceList.value = data || []
    if (showSuccess) {
      Message.success('刷新成功')
    }
  } catch (error) {
    Message.error('获取服务列表失败')
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
  currentPage.value = 1
}

const getTypeTagType = (type: string) => {
  const map: Record<string, string> = {
    ClusterIP: 'success',
    NodePort: 'warning',
    LoadBalancer: 'danger'
  }
  return map[type] || 'info'
}

const handleCreate = () => {
  editDialogRef.value?.openCreate(namespaces.value)
}

const handleCreateYAML = () => {
  const defaultNamespace = props.namespace || 'default'
  // 设置默认 YAML 模板
  createYamlContent.value = `apiVersion: v1
kind: Service
metadata:
  name: my-service
  namespace: ${defaultNamespace}
spec:
  type: ClusterIP
  selector:
    app: my-app
  ports:
    - protocol: TCP
      port: 80
      targetPort: 8080
`
  createYamlDialogVisible.value = true
}

const handleEdit = (service: ServiceInfo) => {
  editDialogRef.value?.openEdit(service, namespaces.value)
}

const handleEditSuccess = () => {
  emit('refresh')
  loadServices()
}

const handleEditYAML = async (service: ServiceInfo) => {
  if (!props.clusterId) return
  selectedService.value = service
  try {
    const response = await getServiceYAML(props.clusterId, service.namespace, service.name)
    // 保存原始 JSON 数据
    originalJsonData.value = response.items || response
    // 转换为 YAML 格式
    const yaml = dump(originalJsonData.value, { indent: 2, lineWidth: -1 })
    yamlContent.value = yaml
    yamlDialogVisible.value = true
  } catch (error) {
    Message.error('获取 YAML 失败')
  }
}


// 使用 js-yaml 库解析 YAML
const yamlToJson = (yaml: string): any => {
  try {
    return load(yaml)
  } catch (error) {
    throw error
  }
}

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedService.value) return

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
      if (!jsonData.metadata.name && selectedService.value) {
        jsonData.metadata.name = selectedService.value.name
      }
      if (!jsonData.metadata.namespace && selectedService.value) {
        jsonData.metadata.namespace = selectedService.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'Service'
      }
    } catch (e) {
      Message.error('YAML 格式错误，请检查缩进和语法')
      saving.value = false
      return
    }

    await updateServiceYAML(
      props.clusterId,
      selectedService.value.namespace,
      selectedService.value.name,
      jsonData
    )
    Message.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadServices()
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

const handleDelete = async (service: ServiceInfo) => {
  if (!props.clusterId) return
  try {
    await confirmModal(`确定要删除服务 ${service.name} 吗？`, '删除确认', { type: 'error' })
    await deleteService(props.clusterId, service.namespace, service.name)
    Message.success('删除成功')
    emit('refresh')
    await loadServices()
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
    // 确保基本的元数据存在
    if (!jsonData.apiVersion) {
      jsonData.apiVersion = 'v1'
    }
    if (!jsonData.kind) {
      jsonData.kind = 'Service'
    }
    if (!jsonData.metadata) {
      jsonData.metadata = {}
    }

    // 从 YAML 中提取命名空间
    const namespace = jsonData.metadata.namespace || props.namespace || 'default'
    jsonData.metadata.namespace = namespace

    await createServiceYAML(
      props.clusterId,
      namespace,
      jsonData
    )
    Message.success('创建成功')
    createYamlDialogVisible.value = false
    emit('refresh')
    await loadServices()
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

watch(() => props.clusterId, () => {
  loadServices()
  loadNamespaces()
})

watch(() => props.namespace, () => {
  filterNamespace.value = props.namespace || ''
  loadServices()
})

// 监听筛选后的数据变化，更新计数
watch(filteredServices, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  loadServices()
  loadNamespaces()
})

const handleShowDetail = (service: ServiceInfo) => {
  detailDialogRef.value?.open(service.namespace, service.name)
}

const handleTerminal = (data: { namespace: string; name: string }) => {
  emit('terminal', data)
}

const handleLogs = (data: { namespace: string; name: string }) => {
  emit('logs', data)
}

// 暴露方法给父组件
defineExpose({
  loadData: () => loadServices(true)
})
</script>

<style scoped>
.service-list {
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

.name-cell:hover {
  opacity: 0.8;
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

.port-item {
  font-size: 12px;
  color: #606266;
  line-height: 1.5;
}

.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px;
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
