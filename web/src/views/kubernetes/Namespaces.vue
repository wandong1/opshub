<template>
  <div class="namespaces-container">
    <!-- 统计卡片 -->
    <a-row :gutter="20" class="stats-row">
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-primary, #165dff)">
              <icon-folder :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ namespaceList.length }}</div>
              <div class="stat-label">命名空间总数</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-success, #00b42a)">
              <icon-check-circle :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ activeNamespaceCount }}</div>
              <div class="stat-label">正常状态</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: var(--ops-warning, #ff7d00)">
              <icon-tag :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ totalLabels }}</div>
              <div class="stat-label">标签总数</div>
            </div>
          </div>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card class="stat-card" hoverable>
          <div class="stat-content">
            <div class="stat-icon" style="background-color: #722ed1">
              <icon-clock-circle :size="28" style="color: #fff" />
            </div>
            <div class="stat-info">
              <div class="stat-value">{{ averageAge }}</div>
              <div class="stat-label">平均运行时间</div>
            </div>
          </div>
        </a-card>
      </a-col>
    </a-row>

    <!-- 页面标题和操作按钮 -->
    <a-card class="page-header-card">
      <div class="page-header">
        <div class="page-title-group">
          <div class="page-title-icon">
            <icon-folder />
          </div>
          <div>
            <h2 class="page-title">命名空间</h2>
            <p class="page-subtitle">管理 Kubernetes 命名空间，实现资源隔离和分组</p>
          </div>
        </div>
        <div class="header-actions">
          <a-select
            v-model="selectedClusterId"
            placeholder="选择集群"
            style="width: 260px"
            @change="handleClusterChange"
          >
            <template #prefix>
              <icon-apps />
            </template>
            <a-option
              v-for="cluster in clusterList"
              :key="cluster.id"
              :label="cluster.alias || cluster.name"
              :value="cluster.id"
            />
          </a-select>
          <a-button @click="loadNamespaces">
            <template #icon><icon-refresh /></template>
            刷新
          </a-button>
          <a-button v-permission="'k8s-namespaces:create'" type="primary" @click="handleCreateNamespace">
            <template #icon><icon-plus /></template>
            新建命名空间
          </a-button>
        </div>
      </div>
    </a-card>

    <!-- 搜索和筛选 -->
    <a-card class="search-card">
      <a-form :model="{}" layout="inline" class="search-form">
        <a-form-item>
          <a-input
            v-model="searchName"
            placeholder="搜索命名空间名称..."
            allow-clear
            @clear="handleSearch"
            @keyup.enter="handleSearch"
            @input="handleSearch"
            style="width: 280px"
          >
            <template #prefix>
              <icon-search />
            </template>
          </a-input>
        </a-form-item>
        <a-form-item>
          <a-select
            v-model="searchStatus"
            placeholder="状态"
            allow-clear
            @change="handleSearch"
            style="width: 150px"
          >
            <a-option label="正常" value="Active" />
            <a-option label="终止中" value="Terminating" />
          </a-select>
        </a-form-item>
      </a-form>
    </a-card>

    <!-- 命名空间列表 -->
    <a-card class="table-card">
      <a-table
        :data="paginatedNamespaceList"
        :loading="loading"
        :bordered="false"
        :columns="tableColumns2"
        :pagination="false"
      >
        <template #namespace="{ record }">
          <div class="namespace-name-cell">
            <div class="namespace-icon-wrapper">
              <icon-folder />
            </div>
            <div class="namespace-name-content">
              <div class="namespace-name">{{ record.name }}</div>
              <div class="namespace-sub">{{ record.status || 'Active' }}</div>
            </div>
          </div>
        </template>
        <template #status="{ record }">
          <a-tag :color="getStatusType(record.status)">
            {{ getStatusText(record.status) }}
          </a-tag>
        </template>
        <template #labels="{ record }">
          <a-button type="text" size="small" @click="showLabels(record)">
            <a-badge :count="Object.keys(record.labels || {}).length" :dot-style="{ background: 'var(--ops-primary, #165dff)' }">
              <icon-tag :size="18" />
            </a-badge>
          </a-button>
        </template>
        <template #uptime="{ record }">
          <div class="age-cell">
            <icon-clock-circle />
            <span>{{ record.age || '-' }}</span>
          </div>
        </template>
        <template #actions="{ record }">
          <div class="action-buttons">
            <a-tooltip content="编辑标签" position="top">
              <a-button type="text" class="action-btn action-edit" @click="handleActionCommand('edit', record)">
                <icon-edit />
              </a-button>
            </a-tooltip>
            <a-tooltip content="YAML" position="top">
              <a-button type="text" class="action-btn" @click="handleActionCommand('yaml', record)">
                <icon-file />
              </a-button>
            </a-tooltip>
            <a-tooltip content="删除" position="top">
              <a-button type="text" class="action-btn action-delete" @click="handleActionCommand('delete', record)">
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
          :total="filteredNamespaceList.length"
          show-total show-page-size show-jumper
          @change="handlePageChange"
          @page-size-change="handleSizeChange"
        />
      </div>
    </a-card>

    <!-- 新建命名空间弹窗 -->
    <a-modal
      v-model:visible="createDialogVisible"
      title="新建命名空间"
      :width="600"
    >
      <a-form :model="createForm" :rules="createRules" ref="createFormRef" auto-label-width>
        <a-form-item label="名称" field="name">
          <a-input
            v-model="createForm.name"
            placeholder="请输入命名空间名称"
            :max-length="63"
            show-word-limit
          />
          <template #extra>
            命名空间名称只能包含小写字母、数字和连字符，且必须以字母或数字开头和结尾
          </template>
        </a-form-item>
      </a-form>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="createDialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleCreateSubmit" :loading="createLoading">
            创建
          </a-button>
        </div>
      </template>
    </a-modal>

    <!-- 标签编辑弹窗 -->
    <a-modal
      v-model:visible="editLabelDialogVisible"
      :title="labelEditMode ? '编辑命名空间标签' : '命名空间标签'"
      :width="850"
      :mask-closable="!labelEditMode"
    >
      <div class="label-dialog-content">
        <!-- 编辑模式 -->
        <div v-if="labelEditMode" class="label-edit-container">
          <a-alert type="info" :show-icon="false" style="margin-bottom: 16px">
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>编辑 {{ selectedNamespace?.name }} 的标签</span>
              <a-tag color="arcoblue">共 {{ editLabelList.length }} 个标签</a-tag>
            </div>
          </a-alert>

          <div class="label-edit-list">
            <div v-for="(label, index) in editLabelList" :key="index" class="label-edit-row">
              <span class="label-row-number">{{ index + 1 }}</span>
              <a-input v-model="label.key" placeholder="Key，如: app" style="flex: 1" />
              <span class="label-separator">=</span>
              <a-input v-model="label.value" placeholder="Value，可为空" style="flex: 1" />
              <a-button type="text" status="danger" @click="removeEditLabel(index)">
                <template #icon><icon-delete /></template>
              </a-button>
            </div>
            <a-empty v-if="editLabelList.length === 0" description="暂无标签，点击下方按钮添加" />
          </div>

          <a-button long type="dashed" @click="addEditLabel" style="margin-top: 12px">
            <template #icon><icon-plus /></template>
            添加标签
          </a-button>
        </div>

        <!-- 查看模式 -->
        <a-table v-else :data="labelList" :bordered="false" :pagination="false" :columns="labelTableColumns" style="max-height: 500px">
          <template #key="{ record }">
            <a-tag color="arcoblue" class="label-key-tag" @click="copyToClipboard(record.key, 'Key')">
              {{ record.key }}
              <icon-copy :size="12" style="margin-left: 4px" />
            </a-tag>
          </template>
          <template #value="{ record }">
            <span class="label-value">{{ record.value || '-' }}</span>
          </template>
        </a-table>
      </div>

      <template #footer>
        <div class="dialog-footer">
          <template v-if="labelEditMode">
            <a-button @click="cancelLabelEdit">取消</a-button>
            <a-button type="primary" @click="saveLabels" :loading="labelSaving">
              保存更改
            </a-button>
          </template>
          <template v-else>
            <a-button @click="editLabelDialogVisible = false">关闭</a-button>
            <a-button type="primary" @click="startLabelEdit">
              编辑标签
            </a-button>
          </template>
        </div>
      </template>
    </a-modal>

    <!-- YAML 查看弹窗 -->
    <a-modal
      v-model:visible="yamlDialogVisible"
      :title="`命名空间 YAML - ${selectedNamespace?.name || ''}`"
      :width="900"
    >
      <div class="yaml-editor-wrapper">
        <div class="line-numbers">
          <div v-for="line in yamlLineCount" :key="line" class="line-number">{{ line }}</div>
        </div>
        <textarea
          v-model="yamlContent"
          class="code-textarea"
          placeholder="YAML 内容"
          spellcheck="false"
          @scroll="handleYamlScroll"
          ref="yamlTextarea"
          readonly
        ></textarea>
      </div>
      <template #footer>
        <div class="dialog-footer">
          <a-button @click="yamlDialogVisible = false">关闭</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { confirmModal } from '@/utils/confirm'
import { ref, onMounted, computed } from 'vue'
import { Message } from '@arco-design/web-vue'
import type { FieldRule } from '@arco-design/web-vue'
import axios from 'axios'
import { getClusterList, type Cluster, getNamespaces, type NamespaceInfo } from '@/api/kubernetes'

const tableColumns2 = [
  { title: '命名空间', slotName: 'namespace', width: 220 },
  { title: '状态', slotName: 'status', width: 120, align: 'center' as const },
  { title: '标签', slotName: 'labels', width: 120, align: 'center' as const },
  { title: '运行时间', slotName: 'uptime', width: 160 },
  { title: '操作', slotName: 'actions', width: 140, align: 'center' as const }
]

const labelTableColumns = [
  { title: 'Key', dataIndex: 'key', slotName: 'key' },
  { title: 'Value', dataIndex: 'value', slotName: 'value' }
]

const loading = ref(false)
const clusterList = ref<Cluster[]>([])
const selectedClusterId = ref<number>()
const namespaceList = ref<NamespaceInfo[]>([])

// 搜索条件
const searchName = ref('')
const searchStatus = ref('')

// 分页状态
const currentPage = ref(1)
const pageSize = ref(10)
const paginationStorageKey = ref('namespaces_pagination')

// 新建命名空间
const createDialogVisible = ref(false)
const createLoading = ref(false)
const createFormRef = ref()
const createForm = ref({ name: '' })
const createRules: Record<string, FieldRule[]> = {
  name: [
    { required: true, message: '请输入命名空间名称' },
    {
      match: /^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/,
      message: '只能包含小写字母、数字和连字符，且必须以字母或数字开头和结尾'
    },
    { maxLength: 63, message: '最多63个字符' }
  ]
}

// 标签弹窗
const editLabelDialogVisible = ref(false)
const labelList = ref<{ key: string; value: string }[]>([])
const labelEditMode = ref(false)
const labelSaving = ref(false)
const editLabelList = ref<{ key: string; value: string }[]>([])
const labelOriginalYaml = ref('')

// YAML 弹窗
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedNamespace = ref<NamespaceInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)

const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const filteredNamespaceList = computed(() => {
  let result = namespaceList.value
  if (searchName.value) {
    result = result.filter(ns => ns.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (searchStatus.value) {
    result = result.filter(ns => ns.status === searchStatus.value)
  }
  return result
})

const paginatedNamespaceList = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  return filteredNamespaceList.value.slice(start, start + pageSize.value)
})

const activeNamespaceCount = computed(() =>
  namespaceList.value.filter(ns => !ns.status || ns.status === 'Active').length
)

const totalLabels = computed(() =>
  namespaceList.value.reduce((sum, ns) => sum + Object.keys(ns.labels || {}).length, 0)
)

const averageAge = computed(() => {
  if (namespaceList.value.length === 0) return '-'
  return namespaceList.value[0]?.age || '-'
})

const getStatusType = (status: string | undefined) => {
  if (!status || status === 'Active') return 'green'
  if (status === 'Terminating') return 'orange'
  return 'blue'
}

const getStatusText = (status: string | undefined) => {
  if (!status || status === 'Active') return '正常'
  if (status === 'Terminating') return '终止中'
  return status
}

const showLabels = (row: NamespaceInfo) => {
  const labels = row.labels || {}
  labelList.value = Object.keys(labels).map(key => ({ key, value: labels[key] }))
  labelEditMode.value = false
  selectedNamespace.value = row
  editLabelDialogVisible.value = true
}

const copyToClipboard = async (text: string, type: string) => {
  try {
    await navigator.clipboard.writeText(text)
    Message.success(`${type} 已复制到剪贴板`)
  } catch {
    const textarea = document.createElement('textarea')
    textarea.value = text
    textarea.style.position = 'fixed'
    textarea.style.opacity = '0'
    document.body.appendChild(textarea)
    textarea.select()
    try {
      document.execCommand('copy')
      Message.success(`${type} 已复制到剪贴板`)
    } catch {
      Message.error('复制失败')
    }
    document.body.removeChild(textarea)
  }
}

const startLabelEdit = async () => {
  if (!selectedNamespace.value) return
  try {
    const token = localStorage.getItem('token')
    const namespaceName = selectedNamespace.value.name
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/namespaces/${namespaceName}/yaml`,
      { params: { clusterId: selectedClusterId.value }, headers: { Authorization: `Bearer ${token}` } }
    )
    labelOriginalYaml.value = response.data.data?.yaml || ''
    editLabelList.value = labelList.value.map(label => ({ key: label.key, value: label.value }))
    labelEditMode.value = true
  } catch {
    Message.error('获取命名空间信息失败')
  }
}

const cancelLabelEdit = () => {
  labelEditMode.value = false
  editLabelList.value = []
}

const addEditLabel = () => {
  editLabelList.value.push({ key: '', value: '' })
}

const removeEditLabel = (index: number) => {
  editLabelList.value.splice(index, 1)
}

const saveLabels = async () => {
  if (!selectedNamespace.value) return
  const validLabels = editLabelList.value.filter(label => label.key.trim() !== '')
  if (validLabels.some(label => !label.key)) {
    Message.warning('标签键不能为空')
    return
  }
  const keys = validLabels.map(l => l.key)
  if (keys.length !== new Set(keys).size) {
    Message.warning('存在重复的标签键，请检查')
    return
  }

  labelSaving.value = true
  try {
    const token = localStorage.getItem('token')
    const namespaceName = selectedNamespace.value.name
    const lines = labelOriginalYaml.value.split('\n')
    let updatedLines: string[] = []
    let i = 0
    let labelsIndent = ''

    while (i < lines.length) {
      const line = lines[i]
      if (/^metadata:\s*$/.test(line)) {
        updatedLines.push(line)
        i++
        while (i < lines.length) {
          const metaLine = lines[i]
          if (/^(\s+)labels:\s*$/.test(metaLine)) {
            labelsIndent = metaLine.match(/^(\s+)labels:\s*$/)?.[1] || ''
            updatedLines.push(metaLine)
            i++
            break
          }
          updatedLines.push(metaLine)
          i++
          if (/^(\s+)labels:\s*$/.test(metaLine)) {
            labelsIndent = metaLine.match(/^(\s+)labels:\s*$/)?.[1] || ''
            break
          }
        }
        if (validLabels.length > 0) {
          for (const label of validLabels) {
            updatedLines.push(`${labelsIndent}  ${label.key}: ${label.value || '""'}`)
          }
        }
        while (i < lines.length && /^\s{2,}/.test(lines[i])) {
          if (!/^(\s+)/.test(lines[i])) break
          const indent = lines[i].match(/^(\s+)/)?.[1]?.length || 0
          if (indent <= labelsIndent.length) break
          i++
        }
        continue
      }
      updatedLines.push(line)
      i++
    }

    const updatedYaml = updatedLines.join('\n')
    await axios.put(
      `/api/v1/plugins/kubernetes/resources/namespaces/${namespaceName}/yaml`,
      { clusterId: selectedClusterId.value, yaml: updatedYaml },
      { headers: { Authorization: `Bearer ${token}` } }
    )
    Message.success('标签保存成功')
    labelEditMode.value = false
    await loadNamespaces()
    const updatedNamespace = namespaceList.value.find(n => n.name === namespaceName)
    if (updatedNamespace) {
      selectedNamespace.value = updatedNamespace
      labelList.value = validLabels
    }
  } catch (error: any) {
    Message.error(`保存失败: ${error.response?.data?.message || error.message}`)
  } finally {
    labelSaving.value = false
  }
}

const savePaginationState = () => {
  try {
    localStorage.setItem(paginationStorageKey.value, JSON.stringify({
      currentPage: currentPage.value, pageSize: pageSize.value
    }))
  } catch {}
}

const restorePaginationState = () => {
  try {
    const saved = localStorage.getItem(paginationStorageKey.value)
    if (saved) {
      const state = JSON.parse(saved)
      currentPage.value = state.currentPage || 1
      pageSize.value = state.pageSize || 10
    }
  } catch {
    currentPage.value = 1
    pageSize.value = 10
  }
}

const handlePageChange = (page: number) => {
  currentPage.value = page
  savePaginationState()
}

const handleSizeChange = (size: number) => {
  pageSize.value = size
  const maxPage = Math.ceil(filteredNamespaceList.value.length / size)
  if (currentPage.value > maxPage) currentPage.value = maxPage || 1
  savePaginationState()
}

const loadClusters = async () => {
  try {
    const data = await getClusterList()
    clusterList.value = data || []
    if (clusterList.value.length > 0) {
      const savedClusterId = localStorage.getItem('namespaces_selected_cluster_id')
      if (savedClusterId) {
        const savedId = parseInt(savedClusterId)
        const exists = clusterList.value.some(c => c.id === savedId)
        selectedClusterId.value = exists ? savedId : clusterList.value[0].id
      } else {
        selectedClusterId.value = clusterList.value[0].id
      }
      await loadNamespaces()
    }
  } catch {
    Message.error('获取集群列表失败')
  }
}

const handleClusterChange = async () => {
  if (selectedClusterId.value) {
    localStorage.setItem('namespaces_selected_cluster_id', selectedClusterId.value.toString())
  }
  currentPage.value = 1
  await loadNamespaces()
}

const loadNamespaces = async () => {
  if (!selectedClusterId.value) return
  loading.value = true
  try {
    const data = await getNamespaces(selectedClusterId.value)
    namespaceList.value = data || []
    restorePaginationState()
  } catch {
    namespaceList.value = []
    Message.error('获取命名空间列表失败')
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  currentPage.value = 1
  savePaginationState()
}

const handleCreateNamespace = () => {
  createForm.value.name = ''
  createDialogVisible.value = true
}

const handleCreateSubmit = async () => {
  if (!createFormRef.value) return
  const errors = await createFormRef.value.validate()
  if (errors) return

  createLoading.value = true
  try {
    const token = localStorage.getItem('token')
    const yaml = `apiVersion: v1\nkind: Namespace\nmetadata:\n  name: ${createForm.value.name}\n`
    await axios.post(
      `/api/v1/plugins/kubernetes/resources/namespaces`,
      { yaml },
      { params: { clusterId: selectedClusterId.value }, headers: { Authorization: `Bearer ${token}` } }
    )
    Message.success('命名空间创建成功')
    createDialogVisible.value = false
    await loadNamespaces()
  } catch (error: any) {
    Message.error(`创建失败: ${error.response?.data?.message || error.message}`)
  } finally {
    createLoading.value = false
  }
}

const handleActionCommand = (command: string, row: NamespaceInfo) => {
  selectedNamespace.value = row
  switch (command) {
    case 'edit': handleEditLabels(); break
    case 'yaml': handleShowYAML(); break
    case 'delete': handleDelete(); break
  }
}

const handleEditLabels = () => {
  if (!selectedNamespace.value) return
  const labels = selectedNamespace.value.labels || {}
  labelList.value = Object.keys(labels).map(key => ({ key, value: labels[key] }))
  labelEditMode.value = false
  editLabelDialogVisible.value = true
}

const handleShowYAML = async () => {
  if (!selectedNamespace.value) return
  try {
    const token = localStorage.getItem('token')
    const response = await axios.get(
      `/api/v1/plugins/kubernetes/resources/namespaces/${selectedNamespace.value.name}/yaml`,
      { params: { clusterId: selectedClusterId.value }, headers: { Authorization: `Bearer ${token}` } }
    )
    yamlContent.value = response.data.data?.yaml || ''
    yamlDialogVisible.value = true
  } catch (error: any) {
    Message.error(`获取 YAML 失败: ${error.response?.data?.message || error.message}`)
  }
}

const handleYamlScroll = (e: Event) => {
  const target = e.target as HTMLTextAreaElement
  const lineNumbers = document.querySelector('.line-numbers') as HTMLElement
  if (lineNumbers) lineNumbers.scrollTop = target.scrollTop
}

const handleDelete = async () => {
  if (!selectedNamespace.value) return
  try {
    await confirmModal(
      `确定要删除命名空间 ${selectedNamespace.value.name} 吗？此操作将删除该命名空间下的所有资源，且不可恢复！`,
      '删除命名空间确认',
      { confirmButtonText: '确定删除', cancelButtonText: '取消', type: 'error' }
    )
    const token = localStorage.getItem('token')
    await axios.delete(
      `/api/v1/plugins/kubernetes/resources/namespaces/${selectedNamespace.value.name}?clusterId=${selectedClusterId.value}`,
      { headers: { Authorization: `Bearer ${token}` } }
    )
    Message.success('命名空间删除成功')
    await loadNamespaces()
  } catch (error: any) {
    if (error !== 'cancel') {
      Message.error(`删除失败: ${error.response?.data?.message || error.message}`)
    }
  }
}

onMounted(() => { loadClusters() })
</script>

<style scoped>
.namespaces-container {
  padding: 0;
}

/* 统计卡片 */
.stats-row {
  margin-bottom: 16px;
}

.stat-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.stat-content {
  display: flex;
  align-items: center;
}

.stat-icon {
  width: 56px;
  height: 56px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-right: 16px;
  flex-shrink: 0;
}

.stat-info {
  flex: 1;
}

.stat-value {
  font-size: 28px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1;
  margin-bottom: 6px;
}

.stat-label {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
}

/* 页面头部 */
.page-header-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 14px;
}

.page-title-icon {
  width: 44px;
  height: 44px;
  background: var(--ops-primary, #165dff);
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  line-height: 1.4;
}

.header-actions {
  display: flex;
  gap: 10px;
  align-items: center;
}

/* 搜索卡片 */
.search-card {
  margin-bottom: 16px;
  border-radius: var(--ops-border-radius-md, 8px);
}

.search-form :deep(.arco-form-item) {
  margin-bottom: 0;
}

/* 表格卡片 */
.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

/* 命名空间名称单元格 */
.namespace-name-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.namespace-icon-wrapper {
  width: 34px;
  height: 34px;
  border-radius: 8px;
  background: var(--ops-primary-bg, #e8f0ff);
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--ops-primary, #165dff);
  flex-shrink: 0;
}

.namespace-name-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.namespace-name {
  font-weight: 500;
  color: var(--ops-text-primary, #1d2129);
}

.namespace-sub {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

/* 运行时间 */
.age-cell {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--ops-text-secondary, #4e5969);
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 4px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  padding: 0;
  border-radius: var(--ops-border-radius-sm, 4px);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s;
  color: var(--ops-text-secondary, #4e5969);
  font-size: 16px;
}

.action-btn:deep(.arco-btn-icon) {
  font-size: 16px;
}

.action-btn:hover {
  background-color: var(--ops-primary-bg, #e8f0ff);
  color: var(--ops-primary, #165dff);
}

.action-edit:hover {
  background-color: #fff7e8;
  color: var(--ops-warning, #ff7d00);
}

.action-delete:hover {
  background-color: #ffece8;
  color: var(--ops-danger, #f53f3f);
}

/* 分页 */
.pagination-wrapper {
  display: flex;
  justify-content: flex-end;
  padding: 16px 0 0;
  border-top: 1px solid var(--ops-border-color, #e5e6eb);
}

/* 标签编辑 */
.label-edit-list {
  max-height: 400px;
  overflow-y: auto;
}

.label-edit-row {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.label-row-number {
  flex-shrink: 0;
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--ops-primary-bg, #e8f0ff);
  color: var(--ops-primary, #165dff);
  border-radius: 6px;
  font-weight: 600;
  font-size: 12px;
}

.label-separator {
  color: var(--ops-text-tertiary, #86909c);
  font-weight: 600;
  font-size: 16px;
  flex-shrink: 0;
}

.label-key-tag {
  cursor: pointer;
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 12px;
}

.label-value {
  font-family: 'Monaco', 'Menlo', monospace;
  font-size: 13px;
  color: var(--ops-text-secondary, #4e5969);
  word-break: break-all;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* YAML 编辑器 */
.yaml-editor-wrapper {
  display: flex;
  width: 100%;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: var(--ops-border-radius-sm, 4px);
  overflow: hidden;
  background-color: #282c34;
}

.line-numbers {
  display: flex;
  flex-direction: column;
  padding: 12px 8px;
  background-color: #21252b;
  border-right: 1px solid #3e4451;
  user-select: none;
  min-width: 40px;
  text-align: right;
  overflow: hidden;
}

.line-number {
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #5c6370;
  min-height: 20.8px;
}

.code-textarea {
  flex: 1;
  min-height: 400px;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #abb2bf;
  background-color: #282c34;
  border: none;
  outline: none;
  resize: vertical;
}

.code-textarea::placeholder {
  color: #5c6370;
}

@media (max-width: 1200px) {
  .stat-value { font-size: 24px; }
  .stat-icon { width: 48px; height: 48px; }
}
</style>
