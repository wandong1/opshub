<template>
  <div class="configmap-list">
    <!-- 搜索和筛选 -->
    <div class="search-bar">
      <div class="search-bar-left">
        <a-input
          v-model="searchName"
          placeholder="搜索 ConfigMap 名称..."
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
        <a-button v-permission="'k8s-configmaps:create'" type="primary" @click="handleCreateYAML">
          <icon-file />
          YAML创建
        </a-button>

        <a-button v-permission="'k8s-configmaps:create'" type="primary" @click="handleCreateForm">
          <icon-plus />
          表单创建
        </a-button>
      </div>
    </div>

    <!-- ConfigMap 列表 -->
    <div class="table-wrapper">
      <a-table
        :data="paginatedConfigMaps"
        :loading="loading"
        class="modern-table"
        size="default"
       :columns="tableColumns5">
          <template #name="{ record }">
            <div class="name-cell">
              <div class="name-icon-wrapper">
                <icon-safe />
              </div>
              <div class="name-content">
                <div class="name-text">{{ record.name }}</div>
                <div class="namespace-text">{{ record.namespace }}</div>
              </div>
            </div>
          </template>
          <template #dataCount="{ record }">
            <a-tag color="gray" size="small">{{ record.dataCount }}</a-tag>
          </template>
          <template #createdAt="{ record }">
            {{ record.createdAt || '-' }}
          </template>
          <template #actions="{ record }">
            <div class="action-buttons">
              <a-tooltip content="编辑 YAML" placement="top">
                <a-button v-permission="'k8s-configmaps:update'" type="text" class="action-btn" @click="handleEditYAML(record)">
                  <icon-file />
                </a-button>
              </a-tooltip>
              <a-tooltip content="编辑" placement="top">
                <a-button v-permission="'k8s-configmaps:update'" type="text" class="action-btn" @click="handleEditForm(record)">
                  <icon-edit />
                </a-button>
              </a-tooltip>
              <a-tooltip content="删除" placement="top">
                <a-button v-permission="'k8s-configmaps:delete'" type="text" class="action-btn danger" @click="handleDelete(record)">
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
          :total="filteredConfigMaps.length"
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

    <!-- 表单创建弹窗 -->
    <a-modal v-model:visible="formDialogVisible" :title="formDialogTitle" width="1200px" class="form-dialog">
      <a-form :model="formData" label-width="100px" class="configmap-form">
        <div class="form-row">
          <a-form-item label="名称" required>
            <a-input v-model="formData.name" placeholder="请输入 ConfigMap 名称" style="width: 100%;" />
          </a-form-item>
          <a-form-item label="命名空间" required>
            <a-select v-model="formData.namespace" placeholder="请选择命名空间" style="width: 100%;">
              <a-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
            </a-select>
          </a-form-item>
        </div>

        <!-- 标签页 -->
        <a-tabs v-model:active-key="activeTab" class="form-tabs">
          <!-- 数据标签页 -->
          <a-tab-pane title="数据" key="data">
            <div class="tab-content">
              <!-- Data 部分 -->
              <div class="data-section">
                <div class="section-header">
                  <span class="section-title">Data</span>
                  <a-button size="small" type="primary" @click="addDataRow">
                    <icon-plus /> 添加数据
                  </a-button>
                </div>
                <a-table :data="formData.data" border class="form-table" :columns="tableColumns4">
          <template #col_Key="{ record }">
                      <a-input v-model="record.key" placeholder="请输入 Key" />
                    </template>
          <template #col_Value="{ record }">
                      <a-textarea v-model="record.value" :rows="3" placeholder="请输入 Value" />
                    </template>
          <template #actions="{ rowIndex }">
                      <a-button type="text" status="danger" @click="removeDataRow(rowIndex)">
                        <icon-delete />
                      </a-button>
                    </template>
        </a-table>
              </div>

              <!-- BinaryData 部分 -->
              <div class="binarydata-section">
                <div class="section-header">
                  <span class="section-title">BinaryData</span>
                  <a-button size="small" type="primary" @click="addBinaryDataRow">
                    <icon-plus /> 添加二进制数据
                  </a-button>
                </div>
                <a-table :data="formData.binaryData" border class="form-table" :columns="tableColumns3">
          <template #col_Key="{ record }">
                      <a-input v-model="record.key" placeholder="请输入 Key" />
                    </template>
          <template #col_Value="{ record }">
                      <a-textarea v-model="record.value" :rows="3" placeholder="请输入 Value (Base64编码)" />
                    </template>
          <template #actions="{ rowIndex }">
                      <a-button type="text" status="danger" @click="removeBinaryDataRow(rowIndex)">
                        <icon-delete />
                      </a-button>
                    </template>
        </a-table>
              </div>
            </div>
          </a-tab-pane>

          <!-- 标签/注解标签页 -->
          <a-tab-pane title="标签/注解" key="metadata">
            <div class="tab-content">
              <div class="metadata-section">
                <div class="metadata-header">
                  <span class="metadata-title">标签</span>
                  <a-button size="small" @click="addLabelRow">
                    <icon-plus /> 添加
                  </a-button>
                </div>
                <a-table :data="formData.labels" border class="form-table" :columns="tableColumns2">
          <template #col_Key="{ record }">
                      <a-input v-model="record.key" placeholder="请输入 Key" />
                    </template>
          <template #col_Value="{ record }">
                      <a-input v-model="record.value" placeholder="请输入 Value" />
                    </template>
          <template #actions="{ rowIndex }">
                      <a-button type="text" status="danger" @click="removeLabelRow(rowIndex)">
                        <icon-delete />
                      </a-button>
                    </template>
        </a-table>
              </div>

              <div class="metadata-section">
                <div class="metadata-header">
                  <span class="metadata-title">注解</span>
                  <a-button size="small" @click="addAnnotationRow">
                    <icon-plus /> 添加
                  </a-button>
                </div>
                <a-table :data="formData.annotations" border class="form-table" :columns="tableColumns">
          <template #col_Key="{ record }">
                      <a-input v-model="record.key" placeholder="请输入 Key" />
                    </template>
          <template #col_Value="{ record }">
                      <a-input v-model="record.value" placeholder="请输入 Value" />
                    </template>
          <template #actions="{ rowIndex }">
                      <a-button type="text" status="danger" @click="removeAnnotationRow(rowIndex)">
                        <icon-delete />
                      </a-button>
                    </template>
        </a-table>
              </div>
            </div>
          </a-tab-pane>
        </a-tabs>
      </a-form>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="formDialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleSaveForm" :loading="saving">{{ isEditMode ? '保存' : '创建' }}</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
const tableColumns5 = [
  { title: '名称', dataIndex: 'name', slotName: 'name', width: 200 },
  { title: '数据项', dataIndex: 'dataCount', slotName: 'dataCount', width: 100, align: 'center' },
  { title: '存活时间', dataIndex: 'age', width: 140 },
  { title: '创建时间', dataIndex: 'createdAt', slotName: 'createdAt', width: 180 },
  { title: '操作', slotName: 'actions', width: 160, fixed: 'right', align: 'center' }
]

const tableColumns4 = [
  { title: 'Key', slotName: 'col_Key', width: 200 },
  { title: 'Value', slotName: 'col_Value' },
  { title: '操作', slotName: 'actions', width: 80 }
]

const tableColumns3 = [
  { title: 'Key', slotName: 'col_Key', width: 200 },
  { title: 'Value', slotName: 'col_Value' },
  { title: '操作', slotName: 'actions', width: 80 }
]

const tableColumns2 = [
  { title: 'Key', slotName: 'col_Key', width: 200 },
  { title: 'Value', slotName: 'col_Value' },
  { title: '操作', slotName: 'actions', width: 80 }
]

const tableColumns = [
  { title: 'Key', slotName: 'col_Key', width: 200 },
  { title: 'Value', slotName: 'col_Value' },
  { title: '操作', slotName: 'actions', width: 80 }
]

import { ref, computed, onMounted, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import { getNamespaces } from '@/api/kubernetes'
import axios from 'axios'
import * as yaml from 'js-yaml'

interface ConfigMapInfo {
  name: string
  namespace: string
  dataCount: number
  age: string
  createdAt?: string
}

interface KeyValueRow {
  key: string
  value: string
  _id: number
}

let _rowId = 0
const nextRowId = () => ++_rowId

const props = defineProps<{
  clusterId?: number
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh', 'count-update'])

const loading = ref(false)
const configMapList = ref<ConfigMapInfo[]>([])
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
const selectedConfigMap = ref<ConfigMapInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const saving = ref(false)
const isCreateMode = ref(false)
const isEditMode = ref(false)

// YAML对话框标题
const yamlDialogTitle = computed(() => {
  if (isCreateMode.value) {
    return '新增 ConfigMap (YAML)'
  }
  return `ConfigMap YAML - ${selectedConfigMap.value?.name || ''}`
})

// 表单对话框标题
const formDialogTitle = computed(() => {
  if (isEditMode.value) {
    return `编辑 ConfigMap - ${formData.value.name}`
  }
  return '新增 ConfigMap'
})

// 表单创建
const formDialogVisible = ref(false)
const activeTab = ref('data')
const formData = ref({
  name: '',
  namespace: '',
  data: [] as KeyValueRow[],
  binaryData: [] as KeyValueRow[],
  labels: [] as KeyValueRow[],
  annotations: [] as KeyValueRow[]
})

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

// 过滤后的列表
const filteredConfigMaps = computed(() => {
  let result = configMapList.value

  if (searchName.value) {
    result = result.filter(cm =>
      cm.name.toLowerCase().includes(searchName.value.toLowerCase())
    )
  }

  if (filterNamespace.value) {
    result = result.filter(cm => cm.namespace === filterNamespace.value)
  }

  return result
})

// 分页后的列表
const paginatedConfigMaps = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredConfigMaps.value.slice(start, end)
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

// 加载 ConfigMap 列表
const loadConfigMaps = async () => {
  if (!props.clusterId) return

  loading.value = true
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(`/api/v1/plugins/kubernetes/resources/configmaps`, {
      params: { clusterId: props.clusterId },
      headers: { Authorization: `Bearer ${token}` }
    })
    configMapList.value = response.data.data || []
  } catch (error) {
    configMapList.value = []
    // 不显示错误提示，避免频繁弹出
  } finally {
    loading.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  currentPage.value = 1
}

// YAML 创建
const handleCreateYAML = () => {
  isCreateMode.value = true
  selectedConfigMap.value = null
  // 默认 ConfigMap YAML 模板
  yamlContent.value = `apiVersion: v1
kind: ConfigMap
metadata:
  name: example-configmap
  namespace: default
data:
  key1: value1
  key2: value2
`
  yamlDialogVisible.value = true
}

// 表单创建
const handleCreateForm = () => {
  isEditMode.value = false
  formData.value = {
    name: '',
    namespace: namespaces.value[0]?.name || '',
    data: [],
    binaryData: [],
    labels: [],
    annotations: []
  }
  formDialogVisible.value = true
}

// 编辑表单
const handleEditForm = async (row: ConfigMapInfo) => {
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )


    // 获取ConfigMap对象，可能是items或yaml字段
    let configMap: any = response.data?.data?.items || response.data?.data?.yaml

    // 如果是yaml字符串，需要解析
    if (typeof configMap === 'string') {
      configMap = yaml.load(configMap)
    }


    if (!configMap || !configMap.metadata) {
      Message.error('获取ConfigMap数据失败')
      return
    }

    // 填充表单数据
    formData.value = {
      name: configMap.metadata?.name || '',
      namespace: configMap.metadata?.namespace || '',
      data: configMap.data ? Object.entries(configMap.data).map(([key, value]) => ({ key, value: String(value) })) : [],
      binaryData: configMap.binaryData ? Object.entries(configMap.binaryData).map(([key, value]) => ({ key, value: String(value) })) : [],
      labels: configMap.metadata?.labels ? Object.entries(configMap.metadata.labels).map(([key, value]) => ({ key, value })) : [],
      annotations: configMap.metadata?.annotations ? Object.entries(configMap.metadata.annotations).map(([key, value]) => ({ key, value })) : []
    }


    isEditMode.value = true
    formDialogVisible.value = true
  } catch (error: any) {
    Message.error(`获取详情失败: ${error.response?.data?.message || error.message}`)
  }
}

// 编辑 YAML
const handleEditYAML = async (row: ConfigMapInfo) => {
  selectedConfigMap.value = row
  isCreateMode.value = false

  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}/yaml`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )
    // 后端返回的是 {data: {items: ConfigMap对象}}
    const jsonData = response.data.data?.items
    if (jsonData) {
      yamlContent.value = yaml.dump(jsonData, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      })
      yamlDialogVisible.value = true
    }
  } catch (error: any) {
    Message.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

// 保存 YAML
const handleSaveYAML = async () => {
  saving.value = true
  try {
    const token = localStorage.getItem('token')

    // 从 YAML 中解析对象
    const yamlObj: any = yaml.load(yamlContent.value)
    if (!yamlObj || !yamlObj.metadata || !yamlObj.metadata.name) {
      Message.error('YAML 中缺少必要的 metadata.name 字段')
      return
    }
    const name = yamlObj.metadata.name
    const namespace = yamlObj.metadata.namespace || 'default'

    if (isCreateMode.value) {
      // 创建模式 - 发送 JSON 对象
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/configmaps/${namespace}/yaml`,
        yamlObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      Message.success('创建成功')
    } else {
      // 编辑模式 - 发送 JSON 对象
      if (!selectedConfigMap.value) return
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/configmaps/${selectedConfigMap.value.namespace}/${selectedConfigMap.value.name}/yaml`,
        yamlObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      Message.success('保存成功')
    }

    yamlDialogVisible.value = false
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    Message.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    saving.value = false
  }
}

// 保存表单
const handleSaveForm = async () => {
  if (!formData.value.name) {
    Message.error('请输入名称')
    return
  }
  if (!formData.value.namespace) {
    Message.error('请选择命名空间')
    return
  }

  saving.value = true
  try {
    const token = localStorage.getItem('token')


    // 构建 ConfigMap 对象
    const configMapObj: any = {
      apiVersion: 'v1',
      kind: 'ConfigMap',
      metadata: {
        name: formData.value.name,
        namespace: formData.value.namespace
      }
    }

    // 添加 Data
    if (formData.value.data.length > 0) {
      configMapObj.data = {}
      formData.value.data.forEach(row => {
        if (row.key) {
          configMapObj.data[row.key] = row.value
        }
      })
    }

    // 添加 BinaryData
    if (formData.value.binaryData.length > 0) {
      configMapObj.binaryData = {}
      formData.value.binaryData.forEach(row => {
        if (row.key) {
          configMapObj.binaryData[row.key] = row.value
        }
      })
    }

    // 添加标签
    if (formData.value.labels.length > 0) {
      configMapObj.metadata.labels = {}
      formData.value.labels.forEach(row => {
        if (row.key) {
          configMapObj.metadata.labels[row.key] = row.value
        }
      })
    }

    // 添加注解
    if (formData.value.annotations.length > 0) {
      configMapObj.metadata.annotations = {}
      formData.value.annotations.forEach(row => {
        if (row.key) {
          configMapObj.metadata.annotations[row.key] = row.value
        }
      })
    }

    if (isEditMode.value) {
      // 编辑模式：使用 PUT 请求
      await axios.put(
        `/api/v1/plugins/kubernetes/resources/configmaps/${formData.value.namespace}/${formData.value.name}/yaml`,
        configMapObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      Message.success('更新成功')
    } else {
      // 创建模式：使用 POST 请求
      await axios.post(
        `/api/v1/plugins/kubernetes/resources/configmaps/${formData.value.namespace}/yaml`,
        configMapObj,
        {
          params: { clusterId: props.clusterId },
          headers: { Authorization: `Bearer ${token}` }
        }
      )
      Message.success('创建成功')
    }

    formDialogVisible.value = false
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    Message.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    saving.value = false
  }
}

// 删除 ConfigMap
const handleDelete = async (row: ConfigMapInfo) => {
  try {
    await confirmModal(
      `确定要删除 ConfigMap ${row.name} 吗？此操作不可恢复！`,
      '删除 ConfigMap 确认',
      {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'error'
      }
    )

    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/configmaps/${row.namespace}/${row.name}`,
      {
        params: { clusterId: props.clusterId },
        headers: { Authorization: `Bearer ${token}` }
      }
    )

    Message.success('删除成功')
    await loadConfigMaps()
    emit('refresh')
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

// 数据行操作
const addDataRow = () => {
  formData.value.data.push({ key: '', value: '', _id: nextRowId() })
}

const removeDataRow = (index: number) => {
  formData.value.data.splice(index, 1)
}

// BinaryData 行操作
const addBinaryDataRow = () => {
  formData.value.binaryData.push({ key: '', value: '', _id: nextRowId() })
}

const removeBinaryDataRow = (index: number) => {
  formData.value.binaryData.splice(index, 1)
}

// 标签行操作
const addLabelRow = () => {
  formData.value.labels.push({ key: '', value: '', _id: nextRowId() })
}

const removeLabelRow = (index: number) => {
  formData.value.labels.splice(index, 1)
}

// 注解行操作
const addAnnotationRow = () => {
  formData.value.annotations.push({ key: '', value: '', _id: nextRowId() })
}

const removeAnnotationRow = (index: number) => {
  formData.value.annotations.splice(index, 1)
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
    loadConfigMaps()
  }
})

// 监听筛选后的数据变化，更新计数
watch(filteredConfigMaps, (newData) => {
  emit('count-update', newData.length)
})

onMounted(() => {
  if (props.clusterId) {
    loadNamespaces()
    loadConfigMaps()
  }
})

// 暴露方法给父组件
defineExpose({
  loadConfigMaps
})
</script>

<style scoped>
.configmap-list {
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

/* 表单弹窗 */
.form-dialog :deep(.arco-dialog__body) {
  padding: 20px 24px;
  max-height: 600px;
  overflow-y: auto;
}

.configmap-form {
  max-width: 100%;
}

.form-row {
  display: flex;
  gap: 16px;
  margin-bottom: 16px;
}

.form-row .arco-form-item {
  flex: 1;
  margin-bottom: 0;
}

.form-tabs {
  margin-top: 16px;
}

.tab-content {
  padding: 16px 0;
}

.table-actions-wrapper {
  margin-bottom: 12px;
}

.data-section {
  margin-bottom: 32px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.binarydata-section {
  margin-bottom: 32px;
  padding: 16px;
  background-color: #f5f7fa;
  border-radius: 8px;
  border: 1px solid #e4e7ed;
}

.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.section-title {
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.data-title {
  font-weight: 600;
  color: #333;
}

.metadata-section {
  margin-bottom: 24px;
}

.metadata-section:last-child {
  margin-bottom: 0;
}

.metadata-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.metadata-title {
  font-weight: 600;
  color: #333;
}

.table-header-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.table-title {
  font-weight: 600;
  color: #165dff;
}

.form-table {
  width: 100%;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

</style>
