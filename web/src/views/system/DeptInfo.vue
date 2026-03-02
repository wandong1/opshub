<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-common />
        </div>
        <div>
          <div class="page-title">部门信息</div>
          <div class="page-desc">管理组织架构，支持公司、中心、部门三级层级结构</div>
        </div>
      </div>
    </div>

    <!-- 搜索区域 -->
    <a-card class="search-card" :bordered="false">
      <a-row :gutter="16" align="center">
        <a-col :flex="'auto'">
          <a-space :size="16" wrap>
            <a-space>
              <span class="search-label">部门名称:</span>
              <a-input v-model="searchForm.deptName" placeholder="请输入" allow-clear style="width: 200px" @press-enter="handleSearch" />
            </a-space>
            <a-space>
              <span class="search-label">部门状态:</span>
              <a-select v-model="searchForm.deptStatus" placeholder="请选择" allow-clear style="width: 140px" @change="handleSearch">
                <a-option :value="1">正常</a-option>
                <a-option :value="0">停用</a-option>
              </a-select>
            </a-space>
            <a-button type="primary" @click="handleSearch">
              <template #icon><icon-search /></template>
              搜索
            </a-button>
            <a-button @click="handleReset">
              <template #icon><icon-refresh /></template>
              重置
            </a-button>
          </a-space>
        </a-col>
      </a-row>
    </a-card>

    <!-- 部门树表格 -->
    <a-card class="table-card" :bordered="false">
      <template #title>
        <span class="card-title">部门列表</span>
      </template>
      <template #extra>
        <a-space>
          <a-button v-permission="'depts:create'" type="primary" @click="handleAdd">
            <template #icon><icon-plus /></template>
            新增部门
          </a-button>
          <a-button @click="toggleExpandAll">
            <template #icon><icon-swap /></template>
            {{ isExpandAll ? '折叠全部' : '展开全部' }}
          </a-button>
          <a-button @click="loadDeptTree">
            <template #icon><icon-refresh /></template>
          </a-button>
        </a-space>
      </template>

      <a-table
        ref="tableRef"
        :data="filteredDeptTree"
        :loading="loading"
        row-key="id"
        :default-expand-all-rows="isExpandAll"
        :expanded-keys="expandedKeys"
        @expand="handleExpand"
        :pagination="false"
        :key="tableKey"
      >
        <template #columns>
          <a-table-column title="部门名称" data-index="deptName" :width="350">
            <template #cell="{ record }">
              <a-space>
                <icon-home v-if="record.deptType === 1" style="color: var(--ops-primary);" />
                <icon-location v-else-if="record.deptType === 2" style="color: var(--ops-success);" />
                <icon-folder v-else style="color: var(--ops-warning);" />
                <span>{{ record.deptName }}</span>
              </a-space>
            </template>
          </a-table-column>
          <a-table-column title="部门编码" data-index="code" :width="140" />
          <a-table-column title="部门类型" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag v-if="record.deptType === 1" color="arcoblue" size="small">公司</a-tag>
              <a-tag v-else-if="record.deptType === 2" color="green" size="small">中心</a-tag>
              <a-tag v-else color="orangered" size="small">部门</a-tag>
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.deptStatus === 1 ? 'green' : 'red'" size="small">
                {{ record.deptStatus === 1 ? '正常' : '停用' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="createTime" :width="180" />
          <a-table-column title="操作" :width="120" align="center" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-link v-permission="'depts:update'" @click="handleEdit(record)">编辑</a-link>
                <a-popconfirm :content="getDeleteMsg(record)" @ok="handleDelete(record)">
                  <a-link v-permission="'depts:delete'" status="danger">删除</a-link>
                </a-popconfirm>
              </a-space>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </a-card>

    <!-- 新增/编辑对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="dialogTitle"
      :width="600"
      :unmount-on-close="true"
      @ok="handleSubmit"
      @cancel="dialogVisible = false"
    >
      <a-form :model="deptForm" layout="vertical" ref="formRef">
        <a-form-item label="部门类型" field="deptType" :rules="[{ required: true, message: '请选择部门类型' }]">
          <a-radio-group v-model="deptForm.deptType" type="button" :disabled="isEdit" @change="handleDeptTypeChange">
            <a-radio :value="1">公司</a-radio>
            <a-radio :value="2">中心</a-radio>
            <a-radio :value="3">部门</a-radio>
          </a-radio-group>
          <div class="form-tip">部门类型决定层级关系：公司 > 中心 > 部门</div>
        </a-form-item>

        <a-form-item
          v-if="deptForm.deptType !== 1 && !isRootDept"
          label="上级部门"
          field="parentId"
        >
          <a-cascader
            v-model="parentPath"
            :options="filteredParentOptions"
            :field-names="{ value: 'id', label: 'label' }"
            placeholder="请选择上级部门"
            allow-clear
            check-strictly
            style="width: 100%"
            @change="handleParentChange"
          />
          <div class="form-tip" v-if="deptForm.deptType === 2">中心的上级只能是公司</div>
          <div class="form-tip" v-else-if="deptForm.deptType === 3">部门的上级可以是公司或中心</div>
        </a-form-item>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="部门名称" field="deptName" :rules="[{ required: true, message: '请输入部门名称' }, { minLength: 2, maxLength: 50, message: '长度在 2 到 50 个字符' }]">
              <a-input v-model="deptForm.deptName" placeholder="请输入部门名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="部门编码" field="code" :rules="[{ required: true, message: '请输入部门编码' }, { minLength: 2, maxLength: 50, message: '长度在 2 到 50 个字符' }]">
              <a-input v-model="deptForm.code" placeholder="请输入部门编码" />
            </a-form-item>
          </a-col>
        </a-row>

        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="显示顺序" field="sort">
              <a-input-number v-model="deptForm.sort" :min="0" style="width: 100%;" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态" field="deptStatus" :rules="[{ required: true, message: '请选择状态' }]">
              <a-radio-group v-model="deptForm.deptStatus" type="button">
                <a-radio :value="1">正常</a-radio>
                <a-radio :value="0">停用</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
        </a-row>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, nextTick, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconCommon,
  IconSearch,
  IconRefresh,
  IconPlus,
  IconSwap,
  IconHome,
  IconLocation,
  IconFolder,
} from '@arco-design/web-vue/es/icon'
import {
  getDepartmentTree,
  getParentOptions,
  createDepartment,
  updateDepartment,
  deleteDepartment
} from '@/api/department'

const loading = ref(false)
const submitting = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const isRootDept = ref(false)
const formRef = ref()
const tableRef = ref()
const tableKey = ref(0)

// 部门树数据
const deptTree = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  deptName: '',
  deptStatus: undefined as number | undefined
})

// 展开/折叠
const isExpandAll = ref(true)
const expandedKeys = ref<(string | number)[]>([])

// 收集所有有children的节点id
const collectExpandableKeys = (nodes: any[]): (string | number)[] => {
  const keys: (string | number)[] = []
  for (const node of nodes) {
    if (node.children && node.children.length > 0) {
      keys.push(node.id)
      keys.push(...collectExpandableKeys(node.children))
    }
  }
  return keys
}

// 过滤后的部门树
const filteredDeptTree = computed(() => {
  if (!searchForm.deptName && searchForm.deptStatus === undefined) {
    return deptTree.value
  }
  return filterTree(deptTree.value)
})

// 递归过滤树节点
const filterTree = (nodes: any[]): any[] => {
  const result: any[] = []
  for (const node of nodes) {
    const matchName = !searchForm.deptName || node.deptName?.includes(searchForm.deptName)
    const matchStatus = searchForm.deptStatus === undefined || node.deptStatus === searchForm.deptStatus

    let filteredChildren: any[] = []
    if (node.children && node.children.length > 0) {
      filteredChildren = filterTree(node.children)
    }

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
const parentPath = ref<(string | number)[]>([])

// 根据部门类型过滤上级选项
const filteredParentOptions = computed(() => {
  const currentType = deptForm.deptType

  const deptTypeMap = new Map<number, number>()
  const buildMap = (nodes: any[]) => {
    for (const node of nodes) {
      deptTypeMap.set(node.id, node.deptType)
      if (node.children) buildMap(node.children)
    }
  }
  buildMap(deptTree.value)

  const filterNodes = (nodes: any[]): any[] => {
    const result: any[] = []
    for (const node of nodes) {
      const nodeType = deptTypeMap.get(node.id)
      if (currentType === 2) {
        if (nodeType === 1) {
          result.push({ ...node, children: undefined })
        }
      } else if (currentType === 3) {
        if (nodeType === 1) {
          const filteredNode = { ...node }
          if (node.children && node.children.length > 0) {
            filteredNode.children = node.children
              .map((child: any) => {
                const childType = deptTypeMap.get(child.id)
                if (childType === 2) return { ...child, children: undefined }
                return null
              })
              .filter((c: any) => c !== null)
          }
          result.push(filteredNode)
        }
      }
    }
    return result
  }

  return currentType === 1 ? [] : filterNodes(parentOptions.value)
})

// 部门表单
const deptForm = reactive({
  id: 0,
  parentId: 0,
  deptType: 3,
  deptName: '',
  code: '',
  sort: 0,
  deptStatus: 1
})

// 搜索
const handleSearch = () => {
  if (searchForm.deptName || searchForm.deptStatus !== undefined) {
    isExpandAll.value = true
    nextTick(() => {
      expandedKeys.value = collectExpandableKeys(filteredDeptTree.value)
      tableKey.value++
    })
  }
}

watch([() => searchForm.deptName, () => searchForm.deptStatus], () => {
  handleSearch()
})

const handleReset = () => {
  searchForm.deptName = ''
  searchForm.deptStatus = undefined
  isExpandAll.value = false
  expandedKeys.value = []
  tableKey.value++
}

const toggleExpandAll = () => {
  isExpandAll.value = !isExpandAll.value
  if (isExpandAll.value) {
    expandedKeys.value = collectExpandableKeys(filteredDeptTree.value)
  } else {
    expandedKeys.value = []
  }
  tableKey.value++
}

const handleExpand = (rowKey: string | number, record: any) => {
  const idx = expandedKeys.value.indexOf(rowKey)
  if (idx >= 0) {
    expandedKeys.value.splice(idx, 1)
  } else {
    expandedKeys.value.push(rowKey)
  }
}

// 删除确认消息
const getDeleteMsg = (row: any) => {
  const hasChildren = row.children && row.children.length > 0
  return hasChildren
    ? `该部门下有 ${row.children.length} 个子部门，确定要删除吗？`
    : `确定要删除部门"${row.deptName}"吗？`
}

// 加载部门树
const loadDeptTree = async () => {
  loading.value = true
  try {
    const data = await getDepartmentTree()
    deptTree.value = data || []
    // 初始展开所有
    expandedKeys.value = collectExpandableKeys(deptTree.value)
    tableKey.value++
  } catch {
    Message.error('获取部门列表失败')
  } finally {
    loading.value = false
  }
}

// 加载父级选项
const loadParentOptions = async () => {
  try {
    const data = await getParentOptions()
    parentOptions.value = data || []
  } catch {
    // ignore
  }
}

// 重置表单
const resetForm = () => {
  deptForm.id = 0
  deptForm.parentId = 0
  deptForm.deptType = 3
  deptForm.deptName = ''
  deptForm.code = ''
  deptForm.sort = 0
  deptForm.deptStatus = 1
  parentPath.value = []
  isRootDept.value = false
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  resetForm()
  loadParentOptions()
  dialogTitle.value = '新增部门'
  isEdit.value = false
  isRootDept.value = false
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  Object.assign(deptForm, {
    id: row.id,
    parentId: row.parentId || 0,
    deptType: row.deptType,
    deptName: row.deptName,
    code: row.code || '',
    sort: row.sort || 0,
    deptStatus: row.deptStatus
  })
  dialogTitle.value = '编辑部门'
  isEdit.value = true
  isRootDept.value = !row.parentId || row.parentId === 0
  parentPath.value = row.parentId && row.parentId !== 0 ? [row.parentId] : []
  loadParentOptions()
  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await deleteDepartment(row.id)
    Message.success('删除成功')
    loadDeptTree()
    loadParentOptions()
  } catch (error: any) {
    Message.error(error.message || '删除失败')
  }
}

const handleDeptTypeChange = () => {
  parentPath.value = []
  deptForm.parentId = 0
}

const handleParentChange = (value: any) => {
  if (value && Array.isArray(value) && value.length > 0) {
    deptForm.parentId = value[value.length - 1]
  } else if (value && !Array.isArray(value)) {
    deptForm.parentId = value
  } else {
    deptForm.parentId = 0
  }
}

const handleSubmit = async () => {
  const errors = await formRef.value?.validate()
  if (errors) return

  submitting.value = true
  try {
    const data = { ...deptForm }
    if (isEdit.value) {
      await updateDepartment(data.id, data)
      Message.success('更新成功')
    } else {
      await createDepartment(data)
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadDeptTree()
    loadParentOptions()
  } catch (error: any) {
    Message.error(error.message || (isEdit.value ? '更新失败' : '创建失败'))
  } finally {
    submitting.value = false
  }
}

onMounted(() => {
  loadDeptTree()
  loadParentOptions()
})
</script>

<style scoped lang="scss">
.page-container {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.page-header-card {
  background: #fff;
  border-radius: var(--ops-border-radius-md, 8px);
  padding: 20px 24px;
}

.page-header-inner {
  display: flex;
  align-items: center;
  gap: 16px;
}

.page-icon {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  background: linear-gradient(135deg, var(--ops-primary, #165dff) 0%, #4080ff 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 22px;
  flex-shrink: 0;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
  line-height: 1.4;
}

.page-desc {
  font-size: 13px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 2px;
}

.search-card {
  border-radius: var(--ops-border-radius-md, 8px);
}

.search-label {
  font-size: 14px;
  color: var(--ops-text-secondary, #4e5969);
  white-space: nowrap;
}

.table-card {
  border-radius: var(--ops-border-radius-md, 8px);
  flex: 1;

  .card-title {
    font-weight: 600;
  }
}

.form-tip {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 4px;
}
</style>
