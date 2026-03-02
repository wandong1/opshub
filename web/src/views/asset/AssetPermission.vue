<template>
  <div class="permission-container">
    <!-- 页面标题和操作按钮 -->
    <div class="page-header">
      <div class="page-title-group">
        <div class="page-title-icon"><icon-lock /></div>
        <div>
          <h2 class="page-title">资产权限</h2>
          <p class="page-subtitle">管理资产分组和主机的访问权限</p>
        </div>
      </div>
      <a-button v-permission="'asset-perms:create'" type="primary" @click="handleAdd">
        <template #icon><icon-plus /></template>
        添加权限
      </a-button>
    </div>

    <!-- 搜索栏 -->
    <div class="filter-bar">
      <a-input
        v-model="searchForm.roleName"
        placeholder="搜索角色名称..."
        allow-clear
        style="width: 280px;"
        @input="handleSearch"
      >
        <template #prefix><icon-search /></template>
      </a-input>

      <a-input
        v-model="searchForm.groupName"
        placeholder="搜索资产分组..."
        allow-clear
        style="width: 280px;"
        @input="handleSearch"
      >
        <template #prefix><icon-search /></template>
      </a-input>

      <a-button @click="handleReset">
        <template #icon><icon-refresh /></template>
        重置
      </a-button>
    </div>

    <!-- 表格 -->
    <div class="table-wrapper">
      <a-table
        :data="filteredPermissions"
        :loading="loading"
        :bordered="{ cell: true }"
        stripe
        :pagination="{ current: page, pageSize: pageSize, total: total, showTotal: true, showPageSize: true, pageSizeOptions: [10, 20, 50, 100] }"
        @page-change="handlePageChange"
        @page-size-change="handleSizeChange"
      >
        <template #columns>
          <a-table-column title="ID" :width="80" align="center" data-index="id" />

          <a-table-column title="角色" :min-width="150">
            <template #cell="{ record }">
              <a-tag color="arcoblue">{{ record.roleName }}</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="资产分组" :min-width="180">
            <template #cell="{ record }">
              <a-tag color="green">{{ record.assetGroupName }}</a-tag>
            </template>
          </a-table-column>

          <a-table-column title="主机" :min-width="200">
            <template #cell="{ record }">
              <a-tag v-if="!record.hostId" color="gray">全部主机</a-tag>
              <div v-else>
                <div>{{ record.hostName }}</div>
                <div class="host-ip">{{ record.hostIp }}</div>
              </div>
            </template>
          </a-table-column>

          <a-table-column title="操作权限" :min-width="200">
            <template #cell="{ record }">
              <div class="permission-tags">
                <a-tag v-if="(record.permissions & 1) > 0" size="small" color="green">查看</a-tag>
                <a-tag v-if="(record.permissions & 2) > 0" size="small" color="arcoblue">编辑</a-tag>
                <a-tag v-if="(record.permissions & 4) > 0" size="small" color="red">删除</a-tag>
                <a-tag v-if="(record.permissions & 8) > 0" size="small" color="orangered">终端</a-tag>
                <a-tag v-if="(record.permissions & 16) > 0" size="small" color="gray">文件</a-tag>
                <a-tag v-if="(record.permissions & 32) > 0" size="small">采集</a-tag>
              </div>
            </template>
          </a-table-column>

          <a-table-column title="创建时间" :min-width="180" data-index="createdAt">
            <template #cell="{ record }">
              {{ formatTime(record.createdAt) }}
            </template>
          </a-table-column>

          <a-table-column title="操作" :width="120" align="center" fixed="right">
            <template #cell="{ record }">
              <div class="action-buttons">
                <a-tooltip content="编辑" position="top">
                  <a-button
                    v-permission="'asset-perms:update'"
                    type="text"
                    class="action-btn action-edit"
                    @click="handleEditClick(record)"
                  >
                    <template #icon><icon-edit /></template>
                  </a-button>
                </a-tooltip>
                <a-tooltip content="删除" position="top">
                  <a-button
                    v-permission="'asset-perms:delete'"
                    type="text"
                    class="action-btn action-delete"
                    @click="handleDeleteClick(record)"
                  >
                    <template #icon><icon-delete /></template>
                  </a-button>
                </a-tooltip>
              </div>
            </template>
          </a-table-column>
        </template>
      </a-table>
    </div>

    <!-- 添加权限对话框 -->
    <a-modal
      v-model:visible="dialogVisible"
      title="添加权限"
      :width="600"
      unmount-on-close
      :mask-closable="false"
      @close="handleDialogClose"
    >
      <a-form
        ref="formRef"
        :model="formData"
        :rules="formRules"
        auto-label-width
        layout="horizontal"
      >
        <a-form-item label="角色" field="roleId">
          <a-select
            v-model="formData.roleId"
            placeholder="请选择角色"
            allow-clear
            :allow-search="true"
          >
            <a-option
              v-for="role in roleList"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            />
          </a-select>
        </a-form-item>

        <a-form-item label="资产分组" field="assetGroupId">
          <a-tree-select
            v-model="formData.assetGroupId"
            :data="groupTreeData"
            :field-names="{ key: 'id', title: 'name', children: 'children' }"
            :tree-check-strictly="true"
            placeholder="请选择资产分组"
            @change="handleGroupChange"
          />
        </a-form-item>

        <a-form-item label="主机">
          <a-radio-group v-model="hostSelectionType" @change="handleHostTypeChange">
            <a-radio :value="'all'">全部主机</a-radio>
            <a-radio :value="'specific'">指定主机</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="hostSelectionType === 'specific'" label="选择主机" field="hostIds">
          <a-select
            v-model="formData.hostIds"
            :multiple="true"
            placeholder="请选择主机"
            :loading="loadingHosts"
          >
            <a-option
              v-for="host in hostList"
              :key="host.id"
              :label="`${host.name} (${host.ip})`"
              :value="host.id"
            />
          </a-select>
        </a-form-item>

        <a-form-item label="操作权限">
          <a-checkbox-group v-model="selectedPermissions">
            <a-checkbox :value="1">查看 - 查看主机详情</a-checkbox>
            <a-checkbox :value="2">编辑 - 创建、修改主机配置</a-checkbox>
            <a-checkbox :value="4">删除 - 删除主机</a-checkbox>
            <a-checkbox :value="8">终端 - SSH连接主机</a-checkbox>
            <a-checkbox :value="16">文件 - 文件上传、下载、删除</a-checkbox>
            <a-checkbox :value="32">采集 - 采集主机系统信息</a-checkbox>
          </a-checkbox-group>
          <div class="permission-tip">默认仅授予查看权限，请根据需要勾选其他操作权限</div>
        </a-form-item>
      </a-form>

      <template #footer>
        <div class="dialog-footer">
          <a-button @click="dialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleSubmit" :loading="submitting">确定</a-button>
        </div>
      </template>
    </a-modal>

    <!-- 编辑权限对话框 -->
    <a-modal
      v-model:visible="editDialogVisible"
      title="编辑权限"
      :width="600"
      unmount-on-close
      :mask-closable="false"
      @close="handleEditDialogClose"
    >
      <a-form
        ref="editFormRef"
        :model="editFormData"
        :rules="formRules"
        auto-label-width
        layout="horizontal"
      >
        <a-form-item label="角色" field="roleId">
          <a-select
            v-model="editFormData.roleId"
            placeholder="请选择角色"
            allow-clear
            :allow-search="true"
            disabled
          >
            <a-option
              v-for="role in roleList"
              :key="role.id"
              :label="role.name"
              :value="role.id"
            />
          </a-select>
        </a-form-item>

        <a-form-item label="资产分组" field="assetGroupId">
          <a-tree-select
            v-model="editFormData.assetGroupId"
            :data="groupTreeData"
            :field-names="{ key: 'id', title: 'name', children: 'children' }"
            :tree-check-strictly="true"
            placeholder="请选择资产分组"
            disabled
          />
        </a-form-item>

        <a-form-item label="主机">
          <a-radio-group v-model="editHostSelectionType" @change="handleEditHostTypeChange">
            <a-radio :value="'all'">全部主机</a-radio>
            <a-radio :value="'specific'">指定主机</a-radio>
          </a-radio-group>
        </a-form-item>

        <a-form-item v-if="editHostSelectionType === 'specific'" label="选择主机" field="hostIds">
          <a-select
            v-model="editFormData.hostIds"
            :multiple="true"
            placeholder="请选择主机"
            :loading="editLoadingHosts"
          >
            <a-option
              v-for="host in editHostList"
              :key="host.id"
              :label="`${host.name} (${host.ip})`"
              :value="host.id"
            />
          </a-select>
        </a-form-item>

        <a-form-item label="操作权限">
          <a-checkbox-group v-model="editFormData.permissions">
            <a-checkbox :value="1">查看 - 查看主机详情</a-checkbox>
            <a-checkbox :value="2">编辑 - 创建、修改主机配置</a-checkbox>
            <a-checkbox :value="4">删除 - 删除主机</a-checkbox>
            <a-checkbox :value="8">终端 - SSH连接主机</a-checkbox>
            <a-checkbox :value="16">文件 - 文件上传、下载、删除</a-checkbox>
            <a-checkbox :value="32">采集 - 采集主机系统信息</a-checkbox>
          </a-checkbox-group>
        </a-form-item>
      </a-form>

      <template #footer>
        <div class="dialog-footer">
          <a-button @click="editDialogVisible = false">取消</a-button>
          <a-button type="primary" @click="handleEditSubmit" :loading="editSubmitting">确定</a-button>
        </div>
      </template>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { Message, Modal } from '@arco-design/web-vue'
import type { FormInstance } from '@arco-design/web-vue'
import {
  IconPlus,
  IconDelete,
  IconSearch,
  IconRefresh,
  IconLock,
  IconEdit
} from '@arco-design/web-vue/es/icon'
import {
  getAssetPermissions,
  createAssetPermission,
  deleteAssetPermission,
  getAssetPermissionDetail,
  updateAssetPermission
} from '@/api/assetPermission'
import { getAllRoles } from '@/api/role'
import { getGroupTree } from '@/api/assetGroup'
import { getHostList } from '@/api/host'

const loading = ref(false)
const permissions = ref<any[]>([])
const page = ref(1)
const pageSize = ref(10)
const total = ref(0)
const deletingId = ref(0)

// 对话框相关
const dialogVisible = ref(false)
const editDialogVisible = ref(false)
const formRef = ref<FormInstance>()
const submitting = ref(false)
const hostSelectionType = ref('all')
const loadingHosts = ref(false)
const selectedPermissions = ref<number[]>([1]) // 默认仅查看权限

// 编辑表单数据
const editFormData = reactive({
  id: null as number | null,
  roleId: null as number | null,
  assetGroupId: null as number | null,
  hostIds: [] as number[],
  permissions: [] as number[]
})
const editSubmitting = ref(false)
const editFormRef = ref<FormInstance>()
const editHostSelectionType = ref('all')
const editLoadingHosts = ref(false)
const editHostList = ref<any[]>([])

// 搜索表单
const searchForm = reactive({
  roleName: '',
  groupName: ''
})

// 使用普通 ref 存储列表数据
const roleList = ref<any[]>([])
const groupTreeData = ref<any[]>([])
const hostList = ref<any[]>([])

// 表单数据 - 使用 reactive 以便更好地支持 v-model 绑定
const formData = reactive({
  roleId: null as number | null,
  assetGroupId: null as number | null,
  hostIds: [] as number[]
})

// 过滤后的权限列表
const filteredPermissions = computed(() => {
  let result = permissions.value

  if (searchForm.roleName) {
    result = result.filter(item =>
      item.roleName?.includes(searchForm.roleName)
    )
  }

  if (searchForm.groupName) {
    result = result.filter(item =>
      item.assetGroupName?.includes(searchForm.groupName)
    )
  }

  return result
})

// 表单验证规则
const formRules = {
  roleId: [{ required: true, message: '请选择角色' }],
  assetGroupId: [{ required: true, message: '请选择资产分组' }]
}

// 加载权限列表
const loadPermissions = async () => {
  loading.value = true
  try {
    const response = await getAssetPermissions({
      page: page.value,
      pageSize: pageSize.value
    })
    permissions.value = response.list || []
    total.value = response.total || 0
  } catch (error: any) {
    Message.error('加载权限列表失败: ' + (error.message || '未知错误'))
  } finally {
    loading.value = false
  }
}

// 加载角色列表
const loadRoles = async () => {
  try {
    const response = await getAllRoles()
    // API 返回的字段是 ID (大写)，不是 id
    roleList.value = (response || []).map((item: any) => ({
      id: item.ID,
      name: item.name,
      code: item.code
    }))
  } catch (error: any) {
    Message.error('加载角色列表失败: ' + (error.message || '未知错误'))
  }
}

// 加载资产分组树
const loadAssetGroupTree = async () => {
  try {
    const data = await getGroupTree()
    groupTreeData.value = convertTreeData(data || [])
  } catch (error: any) {
    Message.error('加载资产分组失败: ' + (error.message || '未知错误'))
  }
}

// 转换树形数据格式
const convertTreeData = (nodes: any[]): any[] => {
  return nodes.map((node: any) => ({
    id: node.id,
    name: node.name,
    children: node.children ? convertTreeData(node.children) : undefined
  }))
}

// 加载主机列表
const loadHosts = async (groupId?: number) => {
  if (!groupId) return

  loadingHosts.value = true
  try {
    const response = await getHostList({ page: 1, pageSize: 1000, groupId })
    hostList.value = (response.list || []).map((item: any) => ({
      id: item.id,
      name: item.name,
      ip: item.ip
    }))
  } catch (error: any) {
    Message.error('加载主机列表失败: ' + (error.message || '未知错误'))
  } finally {
    loadingHosts.value = false
  }
}

// 处理搜索
const handleSearch = () => {
  page.value = 1
  loadPermissions()
}

// 重置搜索
const handleReset = () => {
  searchForm.roleName = ''
  searchForm.groupName = ''
  page.value = 1
  loadPermissions()
}

// 处理资产分组变化
const handleGroupChange = (value: number) => {
  formData.hostIds = []
  hostList.value = []
  if (value && hostSelectionType.value === 'specific') {
    loadHosts(value)
  }
}

// 处理主机类型变化
const handleHostTypeChange = (value: string) => {
  if (value === 'specific' && formData.assetGroupId) {
    loadHosts(formData.assetGroupId)
  } else {
    formData.hostIds = []
  }
}

// 添加权限
const handleAdd = () => {
  resetForm()
  dialogVisible.value = true
}

// 删除权限
const handleDeleteClick = (row: any) => {
  Modal.warning({
    title: '提示',
    content: '确定删除此权限吗？',
    hideCancel: false,
    onOk: async () => {
      await handleDelete(row.id)
    }
  })
}

const handleDelete = async (id: number) => {
  deletingId.value = id
  try {
    await deleteAssetPermission(id)
    Message.success('删除成功')
    loadPermissions()
  } catch (error: any) {
    Message.error('删除失败: ' + (error.message || '未知错误'))
  } finally {
    deletingId.value = 0
  }
}

// 编辑权限
const handleEditClick = async (row: any) => {
  try {
    const detail = await getAssetPermissionDetail(row.id)

    editFormData.id = detail.id
    editFormData.roleId = detail.roleId
    editFormData.assetGroupId = detail.assetGroupId
    editFormData.hostIds = detail.hostIds || []
    editFormData.permissions = []

    // 根据权限位掩码设置checkbox
    if ((detail.permissions & 1) > 0) editFormData.permissions.push(1)
    if ((detail.permissions & 2) > 0) editFormData.permissions.push(2)
    if ((detail.permissions & 4) > 0) editFormData.permissions.push(4)
    if ((detail.permissions & 8) > 0) editFormData.permissions.push(8)
    if ((detail.permissions & 16) > 0) editFormData.permissions.push(16)
    if ((detail.permissions & 32) > 0) editFormData.permissions.push(32)

    // 设置主机选择类型：如果hostIds为空或长度为0，则为全部主机，否则为指定主机
    editHostSelectionType.value = (!detail.hostIds || detail.hostIds.length === 0) ? 'all' : 'specific'

    // 加载主机列表
    if (editHostSelectionType.value === 'specific') {
      await loadEditHosts(detail.assetGroupId)
    }

    editDialogVisible.value = true
  } catch (error: any) {
    Message.error('加载权限详情失败: ' + (error.message || '未知错误'))
  }
}

// 加载编辑时的主机列表
const loadEditHosts = async (groupId?: number) => {
  if (!groupId) return

  editLoadingHosts.value = true
  try {
    const response = await getHostList({ page: 1, pageSize: 1000, groupId })
    editHostList.value = (response.list || []).map((item: any) => ({
      id: item.id,
      name: item.name,
      ip: item.ip
    }))
  } catch (error: any) {
    Message.error('加载主机列表失败: ' + (error.message || '未知错误'))
  } finally {
    editLoadingHosts.value = false
  }
}

// 处理编辑时的主机类型变化
const handleEditHostTypeChange = (value: string) => {
  if (value === 'specific' && editFormData.assetGroupId) {
    loadEditHosts(editFormData.assetGroupId)
  } else {
    editFormData.hostIds = []
  }
}

// 关闭编辑对话框
const handleEditDialogClose = () => {
  editFormData.id = null
  editFormData.roleId = null
  editFormData.assetGroupId = null
  editFormData.hostIds = []
  editFormData.permissions = []
  editHostSelectionType.value = 'all'
  editHostList.value = []
  editFormRef.value?.clearValidate()
}

// 提交编辑
const handleEditSubmit = async () => {
  if (editFormData.id === null) return

  editSubmitting.value = true
  try {
    // 计算权限位掩码
    const permissions = editFormData.permissions.reduce((acc, val) => acc | val, 0)

    await updateAssetPermission(editFormData.id, {
      roleId: editFormData.roleId!,
      assetGroupId: editFormData.assetGroupId!,
      hostIds: editHostSelectionType.value === 'all' ? [] : editFormData.hostIds,
      permissions: permissions
    })
    Message.success('更新成功')
    editDialogVisible.value = false
    loadPermissions()
  } catch (error: any) {
    Message.error('更新失败: ' + (error.message || '未知错误'))
  } finally {
    editSubmitting.value = false
  }
}

// 分页变化
const handleSizeChange = (newPageSize: number) => {
  pageSize.value = newPageSize
  page.value = 1
  loadPermissions()
}

const handlePageChange = (newPage: number) => {
  page.value = newPage
  loadPermissions()
}

// 重置表单
const resetForm = () => {
  formData.roleId = null
  formData.assetGroupId = null
  formData.hostIds = []
  hostSelectionType.value = 'all'
  hostList.value = []
  selectedPermissions.value = [1] // 重置为仅查看权限
  formRef.value?.clearValidate()
}

// 关闭对话框
const handleDialogClose = () => {
  formRef.value?.resetFields()
  resetForm()
}

// 提交表单
const handleSubmit = async () => {
  if (!formRef.value) return

  const errors = await formRef.value.validate()
  if (errors) return

  submitting.value = true
  try {
    // 计算权限位掩码
    const permissions = selectedPermissions.value.reduce((acc, val) => acc | val, 0)

    await createAssetPermission({
      roleId: formData.roleId!,
      assetGroupId: formData.assetGroupId!,
      hostIds: hostSelectionType.value === 'all' ? [] : formData.hostIds,
      permissions: permissions
    })
    Message.success('添加成功')
    dialogVisible.value = false
    loadPermissions()
  } catch (error: any) {
    Message.error('添加失败: ' + (error.message || '未知错误'))
  } finally {
    submitting.value = false
  }
}

// 格式化时间
const formatTime = (time: string) => {
  if (!time) return ''
  return new Date(time).toLocaleString('zh-CN')
}

onMounted(() => {
  loadPermissions()
  loadRoles()
  loadAssetGroupTree()
})
</script>

<style scoped>
.permission-container {
  padding: 0;
  background-color: transparent;
}

/* 页面头部 */
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 12px;
  padding: 16px 20px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
}

.page-title-group {
  display: flex;
  align-items: flex-start;
  gap: 16px;
}

.page-title-icon {
  width: 36px;
  height: 36px;
  background: var(--ops-primary);
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  font-size: 18px;
  flex-shrink: 0;
}

.page-title {
  margin: 0;
  font-size: 20px;
  font-weight: 600;
  color: var(--ops-text-primary);
  line-height: 1.3;
}

.page-subtitle {
  margin: 4px 0 0 0;
  font-size: 13px;
  color: var(--ops-text-tertiary);
  line-height: 1.4;
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

/* 表格容器 */
.table-wrapper {
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
  overflow: hidden;
}

/* 操作按钮 */
.action-buttons {
  display: flex;
  gap: 8px;
  align-items: center;
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
  background-color: #e6f7ff;
  color: #1890ff;
}

.action-delete:hover {
  background-color: #fee;
  color: #f56c6c;
}

/* 主机IP样式 */
.host-ip {
  font-size: 12px;
  color: var(--ops-text-tertiary);
  font-family: 'Consolas', 'Monaco', monospace;
  margin-top: 4px;
}

/* 对话框样式 */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 权限标签样式 */
.permission-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}

/* 权限表单提示 */
.permission-tip {
  font-size: 12px;
  color: var(--ops-text-tertiary);
  margin-top: 8px;
}

@media (max-width: 768px) {
  .filter-bar {
    flex-wrap: wrap;
  }
}
</style>