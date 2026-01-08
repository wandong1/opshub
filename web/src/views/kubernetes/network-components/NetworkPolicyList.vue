<template>
  <div class="networkpolicy-list">
    <div class="search-bar">
      <el-input v-model="searchName" placeholder="搜索 NetworkPolicy 名称..." clearable class="search-input" @input="handleSearch">
        <template #prefix>
          <el-icon class="search-icon"><Search /></el-icon>
        </template>
      </el-input>

      <el-select v-model="filterNamespace" placeholder="命名空间" clearable @change="handleSearch" class="filter-select">
        <el-option label="全部" value="" />
        <el-option v-for="ns in namespaces" :key="ns.name" :label="ns.name" :value="ns.name" />
      </el-select>

      <el-button type="primary" @click="handleCreate">创建 NetworkPolicy</el-button>
    </div>

    <div class="table-wrapper">
      <el-table :data="filteredPolicies" v-loading="loading" class="modern-table" size="default">
        <el-table-column label="名称" prop="name" min-width="180" fixed>
          <template #default="{ row }">
            <div class="name-cell">
              <el-icon class="name-icon"><Lock /></el-icon>
              <div class="name-text">{{ row.name }}</div>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="命名空间" prop="namespace" width="140" />
        <el-table-column label="Pod 选择器" min-width="200">
          <template #default="{ row }">
            <div v-if="Object.keys(row.podSelector).length" class="selector-display">
              <span v-for="(value, key) in row.podSelector" :key="key" class="selector-item">
                {{ key }}: {{ value }}
              </span>
            </div>
            <div v-else>-</div>
          </template>
        </el-table-column>
        <el-table-column label="入站规则" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.ingress.length > 0 ? 'warning' : 'info'" size="small">
              {{ row.ingress.length }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="出站规则" width="100" align="center">
          <template #default="{ row }">
            <el-tag :type="row.egress.length > 0 ? 'danger' : 'info'" size="small">
              {{ row.egress.length }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="存活时间" prop="age" width="120" />
        <el-table-column label="操作" width="160" fixed="right" align="center">
          <template #default="{ row }">
            <div class="action-buttons">
              <el-tooltip content="编辑 YAML" placement="top">
                <el-button link class="action-btn" @click="handleEditYAML(row)">
                  <el-icon :size="18"><Document /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="编辑" placement="top">
                <el-button link class="action-btn" @click="handleEdit(row)">
                  <el-icon :size="18"><Edit /></el-icon>
                </el-button>
              </el-tooltip>
              <el-tooltip content="删除" placement="top">
                <el-button link class="action-btn danger" @click="handleDelete(row)">
                  <el-icon :size="18"><Delete /></el-icon>
                </el-button>
              </el-tooltip>
            </div>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog v-model="yamlDialogVisible" :title="`NetworkPolicy YAML - ${selectedPolicy?.name}`" width="900px" class="yaml-dialog">
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
          <el-button @click="yamlDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSaveYAML" :loading="saving">保存</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Search, Lock, Document, Edit, Delete } from '@element-plus/icons-vue'
import { getNetworkPolicies, getNetworkPolicyYAML, updateNetworkPolicyYAML, deleteNetworkPolicy, getNamespaces, type NetworkPolicyDetailInfo } from '@/api/kubernetes'

const props = defineProps<{
  clusterId?: number
  namespace?: string
}>()

const emit = defineEmits(['edit', 'yaml', 'refresh'])

const loading = ref(false)
const saving = ref(false)
const policyList = ref<NetworkPolicyDetailInfo[]>([])
const namespaces = ref<any[]>([])
const searchName = ref('')
const filterNamespace = ref('')
const yamlDialogVisible = ref(false)
const yamlContent = ref('')
const selectedPolicy = ref<NetworkPolicyDetailInfo | null>(null)
const yamlTextarea = ref<HTMLTextAreaElement | null>(null)
const originalJsonData = ref<any>(null) // 保存原始 JSON 数据

// 计算YAML行数
const yamlLineCount = computed(() => {
  if (!yamlContent.value) return 1
  return yamlContent.value.split('\n').length
})

const filteredPolicies = computed(() => {
  let result = policyList.value
  if (searchName.value) {
    result = result.filter(p => p.name.toLowerCase().includes(searchName.value.toLowerCase()))
  }
  if (filterNamespace.value) {
    result = result.filter(p => p.namespace === filterNamespace.value)
  }
  return result
})

const loadPolicies = async () => {
  if (!props.clusterId) return
  loading.value = true
  try {
    const data = await getNetworkPolicies(props.clusterId, props.namespace || undefined)
    policyList.value = data || []
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 NetworkPolicy 列表失败')
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
    console.error(error)
  }
}

const handleSearch = () => {
  // 本地过滤
}

const handleCreate = () => {
  ElMessage.info('创建 NetworkPolicy 功能开发中...')
}

const handleEdit = (policy: NetworkPolicyDetailInfo) => {
  emit('edit', policy)
}

const handleEditYAML = async (policy: NetworkPolicyDetailInfo) => {
  if (!props.clusterId) return
  selectedPolicy.value = policy
  try {
    const response = await getNetworkPolicyYAML(props.clusterId, policy.namespace, policy.name)
    // 保存原始 JSON 数据
    originalJsonData.value = response.items || response
    // 转换为 YAML 格式
    const yaml = jsonToYaml(originalJsonData.value)
    yamlContent.value = yaml
    yamlDialogVisible.value = true
  } catch (error) {
    console.error(error)
    ElMessage.error('获取 YAML 失败')
  }
}

const jsonToYaml = (obj: any, indent = 0): string => {
  const spaces = '  '.repeat(indent)
  let result = ''

  if (Array.isArray(obj)) {
    for (const item of obj) {
      result += `${spaces}- ${jsonToYaml(item, indent).trim()}\n`
    }
  } else if (typeof obj === 'object' && obj !== null) {
    for (const [key, value] of Object.entries(obj)) {
      if (value === null || value === undefined) {
        result += `${spaces}${key}: null\n`
      } else if (typeof value === 'object') {
        result += `${spaces}${key}:\n${jsonToYaml(value, indent + 1)}`
      } else {
        result += `${spaces}${key}: ${value}\n`
      }
    }
  } else {
    result = `${obj}\n`
  }

  return result
}

// 简单的 YAML 到 JSON 解析器
const yamlToJson = (yaml: string): any => {
  const lines = yaml.split('\n')
  const result: any = {}
  const stack: Array<{ obj: any; indent: number }> = [{ obj: result, indent: -1 }]

  for (const line of lines) {
    const trimmed = line.trim()
    if (!trimmed || trimmed.startsWith('#')) continue

    const indent = line.search(/\S/)
    const current = stack[stack.length - 1]

    // 弹出缩进级别更高的项
    while (stack.length > 1 && stack[stack.length - 1].indent >= indent) {
      stack.pop()
    }

    // 数组项
    if (trimmed.startsWith('- ')) {
      const content = trimmed.substring(2)
      const parent = stack[stack.length - 1]

      if (parent && !Array.isArray(parent.obj)) {
        // 将父对象转换为数组
        const key = Object.keys(parent.obj).pop() || ''
        if (key) {
          const arr = [parent.obj[key]]
          parent.obj[key] = arr
          stack.push({ obj: arr[0], indent })
        }
      }
    }

    // 键值对
    const colonIndex = trimmed.indexOf(':')
    if (colonIndex > 0) {
      const key = trimmed.substring(0, colonIndex).trim()
      let value: any = trimmed.substring(colonIndex + 1).trim()

      if (value === 'null' || value === '') {
        value = null
      } else if (value === 'true') {
        value = true
      } else if (value === 'false') {
        value = false
      } else if (!isNaN(Number(value))) {
        value = Number(value)
      } else if (value.startsWith('"') || value.startsWith("'")) {
        value = value.slice(1, -1)
      }

      const parent = stack[stack.length - 1]
      if (parent && Array.isArray(parent.obj)) {
        parent.obj.push({ [key]: value })
        stack.push({ obj: parent.obj[parent.obj.length - 1], indent })
      } else if (parent) {
        parent.obj[key] = value
        if (typeof value === 'object' && value !== null) {
          stack.push({ obj: value, indent })
        }
      }
    }
  }

  return result
}

const handleSaveYAML = async () => {
  if (!props.clusterId || !selectedPolicy.value) return

  saving.value = true
  try {
    // 尝试将 YAML 转回 JSON，如果失败则使用原始 JSON
    let jsonData = originalJsonData.value
    try {
      jsonData = yamlToJson(yamlContent.value)
      // 确保基本的元数据存在
      if (!jsonData.metadata) {
        jsonData.metadata = {}
      }
      if (!jsonData.metadata.name && selectedPolicy.value) {
        jsonData.metadata.name = selectedPolicy.value.name
      }
      if (!jsonData.metadata.namespace && selectedPolicy.value) {
        jsonData.metadata.namespace = selectedPolicy.value.namespace
      }
      if (!jsonData.apiVersion) {
        jsonData.apiVersion = 'networking.k8s.io/v1'
      }
      if (!jsonData.kind) {
        jsonData.kind = 'NetworkPolicy'
      }
    } catch (e) {
      console.warn('YAML 解析失败，使用原始 JSON:', e)
      jsonData = originalJsonData.value
    }

    await updateNetworkPolicyYAML(
      props.clusterId,
      selectedPolicy.value.namespace,
      selectedPolicy.value.name,
      jsonData
    )
    ElMessage.success('保存成功')
    yamlDialogVisible.value = false
    emit('refresh')
    await loadPolicies()
  } catch (error) {
    console.error(error)
    ElMessage.error('保存失败')
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

const handleDelete = async (policy: NetworkPolicyDetailInfo) => {
  if (!props.clusterId) return
  try {
    await ElMessageBox.confirm(`确定要删除 NetworkPolicy ${policy.name} 吗？`, '删除确认', { type: 'error' })
    await deleteNetworkPolicy(props.clusterId, policy.namespace, policy.name)
    ElMessage.success('删除成功')
    emit('refresh')
    await loadPolicies()
  } catch (error) {
    if (error !== 'cancel') {
      console.error(error)
      ElMessage.error('删除失败')
    }
  }
}

watch(() => props.clusterId, () => {
  loadPolicies()
  loadNamespaces()
})

watch(() => props.namespace, () => {
  filterNamespace.value = props.namespace || ''
  loadPolicies()
})

// 修复 YAML 弹窗打开时页面偏移的问题
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

onMounted(() => {
  loadPolicies()
  loadNamespaces()
})
</script>

<style scoped>
.networkpolicy-list {
  width: 100%;
}

.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 16px;
}

.search-input {
  width: 280px;
}

.filter-select {
  width: 180px;
}

.search-icon {
  color: #d4af37;
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
  color: #d4af37;
  font-size: 18px;
}

.name-text {
  font-weight: 500;
  color: #d4af37;
}

.selector-display {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

.selector-item {
  font-size: 12px;
  padding: 2px 6px;
  background: #f0f0f0;
  border-radius: 3px;
  color: #606266;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.action-btn {
  color: #d4af37;
  transition: all 0.3s;
}

.action-btn:hover {
  color: #bfa13f;
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
  border: 1px solid #d4af37;
  border-radius: 6px;
  overflow: hidden;
  background-color: #000000;
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
  background-color: #000000;
  color: #d4af37;
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

.yaml-dialog :deep(.el-dialog__body) {
  padding: 0;
  background-color: #1a1a1a;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
