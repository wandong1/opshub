<template>
  <div class="groups-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-folder /></div>
        <div>
          <h2 class="page-title">业务分组</h2>
          <p class="page-subtitle">管理主机业务分组，支持多级层级结构</p>
        </div>
      </div>
      <div class="header-actions">
        <a-button v-permission="'asset-groups:create'" type="primary" @click="handleAdd">
          <template #icon><icon-plus /></template>
          新增分组
        </a-button>
        <a-button @click="toggleExpandAll">
          <template #icon><icon-sort /></template>
          {{ expandedKeys.length > 0 ? '折叠全部' : '展开全部' }}
        </a-button>
      </div>
    </div>

    <!-- 搜索栏 -->
    <div class="filter-bar">
      <a-input
        v-model="searchForm.name"
        placeholder="搜索分组名称..."
        allow-clear
        class="search-input"
      >
        <template #prefix>
          <icon-search />
        </template>
      </a-input>

      <a-select
        v-model="searchForm.status"
        placeholder="分组状态"
        allow-clear
        class="search-input"
      >
        <a-option :value="1">正常</a-option>
        <a-option :value="0">停用</a-option>
      </a-select>

      <a-button @click="handleReset">
        <template #icon><icon-refresh /></template>
        重置
      </a-button>
    </div>

    <!-- 表格容器 -->
    <div class="table-wrapper">
      <a-table
        :data="filteredGroupTree"
        :loading="loading"
        row-key="id"
        :bordered="{ cell: true }"
        stripe
        :default-expand-all-rows="false"
        v-model:expanded-keys="expandedKeys"
        :pagination="false"
      >
        <template #columns>
          <a-table-column title="分组名称" data-index="name" :width="300">
            <template #cell="{ record }">
              <span style="display: inline-flex; align-items: center;">
                <icon-folder v-if="!record.parentId || record.parentId === 0" style="color: #00b42a; margin-right: 8px;" />
                <icon-folder v-else style="color: #165dff; margin-right: 8px;" />
                {{ record.name }}
              </span>
            </template>
          </a-table-column>

          <a-table-column data-index="code" :width="150">
            <template #title>
              <span class="header-with-icon">
                <icon-lock class="header-icon header-icon-gold" />
                分组编码
              </span>
            </template>
          </a-table-column>

          <a-table-column title="描述" data-index="description" :width="300" ellipsis tooltip />

          <a-table-column title="主机数量" :width="120" align="center">
            <template #cell="{ record }">
              <a-tag color="arcoblue">{{ record.hostCount || 0 }}</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="中间件数量" :width="120" align="center">
            <template #cell="{ record }">
              <a-tag color="orangered">{{ record.middlewareCount || 0 }}</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="状态" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.status === 1 ? 'green' : 'red'">
                {{ record.status === 1 ? '正常' : '停用' }}
              </a-tag>
            </template>
          </a-table-column>

          <a-table-column title="创建时间" data-index="createTime" :width="180" />

          <a-table-column title="操作" :width="200" fixed="right" align="center">
            <template #cell="{ record }">
              <div class="action-buttons">
                <a-tooltip content="编辑" position="top">
                  <a-button v-permission="'asset-groups:update'" type="text" class="action-btn action-edit" @click="handleEdit(record)">
                    <template #icon><icon-edit /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="删除" position="top">
                  <a-button v-permission="'asset-groups:delete'" type="text" class="action-btn action-delete" @click="handleDelete(record)">
                    <template #icon><icon-delete /></template>
                  </a-button>
                </a-tooltip>
              </div>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <!-- 新增/编辑对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="dialogTitle"
      :width="600"
      unmount-on-close
      :mask-closable="false"
      @close="handleDialogClose"
    >
      <a-form :model="groupForm" :rules="rules" ref="formRef" auto-label-width layout="horizontal">
        <a-form-item label="上级分组" field="parentId" v-if="showParentSelect && !isRootGroup">
          <a-cascader
            v-model="parentPath"
            :options="parentOptions"
            :field-names="{ value: 'id', label: 'label' }"
            check-strictly
            allow-search
            allow-clear
            placeholder="请选择上级分组"
            style="width: 100%"
            @change="handleParentChange"
          />
          <div class="form-tip">不选择则为顶级分组</div>
        </a-form-item>

        <a-form-item label="分组名称" field="name">
          <a-input v-model="groupForm.name" placeholder="请输入分组名称" />
        </a-form-item>

        <a-form-item label="分组编码" field="code">
          <a-input v-model="groupForm.code" placeholder="请输入分组编码" />
        </a-form-item>

        <a-form-item label="描述" field="description">
          <a-textarea v-model="groupForm.description" :auto-size="{ minRows: 3 }" placeholder="请输入分组描述" />
        </a-form-item>

        <a-form-item label="显示顺序" field="sort">
          <a-input-number v-model="groupForm.sort" :min="0" />
        </a-form-item>

        <a-form-item label="状态" field="status">
          <a-radio-group v-model="groupForm.status">
            <a-radio :value="1">正常</a-radio>
            <a-radio :value="0">停用</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>

      <template #footer>
        <div class="dialog-footer">
          <a-button @click="dialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleSubmit" :loading="submitting">确定</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick, watch } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import {
  IconPlus,
  IconEdit,
  IconDelete,
  IconFolder,
  IconSearch,
  IconRefresh,
  IconSort,
  IconLock
} from '@arco-design/web-vue/es/icon'
import {
  getGroupTree,
  getParentOptions,
  createGroup,
  updateGroup,
  deleteGroup
} from '@/api/assetGroup'

// 加载状态
const loading = ref(false)
const submitting = ref(false)

// 对话框状态
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const isRootGroup = ref(false)

// 表单引用
const formRef = ref<FormInstance>()

// 分组树数据
const groupTree = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  name: '',
  status: undefined as number | undefined
})

// 展开/折叠状态
const expandedKeys = ref<(string | number)[]>([])

// Collect all row IDs recursively
const getAllKeys = (data: any[]): (string | number)[] => {
  const keys: (string | number)[] = []
  const traverse = (rows: any[]) => {
    rows.forEach(row => {
      keys.push(row.id)
      if (row.children?.length) traverse(row.children)
    })
  }
  traverse(data)
  return keys
}

// 过滤后的分组树
const filteredGroupTree = computed(() => {
  if (!searchForm.name && searchForm.status === undefined) {
    return groupTree.value
  }
  return filterTree(groupTree.value)
})

// 递归过滤树节点
const filterTree = (nodes: any[]): any[] => {
  const result: any[] = []

  for (const node of nodes) {
    const matchName = !searchForm.name || node.name?.includes(searchForm.name)
    const matchStatus = searchForm.status === undefined || node.status === searchForm.status

    let filteredChildren: any[] = []
    if (node.children && node.children.length > 0) {
      filteredChildren = filterTree(node.children)
    }

    // 如果当前节点匹配或有匹配的子节点，则保留
    if ((matchName && matchStatus) || filteredChildren.length > 0) {
      result.push({
        ...node,
        children: filteredChildren.length > 0 ? filteredChildren : undefined
      })
    }
  }

  return result
}

// 父级选项
const parentOptions = ref([])
const parentPath = ref<number[]>([])

// 分组表单
const groupForm = reactive({
  id: 0,
  parentId: 0,
  name: '',
  code: '',
  description: '',
  sort: 0,
  status: 1
})

// 分组类型是否禁用
const showParentSelect = computed(() => {
  // 编辑模式且不是顶级分组时显示上级选择
  return !isEdit.value || (isEdit.value && !isRootGroup.value)
})

// 表单验证规则
const rules = {
  name: [
    { required: true, message: '请输入分组名称' },
    { minLength: 2, maxLength: 100, message: '分组名称长度在 2 到 100 个字符' }
  ],
  code: [
    { required: true, message: '请输入分组编码' },
    { minLength: 2, maxLength: 50, message: '分组编码长度在 2 到 50 个字符' }
  ],
  status: [{ required: true, message: '请选择状态' }]
}

// 监听搜索条件变化
watch([() => searchForm.name, () => searchForm.status], () => {
  if (searchForm.name || searchForm.status !== undefined) {
    nextTick(() => {
      expandedKeys.value = getAllKeys(filteredGroupTree.value)
    })
  }
})

// 重置搜索
const handleReset = () => {
  searchForm.name = ''
  searchForm.status = undefined
  expandedKeys.value = []
}

// 切换全部展开/折叠
const toggleExpandAll = () => {
  if (expandedKeys.value.length > 0) {
    expandedKeys.value = []
  } else {
    expandedKeys.value = getAllKeys(filteredGroupTree.value)
  }
}

// 加载分组树
const loadGroupTree = async () => {
  loading.value = true
  try {
    const data = await getGroupTree()
    // 后端已经返回树形结构，直接使用
    groupTree.value = data || []
  } catch (error) {
    Message.error('获取分组列表失败')
  } finally {
    loading.value = false
  }
}

// 加载父级选项
const loadParentOptions = async () => {
  try {
    const data = await getParentOptions()
    // 后端已经返回树形结构，直接使用
    parentOptions.value = data || []
  } catch (error) {
  }
}

// 重置表单
const resetForm = () => {
  groupForm.id = 0
  groupForm.parentId = 0
  groupForm.name = ''
  groupForm.code = ''
  groupForm.description = ''
  groupForm.sort = 0
  groupForm.status = 1
  parentPath.value = []
  isRootGroup.value = false
  formRef.value?.clearValidate()
}

// 新增顶级分组
const handleAdd = () => {
  resetForm()
  loadParentOptions()
  dialogTitle.value = '新增分组'
  isEdit.value = false
  isRootGroup.value = false
  dialogVisible.value = true
}

// 编辑分组
const handleEdit = (row: any) => {
  Object.assign(groupForm, {
    id: row.id,
    parentId: row.parentId || 0,
    name: row.name,
    code: row.code || '',
    description: row.description || '',
    sort: row.sort || 0,
    status: row.status
  })
  dialogTitle.value = '编辑分组'
  isEdit.value = true
  isRootGroup.value = !row.parentId || row.parentId === 0
  if (row.parentId && row.parentId !== 0) {
    parentPath.value = [row.parentId]
  } else {
    parentPath.value = []
  }
  loadParentOptions()
  dialogVisible.value = true
}

// 删除分组
const handleDelete = (row: any) => {
  const hasChildren = row.children && row.children.length > 0
  const confirmMsg = hasChildren
    ? `该分组下有 ${row.children.length} 个子分组，确定要删除分组"${row.name}"及其所有子分组吗？`
    : `确定要删除分组"${row.name}"吗？`

  Modal.warning({
    title: '提示',
    content: confirmMsg,
    hideCancel: false,
    onOk: async () => {
      try {
        await deleteGroup(row.id)
        Message.success('删除成功')
        loadGroupTree()
        loadParentOptions()
      } catch (error: any) {
        Message.error(error.message || '删除失败')
      }
    }
  })
}

const handleParentChange = (value: number[]) => {
  if (value && value.length > 0) {
    const parentId = value[value.length - 1]
    groupForm.parentId = parentId
  } else {
    groupForm.parentId = 0
  }
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  const errors = await formRef.value.validate()
  if (errors) return

  submitting.value = true
  try {
    const data = { ...groupForm }

    if (isEdit.value) {
      await updateGroup(data.id, data)
      Message.success('更新成功')
    } else {
      await createGroup(data)
      Message.success('创建成功')
    }

    dialogVisible.value = false
    loadGroupTree()
    loadParentOptions()
  } catch (error: any) {
    Message.error(error.message || (isEdit.value ? '更新失败' : '创建失败'))
  } finally {
    submitting.value = false
  }
}

// 对话框关闭事件
const handleDialogClose = () => {
  formRef.value?.resetFields()
  resetForm()
}

onMounted(() => {
  loadGroupTree()
  loadParentOptions()
})
</script>

<style scoped>
.groups-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: center;
  gap: 12px;
}

.page-title-icon {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  background: var(--ops-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
}

.page-title {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
  color: var(--ops-text-primary);
}

.page-subtitle {
  margin: 2px 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary);
}

.header-actions {
  display: flex;
  gap: 12px;
  align-items: center;
}

/* 搜索栏 */
.filter-bar {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.search-input {
  width: 280px;
}

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
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

.header-icon-gold {
  color: #d4af37;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
  justify-content: center;
}

.action-btn {
  width: 32px;
  height: 32px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.2s ease;
}

.action-btn:hover {
  transform: scale(1.1);
}

.action-edit:hover {
  background-color: #e8f4ff;
  color: #165dff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f53f3f;
}

.black-button {
  background-color: #000000 !important;
  color: #ffffff !important;
  border-color: #000000 !important;
  border-radius: 8px;
  padding: 10px 20px;
  font-weight: 500;
}

.black-button:hover {
  background-color: #333333 !important;
  border-color: #333333 !important;
}

/* 表单提示 */
.form-tip {
  font-size: 12px;
  color: #999;
  margin-top: 4px;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>
