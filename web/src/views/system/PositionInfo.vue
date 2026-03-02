<template>
  <div class="page-container">
    <!-- 页面头部 -->
    <div class="page-header-card">
      <div class="page-header-inner">
        <div class="page-icon">
          <icon-bookmark />
        </div>
        <div>
          <div class="page-title">岗位信息</div>
          <div class="page-desc">管理系统岗位信息，支持岗位与用户的关联管理</div>
        </div>
      </div>
    </div>

    <!-- 搜索区域 -->
    <a-card class="search-card" :bordered="false">
      <a-row :gutter="16" align="center">
        <a-col :flex="'auto'">
          <a-space :size="16" wrap>
            <a-space>
              <span class="search-label">岗位编码:</span>
              <a-input v-model="searchForm.postCode" placeholder="请输入" allow-clear style="width: 200px" @press-enter="handleSearch" />
            </a-space>
            <a-space>
              <span class="search-label">岗位名称:</span>
              <a-input v-model="searchForm.postName" placeholder="请输入" allow-clear style="width: 200px" @press-enter="handleSearch" />
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
      </a-row>
    </a-card>

    <!-- 岗位列表 -->
    <a-card class="table-card" :bordered="false">
      <template #title>
        <span class="card-title">岗位列表</span>
      </template>
      <template #extra>
        <a-space>
          <a-button v-permission="'positions:create'" type="primary" @click="handleAdd">
            <template #icon><icon-plus /></template>
            新增岗位
          </a-button>
          <a-button @click="loadPositionList">
            <template #icon><icon-refresh /></template>
          </a-button>
        </a-space>
      </template>

      <a-table
        :data="positionList"
        :loading="loading"
        row-key="id"
        :pagination="tablePagination"
        @page-change="handlePageChange"
        @page-size-change="handlePageSizeChange"
      >
        <template #columns>
          <a-table-column title="序号" :width="70" align="center">
            <template #cell="{ rowIndex }">
              {{ (pagination.page - 1) * pagination.pageSize + rowIndex + 1 }}
            </template>
          </a-table-column>
          <a-table-column title="岗位编码" data-index="postCode" :min-width="140" />
          <a-table-column title="岗位名称" data-index="postName" :min-width="160" />
          <a-table-column title="状态" :width="100" align="center">
            <template #cell="{ record }">
              <a-tag :color="record.postStatus === 1 ? 'green' : 'red'" size="small">
                {{ record.postStatus === 1 ? '启用' : '禁用' }}
              </a-tag>
            </template>
          </a-table-column>
          <a-table-column title="创建时间" data-index="createTime" :min-width="180" />
          <a-table-column title="备注" data-index="remark" ellipsis tooltip :min-width="200" />
          <a-table-column title="操作" :width="160" align="center" fixed="right">
            <template #cell="{ record }">
              <a-space>
                <a-link v-permission="'positions:update'" @click="handleEdit(record)">编辑</a-link>
                <a-link v-permission="'positions:assign-users'" @click="handleAssignUsers(record)">分配</a-link>
                <a-popconfirm content="确定要删除该岗位吗？" @ok="handleDelete(record)">
                  <a-link v-permission="'positions:delete'" status="danger">删除</a-link>
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
      :title="isEdit ? '编辑岗位' : '新增岗位'"
      :width="560"
      :unmount-on-close="true"
      @ok="handleSubmit"
      @cancel="dialogVisible = false"
    >
      <a-form :model="positionForm" layout="vertical" ref="formRef">
        <a-form-item label="岗位名称" field="postName" :rules="[{ required: true, message: '请输入岗位名称' }]">
          <a-input v-model="positionForm.postName" placeholder="请输入岗位名称" />
        </a-form-item>
        <a-form-item label="岗位编码" field="postCode" :rules="[{ required: true, message: '请输入岗位编码' }]">
          <a-input v-model="positionForm.postCode" placeholder="请输入岗位编码" />
        </a-form-item>
        <a-form-item label="备注" field="remark">
          <a-textarea v-model="positionForm.remark" placeholder="请输入备注" :auto-size="{ minRows: 3, maxRows: 6 }" />
        </a-form-item>
        <a-form-item label="状态" field="postStatus">
          <a-radio-group v-model="positionForm.postStatus" type="button">
            <a-radio :value="1">启用</a-radio>
            <a-radio :value="2">禁用</a-radio>
          </a-radio-group>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 分配用户对话框 -->
    <a-modal
      v-model:visible="assignUsersVisible"
      title="分配用户"
      :width="860"
      :unmount-on-close="true"
      @ok="handleAssignUsersSubmit"
      @cancel="assignUsersVisible = false"
    >
      <div class="assign-content">
        <!-- 已分配用户 -->
        <div class="assign-panel">
          <div class="panel-header">
            <span class="panel-title">
              <icon-user-group style="margin-right: 6px; color: var(--ops-primary);" />
              已分配用户
              <a-badge :count="selectedUsers.length" :dot-style="{ background: 'var(--ops-primary)' }" style="margin-left: 8px;" />
            </span>
          </div>
          <div class="panel-body">
            <a-empty v-if="selectedUsers.length === 0" description="暂无已分配用户" />
            <div v-else class="user-cards">
              <div v-for="user in selectedUsers" :key="user.id || user.ID" class="user-card selected">
                <a-avatar :size="36" :style="{ backgroundColor: 'var(--ops-primary)' }">
                  {{ (user.realName || user.username || '').charAt(0) }}
                </a-avatar>
                <div class="user-info">
                  <div class="user-name">{{ user.realName || user.username }}</div>
                  <div class="user-sub">@{{ user.username }}</div>
                </div>
                <a-button type="text" status="danger" size="mini" @click="removeUser(user)">
                  <template #icon><icon-close /></template>
                </a-button>
              </div>
            </div>
          </div>
        </div>

        <!-- 可用用户 -->
        <div class="assign-panel">
          <div class="panel-header">
            <span class="panel-title">
              <icon-user style="margin-right: 6px; color: var(--ops-primary);" />
              可用用户
            </span>
            <a-input
              v-model="userSearchKeyword"
              placeholder="搜索用户名..."
              allow-clear
              size="small"
              style="width: 180px;"
              @press-enter="loadAvailableUsers"
              @clear="loadAvailableUsers"
            >
              <template #prefix><icon-search /></template>
            </a-input>
          </div>
          <div class="panel-body">
            <a-spin :loading="usersLoading" style="width: 100%;">
              <a-empty v-if="availableUsers.length === 0 && !usersLoading" description="暂无可分配用户" />
              <div v-else class="user-cards">
                <div v-for="user in availableUsers" :key="user.id || user.ID" class="user-card">
                  <a-avatar :size="36" :style="{ backgroundColor: '#86909c' }">
                    {{ (user.realName || user.username || '').charAt(0) }}
                  </a-avatar>
                  <div class="user-info">
                    <div class="user-name">{{ user.realName || user.username }}</div>
                    <div class="user-sub">@{{ user.username }}</div>
                  </div>
                  <a-button type="text" status="normal" size="mini" @click="addUser(user)">
                    <template #icon><icon-plus /></template>
                  </a-button>
                </div>
              </div>
            </a-spin>
            <div v-if="userPagination.total > 0" class="panel-pagination">
              <a-pagination
                v-model:current="userPagination.page"
                v-model:page-size="userPagination.pageSize"
                :total="userPagination.total"
                :page-size-options="[10, 20, 50]"
                size="small"
                show-total
                show-page-size
                @change="loadAvailableUsers"
                @page-size-change="loadAvailableUsers"
              />
            </div>
          </div>
        </div>
      </div>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { Message } from '@arco-design/web-vue'
import {
  IconBookmark,
  IconSearch,
  IconRefresh,
  IconPlus,
  IconClose,
  IconUser,
  IconUserGroup,
} from '@arco-design/web-vue/es/icon'
import {
  getPositionList,
  createPosition,
  updatePosition,
  deletePosition,
  getPositionUsers,
  assignUsersToPosition
} from '@/api/position'
import { getUserList } from '@/api/user'

const loading = ref(false)
const submitLoading = ref(false)
const dialogVisible = ref(false)
const assignUsersVisible = ref(false)
const usersLoading = ref(false)
const isEdit = ref(false)
const formRef = ref()

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

// 搜索表单
const searchForm = reactive({
  postCode: '',
  postName: ''
})

// 岗位列表
const positionList = ref<any[]>([])

// 岗位表单
const positionForm = reactive({
  id: 0,
  postName: '',
  postCode: '',
  remark: '',
  postStatus: 1
})

// 用户相关
const currentPositionId = ref(0)
const userSearchKeyword = ref('')
const selectedUsers = ref<any[]>([])
const availableUsers = ref<any[]>([])
const userPagination = reactive({
  page: 1,
  pageSize: 10,
  total: 0
})

// 加载岗位列表
const loadPositionList = async () => {
  loading.value = true
  try {
    const res: any = await getPositionList({
      page: pagination.page,
      pageSize: pagination.pageSize,
      postCode: searchForm.postCode || undefined,
      postName: searchForm.postName || undefined
    })
    positionList.value = res.list || []
    pagination.total = res.total || 0
  } catch {
    positionList.value = []
    pagination.total = 0
  } finally {
    loading.value = false
  }
}

const handleSearch = () => {
  pagination.page = 1
  loadPositionList()
}

const resetSearch = () => {
  searchForm.postCode = ''
  searchForm.postName = ''
  pagination.page = 1
  loadPositionList()
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadPositionList()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.pageSize = pageSize
  pagination.page = 1
  loadPositionList()
}

// 重置表单
const resetForm = () => {
  positionForm.id = 0
  positionForm.postName = ''
  positionForm.postCode = ''
  positionForm.remark = ''
  positionForm.postStatus = 1
  formRef.value?.clearValidate()
}

// 新增岗位
const handleAdd = () => {
  resetForm()
  isEdit.value = false
  dialogVisible.value = true
}

// 编辑岗位
const handleEdit = (row: any) => {
  Object.assign(positionForm, row)
  isEdit.value = true
  dialogVisible.value = true
}

// 删除岗位
const handleDelete = async (row: any) => {
  try {
    await deletePosition(row.id)
    Message.success('删除成功')
    loadPositionList()
  } catch {
    Message.error('删除失败')
  }
}

// 提交表单
const handleSubmit = async () => {
  const errors = await formRef.value?.validate()
  if (errors) return

  submitLoading.value = true
  try {
    if (isEdit.value) {
      await updatePosition(positionForm.id, positionForm)
      Message.success('更新成功')
    } else {
      await createPosition(positionForm)
      Message.success('创建成功')
    }
    dialogVisible.value = false
    loadPositionList()
  } catch {
    Message.error(isEdit.value ? '更新失败' : '创建失败')
  } finally {
    submitLoading.value = false
  }
}

// 分配用户 - 打开对话框
const handleAssignUsers = async (row: any) => {
  currentPositionId.value = row.id
  selectedUsers.value = []
  userSearchKeyword.value = ''
  userPagination.page = 1

  try {
    const res: any = await getPositionUsers(row.id)
    selectedUsers.value = res.list || res.data || []
  } catch {
    selectedUsers.value = []
  }

  await loadAvailableUsers()
  assignUsersVisible.value = true
}

// 加载可用用户列表
const loadAvailableUsers = async () => {
  usersLoading.value = true
  try {
    const selectedUserIds = selectedUsers.value.map(u => u.id || u.ID)
    const res: any = await getUserList({
      page: userPagination.page,
      pageSize: userPagination.pageSize,
      keyword: userSearchKeyword.value || undefined
    })
    const userList = res.list || []
    availableUsers.value = userList.filter((u: any) => !selectedUserIds.includes(u.id || u.ID))
    userPagination.total = (res.total || 0) - selectedUsers.value.length
  } catch {
    availableUsers.value = []
  } finally {
    usersLoading.value = false
  }
}

// 添加用户到已选列表
const addUser = (user: any) => {
  const userId = user.id || user.ID
  if (selectedUsers.value.some(u => (u.id || u.ID) === userId)) {
    Message.warning('该用户已添加')
    return
  }
  selectedUsers.value.push(user)
  loadAvailableUsers()
}

// 从已选列表移除用户
const removeUser = (user: any) => {
  const userId = user.id || user.ID
  selectedUsers.value = selectedUsers.value.filter(u => (u.id || u.ID) !== userId)
  loadAvailableUsers()
}

// 提交分配用户
const handleAssignUsersSubmit = async () => {
  submitLoading.value = true
  try {
    const userIds = selectedUsers.value.map(u => u.id || u.ID)
    await assignUsersToPosition(currentPositionId.value, userIds)
    Message.success('分配用户成功')
    assignUsersVisible.value = false
  } catch {
    Message.error('分配用户失败')
  } finally {
    submitLoading.value = false
  }
}

onMounted(() => {
  loadPositionList()
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

/* 分配用户对话框 */
.assign-content {
  display: flex;
  gap: 16px;
  min-height: 380px;
}

.assign-panel {
  flex: 1;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: var(--ops-border-radius-md, 8px);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.panel-header {
  padding: 12px 16px;
  border-bottom: 1px solid var(--ops-border-color, #e5e6eb);
  display: flex;
  justify-content: space-between;
  align-items: center;
  background: #fafbfc;
}

.panel-title {
  display: flex;
  align-items: center;
  font-weight: 600;
  font-size: 14px;
  color: var(--ops-text-primary, #1d2129);
}

.panel-body {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
  max-height: 380px;
}

.user-cards {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.user-card {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 12px;
  border: 1px solid var(--ops-border-color, #e5e6eb);
  border-radius: var(--ops-border-radius-md, 8px);
  transition: all 0.2s;

  &:hover {
    background: #f7f8fa;
    border-color: var(--ops-primary-lighter, #6694ff);
  }

  &.selected {
    border-color: var(--ops-primary, #165dff);
    background: var(--ops-primary-bg, #e8f0ff);
  }
}

.user-info {
  flex: 1;
  min-width: 0;
}

.user-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--ops-text-primary, #1d2129);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-sub {
  font-size: 12px;
  color: var(--ops-text-tertiary, #86909c);
  margin-top: 1px;
}

.panel-pagination {
  margin-top: 12px;
  display: flex;
  justify-content: center;
}
</style>
