<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-user />
        </div>
        <div>
          <div class="page-title">用户管理</div>
          <div class="page-desc">管理系统用户，支持按部门筛选、角色分配与岗位关联</div>
        </div>
      </div>
    </div>

    <div class="content-wrapper">
      <!-- 左侧部门树 -->
      <a-card class="dept-tree-card" :bordered="false">
        <template #title>
          <span class="card-title">
            <icon-common style="margin-right: 6px;" />
            部门组织
          </span>
        </template>
        <a-tree
          ref="treeRef"
          :data="departmentTreeData"
          :field-names="{ key: 'id', title: 'deptName', children: 'children' }"
          :default-expand-all="true"
          :selected-keys="selectedTreeKeys"
          @select="handleNodeClick"
        >
          <template #title="nodeData">
            <span class="tree-node">
              <span class="node-label">{{ nodeData.deptName }}</span>
              <span class="node-count">({{ nodeData.userCount || 0 }})</span>
            </span>
          </template>
        </a-tree>
      </a-card>

      <!-- 右侧用户列表 -->
      <div class="user-list-panel">
        <!-- 当前选中部门 -->
        <a-alert v-if="selectedDepartment" class="dept-alert" type="info" :show-icon="false" closable @close="clearDepartmentSelection">
          <template #default>
            <div class="dept-alert-content">
              <span>
                <span class="label">当前部门：</span>
                <span class="path">{{ selectedDepartmentPath }}</span>
              </span>
              <a-link @click="clearDepartmentSelection">查看全部用户</a-link>
            </div>
          </template>
        </a-alert>

        <!-- 搜索区域 -->
        <a-card class="search-card" :bordered="false">
          <a-row :gutter="16" align="center">
            <a-col :flex="'auto'">
              <a-space :size="16" wrap>
                <a-space>
                  <span class="search-label">关键词:</span>
                  <a-input v-model="searchForm.keyword" placeholder="用户名/邮箱" allow-clear style="width: 200px" @press-enter="handleSearch" />
                </a-space>
                <a-button type="primary" @click="handleSearch">
                  <template #icon><icon-search /></template>
                  搜索
                </a-button>
                <a-button @click="resetSearch">
                  <template #icon><icon-refresh /></template>
                  重置
                </a-button>
              </a-space>
            </a-col>
            <a-col :flex="'none'">
              <a-button v-permission="'users:create'" type="primary" @click="handleAdd">
                <template #icon><icon-plus /></template>
                新增用户
              </a-button>
            </a-col>
          </a-row>
        </a-card>

        <!-- 用户表格 -->
        <a-card class="table-card" :bordered="false">
          <a-table
            :data="userList"
            :loading="loading"
            row-key="id"
            :pagination="tablePagination"
            @page-change="handlePageChange"
            @page-size-change="handlePageSizeChange"
          >
            <template #columns>
              <a-table-column title="用户" :width="220">
                <template #cell="{ record }">
                  <div class="user-cell">
                    <a-avatar v-if="record.avatar" :size="36" :image-url="record.avatar" />
                    <a-avatar v-else :size="36" :style="{ backgroundColor: 'var(--ops-primary)' }">
                      {{ (record.realName || record.username || '').charAt(0) }}
                    </a-avatar>
                    <div class="user-cell-info">
                      <div class="user-cell-name">{{ record.realName || record.username }}</div>
                      <div class="user-cell-sub">@{{ record.username }}</div>
                    </div>
                  </div>
                </template>
              </a-table-column>
              <a-table-column title="邮箱" data-index="email" :min-width="180" ellipsis tooltip />
              <a-table-column title="手机号" data-index="phone" :min-width="130" />
              <a-table-column title="部门" :min-width="150">
                <template #cell="{ record }">
                  {{ record.department?.name || record.department?.deptName || '-' }}
                </template>
              </a-table-column>
              <a-table-column title="状态" :width="100" align="center">
                <template #cell="{ record }">
                  <a-tag v-if="record.isLocked" color="orangered" size="small">锁定中</a-tag>
                  <a-tag v-else :color="record.status === 1 ? 'green' : 'red'" size="small">
                    {{ record.status === 1 ? '启用' : '禁用' }}
                  </a-tag>
                </template>
              </a-table-column>
              <a-table-column title="操作" :width="180" align="center" fixed="right">
                <template #cell="{ record }">
                  <a-space>
                    <a-tooltip v-if="record.isLocked" content="解锁">
                      <a-link v-permission="'users:unlock'" status="warning" @click="handleUnlock(record)">
                        <icon-unlock />
                      </a-link>
                    </a-tooltip>
                    <a-link v-permission="'users:update'" @click="handleEdit(record)">编辑</a-link>
                    <a-link v-permission="'users:reset-pwd'" @click="handleResetPassword(record)">重置密码</a-link>
                    <a-popconfirm content="确定要删除该用户吗？" @ok="handleDelete(record)">
                      <a-link v-permission="'users:delete'" status="danger">删除</a-link>
                    </a-popconfirm>
                  </a-space>
                </template>
              </a-table-column>
            </template>
          </a-table>
        </a-card>
      </div>
    </div>

    <!-- 新增/编辑对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      :title="dialogTitle"
      :width="720"
      :unmount-on-close="true"
      @ok="handleSubmit"
      @cancel="handleDialogClose"
      :ok-loading="submitLoading"
    >
      <a-form :model="userForm" layout="vertical" ref="formRef">
        <!-- 基本信息 -->
        <div class="form-section">
          <div class="form-section-title">
            <icon-user style="color: var(--ops-primary);" />
            <span>基本信息</span>
          </div>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="用户名" field="username" :rules="[{ required: true, message: '请输入用户名' }]">
                <a-input v-model="userForm.username" :disabled="isEdit" placeholder="请输入用户名" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="真实姓名" field="realName">
                <a-input v-model="userForm.realName" placeholder="请输入真实姓名" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12">
              <a-form-item label="邮箱" field="email" :rules="[{ required: true, message: '请输入邮箱' }]">
                <a-input v-model="userForm.email" placeholder="请输入邮箱" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="手机号" field="phone">
                <a-input v-model="userForm.phone" placeholder="请输入手机号" />
              </a-form-item>
            </a-col>
          </a-row>
          <a-row :gutter="16">
            <a-col :span="12" v-if="!isEdit">
              <a-form-item label="密码" field="password" :rules="[{ required: true, message: '请输入密码' }, { minLength: 6, message: '密码长度不能少于6位' }]">
                <a-input-password v-model="userForm.password" placeholder="请输入密码（至少6位）" />
              </a-form-item>
            </a-col>
            <a-col :span="12">
              <a-form-item label="状态" field="status">
                <a-radio-group v-model="userForm.status" type="button">
                  <a-radio :value="1">启用</a-radio>
                  <a-radio :value="0">禁用</a-radio>
                </a-radio-group>
              </a-form-item>
            </a-col>
          </a-row>
        </div>

        <!-- 组织信息 -->
        <div class="form-section">
          <div class="form-section-title">
            <icon-home style="color: var(--ops-primary);" />
            <span>组织信息</span>
          </div>
          <a-form-item label="部门" field="departmentId">
            <a-tree-select
              v-model="userForm.departmentId"
              :data="deptSelectTreeData"
              :field-names="{ key: 'id', title: 'label', children: 'children' }"
              placeholder="请选择部门"
              allow-clear
              style="width: 100%"
            />
          </a-form-item>
          <a-form-item label="岗位" field="positionIds">
            <a-select
              v-model="userForm.positionIds"
              :options="positionSelectOptions"
              multiple
              placeholder="请选择岗位"
              allow-clear
              style="width: 100%"
            />
          </a-form-item>
        </div>

        <!-- 权限信息 -->
        <div class="form-section">
          <div class="form-section-title">
            <icon-safe style="color: var(--ops-primary);" />
            <span>权限信息</span>
          </div>
          <a-form-item label="角色" field="roleIds">
            <a-select
              v-model="userForm.roleIds"
              :options="roleSelectOptions"
              multiple
              placeholder="请选择角色"
              allow-clear
              style="width: 100%"
            />
          </a-form-item>
        </div>

        <!-- 其他信息 -->
        <div class="form-section">
          <div class="form-section-title">
            <icon-file style="color: var(--ops-primary);" />
            <span>其他信息</span>
          </div>
          <a-form-item label="个人简介" field="bio">
            <a-textarea v-model="userForm.bio" placeholder="请输入个人简介" :auto-size="{ minRows: 3, maxRows: 6 }" />
          </a-form-item>
        </div>
      </a-form>
    </a-modal>

    <!-- 重置密码对话框 -->
    <a-modal
      v-model:visible="resetPasswordVisible"
      title="重置密码"
      :width="480"
      :unmount-on-close="true"
      @ok="handleResetPasswordSubmit"
      @cancel="resetPasswordVisible = false"
      :ok-loading="resetPasswordLoading"
    >
      <a-form :model="resetPasswordForm" layout="vertical" ref="resetPasswordFormRef">
        <a-form-item label="用户名">
          <a-input v-model="resetPasswordForm.username" disabled />
        </a-form-item>
        <a-form-item label="新密码" field="password" :rules="[{ required: true, message: '请输入新密码' }, { minLength: 6, message: '密码长度不能少于6位' }]">
          <a-input-password v-model="resetPasswordForm.password" placeholder="请输入新密码（至少6位）" />
        </a-form-item>
        <a-form-item label="确认密码" field="confirmPassword" :rules="[{ required: true, message: '请再次输入新密码' }, { validator: validateConfirmPassword }]">
          <a-input-password v-model="resetPasswordForm.confirmPassword" placeholder="请再次输入新密码" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import {
  IconUser,
  IconCommon,
  IconSearch,
  IconRefresh,
  IconPlus,
  IconUnlock,
  IconHome,
  IconSafe,
  IconFile,
} from '@arco-design/web-vue/es/icon'
import { getUserList, createUser, updateUser, deleteUser, resetUserPassword, assignUserRoles, assignUserPositions, unlockUser } from '@/api/user'
import { getDepartmentTree } from '@/api/department'
import { getAllRoles } from '@/api/role'
import { getPositionList } from '@/api/position'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const dialogTitle = ref('')
const isEdit = ref(false)
const formRef = ref()
const treeRef = ref()

// 部门树
const departmentTree = ref<any[]>([])
const selectedDepartment = ref<any>(null)
const selectedDepartmentPath = ref('')
const selectedTreeKeys = ref<(string | number)[]>([])

// 角色和岗位选项
const roleOptions = ref<any[]>([])
const positionOptions = ref<any[]>([])

// 转换为 a-tree-select 格式
const departmentTreeData = computed(() => departmentTree.value)

const deptSelectTreeData = computed(() => {
  const convert = (nodes: any[]): any[] =>
    nodes.map(node => ({
      id: node.id,
      label: node.deptName || node.name,
      children: node.children ? convert(node.children) : []
    }))
  return convert(departmentTree.value)
})

const roleSelectOptions = computed(() =>
  roleOptions.value.map(r => ({
    value: r.ID || r.id,
    label: r.name
  }))
)

const positionSelectOptions = computed(() =>
  positionOptions.value.map(p => ({
    value: p.ID || p.id,
    label: p.postName
  }))
)

// 重置密码
const resetPasswordVisible = ref(false)
const resetPasswordLoading = ref(false)
const resetPasswordFormRef = ref()
const resetPasswordForm = reactive({
  userId: 0,
  username: '',
  password: '',
  confirmPassword: ''
})

const validateConfirmPassword = (value: any, callback: (error?: string) => void) => {
  if (value !== resetPasswordForm.password) {
    callback('两次输入的密码不一致')
  } else {
    callback()
  }
}

// 搜索
const searchForm = reactive({
  keyword: '',
  departmentId: null as number | null
})

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

const userList = ref<any[]>([])

const userForm = reactive({
  id: 0,
  username: '',
  password: '',
  realName: '',
  email: '',
  phone: '',
  status: 1,
  departmentId: null as number | null,
  positionIds: [] as number[],
  roleIds: [] as number[],
  bio: ''
})

// 加载用户列表
const loadUsers = async () => {
  loading.value = true
  try {
    const res: any = await getUserList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      keyword: searchForm.keyword || undefined,
      departmentId: searchForm.departmentId || undefined
    })
    userList.value = res.list || []
    pagination.total = res.total || 0
  } catch {
    userList.value = []
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  loadUsers()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadUsers()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  pagination.page = 1
  loadUsers()
}

// 加载部门树
const loadDepartmentTree = async () => {
  try {
    const res = await getDepartmentTree()
    departmentTree.value = res || []
  } catch { /* ignore */ }
}

// 加载角色选项
const loadRoleOptions = async () => {
  try {
    const res = await getAllRoles()
    roleOptions.value = res || []
  } catch { /* ignore */ }
}

// 加载岗位选项
const loadPositionOptions = async () => {
  try {
    const res: any = await getPositionList({ page: 1, pageSize: 1000 })
    positionOptions.value = res.list || []
  } catch { /* ignore */ }
}

// 构建部门路径
const buildDepartmentPath = (node: any, path: string[] = []): string => {
  path.unshift(node.deptName || node.name)
  if (node.parentId && departmentTree.value) {
    const findParent = (nodes: any[], id: number): any => {
      for (const n of nodes) {
        if (n.id === id) return n
        if (n.children) {
          const found = findParent(n.children, id)
          if (found) return found
        }
      }
      return null
    }
    const parent = findParent(departmentTree.value, node.parentId)
    if (parent) return buildDepartmentPath(parent, path)
  }
  return path.join(' / ')
}

// 部门节点点击
const handleNodeClick = (keys: (string | number)[], data: any) => {
  if (keys.length === 0) {
    clearDepartmentSelection()
    return
  }
  const node = data.node
  selectedTreeKeys.value = keys
  selectedDepartment.value = node
  selectedDepartmentPath.value = buildDepartmentPath(node)
  searchForm.departmentId = node.id
  pagination.page = 1
  loadUsers()
}

// 清除部门选择
const clearDepartmentSelection = () => {
  selectedDepartment.value = null
  selectedDepartmentPath.value = ''
  selectedTreeKeys.value = []
  searchForm.departmentId = null
  pagination.page = 1
  loadUsers()
}

const resetSearch = () => {
  searchForm.keyword = ''
  pagination.page = 1
  loadUsers()
}

// 重置表单
const resetForm = () => {
  Object.assign(userForm, {
    id: 0, username: '', password: '', realName: '', email: '',
    phone: '', status: 1, departmentId: null, positionIds: [], roleIds: [], bio: ''
  })
  formRef.value?.clearValidate()
}

const handleAdd = () => {
  resetForm()
  isEdit.value = false
  dialogTitle.value = '新增用户'
  dialogVisible.value = true
}

const handleEdit = (row: any) => {
  isEdit.value = true
  dialogTitle.value = '编辑用户'

  userForm.id = Number(row.ID || row.id)
  userForm.username = row.username
  userForm.realName = row.realName || ''
  userForm.email = row.email || ''
  userForm.phone = row.phone || ''
  userForm.status = row.status ?? 1
  userForm.departmentId = row.departmentId ? Number(row.departmentId) : null

  if (row.positionIds && Array.isArray(row.positionIds)) {
    userForm.positionIds = row.positionIds.map((id: any) => Number(id))
  } else if (row.positions && Array.isArray(row.positions) && row.positions.length > 0) {
    userForm.positionIds = row.positions.map((p: any) => Number(p.ID || p.id))
  } else {
    userForm.positionIds = []
  }

  if (row.roleIds && Array.isArray(row.roleIds)) {
    userForm.roleIds = row.roleIds
  } else if (row.roles && Array.isArray(row.roles) && row.roles.length > 0) {
    userForm.roleIds = row.roles.map((r: any) => r.ID || r.id)
  } else {
    userForm.roleIds = []
  }

  userForm.bio = row.bio || ''
  dialogVisible.value = true
}

const handleDelete = async (row: any) => {
  try {
    await deleteUser(row.ID || row.id)
    Message.success('删除成功')
    loadUsers()
  } catch {
    Message.error('删除失败')
  }
}

const handleUnlock = (row: any) => {
  Modal.confirm({
    title: '确认解锁',
    content: `确定要解锁用户 "${row.username}" 吗？`,
    onOk: async () => {
      try {
        await unlockUser(row.ID || row.id)
        Message.success('用户已解锁')
        loadUsers()
      } catch {
        Message.error('解锁失败')
      }
    }
  })
}

const handleResetPassword = (row: any) => {
  resetPasswordForm.userId = row.ID || row.id
  resetPasswordForm.username = row.username
  resetPasswordForm.password = ''
  resetPasswordForm.confirmPassword = ''
  resetPasswordVisible.value = true
}

const handleResetPasswordSubmit = async () => {
  const errors = await resetPasswordFormRef.value?.validate()
  if (errors) return

  resetPasswordLoading.value = true
  try {
    await resetUserPassword(resetPasswordForm.userId, resetPasswordForm.password)
    Message.success('密码重置成功')
    resetPasswordVisible.value = false
  } catch {
    Message.error('密码重置失败')
  } finally {
    resetPasswordLoading.value = false
  }
}

const handleSubmit = async () => {
  const errors = await formRef.value?.validate()
  if (errors) return

  submitLoading.value = true
  try {
    const roleIds = (userForm.roleIds || []).filter((id: any) => id != null)
    const positionIds = (userForm.positionIds || []).filter((id: any) => id != null)

    if (isEdit.value) {
      const userData = { ...userForm, positionIds, roleIds }
      await updateUser(userForm.id, userData)
      await assignUserRoles(userForm.id, roleIds)
      await assignUserPositions(userForm.id, positionIds)
      Message.success('更新成功')
    } else {
      await createUser(userForm)
      Message.success('创建成功')
    }

    dialogVisible.value = false
    loadUsers()
  } catch {
    Message.error('操作失败')
  } finally {
    submitLoading.value = false
  }
}

const handleDialogClose = () => {
  resetForm()
  dialogVisible.value = false
}

onMounted(() => {
  loadDepartmentTree()
  loadRoleOptions()
  loadPositionOptions()
  loadUsers()
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

.content-wrapper {
  display: flex;
  gap: 16px;
  min-height: 0;
  flex: 1;
}

/* 左侧部门树 */
.dept-tree-card {
  width: 280px;
  min-width: 280px;
  border-radius: var(--ops-border-radius-md, 8px);
  flex-shrink: 0;

  .card-title {
    display: flex;
    align-items: center;
    font-weight: 600;
  }
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 4px;
}

.node-label {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.node-count {
  color: var(--ops-text-tertiary, #86909c);
  font-size: 12px;
}

/* 右侧用户列表 */
.user-list-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

.dept-alert {
  border-radius: var(--ops-border-radius-md, 8px);
}

.dept-alert-content {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;

  .label {
    color: var(--ops-text-secondary, #4e5969);
    font-weight: 500;
  }

  .path {
    color: var(--ops-primary, #165dff);
  }
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
}

/* 用户单元格 */
.user-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}

.user-cell-info {
  min-width: 0;
}

.user-cell-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--ops-text-primary, #1d2129);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-cell-sub {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
}

/* 表单分区 */
.form-section {
  margin-bottom: 8px;
}

.form-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding-bottom: 10px;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
  font-size: 15px;
  font-weight: 600;
  color: var(--ops-text-primary, #1d2129);
}
</style>
