<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-safe />
        </div>
        <div>
          <div class="page-title">角色管理</div>
          <div class="page-desc">管理系统角色权限，支持角色创建、编辑与权限分配</div>
        </div>
      </div>
    </div>

    <!-- 搜索区域 -->
    <a-card class="search-card" :bordered="false">
      <a-row :gutter="16" align="center">
        <a-col :flex="'auto'">
          <a-space :size="16" wrap>
            <a-space>
              <span class="search-label">角色名称:</span>
              <a-input v-model="searchForm.name" placeholder="搜索角色名称" allow-clear style="width: 200px" @press-enter="handleSearch" />
            </a-space>
            <a-space>
              <span class="search-label">角色编码:</span>
              <a-input v-model="searchForm.code" placeholder="搜索角色编码" allow-clear style="width: 200px" @press-enter="handleSearch" />
            </a-space>
            <a-space>
              <span class="search-label">状态:</span>
              <a-select v-model="searchForm.status" placeholder="全部" allow-clear style="width: 120px" @change="handleSearch">
                <a-option :value="1">启用</a-option>
                <a-option :value="0">禁用</a-option>
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
        <a-col :flex="'none'">
          <a-button v-permission="'roles:create'" type="primary" @click="handleAdd">
            <template #icon><icon-plus /></template>
            新增角色
          </a-button>
        </a-col>
      </a-row>
    </a-card>

    <!-- 角色表格 -->
    <a-card class="table-card" :bordered="false">
      <a-table
        :data="filteredRoleList"
        :loading="loading"
        row-key="ID"
        :pagination="tablePagination"
        @page-change="handlePageChange"
        @page-size-change="handlePageSizeChange"
      >
        <template #columns>
          <a-table-column title="ID" data-index="ID" :width="80" align="center" />
          <a-table-column title="角色名称" :min-width="150">
            <template #cell="{ record }">
              <div class="role-name-cell">
                <icon-user-group class="role-icon" />
                <span>{{ record.name }}</span>
              </div>
            </template>
          </a-table-column>
          <a-table-column title="角色编码" data-index="code" :min-width="150" />
          <a-table-column title="描述" :min-width="200">
            <template #cell="{ record }">
              <span class="desc-text">{{ record.description || '-' }}</span>
            </template>
          </a-table-column>
          <a-table-column title="状态" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.status === 1 ? 'green' : 'red'" size="small">
                {{ record.status === 1 ? '启用' : '禁用' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="createTime" :min-width="180" />
          <a-table-column title="操作" :width="180" align="center" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-tooltip content="授权">
                  <a-link v-permission="'roles:assign-menus'" status="success" @click="handlePermission(record)">
                    <icon-settings />
                  </a-link>
                </a-tooltip>
                <a-link v-permission="'roles:update'" @click="handleEdit(record)">编辑</a-link>
                <a-popconfirm :content="`确定要删除角色「${record.name}」吗？`" @ok="handleDelete(record)">
                  <a-link v-permission="'roles:delete'" status="danger">删除</a-link>
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
      @cancel="handleDialogClose"
      :ok-loading="submitting"
    >
      <a-form :model="roleForm" layout="vertical" ref="formRef">
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="角色名称" field="name" :rules="[{ required: true, message: '请输入角色名称' }, { minLength: 2, maxLength: 50, message: '长度 2-50 个字符' }]">
              <a-input v-model="roleForm.name" placeholder="请输入角色名称" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="角色编码" field="code" :rules="[{ required: true, message: '请输入角色编码' }, { minLength: 2, maxLength: 50, message: '长度 2-50 个字符' }]">
              <a-input v-model="roleForm.code" placeholder="请输入角色编码" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="16">
          <a-col :span="12">
            <a-form-item label="显示顺序" field="sort">
              <a-input-number v-model="roleForm.sort" :min="0" style="width: 100%" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="状态" field="status">
              <a-radio-group v-model="roleForm.status" type="button">
                <a-radio :value="1">启用</a-radio>
                <a-radio :value="0">禁用</a-radio>
              </a-radio-group>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item label="描述" field="description">
          <a-textarea v-model="roleForm.description" placeholder="请输入角色描述" :auto-size="{ minRows: 3, maxRows: 6 }" />
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 授权对话框 -->
    <a-modal
      v-model:visible="permissionDialogVisible"
      title="角色授权"
      :width="700"
      :unmount-on-close="true"
      @ok="handlePermissionSubmit"
      @cancel="handlePermissionDialogClose"
      :ok-loading="submitting"
      ok-text="保存授权"
    >
      <a-alert type="info" style="margin-bottom: 16px;">
        为角色 <strong>{{ currentRole.name }}</strong> 分配菜单权限。勾选的菜单及其下属接口将被授权给该角色。
      </a-alert>

      <a-spin :loading="menuLoading" style="width: 100%; min-height: 300px;">
        <div class="permission-tree-wrapper">
          <a-tree
            ref="menuTreeRef"
            :data="treeData"
            checkable
            v-model:checked-keys="checkedKeys"
            :half-checked-keys="halfCheckedKeys"
            :default-expand-all="true"
            @check="handleTreeCheck"
          >
            <template #title="node">
              <div class="tree-node">
                <span class="tree-node-label">{{ node.title }}</span>
                <a-tag v-if="node.raw?.type === 1" size="small" color="gray">目录</a-tag>
                <a-tag v-else-if="node.raw?.type === 2" size="small" color="arcoblue">菜单</a-tag>
                <a-tag v-else-if="node.raw?.type === 3" size="small" color="orangered">按钮</a-tag>
                <template v-if="node.raw?.type === 3 && node.raw?.apis && node.raw.apis.length > 0">
                  <span v-for="(api, idx) in node.raw.apis" :key="idx" class="api-info">
                    <a-tag size="small" :color="getMethodColor(api.apiMethod)">{{ api.apiMethod }}</a-tag>
                    <span class="api-path">{{ api.apiPath }}</span>
                  </span>
                </template>
                <template v-else-if="node.raw?.type === 3 && node.raw?.apiPath">
                  <span class="api-info">
                    <a-tag size="small" :color="getMethodColor(node.raw.apiMethod)">{{ node.raw.apiMethod }}</a-tag>
                    <span class="api-path">{{ node.raw.apiPath }}</span>
                  </span>
                </template>
              </div>
            </template>
          </a-tree>
        </div>
      </a-spin>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconSafe,
  IconSearch,
  IconRefresh,
  IconPlus,
  IconUserGroup,
  IconSettings,
} from '@arco-design/web-vue/es/icon'
import { getRoleList, createRole, updateRole, deleteRole, getRoleMenus, assignRoleMenus } from '@/api/role'
import { getMenuTree } from '@/api/menu'

const loading = ref(false)
const submitting = ref(false)
const menuLoading = ref(false)

// 对话框
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const formRef = ref()

// 授权对话框
const permissionDialogVisible = ref(false)
const currentRole = ref<any>({})
const menuTree = ref<any[]>([])
const treeData = ref<any[]>([])
const menuTreeRef = ref()
const checkedKeys = ref<(string | number)[]>([])
const halfCheckedKeys = ref<(string | number)[]>([])

// 分页
const pagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

const tablePagination = computed(() => ({
  current: pagination.page,
  pageSize: pagination.pageSize,
  total: pagination.total,
  showTotal: true,
  showPageSize: true,
  pageSizeOptions: [10, 20, 50, 100],
}))

// 角色列表
const roleList = ref<any[]>([])

// 搜索
const searchForm = reactive({
  name: '',
  code: '',
  status: undefined as number | undefined
})

const filteredRoleList = computed(() => {
  let result = [...roleList.value]
  if (searchForm.name) {
    result = result.filter(item => item.name?.includes(searchForm.name))
  }
  if (searchForm.code) {
    result = result.filter(item => item.code?.includes(searchForm.code))
  }
  if (searchForm.status !== undefined) {
    result = result.filter(item => item.status === searchForm.status)
  }
  return result
})

// 角色表单
const roleForm = reactive({
  id: 0,
  name: '',
  code: '',
  description: '',
  status: 1,
  sort: 0
})

const handleSearch = () => {
  pagination.page = 1
}

const handleReset = () => {
  searchForm.name = ''
  searchForm.code = ''
  searchForm.status = undefined
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadRoles()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  pagination.page = 1
  loadRoles()
}

// HTTP 方法颜色
const getMethodColor = (method: string) => {
  switch (method) {
    case 'GET': return 'green'
    case 'POST': return 'arcoblue'
    case 'PUT': return 'orangered'
    case 'DELETE': return 'red'
    default: return 'gray'
  }
}

// 加载角色列表
const loadRoles = async () => {
  loading.value = true
  try {
    const res: any = await getRoleList({
      page: pagination.page,
      pageSize: pagination.pageSize
    })
    roleList.value = res.list || []
    pagination.total = res.total || 0
  } catch {
    Message.error('获取角色列表失败')
  } finally {
    loading.value = false
  }
}

const resetForm = () => {
  Object.assign(roleForm, { id: 0, name: '', code: '', description: '', status: 1, sort: 0 })
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  resetForm()
  dialogTitle.value = '新增角色'
  isEdit.value = false
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  Object.assign(roleForm, {
    id: row.ID || row.id,
    name: row.name,
    code: row.code,
    description: row.description || '',
    status: row.status,
    sort: row.sort || 0
  })
  dialogTitle.value = '编辑角色'
  isEdit.value = true
  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await deleteRole(row.ID || row.id)
    Message.success('删除成功')
    loadRoles()
  } catch {
    Message.error('删除失败')
  }
}

const handleSubmit = async () => {
  const errors = await formRef.value?.validate()
  if (errors) return

  submitting.value = true
  try {
    if (isEdit.value) {
      await updateRole(roleForm.id, { ...roleForm })
      Message.success('更新成功')
    } else {
      await createRole({ ...roleForm })
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadRoles()
  } catch {
    Message.error(isEdit.value ? '更新失败' : '创建失败')
  } finally {
    submitting.value = false
  }
}

const handleDialogClose = () => {
  resetForm()
  dialogVisible.value = false
}

// === 授权相关 ===

// 将后端菜单树转换为 Arco Tree 格式
const convertToTreeData = (nodes: any[]): any[] => {
  return nodes.map(node => {
    const item: any = {
      key: node.ID || node.id,
      title: node.name,
      raw: node, // 保留原始数据供 slot 使用
    }
    if (node.children && node.children.length > 0) {
      item.children = convertToTreeData(node.children)
    }
    return item
  })
}

// 从菜单树中获取叶子节点ID
const getLeafMenuIdsFromTree = (tree: any[], authorizedIds: number[]): number[] => {
  const leafIds: number[] = []
  const traverse = (nodes: any[]) => {
    nodes.forEach(node => {
      const nodeId = node.ID || node.id
      if (authorizedIds.includes(nodeId)) {
        if (!node.children || node.children.length === 0) {
          leafIds.push(nodeId)
        } else {
          traverse(node.children)
        }
      }
    })
  }
  traverse(tree)
  return leafIds
}

// 打开授权对话框
const handlePermission = async (row: any) => {
  currentRole.value = row
  permissionDialogVisible.value = true

  menuLoading.value = true
  try {
    // 加载菜单树
    const treeRes: any = await getMenuTree()
    menuTree.value = treeRes || []
    treeData.value = convertToTreeData(menuTree.value)

    // 加载角色已有权限
    const roleId = row.ID || row.id
    const roleRes: any = await getRoleMenus(roleId)
    const menus = roleRes?.menus || []
    const allMenuIds = menus.map((m: any) => m.ID || m.id).filter((id: number) => id && id > 0)

    // Arco tree 的 checkedKeys 需要设置叶子节点，父节点会自动半选
    const leafIds = getLeafMenuIdsFromTree(menuTree.value, allMenuIds)
    checkedKeys.value = leafIds
    halfCheckedKeys.value = []
  } catch {
    Message.error('获取权限数据失败')
  } finally {
    menuLoading.value = false
  }
}

// 树节点勾选
const handleTreeCheck = (newCheckedKeys: (string | number)[], data: { checkedKeys: (string | number)[]; halfCheckedKeys: (string | number)[]; node?: any; event?: Event }) => {
  checkedKeys.value = newCheckedKeys
  halfCheckedKeys.value = data.halfCheckedKeys || []
}

// 提交授权
const handlePermissionSubmit = async () => {
  submitting.value = true
  try {
    // 合并完全选中 + 半选中的节点
    const allKeys = [...new Set([...checkedKeys.value, ...halfCheckedKeys.value])]
    const validKeys = allKeys.filter(key => key && Number(key) > 0).map(Number)

    const roleId = currentRole.value.ID || currentRole.value.id

    await assignRoleMenus(roleId, validKeys)
    Message.success('授权成功')
    permissionDialogVisible.value = false
  } catch {
    Message.error('授权失败')
  } finally {
    submitting.value = false
  }
}

const handlePermissionDialogClose = () => {
  currentRole.value = {}
  menuTree.value = []
  treeData.value = []
  checkedKeys.value = []
  halfCheckedKeys.value = []
}

onMounted(() => {
  loadRoles()
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
}

/* 角色名称 */
.role-name-cell {
  display: flex;
  align-items: center;
  gap: 8px;
}

.role-icon {
  color: var(--ops-primary, #165dff);
  font-size: 16px;
  flex-shrink: 0;
}

.desc-text {
  color: var(--ops-text-secondary, #4e5969);
}

/* 授权树 */
.permission-tree-wrapper {
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: 8px;
  padding: 12px;
  max-height: 500px;
  overflow-y: auto;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.tree-node-label {
  font-size: 14px;
  color: var(--ops-text-primary, #1d2129);
}

.api-info {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  margin-left: 4px;
}

.api-path {
  font-size: 11px;
  color: var(--ops-text-tertiary, #86909c);
}
</style>
